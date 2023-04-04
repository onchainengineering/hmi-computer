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
	"sync/atomic"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
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
	serveWg sync.WaitGroup
	logger  slog.Logger

	srv *ssh.Server

	Env        map[string]string
	AgentToken func() string

	manifest atomic.Pointer[agentsdk.Manifest]

	connCountVSCode     atomic.Int64
	connCountJetBrains  atomic.Int64
	connCountSSHSession atomic.Int64
}

func NewServer(ctx context.Context, logger slog.Logger, maxTimeout time.Duration) (*Server, error) {
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

	forwardHandler := &ssh.ForwardedTCPHandler{}
	unixForwardHandler := &forwardedUnixHandler{log: logger}

	s := &Server{
		logger: logger,
	}

	s.srv = &ssh.Server{
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip":                   ssh.DirectTCPIPHandler,
			"direct-streamlocal@openssh.com": directStreamLocalHandler,
			"session":                        ssh.DefaultSessionHandler,
		},
		ConnectionFailedCallback: func(_ net.Conn, err error) {
			s.logger.Info(ctx, "ssh connection ended", slog.Error(err))
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
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			return &gossh.ServerConfig{
				NoClientAuth: true,
			}
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": s.sftpHandler,
		},
		MaxTimeout: maxTimeout,
	}

	return s, nil
}

// SetManifest sets the manifest used for starting commands.
func (s *Server) SetManifest(m *agentsdk.Manifest) {
	s.manifest.Store(m)
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
	ctx := session.Context()
	err := s.sessionStart(session)
	var exitError *exec.ExitError
	if xerrors.As(err, &exitError) {
		s.logger.Debug(ctx, "ssh session returned", slog.Error(exitError))
		_ = session.Exit(exitError.ExitCode())
		return
	}
	if err != nil {
		s.logger.Warn(ctx, "ssh session failed", slog.Error(err))
		// This exit code is designed to be unlikely to be confused for a legit exit code
		// from the process.
		_ = session.Exit(MagicSessionErrorCode)
		return
	}
	_ = session.Exit(0)
}

func (s *Server) sessionStart(session ssh.Session) (retErr error) {
	ctx := session.Context()
	env := session.Environ()
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
		return err
	}

	if ssh.AgentRequested(session) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			return xerrors.Errorf("new agent listener: %w", err)
		}
		defer l.Close()
		go ssh.ForwardAgentConnections(l, session)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	sshPty, windowSize, isPty := session.Pty()
	if isPty {
		// Disable minimal PTY emulation set by gliderlabs/ssh (NL-to-CRNL).
		// See https://github.com/coder/coder/issues/3371.
		session.DisablePTYEmulation()

		if !isQuietLogin(session.RawCommand()) {
			manifest := s.manifest.Load()
			if manifest != nil {
				err = showMOTD(session, manifest.MOTDFile)
				if err != nil {
					s.logger.Error(ctx, "show MOTD", slog.Error(err))
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
			return xerrors.Errorf("start command: %w", err)
		}
		var wg sync.WaitGroup
		defer func() {
			defer wg.Wait()
			closeErr := ptty.Close()
			if closeErr != nil {
				s.logger.Warn(ctx, "failed to close tty", slog.Error(closeErr))
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
				}
			}
		}()
		// We don't add input copy to wait group because
		// it won't return until the session is closed.
		go func() {
			_, _ = io.Copy(ptty.Input(), session)
		}()

		// In low parallelism scenarios, the command may exit and we may close
		// the pty before the output copy has started. This can result in the
		// output being lost. To avoid this, we wait for the output copy to
		// start before waiting for the command to exit. This ensures that the
		// output copy goroutine will be scheduled before calling close on the
		// pty. This shouldn't be needed because of `pty.Dup()` below, but it
		// may not be supported on all platforms.
		outputCopyStarted := make(chan struct{})
		ptyOutput := func() io.ReadCloser {
			defer close(outputCopyStarted)
			// Try to dup so we can separate stdin and stdout closure.
			// Once the original pty is closed, the dup will return
			// input/output error once the buffered data has been read.
			stdout, err := ptty.Dup()
			if err == nil {
				return stdout
			}
			// If we can't dup, we shouldn't close
			// the fd since it's tied to stdin.
			return readNopCloser{ptty.Output()}
		}
		wg.Add(1)
		go func() {
			// Ensure data is flushed to session on command exit, if we
			// close the session too soon, we might lose data.
			defer wg.Done()

			stdout := ptyOutput()
			defer stdout.Close()

			_, _ = io.Copy(session, stdout)
		}()
		<-outputCopyStarted

		err = process.Wait()
		var exitErr *exec.ExitError
		// ExitErrors just mean the command we run returned a non-zero exit code, which is normal
		// and not something to be concerned about.  But, if it's something else, we should log it.
		if err != nil && !xerrors.As(err, &exitErr) {
			s.logger.Warn(ctx, "wait error", slog.Error(err))
		}
		return err
	}

	cmd.Stdout = session
	cmd.Stderr = session.Stderr()
	// This blocks forever until stdin is received if we don't
	// use StdinPipe. It's unknown what causes this.
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return xerrors.Errorf("create stdin pipe: %w", err)
	}
	go func() {
		_, _ = io.Copy(stdinPipe, session)
		_ = stdinPipe.Close()
	}()
	err = cmd.Start()
	if err != nil {
		return xerrors.Errorf("start: %w", err)
	}
	return cmd.Wait()
}

type readNopCloser struct{ io.Reader }

// Close implements io.Closer.
func (readNopCloser) Close() error { return nil }

func (s *Server) sftpHandler(session ssh.Session) {
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
	_ = session.Exit(1)
}

// CreateCommand processes raw command input with OpenSSH-like behavior.
// If the script provided is empty, it will default to the users shell.
// This injects environment variables specified by the user at launch too.
func (s *Server) CreateCommand(ctx context.Context, script string, env []string) (*exec.Cmd, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, xerrors.Errorf("get current user: %w", err)
	}
	username := currentUser.Username

	shell, err := usershell.Get(username)
	if err != nil {
		return nil, xerrors.Errorf("get user shell: %w", err)
	}

	manifest := s.manifest.Load()
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

	cmd := exec.CommandContext(ctx, shell, args...)
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
	s.serveWg.Add(1)
	defer s.serveWg.Done()
	return s.srv.Serve(l)
}

func (s *Server) Close() error {
	err := s.srv.Close()
	s.serveWg.Wait()
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
