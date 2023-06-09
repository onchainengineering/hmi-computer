package wsproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/buildinfo"
	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/tracing"
	"github.com/coder/coder/coderd/workspaceapps"
	"github.com/coder/coder/coderd/wsconncache"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/enterprise/wsproxy/wsproxysdk"
	"github.com/coder/coder/site"
)

type Options struct {
	Logger slog.Logger

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
	// generated by httpapi.CompileHostnamePattern(). It MUST be set if
	// options.AppHostname is set.
	AppHostnameRegex *regexp.Regexp

	RealIPConfig *httpmw.RealIPConfig

	Tracing            trace.TracerProvider
	PrometheusRegistry *prometheus.Registry

	APIRateLimit     int
	SecureAuthCookie bool
	DisablePathApps  bool

	ProxySessionToken string
	// AllowAllCors will set all CORs headers to '*'.
	// By default, CORs is set to accept external requests
	// from the dashboardURL. This should only be used in development.
	AllowAllCors bool
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

	// TODO: Missing:
	//		- derpserver

	// Used for graceful shutdown. Required for the dialer.
	ctx    context.Context
	cancel context.CancelFunc
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
		return nil, xerrors.Errorf("failed to fetch build info from %q: %w", opts.DashboardURL, err)
	}
	if info.WorkspaceProxy {
		return nil, xerrors.Errorf("%q is a workspace proxy, not a primary coderd instance", opts.DashboardURL)
	}

	regResp, err := client.RegisterWorkspaceProxy(ctx, wsproxysdk.RegisterWorkspaceProxyRequest{
		AccessURL:        opts.AccessURL.String(),
		WildcardHostname: opts.AppHostname,
	})
	if err != nil {
		return nil, xerrors.Errorf("register proxy: %w", err)
	}

	secKey, err := workspaceapps.KeyFromString(regResp.AppSecurityKey)
	if err != nil {
		return nil, xerrors.Errorf("parse app security key: %w", err)
	}

	r := chi.NewRouter()
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		Options:            opts,
		Handler:            r,
		DashboardURL:       opts.DashboardURL,
		Logger:             opts.Logger.Named("workspace-proxy"),
		TracerProvider:     opts.Tracing,
		PrometheusRegistry: opts.PrometheusRegistry,
		SDKClient:          client,
		ctx:                ctx,
		cancel:             cancel,
	}

	s.AppServer = &workspaceapps.Server{
		Logger:        opts.Logger.Named("workspaceapps"),
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
		WorkspaceConnCache: wsconncache.New(s.DialWorkspaceAgent, 0),
		AppSecurityKey:     secKey,

		DisablePathApps:  opts.DisablePathApps,
		SecureAuthCookie: opts.SecureAuthCookie,
	}

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
				w.Header().Add("X-Coder-Build-Version", buildinfo.Version())
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
		// TODO: @emyrk we might not need this? But good to have if it does
		// 		not break anything.
		httpmw.CSRF(s.Options.SecureAuthCookie),
	)

	// Attach workspace apps routes.
	r.Group(func(r chi.Router) {
		r.Use(apiRateLimiter)
		s.AppServer.Attach(r)
	})

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

func (s *Server) Close() error {
	s.cancel()

	// A timeout to prevent the SDK from blocking the server shutdown.
	tmp, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.SDKClient.WorkspaceProxyGoingAway(tmp)

	return s.AppServer.Close()
}

func (s *Server) DialWorkspaceAgent(id uuid.UUID) (*codersdk.WorkspaceAgentConn, error) {
	return s.SDKClient.DialWorkspaceAgent(s.ctx, id, nil)
}

func (s *Server) buildInfo(rw http.ResponseWriter, r *http.Request) {
	httpapi.Write(r.Context(), rw, http.StatusOK, codersdk.BuildInfoResponse{
		ExternalURL:    buildinfo.ExternalURL(),
		Version:        buildinfo.Version(),
		DashboardURL:   s.DashboardURL.String(),
		WorkspaceProxy: true,
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
