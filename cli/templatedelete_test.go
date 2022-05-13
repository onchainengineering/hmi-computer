package cli_test

import (
	"context"
	"testing"

	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/pty/ptytest"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestTemplateDelete(t *testing.T) {
	t.Parallel()

	t.Run("Ok", func(t *testing.T) {
		t.Parallel()

		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		_ = coderdtest.NewProvisionerDaemon(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		_ = coderdtest.AwaitTemplateVersionJob(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		cmd, root := clitest.New(t, "templates", "delete", template.Name)
		clitest.SetupConfig(t, client, root)
		require.NoError(t, cmd.Execute())

		template, err := client.Template(context.Background(), template.ID)
		spew.Dump(template, err)
		require.Error(t, err, "template should not exist")
	})

	t.Run("Selector", func(t *testing.T) {
		t.Parallel()

		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		_ = coderdtest.NewProvisionerDaemon(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		_ = coderdtest.AwaitTemplateVersionJob(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		cmd, root := clitest.New(t, "templates", "delete")
		clitest.SetupConfig(t, client, root)

		pty := ptytest.New(t)
		cmd.SetIn(pty.Input())
		cmd.SetOut(pty.Output())

		execDone := make(chan error)
		go func() {
			execDone <- cmd.Execute()
		}()

		pty.WriteLine("docker-local")
		require.NoError(t, <-execDone)

		_, err := client.Template(context.Background(), template.ID)
		require.Error(t, err, "template should not exist")
	})
}
