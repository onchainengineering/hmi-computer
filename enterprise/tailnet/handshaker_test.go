package tailnet_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	agpltest "github.com/onchainengineering/hmi-computer/test"
	"github.com/onchainengineering/hmi-computer/v2/coderd/database/dbtestutil"
	"github.com/onchainengineering/hmi-computer/v2/enterprise/tailnet"
	"github.com/onchainengineering/hmi-computer/v2/testutil"
)

func TestPGCoordinator_ReadyForHandshake_OK(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := testutil.Logger(t)
	coord1, err := tailnet.NewPGCoord(ctx, logger.Named("coord1"), ps, store)
	require.NoError(t, err)
	defer coord1.Close()

	agpltest.ReadyForHandshakeTest(ctx, t, coord1)
}

func TestPGCoordinator_ReadyForHandshake_NoPermission(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := testutil.Logger(t)
	coord1, err := tailnet.NewPGCoord(ctx, logger.Named("coord1"), ps, store)
	require.NoError(t, err)
	defer coord1.Close()

	agpltest.ReadyForHandshakeNoPermissionTest(ctx, t, coord1)
}
