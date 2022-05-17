package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/gofrs/flock"
	"github.com/google/uuid"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"golang.org/x/xerrors"

	"github.com/coder/coder/cli/cliflag"
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/coderd/autobuild/notify"
	"github.com/coder/coder/coderd/autobuild/schedule"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/codersdk"
)

var autostopPollInterval = 30 * time.Second
var autostopNotifyCountdown = []time.Duration{30 * time.Minute}

func ssh() *cobra.Command {
	var (
		stdio bool
	)
	cmd := &cobra.Command{
		Annotations: workspaceCommand,
		Use:         "ssh <workspace>",
		Short:       "SSH into a workspace",
		Args:        cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}
			organization, err := currentOrganization(cmd, client)
			if err != nil {
				return err
			}

			workspace, agent, err := getWorkspaceAndAgent(cmd.Context(), client, organization.ID, codersdk.Me, args[0])
			if err != nil {
				return err
			}
			if workspace.LatestBuild.Transition != database.WorkspaceTransitionStart {
				return xerrors.New("workspace must be in start transition to ssh")
			}
			if workspace.LatestBuild.Job.CompletedAt == nil {
				err = cliui.WorkspaceBuild(cmd.Context(), cmd.ErrOrStderr(), client, workspace.LatestBuild.ID, workspace.CreatedAt)
				if err != nil {
					return err
				}
			}

			// OpenSSH passes stderr directly to the calling TTY.
			// This is required in "stdio" mode so a connecting indicator can be displayed.
			err = cliui.Agent(cmd.Context(), cmd.ErrOrStderr(), cliui.AgentOptions{
				WorkspaceName: workspace.Name,
				Fetch: func(ctx context.Context) (codersdk.WorkspaceAgent, error) {
					return client.WorkspaceAgent(ctx, agent.ID)
				},
			})
			if err != nil {
				return xerrors.Errorf("await agent: %w", err)
			}

			conn, err := client.DialWorkspaceAgent(cmd.Context(), agent.ID, nil)
			if err != nil {
				return err
			}
			defer conn.Close()

			stopPolling := tryPollWorkspaceAutostop(cmd.Context(), client, workspace)
			defer stopPolling()

			if stdio {
				rawSSH, err := conn.SSH()
				if err != nil {
					return err
				}
				go func() {
					_, _ = io.Copy(cmd.OutOrStdout(), rawSSH)
				}()
				_, _ = io.Copy(rawSSH, cmd.InOrStdin())
				return nil
			}
			sshClient, err := conn.SSHClient()
			if err != nil {
				return err
			}

			sshSession, err := sshClient.NewSession()
			if err != nil {
				return err
			}

			stdoutFile, valid := cmd.OutOrStdout().(*os.File)
			if valid && isatty.IsTerminal(stdoutFile.Fd()) {
				state, err := term.MakeRaw(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				defer func() {
					_ = term.Restore(int(os.Stdin.Fd()), state)
				}()

				windowChange := listenWindowSize(cmd.Context())
				go func() {
					for {
						select {
						case <-cmd.Context().Done():
							return
						case <-windowChange:
						}
						width, height, _ := term.GetSize(int(stdoutFile.Fd()))
						_ = sshSession.WindowChange(height, width)
					}
				}()
			}

			err = sshSession.RequestPty("xterm-256color", 128, 128, gossh.TerminalModes{})
			if err != nil {
				return err
			}

			sshSession.Stdin = cmd.InOrStdin()
			sshSession.Stdout = cmd.OutOrStdout()
			sshSession.Stderr = cmd.OutOrStdout()

			err = sshSession.Shell()
			if err != nil {
				return err
			}

			err = sshSession.Wait()
			if err != nil {
				return err
			}

			return nil
		},
	}
	cliflag.BoolVarP(cmd.Flags(), &stdio, "stdio", "", "CODER_SSH_STDIO", false, "Specifies whether to emit SSH output over stdin/stdout.")

	return cmd
}

func getWorkspaceAndAgent(ctx context.Context, client *codersdk.Client, orgID uuid.UUID, userID string, in string) (codersdk.Workspace, codersdk.WorkspaceAgent, error) {
	workspaceParts := strings.Split(in, ".")
	workspace, err := client.WorkspaceByOwnerAndName(ctx, orgID, userID, workspaceParts[0])
	if err != nil {
		return codersdk.Workspace{}, codersdk.WorkspaceAgent{}, xerrors.Errorf("get workspace %q: %w", workspaceParts[0], err)
	}

	if workspace.LatestBuild.Transition == database.WorkspaceTransitionDelete {
		return codersdk.Workspace{}, codersdk.WorkspaceAgent{}, xerrors.Errorf("workspace %q is being deleted", workspace.Name)
	}

	resources, err := client.WorkspaceResourcesByBuild(ctx, workspace.LatestBuild.ID)
	if err != nil {
		return codersdk.Workspace{}, codersdk.WorkspaceAgent{}, xerrors.Errorf("fetch workspace resources: %w", err)
	}

	agents := make([]codersdk.WorkspaceAgent, 0)
	for _, resource := range resources {
		agents = append(agents, resource.Agents...)
	}
	if len(agents) == 0 {
		return codersdk.Workspace{}, codersdk.WorkspaceAgent{}, xerrors.Errorf("workspace %q has no agents", workspace.Name)
	}

	var (
		// We can't use a pointer because linters are mad about using pointers
		// from loop variables
		agent   codersdk.WorkspaceAgent
		agentOK bool
	)
	if len(workspaceParts) >= 2 {
		for _, otherAgent := range agents {
			if otherAgent.Name != workspaceParts[1] {
				continue
			}
			agent = otherAgent
			agentOK = true
			break
		}

		if !agentOK {
			return codersdk.Workspace{}, codersdk.WorkspaceAgent{}, xerrors.Errorf("agent not found by name %q", workspaceParts[1])
		}
	}

	if !agentOK {
		if len(agents) > 1 {
			return codersdk.Workspace{}, codersdk.WorkspaceAgent{}, xerrors.New("you must specify the name of an agent")
		}
		agent = agents[0]
	}

	return workspace, agent, nil
}

type stdioConn struct {
	io.Reader
	io.Writer
}

func (*stdioConn) Close() (err error) {
	return nil
}

// Attempt to poll workspace autostop. We write a per-workspace lockfile to
// avoid spamming the user with notifications in case of multiple instances
// of the CLI running simultaneously.
func tryPollWorkspaceAutostop(ctx context.Context, client *codersdk.Client, workspace codersdk.Workspace) (stop func()) {
	lock := flock.New(filepath.Join(os.TempDir(), "coder-autostop-notify-"+workspace.ID.String()))
	condition := notifyCondition(ctx, client, workspace.ID, lock)
	return notify.Notify(condition, autostopPollInterval, autostopNotifyCountdown...)
}

// Notify the user if the workspace is due to shutdown.
func notifyCondition(ctx context.Context, client *codersdk.Client, workspaceID uuid.UUID, lock *flock.Flock) notify.Condition {
	return func(now time.Time) (deadline time.Time, callback func()) {
		// Keep trying to regain the lock.
		locked, err := lock.TryLockContext(ctx, autostopPollInterval)
		if err != nil || !locked {
			return time.Time{}, nil
		}

		ws, err := client.Workspace(ctx, workspaceID)
		if err != nil {
			return time.Time{}, nil
		}

		if ws.AutostopSchedule == "" {
			return time.Time{}, nil
		}

		sched, err := schedule.Weekly(ws.AutostopSchedule)
		if err != nil {
			return time.Time{}, nil
		}

		deadline = sched.Next(now)
		callback = func() {
			ttl := deadline.Sub(now)
			var title, body string
			if ttl > time.Minute {
				title = fmt.Sprintf(`Workspace %s stopping in %.0f mins`, ws.Name, ttl.Minutes())
				body = fmt.Sprintf(
					`Your Coder workspace %s is scheduled to stop at %s.`,
					ws.Name,
					deadline.Format(time.Kitchen),
				)
			} else {
				title = fmt.Sprintf("Workspace %s stopping!", ws.Name)
				body = fmt.Sprintf("Your Coder workspace %s is stopping any time now!", ws.Name)
			}
			// notify user with a native system notification (best effort)
			_ = beeep.Notify(title, body, "")
		}
		return deadline.Truncate(time.Minute), callback
	}
}
