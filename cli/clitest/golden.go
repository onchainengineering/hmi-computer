package clitest

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/coderd/database/dbtestutil"

	"github.com/coder/coder/cli/clibase"

	"github.com/coder/coder/cli/config"
	"github.com/coder/coder/codersdk"

	"github.com/charmbracelet/lipgloss"
	"github.com/coder/coder/cli"
	"github.com/coder/coder/testutil"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

// UpdateGoldenFiles indicates golden files should be updated.
// To update the golden files:
// make update-golden-files
var UpdateGoldenFiles = flag.Bool("update", false, "update .golden files")

var timestampRegex = regexp.MustCompile(`(?i)\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(.\d+)?Z`)

type CommandHelpCase struct {
	Name string
	Cmd  []string
}

func DefaultCases() []CommandHelpCase {
	return []CommandHelpCase{
		{
			Name: "coder --help",
			Cmd:  []string{"--help"},
		},
		{
			Name: "coder server --help",
			Cmd:  []string{"server", "--help"},
		},
		{
			Name: "coder agent --help",
			Cmd:  []string{"agent", "--help"},
		},
		{
			Name: "coder list --output json",
			Cmd:  []string{"list", "--output", "json"},
		},
		{
			Name: "coder users list --output json",
			Cmd:  []string{"users", "list", "--output", "json"},
		},
	}
}

// TestCommandHelp will test the help output of the given commands
// using golden files.
//
//nolint:tparallel,paralleltest
func TestCommandHelp(t *testing.T, cmds []*clibase.Cmd, cases []CommandHelpCase) {
	ogColorProfile := lipgloss.ColorProfile()
	// ANSI256 escape codes are far easier for humans to parse in a diff,
	// but TrueColor is probably more popular with modern terminals.
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(ogColorProfile)
	})
	rootClient, replacements := prepareTestData(t)

	rootCmd := new(cli.RootCmd)
	root, err := rootCmd.Command(cmds)
	require.NoError(t, err)

ExtractCommandPathsLoop:
	for _, cp := range extractVisibleCommandPaths(nil, root.Children) {
		name := fmt.Sprintf("coder %s --help", strings.Join(cp, " "))
		cmd := append(cp, "--help")
		for _, tt := range cases {
			if tt.Name == name {
				continue ExtractCommandPathsLoop
			}
		}
		cases = append(cases, CommandHelpCase{Name: name, Cmd: cmd})
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			ctx := testutil.Context(t, testutil.WaitLong)

			var outBuf bytes.Buffer
			inv, cfg := New(t, tt.Cmd...)
			inv.Stderr = &outBuf
			inv.Stdout = &outBuf
			inv.Environ.Set("CODER_URL", rootClient.URL.String())
			inv.Environ.Set("CODER_SESSION_TOKEN", rootClient.SessionToken())
			inv.Environ.Set("CODER_CACHE_DIRECTORY", "~/.cache")

			SetupConfig(t, rootClient, cfg)

			StartWithWaiter(t, inv.WithContext(ctx)).RequireSuccess()

			actual := outBuf.Bytes()
			if len(actual) == 0 {
				t.Fatal("no output")
			}

			for k, v := range replacements {
				actual = bytes.ReplaceAll(actual, []byte(k), []byte(v))
			}

			actual = NormalizeGoldenFile(t, actual)
			goldenPath := filepath.Join("testdata", strings.Replace(tt.Name, " ", "_", -1)+".golden")
			if *UpdateGoldenFiles {
				t.Logf("update golden file for: %q: %s", tt.Name, goldenPath)
				err = os.WriteFile(goldenPath, actual, 0o600)
				require.NoError(t, err, "update golden file")
			}

			expected, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "read golden file, run \"make update-golden-files\" and commit the changes")

			expected = NormalizeGoldenFile(t, expected)
			require.Equal(
				t, string(expected), string(actual),
				"golden file mismatch: %s, run \"make update-golden-files\", verify and commit the changes",
				goldenPath,
			)
		})
	}
}

// NormalizeGoldenFile replaces any strings that are system or timing dependent
// with a placeholder so that the golden files can be compared with a simple
// equality check.
func NormalizeGoldenFile(t *testing.T, byt []byte) []byte {
	// Replace any timestamps with a placeholder.
	byt = timestampRegex.ReplaceAll(byt, []byte("[timestamp]"))

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	configDir := config.DefaultDir()
	byt = bytes.ReplaceAll(byt, []byte(configDir), []byte("~/.config/coderv2"))

	byt = bytes.ReplaceAll(byt, []byte(codersdk.DefaultCacheDir()), []byte("[cache dir]"))

	// The home directory changes depending on the test environment.
	byt = bytes.ReplaceAll(byt, []byte(homeDir), []byte("~"))
	for _, r := range []struct {
		old string
		new string
	}{
		{"\r\n", "\n"},
		{`~\.cache\coder`, "~/.cache/coder"},
		{`C:\Users\RUNNER~1\AppData\Local\Temp`, "/tmp"},
		{os.TempDir(), "/tmp"},
	} {
		byt = bytes.ReplaceAll(byt, []byte(r.old), []byte(r.new))
	}
	return byt
}

func extractVisibleCommandPaths(cmdPath []string, cmds []*clibase.Cmd) [][]string {
	var cmdPaths [][]string
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		cmdPath := append(cmdPath, c.Name())
		cmdPaths = append(cmdPaths, cmdPath)
		cmdPaths = append(cmdPaths, extractVisibleCommandPaths(cmdPath, c.Children)...)
	}
	return cmdPaths
}

func prepareTestData(t *testing.T) (*codersdk.Client, map[string]string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	db, pubsub := dbtestutil.NewDB(t)
	rootClient := coderdtest.New(t, &coderdtest.Options{
		Database:                 db,
		Pubsub:                   pubsub,
		IncludeProvisionerDaemon: true,
	})
	firstUser := coderdtest.CreateFirstUser(t, rootClient)
	secondUser, err := rootClient.CreateUser(ctx, codersdk.CreateUserRequest{
		Email:          "testuser2@coder.com",
		Username:       "testuser2",
		Password:       coderdtest.FirstUserParams.Password,
		OrganizationID: firstUser.OrganizationID,
	})
	require.NoError(t, err)
	version := coderdtest.CreateTemplateVersion(t, rootClient, firstUser.OrganizationID, nil)
	version = coderdtest.AwaitTemplateVersionJob(t, rootClient, version.ID)
	template := coderdtest.CreateTemplate(t, rootClient, firstUser.OrganizationID, version.ID, func(req *codersdk.CreateTemplateRequest) {
		req.Name = "test-template"
	})
	workspace := coderdtest.CreateWorkspace(t, rootClient, firstUser.OrganizationID, template.ID, func(req *codersdk.CreateWorkspaceRequest) {
		req.Name = "test-workspace"
	})
	workspaceBuild := coderdtest.AwaitWorkspaceBuildJob(t, rootClient, workspace.LatestBuild.ID)

	replacements := map[string]string{
		firstUser.UserID.String():            "[first user ID]",
		secondUser.ID.String():               "[second user ID]",
		firstUser.OrganizationID.String():    "[first org ID]",
		version.ID.String():                  "[version ID]",
		version.Name:                         "[version name]",
		version.Job.ID.String():              "[version job ID]",
		version.Job.FileID.String():          "[version file ID]",
		version.Job.WorkerID.String():        "[version worker ID]",
		template.ID.String():                 "[template ID]",
		workspace.ID.String():                "[workspace ID]",
		workspaceBuild.ID.String():           "[workspace build ID]",
		workspaceBuild.Job.ID.String():       "[workspace build job ID]",
		workspaceBuild.Job.FileID.String():   "[workspace build file ID]",
		workspaceBuild.Job.WorkerID.String(): "[workspace build worker ID]",
	}

	return rootClient, replacements
}
