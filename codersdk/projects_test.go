package codersdk_test

import (
	"archive/tar"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/database"
)

func TestProjects(t *testing.T) {
	t.Parallel()

	t.Run("UnauthenticatedList", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_, err := client.Projects(context.Background(), "")
		require.Error(t, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		_, err := client.Projects(context.Background(), "")
		require.NoError(t, err)
		_, err = client.Projects(context.Background(), user.Organization)
		require.NoError(t, err)
	})

	t.Run("UnauthenticatedCreate", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_, err := client.CreateProject(context.Background(), "", coderd.CreateProjectRequest{})
		require.Error(t, err)
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		_, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "bananas",
			Provisioner: database.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
	})

	t.Run("UnauthenticatedSingle", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_, err := client.Project(context.Background(), "wow", "example")
		require.Error(t, err)
	})

	t.Run("Single", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		_, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "bananas",
			Provisioner: database.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		_, err = client.Project(context.Background(), user.Organization, "bananas")
		require.NoError(t, err)
	})

	t.Run("UnauthenticatedHistory", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_, err := client.ProjectVersions(context.Background(), "org", "project")
		require.Error(t, err)
	})

	t.Run("History", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "bananas",
			Provisioner: database.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		_, err = client.ProjectVersions(context.Background(), user.Organization, project.Name)
		require.NoError(t, err)
	})

	t.Run("CreateHistoryUnauthenticated", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_, err := client.CreateProjectVersion(context.Background(), "org", "project", coderd.CreateProjectVersionRequest{
			StorageMethod: database.ProjectStorageMethodInlineArchive,
			StorageSource: []byte{},
		})
		require.Error(t, err)
	})

	t.Run("CreateHistory", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "bananas",
			Provisioner: database.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		err = writer.WriteHeader(&tar.Header{
			Name: "file",
			Size: 1 << 10,
		})
		require.NoError(t, err)
		_, err = writer.Write(make([]byte, 1<<10))
		require.NoError(t, err)
		version, err := client.CreateProjectVersion(context.Background(), user.Organization, project.Name, coderd.CreateProjectVersionRequest{
			StorageMethod: database.ProjectStorageMethodInlineArchive,
			StorageSource: buffer.Bytes(),
		})
		require.NoError(t, err)

		_, err = client.ProjectVersion(context.Background(), user.Organization, project.Name, version.Name)
		require.NoError(t, err)
	})

	t.Run("Parameters", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "someproject",
			Provisioner: database.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		params, err := client.ProjectParameters(context.Background(), user.Organization, project.Name)
		require.NoError(t, err)
		require.NotNil(t, params)
		require.Len(t, params, 0)
	})

	t.Run("CreateParameter", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		project, err := client.CreateProject(context.Background(), user.Organization, coderd.CreateProjectRequest{
			Name:        "someproject",
			Provisioner: database.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		param, err := client.CreateProjectParameter(context.Background(), user.Organization, project.Name, coderd.CreateParameterValueRequest{
			Name:              "hi",
			SourceValue:       "tomato",
			SourceScheme:      database.ParameterSourceSchemeData,
			DestinationScheme: database.ParameterDestinationSchemeEnvironmentVariable,
			DestinationValue:  "moo",
		})
		require.NoError(t, err)
		require.Equal(t, "hi", param.Name)
	})

	t.Run("HistoryParametersError", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		user := coderdtest.CreateInitialUser(t, client)
		_, err := client.ProjectVersionParameters(context.Background(), user.Organization, "nothing", "nope")
		require.Error(t, err)
	})
}
