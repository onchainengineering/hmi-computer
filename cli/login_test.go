package cli_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/pty/ptytest"
)

func TestLogin(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip()
		return
	}

	t.Run("InitialUserNoTTY", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		root, _ := clitest.New(t, "login", client.URL.String())
		err := root.Execute()
		require.Error(t, err)
	})

	t.Run("InitialUserTTY", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		// The --force-tty flag is required on Windows, because the `isatty` library does not
		// accurately detect Windows ptys when they are not attached to a process:
		// https://github.com/mattn/go-isatty/issues/59
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", client.URL.String(), "--force-tty")
		pty := ptytest.New(t)
		root.SetIn(pty.Input().Reader)
		root.SetOut(pty.Output())
		go func() {
			defer close(doneChan)
			err := root.Execute()
			require.NoError(t, err)
		}()

		matches := []string{
			"first user?", "yes",
			"username", "testuser",
			"email", "user@coder.com",
			"password", "password",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			pty.ExpectMatch(match)
			pty.WriteLine(value)
		}
		pty.ExpectMatch("Welcome to Coder")
		<-doneChan
	})

	t.Run("ExistingUserValidTokenTTY", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)

		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", client.URL.String(), "--force-tty", "--no-open")
		pty := ptytest.New(t)
		root.SetIn(pty.Input().Reader)
		root.SetOut(pty.Output())
		go func() {
			defer close(doneChan)
			err := root.Execute()
			require.NoError(t, err)
		}()

		pty.ExpectMatch("Paste your token here:")
		pty.WriteLine(client.SessionToken)
		pty.ExpectMatch("Welcome to Coder")
		<-doneChan
	})

	t.Run("ExistingUserInvalidTokenTTY", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)

		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", client.URL.String(), "--force-tty", "--no-open")
		pty := ptytest.New(t)
		root.SetIn(pty.Input().Reader)
		root.SetOut(pty.Output())
		go func() {
			defer close(doneChan)
			err := root.ExecuteContext(ctx)
			// An error is expected in this case, since the login wasn't successful:
			require.Error(t, err)
		}()

		pty.ExpectMatch("Paste your token here:")
		pty.WriteLine("an-invalid-token")
		pty.ExpectMatch("That's not a valid token!")
		cancelFunc()
		<-doneChan
	})
}
