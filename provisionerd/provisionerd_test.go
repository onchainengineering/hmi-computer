package provisionerd_test

import (
	"archive/tar"
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"storj.io/drpc/drpcconn"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/database"
	"github.com/coder/coder/provisioner/terraform"
	"github.com/coder/coder/provisionerd"
	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/provisionersdk/proto"
)

func TestProvisionerd(t *testing.T) {
	t.Parallel()

	setupProjectAndWorkspace := func(t *testing.T, client *codersdk.Client, user coderd.CreateInitialUserRequest) (coderd.Project, coderd.Workspace) {
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "banana",
			Provisioner: database.ProvisionerTypeTerraform,
		})
		require.NoError(t, err)
		workspace, err := client.CreateWorkspace(context.Background(), "", coderd.CreateWorkspaceRequest{
			Name:      "hiii",
			ProjectID: project.ID,
		})
		require.NoError(t, err)
		return project, workspace
	}

	setupProjectVersion := func(t *testing.T, client *codersdk.Client, user coderd.CreateInitialUserRequest, project coderd.Project) coderd.ProjectHistory {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		content := `resource "null_resource" "hi" {}`
		err := writer.WriteHeader(&tar.Header{
			Name: "main.tf",
			Size: int64(len(content)),
		})
		require.NoError(t, err)
		_, err = writer.Write([]byte(content))
		require.NoError(t, err)
		projectHistory, err := client.CreateProjectHistory(context.Background(), user.Organization, project.Name, coderd.CreateProjectVersionRequest{
			StorageMethod: database.ProjectStorageMethodInlineArchive,
			StorageSource: buffer.Bytes(),
		})
		require.NoError(t, err)
		return projectHistory
	}

	t.Run("InstantClose", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		api := provisionerd.New(server.Client.ProvisionerDaemonClient, provisionerd.Provisioners{}, &provisionerd.Options{
			Logger: slogtest.Make(t, nil),
		})
		defer api.Close()
	})

	t.Run("ProcessJob", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, workspace := setupProjectAndWorkspace(t, server.Client, user)
		projectVersion := setupProjectVersion(t, server.Client, user, project)
		_, err := server.Client.CreateWorkspaceHistory(context.Background(), "", workspace.Name, coderd.CreateWorkspaceHistoryRequest{
			ProjectHistoryID: projectVersion.ID,
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.NoError(t, err)

		clientPipe, serverPipe := provisionersdk.TransportPipe()
		ctx, cancelFunc := context.WithCancel(context.Background())
		t.Cleanup(func() {
			_ = clientPipe.Close()
			_ = serverPipe.Close()
			cancelFunc()
		})
		go func() {
			err := terraform.Serve(ctx, &terraform.ServeOptions{
				ServeOptions: &provisionersdk.ServeOptions{
					Transport: serverPipe,
				},
			})
			require.NoError(t, err)
		}()

		api := provisionerd.New(server.Client.ProvisionerDaemonClient, provisionerd.Provisioners{
			string(database.ProvisionerTypeTerraform): proto.NewDRPCProvisionerClient(drpcconn.New(clientPipe)),
		}, &provisionerd.Options{
			Logger:          slogtest.Make(t, nil).Leveled(slog.LevelDebug),
			AcquireInterval: 50 * time.Millisecond,
			WorkDirectory:   t.TempDir(),
		})
		defer api.Close()
		time.Sleep(time.Millisecond * 1500)
	})
}
