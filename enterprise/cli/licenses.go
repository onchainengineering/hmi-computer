package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	agpl "github.com/coder/coder/cli"
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/codersdk"
)

var jwtRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

func licenses() *cobra.Command {
	cmd := &cobra.Command{
		Short:   "Add, remove, and list licenses",
		Use:     "licenses",
		Aliases: []string{"license"},
	}
	cmd.AddCommand(
		licenseAdd(),
	)
	return cmd
}

func licenseAdd() *cobra.Command {
	var (
		filename string
		license  string
		debug    bool
	)
	cmd := &cobra.Command{
		Use:   "add [-f file | -l license]",
		Short: "Add license to Coder deployment",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := agpl.CreateClient(cmd)
			if err != nil {
				return err
			}

			switch {
			case filename != "" && license != "":
				return xerrors.New("only one of (--file, --license) may be specified")

			case filename == "" && license == "":
				license, err = cliui.Prompt(cmd, cliui.PromptOptions{
					Text:     "Paste license:",
					Secret:   true,
					Validate: validJWT,
				})
				if err != nil {
					return err
				}

			case filename != "" && license == "":
				var r io.Reader
				if filename == "-" {
					r = cmd.InOrStdin()
				} else {
					f, err := os.Open(filename)
					if err != nil {
						return err
					}
					defer f.Close()
					r = f
				}
				lb, err := io.ReadAll(r)
				if err != nil {
					return err
				}
				license = string(lb)
			}
			license = strings.Trim(license, " \n")
			err = validJWT(license)
			if err != nil {
				return err
			}

			licResp, err := client.AddLicense(
				cmd.Context(),
				codersdk.AddLicenseRequest{License: license},
			)
			if err != nil {
				return err
			}
			if debug {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(licResp)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "License with ID %d added\n", licResp.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&filename, "file", "f", "", "Load license from file")
	cmd.Flags().StringVarP(&license, "license", "l", "", "License string")
	cmd.Flags().BoolVar(&debug, "debug", false, "Output license claims for debugging")
	return cmd
}

func validJWT(s string) error {
	if jwtRegexp.MatchString(s) {
		return nil
	}
	return xerrors.New("Invalid license")
}
