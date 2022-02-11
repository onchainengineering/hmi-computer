package cli_test

import (
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/database"
	"github.com/coder/coder/provisioner/echo"
	"github.com/coder/coder/provisionersdk/proto"
	"github.com/stretchr/testify/require"
)

func TestProjectCreate(t *testing.T) {
	t.Parallel()
	t.Run("NoParameters", func(t *testing.T) {
		t.Parallel()
		console, err := expect.NewConsole(expect.WithStdout(clitest.StdoutLogs(t)))
		require.NoError(t, err)
		client := coderdtest.New(t)
		_ = coderdtest.NewProvisionerDaemon(t, client)
		source := clitest.CreateProjectVersionSource(t, &echo.Responses{
			Parse:     echo.ParseComplete,
			Provision: echo.ProvisionComplete,
		})
		cmd, root := clitest.New(t, "projects", "create", "--directory", source, "--provisioner", string(database.ProvisionerTypeEcho))
		_ = clitest.CreateInitialUser(t, client, root)
		cmd.SetIn(console.Tty())
		cmd.SetOut(console.Tty())
		closeChan := make(chan struct{})
		go func() {
			err := cmd.Execute()
			require.NoError(t, err)
			close(closeChan)
		}()

		matches := []string{
			"organization?", "y",
			"name?", "test-project",
			"project?", "y",
			"created!", "n",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			_, err = console.ExpectString(match)
			require.NoError(t, err)
			_, err = console.SendLine(value)
			require.NoError(t, err)
		}
		<-closeChan
	})

	t.Run("Parameter", func(t *testing.T) {
		t.Parallel()
		console, err := expect.NewConsole(expect.WithStdout(clitest.StdoutLogs(t)))
		require.NoError(t, err)
		client := coderdtest.New(t)
		_ = coderdtest.NewProvisionerDaemon(t, client)
		source := clitest.CreateProjectVersionSource(t, &echo.Responses{
			Parse: []*proto.Parse_Response{{
				Type: &proto.Parse_Response_Complete{
					Complete: &proto.Parse_Complete{
						ParameterSchemas: []*proto.ParameterSchema{{
							Name: "somevar",
							DefaultDestination: &proto.ParameterDestination{
								Scheme: proto.ParameterDestination_PROVISIONER_VARIABLE,
							},
						}},
					},
				},
			}},
			Provision: echo.ProvisionComplete,
		})
		cmd, root := clitest.New(t, "projects", "create", "--directory", source, "--provisioner", string(database.ProvisionerTypeEcho))
		_ = clitest.CreateInitialUser(t, client, root)
		cmd.SetIn(console.Tty())
		cmd.SetOut(console.Tty())
		closeChan := make(chan struct{})
		go func() {
			err := cmd.Execute()
			require.NoError(t, err)
			close(closeChan)
		}()

		matches := []string{
			"organization?", "y",
			"name?", "test-project",
			"somevar:", "value",
			"project?", "y",
			"created!", "n",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			_, err = console.ExpectString(match)
			require.NoError(t, err)
			_, err = console.SendLine(value)
			require.NoError(t, err)
		}
		<-closeChan
	})
}
