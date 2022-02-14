package clitest_test

import (
	"testing"

	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCli(t *testing.T) {
	t.Parallel()
	clitest.CreateProjectVersionSource(t, nil)
	client := coderdtest.New(t)
	cmd, config := clitest.New(t)
	clitest.SetupConfig(t, client, config)
	console := clitest.NewConsole(t, cmd)
	t.Cleanup(func () {
		console.Close()
	})
	go func() {
		err := cmd.Execute()
		require.NoError(t, err)
	}()
	_, err := console.ExpectString("coder")
	require.NoError(t, err)
}
