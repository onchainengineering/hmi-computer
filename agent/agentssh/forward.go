package agentssh

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
)

// streamLocalForwardPayload describes the extra data sent in a
// streamlocal-forward@openssh.com containing the socket path to bind to.
type streamLocalForwardPayload struct {
	SocketPath string
}

// forwardedStreamLocalPayload describes the data sent as the payload in the new
// channel request when a Unix connection is accepted by the listener.
type forwardedStreamLocalPayload struct {
	SocketPath string
	Reserved   uint32
}

// forwardedUnixHandler is a clone of ssh.ForwardedTCPHandler that does
// streamlocal forwarding (aka. unix forwarding) instead of TCP forwarding.
type forwardedUnixHandler struct {
	sync.Mutex
	log      slog.Logger
	forwards map[forwardKey]net.Listener
}

type forwardKey struct {
	sessionID string
	addr      string
}

func newForwardedUnixHandler(log slog.Logger) *forwardedUnixHandler {
	return &forwardedUnixHandler{
		log:      log,
		forwards: make(map[forwardKey]net.Listener),
	}
}

func (h *forwardedUnixHandler) HandleSSHRequest(ctx ssh.Context, _ *ssh.Server, req *gossh.Request) (bool, []byte) {
	h.log.Debug(ctx, "handling SSH unix forward")
	conn, ok := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
	if !ok {
		h.log.Warn(ctx, "SSH unix forward request from client with no gossh connection")
		return false, nil
	}
	log := h.log.With(slog.F("session_id", ctx.SessionID()), slog.F("remote_addr", conn.RemoteAddr()))

	switch req.Type {
	case "streamlocal-forward@openssh.com":
		var reqPayload streamLocalForwardPayload
		err := gossh.Unmarshal(req.Payload, &reqPayload)
		if err != nil {
			h.log.Warn(ctx, "parse streamlocal-forward@openssh.com request (SSH unix forward) payload from client", slog.Error(err))
			return false, nil
		}

		addr := reqPayload.SocketPath
		log = log.With(slog.F("socket_path", addr))
		log.Debug(ctx, "request begin SSH unix forward")

		key := forwardKey{
			sessionID: ctx.SessionID(),
			addr:      addr,
		}

		h.Lock()
		_, ok := h.forwards[key]
		h.Unlock()
		if ok {
			// In cases where `ExitOnForwardFailure=yes` is set, returning false
			// here will cause the connection to be closed. To avoid this, and
			// to match OpenSSH behavior, we silently ignore the second forward
			// request.
			log.Warn(ctx, "SSH unix forward request for socket path that is already being forwarded on this session, ignoring")
			return true, nil
		}

		// Create socket parent dir if not exists.
		parentDir := filepath.Dir(addr)
		err = os.MkdirAll(parentDir, 0o700)
		if err != nil {
			log.Warn(ctx, "create parent dir for SSH unix forward request",
				slog.F("parent_dir", parentDir),
				slog.Error(err),
			)
			return false, nil
		}

		// Remove existing socket if it exists. It's possible we will overwrite
		// a regular file here, but this matches the behavior of OpenSSH.
		err = unlink(addr)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Warn(ctx, "remove existing socket for SSH unix forward request", slog.Error(err))
			return false, nil
		}

		lc := &net.ListenConfig{}
		ln, err := lc.Listen(ctx, "unix", addr)
		if err != nil {
			log.Warn(ctx, "listen on Unix socket for SSH unix forward request", slog.Error(err))
			return false, nil
		}
		log.Debug(ctx, "SSH unix forward listening on socket")

		// The listener needs to successfully start before it can be added to
		// the map, so we don't have to worry about checking for an existing
		// listener.
		//
		// This is also what the upstream TCP version of this code does.
		h.Lock()
		h.forwards[key] = ln
		h.Unlock()
		log.Debug(ctx, "SSH unix forward added to cache")

		ctx, cancel := context.WithCancel(ctx)
		go func() {
			<-ctx.Done()
			_ = ln.Close()
		}()
		go func() {
			defer cancel()

			for {
				c, err := ln.Accept()
				if err != nil {
					if !xerrors.Is(err, net.ErrClosed) {
						log.Warn(ctx, "accept on local Unix socket for SSH unix forward request", slog.Error(err))
					}
					// closed below
					log.Debug(ctx, "SSH unix forward listener closed")
					break
				}
				log.Debug(ctx, "accepted SSH unix forward connection")
				payload := gossh.Marshal(&forwardedStreamLocalPayload{
					SocketPath: addr,
				})

				go func() {
					ch, reqs, err := conn.OpenChannel("forwarded-streamlocal@openssh.com", payload)
					if err != nil {
						h.log.Warn(ctx, "open SSH unix forward channel to client", slog.Error(err))
						_ = c.Close()
						return
					}
					go gossh.DiscardRequests(reqs)
					Bicopy(ctx, ch, c)
				}()
			}

			h.Lock()
			if ln2, ok := h.forwards[key]; ok && ln2 == ln {
				delete(h.forwards, key)
			}
			h.Unlock()
			log.Debug(ctx, "SSH unix forward listener removed from cache")
			_ = ln.Close()
		}()

		return true, nil

	case "cancel-streamlocal-forward@openssh.com":
		var reqPayload streamLocalForwardPayload
		err := gossh.Unmarshal(req.Payload, &reqPayload)
		if err != nil {
			h.log.Warn(ctx, "parse cancel-streamlocal-forward@openssh.com (SSH unix forward) request payload from client", slog.Error(err))
			return false, nil
		}
		log.Debug(ctx, "request to cancel SSH unix forward", slog.F("socket_path", reqPayload.SocketPath))

		key := forwardKey{
			sessionID: ctx.SessionID(),
			addr:      reqPayload.SocketPath,
		}

		h.Lock()
		ln, ok := h.forwards[key]
		delete(h.forwards, key)
		h.Unlock()
		if !ok {
			log.Warn(ctx, "SSH unix forward not found in cache")
			return true, nil
		}
		_ = ln.Close()
		return true, nil

	default:
		return false, nil
	}
}

// directStreamLocalPayload describes the extra data sent in a
// direct-streamlocal@openssh.com channel request containing the socket path.
type directStreamLocalPayload struct {
	SocketPath string

	Reserved1 string
	Reserved2 uint32
}

func directStreamLocalHandler(_ *ssh.Server, _ *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
	var reqPayload directStreamLocalPayload
	err := gossh.Unmarshal(newChan.ExtraData(), &reqPayload)
	if err != nil {
		_ = newChan.Reject(gossh.ConnectionFailed, "could not parse direct-streamlocal@openssh.com channel payload")
		return
	}

	var dialer net.Dialer
	dconn, err := dialer.DialContext(ctx, "unix", reqPayload.SocketPath)
	if err != nil {
		_ = newChan.Reject(gossh.ConnectionFailed, fmt.Sprintf("dial unix socket %q: %+v", reqPayload.SocketPath, err.Error()))
		return
	}

	ch, reqs, err := newChan.Accept()
	if err != nil {
		_ = dconn.Close()
		return
	}
	go gossh.DiscardRequests(reqs)

	Bicopy(ctx, ch, dconn)
}

// unlink removes files and unlike os.Remove, directories are kept.
func unlink(path string) error {
	// Ignore EINTR like os.Remove, see ignoringEINTR in os/file_posix.go
	// for more details.
	for {
		err := syscall.Unlink(path)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}
