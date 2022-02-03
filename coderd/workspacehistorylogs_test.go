package coderd_test

import (
	"archive/tar"
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/database"
)

func TestWorkspaceHistoryLogs(t *testing.T) {
	t.Parallel()

	setupProjectAndWorkspace := func(t *testing.T, client *codersdk.Client, user coderd.CreateInitialUserRequest) (coderd.Project, coderd.Workspace) {
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "banana",
			Provisioner: database.ProvisionerTypeTerraform,
		})
		require.NoError(t, err)
		workspace, err := client.CreateWorkspace(context.Background(), "", coderd.CreateWorkspaceRequest{
			Name:      "example",
			ProjectID: project.ID,
		})
		require.NoError(t, err)
		return project, workspace
	}

	setupProjectHistory := func(t *testing.T, client *codersdk.Client, user coderd.CreateInitialUserRequest, project coderd.Project, files map[string]string) coderd.ProjectHistory {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		for path, content := range files {
			err := writer.WriteHeader(&tar.Header{
				Name: path,
				Size: int64(len(content)),
			})
			require.NoError(t, err)
			_, err = writer.Write([]byte(content))
			require.NoError(t, err)
		}
		err := writer.Flush()
		require.NoError(t, err)

		projectHistory, err := client.CreateProjectHistory(context.Background(), user.Organization, project.Name, coderd.CreateProjectHistoryRequest{
			StorageMethod: database.ProjectStorageMethodInlineArchive,
			StorageSource: buffer.Bytes(),
		})
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			hist, err := client.ProjectHistory(context.Background(), user.Organization, project.Name, projectHistory.Name)
			require.NoError(t, err)
			return hist.Import.Status.Completed()
		}, 15*time.Second, 50*time.Millisecond)
		return projectHistory
	}

	server := coderdtest.New(t)
	user := server.RandomInitialUser(t)
	_ = server.AddProvisionerd(t)
	project, workspace := setupProjectAndWorkspace(t, server.Client, user)
	projectHistory := setupProjectHistory(t, server.Client, user, project, map[string]string{
		"main.tf": `resource "null_resource" "test" {}`,
	})

	workspaceHistory, err := server.Client.CreateWorkspaceHistory(context.Background(), "", workspace.Name, coderd.CreateWorkspaceHistoryRequest{
		ProjectHistoryID: projectHistory.ID,
		Transition:       database.WorkspaceTransitionCreate,
	})
	require.NoError(t, err)

	now := database.Now()
	logChan, err := server.Client.FollowWorkspaceHistoryLogsAfter(context.Background(), "", workspace.Name, workspaceHistory.Name, now)
	require.NoError(t, err)

	for {
		log, more := <-logChan
		if !more {
			break
		}
		t.Logf("Output: %s", log.Output)
	}

	_, err = server.Client.WorkspaceHistoryLogs(context.Background(), "", workspace.Name, workspaceHistory.Name)
	require.NoError(t, err)
}
