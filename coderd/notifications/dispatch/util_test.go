package dispatch_test

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"golang.org/x/xerrors"
)

type Config struct {
	AuthMechanisms                                       []string
	AcceptedIdentity, AcceptedUsername, AcceptedPassword string
}

type Message struct {
	AuthMech                     string
	Identity, Username, Password string // Auth
	From                         string
	To                           []string // Address
	Subject, Contents            string   // Content
}

// The Backend implements SMTP server methods.
type Backend struct {
	cfg Config

	mu      sync.Mutex
	lastMsg *Message
}

func NewBackend(cfg Config) *Backend {
	return &Backend{
		cfg: cfg,
	}
}

// NewSession is called after client greeting (EHLO, HELO).
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{conn: c, backend: b}, nil
}

func (b *Backend) LastMessage() *Message {
	return b.lastMsg
}

func (b *Backend) Reset() {
	b.lastMsg = nil
}

// A Session is returned after successful login.
type Session struct {
	conn    *smtp.Conn
	backend *Backend
}

// AuthMechanisms returns a slice of available auth mechanisms; only PLAIN is
// supported in this example.
func (s *Session) AuthMechanisms() []string {
	return s.backend.cfg.AuthMechanisms
}

// Auth is the handler for supported authenticators.
func (s *Session) Auth(mech string) (sasl.Server, error) {
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()

	if s.backend.lastMsg == nil {
		s.backend.lastMsg = &Message{AuthMech: mech}
	}

	switch mech {
	case sasl.Plain:
		return sasl.NewPlainServer(func(identity, username, password string) error {
			s.backend.lastMsg.Identity = identity
			s.backend.lastMsg.Username = username
			s.backend.lastMsg.Password = password

			if identity != s.backend.cfg.AcceptedIdentity {
				return xerrors.Errorf("unknown identity: %q", identity)
			}
			if username != s.backend.cfg.AcceptedUsername {
				return xerrors.Errorf("unknown user: %q", username)
			}
			if password != s.backend.cfg.AcceptedPassword {
				return xerrors.Errorf("incorrect password for username: %q", username)
			}

			return nil
		}), nil
	case sasl.Login:
		return sasl.NewLoginServer(func(username, password string) error {
			s.backend.lastMsg.Username = username
			s.backend.lastMsg.Password = password

			if username != s.backend.cfg.AcceptedUsername {
				return xerrors.Errorf("unknown user: %q", username)
			}
			if password != s.backend.cfg.AcceptedPassword {
				return xerrors.Errorf("incorrect password for username: %q", username)
			}

			return nil
		}), nil
	default:
		return nil, xerrors.Errorf("unexpected auth mechanism: %q", mech)
	}
}

func (s *Session) Mail(from string, _ *smtp.MailOptions) error {
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()

	if s.backend.lastMsg == nil {
		s.backend.lastMsg = &Message{}
	}

	s.backend.lastMsg.From = from
	return nil
}

func (s *Session) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()

	s.backend.lastMsg.To = append(s.backend.lastMsg.To, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()

	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	s.backend.lastMsg.Contents = string(b)

	return nil
}

func (s *Session) Reset() {}

// TODO: test logout failure
func (s *Session) Logout() error { return nil }

func createMockServer(be *Backend) (*smtp.Server, net.Listener, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, err
	}

	s := smtp.NewServer(be)

	s.Addr = l.Addr().String()
	s.Domain = "localhost"
	s.WriteTimeout = 10 * time.Second
	s.ReadTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	return s, l, nil
}
