package vpn

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/tailscale/wireguard-go/tun"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"tailscale.com/net/dns"
	"tailscale.com/wgengine/router"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/workspacesdk"
	"github.com/coder/coder/v2/tailnet"
	"github.com/coder/coder/v2/tailnet/proto"
	"github.com/coder/quartz"
)

type Client struct {
	sdk *codersdk.Client
}

// NewClient creates a new VPN client.
func NewClient(c *codersdk.Client) *Client {
	return &Client{
		sdk: c,
	}
}

type DialOptions struct {
	Logger          slog.Logger
	DNSConfigurator dns.OSConfigurator
	Router          router.Router
	TUNDev          tun.Device
	UpdatesCallback func(*proto.WorkspaceUpdate)
}

func (c *Client) Dial(dialCtx context.Context, options *DialOptions) (vpnConn Conn, err error) {
	if options == nil {
		options = &DialOptions{}
	}

	var headers http.Header
	if headerTransport, ok := c.sdk.HTTPClient.Transport.(*codersdk.HeaderTransport); ok {
		headers = headerTransport.Header
	}
	headers.Set(codersdk.SessionTokenHeader, c.sdk.SessionToken())

	// New context, separate from dialCtx. We don't want to cancel the
	// connection if dialCtx is canceled.
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	rpcURL, err := c.sdk.URL.Parse("/api/v2/tailnet")
	if err != nil {
		return Conn{}, xerrors.Errorf("parse rpc url: %w", err)
	}

	me, err := c.sdk.User(dialCtx, codersdk.Me)
	if err != nil {
		return Conn{}, xerrors.Errorf("get user: %w", err)
	}

	connInfo, err := workspacesdk.New(c.sdk).AgentConnectionInfoGeneric(dialCtx)
	if err != nil {
		return Conn{}, xerrors.Errorf("get connection info: %w", err)
	}

	dialer := workspacesdk.NewWebsocketDialer(options.Logger, rpcURL, &websocket.DialOptions{
		HTTPClient:      c.sdk.HTTPClient,
		HTTPHeader:      headers,
		CompressionMode: websocket.CompressionDisabled,
	}, workspacesdk.WithWorkspaceUpdates(&proto.WorkspaceUpdatesRequest{
		WorkspaceOwnerId: tailnet.UUIDToByteSlice(me.ID),
	}))

	ip := tailnet.CoderServicePrefix.RandomAddr()
	conn, err := tailnet.NewConn(&tailnet.Options{
		Addresses:           []netip.Prefix{netip.PrefixFrom(ip, 128)},
		DERPMap:             connInfo.DERPMap,
		DERPHeader:          &headers,
		DERPForceWebSockets: connInfo.DERPForceWebSockets,
		Logger:              options.Logger,
		BlockEndpoints:      connInfo.DisableDirectConnections,
		DNSConfigurator:     options.DNSConfigurator,
		Router:              options.Router,
		TUNDev:              options.TUNDev,
	})
	if err != nil {
		return Conn{}, xerrors.Errorf("create tailnet: %w", err)
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	clk := quartz.NewReal()
	controller := tailnet.NewController(options.Logger, dialer)
	coordCtrl := tailnet.NewTunnelSrcCoordController(options.Logger, conn)
	controller.ResumeTokenCtrl = tailnet.NewBasicResumeTokenController(options.Logger, clk)
	controller.CoordCtrl = coordCtrl
	controller.DERPCtrl = tailnet.NewBasicDERPController(options.Logger, conn)
	controller.WorkspaceUpdatesCtrl = tailnet.NewTunnelAllWorkspaceUpdatesController(
		options.Logger,
		coordCtrl,
		tailnet.WithDNS(conn, me.Name),
		tailnet.WithCallback(options.UpdatesCallback),
	)
	controller.Run(ctx)

	options.Logger.Debug(ctx, "running tailnet API v2+ connector")

	select {
	case <-dialCtx.Done():
		return Conn{}, xerrors.Errorf("timed out waiting for coordinator and derp map: %w", dialCtx.Err())
	case err = <-dialer.Connected():
		if err != nil {
			options.Logger.Error(ctx, "failed to connect to tailnet v2+ API", slog.Error(err))
			return Conn{}, xerrors.Errorf("start connector: %w", err)
		}
		options.Logger.Debug(ctx, "connected to tailnet v2+ API")
	}

	return Conn{
		Conn:       conn,
		cancelFn:   cancel,
		controller: controller,
	}, nil
}

type Conn struct {
	*tailnet.Conn

	cancelFn   func()
	controller *tailnet.Controller
}

func (c Conn) CurrentWorkspaceState() *proto.WorkspaceUpdate {
	return c.controller.WorkspaceUpdatesCtrl.CurrentState()
}

func (c Conn) Close() error {
	c.cancelFn()
	<-c.controller.Closed()
	return c.Conn.Close()
}
