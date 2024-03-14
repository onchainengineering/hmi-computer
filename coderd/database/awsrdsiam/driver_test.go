package awsrdsiam_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/slog/sloggers/slogtest"

	"github.com/coder/coder/v2/cli"
	"github.com/coder/coder/v2/coderd/database/awsrdsiam"
	"github.com/coder/coder/v2/testutil"
)

func TestDriver(t *testing.T) {
	t.Parallel()
	// Be sure to set AWS_DEFAULT_REGION to the database region as well.
	url := os.Getenv("DBAWSIAMRDS_TEST_URL")
	if url == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	defer cancel()

	sqlDriver, err := awsrdsiam.Register("postgres")
	require.NoError(t, err)

	db, err := cli.ConnectToPostgres(ctx, slogtest.Make(t, nil), sqlDriver, url)
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	i, err := db.QueryContext(ctx, "select 1;")
	require.NoError(t, err)
	defer func() {
		_ = i.Close()
	}()

	require.True(t, i.Next())
	var one int
	require.NoError(t, i.Scan(&one))
	require.Equal(t, 1, one)
}
