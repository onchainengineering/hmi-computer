package cli

import (
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/codersdk"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func workspaceShow() *cobra.Command {
	return &cobra.Command{
		Use:  "show",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}
			workspace, err := client.WorkspaceByName(cmd.Context(), codersdk.Me, args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}
			resources, err := client.WorkspaceResourcesByBuild(cmd.Context(), workspace.LatestBuild.ID)
			if err != nil {
				return xerrors.Errorf("get workspace resources: %w", err)
			}
			return cliui.WorkspaceResources(cmd.OutOrStdout(), resources, cliui.WorkspaceResourcesOptions{
				WorkspaceName: workspace.Name,
			})
		},
	}
}
