package cli

import (
	"fmt"
	"time"

	"golang.org/x/xerrors"

	"github.com/coder/pretty"

	"github.com/coder/coder/v2/cli/clibase"
	"github.com/coder/coder/v2/cli/cliui"
	"github.com/coder/coder/v2/codersdk"
)

func (r *RootCmd) restart() *clibase.Cmd {
	var parameterFlags workspaceParameterFlags

	client := new(codersdk.Client)
	cmd := &clibase.Cmd{
		Annotations: workspaceCommand,
		Use:         "restart <workspace>",
		Short:       "Restart a workspace",
		Middleware: clibase.Chain(
			clibase.RequireNArgs(1),
			r.InitClient(client),
		),
		Options: append(parameterFlags.cliBuildOptions(), cliui.SkipPromptOption()),
		Handler: func(inv *clibase.Invocation) error {
			ctx := inv.Context()
			out := inv.Stdout

			workspace, err := namedWorkspace(inv.Context(), client, inv.Args[0])
			if err != nil {
				return err
			}

			lastBuildParameters, err := client.WorkspaceBuildParameters(inv.Context(), workspace.LatestBuild.ID)
			if err != nil {
				return err
			}

			buildOptions, err := asWorkspaceBuildParameters(parameterFlags.buildOptions)
			if err != nil {
				return xerrors.Errorf("can't parse build options: %w", err)
			}

			template, err := client.Template(inv.Context(), workspace.TemplateID)
			if err != nil {
				return xerrors.Errorf("get template: %w", err)
			}

			versionID := workspace.LatestBuild.TemplateVersionID
			if template.RequireActiveVersion {
				key := "template"
				resp, err := client.AuthCheck(inv.Context(), codersdk.AuthorizationRequest{
					Checks: map[string]codersdk.AuthorizationCheck{
						key: {
							Object: codersdk.AuthorizationObject{
								ResourceType:   codersdk.ResourceTemplate,
								OwnerID:        workspace.OwnerID.String(),
								OrganizationID: workspace.OrganizationID.String(),
								ResourceID:     template.ID.String(),
							},
							Action: "update",
						},
					},
				})
				if err != nil {
					return xerrors.Errorf("auth check: %w", err)
				}
				// We don't have template admin privileges.
				if !resp[key] {
					versionID = template.ActiveVersionID
				}
			}

			buildParameters, err := prepStartWorkspace(inv, client, prepStartWorkspaceArgs{
				Action:            WorkspaceRestart,
				TemplateVersionID: versionID,

				LastBuildParameters: lastBuildParameters,

				PromptBuildOptions: parameterFlags.promptBuildOptions,
				BuildOptions:       buildOptions,
			})
			if err != nil {
				return err
			}

			_, err = cliui.Prompt(inv, cliui.PromptOptions{
				Text:      "Confirm restart workspace?",
				IsConfirm: true,
			})
			if err != nil {
				return err
			}

			build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, codersdk.CreateWorkspaceBuildRequest{
				Transition: codersdk.WorkspaceTransitionStop,
			})
			if err != nil {
				return err
			}
			err = cliui.WorkspaceBuild(ctx, out, client, build.ID)
			if err != nil {
				return err
			}

			build, err = client.CreateWorkspaceBuild(ctx, workspace.ID, codersdk.CreateWorkspaceBuildRequest{
				Transition:          codersdk.WorkspaceTransitionStart,
				RichParameterValues: buildParameters,
				TemplateVersionID:   versionID,
			})
			if err != nil {
				return err
			}
			err = cliui.WorkspaceBuild(ctx, out, client, build.ID)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(out,
				"\nThe %s workspace has been restarted at %s!\n",
				pretty.Sprint(cliui.DefaultStyles.Keyword, workspace.Name), cliui.Timestamp(time.Now()),
			)
			return nil
		},
	}
	return cmd
}
