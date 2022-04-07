package agent_test

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/crypto/ssh"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/agent"
	"github.com/coder/coder/peer"
	"github.com/coder/coder/peerbroker"
	"github.com/coder/coder/peerbroker/proto"
	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/pty/ptytest"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestAgent(t *testing.T) {
	t.Parallel()
	t.Run("SessionExec", func(t *testing.T) {
		t.Parallel()
		session := setupSSH(t)
		command := "echo test"
		if runtime.GOOS == "windows" {
			command = "cmd.exe /c echo test"
		}
		output, err := session.Output(command)
		require.NoError(t, err)
		require.Equal(t, "test", strings.TrimSpace(string(output)))
	})

	t.Run("GitSSH", func(t *testing.T) {
		t.Parallel()
		session := setupSSH(t)
		command := "echo $GIT_SSH_COMMAND"
		if runtime.GOOS == "windows" {
			command = "cmd.exe /c echo $GIT_SSH_COMMAND"
		}
		output, err := session.Output(command)
		require.NoError(t, err)
		require.Contains(t, string(output), "gitssh --")
	})

	t.Run("SessionTTY", func(t *testing.T) {
		t.Parallel()
		session := setupSSH(t)
		prompt := "$"
		command := "bash"
		if runtime.GOOS == "windows" {
			command = "cmd.exe"
			prompt = ">"
		}
		err := session.RequestPty("xterm", 128, 128, ssh.TerminalModes{})
		require.NoError(t, err)
		ptty := ptytest.New(t)
		require.NoError(t, err)
		session.Stdout = ptty.Output()
		session.Stderr = ptty.Output()
		session.Stdin = ptty.Input()
		err = session.Start(command)
		require.NoError(t, err)
		ptty.ExpectMatch(prompt)
		ptty.WriteLine("echo test")
		ptty.ExpectMatch("test")
		ptty.WriteLine("exit")
		err = session.Wait()
		require.NoError(t, err)
	})
}

func setupSSH(t *testing.T) *ssh.Session {
	client, server := provisionersdk.TransportPipe()
	closer := agent.New(func(ctx context.Context, opts *peer.ConnOptions) (*peerbroker.Listener, error) {
		return peerbroker.Listen(server, nil, opts)
	}, &peer.ConnOptions{
		Logger: slogtest.Make(t, nil).Leveled(slog.LevelDebug),
	})
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
		_ = closer.Close()
	})
	api := proto.NewDRPCPeerBrokerClient(provisionersdk.Conn(client))
	stream, err := api.NegotiateConnection(context.Background())
	require.NoError(t, err)
	conn, err := peerbroker.Dial(stream, []webrtc.ICEServer{}, &peer.ConnOptions{
		Logger: slogtest.Make(t, nil),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})
	agentClient := &agent.Conn{
		Negotiator: api,
		Conn:       conn,
	}
	sshClient, err := agentClient.SSHClient()
	require.NoError(t, err)
	session, err := sshClient.NewSession()
	require.NoError(t, err)

	return session
}
