package agentssh

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"

	"cdr.dev/slog"

	"github.com/coder/coder/agent/usershell"
	"github.com/coder/coder/codersdk/agentsdk"
	"github.com/coder/coder/pty"
)

const (
	// MagicSessionErrorCode indicates that something went wrong with the session, rather than the
	// command just returning a nonzero exit code, and is chosen as an arbitrary, high number
	// unlikely to shadow other exit codes, which are typically 1, 2, 3, etc.
	MagicSessionErrorCode = 229

	// MagicSessionTypeEnvironmentVariable is used to track the purpose behind an SSH connection.
	// This is stripped from any commands being executed, and is counted towards connection stats.
	MagicSessionTypeEnvironmentVariable = "CODER_SSH_SESSION_TYPE"
	// MagicSessionTypeVSCode is set in the SSH config by the VS Code extension to identify itself.
	MagicSessionTypeVSCode = "vscode"
	// MagicSessionTypeJetBrains is set in the SSH config by the JetBrains extension to identify itself.
	MagicSessionTypeJetBrains = "jetbrains"
)

type Server struct {
	mu        sync.RWMutex // Protects following.
	fs        afero.Fs
	listeners map[net.Listener]struct{}
	conns     map[net.Conn]struct{}
	sessions  map[ssh.Session]struct{}
	closing   chan struct{}
	// Wait for goroutines to exit, waited without
	// a lock on mu but protected by closing.
	wg sync.WaitGroup

	logger       slog.Logger
	srv          *ssh.Server
	x11SocketDir string

	Env        map[string]string
	AgentToken func() string
	Manifest   *atomic.Pointer[agentsdk.Manifest]

	connCountVSCode     atomic.Int64
	connCountJetBrains  atomic.Int64
	connCountSSHSession atomic.Int64

	metrics *sshServerMetrics
}

func NewServer(ctx context.Context, logger slog.Logger, prometheusRegistry *prometheus.Registry, fs afero.Fs, maxTimeout time.Duration, x11SocketDir string) (*Server, error) {
	// Clients' should ignore the host key when connecting.
	// The agent needs to authenticate with coderd to SSH,
	// so SSH authentication doesn't improve security.
	randomHostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	randomSigner, err := gossh.NewSignerFromKey(randomHostKey)
	if err != nil {
		return nil, err
	}
	if x11SocketDir == "" {
		x11SocketDir = filepath.Join(os.TempDir(), ".X11-unix")
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	unixForwardHandler := &forwardedUnixHandler{log: logger}

	metrics := newSSHServerMetrics(prometheusRegistry)
	s := &Server{
		listeners:    make(map[net.Listener]struct{}),
		fs:           fs,
		conns:        make(map[net.Conn]struct{}),
		sessions:     make(map[ssh.Session]struct{}),
		logger:       logger,
		x11SocketDir: x11SocketDir,

		metrics: metrics,
	}

	s.srv = &ssh.Server{
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip":                   ssh.DirectTCPIPHandler,
			"direct-streamlocal@openssh.com": directStreamLocalHandler,
			"session":                        ssh.DefaultSessionHandler,
		},
		ConnectionFailedCallback: func(_ net.Conn, err error) {
			s.logger.Warn(ctx, "ssh connection failed", slog.Error(err))
			metrics.connectionFailedCallback.Add(1)
		},
		Handler:     s.sessionHandler,
		HostSigners: []ssh.Signer{randomSigner},
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			// Allow local port forwarding all!
			s.logger.Debug(ctx, "local port forward",
				slog.F("destination-host", destinationHost),
				slog.F("destination-port", destinationPort))
			return true
		},
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			return true
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			// Allow reverse port forwarding all!
			s.logger.Debug(ctx, "local port forward",
				slog.F("bind-host", bindHost),
				slog.F("bind-port", bindPort))
			return true
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":                          forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward":                   forwardHandler.HandleSSHRequest,
			"streamlocal-forward@openssh.com":        unixForwardHandler.HandleSSHRequest,
			"cancel-streamlocal-forward@openssh.com": unixForwardHandler.HandleSSHRequest,
		},
		X11Callback: s.x11Callback,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			return &gossh.ServerConfig{
				NoClientAuth: true,
			}
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": s.sessionHandler,
		},
		MaxTimeout: maxTimeout,
	}

	return s, nil
}

type ConnStats struct {
	Sessions  int64
	VSCode    int64
	JetBrains int64
}

func (s *Server) ConnStats() ConnStats {
	return ConnStats{
		Sessions:  s.connCountSSHSession.Load(),
		VSCode:    s.connCountVSCode.Load(),
		JetBrains: s.connCountJetBrains.Load(),
	}
}

func (s *Server) sessionHandler(session ssh.Session) {
	if !s.trackSession(session, true) {
		// See (*Server).Close() for why we call Close instead of Exit.
		_ = session.Close()
		return
	}
	defer s.trackSession(session, false)

	ctx := session.Context()

	extraEnv := make([]string, 0)
	x11, hasX11 := session.X11()
	if hasX11 {
		handled := s.x11Handler(session.Context(), x11)
		if !handled {
			_ = session.Exit(1)
			return
		}
		extraEnv = append(extraEnv, fmt.Sprintf("DISPLAY=:%d.0", x11.ScreenNumber))
	}

	switch ss := session.Subsystem(); ss {
	case "":
	case "sftp":
		s.sftpHandler(session)
		return
	default:
		s.logger.Debug(ctx, "unsupported subsystem", slog.F("subsystem", ss))
		_ = session.Exit(1)
		return
	}

	m := metricsForSession(s.metrics.sessions, magicType(session))
	err := s.sessionStart(session, m, extraEnv)
	var exitError *exec.ExitError
	if xerrors.As(err, &exitError) {
		s.logger.Warn(ctx, "ssh session returned", slog.Error(exitError))
		m.sessionError.Add(1)
		_ = session.Exit(exitError.ExitCode())
		return
	}
	if err != nil {
		s.logger.Warn(ctx, "ssh session failed", slog.Error(err))
		// This exit code is designed to be unlikely to be confused for a legit exit code
		// from the process.
		m.sessionError.Add(1)
		_ = session.Exit(MagicSessionErrorCode)
		return
	}
	_ = session.Exit(0)
}

func magicType(session ssh.Session) string {
	for _, kv := range session.Environ() {
		if !strings.HasPrefix(kv, MagicSessionTypeEnvironmentVariable) {
			continue
		}
		return strings.TrimPrefix(kv, MagicSessionTypeEnvironmentVariable+"=")
	}
	return ""
}

func (s *Server) sessionStart(session ssh.Session, m sessionMetricsObject, extraEnv []string) (retErr error) {
	ctx := session.Context()
	env := append(session.Environ(), extraEnv...)
	var magicType string
	for index, kv := range env {
		if !strings.HasPrefix(kv, MagicSessionTypeEnvironmentVariable) {
			continue
		}
		magicType = strings.TrimPrefix(kv, MagicSessionTypeEnvironmentVariable+"=")
		env = append(env[:index], env[index+1:]...)
	}
	switch magicType {
	case MagicSessionTypeVSCode:
		s.connCountVSCode.Add(1)
		defer s.connCountVSCode.Add(-1)
	case MagicSessionTypeJetBrains:
		s.connCountJetBrains.Add(1)
		defer s.connCountJetBrains.Add(-1)
	case "":
		s.connCountSSHSession.Add(1)
		defer s.connCountSSHSession.Add(-1)
	default:
		s.logger.Warn(ctx, "invalid magic ssh session type specified", slog.F("type", magicType))
	}

	cmd, err := s.CreateCommand(ctx, session.RawCommand(), env)
	if err != nil {
		m.agentCreateCommandError.Add(1)
		return err
	}

	if ssh.AgentRequested(session) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			m.agentListenerError.Add(1)
			return xerrors.Errorf("new agent listener: %w", err)
		}
		defer l.Close()
		go ssh.ForwardAgentConnections(l, session)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	sshPty, windowSize, isPty := session.Pty()
	if isPty {
		return s.startPTYSession(session, m, cmd, sshPty, windowSize)
	}
	return startNonPTYSession(session, m, cmd.AsExec())
}

func startNonPTYSession(session ssh.Session, m sessionMetricsObject, cmd *exec.Cmd) error {
	m.startNonPTYSession.Add(1)

	cmd.Stdout = session
	cmd.Stderr = session.Stderr()
	// This blocks forever until stdin is received if we don't
	// use StdinPipe. It's unknown what causes this.
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		m.nonPTYStdinPipeError.Add(1)
		return xerrors.Errorf("create stdin pipe: %w", err)
	}
	go func() {
		_, err := io.Copy(stdinPipe, session)
		if err != nil {
			m.nonPTYStdinIoCopyError.Add(1)
		}
		_ = stdinPipe.Close()
	}()
	err = cmd.Start()
	if err != nil {
		m.nonPTYCmdStartError.Add(1)
		return xerrors.Errorf("start: %w", err)
	}
	return cmd.Wait()
}

// ptySession is the interface to the ssh.Session that startPTYSession uses
// we use an interface here so that we can fake it in tests.
type ptySession interface {
	io.ReadWriter
	Context() ssh.Context
	DisablePTYEmulation()
	RawCommand() string
}

func (s *Server) startPTYSession(session ptySession, m sessionMetricsObject, cmd *pty.Cmd, sshPty ssh.Pty, windowSize <-chan ssh.Window) (retErr error) {
	m.startPTYSession.Add(1)

	ctx := session.Context()
	// Disable minimal PTY emulation set by gliderlabs/ssh (NL-to-CRNL).
	// See https://github.com/coder/coder/issues/3371.
	session.DisablePTYEmulation()

	if !isQuietLogin(session.RawCommand()) {
		manifest := s.Manifest.Load()
		if manifest != nil {
			err := showMOTD(session, manifest.MOTDFile)
			if err != nil {
				s.logger.Error(ctx, "show MOTD", slog.Error(err))
				m.ptyMotdError.Add(1)
			}
		} else {
			s.logger.Warn(ctx, "metadata lookup failed, unable to show MOTD")
		}
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", sshPty.Term))

	// The pty package sets `SSH_TTY` on supported platforms.
	ptty, process, err := pty.Start(cmd, pty.WithPTYOption(
		pty.WithSSHRequest(sshPty),
		pty.WithLogger(slog.Stdlib(ctx, s.logger, slog.LevelInfo)),
	))
	if err != nil {
		m.ptyCmdStartError.Add(1)
		return xerrors.Errorf("start command: %w", err)
	}
	defer func() {
		closeErr := ptty.Close()
		if closeErr != nil {
			s.logger.Warn(ctx, "failed to close tty", slog.Error(closeErr))
			m.ptyCloseError.Add(1)
			if retErr == nil {
				retErr = closeErr
			}
		}
	}()
	go func() {
		for win := range windowSize {
			resizeErr := ptty.Resize(uint16(win.Height), uint16(win.Width))
			// If the pty is closed, then command has exited, no need to log.
			if resizeErr != nil && !errors.Is(resizeErr, pty.ErrClosed) {
				s.logger.Warn(ctx, "failed to resize tty", slog.Error(resizeErr))
				m.ptyResizeError.Add(1)
			}
		}
	}()

	go func() {
		_, err := io.Copy(ptty.InputWriter(), session)
		if err != nil {
			m.ptyInputIoCopyError.Add(1)
		}
	}()

	// We need to wait for the command output to finish copying.  It's safe to
	// just do this copy on the main handler goroutine because one of two things
	// will happen:
	//
	// 1. The command completes & closes the TTY, which then triggers an error
	//    after we've Read() all the buffered data from the PTY.
	// 2. The client hangs up, which cancels the command's Context, and go will
	//    kill the command's process.  This then has the same effect as (1).
	n, err := io.Copy(session, ptty.OutputReader())
	s.logger.Debug(ctx, "copy output done", slog.F("bytes", n), slog.Error(err))
	if err != nil {
		m.ptyOutputIoCopyError.Add(1)
		return xerrors.Errorf("copy error: %w", err)
	}
	// We've gotten all the output, but we need to wait for the process to
	// complete so that we can get the exit code.  This returns
	// immediately if the TTY was closed as part of the command exiting.
	err = process.Wait()
	var exitErr *exec.ExitError
	// ExitErrors just mean the command we run returned a non-zero exit code, which is normal
	// and not something to be concerned about.  But, if it's something else, we should log it.
	if err != nil && !xerrors.As(err, &exitErr) {
		s.logger.Warn(ctx, "wait error", slog.Error(err))
		m.ptyWaitError.Add(1)
	}
	if err != nil {
		return xerrors.Errorf("process wait: %w", err)
	}
	return nil
}

func (s *Server) sftpHandler(session ssh.Session) {
	s.metrics.sftpHandler.Add(1)

	ctx := session.Context()

	// Typically sftp sessions don't request a TTY, but if they do,
	// we must ensure the gliderlabs/ssh CRLF emulation is disabled.
	// Otherwise sftp will be broken. This can happen if a user sets
	// `RequestTTY force` in their SSH config.
	session.DisablePTYEmulation()

	var opts []sftp.ServerOption
	// Change current working directory to the users home
	// directory so that SFTP connections land there.
	homedir, err := userHomeDir()
	if err != nil {
		s.logger.Warn(ctx, "get sftp working directory failed, unable to get home dir", slog.Error(err))
	} else {
		opts = append(opts, sftp.WithServerWorkingDirectory(homedir))
	}

	server, err := sftp.NewServer(session, opts...)
	if err != nil {
		s.logger.Debug(ctx, "initialize sftp server", slog.Error(err))
		return
	}
	defer server.Close()

	err = server.Serve()
	if errors.Is(err, io.EOF) {
		// Unless we call `session.Exit(0)` here, the client won't
		// receive `exit-status` because `(*sftp.Server).Close()`
		// calls `Close()` on the underlying connection (session),
		// which actually calls `channel.Close()` because it isn't
		// wrapped. This causes sftp clients to receive a non-zero
		// exit code. Typically sftp clients don't echo this exit
		// code but `scp` on macOS does (when using the default
		// SFTP backend).
		_ = session.Exit(0)
		return
	}
	s.logger.Warn(ctx, "sftp server closed with error", slog.Error(err))
	s.metrics.sftpServerError.Add(1)
	_ = session.Exit(1)
}

// CreateCommand processes raw command input with OpenSSH-like behavior.
// If the script provided is empty, it will default to the users shell.
// This injects environment variables specified by the user at launch too.
func (s *Server) CreateCommand(ctx context.Context, script string, env []string) (*pty.Cmd, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, xerrors.Errorf("get current user: %w", err)
	}
	username := currentUser.Username

	shell, err := usershell.Get(username)
	if err != nil {
		return nil, xerrors.Errorf("get user shell: %w", err)
	}

	manifest := s.Manifest.Load()
	if manifest == nil {
		return nil, xerrors.Errorf("no metadata was provided")
	}

	// OpenSSH executes all commands with the users current shell.
	// We replicate that behavior for IDE support.
	caller := "-c"
	if runtime.GOOS == "windows" {
		caller = "/c"
	}
	args := []string{caller, script}

	// gliderlabs/ssh returns a command slice of zero
	// when a shell is requested.
	if len(script) == 0 {
		args = []string{}
		if runtime.GOOS != "windows" {
			// On Linux and macOS, we should start a login
			// shell to consume juicy environment variables!
			args = append(args, "-l")
		}
	}

	cmd := pty.CommandContext(ctx, shell, args...)
	cmd.Dir = manifest.Directory

	// If the metadata directory doesn't exist, we run the command
	// in the users home directory.
	_, err = os.Stat(cmd.Dir)
	if cmd.Dir == "" || err != nil {
		// Default to user home if a directory is not set.
		homedir, err := userHomeDir()
		if err != nil {
			return nil, xerrors.Errorf("get home dir: %w", err)
		}
		cmd.Dir = homedir
	}
	cmd.Env = append(os.Environ(), env...)
	executablePath, err := os.Executable()
	if err != nil {
		return nil, xerrors.Errorf("getting os executable: %w", err)
	}
	// Set environment variables reliable detection of being inside a
	// Coder workspace.
	cmd.Env = append(cmd.Env, "CODER=true")
	cmd.Env = append(cmd.Env, fmt.Sprintf("USER=%s", username))
	// Git on Windows resolves with UNIX-style paths.
	// If using backslashes, it's unable to find the executable.
	unixExecutablePath := strings.ReplaceAll(executablePath, "\\", "/")
	cmd.Env = append(cmd.Env, fmt.Sprintf(`GIT_SSH_COMMAND=%s gitssh --`, unixExecutablePath))

	// Specific Coder subcommands require the agent token exposed!
	cmd.Env = append(cmd.Env, fmt.Sprintf("CODER_AGENT_TOKEN=%s", s.AgentToken()))

	// Set SSH connection environment variables (these are also set by OpenSSH
	// and thus expected to be present by SSH clients). Since the agent does
	// networking in-memory, trying to provide accurate values here would be
	// nonsensical. For now, we hard code these values so that they're present.
	srcAddr, srcPort := "0.0.0.0", "0"
	dstAddr, dstPort := "0.0.0.0", "0"
	cmd.Env = append(cmd.Env, fmt.Sprintf("SSH_CLIENT=%s %s %s", srcAddr, srcPort, dstPort))
	cmd.Env = append(cmd.Env, fmt.Sprintf("SSH_CONNECTION=%s %s %s %s", srcAddr, srcPort, dstAddr, dstPort))

	// This adds the ports dialog to code-server that enables
	// proxying a port dynamically.
	cmd.Env = append(cmd.Env, fmt.Sprintf("VSCODE_PROXY_URI=%s", manifest.VSCodePortProxyURI))

	// Hide Coder message on code-server's "Getting Started" page
	cmd.Env = append(cmd.Env, "CS_DISABLE_GETTING_STARTED_OVERRIDE=true")

	// Load environment variables passed via the agent.
	// These should override all variables we manually specify.
	for envKey, value := range manifest.EnvironmentVariables {
		// Expanding environment variables allows for customization
		// of the $PATH, among other variables. Customers can prepend
		// or append to the $PATH, so allowing expand is required!
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, os.ExpandEnv(value)))
	}

	// Agent-level environment variables should take over all!
	// This is used for setting agent-specific variables like "CODER_AGENT_TOKEN".
	for envKey, value := range s.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, value))
	}

	return cmd, nil
}

func (s *Server) Serve(l net.Listener) error {
	defer l.Close()

	s.trackListener(l, true)
	defer s.trackListener(l, false)
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(l, conn)
	}
}

func (s *Server) handleConn(l net.Listener, c net.Conn) {
	defer c.Close()

	if !s.trackConn(l, c, true) {
		// Server is closed or we no longer want
		// connections from this listener.
		s.logger.Debug(context.Background(), "received connection after server closed")
		return
	}
	defer s.trackConn(l, c, false)

	s.srv.HandleConn(c)
}

// trackListener registers the listener with the server. If the server is
// closing, the function will block until the server is closed.
//
//nolint:revive
func (s *Server) trackListener(l net.Listener, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		for s.closing != nil {
			closing := s.closing
			// Wait until close is complete before
			// serving a new listener.
			s.mu.Unlock()
			<-closing
			s.mu.Lock()
		}
		s.wg.Add(1)
		s.listeners[l] = struct{}{}
		return
	}
	s.wg.Done()
	delete(s.listeners, l)
}

// trackConn registers the connection with the server. If the server is
// closed or the listener is closed, the connection is not registered
// and should be closed.
//
//nolint:revive
func (s *Server) trackConn(l net.Listener, c net.Conn, add bool) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		found := false
		for ll := range s.listeners {
			if l == ll {
				found = true
				break
			}
		}
		if s.closing != nil || !found {
			// Server or listener closed.
			return false
		}
		s.wg.Add(1)
		s.conns[c] = struct{}{}
		return true
	}
	s.wg.Done()
	delete(s.conns, c)
	return true
}

// trackSession registers the session with the server. If the server is
// closing, the session is not registered and should be closed.
//
//nolint:revive
func (s *Server) trackSession(ss ssh.Session, add bool) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		if s.closing != nil {
			// Server closed.
			return false
		}
		s.sessions[ss] = struct{}{}
		return true
	}
	delete(s.sessions, ss)
	return true
}

// Close the server and all active connections. Server can be re-used
// after Close is done.
func (s *Server) Close() error {
	s.mu.Lock()

	// Guard against multiple calls to Close and
	// accepting new connections during close.
	if s.closing != nil {
		s.mu.Unlock()
		return xerrors.New("server is closing")
	}
	s.closing = make(chan struct{})

	// Close all active sessions to gracefully
	// terminate client connections.
	for ss := range s.sessions {
		// We call Close on the underlying channel here because we don't
		// want to send an exit status to the client (via Exit()).
		// Typically OpenSSH clients will return 255 as the exit status.
		_ = ss.Close()
	}

	// Close all active listeners and connections.
	for l := range s.listeners {
		_ = l.Close()
	}
	for c := range s.conns {
		_ = c.Close()
	}

	// Close the underlying SSH server.
	err := s.srv.Close()

	s.mu.Unlock()
	s.wg.Wait() // Wait for all goroutines to exit.

	s.mu.Lock()
	close(s.closing)
	s.closing = nil
	s.mu.Unlock()

	return err
}

// Shutdown gracefully closes all active SSH connections and stops
// accepting new connections.
//
// Shutdown is not implemented.
func (*Server) Shutdown(_ context.Context) error {
	// TODO(mafredri): Implement shutdown, SIGHUP running commands, etc.
	return nil
}

// isQuietLogin checks if the SSH server should perform a quiet login or not.
//
// https://github.com/openssh/openssh-portable/blob/25bd659cc72268f2858c5415740c442ee950049f/session.c#L816
func isQuietLogin(rawCommand string) bool {
	// We are always quiet unless this is a login shell.
	if len(rawCommand) != 0 {
		return true
	}

	// Best effort, if we can't get the home directory,
	// we can't lookup .hushlogin.
	homedir, err := userHomeDir()
	if err != nil {
		return false
	}

	_, err = os.Stat(filepath.Join(homedir, ".hushlogin"))
	return err == nil
}

// showMOTD will output the message of the day from
// the given filename to dest, if the file exists.
//
// https://github.com/openssh/openssh-portable/blob/25bd659cc72268f2858c5415740c442ee950049f/session.c#L784
func showMOTD(dest io.Writer, filename string) error {
	if filename == "" {
		return nil
	}

	f, err := os.Open(filename)
	if err != nil {
		if xerrors.Is(err, os.ErrNotExist) {
			// This is not an error, there simply isn't a MOTD to show.
			return nil
		}
		return xerrors.Errorf("open MOTD: %w", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		// Carriage return ensures each line starts
		// at the beginning of the terminal.
		_, err = fmt.Fprint(dest, s.Text()+"\r\n")
		if err != nil {
			return xerrors.Errorf("write MOTD: %w", err)
		}
	}
	if err := s.Err(); err != nil {
		return xerrors.Errorf("read MOTD: %w", err)
	}

	return nil
}

// userHomeDir returns the home directory of the current user, giving
// priority to the $HOME environment variable.
func userHomeDir() (string, error) {
	// First we check the environment.
	homedir, err := os.UserHomeDir()
	if err == nil {
		return homedir, nil
	}

	// As a fallback, we try the user information.
	u, err := user.Current()
	if err != nil {
		return "", xerrors.Errorf("current user: %w", err)
	}
	return u.HomeDir, nil
}
