package workspacesdk

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
	"tailscale.com/tailcfg"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/apiversion"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/tailnet"
	"github.com/coder/coder/v2/tailnet/proto"
	"github.com/coder/coder/v2/tailnet/tailnettest"
	"github.com/coder/coder/v2/testutil"
)

func init() {
	// Give tests a bit more time to timeout. Darwin is particularly slow.
	tailnetConnectorGracefulTimeout = 5 * time.Second
}

func TestTailnetAPIConnector_Disconnects(t *testing.T) {
	t.Parallel()
	testCtx := testutil.Context(t, testutil.WaitShort)
	ctx, cancel := context.WithCancel(testCtx)
	logger := slogtest.Make(t, &slogtest.Options{
		IgnoredErrorIs: append(slogtest.DefaultIgnoredErrorIs,
			io.EOF,                   // we get EOF when we simulate a DERPMap error
			yamux.ErrSessionShutdown, // coordination can throw these when DERP error tears down session
		),
	}).Leveled(slog.LevelDebug)
	agentID := uuid.UUID{0x55}
	clientID := uuid.UUID{0x66}
	fCoord := tailnettest.NewFakeCoordinator()
	var coord tailnet.Coordinator = fCoord
	coordPtr := atomic.Pointer[tailnet.Coordinator]{}
	coordPtr.Store(&coord)
	derpMapCh := make(chan *tailcfg.DERPMap)
	defer close(derpMapCh)
	svc, err := tailnet.NewClientService(
		logger, &coordPtr,
		time.Millisecond, func() *tailcfg.DERPMap { return <-derpMapCh },
	)
	require.NoError(t, err)

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sws, err := websocket.Accept(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		ctx, nc := codersdk.WebsocketNetConn(r.Context(), sws, websocket.MessageBinary)
		err = svc.ServeConnV2(ctx, nc, tailnet.StreamID{
			Name: "client",
			ID:   clientID,
			Auth: tailnet.ClientCoordinateeAuth{AgentID: agentID},
		})
		assert.NoError(t, err)
	}))

	fConn := newFakeTailnetConn()

	uut := newTailnetAPIConnector(ctx, logger, agentID, svr.URL, &websocket.DialOptions{})
	uut.runConnector(fConn)

	call := testutil.RequireRecvCtx(ctx, t, fCoord.CoordinateCalls)
	reqTun := testutil.RequireRecvCtx(ctx, t, call.Reqs)
	require.NotNil(t, reqTun.AddTunnel)

	_ = testutil.RequireRecvCtx(ctx, t, uut.connected)

	// simulate a problem with DERPMaps by sending nil
	testutil.RequireSendCtx(ctx, t, derpMapCh, nil)

	// this should cause the coordinate call to hang up WITHOUT disconnecting
	reqNil := testutil.RequireRecvCtx(ctx, t, call.Reqs)
	require.Nil(t, reqNil)

	// ...and then reconnect
	call = testutil.RequireRecvCtx(ctx, t, fCoord.CoordinateCalls)
	reqTun = testutil.RequireRecvCtx(ctx, t, call.Reqs)
	require.NotNil(t, reqTun.AddTunnel)

	// canceling the context should trigger the disconnect message
	cancel()
	reqDisc := testutil.RequireRecvCtx(testCtx, t, call.Reqs)
	require.NotNil(t, reqDisc)
	require.NotNil(t, reqDisc.Disconnect)
}

func TestTailnetAPIConnector_UplevelVersion(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitShort)
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	agentID := uuid.UUID{0x55}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sVer := apiversion.New(proto.CurrentMajor, proto.CurrentMinor-1)

		// the following matches what Coderd does;
		// c.f. coderd/workspaceagents.go: workspaceAgentClientCoordinate
		cVer := r.URL.Query().Get("version")
		if err := sVer.Validate(cVer); err != nil {
			httpapi.Write(ctx, w, http.StatusBadRequest, codersdk.Response{
				Message: AgentAPIMismatchMessage,
				Validations: []codersdk.ValidationError{
					{Field: "version", Detail: err.Error()},
				},
			})
			return
		}
	}))

	fConn := newFakeTailnetConn()

	uut := runTailnetAPIConnector(ctx, logger, agentID, svr.URL, &websocket.DialOptions{}, fConn)

	err := testutil.RequireRecvCtx(ctx, t, uut.connected)
	var sdkErr *codersdk.Error
	require.ErrorAs(t, err, &sdkErr)
	require.Equal(t, http.StatusBadRequest, sdkErr.StatusCode())
	require.Equal(t, AgentAPIMismatchMessage, sdkErr.Message)
	require.NotEmpty(t, sdkErr.Helper)
}

type fakeTailnetConn struct{}

func (*fakeTailnetConn) UpdatePeers([]*proto.CoordinateResponse_PeerUpdate) error {
	// TODO implement me
	panic("implement me")
}

func (*fakeTailnetConn) SetAllPeersLost() {}

func (*fakeTailnetConn) SetNodeCallback(func(*tailnet.Node)) {}

func (*fakeTailnetConn) SetDERPMap(*tailcfg.DERPMap) {}

func (*fakeTailnetConn) SetTunnelDestination(uuid.UUID) {}

func newFakeTailnetConn() *fakeTailnetConn {
	return &fakeTailnetConn{}
}
