package cli

import "github.com/spf13/cobra"

func tunnel() *cobra.Command {
	return &cobra.Command{
		Use: "tunnel",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
