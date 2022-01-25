package coderd_test

import (
	"archive/tar"
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/database"
)

func TestWorkspaces(t *testing.T) {
	t.Parallel()

	t.Run("ListNone", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		_ = server.RandomInitialUser(t)
		workspaces, err := server.Client.Workspaces(context.Background(), "", "")
		require.NoError(t, err)
		require.Len(t, workspaces, 0)
	})

	setupProjectAndWorkspace := func(t *testing.T, client *codersdk.Client, user coderd.CreateInitialUserRequest) (coderd.Project, coderd.Workspace) {
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "banana",
			Provisioner: database.ProvisionerTypeTerraform,
		})
		require.NoError(t, err)
		workspace, err := client.CreateWorkspace(context.Background(), user.Organization, project.Name, coderd.CreateWorkspaceRequest{
			Name: "hiii",
		})
		require.NoError(t, err)
		return project, workspace
	}

	setupProjectVersion := func(t *testing.T, client *codersdk.Client, user coderd.CreateInitialUserRequest, project coderd.Project) coderd.ProjectHistory {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		err := writer.WriteHeader(&tar.Header{
			Name: "file",
			Size: 1 << 10,
		})
		require.NoError(t, err)
		_, err = writer.Write(make([]byte, 1<<10))
		require.NoError(t, err)
		projectHistory, err := client.CreateProjectHistory(context.Background(), user.Organization, project.Name, coderd.CreateProjectVersionRequest{
			StorageMethod: database.ProjectStorageMethodInlineArchive,
			StorageSource: buffer.Bytes(),
		})
		require.NoError(t, err)
		return projectHistory
	}

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		_, _ = setupProjectAndWorkspace(t, server.Client, user)
		workspaces, err := server.Client.Workspaces(context.Background(), "", "")
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
	})

	t.Run("ListNoneForProject", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, err := server.Client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "banana",
			Provisioner: database.ProvisionerTypeTerraform,
		})
		require.NoError(t, err)
		workspaces, err := server.Client.Workspaces(context.Background(), user.Organization, project.Name)
		require.NoError(t, err)
		require.Len(t, workspaces, 0)
	})

	t.Run("ListForProject", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, _ := setupProjectAndWorkspace(t, server.Client, user)
		workspaces, err := server.Client.Workspaces(context.Background(), user.Organization, project.Name)
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
	})

	t.Run("CreateInvalidInput", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, err := server.Client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "banana",
			Provisioner: database.ProvisionerTypeTerraform,
		})
		require.NoError(t, err)
		_, err = server.Client.CreateWorkspace(context.Background(), user.Organization, project.Name, coderd.CreateWorkspaceRequest{
			Name: "$$$",
		})
		require.Error(t, err)
	})

	t.Run("CreateAlreadyExists", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, workspace := setupProjectAndWorkspace(t, server.Client, user)
		_, err := server.Client.CreateWorkspace(context.Background(), user.Organization, project.Name, coderd.CreateWorkspaceRequest{
			Name: workspace.Name,
		})
		require.Error(t, err)
	})

	t.Run("Single", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		_, workspace := setupProjectAndWorkspace(t, server.Client, user)
		_, err := server.Client.Workspace(context.Background(), "", workspace.Name)
		require.NoError(t, err)
	})

	t.Run("AllHistory", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, workspace := setupProjectAndWorkspace(t, server.Client, user)
		history, err := server.Client.WorkspaceHistory(context.Background(), "", workspace.Name)
		require.NoError(t, err)
		require.Len(t, history, 0)
		projectVersion := setupProjectVersion(t, server.Client, user, project)
		_, err = server.Client.CreateWorkspaceVersion(context.Background(), "", workspace.Name, coderd.CreateWorkspaceBuildRequest{
			ProjectHistoryID: projectVersion.ID,
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.NoError(t, err)
		history, err = server.Client.WorkspaceHistory(context.Background(), "", workspace.Name)
		require.NoError(t, err)
		require.Len(t, history, 1)
	})

	t.Run("LatestHistory", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, workspace := setupProjectAndWorkspace(t, server.Client, user)
		_, err := server.Client.LatestWorkspaceHistory(context.Background(), "", workspace.Name)
		require.Error(t, err)
		projectVersion := setupProjectVersion(t, server.Client, user, project)
		_, err = server.Client.CreateWorkspaceVersion(context.Background(), "", workspace.Name, coderd.CreateWorkspaceBuildRequest{
			ProjectHistoryID: projectVersion.ID,
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.NoError(t, err)
		_, err = server.Client.LatestWorkspaceHistory(context.Background(), "", workspace.Name)
		require.NoError(t, err)
	})

	t.Run("CreateHistory", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, workspace := setupProjectAndWorkspace(t, server.Client, user)
		projectHistory := setupProjectVersion(t, server.Client, user, project)

		_, err := server.Client.CreateWorkspaceVersion(context.Background(), "", workspace.Name, coderd.CreateWorkspaceBuildRequest{
			ProjectHistoryID: projectHistory.ID,
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.NoError(t, err)
	})

	t.Run("CreateHistoryAlreadyInProgress", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		project, workspace := setupProjectAndWorkspace(t, server.Client, user)
		projectHistory := setupProjectVersion(t, server.Client, user, project)

		_, err := server.Client.CreateWorkspaceVersion(context.Background(), "", workspace.Name, coderd.CreateWorkspaceBuildRequest{
			ProjectHistoryID: projectHistory.ID,
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.NoError(t, err)

		_, err = server.Client.CreateWorkspaceVersion(context.Background(), "", workspace.Name, coderd.CreateWorkspaceBuildRequest{
			ProjectHistoryID: projectHistory.ID,
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.Error(t, err)
	})

	t.Run("CreateHistoryInvalidProjectVersion", func(t *testing.T) {
		t.Parallel()
		server := coderdtest.New(t)
		user := server.RandomInitialUser(t)
		_, workspace := setupProjectAndWorkspace(t, server.Client, user)

		_, err := server.Client.CreateWorkspaceVersion(context.Background(), "", workspace.Name, coderd.CreateWorkspaceBuildRequest{
			ProjectHistoryID: uuid.New(),
			Transition:       database.WorkspaceTransitionCreate,
		})
		require.Error(t, err)
	})
}
