package cli_test

import (
	"context"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/coderd/database/postgres"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/pty/ptytest"
)

// nolint:paralleltest
func TestResetPassword(t *testing.T) {
	// postgres.Open() seems to be creating race conditions when run in parallel.
	// t.Parallel()

	if runtime.GOOS != "linux" || testing.Short() {
		// Skip on non-Linux because it spawns a PostgreSQL instance.
		t.SkipNow()
	}

	const email = "some@one.com"
	const username = "example"
	const oldPassword = "password"
	const newPassword = "password2"

	// start postgres and coder server processes

	connectionURL, closeFunc, err := postgres.Open()
	require.NoError(t, err)
	defer closeFunc()
	ctx, cancelFunc := context.WithCancel(context.Background())
	serverDone := make(chan struct{})
	serverCmd, cfg := clitest.New(t, "server", "--address", ":0", "--postgres-url", connectionURL)
	go func() {
		defer close(serverDone)
		err = serverCmd.ExecuteContext(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()
	var client *codersdk.Client
	require.Eventually(t, func() bool {
		rawURL, err := cfg.URL().Read()
		if err != nil {
			return false
		}
		accessURL, err := url.Parse(rawURL)
		require.NoError(t, err)
		client = codersdk.New(accessURL)
		return true
	}, 15*time.Second, 25*time.Millisecond)
	_, err = client.CreateFirstUser(ctx, codersdk.CreateFirstUserRequest{
		Email:            email,
		Username:         username,
		Password:         oldPassword,
		OrganizationName: "example",
	})
	require.NoError(t, err)

	// reset the password

	resetCmd, cmdCfg := clitest.New(t, "reset-password", "--postgres-url", connectionURL, username)
	clitest.SetupConfig(t, client, cmdCfg)
	cmdDone := make(chan struct{})
	pty := ptytest.New(t)
	resetCmd.SetIn(pty.Input())
	resetCmd.SetOut(pty.Output())
	go func() {
		defer close(cmdDone)
		err = resetCmd.Execute()
		require.NoError(t, err)
	}()

	matches := []struct {
		output string
		input  string
	}{
		{"Enter new", newPassword},
		{"Confirm", newPassword},
	}
	for _, match := range matches {
		pty.ExpectMatch(match.output)
		pty.WriteLine(match.input)
	}
	<-cmdDone

	// now try logging in

	_, err = client.LoginWithPassword(ctx, codersdk.LoginWithPasswordRequest{
		Email:    email,
		Password: oldPassword,
	})
	require.Error(t, err)

	_, err = client.LoginWithPassword(ctx, codersdk.LoginWithPasswordRequest{
		Email:    email,
		Password: newPassword,
	})
	require.NoError(t, err)

	cancelFunc()
	<-serverDone
}
