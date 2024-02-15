package wsproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/xerrors"
	"tailscale.com/derp"
	"tailscale.com/derp/derphttp"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/buildinfo"
	"github.com/coder/coder/v2/cli/cliutil"
	"github.com/coder/coder/v2/coderd"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/coderd/tracing"
	"github.com/coder/coder/v2/coderd/workspaceapps"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/derpmesh"
	"github.com/coder/coder/v2/enterprise/wsproxy/wsproxysdk"
	"github.com/coder/coder/v2/site"
	"github.com/coder/coder/v2/tailnet"
)

type Options struct {
	Logger      slog.Logger
	Experiments codersdk.Experiments

	HTTPClient *http.Client
	// DashboardURL is the URL of the primary coderd instance.
	DashboardURL *url.URL
	// AccessURL is the URL of the WorkspaceProxy.
	AccessURL *url.URL

	// TODO: @emyrk We use these two fields in many places with this comment.
	//		Maybe we should make some shared options struct?
	// AppHostname should be the wildcard hostname to use for workspace
	// applications INCLUDING the asterisk, (optional) suffix and leading dot.
	// It will use the same scheme and port number as the access URL.
	// E.g. "*.apps.coder.com" or "*-apps.coder.com".
	AppHostname string
	// AppHostnameRegex contains the regex version of options.AppHostname as
	// generated by appurl.CompileHostnamePattern(). It MUST be set if
	// options.AppHostname is set.
	AppHostnameRegex *regexp.Regexp

	RealIPConfig       *httpmw.RealIPConfig
	Tracing            trace.TracerProvider
	PrometheusRegistry *prometheus.Registry
	TLSCertificates    []tls.Certificate

	APIRateLimit           int
	SecureAuthCookie       bool
	DisablePathApps        bool
	DERPEnabled            bool
	DERPServerRelayAddress string
	// DERPOnly determines whether this proxy only provides DERP and does not
	// provide access to workspace apps/terminal.
	DERPOnly bool
	// BlockDirect controls the servertailnet of the proxy, forcing it from
	// negotiating direct connections.
	BlockDirect bool

	ProxySessionToken string
	// AllowAllCors will set all CORs headers to '*'.
	// By default, CORs is set to accept external requests
	// from the dashboardURL. This should only be used in development.
	AllowAllCors bool

	StatsCollectorOptions workspaceapps.StatsCollectorOptions
}

func (o *Options) Validate() error {
	var errs optErrors

	errs.Required("Logger", o.Logger)
	errs.Required("DashboardURL", o.DashboardURL)
	errs.Required("AccessURL", o.AccessURL)
	errs.Required("RealIPConfig", o.RealIPConfig)
	errs.Required("PrometheusRegistry", o.PrometheusRegistry)
	errs.NotEmpty("ProxySessionToken", o.ProxySessionToken)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Server is an external workspace proxy server. This server can communicate
// directly with a workspace. It requires a primary coderd to establish a said
// connection.
type Server struct {
	Options *Options
	Handler chi.Router

	DashboardURL *url.URL
	AppServer    *workspaceapps.Server

	// Logging/Metrics
	Logger             slog.Logger
	TracerProvider     trace.TracerProvider
	PrometheusRegistry *prometheus.Registry

	// SDKClient is a client to the primary coderd instance authenticated with
	// the moon's token.
	SDKClient *wsproxysdk.Client

	// DERP
	derpMesh      *derpmesh.Mesh
	latestDERPMap atomic.Pointer[tailcfg.DERPMap]

	// Used for graceful shutdown. Required for the dialer.
	ctx           context.Context
	cancel        context.CancelFunc
	derpCloseFunc func()
	registerLoop  *wsproxysdk.RegisterWorkspaceProxyLoop
}

// New creates a new workspace proxy server. This requires a primary coderd
// instance to be reachable and the correct authorization access token to be
// provided. If the proxy cannot authenticate with the primary, this will fail.
func New(ctx context.Context, opts *Options) (*Server, error) {
	if opts.PrometheusRegistry == nil {
		opts.PrometheusRegistry = prometheus.NewRegistry()
	}

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	client := wsproxysdk.New(opts.DashboardURL)
	err := client.SetSessionToken(opts.ProxySessionToken)
	if err != nil {
		return nil, xerrors.Errorf("set client token: %w", err)
	}

	// Use the configured client if provided.
	if opts.HTTPClient != nil {
		client.SDKClient.HTTPClient = opts.HTTPClient
	}

	// TODO: Probably do some version checking here
	info, err := client.SDKClient.BuildInfo(ctx)
	if err != nil {
		return nil, xerrors.Errorf("buildinfo: %w", errors.Join(
			xerrors.Errorf("unable to fetch build info from primary coderd. Are you sure %q is a coderd instance?", opts.DashboardURL),
			err,
		))
	}
	if info.WorkspaceProxy {
		return nil, xerrors.Errorf("%q is a workspace proxy, not a primary coderd instance", opts.DashboardURL)
	}

	meshRootCA := x509.NewCertPool()
	for _, certificate := range opts.TLSCertificates {
		for _, certificatePart := range certificate.Certificate {
			certificate, err := x509.ParseCertificate(certificatePart)
			if err != nil {
				return nil, xerrors.Errorf("parse certificate %s: %w", certificate.Subject.CommonName, err)
			}
			meshRootCA.AddCert(certificate)
		}
	}
	// This TLS configuration spoofs access from the access URL hostname
	// assuming that the certificates provided will cover that hostname.
	//
	// Replica sync and DERP meshing require accessing replicas via their
	// internal IP addresses, and if TLS is configured we use the same
	// certificates.
	meshTLSConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: opts.TLSCertificates,
		RootCAs:      meshRootCA,
		ServerName:   opts.AccessURL.Hostname(),
	}

	derpServer := derp.NewServer(key.NewNode(), tailnet.Logger(opts.Logger.Named("net.derp")))

	ctx, cancel := context.WithCancel(context.Background())
	r := chi.NewRouter()
	s := &Server{
		Options:            opts,
		Handler:            r,
		DashboardURL:       opts.DashboardURL,
		Logger:             opts.Logger.Named("net.workspace-proxy"),
		TracerProvider:     opts.Tracing,
		PrometheusRegistry: opts.PrometheusRegistry,
		SDKClient:          client,
		derpMesh:           derpmesh.New(opts.Logger.Named("net.derpmesh"), derpServer, meshTLSConfig),
		ctx:                ctx,
		cancel:             cancel,
	}

	// Register the workspace proxy with the primary coderd instance and start a
	// goroutine to periodically re-register.
	replicaID := uuid.New()
	osHostname := cliutil.Hostname()
	registerLoop, regResp, err := client.RegisterWorkspaceProxyLoop(ctx, wsproxysdk.RegisterWorkspaceProxyLoopOpts{
		Logger: opts.Logger,
		Request: wsproxysdk.RegisterWorkspaceProxyRequest{
			AccessURL:           opts.AccessURL.String(),
			WildcardHostname:    opts.AppHostname,
			DerpEnabled:         opts.DERPEnabled,
			DerpOnly:            opts.DERPOnly,
			ReplicaID:           replicaID,
			ReplicaHostname:     osHostname,
			ReplicaError:        "",
			ReplicaRelayAddress: opts.DERPServerRelayAddress,
			Version:             buildinfo.Version(),
		},
		MutateFn:   s.mutateRegister,
		CallbackFn: s.handleRegister,
		FailureFn:  s.handleRegisterFailure,
	})
	if err != nil {
		return nil, xerrors.Errorf("register proxy: %w", err)
	}
	s.registerLoop = registerLoop

	derpServer.SetMeshKey(regResp.DERPMeshKey)
	err = s.handleRegister(regResp)
	if err != nil {
		return nil, xerrors.Errorf("handle register: %w", err)
	}

	secKey, err := workspaceapps.KeyFromString(regResp.AppSecurityKey)
	if err != nil {
		return nil, xerrors.Errorf("parse app security key: %w", err)
	}

	agentProvider, err := coderd.NewServerTailnet(ctx,
		s.Logger,
		nil,
		func() *tailcfg.DERPMap {
			return s.latestDERPMap.Load()
		},
		regResp.DERPForceWebSockets,
		s.DialCoordinator,
		opts.BlockDirect,
		s.TracerProvider,
	)
	if err != nil {
		return nil, xerrors.Errorf("create server tailnet: %w", err)
	}

	workspaceAppsLogger := opts.Logger.Named("workspaceapps")
	if opts.StatsCollectorOptions.Logger == nil {
		named := workspaceAppsLogger.Named("stats_collector")
		opts.StatsCollectorOptions.Logger = &named
	}
	if opts.StatsCollectorOptions.Reporter == nil {
		opts.StatsCollectorOptions.Reporter = &appStatsReporter{Client: client}
	}

	s.AppServer = &workspaceapps.Server{
		Logger:        workspaceAppsLogger,
		DashboardURL:  opts.DashboardURL,
		AccessURL:     opts.AccessURL,
		Hostname:      opts.AppHostname,
		HostnameRegex: opts.AppHostnameRegex,
		RealIPConfig:  opts.RealIPConfig,
		SignedTokenProvider: &TokenProvider{
			DashboardURL: opts.DashboardURL,
			AccessURL:    opts.AccessURL,
			AppHostname:  opts.AppHostname,
			Client:       client,
			SecurityKey:  secKey,
			Logger:       s.Logger.Named("proxy_token_provider"),
		},
		AppSecurityKey: secKey,

		DisablePathApps:  opts.DisablePathApps,
		SecureAuthCookie: opts.SecureAuthCookie,

		AgentProvider:  agentProvider,
		StatsCollector: workspaceapps.NewStatsCollector(opts.StatsCollectorOptions),
	}

	derpHandler := derphttp.Handler(derpServer)
	derpHandler, s.derpCloseFunc = tailnet.WithWebsocketSupport(derpServer, derpHandler)

	// The primary coderd dashboard needs to make some GET requests to
	// the workspace proxies to check latency.
	corsMW := httpmw.Cors(opts.AllowAllCors, opts.DashboardURL.String())
	prometheusMW := httpmw.Prometheus(s.PrometheusRegistry)

	// Routes
	apiRateLimiter := httpmw.RateLimit(opts.APIRateLimit, time.Minute)
	// Persistent middlewares to all routes
	r.Use(
		// TODO: @emyrk Should we standardize these in some other package?
		httpmw.Recover(s.Logger),
		tracing.StatusWriterMiddleware,
		tracing.Middleware(s.TracerProvider),
		httpmw.AttachRequestID,
		httpmw.ExtractRealIP(s.Options.RealIPConfig),
		httpmw.Logger(s.Logger),
		prometheusMW,
		corsMW,

		// HandleSubdomain is a middleware that handles all requests to the
		// subdomain-based workspace apps.
		s.AppServer.HandleSubdomain(apiRateLimiter),
		// Build-Version is helpful for debugging.
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add(codersdk.BuildVersionHeader, buildinfo.Version())
				next.ServeHTTP(w, r)
			})
		},
		// This header stops a browser from trying to MIME-sniff the content type and
		// forces it to stick with the declared content-type. This is the only valid
		// value for this header.
		// See: https://github.com/coder/security/issues/12
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Content-Type-Options", "nosniff")
				next.ServeHTTP(w, r)
			})
		},
		// CSRF is required here because we need to set the CSRF cookies on
		// responses.
		httpmw.CSRF(s.Options.SecureAuthCookie),
	)

	// Attach workspace apps routes.
	if !opts.DERPOnly {
		r.Group(func(r chi.Router) {
			r.Use(apiRateLimiter)
			s.AppServer.Attach(r)
		})
	} else {
		r.Group(func(r chi.Router) {
			derpOnlyHandler := func(rw http.ResponseWriter, r *http.Request) {
				site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
					Title:      "Head to the Dashboard",
					Status:     http.StatusBadRequest,
					HideStatus: true,
					Description: "This workspace proxy is DERP-only and cannot be used for browser connections. " +
						"Please use a different region directly from the dashboard. Click to be redirected!",
					RetryEnabled: false,
					DashboardURL: opts.DashboardURL.String(),
				})
			}
			serveDerpOnlyHandler := func(r chi.Router) {
				r.HandleFunc("/*", derpOnlyHandler)
			}

			r.Route("/%40{user}/{workspace_and_agent}/apps/{workspaceapp}", serveDerpOnlyHandler)
			r.Route("/@{user}/{workspace_and_agent}/apps/{workspaceapp}", serveDerpOnlyHandler)
			r.Get("/api/v2/workspaceagents/{workspaceagent}/pty", derpOnlyHandler)
		})
	}

	if opts.DERPEnabled {
		r.Route("/derp", func(r chi.Router) {
			r.Get("/", derpHandler.ServeHTTP)
			// This is used when UDP is blocked, and latency must be checked via HTTP(s).
			r.Get("/latency-check", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		})
	} else {
		r.Route("/derp", func(r chi.Router) {
			r.HandleFunc("/*", func(rw http.ResponseWriter, r *http.Request) {
				httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
					Message: "DERP is disabled on this proxy.",
				})
			})
		})
	}

	r.Get("/api/v2/buildinfo", s.buildInfo)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("OK")) })
	// TODO: @emyrk should this be authenticated or debounced?
	r.Get("/healthz-report", s.healthReport)
	r.NotFound(func(rw http.ResponseWriter, r *http.Request) {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Title:      "Head to the Dashboard",
			Status:     http.StatusBadRequest,
			HideStatus: true,
			Description: "Workspace Proxies route traffic in terminals and apps directly to your workspace. " +
				"This page must be loaded from the dashboard. Click to be redirected!",
			RetryEnabled: false,
			DashboardURL: opts.DashboardURL.String(),
		})
	})

	// See coderd/coderd.go for why we need this.
	rootRouter := chi.NewRouter()
	// Make sure to add the cors middleware to the latency check route.
	rootRouter.Get("/latency-check", tracing.StatusWriterMiddleware(prometheusMW(coderd.LatencyCheck())).ServeHTTP)
	rootRouter.Mount("/", r)
	s.Handler = rootRouter

	return s, nil
}

func (s *Server) RegisterNow() error {
	_, err := s.registerLoop.RegisterNow()
	return err
}

func (s *Server) Close() error {
	s.cancel()

	var err error
	s.registerLoop.Close()
	s.derpCloseFunc()
	appServerErr := s.AppServer.Close()
	if appServerErr != nil {
		err = multierror.Append(err, appServerErr)
	}
	agentProviderErr := s.AppServer.AgentProvider.Close()
	if agentProviderErr != nil {
		err = multierror.Append(err, agentProviderErr)
	}
	s.SDKClient.SDKClient.HTTPClient.CloseIdleConnections()
	return err
}

func (*Server) mutateRegister(_ *wsproxysdk.RegisterWorkspaceProxyRequest) {
	// TODO: we should probably ping replicas similarly to the replicasync
	// package in the primary and update req.ReplicaError accordingly.
}

func (s *Server) handleRegister(res wsproxysdk.RegisterWorkspaceProxyResponse) error {
	addresses := make([]string, len(res.SiblingReplicas))
	for i, replica := range res.SiblingReplicas {
		addresses[i] = replica.RelayAddress
	}
	s.Logger.Debug(s.ctx, "setting DERP mesh sibling addresses", slog.F("addresses", addresses))
	s.derpMesh.SetAddresses(addresses, false)

	s.latestDERPMap.Store(res.DERPMap)

	return nil
}

func (s *Server) handleRegisterFailure(err error) {
	if s.ctx.Err() != nil {
		return
	}
	s.Logger.Fatal(s.ctx,
		"failed to periodically re-register workspace proxy with primary Coder deployment",
		slog.Error(err),
	)
}

func (s *Server) DialCoordinator(ctx context.Context) (tailnet.MultiAgentConn, error) {
	return s.SDKClient.DialCoordinator(ctx)
}

func (s *Server) buildInfo(rw http.ResponseWriter, r *http.Request) {
	httpapi.Write(r.Context(), rw, http.StatusOK, codersdk.BuildInfoResponse{
		ExternalURL:     buildinfo.ExternalURL(),
		Version:         buildinfo.Version(),
		AgentAPIVersion: coderd.AgentAPIVersionREST,
		DashboardURL:    s.DashboardURL.String(),
		WorkspaceProxy:  true,
	})
}

// healthReport is a more thorough health check than the '/healthz' endpoint.
// This endpoint not only responds if the server is running, but can do some
// internal diagnostics to ensure that the server is running correctly. The
// primary coderd will use this to determine if this workspace proxy can be used
// by the users. This endpoint will take longer to respond than the '/healthz'.
// Checks:
// - Can communicate with primary coderd
//
// TODO: Config checks to ensure consistent with primary
func (s *Server) healthReport(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var report codersdk.ProxyHealthReport

	// This is to catch edge cases where the server is shutting down, but might
	// still serve a web request that returns "healthy". This is mainly just for
	// unit tests, as shutting down the test webserver is tied to the lifecycle
	// of the test. In practice, the webserver is tied to the lifecycle of the
	// app, so the webserver AND the proxy will be shut down at the same time.
	if s.ctx.Err() != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, "workspace proxy in middle of shutting down")
		return
	}

	// Hit the build info to do basic version checking.
	primaryBuild, err := s.SDKClient.SDKClient.BuildInfo(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to get build info: %s", err.Error()))
		httpapi.Write(r.Context(), rw, http.StatusOK, report)
		return
	}

	if primaryBuild.WorkspaceProxy {
		// This could be a simple mistake of using a proxy url as the dashboard url.
		report.Errors = append(report.Errors,
			fmt.Sprintf("dashboard url (%s) is a workspace proxy, must be a primary coderd", s.DashboardURL.String()))
	}

	// If we are in dev mode, never check versions.
	if !buildinfo.IsDev() && !buildinfo.VersionsMatch(primaryBuild.Version, buildinfo.Version()) {
		// Version mismatches are not fatal, but should be reported.
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("version mismatch: primary coderd (%s) != workspace proxy (%s)", primaryBuild.Version, buildinfo.Version()))
	}

	// TODO: We should hit the deployment config endpoint and do some config
	// checks. We can check the version from the X-CODER-BUILD-VERSION header

	httpapi.Write(r.Context(), rw, http.StatusOK, report)
}

type optErrors []error

func (e optErrors) Error() string {
	var b strings.Builder
	for _, err := range e {
		_, _ = b.WriteString(err.Error())
		_, _ = b.WriteString("\n")
	}
	return b.String()
}

func (e *optErrors) Required(name string, v any) {
	if v == nil {
		*e = append(*e, xerrors.Errorf("%s is required, got <nil>", name))
	}
}

func (e *optErrors) NotEmpty(name string, v any) {
	if reflect.ValueOf(v).IsZero() {
		*e = append(*e, xerrors.Errorf("%s is required, got the zero value", name))
	}
}
