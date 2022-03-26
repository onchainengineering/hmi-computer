package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	"github.com/coreos/go-systemd/daemon"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/cli/config"
	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/databasefake"
	"github.com/coder/coder/coderd/tunnel"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/provisioner/terraform"
	"github.com/coder/coder/provisionerd"
	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/provisionersdk/proto"
)

func start() *cobra.Command {
	var (
		accessURL              string
		address                string
		cacheDir               string
		dev                    bool
		postgresURL            string
		provisionerDaemonCount uint8
		tlsCertFile            string
		tlsClientCAFile        string
		tlsClientAuth          string
		tlsEnable              bool
		tlsKeyFile             string
		tlsMinVersion          string
		useTunnel              bool
	)
	root := &cobra.Command{
		Use: "start",
		RunE: func(cmd *cobra.Command, args []string) error {
			printLogo(cmd)
			if postgresURL == "" {
				// Default to the environment variable!
				postgresURL = os.Getenv("CODER_PG_CONNECTION_URL")
			}

			listener, err := net.Listen("tcp", address)
			if err != nil {
				return xerrors.Errorf("listen %q: %w", address, err)
			}
			defer listener.Close()

			if tlsEnable {
				listener, err = configureTLS(listener, tlsMinVersion, tlsClientAuth, tlsCertFile, tlsKeyFile, tlsClientCAFile)
				if err != nil {
					return xerrors.Errorf("configure tls: %w", err)
				}
			}

			tcpAddr, valid := listener.Addr().(*net.TCPAddr)
			if !valid {
				return xerrors.New("must be listening on tcp")
			}
			// If just a port is specified, assume localhost.
			if tcpAddr.IP.IsUnspecified() {
				tcpAddr.IP = net.IPv4(127, 0, 0, 1)
			}

			localURL := &url.URL{
				Scheme: "http",
				Host:   tcpAddr.String(),
			}
			if tlsEnable {
				localURL.Scheme = "https"
			}
			if accessURL == "" {
				accessURL = localURL.String()
			}
			var tunnelErr <-chan error
			// If we're attempting to tunnel in dev-mode, the access URL
			// needs to be changed to use the tunnel.
			if dev && useTunnel {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), cliui.Styles.Paragraph.Render("Coder requires a network endpoint that can be accessed by provisioned workspaces. In dev mode, a free tunnel can be created for you. This will expose your Coder deployment to the internet.")+"\n")

				_, err = cliui.Prompt(cmd, cliui.PromptOptions{
					Text:      "Would you like Coder to start a tunnel for simple setup?",
					IsConfirm: true,
				})
				if err == nil {
					accessURL, tunnelErr, err = tunnel.New(cmd.Context(), localURL.String())
					if err != nil {
						return xerrors.Errorf("create tunnel: %w", err)
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), cliui.Styles.Paragraph.Render(cliui.Styles.Wrap.Render(cliui.Styles.Prompt.String()+`Tunnel started. Your deployment is accessible at:`))+"\n  "+cliui.Styles.Field.Render(accessURL))
				}
			}
			validator, err := idtoken.NewValidator(cmd.Context(), option.WithoutAuthentication())
			if err != nil {
				return err
			}

			accessURLParsed, err := url.Parse(accessURL)
			if err != nil {
				return xerrors.Errorf("parse access url %q: %w", accessURL, err)
			}
			logger := slog.Make(sloghuman.Sink(os.Stderr))
			options := &coderd.Options{
				AccessURL:            accessURLParsed,
				Logger:               logger.Named("coderd"),
				Database:             databasefake.New(),
				Pubsub:               database.NewPubsubInMemory(),
				GoogleTokenValidator: validator,
			}

			if !dev {
				sqlDB, err := sql.Open("postgres", postgresURL)
				if err != nil {
					return xerrors.Errorf("dial postgres: %w", err)
				}
				err = sqlDB.Ping()
				if err != nil {
					return xerrors.Errorf("ping postgres: %w", err)
				}
				err = database.MigrateUp(sqlDB)
				if err != nil {
					return xerrors.Errorf("migrate up: %w", err)
				}
				options.Database = database.New(sqlDB)
				options.Pubsub, err = database.NewPubsub(cmd.Context(), sqlDB, postgresURL)
				if err != nil {
					return xerrors.Errorf("create pubsub: %w", err)
				}
			}

			handler, closeCoderd := coderd.New(options)
			client := codersdk.New(localURL)
			if tlsEnable {
				// Secure transport isn't needed for locally communicating!
				client.HTTPClient.Transport = &http.Transport{
					TLSClientConfig: &tls.Config{
						//nolint:gosec
						InsecureSkipVerify: true,
					},
				}
			}

			provisionerDaemons := make([]*provisionerd.Server, 0)
			for i := uint8(0); i < provisionerDaemonCount; i++ {
				daemonClose, err := newProvisionerDaemon(cmd.Context(), client, logger, cacheDir)
				if err != nil {
					return xerrors.Errorf("create provisioner daemon: %w", err)
				}
				provisionerDaemons = append(provisionerDaemons, daemonClose)
			}
			defer func() {
				for _, provisionerDaemon := range provisionerDaemons {
					_ = provisionerDaemon.Close()
				}
			}()

			errCh := make(chan error, 1)
			shutdownConnsCtx, shutdownConns := context.WithCancel(cmd.Context())
			defer shutdownConns()
			go func() {
				defer close(errCh)
				server := http.Server{
					Handler: handler,
					BaseContext: func(_ net.Listener) context.Context {
						return shutdownConnsCtx
					},
				}
				errCh <- server.Serve(listener)
			}()

			config := createConfig(cmd)

			if dev {
				err = createFirstUser(cmd, client, config)
				if err != nil {
					return xerrors.Errorf("create first user: %w", err)
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), cliui.Styles.Paragraph.Render(cliui.Styles.Wrap.Render(cliui.Styles.Prompt.String()+`Started in `+
					cliui.Styles.Field.Render("dev")+` mode. All data is in-memory! Do not use in production. Press `+cliui.Styles.Field.Render("ctrl+c")+` to clean up provisioned infrastructure.`))+
					`
`+
					cliui.Styles.Paragraph.Render(cliui.Styles.Wrap.Render(cliui.Styles.Prompt.String()+`Run `+cliui.Styles.Code.Render("coder projects init")+" in a new terminal to get started.\n"))+`
`)
			} else {
				// This is helpful for tests, but can be silently ignored.
				// Coder may be ran as users that don't have permission to write in the homedir,
				// such as via the systemd service.
				_ = config.URL().Write(client.URL.String())

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), cliui.Styles.Paragraph.Render(cliui.Styles.Wrap.Render(cliui.Styles.Prompt.String()+`Started in `+
					cliui.Styles.Field.Render("production")+` mode. All data is stored in the PostgreSQL provided! Press `+cliui.Styles.Field.Render("ctrl+c")+` to gracefully shutdown.`))+"\n")

				hasFirstUser, err := client.HasFirstUser(cmd.Context())
				if !hasFirstUser && err == nil {
					// This could fail for a variety of TLS-related reasons.
					// This is a helpful starter message, and not critical for user interaction.
					_, _ = fmt.Fprint(cmd.OutOrStdout(), cliui.Styles.Paragraph.Render(cliui.Styles.Wrap.Render(cliui.Styles.FocusedPrompt.String()+`Run `+cliui.Styles.Code.Render("coder login "+client.URL.String())+" in a new terminal to get started.\n")))
				}
			}

			// Updates the systemd status from activating to activated.
			_, err = daemon.SdNotify(false, daemon.SdNotifyReady)
			if err != nil {
				return xerrors.Errorf("notify systemd: %w", err)
			}

			stopChan := make(chan os.Signal, 1)
			defer signal.Stop(stopChan)
			signal.Notify(stopChan, os.Interrupt)
			select {
			case <-cmd.Context().Done():
				closeCoderd()
				return cmd.Context().Err()
			case err := <-tunnelErr:
				return err
			case err := <-errCh:
				closeCoderd()
				return err
			case <-stopChan:
			}
			signal.Stop(stopChan)
			_, err = daemon.SdNotify(false, daemon.SdNotifyStopping)
			if err != nil {
				return xerrors.Errorf("notify systemd: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n\n"+cliui.Styles.Bold.Render("Interrupt caught. Gracefully exiting..."))

			if dev {
				workspaces, err := client.WorkspacesByUser(cmd.Context(), "")
				if err != nil {
					return xerrors.Errorf("get workspaces: %w", err)
				}
				for _, workspace := range workspaces {
					before := time.Now()
					build, err := client.CreateWorkspaceBuild(cmd.Context(), workspace.ID, codersdk.CreateWorkspaceBuildRequest{
						Transition: database.WorkspaceTransitionDelete,
					})
					if err != nil {
						return xerrors.Errorf("delete workspace: %w", err)
					}

					_, err = cliui.Job(cmd, cliui.JobOptions{
						Title: fmt.Sprintf("Deleting workspace %s...", cliui.Styles.Keyword.Render(workspace.Name)),
						Fetch: func() (codersdk.ProvisionerJob, error) {
							build, err := client.WorkspaceBuild(cmd.Context(), build.ID)
							return build.Job, err
						},
						Cancel: func() error {
							return client.CancelWorkspaceBuild(cmd.Context(), build.ID)
						},
						Logs: func() (<-chan codersdk.ProvisionerJobLog, error) {
							return client.WorkspaceBuildLogsAfter(cmd.Context(), build.ID, before)
						},
					})
					if err != nil {
						return xerrors.Errorf("delete workspace %s: %w", workspace.Name, err)
					}
				}
			}

			for _, provisionerDaemon := range provisionerDaemons {
				spin := spinner.New(spinner.CharSets[5], 100*time.Millisecond)
				spin.Writer = cmd.OutOrStdout()
				spin.Suffix = cliui.Styles.Keyword.Render(" Shutting down provisioner daemon...")
				spin.Start()
				err = provisionerDaemon.Shutdown(cmd.Context())
				if err != nil {
					spin.FinalMSG = cliui.Styles.Prompt.String() + "Failed to shutdown provisioner daemon: " + err.Error()
					spin.Stop()
				}
				err = provisionerDaemon.Close()
				if err != nil {
					spin.Stop()
					return xerrors.Errorf("close provisioner daemon: %w", err)
				}
				spin.FinalMSG = cliui.Styles.Prompt.String() + "Gracefully shut down provisioner daemon!\n"
				spin.Stop()
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), cliui.Styles.Prompt.String()+"Waiting for WebSocket connections to close...\n")
			shutdownConns()
			closeCoderd()
			return nil
		},
	}
	defaultAddress := os.Getenv("CODER_ADDRESS")
	if defaultAddress == "" {
		defaultAddress = "127.0.0.1:3000"
	}
	root.Flags().StringVarP(&accessURL, "access-url", "", os.Getenv("CODER_ACCESS_URL"), "Specifies the external URL to access Coder (uses $CODER_ACCESS_URL).")
	root.Flags().StringVarP(&address, "address", "a", defaultAddress, "The address to serve the API and dashboard (uses $CODER_ADDRESS).")
	// systemd uses the CACHE_DIRECTORY environment variable!
	defaultCacheDir := os.Getenv("CACHE_DIRECTORY")
	if defaultCacheDir == "" {
		defaultCacheDir = filepath.Join(os.TempDir(), ".coder-cache")
	}
	root.Flags().StringVarP(&cacheDir, "cache-dir", "", defaultCacheDir, "Specify a directory to cache binaries for provision operations.")
	defaultDev, _ := strconv.ParseBool(os.Getenv("CODER_DEV_MODE"))
	root.Flags().BoolVarP(&dev, "dev", "", defaultDev, "Serve Coder in dev mode for tinkering (uses $CODER_DEV_MODE).")
	root.Flags().StringVarP(&postgresURL, "postgres-url", "", "",
		"URL of a PostgreSQL database to connect to (defaults to $CODER_PG_CONNECTION_URL).")
	root.Flags().Uint8VarP(&provisionerDaemonCount, "provisioner-daemons", "", 1, "The amount of provisioner daemons to create on start.")
	defaultTLSEnable, _ := strconv.ParseBool(os.Getenv("CODER_TLS_ENABLE"))
	root.Flags().BoolVarP(&tlsEnable, "tls-enable", "", defaultTLSEnable, "Specifies if TLS will be enabled (uses $CODER_TLS_ENABLE).")
	root.Flags().StringVarP(&tlsCertFile, "tls-cert-file", "", os.Getenv("CODER_TLS_CERT_FILE"),
		"Specifies the path to the certificate for TLS. It requires a PEM-encoded file. "+
			"To configure the listener to use a CA certificate, concatenate the primary certificate "+
			"and the CA certificate together. The primary certificate should appear first in the combined file (uses $CODER_TLS_CERT_FILE).")
	root.Flags().StringVarP(&tlsClientCAFile, "tls-client-ca-file", "", os.Getenv("CODER_TLS_CLIENT_CA_FILE"),
		"PEM-encoded Certificate Authority file used for checking the authenticity of client (uses $CODER_TLS_CLIENT_CA_FILE).")
	defaultTLSClientAuth := os.Getenv("CODER_TLS_CLIENT_AUTH")
	if defaultTLSClientAuth == "" {
		defaultTLSClientAuth = "request"
	}
	root.Flags().StringVarP(&tlsClientAuth, "tls-client-auth", "", defaultTLSClientAuth,
		`Specifies the policy the server will follow for TLS Client Authentication. `+
			`Accepted values are "none", "request", "require-any", "verify-if-given", or "require-and-verify" (uses $CODER_TLS_CLIENT_AUTH).`)
	root.Flags().StringVarP(&tlsKeyFile, "tls-key-file", "", os.Getenv("CODER_TLS_KEY_FILE"),
		"Specifies the path to the private key for the certificate. It requires a PEM-encoded file (uses $CODER_TLS_KEY_FILE).")
	defaultTLSMinVersion := os.Getenv("CODER_TLS_MIN_VERSION")
	if defaultTLSMinVersion == "" {
		defaultTLSMinVersion = "tls12"
	}
	root.Flags().StringVarP(&tlsMinVersion, "tls-min-version", "", defaultTLSMinVersion,
		`Specifies the minimum supported version of TLS. Accepted values are "tls10", "tls11", "tls12" or "tls13" (uses $CODER_TLS_MIN_VERSION).`)
	defaultTunnelRaw := os.Getenv("CODER_DEV_TUNNEL")
	if defaultTunnelRaw == "" {
		defaultTunnelRaw = "true"
	}
	defaultTunnel, _ := strconv.ParseBool(defaultTunnelRaw)
	root.Flags().BoolVarP(&useTunnel, "tunnel", "", defaultTunnel, "Serve dev mode through a Cloudflare Tunnel for easy setup (uses $CODER_DEV_TUNNEL).")
	_ = root.Flags().MarkHidden("tunnel")

	return root
}

func createFirstUser(cmd *cobra.Command, client *codersdk.Client, cfg config.Root) error {
	_, err := client.CreateFirstUser(cmd.Context(), codersdk.CreateFirstUserRequest{
		Email:        "admin@coder.com",
		Username:     "developer",
		Password:     "password",
		Organization: "acme-corp",
	})
	if err != nil {
		return xerrors.Errorf("create first user: %w", err)
	}
	token, err := client.LoginWithPassword(cmd.Context(), codersdk.LoginWithPasswordRequest{
		Email:    "admin@coder.com",
		Password: "password",
	})
	if err != nil {
		return xerrors.Errorf("login with first user: %w", err)
	}
	client.SessionToken = token.SessionToken

	err = cfg.URL().Write(client.URL.String())
	if err != nil {
		return xerrors.Errorf("write local url: %w", err)
	}
	err = cfg.Session().Write(token.SessionToken)
	if err != nil {
		return xerrors.Errorf("write session token: %w", err)
	}
	return nil
}

func newProvisionerDaemon(ctx context.Context, client *codersdk.Client, logger slog.Logger, cacheDir string) (*provisionerd.Server, error) {
	terraformClient, terraformServer := provisionersdk.TransportPipe()
	go func() {
		err := terraform.Serve(ctx, &terraform.ServeOptions{
			ServeOptions: &provisionersdk.ServeOptions{
				Listener: terraformServer,
			},
			CachePath: cacheDir,
			Logger:    logger,
		})
		if err != nil {
			panic(err)
		}
	}()
	tempDir, err := ioutil.TempDir("", "provisionerd")
	if err != nil {
		return nil, err
	}
	return provisionerd.New(client.ListenProvisionerDaemon, &provisionerd.Options{
		Logger:         logger,
		PollInterval:   50 * time.Millisecond,
		UpdateInterval: 50 * time.Millisecond,
		Provisioners: provisionerd.Provisioners{
			string(database.ProvisionerTypeTerraform): proto.NewDRPCProvisionerClient(provisionersdk.Conn(terraformClient)),
		},
		WorkDirectory: tempDir,
	}), nil
}

func printLogo(cmd *cobra.Command) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), `    ▄█▀    ▀█▄
     ▄▄ ▀▀▀  █▌   ██▀▀█▄          ▐█
 ▄▄██▀▀█▄▄▄  ██  ██      █▀▀█ ▐█▀▀██ ▄█▀▀█ █▀▀
█▌   ▄▌   ▐█ █▌  ▀█▄▄▄█▌ █  █ ▐█  ██ ██▀▀  █
     ██████▀▄█    ▀▀▀▀   ▀▀▀▀  ▀▀▀▀▀  ▀▀▀▀ ▀

`)
}

func configureTLS(listener net.Listener, tlsMinVersion, tlsClientAuth, tlsCertFile, tlsKeyFile, tlsClientCAFile string) (net.Listener, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	switch tlsMinVersion {
	case "tls10":
		tlsConfig.MinVersion = tls.VersionTLS10
	case "tls11":
		tlsConfig.MinVersion = tls.VersionTLS11
	case "tls12":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "tls13":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		return nil, xerrors.Errorf("unrecognized tls version: %q", tlsMinVersion)
	}

	switch tlsClientAuth {
	case "none":
		tlsConfig.ClientAuth = tls.NoClientCert
	case "request":
		tlsConfig.ClientAuth = tls.RequestClientCert
	case "require-any":
		tlsConfig.ClientAuth = tls.RequireAnyClientCert
	case "verify-if-given":
		tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	case "require-and-verify":
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	default:
		return nil, xerrors.Errorf("unrecognized tls client auth: %q", tlsClientAuth)
	}

	if tlsCertFile == "" {
		return nil, xerrors.New("tls-cert-file is required when tls is enabled")
	}
	if tlsKeyFile == "" {
		return nil, xerrors.New("tls-key-file is required when tls is enabled")
	}

	certPEMBlock, err := os.ReadFile(tlsCertFile)
	if err != nil {
		return nil, xerrors.Errorf("read file %q: %w", tlsCertFile, err)
	}
	keyPEMBlock, err := os.ReadFile(tlsKeyFile)
	if err != nil {
		return nil, xerrors.Errorf("read file %q: %w", tlsKeyFile, err)
	}
	keyBlock, _ := pem.Decode(keyPEMBlock)
	if keyBlock == nil {
		return nil, xerrors.New("decoded pem is blank")
	}
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, xerrors.Errorf("create key pair: %w", err)
	}
	tlsConfig.GetCertificate = func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return &cert, nil
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certPEMBlock)
	tlsConfig.RootCAs = certPool

	if tlsClientCAFile != "" {
		caPool := x509.NewCertPool()
		data, err := ioutil.ReadFile(tlsClientCAFile)
		if err != nil {
			return nil, xerrors.Errorf("read %q: %w", tlsClientCAFile, err)
		}
		if !caPool.AppendCertsFromPEM(data) {
			return nil, xerrors.Errorf("failed to parse CA certificate in tls-client-ca-file")
		}
		tlsConfig.ClientCAs = caPool
	}

	return tls.NewListener(listener, tlsConfig), nil
}
