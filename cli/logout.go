package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func logout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the local authenticated session",
		RunE: func(cmd *cobra.Command, args []string) error {
			var isLoggedOut bool

			config := createConfig(cmd)

			err := config.URL().Delete()
			if err != nil {
				// Only throw error if the URL configuration file is present,
				// otherwise the user is already logged out, and we proceed
				if !os.IsNotExist(err) {
					return xerrors.Errorf("remove URL file: %w", err)
				}
				isLoggedOut = true
			}

			err = config.Session().Delete()
			if err != nil {
				// Only throw error if the session configuration file is present,
				// otherwise the user is already logged out, and we proceed
				if !os.IsNotExist(err) {
					return xerrors.Errorf("remove session file: %w", err)
				}
				isLoggedOut = true
			}

			err = config.Organization().Delete()
			// If the organization configuration file is absent, we still proceed
			if err != nil && !os.IsNotExist(err) {
				return xerrors.Errorf("remove organization file: %w", err)
			}

			// If the user was already logged out, we show them a message
			if isLoggedOut {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), notLoggedInMessage+"\n")
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), caret+"Successfully logged out.\n")
			return nil
		},
	}
}
