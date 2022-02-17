//go:build !windows
// +build !windows

package pty_test

import (
	"os/exec"
	"testing"

	"github.com/coder/coder/pty/ptytest"
)

func TestStart(t *testing.T) {
	t.Parallel()
	t.Run("Echo", func(t *testing.T) {
		t.Parallel()
		pty := ptytest.Start(t, exec.Command("echo", "test"))
		pty.ExpectMatch("test")
	})
}
