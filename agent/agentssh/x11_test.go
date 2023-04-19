package agentssh_test

import (
	"context"
	"encoding/hex"
	"net"
	"path/filepath"
	"testing"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	gossh "golang.org/x/crypto/ssh"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/agent/agentssh"
	"github.com/coder/coder/codersdk/agentsdk"
	"github.com/coder/coder/testutil"
)

func TestServer_X11(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	fs := afero.NewOsFs()
	dir := t.TempDir()
	s, err := agentssh.NewServer(ctx, logger, fs, 0, dir)
	require.NoError(t, err)
	defer s.Close()

	// The assumption is that these are set before serving SSH connections.
	s.AgentToken = func() string { return "" }
	s.Manifest = atomic.NewPointer(&agentsdk.Manifest{})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := s.Serve(ln)
		assert.Error(t, err) // Server is closed.
	}()

	c := sshClient(t, ln.Addr().String())

	sess, err := c.NewSession()
	require.NoError(t, err)

	reply, err := sess.SendRequest("x11-req", true, gossh.Marshal(ssh.X11{
		AuthProtocol: "MIT-MAGIC-COOKIE-1",
		AuthCookie:   hex.EncodeToString([]byte("cookie")),
		ScreenNumber: 0,
	}))
	require.NoError(t, err)
	assert.True(t, reply)

	err = sess.Shell()
	require.NoError(t, err)

	x11Chans := c.HandleChannelOpen("x11")
	require.Eventually(t, func() bool {
		conn, err := net.Dial("unix", filepath.Join(dir, "X0"))
		if err == nil {
			_ = conn.Close()
		}
		return err == nil
	}, testutil.WaitShort, testutil.IntervalFast)

	x11 := <-x11Chans
	ch, reqs, err := x11.Accept()
	require.NoError(t, err)
	go gossh.DiscardRequests(reqs)
	err = ch.Close()
	require.NoError(t, err)
	s.Close()
	<-done
}
