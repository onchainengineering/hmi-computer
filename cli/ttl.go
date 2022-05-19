package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/codersdk"
)

const ttlDescriptionLong = `To have your workspace stop automatically after a configurable interval has passed.
Minimum TTL is 1 minute.
`

func ttl() *cobra.Command {
	ttlCmd := &cobra.Command{
		Annotations: workspaceCommand,
		Use:         "ttl enable <workspace>",
		Short:       "schedule a workspace to automatically stop after a configurable interval",
		Long:        ttlDescriptionLong,
		Example:     "coder ttl enable my-workspace 8h30m",
	}

	ttlCmd.AddCommand(ttlShow())
	ttlCmd.AddCommand(ttlEnable())
	ttlCmd.AddCommand(ttlDisable())

	return ttlCmd
}

func ttlShow() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "show <workspace_name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}
			organization, err := currentOrganization(cmd, client)
			if err != nil {
				return err
			}

			workspace, err := client.WorkspaceByOwnerAndName(cmd.Context(), organization.ID, codersdk.Me, args[0])
			if err != nil {
				return err
			}

			if workspace.TTL == nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "not enabled\n")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", workspace.TTL)

			return nil
		},
	}
	return cmd
}

func ttlEnable() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "enable <workspace_name> <ttl>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}
			organization, err := currentOrganization(cmd, client)
			if err != nil {
				return err
			}

			workspace, err := client.WorkspaceByOwnerAndName(cmd.Context(), organization.ID, codersdk.Me, args[0])
			if err != nil {
				return err
			}

			ttl, err := time.ParseDuration(args[1])
			if err != nil {
				return err
			}

			truncated := ttl.Truncate(time.Minute)

			if truncated == 0 {
				return xerrors.Errorf("ttl must be at least 1m")
			}

			if truncated != ttl {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "warning: ttl rounded down to %s", truncated)
			}

			err = client.UpdateWorkspaceTTL(cmd.Context(), workspace.ID, codersdk.UpdateWorkspaceTTLRequest{
				TTL: &truncated,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func ttlDisable() *cobra.Command {
	return &cobra.Command{
		Use:  "disable <workspace_name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient(cmd)
			if err != nil {
				return err
			}
			organization, err := currentOrganization(cmd, client)
			if err != nil {
				return err
			}

			workspace, err := client.WorkspaceByOwnerAndName(cmd.Context(), organization.ID, codersdk.Me, args[0])
			if err != nil {
				return err
			}

			err = client.UpdateWorkspaceTTL(cmd.Context(), workspace.ID, codersdk.UpdateWorkspaceTTLRequest{
				TTL: nil,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nThe %s workspace will no longer automatically stop.\n\n", workspace.Name)

			return nil
		},
	}
}
