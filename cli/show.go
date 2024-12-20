package cli

import (
	"golang.org/x/xerrors"

	"github.com/coder/serpent"
	"github.com/onchainengineering/hmi-computer/v2/cli/cliui"
	"github.com/onchainengineering/hmi-computer/v2/codersdk"
)

func (r *RootCmd) show() *serpent.Command {
	client := new(codersdk.Client)
	return &serpent.Command{
		Use:   "show <workspace>",
		Short: "Display details of a workspace's resources and agents",
		Middleware: serpent.Chain(
			serpent.RequireNArgs(1),
			r.InitClient(client),
		),
		Handler: func(inv *serpent.Invocation) error {
			buildInfo, err := client.BuildInfo(inv.Context())
			if err != nil {
				return xerrors.Errorf("get server version: %w", err)
			}
			workspace, err := namedWorkspace(inv.Context(), client, inv.Args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}
			return cliui.WorkspaceResources(inv.Stdout, workspace.LatestBuild.Resources, cliui.WorkspaceResourcesOptions{
				WorkspaceName: workspace.Name,
				ServerVersion: buildInfo.Version,
			})
		},
	}
}
