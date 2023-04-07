package coderdenttest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sync"
	"testing"

	"github.com/coder/coder/codersdk"
	"github.com/moby/moby/pkg/namesgenerator"

	"github.com/coder/coder/enterprise/coderd"

	"github.com/coder/coder/enterprise/wsproxy"

	"github.com/coder/coder/coderd/httpapi"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/coderdtest"

	"github.com/coder/coder/coderd/rbac"
	"github.com/prometheus/client_golang/prometheus"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbauthz"
	"github.com/coder/coder/coderd/database/dbtestutil"
)

type ProxyOptions struct {
	Name string

	Database        database.Store
	Pubsub          database.Pubsub
	Authorizer      rbac.Authorizer
	TLSCertificates []tls.Certificate
	ProxyURL        *url.URL
	AppHostname     string
}

// NewWorkspaceProxy will configure a wsproxy.Server with the given options.
// The new wsproxy will register itself with the given coderd.API instance.
// The first user owner client is required to create the wsproxy on the coderd
// api server.
func NewWorkspaceProxy(t *testing.T, coderd *coderd.API, owner *codersdk.Client, options *ProxyOptions) *wsproxy.Server {
	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)

	if options == nil {
		options = &ProxyOptions{}
	}

	if options.Authorizer == nil {
		options.Authorizer = &coderdtest.RecordingAuthorizer{
			Wrapped: rbac.NewCachingAuthorizer(prometheus.NewRegistry()),
		}
	}

	if options.Database == nil {
		options.Database, options.Pubsub = dbtestutil.NewDB(t)
		options.Database = dbauthz.New(options.Database, options.Authorizer, slogtest.Make(t, nil).Leveled(slog.LevelDebug))
	}

	// HTTP Server
	var mutex sync.RWMutex
	var handler http.Handler
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.RLock()
		defer mutex.RUnlock()
		if handler != nil {
			handler.ServeHTTP(w, r)
		}
	}))
	srv.Config.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	if options.TLSCertificates != nil {
		srv.TLS = &tls.Config{
			Certificates: options.TLSCertificates,
			MinVersion:   tls.VersionTLS12,
		}
		srv.StartTLS()
	} else {
		srv.Start()
	}
	t.Cleanup(srv.Close)

	tcpAddr, ok := srv.Listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	serverURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	serverURL.Host = fmt.Sprintf("localhost:%d", tcpAddr.Port)

	accessURL := options.ProxyURL
	if accessURL == nil {
		accessURL = serverURL
	}

	// TODO: Stun and derp stuff
	//derpPort, err := strconv.Atoi(serverURL.Port())
	//require.NoError(t, err)
	//
	//stunAddr, stunCleanup := stuntest.ServeWithPacketListener(t, nettype.Std{})
	//t.Cleanup(stunCleanup)
	//
	//derpServer := derp.NewServer(key.NewNode(), tailnet.Logger(slogtest.Make(t, nil).Named("derp").Leveled(slog.LevelDebug)))
	//derpServer.SetMeshKey("test-key")

	var appHostnameRegex *regexp.Regexp
	if options.AppHostname != "" {
		var err error
		appHostnameRegex, err = httpapi.CompileHostnamePattern(options.AppHostname)
		require.NoError(t, err)
	}

	if options.Name == "" {
		options.Name = namesgenerator.GetRandomName(1)
	}

	proxyRes, err := owner.CreateWorkspaceProxy(ctx, codersdk.CreateWorkspaceProxyRequest{
		Name:             options.Name,
		Icon:             "/emojis/flag.png",
		URL:              accessURL.String(),
		WildcardHostname: options.AppHostname,
	})

	wssrv, err := wsproxy.New(&wsproxy.Options{
		Logger:            slogtest.Make(t, nil).Leveled(slog.LevelDebug),
		PrimaryAccessURL:  coderd.AccessURL,
		AccessURL:         options.ProxyURL,
		AppHostname:       options.AppHostname,
		AppHostnameRegex:  appHostnameRegex,
		RealIPConfig:      coderd.RealIPConfig,
		AppSecurityKey:    coderd.AppSecurityKey,
		Tracing:           coderd.TracerProvider,
		APIRateLimit:      coderd.APIRateLimit,
		SecureAuthCookie:  coderd.SecureAuthCookie,
		ProxySessionToken: proxyRes.ProxyToken,
		// We need a new registry to not conflict with the coderd internal
		// proxy metrics.
		PrometheusRegistry: prometheus.NewRegistry(),
	})
	require.NoError(t, err)
	return wssrv
}
