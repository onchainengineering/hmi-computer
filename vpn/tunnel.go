package vpn

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"unicode"

	"golang.org/x/xerrors"

	"github.com/hashicorp/go-multierror"

	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/tailnet/proto"

	"cdr.dev/slog"
)

type Tunnel struct {
	speaker[*TunnelMessage, *ManagerMessage, ManagerMessage]
	ctx             context.Context
	logger          slog.Logger
	requestLoopDone chan struct{}

	logMu sync.Mutex
	logs  []*TunnelMessage

	conn Conn
}

func NewTunnel(
	ctx context.Context, logger slog.Logger, conn io.ReadWriteCloser,
) (*Tunnel, error) {
	logger = logger.Named("vpn")
	s, err := newSpeaker[*TunnelMessage, *ManagerMessage](
		ctx, logger, conn, SpeakerRoleTunnel, SpeakerRoleManager)
	if err != nil {
		return nil, err
	}
	t := &Tunnel{
		// nolint: govet // safe to copy the locks here because we haven't started the speaker
		speaker:         *(s),
		ctx:             ctx,
		logger:          logger,
		requestLoopDone: make(chan struct{}),
	}
	logger.AppendSinks(t)
	t.speaker.start()
	go t.requestLoop()
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
				// The speaker has been closed
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

// handleRPC handles unary RPCs from the manager.
func (t *Tunnel) handleRPC(req *ManagerMessage, msgID uint64) *TunnelMessage {
	resp := &TunnelMessage{}
	resp.Rpc = &RPC{ResponseTo: msgID}
	switch msg := req.GetMsg().(type) {
	case *ManagerMessage_GetPeerUpdate:
		resp.Msg = &TunnelMessage_PeerUpdate{
			PeerUpdate: convertWorkspaceUpdate(t.conn.CurrentWorkspaceState()),
		}
		return resp
	case *ManagerMessage_Start:
		startReq := msg.Start
		t.logger.Info(t.ctx, "starting CoderVPN tunnel",
			slog.F("url", startReq.CoderUrl),
			slog.F("tunnel_fd", startReq.TunnelFileDescriptor),
		)
		err := t.start(startReq)
		if err != nil {
			t.logger.Error(t.ctx, "failed to start tunnel", slog.Error(err))
		}
		resp.Msg = &TunnelMessage_Start{
			Start: &StartResponse{
				Success: err == nil,
			},
		}
		return resp
	case *ManagerMessage_Stop:
		t.logger.Info(t.ctx, "stopping CoderVPN tunnel")
		err := t.stop(msg.Stop)
		if err != nil {
			t.logger.Error(t.ctx, "failed to stop tunnel", slog.Error(err))
		} else {
			t.logger.Info(t.ctx, "coderVPN tunnel stopped")
		}
		resp.Msg = &TunnelMessage_Stop{
			Stop: &StopResponse{
				Success: err == nil,
			},
		}
		return resp
	default:
		t.logger.Warn(t.ctx, "unhandled manager request", slog.F("request", msg))
		return resp
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

func (t *Tunnel) PushWorkspaceUpdate(ctx context.Context, update *proto.WorkspaceUpdate) error {
	msg := &TunnelMessage{
		Msg: &TunnelMessage_PeerUpdate{
			PeerUpdate: convertWorkspaceUpdate(update),
		},
	}
	select {
	case <-t.ctx.Done():
		return ctx.Err()
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
		return xerrors.Errorf("parse url: %w", err)
	}
	apiToken := req.GetApiToken()
	if apiToken == "" {
		return xerrors.New("missing api token")
	}

	sdk := codersdk.New(svrURL)
	sdk.SetSessionToken(apiToken)
	client := &Client{sdk: sdk}

	transport := &codersdk.HeaderTransport{
		Transport: http.DefaultTransport,
		Header:    http.Header{},
	}
	for _, header := range req.GetHeaders() {
		transport.Header.Add(header.Name, header.Value)
	}
	sdk.HTTPClient.Transport = transport

	// No-op on non-Darwin platforms.
	dev, err := makeTUN(int(req.GetTunnelFileDescriptor()))
	if err != nil {
		return xerrors.Errorf("make TUN: %w", err)
	}

	t.conn, err = client.Dial(t.ctx, &DialOptions{
		Logger:          t.logger,
		DNSConfigurator: NewDNSConfigurator(t),
		Router:          NewRouter(t),
		TUNDev:          dev,
		UpdatesCallback: func(wu *proto.WorkspaceUpdate) {
			err := t.PushWorkspaceUpdate(t.ctx, wu)
			if err != nil {
				t.logger.Error(t.ctx, "failed to push workspace update", slog.Error(err))
			}
		},
	})
	return err
}

func (t *Tunnel) stop(*StopRequest) error {
	var err error
	cErr := t.conn.Close()
	if cErr != nil {
		err = multierror.Append(err, xerrors.Errorf("close VPN connection: %w", cErr))
	}
	sErr := t.speaker.Close()
	if sErr != nil {
		err = multierror.Append(err, xerrors.Errorf("close speaker: %w", sErr))
	}
	return err
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

func convertWorkspaceUpdate(update *proto.WorkspaceUpdate) *PeerUpdate {
	out := &PeerUpdate{
		UpsertedWorkspaces: make([]*Workspace, len(update.UpsertedWorkspaces)),
		UpsertedAgents:     make([]*Agent, len(update.UpsertedAgents)),
		DeletedWorkspaces:  make([]*Workspace, len(update.DeletedWorkspaces)),
		DeletedAgents:      make([]*Agent, len(update.DeletedAgents)),
	}
	for i, ws := range update.UpsertedWorkspaces {
		out.UpsertedWorkspaces[i] = &Workspace{
			Id:   ws.Id,
			Name: ws.Name,
		}
	}
	for i, agent := range update.UpsertedAgents {
		out.UpsertedAgents[i] = &Agent{
			Id:   agent.Id,
			Name: agent.Name,
		}
	}
	for i, ws := range update.DeletedWorkspaces {
		out.DeletedWorkspaces[i] = &Workspace{
			Id:   ws.Id,
			Name: ws.Name,
		}
	}
	for i, agent := range update.DeletedAgents {
		out.DeletedAgents[i] = &Agent{
			Id:   agent.Id,
			Name: agent.Name,
		}
	}
	return out
}
