package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/autobuild/schedule"
	"github.com/coder/coder/coderd/util/ptr"
	"github.com/coder/coder/codersdk"
)

const ttlDescriptionLong = `To have your workspace stop automatically after a configurable interval has passed.
Minimum TTL is 1 minute.
`

func ttl() *cobra.Command {
	ttlCmd := &cobra.Command{
		Annotations: workspaceCommand,
		Use:         "ttl [command]",
		Short:       "Schedule a workspace to automatically stop after a configurable interval",
		Long:        ttlDescriptionLong,
		Example:     "coder ttl set my-workspace 8h30m",
	}

	ttlCmd.AddCommand(ttlShow())
	ttlCmd.AddCommand(ttlset())
	ttlCmd.AddCommand(ttlunset())

	return ttlCmd
}

func ttlShow() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "show <workspace_name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return xerrors.Errorf("create client: %w", err)
			}

			workspace, err := namedWorkspace(cmd, client, args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}

			if workspace.TTLMillis == nil || *workspace.TTLMillis == 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "not set\n")
				return nil
			}

			dur := time.Duration(*workspace.TTLMillis) * time.Millisecond
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", dur)

			return nil
		},
	}
	return cmd
}

func ttlset() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "set <workspace_name> <ttl>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return xerrors.Errorf("create client: %w", err)
			}

			workspace, err := namedWorkspace(cmd, client, args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}

			ttl, err := time.ParseDuration(args[1])
			if err != nil {
				return xerrors.Errorf("parse ttl: %w", err)
			}

			truncated := ttl.Truncate(time.Minute)

			if truncated == 0 {
				return xerrors.Errorf("ttl must be at least 1m")
			}

			if truncated != ttl {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "warning: ttl rounded down to %s\n", truncated)
			}

			millis := truncated.Milliseconds()
			if err = client.UpdateWorkspaceTTL(cmd.Context(), workspace.ID, codersdk.UpdateWorkspaceTTLRequest{
				TTLMillis: &millis,
			}); err != nil {
				return xerrors.Errorf("update workspace ttl: %w", err)
			}

			if ptr.NilOrEmpty(workspace.AutostartSchedule) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%q will shut down %s after start.\n", workspace.Name, truncated)
				return nil
			}

			sched, err := schedule.Weekly(*workspace.AutostartSchedule)
			if err != nil {
				return xerrors.Errorf("parse workspace schedule: %w", err)
			}

			loc, err := time.LoadLocation(sched.Timezone())
			if err != nil {
				return xerrors.Errorf("schedule has invalid timezone: %w", err)
			}

			nextShutdown := sched.Next(time.Now()).Add(truncated).In(loc)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%q will shut down at %s on %s (%s after start).\n",
				workspace.Name,
				nextShutdown.Format("15:04:05 MST"),
				nextShutdown.Format("2006-02-01"),
				truncated,
			)
			return nil
		},
	}

	return cmd
}

func ttlunset() *cobra.Command {
	return &cobra.Command{
		Use:  "unset <workspace_name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return xerrors.Errorf("create client: %w", err)
			}

			workspace, err := namedWorkspace(cmd, client, args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}

			err = client.UpdateWorkspaceTTL(cmd.Context(), workspace.ID, codersdk.UpdateWorkspaceTTLRequest{
				TTLMillis: nil,
			})
			if err != nil {
				return xerrors.Errorf("update workspace ttl: %w", err)
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), "ttl unset\n", workspace.Name)

			return nil
		},
	}
}
