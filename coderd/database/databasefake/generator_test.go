package databasefake_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/databasefake"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("APIKey", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp, _ := databasefake.GenerateAPIKey(t, db, database.APIKey{})
		require.Equal(t, exp, must(db.GetAPIKeyByID(context.Background(), exp.ID)))
	})

	t.Run("File", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.File{})
		require.Equal(t, exp, must(db.GetFileByID(context.Background(), exp.ID)))
	})

	t.Run("UserLink", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.UserLink{})
		require.Equal(t, exp, must(db.GetUserLinkByLinkedID(context.Background(), exp.LinkedID)))
	})

	t.Run("WorkspaceResource", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.WorkspaceResource{})
		require.Equal(t, exp, must(db.GetWorkspaceResourceByID(context.Background(), exp.ID)))
	})

	t.Run("Job", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.ProvisionerJob{})
		require.Equal(t, exp, must(db.GetProvisionerJobByID(context.Background(), exp.ID)))
	})

	t.Run("Group", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.Group{})
		require.Equal(t, exp, must(db.GetGroupByID(context.Background(), exp.ID)))
	})

	t.Run("Organization", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.Organization{})
		require.Equal(t, exp, must(db.GetOrganizationByID(context.Background(), exp.ID)))
	})

	t.Run("Workspace", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.Workspace{})
		require.Equal(t, exp, must(db.GetWorkspaceByID(context.Background(), exp.ID)))
	})

	t.Run("Template", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.Template{})
		require.Equal(t, exp, must(db.GetTemplateByID(context.Background(), exp.ID)))
	})

	t.Run("TemplateVersion", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.TemplateVersion{})
		require.Equal(t, exp, must(db.GetTemplateVersionByID(context.Background(), exp.ID)))
	})

	t.Run("WorkspaceBuild", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.WorkspaceBuild{})
		require.Equal(t, exp, must(db.GetWorkspaceBuildByID(context.Background(), exp.ID)))
	})

	t.Run("User", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		exp := databasefake.Generate(t, db, database.User{})
		require.Equal(t, exp, must(db.GetUserByID(context.Background(), exp.ID)))
	})
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
