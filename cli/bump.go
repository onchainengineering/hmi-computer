package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/coder/coder/coderd/util/tz"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/codersdk"
)

const (
	bumpDescriptionLong = `Make your workspace stop at a certain point in the future.`
)

func bump() *cobra.Command {
	bumpCmd := &cobra.Command{
		Args:        cobra.RangeArgs(1, 2),
		Annotations: workspaceCommand,
		Use:         "bump <workspace-name> <duration from now>",
		Short:       "Make your workspace stop at a certain point in the future.",
		Long:        bumpDescriptionLong,
		Example:     "coder bump my-workspace 90m",
		RunE: func(cmd *cobra.Command, args []string) error {
			bumpDuration, err := tryParseDuration(args[1])
			if err != nil {
				return err
			}

			client, err := createClient(cmd)
			if err != nil {
				return xerrors.Errorf("create client: %w", err)
			}

			workspace, err := namedWorkspace(cmd, client, args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}

			loc, err := tz.TimezoneIANA()
			if err != nil {
				loc = time.UTC // best guess
			}

			if bumpDuration < 29*time.Minute {
				_, _ = fmt.Fprintf(
					cmd.OutOrStdout(),
					"Please specify a duration of at least 30 minutes.\n",
				)
				return nil
			}

			newDeadline := time.Now().In(loc).Add(bumpDuration)
			if err := client.PutExtendWorkspace(cmd.Context(), workspace.ID, codersdk.PutExtendWorkspaceRequest{
				Deadline: newDeadline,
			}); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(
				cmd.OutOrStdout(),
				"Workspace %q will now stop at %s on %s\n", workspace.Name,
				newDeadline.Format(timeFormat),
				newDeadline.Format(dateFormat),
			)
			return nil
		},
	}

	return bumpCmd
}

func tryParseDuration(raw string) (time.Duration, error) {
	// If the user input a raw number, assume minutes
	if isDigit(raw) {
		raw = raw + "m"
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return d, nil
}

func isDigit(s string) bool {
	return strings.IndexFunc(s, func(c rune) bool {
		return c < '0' || c > '9'
	}) == -1
}
