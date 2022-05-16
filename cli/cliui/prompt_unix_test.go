//go:build linux || darwin

package cliui_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/pty/ptytest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestPasswordTerminalState(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		passwordHelper()
		return
	}
	t.Parallel()

	ptty := ptytest.New(t)

	cmd := exec.Command(os.Args[0], "-test.run=TestPasswordTerminalState")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	// connect the child process's stdio to the PTY directly, not via a pipe
	cmd.Stdin = ptty.PTY.PTYFile()
	cmd.Stdout = ptty.PTY.PTYFile()
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	require.NoError(t, err)
	defer cmd.Process.Kill()

	ptty.ExpectMatch("Password: ")
	time.Sleep(10 * time.Millisecond) // wait for child process to turn off echo and start reading input

	termios, err := unix.IoctlGetTermios(int(ptty.PTY.PTYFile().Fd()), unix.TCGETS)
	require.NoError(t, err)
	require.Zero(t, termios.Lflag&unix.ECHO, "echo is on while reading password")

	cmd.Process.Signal(os.Interrupt)
	_, err = cmd.Process.Wait()
	require.NoError(t, err)

	termios, err = unix.IoctlGetTermios(int(ptty.PTY.PTYFile().Fd()), unix.TCGETS)
	require.NoError(t, err)
	require.NotZero(t, termios.Lflag&unix.ECHO, "echo is off after reading password")
}

func passwordHelper() {
	cmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			cliui.Prompt(cmd, cliui.PromptOptions{
				Text:   "Password:",
				Secret: true,
			})
		},
	}
	cmd.ExecuteContext(context.Background())
}
