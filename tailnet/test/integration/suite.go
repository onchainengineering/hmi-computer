//go:build linux
// +build linux

package integration

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/tailnet"
	"github.com/coder/coder/v2/testutil"
)

// TODO: instead of reusing one conn for each suite, maybe we should make a new
// one for each subtest?
func TestSuite(t *testing.T, _ slog.Logger, _ *url.URL, conn *tailnet.Conn, _, peer Client) {
	t.Parallel()

	t.Run("Connectivity", func(t *testing.T) {
		t.Parallel()
		peerIP := tailnet.IPFromUUID(peer.ID)
		_, _, _, err := conn.Ping(testutil.Context(t, testutil.WaitLong), peerIP)
		require.NoError(t, err, "ping peer")
	})

	// TODO: more
}
