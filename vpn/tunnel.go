package vpn

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unicode"

	"golang.org/x/exp/maps"
	"golang.org/x/xerrors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"tailscale.com/net/dns"
	"tailscale.com/util/dnsname"
	"tailscale.com/wgengine/router"

	"github.com/google/uuid"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/coderd/util/ptr"
	"github.com/coder/coder/v2/tailnet"
)

// The interval at which the tunnel sends network status updates to the manager.
const netStatusInterval = 30 * time.Second

type Tunnel struct {
	speaker[*TunnelMessage, *ManagerMessage, ManagerMessage]
	ctx             context.Context
	netLoopDone     chan struct{}
	requestLoopDone chan struct{}

	logger slog.Logger

	logMu sync.Mutex
	logs  []*TunnelMessage

	client Client
	conn   Conn

	mu sync.Mutex
	// agents contains the agents that are currently connected to the tunnel.
	agents map[uuid.UUID]*tailnet.Agent

	// clientLogger is deliberately separate, to avoid the tunnel using itself
	// as a sink for it's own logs, which could lead to deadlocks
	clientLogger slog.Logger
	// router and dnsConfigurator may be nil
	router          router.Router
	dnsConfigurator dns.OSConfigurator
}

type TunnelOption func(t *Tunnel)

func NewTunnel(
	ctx context.Context,
	logger slog.Logger,
	mgrConn io.ReadWriteCloser,
	client Client,
	opts ...TunnelOption,
) (*Tunnel, error) {
	logger = logger.Named("vpn")
	s, err := newSpeaker[*TunnelMessage, *ManagerMessage](
		ctx, logger, mgrConn, SpeakerRoleTunnel, SpeakerRoleManager)
	if err != nil {
		return nil, err
	}
	t := &Tunnel{
		// nolint: govet // safe to copy the locks here because we haven't started the speaker
		speaker:         *(s),
		ctx:             ctx,
		logger:          logger,
		clientLogger:    slog.Make(),
		requestLoopDone: make(chan struct{}),
		netLoopDone:     make(chan struct{}),
		client:          client,
		agents:          make(map[uuid.UUID]*tailnet.Agent),
	}

	for _, opt := range opts {
		opt(t)
	}
	t.speaker.start()
	go t.requestLoop()
	go t.netStatusLoop()
	return t, nil
}

func (t *Tunnel) requestLoop() {
	defer close(t.requestLoopDone)
	for req := range t.speaker.requests {
		if req.msg.Rpc != nil && req.msg.Rpc.MsgId != 0 {
			resp := t.handleRPC(req.msg, req.msg.Rpc.MsgId)
			if err := req.sendReply(resp); err != nil {
				t.logger.Debug(t.ctx, "failed to send RPC reply", slog.Error(err))
			}
			if _, ok := resp.GetMsg().(*TunnelMessage_Stop); ok {
				// TODO: Wait for the reply to be sent before closing the speaker.
				// err := t.speaker.Close()
				// if err != nil {
				// 	t.logger.Error(t.ctx, "failed to close speaker", slog.Error(err))
				// }
				return
			}
			continue
		}
		// Not a unary RPC. We don't know of any message types that are neither a response nor a
		// unary RPC from the Manager.  This shouldn't ever happen because we checked the protocol
		// version during the handshake.
		t.logger.Critical(t.ctx, "unknown request", slog.F("msg", req.msg))
	}
}

func (t *Tunnel) netStatusLoop() {
	ticker := time.NewTicker(netStatusInterval)
	defer ticker.Stop()
	defer close(t.netLoopDone)
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.sendAgentUpdate()
		}
	}
}

// handleRPC handles unary RPCs from the manager.
func (t *Tunnel) handleRPC(req *ManagerMessage, msgID uint64) *TunnelMessage {
	resp := &TunnelMessage{}
	resp.Rpc = &RPC{ResponseTo: msgID}
	switch msg := req.GetMsg().(type) {
	case *ManagerMessage_GetPeerUpdate:
		state, err := t.conn.CurrentWorkspaceState()
		if err != nil {
			t.logger.Critical(t.ctx, "failed to get current workspace state", slog.Error(err))
		}
		update, err := t.createPeerUpdate(state)
		if err != nil {
			t.logger.Error(t.ctx, "failed to populate agent network info", slog.Error(err))
		}
		resp.Msg = &TunnelMessage_PeerUpdate{
			PeerUpdate: update,
		}
		return resp
	case *ManagerMessage_Start:
		startReq := msg.Start
		t.logger.Info(t.ctx, "starting CoderVPN tunnel",
			slog.F("url", startReq.CoderUrl),
			slog.F("tunnel_fd", startReq.TunnelFileDescriptor),
		)
		err := t.start(startReq)
		var errStr string
		if err != nil {
			t.logger.Error(t.ctx, "failed to start tunnel", slog.Error(err))
			errStr = err.Error()
		}
		resp.Msg = &TunnelMessage_Start{
			Start: &StartResponse{
				Success:      err == nil,
				ErrorMessage: errStr,
			},
		}
		return resp
	case *ManagerMessage_Stop:
		t.logger.Info(t.ctx, "stopping CoderVPN tunnel")
		err := t.stop(msg.Stop)
		var errStr string
		if err != nil {
			t.logger.Error(t.ctx, "failed to stop tunnel", slog.Error(err))
			errStr = err.Error()
		} else {
			t.logger.Info(t.ctx, "coderVPN tunnel stopped")
		}
		resp.Msg = &TunnelMessage_Stop{
			Stop: &StopResponse{
				Success:      err == nil,
				ErrorMessage: errStr,
			},
		}
		return resp
	default:
		t.logger.Warn(t.ctx, "unhandled manager request", slog.F("request", msg))
		return resp
	}
}

func UseAsRouter() TunnelOption {
	return func(t *Tunnel) {
		t.router = NewRouter(t)
	}
}

func UseAsLogger() TunnelOption {
	return func(t *Tunnel) {
		t.clientLogger = t.clientLogger.AppendSinks(t)
	}
}

func UseAsDNSConfig() TunnelOption {
	return func(t *Tunnel) {
		t.dnsConfigurator = NewDNSConfigurator(t)
	}
}

// ApplyNetworkSettings sends a request to the manager to apply the given network settings
func (t *Tunnel) ApplyNetworkSettings(ctx context.Context, ns *NetworkSettingsRequest) error {
	msg, err := t.speaker.unaryRPC(ctx, &TunnelMessage{
		Msg: &TunnelMessage_NetworkSettings{
			NetworkSettings: ns,
		},
	})
	if err != nil {
		return xerrors.Errorf("rpc failure: %w", err)
	}
	resp := msg.GetNetworkSettings()
	if !resp.Success {
		return xerrors.Errorf("network settings failed: %s", resp.ErrorMessage)
	}
	return nil
}

func (t *Tunnel) Update(update tailnet.WorkspaceUpdate) error {
	peerUpdate, err := t.createPeerUpdate(update)
	if err != nil {
		t.logger.Error(t.ctx, "failed to populate agent network info", slog.Error(err))
	}
	msg := &TunnelMessage{
		Msg: &TunnelMessage_PeerUpdate{
			PeerUpdate: peerUpdate,
		},
	}
	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	case t.sendCh <- msg:
	}
	return nil
}

func (t *Tunnel) start(req *StartRequest) error {
	rawURL := req.GetCoderUrl()
	if rawURL == "" {
		return xerrors.New("missing coder url")
	}
	svrURL, err := url.Parse(rawURL)
	if err != nil {
		return xerrors.Errorf("parse url %q: %w", rawURL, err)
	}
	apiToken := req.GetApiToken()
	if apiToken == "" {
		return xerrors.New("missing api token")
	}
	var header http.Header
	for _, h := range req.GetHeaders() {
		header.Add(h.GetName(), h.GetValue())
	}

	if t.conn == nil {
		t.conn, err = t.client.NewConn(
			t.ctx,
			svrURL,
			apiToken,
			&Options{
				Headers:           header,
				Logger:            t.clientLogger,
				DNSConfigurator:   t.dnsConfigurator,
				Router:            t.router,
				TUNFileDescriptor: ptr.Ref(int(req.GetTunnelFileDescriptor())),
				UpdateHandler:     t,
			},
		)
	} else {
		t.logger.Warn(t.ctx, "asked to start tunnel, but tunnel is already running")
	}
	return err
}

func (t *Tunnel) stop(*StopRequest) error {
	if t.conn == nil {
		return nil
	}
	return t.conn.Close()
}

var _ slog.Sink = &Tunnel{}

func (t *Tunnel) LogEntry(_ context.Context, e slog.SinkEntry) {
	t.logMu.Lock()
	defer t.logMu.Unlock()
	t.logs = append(t.logs, &TunnelMessage{
		Msg: &TunnelMessage_Log{
			Log: sinkEntryToPb(e),
		},
	})
}

func (t *Tunnel) Sync() {
	t.logMu.Lock()
	logs := t.logs
	t.logs = nil
	t.logMu.Unlock()
	for _, msg := range logs {
		select {
		case <-t.ctx.Done():
			return
		case t.sendCh <- msg:
		}
	}
}

func sinkEntryToPb(e slog.SinkEntry) *Log {
	l := &Log{
		Level:       Log_Level(e.Level),
		Message:     e.Message,
		LoggerNames: e.LoggerNames,
	}
	for _, field := range e.Fields {
		l.Fields = append(l.Fields, &Log_Field{
			Name:  field.Name,
			Value: formatValue(field.Value),
		})
	}
	return l
}

// createPeerUpdate creates a PeerUpdate message from a workspace update, populating
// the network status of the agents.
func (t *Tunnel) createPeerUpdate(update tailnet.WorkspaceUpdate) (*PeerUpdate, error) {
	out := &PeerUpdate{
		UpsertedWorkspaces: make([]*Workspace, len(update.UpsertedWorkspaces)),
		UpsertedAgents:     make([]*Agent, len(update.UpsertedAgents)),
		DeletedWorkspaces:  make([]*Workspace, len(update.DeletedWorkspaces)),
		DeletedAgents:      make([]*Agent, len(update.DeletedAgents)),
	}

	t.saveUpdate(update)

	for i, ws := range update.UpsertedWorkspaces {
		out.UpsertedWorkspaces[i] = &Workspace{
			Id:     tailnet.UUIDToByteSlice(ws.ID),
			Name:   ws.Name,
			Status: Workspace_Status(ws.Status),
		}
	}
	upsertedAgents, err := t.populateAgents(update.UpsertedAgents)
	if err != nil {
		return nil, xerrors.Errorf("failed to populate agent network info: %w", err)
	}
	out.UpsertedAgents = upsertedAgents
	for i, ws := range update.DeletedWorkspaces {
		out.DeletedWorkspaces[i] = &Workspace{
			Id:     tailnet.UUIDToByteSlice(ws.ID),
			Name:   ws.Name,
			Status: Workspace_Status(ws.Status),
		}
	}
	for i, agent := range update.DeletedAgents {
		fqdn := make([]string, 0, len(agent.Hosts))
		for name := range agent.Hosts {
			fqdn = append(fqdn, name.WithTrailingDot())
		}
		out.DeletedAgents[i] = &Agent{
			Id:            tailnet.UUIDToByteSlice(agent.ID),
			Name:          agent.Name,
			WorkspaceId:   tailnet.UUIDToByteSlice(agent.WorkspaceID),
			Fqdn:          fqdn,
			IpAddrs:       hostsToIPStrings(agent.Hosts),
			LastHandshake: nil,
			Latency:       nil,
		}
	}
	return out, nil
}

// Given a list of agents, populate their network info, and return them as proto agents.
func (t *Tunnel) populateAgents(agents []*tailnet.Agent) ([]*Agent, error) {
	if t.conn == nil {
		return nil, xerrors.New("no active connection")
	}

	out := make([]*Agent, 0, len(agents))
	var wg sync.WaitGroup
	pingCtx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	for _, agent := range agents {
		fqdn := make([]string, 0, len(agent.Hosts))
		for name := range agent.Hosts {
			fqdn = append(fqdn, name.WithTrailingDot())
		}
		protoAgent := &Agent{
			Id:          tailnet.UUIDToByteSlice(agent.ID),
			Name:        agent.Name,
			WorkspaceId: tailnet.UUIDToByteSlice(agent.WorkspaceID),
			Fqdn:        fqdn,
			IpAddrs:     hostsToIPStrings(agent.Hosts),
		}
		agentIP := tailnet.CoderServicePrefix.AddrFromUUID(agent.ID)
		wg.Add(1)
		go func() {
			defer wg.Done()
			duration, _, _, err := t.conn.Ping(pingCtx, agentIP)
			if err != nil {
				return
			}
			protoAgent.Latency = durationpb.New(duration)
		}()
		diags := t.conn.GetPeerDiagnostics(agent.ID)
		//nolint:revive // outdated rule
		protoAgent.LastHandshake = timestamppb.New(diags.LastWireguardHandshake)
		out = append(out, protoAgent)
	}
	wg.Wait()

	return out, nil
}

// saveUpdate saves the workspace update to the tunnel's state, such that it can
// be used to populate automated peer updates.
func (t *Tunnel) saveUpdate(update tailnet.WorkspaceUpdate) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, agent := range update.UpsertedAgents {
		t.agents[agent.ID] = agent
	}
	for _, agent := range update.DeletedAgents {
		delete(t.agents, agent.ID)
	}
}

// sendAgentUpdate sends a peer update message to the manager with the current
// state of the agents, including the latest network status.
func (t *Tunnel) sendAgentUpdate() {
	// The lock must be held until we send the message,
	// else we risk upserting a deleted agent.
	t.mu.Lock()
	defer t.mu.Unlock()

	upsertedAgents, err := t.populateAgents(maps.Values(t.agents))
	if err != nil {
		t.logger.Error(t.ctx, "failed to produce agent network status update", slog.Error(err))
		return
	}

	if len(upsertedAgents) == 0 {
		return
	}

	msg := &TunnelMessage{
		Msg: &TunnelMessage_PeerUpdate{
			PeerUpdate: &PeerUpdate{
				UpsertedAgents: upsertedAgents,
			},
		},
	}

	select {
	case <-t.ctx.Done():
		return
	case t.sendCh <- msg:
	}
}

// the following are taken from sloghuman:

func formatValue(v interface{}) string {
	if vr, ok := v.(driver.Valuer); ok {
		var err error
		v, err = vr.Value()
		if err != nil {
			return fmt.Sprintf("error calling Value: %v", err)
		}
	}
	if v == nil {
		return "<nil>"
	}
	typ := reflect.TypeOf(v)
	switch typ.Kind() {
	case reflect.Struct, reflect.Map:
		byt, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(byt)
	case reflect.Slice:
		// Byte slices are optimistically readable.
		if typ.Elem().Kind() == reflect.Uint8 {
			return fmt.Sprintf("%q", v)
		}
		fallthrough
	default:
		return quote(fmt.Sprintf("%+v", v))
	}
}

// quotes quotes a string so that it is suitable
// as a key for a map or in general some output that
// cannot span multiple lines or have weird characters.
func quote(key string) string {
	// strconv.Quote does not quote an empty string so we need this.
	if key == "" {
		return `""`
	}

	var hasSpace bool
	for _, r := range key {
		if unicode.IsSpace(r) {
			hasSpace = true
			break
		}
	}
	quoted := strconv.Quote(key)
	// If the key doesn't need to be quoted, don't quote it.
	// We do not use strconv.CanBackquote because it doesn't
	// account tabs.
	if !hasSpace && quoted[1:len(quoted)-1] == key {
		return key
	}
	return quoted
}

func hostsToIPStrings(hosts map[dnsname.FQDN][]netip.Addr) []string {
	seen := make(map[netip.Addr]struct{})
	var result []string
	for _, inner := range hosts {
		for _, elem := range inner {
			if _, exists := seen[elem]; !exists {
				seen[elem] = struct{}{}
				result = append(result, elem.String())
			}
		}
	}
	return result
}
