package coderd

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"tailscale.com/derp"
	"tailscale.com/tailcfg"

	"cdr.dev/slog"
	"github.com/coder/coder/coderd/wsconncache"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/site"
	"github.com/coder/coder/tailnet"
)

var tailnetTransport *http.Transport

func init() {
	var valid bool
	tailnetTransport, valid = http.DefaultTransport.(*http.Transport)
	if !valid {
		panic("dev error: default transport is the wrong type")
	}
}

// TODO(coadler): ServerTailnet does not currently remove stale peers.

// NewServerTailnet creates a new tailnet intended for use by coderd. It
// automatically falls back to wsconncache if a legacy agent is encountered.
func NewServerTailnet(
	ctx context.Context,
	logger slog.Logger,
	derpServer *derp.Server,
	derpMap *tailcfg.DERPMap,
	coord tailnet.Coordinator,
	cache *wsconncache.Cache,
) (*ServerTailnet, error) {
	logger = logger.Named("servertailnet")
	conn, err := tailnet.NewConn(&tailnet.Options{
		Addresses: []netip.Prefix{netip.PrefixFrom(tailnet.IP(), 128)},
		DERPMap:   derpMap,
		Logger:    logger,
	})
	if err != nil {
		return nil, xerrors.Errorf("create tailnet conn: %w", err)
	}

	id := uuid.New()
	ma := coord.ServeMultiAgent(id)

	serverCtx, cancel := context.WithCancel(ctx)
	tn := &ServerTailnet{
		ctx:        serverCtx,
		cancel:     cancel,
		logger:     logger,
		conn:       conn,
		agentConn:  ma,
		cache:      cache,
		agentNodes: map[uuid.UUID]*tailnetNode{},
		transport:  tailnetTransport.Clone(),
	}
	tn.transport.DialContext = tn.dialContext
	tn.transport.MaxIdleConnsPerHost = 10
	tn.transport.MaxIdleConns = 0

	err = ma.UpdateSelf(conn.Node())
	if err != nil {
		tn.logger.Warn(context.Background(), "server tailnet update self", slog.Error(err))
	}
	conn.SetNodeCallback(func(node *tailnet.Node) {
		err := tn.agentConn.UpdateSelf(node)
		if err != nil {
			tn.logger.Warn(context.Background(), "broadcast server node to agents", slog.Error(err))
		}
	})

	// This is set to allow local DERP traffic to be proxied through memory
	// instead of needing to hit the external access URL. Don't use the ctx
	// given in this callback, it's only valid while connecting.
	conn.SetDERPRegionDialer(func(_ context.Context, region *tailcfg.DERPRegion) net.Conn {
		if !region.EmbeddedRelay {
			return nil
		}
		left, right := net.Pipe()
		go func() {
			defer left.Close()
			defer right.Close()
			brw := bufio.NewReadWriter(bufio.NewReader(right), bufio.NewWriter(right))
			derpServer.Accept(ctx, right, brw, "internal")
		}()
		return left
	})

	go tn.watchAgentUpdates()
	return tn, nil
}

func (s *ServerTailnet) watchAgentUpdates() {
	for {
		nodes := s.agentConn.NextUpdate(s.ctx)
		if nodes == nil {
			return
		}

		err := s.conn.UpdateNodes(nodes, false)
		if err != nil {
			s.logger.Error(context.Background(), "update node in server tailnet", slog.Error(err))
			return
		}
	}
}

type tailnetNode struct {
	lastConnection time.Time
	close          func()
}

type ServerTailnet struct {
	ctx    context.Context
	cancel func()

	logger    slog.Logger
	conn      *tailnet.Conn
	agentConn tailnet.MultiAgentConn
	cache     *wsconncache.Cache
	nodesMu   sync.Mutex
	// agentNodes is a map of agent tailnetNodes the server wants to keep a
	// connection to.
	agentNodes map[uuid.UUID]*tailnetNode

	transport *http.Transport
}

func (s *ServerTailnet) ReverseProxy(targetURL, dashboardURL *url.URL, agentID uuid.UUID) (_ *httputil.ReverseProxy, release func(), _ error) {
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		site.RenderStaticErrorPage(w, r, site.ErrorPageData{
			Status:       http.StatusBadGateway,
			Title:        "Bad Gateway",
			Description:  "Failed to proxy request to application: " + err.Error(),
			RetryEnabled: true,
			DashboardURL: dashboardURL.String(),
		})
	}
	proxy.Director = s.director(agentID, proxy.Director)
	proxy.Transport = s.transport

	return proxy, func() {}, nil
}

type agentIDKey struct{}

// director makes sure agentIDKey is set on the context in the reverse proxy.
// This allows the transport to correctly identify which agent to dial to.
func (*ServerTailnet) director(agentID uuid.UUID, prev func(req *http.Request)) func(req *http.Request) {
	return func(req *http.Request) {
		ctx := context.WithValue(req.Context(), agentIDKey{}, agentID)
		*req = *req.WithContext(ctx)
		prev(req)
	}
}

func (s *ServerTailnet) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	agentID, ok := ctx.Value(agentIDKey{}).(uuid.UUID)
	if !ok {
		return nil, xerrors.Errorf("no agent id attached")
	}

	return s.DialAgentNetConn(ctx, agentID, network, addr)
}

func (s *ServerTailnet) ensureAgent(agentID uuid.UUID) error {
	s.nodesMu.Lock()
	tnode, ok := s.agentNodes[agentID]
	// If we don't have the node, subscribe.
	if !ok {
		s.logger.Debug(s.ctx, "subscribing to agent", slog.F("agent_id", agentID))
		closer, err := s.agentConn.SubscribeAgent(agentID)
		if err != nil {
			return xerrors.Errorf("subscribe agent: %w", err)
		}
		tnode = &tailnetNode{
			lastConnection: time.Now(),
			close:          closer,
		}
		s.agentNodes[agentID] = tnode
	} else {
		tnode.lastConnection = time.Now()
	}
	s.nodesMu.Unlock()

	return nil
}

func (s *ServerTailnet) AgentConn(ctx context.Context, agentID uuid.UUID) (*codersdk.WorkspaceAgentConn, func(), error) {
	var (
		conn *codersdk.WorkspaceAgentConn
		ret  = func() {}
	)

	if s.agentConn.AgentIsLegacy(agentID) {
		s.logger.Debug(s.ctx, "acquiring legacy agent", slog.F("agent_id", agentID))
		cconn, release, err := s.cache.Acquire(agentID)
		if err != nil {
			return nil, nil, xerrors.Errorf("acquire legacy agent conn: %w", err)
		}

		conn = cconn.WorkspaceAgentConn
		ret = release
	} else {
		err := s.ensureAgent(agentID)
		if err != nil {
			return nil, nil, xerrors.Errorf("ensure agent: %w", err)
		}

		s.logger.Debug(s.ctx, "acquiring agent", slog.F("agent_id", agentID))
		conn = codersdk.NewWorkspaceAgentConn(s.conn, codersdk.WorkspaceAgentConnOptions{
			AgentID:   agentID,
			CloseFunc: func() error { return codersdk.ErrSkipClose },
		})
	}

	// Since we now have an open conn, be careful to close it if we error
	// without returning it to the user.

	reachable := conn.AwaitReachable(ctx)
	if !reachable {
		ret()
		conn.Close()
		return nil, nil, xerrors.New("agent is unreachable")
	}

	return conn, ret, nil
}

func (s *ServerTailnet) DialAgentNetConn(ctx context.Context, agentID uuid.UUID, network, addr string) (net.Conn, error) {
	conn, release, err := s.AgentConn(ctx, agentID)
	if err != nil {
		return nil, xerrors.Errorf("acquire agent conn: %w", err)
	}

	// Since we now have an open conn, be careful to close it if we error
	// without returning it to the user.

	nc, err := conn.DialContext(ctx, network, addr)
	if err != nil {
		release()
		conn.Close()
		return nil, xerrors.Errorf("dial context: %w", err)
	}

	return &netConnCloser{Conn: nc, close: func() {
		release()
		conn.Close()
	}}, err
}

type netConnCloser struct {
	net.Conn
	close func()
}

func (c *netConnCloser) Close() error {
	c.close()
	return c.Conn.Close()
}

func (s *ServerTailnet) Close() error {
	s.cancel()
	_ = s.cache.Close()
	_ = s.conn.Close()
	s.transport.CloseIdleConnections()
	return nil
}
