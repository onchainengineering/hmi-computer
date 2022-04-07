package cli_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"

	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/coderd/coderdtest"
)

func TestGitSSH(t *testing.T) {
	t.Parallel()
	t.Run("Dial", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)

		// get user public key
		keypair, err := client.GitSSHKey(ctx)
		require.NoError(t, err)
		publicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(keypair.PublicKey))
		require.NoError(t, err)

		// start ssh server
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer l.Close()
		publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return ssh.KeysEqual(publicKey, key)
		})
		go func() {
			err := ssh.Serve(l, func(s ssh.Session) {
				t.Log("got authenticated sesion")
				err := s.Exit(0)
				require.NoError(t, err)
			}, publicKeyOption)
			require.NoError(t, err)
		}()

		// start ssh session
		addr, ok := l.Addr().(*net.TCPAddr)
		require.True(t, ok)
		cmd, root := clitest.New(t, "gitssh", "--", fmt.Sprintf("-p%d", addr.Port), "-o", "StrictHostKeyChecking=no", "127.0.0.1")
		clitest.SetupConfig(t, client, root)
		doneChan := make(chan struct{})

		go func() {
			defer close(doneChan)
			err := cmd.ExecuteContext(ctx)
			require.NoError(t, err)
			t.Log("done with gitssh")
		}()
		// wait for server to exit the session
		<-doneChan
	})
}
