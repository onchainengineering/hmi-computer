package clitest_test

import (
	"testing"

	"go.uber.org/goleak"

	"github.com/onchainengineering/hmi-computer/v2/cli/clitest"
	"github.com/onchainengineering/hmi-computer/v2/coderd/coderdtest"
	"github.com/onchainengineering/hmi-computer/v2/pty/ptytest"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCli(t *testing.T) {
	t.Parallel()
	clitest.CreateTemplateVersionSource(t, nil)
	client := coderdtest.New(t, nil)
	i, config := clitest.New(t)
	clitest.SetupConfig(t, client, config)
	pty := ptytest.New(t).Attach(i)
	clitest.Start(t, i)
	pty.ExpectMatch("coder")
}
