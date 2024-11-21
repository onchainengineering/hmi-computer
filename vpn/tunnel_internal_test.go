package vpn

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/tailnet/proto"
	"github.com/coder/coder/v2/testutil"
)

func newFakeClient(ctx context.Context, t *testing.T) *fakeClient {
	return &fakeClient{
		t:   t,
		ctx: ctx,
		ch:  make(chan *fakeConn, 1),
	}
}

type fakeClient struct {
	t   *testing.T
	ctx context.Context
	ch  chan *fakeConn
}

var _ Client = (*fakeClient)(nil)

func (f *fakeClient) NewConn(context.Context, *url.URL, string, *Options) (Conn, error) {
	return testutil.RequireRecvCtx(f.ctx, f.t, f.ch), nil
}

func newFakeConn(state *proto.WorkspaceUpdate) *fakeConn {
	return &fakeConn{
		closed: make(chan struct{}),
		state:  state,
	}
}

type fakeConn struct {
	state  *proto.WorkspaceUpdate
	closed chan struct{}
}

var _ Conn = (*fakeConn)(nil)

func (f *fakeConn) CurrentWorkspaceState() *proto.WorkspaceUpdate {
	return f.state
}

func (f *fakeConn) Close() error {
	close(f.closed)
	return nil
}

func TestTunnel_StartStop(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitShort)
	client := newFakeClient(ctx, t)
	conn := newFakeConn(nil)

	_, mgr := setupTunnel(t, ctx, client)

	errCh := make(chan error, 1)
	var resp *TunnelMessage
	// When: we start the tunnel
	go func() {
		r, err := mgr.unaryRPC(ctx, &ManagerMessage{
			Msg: &ManagerMessage_Start{
				Start: &StartRequest{
					TunnelFileDescriptor: 2,
					CoderUrl:             "https://coder.example.com",
					ApiToken:             "fakeToken",
				},
			},
		})
		resp = r
		errCh <- err
	}()
	// Then: `NewConn` is called,
	testutil.RequireSendCtx(ctx, t, client.ch, conn)
	// And: a response is received
	err := testutil.RequireRecvCtx(ctx, t, errCh)
	require.NoError(t, err)
	_, ok := resp.Msg.(*TunnelMessage_Start)
	require.True(t, ok)

	// When: we stop the tunnel
	go func() {
		r, err := mgr.unaryRPC(ctx, &ManagerMessage{
			Msg: &ManagerMessage_Stop{},
		})
		resp = r
		errCh <- err
	}()
	// Then: `Close` is called on the connection
	testutil.RequireRecvCtx(ctx, t, conn.closed)
	// And: a Stop response is received
	err = testutil.RequireRecvCtx(ctx, t, errCh)
	require.NoError(t, err)
	_, ok = resp.Msg.(*TunnelMessage_Stop)
	require.True(t, ok)

	err = mgr.Close()
	require.NoError(t, err)
}

func TestTunnel_PeerUpdate(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitShort)

	client := newFakeClient(ctx, t)
	conn := newFakeConn(&proto.WorkspaceUpdate{
		UpsertedWorkspaces: []*proto.Workspace{
			{
				Id: []byte("1"),
			},
			{
				Id: []byte("2"),
			},
		},
	})

	tun, mgr := setupTunnel(t, ctx, client)

	errCh := make(chan error, 1)
	var resp *TunnelMessage
	// When: we start the tunnel
	go func() {
		r, err := mgr.unaryRPC(ctx, &ManagerMessage{
			Msg: &ManagerMessage_Start{
				Start: &StartRequest{
					TunnelFileDescriptor: 2,
					CoderUrl:             "https://coder.example.com",
					ApiToken:             "fakeToken",
				},
			},
		})
		resp = r
		errCh <- err
	}()
	// Then: `NewConn` is called,
	testutil.RequireSendCtx(ctx, t, client.ch, conn)
	// And: a response is received
	err := testutil.RequireRecvCtx(ctx, t, errCh)
	require.NoError(t, err)
	_, ok := resp.Msg.(*TunnelMessage_Start)
	require.True(t, ok)

	// When: we inform the tunnel of an update
	err = tun.Update(&proto.WorkspaceUpdate{
		UpsertedWorkspaces: []*proto.Workspace{
			{
				Id: []byte("2"),
			},
		},
	})
	require.NoError(t, err)
	// Then: the tunnel sends a PeerUpdate message
	req := testutil.RequireRecvCtx(ctx, t, mgr.requests)
	require.Nil(t, req.msg.Rpc)
	require.NotNil(t, req.msg.GetPeerUpdate())
	require.Len(t, req.msg.GetPeerUpdate().UpsertedWorkspaces, 1)
	require.Equal(t, []byte("2"), req.msg.GetPeerUpdate().UpsertedWorkspaces[0].Id)

	// When: the manager requests a PeerUpdate
	go func() {
		r, err := mgr.unaryRPC(ctx, &ManagerMessage{
			Msg: &ManagerMessage_GetPeerUpdate{},
		})
		resp = r
		errCh <- err
	}()
	// Then: a PeerUpdate message is sent using the Conn's state
	err = testutil.RequireRecvCtx(ctx, t, errCh)
	require.NoError(t, err)
	_, ok = resp.Msg.(*TunnelMessage_PeerUpdate)
	require.True(t, ok)
	require.Len(t, resp.GetPeerUpdate().UpsertedWorkspaces, 2)
	require.Equal(t, []byte("1"), resp.GetPeerUpdate().UpsertedWorkspaces[0].Id)
	require.Equal(t, []byte("2"), resp.GetPeerUpdate().UpsertedWorkspaces[1].Id)
}

//nolint:revive // t takes precedence
func setupTunnel(t *testing.T, ctx context.Context, client *fakeClient) (*Tunnel, *speaker[*ManagerMessage, *TunnelMessage, TunnelMessage]) {
	mp, tp := net.Pipe()
	t.Cleanup(func() { _ = mp.Close() })
	t.Cleanup(func() { _ = tp.Close() })

	var tun *Tunnel
	var mgr *speaker[*ManagerMessage, *TunnelMessage, TunnelMessage]
	errCh := make(chan error, 2)
	go func() {
		tunnel, err := NewTunnel(ctx, testutil.Logger(t).Named("tunnel"), tp, client)
		tun = tunnel
		errCh <- err
	}()
	go func() {
		manager, err := newSpeaker[*ManagerMessage, *TunnelMessage](ctx, testutil.Logger(t).Named("manager"), mp, SpeakerRoleManager, SpeakerRoleTunnel)
		mgr = manager
		errCh <- err
	}()
	err := testutil.RequireRecvCtx(ctx, t, errCh)
	require.NoError(t, err)
	err = testutil.RequireRecvCtx(ctx, t, errCh)
	require.NoError(t, err)
	mgr.start()
	return tun, mgr
}
