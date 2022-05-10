//go:build linux

package database_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/stub"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/postgres"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestMigrate(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
		return
	}

	t.Run("Once", func(t *testing.T) {
		t.Parallel()

		db := testSQLDB(t)

		err := database.MigrateUp(db)
		require.NoError(t, err)
	})

	t.Run("Twice", func(t *testing.T) {
		t.Parallel()

		db := testSQLDB(t)

		err := database.MigrateUp(db)
		require.NoError(t, err)

		err = database.MigrateUp(db)
		require.NoError(t, err)
	})

	t.Run("UpDownUp", func(t *testing.T) {
		t.Parallel()

		db := testSQLDB(t)

		err := database.MigrateUp(db)
		require.NoError(t, err)

		err = database.MigrateDown(db)
		require.NoError(t, err)

		err = database.MigrateUp(db)
		require.NoError(t, err)
	})
}

func testSQLDB(t testing.TB) *sql.DB {
	t.Helper()

	connection, closeFn, err := postgres.Open()
	require.NoError(t, err)
	t.Cleanup(closeFn)

	db, err := sql.Open("postgres", connection)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestCheckLatestVersion(t *testing.T) {
	t.Parallel()

	type test struct {
		currentVersion   uint
		existingVersions []uint
		expectedResult   string
	}

	tests := []test{
		// successful cases
		{1, []uint{1}, ""},
		{3, []uint{1, 2, 3}, ""},
		{3, []uint{1, 3}, ""},

		// failure cases
		{1, []uint{1, 2}, "current version is 1, but later version 2 exists"},
		{2, []uint{1, 2, 3}, "current version is 2, but later version 3 exists"},
		{4, []uint{1, 2, 3}, "get previous migration: prev for version 4 : file does not exist"},
		{4, []uint{1, 2, 3, 5}, "get previous migration: prev for version 4 : file does not exist"},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("entry %d", i), func(t *testing.T) {

			driver, _ := stub.WithInstance(nil, &stub.Config{})
			migrations := driver.(*stub.Stub).Migrations
			for _, version := range tc.existingVersions {
				migrations.Append(&source.Migration{
					Version:    version,
					Identifier: "",
					Direction:  source.Up,
					Raw:        "",
				})
			}

			err := database.CheckLatestVersion(driver, tc.currentVersion)
			var errMessage string
			if err != nil {
				errMessage = err.Error()
			}
			require.Equal(t, tc.expectedResult, errMessage)
		})
	}
}
