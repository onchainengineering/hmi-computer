package turnconn

import (
	"io"
	"net"
	"sync"

	"github.com/pion/logging"
	"github.com/pion/turn/v2"
	"github.com/pion/webrtc/v3"
	"golang.org/x/net/proxy"
	"golang.org/x/xerrors"
)

var (
	reservedHost = "coder"
	credential   = "coder"

	// Proxy is a an ICE Server that uses a special hostname
	// to indicate traffic should be proxied.
	Proxy = webrtc.ICEServer{
		URLs:       []string{"turns:" + reservedHost},
		Username:   "coder",
		Credential: credential,
	}
)

// New constructs a new TURN server binding to the relay address provided.
// The relay address is used to broadcast the location of an accepted connection.
func New(relayAddress *turn.RelayAddressGeneratorStatic) (*Server, error) {
	if relayAddress == nil {
		// Default to localhost.
		relayAddress = &turn.RelayAddressGeneratorStatic{
			RelayAddress: net.IP{127, 0, 0, 1},
			Address:      "127.0.0.1",
		}
	}
	logger := logging.NewDefaultLoggerFactory()
	logger.DefaultLogLevel = logging.LogLevelDisabled
	server := &Server{
		conns:  make(chan net.Conn, 1),
		closed: make(chan struct{}),
	}
	server.listener = &listener{
		srv: server,
	}
	var err error
	server.turn, err = turn.NewServer(turn.ServerConfig{
		AuthHandler: func(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
			return turn.GenerateAuthKey(Proxy.Username, "", credential), true
		},
		ListenerConfigs: []turn.ListenerConfig{{
			Listener:              server.listener,
			RelayAddressGenerator: relayAddress,
		}},
		LoggerFactory: logger,
	})
	if err != nil {
		return nil, xerrors.Errorf("create server: %w", err)
	}

	return server, nil
}

// ProxyDialer accepts a proxy function that's called when the connection
// address matches the reserved host in the "Proxy" ICE server.
//
// This should be passed to WebRTC connections as an ICE dialer.
func ProxyDialer(proxyFunc func() (c net.Conn, err error)) proxy.Dialer {
	return dialer(func(network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if host != reservedHost {
			return proxy.Direct.Dial(network, addr)
		}
		netConn, err := proxyFunc()
		if err != nil {
			return nil, err
		}
		return &Conn{
			localAddress: &net.TCPAddr{
				IP: net.IPv4(127, 0, 0, 1),
			},
			closed: make(chan struct{}),
			Conn:   netConn,
		}, nil
	})
}

// Server accepts and connects TURN allocations.
//
// This is a thin wrapper around pion/turn that pipes
// connections directly to the in-memory handler.
type Server struct {
	listener *listener
	turn     *turn.Server

	closeMutex sync.Mutex
	closed     chan (struct{})
	conns      chan (net.Conn)
}

// Accept consumes a new connection into the TURN server.
// A unique remote address must exist per-connection.
// pion/turn indexes allocations based on the address.
func (s *Server) Accept(nc net.Conn, remoteAddress *net.TCPAddr) *Conn {
	conn := &Conn{
		Conn:          nc,
		remoteAddress: remoteAddress,
		closed:        make(chan struct{}),
	}
	s.conns <- conn
	return conn
}

// Close ends the TURN server.
func (s *Server) Close() error {
	s.closeMutex.Lock()
	defer s.closeMutex.Unlock()
	if s.isClosed() {
		return nil
	}
	err := s.turn.Close()
	close(s.conns)
	close(s.closed)
	return err
}

func (s *Server) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

// listener implements net.Listener for the TURN
// server to consume.
type listener struct {
	srv *Server
}

func (l *listener) Accept() (net.Conn, error) {
	conn, ok := <-l.srv.conns
	if !ok {
		return nil, io.EOF
	}
	return conn, nil
}

func (*listener) Close() error {
	return nil
}

func (*listener) Addr() net.Addr {
	return nil
}

type Conn struct {
	net.Conn
	closed        chan struct{}
	localAddress  *net.TCPAddr
	remoteAddress *net.TCPAddr
}

func (c *Conn) LocalAddr() net.Addr {
	return c.localAddress
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddress
}

// Closed returns a channel which is closed when
// the connection is.
func (c *Conn) Closed() <-chan struct{} {
	return c.closed
}

func (c *Conn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return c.Conn.Close()
}

type dialer func(network, addr string) (c net.Conn, err error)

func (d dialer) Dial(network, addr string) (c net.Conn, err error) {
	return d(network, addr)
}
