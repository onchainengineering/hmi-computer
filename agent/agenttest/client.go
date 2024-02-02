package agenttest

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/xerrors"
	"storj.io/drpc"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"tailscale.com/tailcfg"

	"cdr.dev/slog"
	agentproto "github.com/coder/coder/v2/agent/proto"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/agentsdk"
	drpcsdk "github.com/coder/coder/v2/codersdk/drpc"
	"github.com/coder/coder/v2/tailnet"
	"github.com/coder/coder/v2/tailnet/proto"
	"github.com/coder/coder/v2/testutil"
)

func NewClient(t testing.TB,
	logger slog.Logger,
	agentID uuid.UUID,
	manifest agentsdk.Manifest,
	statsChan chan *agentsdk.Stats,
	coordinator tailnet.Coordinator,
) *Client {
	if manifest.AgentID == uuid.Nil {
		manifest.AgentID = agentID
	}
	coordPtr := atomic.Pointer[tailnet.Coordinator]{}
	coordPtr.Store(&coordinator)
	mux := drpcmux.New()
	derpMapUpdates := make(chan *tailcfg.DERPMap)
	drpcService := &tailnet.DRPCService{
		CoordPtr:               &coordPtr,
		Logger:                 logger,
		DerpMapUpdateFrequency: time.Microsecond,
		DerpMapFn:              func() *tailcfg.DERPMap { return <-derpMapUpdates },
	}
	err := proto.DRPCRegisterTailnet(mux, drpcService)
	require.NoError(t, err)
	mp, err := agentsdk.ProtoFromManifest(manifest)
	require.NoError(t, err)
	fakeAAPI := NewFakeAgentAPI(t, logger, mp)
	err = agentproto.DRPCRegisterAgent(mux, fakeAAPI)
	require.NoError(t, err)
	server := drpcserver.NewWithOptions(mux, drpcserver.Options{
		Log: func(err error) {
			if xerrors.Is(err, io.EOF) {
				return
			}
			logger.Debug(context.Background(), "drpc server error", slog.Error(err))
		},
	})
	return &Client{
		t:              t,
		logger:         logger.Named("client"),
		agentID:        agentID,
		statsChan:      statsChan,
		coordinator:    coordinator,
		server:         server,
		fakeAgentAPI:   fakeAAPI,
		derpMapUpdates: derpMapUpdates,
	}
}

type Client struct {
	t                  testing.TB
	logger             slog.Logger
	agentID            uuid.UUID
	metadata           map[string]agentsdk.Metadata
	statsChan          chan *agentsdk.Stats
	coordinator        tailnet.Coordinator
	server             *drpcserver.Server
	fakeAgentAPI       *FakeAgentAPI
	LastWorkspaceAgent func()
	PatchWorkspaceLogs func() error

	mu              sync.Mutex // Protects following.
	lifecycleStates []codersdk.WorkspaceAgentLifecycle
	logs            []agentsdk.Log
	derpMapUpdates  chan *tailcfg.DERPMap
	derpMapOnce     sync.Once
}

func (*Client) RewriteDERPMap(*tailcfg.DERPMap) {}

func (c *Client) Close() {
	c.derpMapOnce.Do(func() { close(c.derpMapUpdates) })
}

func (c *Client) ConnectRPC(ctx context.Context) (drpc.Conn, error) {
	conn, lis := drpcsdk.MemTransportPipe()
	c.LastWorkspaceAgent = func() {
		_ = conn.Close()
		_ = lis.Close()
	}
	c.t.Cleanup(c.LastWorkspaceAgent)
	serveCtx, cancel := context.WithCancel(ctx)
	c.t.Cleanup(cancel)
	auth := tailnet.AgentTunnelAuth{}
	streamID := tailnet.StreamID{
		Name: "agenttest",
		ID:   c.agentID,
		Auth: auth,
	}
	serveCtx = tailnet.WithStreamID(serveCtx, streamID)
	go func() {
		_ = c.server.Serve(serveCtx, lis)
	}()
	return conn, nil
}

func (c *Client) ReportStats(ctx context.Context, _ slog.Logger, statsChan <-chan *agentsdk.Stats, setInterval func(time.Duration)) (io.Closer, error) {
	doneCh := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(doneCh)

		setInterval(500 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case stat := <-statsChan:
				select {
				case c.statsChan <- stat:
				case <-ctx.Done():
					return
				default:
					// We don't want to send old stats.
					continue
				}
			}
		}
	}()
	return closeFunc(func() error {
		cancel()
		<-doneCh
		close(c.statsChan)
		return nil
	}), nil
}

func (c *Client) GetLifecycleStates() []codersdk.WorkspaceAgentLifecycle {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lifecycleStates
}

func (c *Client) PostLifecycle(ctx context.Context, req agentsdk.PostLifecycleRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lifecycleStates = append(c.lifecycleStates, req.State)
	c.logger.Debug(ctx, "post lifecycle", slog.F("req", req))
	return nil
}

func (c *Client) GetStartup() <-chan *agentproto.Startup {
	return c.fakeAgentAPI.startupCh
}

func (c *Client) GetMetadata() map[string]agentsdk.Metadata {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.metadata)
}

func (c *Client) PostMetadata(ctx context.Context, req agentsdk.PostMetadataRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.metadata == nil {
		c.metadata = make(map[string]agentsdk.Metadata)
	}
	for _, md := range req.Metadata {
		c.metadata[md.Key] = md
		c.logger.Debug(ctx, "post metadata", slog.F("key", md.Key), slog.F("md", md))
	}
	return nil
}

func (c *Client) GetStartupLogs() []agentsdk.Log {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.logs
}

func (c *Client) PatchLogs(ctx context.Context, logs agentsdk.PatchLogs) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.PatchWorkspaceLogs != nil {
		return c.PatchWorkspaceLogs()
	}
	c.logs = append(c.logs, logs.Logs...)
	c.logger.Debug(ctx, "patch startup logs", slog.F("req", logs))
	return nil
}

func (c *Client) SetServiceBannerFunc(f func() (codersdk.ServiceBannerConfig, error)) {
	c.fakeAgentAPI.SetServiceBannerFunc(f)
}

func (c *Client) PushDERPMapUpdate(update *tailcfg.DERPMap) error {
	timer := time.NewTimer(testutil.WaitShort)
	defer timer.Stop()
	select {
	case c.derpMapUpdates <- update:
	case <-timer.C:
		return xerrors.New("timeout waiting to push derp map update")
	}

	return nil
}

type closeFunc func() error

func (c closeFunc) Close() error {
	return c()
}

type FakeAgentAPI struct {
	sync.Mutex
	t      testing.TB
	logger slog.Logger

	manifest  *agentproto.Manifest
	startupCh chan *agentproto.Startup

	getServiceBannerFunc func() (codersdk.ServiceBannerConfig, error)
}

func (f *FakeAgentAPI) GetManifest(context.Context, *agentproto.GetManifestRequest) (*agentproto.Manifest, error) {
	return f.manifest, nil
}

func (f *FakeAgentAPI) SetServiceBannerFunc(fn func() (codersdk.ServiceBannerConfig, error)) {
	f.Lock()
	defer f.Unlock()
	f.getServiceBannerFunc = fn
	f.logger.Info(context.Background(), "updated ServiceBannerFunc")
}

func (f *FakeAgentAPI) GetServiceBanner(context.Context, *agentproto.GetServiceBannerRequest) (*agentproto.ServiceBanner, error) {
	f.Lock()
	defer f.Unlock()
	if f.getServiceBannerFunc == nil {
		return &agentproto.ServiceBanner{}, nil
	}
	sb, err := f.getServiceBannerFunc()
	if err != nil {
		return nil, err
	}
	return agentsdk.ProtoFromServiceBanner(sb), nil
}

func (*FakeAgentAPI) UpdateStats(context.Context, *agentproto.UpdateStatsRequest) (*agentproto.UpdateStatsResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (*FakeAgentAPI) UpdateLifecycle(context.Context, *agentproto.UpdateLifecycleRequest) (*agentproto.Lifecycle, error) {
	// TODO implement me
	panic("implement me")
}

func (f *FakeAgentAPI) BatchUpdateAppHealths(ctx context.Context, req *agentproto.BatchUpdateAppHealthRequest) (*agentproto.BatchUpdateAppHealthResponse, error) {
	f.logger.Debug(ctx, "batch update app health", slog.F("req", req))
	return &agentproto.BatchUpdateAppHealthResponse{}, nil
}

func (f *FakeAgentAPI) UpdateStartup(_ context.Context, req *agentproto.UpdateStartupRequest) (*agentproto.Startup, error) {
	f.startupCh <- req.GetStartup()
	return req.GetStartup(), nil
}

func (*FakeAgentAPI) BatchUpdateMetadata(context.Context, *agentproto.BatchUpdateMetadataRequest) (*agentproto.BatchUpdateMetadataResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (*FakeAgentAPI) BatchCreateLogs(context.Context, *agentproto.BatchCreateLogsRequest) (*agentproto.BatchCreateLogsResponse, error) {
	// TODO implement me
	panic("implement me")
}

func NewFakeAgentAPI(t testing.TB, logger slog.Logger, manifest *agentproto.Manifest) *FakeAgentAPI {
	return &FakeAgentAPI{
		t:         t,
		logger:    logger.Named("FakeAgentAPI"),
		manifest:  manifest,
		startupCh: make(chan *agentproto.Startup, 100),
	}
}
