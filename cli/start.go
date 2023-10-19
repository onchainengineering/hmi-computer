package cli

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/xerrors"

	"github.com/google/uuid"

	"github.com/coder/coder/v2/cli/clibase"
	"github.com/coder/coder/v2/cli/cliui"
	"github.com/coder/coder/v2/codersdk"
)

func (r *RootCmd) start() *clibase.Cmd {
	var parameterFlags workspaceParameterFlags

	client := new(codersdk.Client)
	cmd := &clibase.Cmd{
		Annotations: workspaceCommand,
		Use:         "start <workspace>",
		Short:       "Start a workspace",
		Middleware: clibase.Chain(
			clibase.RequireNArgs(1),
			r.InitClient(client),
		),
		Options: append(parameterFlags.cliBuildOptions(), cliui.SkipPromptOption()),
		Handler: func(inv *clibase.Invocation) error {
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
				return xerrors.Errorf("unable to parse build options: %w", err)
			}

			buildParameters, err := prepStartWorkspace(inv, client, prepStartWorkspaceArgs{
				Action:            WorkspaceStart,
				TemplateVersionID: workspace.LatestBuild.TemplateVersionID,

				LastBuildParameters: lastBuildParameters,

				PromptBuildOptions: parameterFlags.promptBuildOptions,
				BuildOptions:       buildOptions,
			})
			if err != nil {
				return err
			}

			req := codersdk.CreateWorkspaceBuildRequest{
				Transition:          codersdk.WorkspaceTransitionStart,
				RichParameterValues: buildParameters,
				TemplateVersionID:   workspace.LatestBuild.TemplateVersionID,
			}

			build, err := client.CreateWorkspaceBuild(inv.Context(), workspace.ID, req)
			// It's possible for a workspace build to fail due to the template requiring starting
			// workspaces with the active version.
			if cerr, ok := codersdk.AsError(err); ok && cerr.StatusCode() == http.StatusUnauthorized {
				template, err := client.Template(inv.Context(), workspace.TemplateID)
				if err != nil {
					return xerrors.Errorf("get template: %w", err)
				}

				buildParameters, err = prepStartWorkspace(inv, client, prepStartWorkspaceArgs{
					Action:            WorkspaceStart,
					TemplateVersionID: template.ActiveVersionID,

					LastBuildParameters: lastBuildParameters,

					PromptBuildOptions: parameterFlags.promptBuildOptions,
					BuildOptions:       buildOptions,
				})
				if err != nil {
					return err
				}

				req.RichParameterValues = buildParameters
				req.TemplateVersionID = template.ActiveVersionID
				build, err = client.CreateWorkspaceBuild(inv.Context(), workspace.ID, req)
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			err = cliui.WorkspaceBuild(inv.Context(), inv.Stdout, client, build.ID)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(
				inv.Stdout, "\nThe %s workspace has been started at %s!\n",
				cliui.Keyword(workspace.Name), cliui.Timestamp(time.Now()),
			)
			return nil
		},
	}
	return cmd
}

type prepStartWorkspaceArgs struct {
	Action            WorkspaceCLIAction
	TemplateVersionID uuid.UUID

	LastBuildParameters []codersdk.WorkspaceBuildParameter

	PromptBuildOptions bool
	BuildOptions       []codersdk.WorkspaceBuildParameter
}

func prepStartWorkspace(inv *clibase.Invocation, client *codersdk.Client, args prepStartWorkspaceArgs) ([]codersdk.WorkspaceBuildParameter, error) {
	ctx := inv.Context()

	templateVersion, err := client.TemplateVersion(ctx, args.TemplateVersionID)
	if err != nil {
		return nil, xerrors.Errorf("get template version: %w", err)
	}

	templateVersionParameters, err := client.TemplateVersionRichParameters(inv.Context(), templateVersion.ID)
	if err != nil {
		return nil, xerrors.Errorf("get template version rich parameters: %w", err)
	}

	resolver := new(ParameterResolver).
		WithLastBuildParameters(args.LastBuildParameters).
		WithPromptBuildOptions(args.PromptBuildOptions).
		WithBuildOptions(args.BuildOptions)
	return resolver.Resolve(inv, args.Action, templateVersionParameters)
}
