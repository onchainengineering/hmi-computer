package coderd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"tailscale.com/derp"
	"tailscale.com/derp/derphttp"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"

	"cdr.dev/slog"
	"github.com/coder/coder/buildinfo"
	"github.com/coder/coder/coderd/audit"
	"github.com/coder/coder/coderd/awsidentity"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbtype"
	"github.com/coder/coder/coderd/gitauth"
	"github.com/coder/coder/coderd/gitsshkey"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/metricscache"
	"github.com/coder/coder/coderd/provisionerdserver"
	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/coderd/telemetry"
	"github.com/coder/coder/coderd/tracing"
	"github.com/coder/coder/coderd/wsconncache"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/provisionerd/proto"
	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/site"
	"github.com/coder/coder/tailnet"
)

// Options are requires parameters for Coder to start.
type Options struct {
	AccessURL *url.URL
	// AppHostname should be the wildcard hostname to use for workspace
	// applications INCLUDING the asterisk, (optional) suffix and leading dot.
	// E.g. "*.apps.coder.com" or "*-apps.coder.com".
	AppHostname string
	// AppHostnameRegex contains the regex version of options.AppHostname as
	// generated by httpapi.CompileHostnamePattern(). It MUST be set if
	// options.AppHostname is set.
	AppHostnameRegex *regexp.Regexp
	Logger           slog.Logger
	Database         database.Store
	Pubsub           database.Pubsub

	// CacheDir is used for caching files served by the API.
	CacheDir string

	Auditor                        audit.Auditor
	AgentConnectionUpdateFrequency time.Duration
	AgentInactiveDisconnectTimeout time.Duration
	// APIRateLimit is the minutely throughput rate limit per user or ip.
	// Setting a rate limit <0 will disable the rate limiter across the entire
	// app. Specific routes may have their own limiters.
	APIRateLimit         int
	AWSCertificates      awsidentity.Certificates
	Authorizer           rbac.Authorizer
	AzureCertificates    x509.VerifyOptions
	GoogleTokenValidator *idtoken.Validator
	GithubOAuth2Config   *GithubOAuth2Config
	OIDCConfig           *OIDCConfig
	PrometheusRegistry   *prometheus.Registry
	SecureAuthCookie     bool
	SSHKeygenAlgorithm   gitsshkey.Algorithm
	Telemetry            telemetry.Reporter
	TracerProvider       trace.TracerProvider
	AutoImportTemplates  []AutoImportTemplate
	GitAuthConfigs       []*gitauth.Config
	RealIPConfig         *httpmw.RealIPConfig

	// TLSCertificates is used to mesh DERP servers securely.
	TLSCertificates    []tls.Certificate
	TailnetCoordinator tailnet.Coordinator
	DERPServer         *derp.Server
	DERPMap            *tailcfg.DERPMap

	MetricsCacheRefreshInterval time.Duration
	AgentStatsRefreshInterval   time.Duration
	Experimental                bool
	DeploymentConfig            *codersdk.DeploymentConfig
}

// New constructs a Coder API handler.
func New(options *Options) *API {
	if options == nil {
		options = &Options{}
	}
	if options.AppHostname != "" && options.AppHostnameRegex == nil || options.AppHostname == "" && options.AppHostnameRegex != nil {
		panic("coderd: both AppHostname and AppHostnameRegex must be set or unset")
	}
	if options.AgentConnectionUpdateFrequency == 0 {
		options.AgentConnectionUpdateFrequency = 3 * time.Second
	}
	if options.AgentInactiveDisconnectTimeout == 0 {
		// Multiply the update by two to allow for some lag-time.
		options.AgentInactiveDisconnectTimeout = options.AgentConnectionUpdateFrequency * 2
	}
	if options.AgentStatsRefreshInterval == 0 {
		options.AgentStatsRefreshInterval = 10 * time.Minute
	}
	if options.MetricsCacheRefreshInterval == 0 {
		options.MetricsCacheRefreshInterval = time.Hour
	}
	if options.APIRateLimit == 0 {
		options.APIRateLimit = 512
	}
	if options.AgentStatsRefreshInterval == 0 {
		options.AgentStatsRefreshInterval = 10 * time.Minute
	}
	if options.MetricsCacheRefreshInterval == 0 {
		options.MetricsCacheRefreshInterval = time.Hour
	}
	if options.Authorizer == nil {
		options.Authorizer = rbac.NewAuthorizer()
	}
	if options.PrometheusRegistry == nil {
		options.PrometheusRegistry = prometheus.NewRegistry()
	}
	if options.TailnetCoordinator == nil {
		options.TailnetCoordinator = tailnet.NewCoordinator()
	}
	if options.DERPServer == nil {
		options.DERPServer = derp.NewServer(key.NewNode(), tailnet.Logger(options.Logger.Named("derp")))
	}
	if options.Auditor == nil {
		options.Auditor = audit.NewNop()
	}

	siteCacheDir := options.CacheDir
	if siteCacheDir != "" {
		siteCacheDir = filepath.Join(siteCacheDir, "site")
	}
	binFS, err := site.ExtractOrReadBinFS(siteCacheDir, site.FS())
	if err != nil {
		panic(xerrors.Errorf("read site bin failed: %w", err))
	}

	metricsCache := metricscache.New(
		options.Database,
		options.Logger.Named("metrics_cache"),
		options.MetricsCacheRefreshInterval,
	)

	r := chi.NewRouter()
	api := &API{
		ID:          uuid.New(),
		Options:     options,
		RootHandler: r,
		siteHandler: site.Handler(site.FS(), binFS),
		HTTPAuth: &HTTPAuthorizer{
			Authorizer: options.Authorizer,
			Logger:     options.Logger,
		},
		metricsCache: metricsCache,
		Auditor:      atomic.Pointer[audit.Auditor]{},
	}
	api.Auditor.Store(&options.Auditor)
	api.workspaceAgentCache = wsconncache.New(api.dialWorkspaceAgentTailnet, 0)
	api.TailnetCoordinator.Store(&options.TailnetCoordinator)
	oauthConfigs := &httpmw.OAuth2Configs{
		Github: options.GithubOAuth2Config,
		OIDC:   options.OIDCConfig,
	}

	apiKeyMiddleware := httpmw.ExtractAPIKey(httpmw.ExtractAPIKeyConfig{
		DB:              options.Database,
		OAuth2Configs:   oauthConfigs,
		RedirectToLogin: false,
		Optional:        false,
	})
	// Same as above but it redirects to the login page.
	apiKeyMiddlewareRedirect := httpmw.ExtractAPIKey(httpmw.ExtractAPIKeyConfig{
		DB:              options.Database,
		OAuth2Configs:   oauthConfigs,
		RedirectToLogin: true,
		Optional:        false,
	})

	r.Use(
		httpmw.AttachRequestID,
		httpmw.Recover(api.Logger),
		httpmw.ExtractRealIP(api.RealIPConfig),
		httpmw.Logger(api.Logger),
		httpmw.Prometheus(options.PrometheusRegistry),
		// handleSubdomainApplications checks if the first subdomain is a valid
		// app URL. If it is, it will serve that application.
		api.handleSubdomainApplications(
			// Middleware to impose on the served application.
			httpmw.RateLimit(options.APIRateLimit, time.Minute),
			httpmw.ExtractAPIKey(httpmw.ExtractAPIKeyConfig{
				DB:            options.Database,
				OAuth2Configs: oauthConfigs,
				// The code handles the the case where the user is not
				// authenticated automatically.
				RedirectToLogin: false,
				Optional:        true,
			}),
			httpmw.ExtractUserParam(api.Database, false),
			httpmw.ExtractWorkspaceAndAgentParam(api.Database),
		),
		// Build-Version is helpful for debugging.
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Coder-Build-Version", buildinfo.Version())
				next.ServeHTTP(w, r)
			})
		},
		httpmw.CSRF(options.SecureAuthCookie),
	)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("OK")) })

	apps := func(r chi.Router) {
		r.Use(
			tracing.Middleware(api.TracerProvider),
			httpmw.RateLimit(options.APIRateLimit, time.Minute),
			httpmw.ExtractAPIKey(httpmw.ExtractAPIKeyConfig{
				DB:            options.Database,
				OAuth2Configs: oauthConfigs,
				// Optional is true to allow for public apps. If an
				// authorization check fails and the user is not authenticated,
				// they will be redirected to the login page by the app handler.
				RedirectToLogin: false,
				Optional:        true,
			}),
			// Redirect to the login page if the user tries to open an app with
			// "me" as the username and they are not logged in.
			httpmw.ExtractUserParam(api.Database, true),
			// Extracts the <workspace.agent> from the url
			httpmw.ExtractWorkspaceAndAgentParam(api.Database),
		)
		r.HandleFunc("/*", api.workspaceAppsProxyPath)
	}
	// %40 is the encoded character of the @ symbol. VS Code Web does
	// not handle character encoding properly, so it's safe to assume
	// other applications might not as well.
	r.Route("/%40{user}/{workspace_and_agent}/apps/{workspaceapp}", apps)
	r.Route("/@{user}/{workspace_and_agent}/apps/{workspaceapp}", apps)
	r.Route("/derp", func(r chi.Router) {
		r.Get("/", derphttp.Handler(api.DERPServer).ServeHTTP)
		// This is used when UDP is blocked, and latency must be checked via HTTP(s).
		r.Get("/latency-check", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	r.Route("/gitauth", func(r chi.Router) {
		for _, gitAuthConfig := range options.GitAuthConfigs {
			r.Route(fmt.Sprintf("/%s", gitAuthConfig.ID), func(r chi.Router) {
				r.Use(
					httpmw.ExtractOAuth2(gitAuthConfig),
					apiKeyMiddleware,
				)
				r.Get("/callback", api.gitAuthCallback(gitAuthConfig))
			})
		}
	})
	r.Route("/api/v2", func(r chi.Router) {
		api.APIHandler = r

		r.NotFound(func(rw http.ResponseWriter, r *http.Request) { httpapi.RouteNotFound(rw) })
		r.Use(
			tracing.Middleware(api.TracerProvider),
			// Specific routes can specify smaller limits.
			httpmw.RateLimit(options.APIRateLimit, time.Minute),
		)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(r.Context(), w, http.StatusOK, codersdk.Response{
				//nolint:gocritic
				Message: "👋",
			})
		})
		// All CSP errors will be logged
		r.Post("/csp/reports", api.logReportCSPViolations)

		r.Route("/buildinfo", func(r chi.Router) {
			r.Get("/", func(rw http.ResponseWriter, r *http.Request) {
				httpapi.Write(r.Context(), rw, http.StatusOK, codersdk.BuildInfoResponse{
					ExternalURL: buildinfo.ExternalURL(),
					Version:     buildinfo.Version(),
				})
			})
		})
		r.Route("/config", func(r chi.Router) {
			r.Use(apiKeyMiddleware)
			r.Get("/deployment", api.deploymentConfig)
		})
		r.Route("/audit", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
			)

			r.Get("/", api.auditLogs)
			r.Get("/count", api.auditLogCount)
			r.Post("/testgenerate", api.generateFakeAuditLog)
		})
		r.Route("/files", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
				// This number is arbitrary, but reading/writing
				// file content is expensive so it should be small.
				httpmw.RateLimit(12, time.Minute),
			)
			r.Get("/{fileID}", api.fileByID)
			r.Post("/", api.postFile)
		})
		r.Route("/organizations", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
			)
			r.Post("/", api.postOrganizations)
			r.Route("/{organization}", func(r chi.Router) {
				r.Use(
					httpmw.ExtractOrganizationParam(options.Database),
				)
				r.Get("/", api.organization)
				r.Route("/templateversions", func(r chi.Router) {
					r.Post("/", api.postTemplateVersionsByOrganization)
					r.Get("/{templateversionname}", api.templateVersionByOrganizationAndName)
				})
				r.Route("/templates", func(r chi.Router) {
					r.Post("/", api.postTemplateByOrganization)
					r.Get("/", api.templatesByOrganization)
					r.Get("/{templatename}", api.templateByOrganizationAndName)
				})
				r.Route("/members", func(r chi.Router) {
					r.Get("/roles", api.assignableOrgRoles)
					r.Route("/{user}", func(r chi.Router) {
						r.Use(
							httpmw.ExtractUserParam(options.Database, false),
							httpmw.ExtractOrganizationMemberParam(options.Database),
						)
						r.Put("/roles", api.putMemberRoles)
						r.Post("/workspaces", api.postWorkspacesByOrganization)
					})
				})
			})
		})
		r.Route("/parameters/{scope}/{id}", func(r chi.Router) {
			r.Use(apiKeyMiddleware)
			r.Post("/", api.postParameter)
			r.Get("/", api.parameters)
			r.Route("/{name}", func(r chi.Router) {
				r.Delete("/", api.deleteParameter)
			})
		})
		r.Route("/templates/{template}", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
				httpmw.ExtractTemplateParam(options.Database),
			)
			r.Get("/daus", api.templateDAUs)
			r.Get("/", api.template)
			r.Delete("/", api.deleteTemplate)
			r.Patch("/", api.patchTemplateMeta)
			r.Route("/versions", func(r chi.Router) {
				r.Get("/", api.templateVersionsByTemplate)
				r.Patch("/", api.patchActiveTemplateVersion)
				r.Get("/{templateversionname}", api.templateVersionByName)
			})
		})
		r.Route("/templateversions/{templateversion}", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
				httpmw.ExtractTemplateVersionParam(options.Database),
			)

			r.Get("/", api.templateVersion)
			r.Patch("/cancel", api.patchCancelTemplateVersion)
			r.Get("/schema", api.templateVersionSchema)
			r.Get("/parameters", api.templateVersionParameters)
			r.Get("/resources", api.templateVersionResources)
			r.Get("/logs", api.templateVersionLogs)
			r.Route("/dry-run", func(r chi.Router) {
				r.Post("/", api.postTemplateVersionDryRun)
				r.Get("/{jobID}", api.templateVersionDryRun)
				r.Get("/{jobID}/resources", api.templateVersionDryRunResources)
				r.Get("/{jobID}/logs", api.templateVersionDryRunLogs)
				r.Patch("/{jobID}/cancel", api.patchTemplateVersionDryRunCancel)
			})
		})
		r.Route("/users", func(r chi.Router) {
			r.Get("/first", api.firstUser)
			r.Post("/first", api.postFirstUser)
			r.Group(func(r chi.Router) {
				// We use a tight limit for password login to protect
				// against audit-log write DoS, pbkdf2 DoS, and simple
				// brute-force attacks.
				//
				// Making this too small can break tests.
				r.Use(httpmw.RateLimit(60, time.Minute))
				r.Post("/login", api.postLogin)
			})
			r.Get("/authmethods", api.userAuthMethods)
			r.Route("/oauth2", func(r chi.Router) {
				r.Route("/github", func(r chi.Router) {
					r.Use(httpmw.ExtractOAuth2(options.GithubOAuth2Config))
					r.Get("/callback", api.userOAuth2Github)
				})
			})
			r.Route("/oidc/callback", func(r chi.Router) {
				r.Use(httpmw.ExtractOAuth2(options.OIDCConfig))
				r.Get("/", api.userOIDC)
			})
			r.Group(func(r chi.Router) {
				r.Use(
					apiKeyMiddleware,
				)
				r.Post("/", api.postUser)
				r.Get("/", api.users)
				r.Post("/logout", api.postLogout)
				// These routes query information about site wide roles.
				r.Route("/roles", func(r chi.Router) {
					r.Get("/", api.assignableSiteRoles)
				})
				r.Route("/{user}", func(r chi.Router) {
					r.Use(httpmw.ExtractUserParam(options.Database, false))
					r.Delete("/", api.deleteUser)
					r.Get("/", api.userByName)
					r.Put("/profile", api.putUserProfile)
					r.Route("/status", func(r chi.Router) {
						r.Put("/suspend", api.putUserStatus(database.UserStatusSuspended))
						r.Put("/activate", api.putUserStatus(database.UserStatusActive))
					})
					r.Route("/password", func(r chi.Router) {
						r.Put("/", api.putUserPassword)
					})
					// These roles apply to the site wide permissions.
					r.Put("/roles", api.putUserRoles)
					r.Get("/roles", api.userRoles)

					r.Route("/keys", func(r chi.Router) {
						r.Post("/", api.postAPIKey)
						r.Route("/tokens", func(r chi.Router) {
							r.Post("/", api.postToken)
							r.Get("/", api.tokens)
						})
						r.Route("/{keyid}", func(r chi.Router) {
							r.Get("/", api.apiKey)
							r.Delete("/", api.deleteAPIKey)
						})
					})

					r.Route("/organizations", func(r chi.Router) {
						r.Get("/", api.organizationsByUser)
						r.Get("/{organizationname}", api.organizationByUserAndName)
					})
					r.Route("/workspace/{workspacename}", func(r chi.Router) {
						r.Get("/", api.workspaceByOwnerAndName)
						r.Get("/builds/{buildnumber}", api.workspaceBuildByBuildNumber)
					})
					r.Get("/gitsshkey", api.gitSSHKey)
					r.Put("/gitsshkey", api.regenerateGitSSHKey)
				})
			})
		})
		r.Route("/workspaceagents", func(r chi.Router) {
			r.Post("/azure-instance-identity", api.postWorkspaceAuthAzureInstanceIdentity)
			r.Post("/aws-instance-identity", api.postWorkspaceAuthAWSInstanceIdentity)
			r.Post("/google-instance-identity", api.postWorkspaceAuthGoogleInstanceIdentity)
			r.Route("/me", func(r chi.Router) {
				r.Use(httpmw.ExtractWorkspaceAgent(options.Database))
				r.Get("/metadata", api.workspaceAgentMetadata)
				r.Post("/version", api.postWorkspaceAgentVersion)
				r.Post("/app-health", api.postWorkspaceAppHealth)
				r.Get("/gitauth", api.workspaceAgentsGitAuth)
				r.Get("/gitsshkey", api.agentGitSSHKey)
				r.Get("/coordinate", api.workspaceAgentCoordinate)
				r.Get("/report-stats", api.workspaceAgentReportStats)
			})
			r.Route("/{workspaceagent}", func(r chi.Router) {
				r.Use(
					apiKeyMiddleware,
					httpmw.ExtractWorkspaceAgentParam(options.Database),
					httpmw.ExtractWorkspaceParam(options.Database),
				)
				r.Get("/", api.workspaceAgent)
				r.Get("/pty", api.workspaceAgentPTY)
				r.Get("/listening-ports", api.workspaceAgentListeningPorts)
				r.Get("/connection", api.workspaceAgentConnection)
				r.Get("/coordinate", api.workspaceAgentClientCoordinate)
				// TODO: This can be removed in October. It allows for a friendly
				// error message when transitioning from WebRTC to Tailscale. See:
				// https://github.com/coder/coder/issues/4126
				r.Get("/dial", func(w http.ResponseWriter, r *http.Request) {
					httpapi.Write(r.Context(), w, http.StatusGone, codersdk.Response{
						Message: "Your Coder CLI is out of date, and requires v0.8.15+ to connect!",
					})
				})
			})
		})
		r.Route("/workspaces", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
			)
			r.Get("/", api.workspaces)
			r.Route("/{workspace}", func(r chi.Router) {
				r.Use(
					httpmw.ExtractWorkspaceParam(options.Database),
				)
				r.Get("/", api.workspace)
				r.Patch("/", api.patchWorkspace)
				r.Route("/builds", func(r chi.Router) {
					r.Get("/", api.workspaceBuilds)
					r.Post("/", api.postWorkspaceBuilds)
				})
				r.Route("/autostart", func(r chi.Router) {
					r.Put("/", api.putWorkspaceAutostart)
				})
				r.Route("/ttl", func(r chi.Router) {
					r.Put("/", api.putWorkspaceTTL)
				})
				r.Get("/watch", api.watchWorkspace)
				r.Put("/extend", api.putExtendWorkspace)
			})
		})
		r.Route("/workspacebuilds/{workspacebuild}", func(r chi.Router) {
			r.Use(
				apiKeyMiddleware,
				httpmw.ExtractWorkspaceBuildParam(options.Database),
				httpmw.ExtractWorkspaceParam(options.Database),
			)
			r.Get("/", api.workspaceBuild)
			r.Patch("/cancel", api.patchCancelWorkspaceBuild)
			r.Get("/logs", api.workspaceBuildLogs)
			r.Get("/resources", api.workspaceBuildResources)
			r.Get("/state", api.workspaceBuildState)
		})
		r.Route("/authcheck", func(r chi.Router) {
			r.Use(apiKeyMiddleware)
			r.Post("/", api.checkAuthorization)
		})
		r.Route("/applications", func(r chi.Router) {
			r.Route("/host", func(r chi.Router) {
				// Don't leak the hostname to unauthenticated users.
				r.Use(apiKeyMiddleware)
				r.Get("/", api.appHost)
			})
			r.Route("/auth-redirect", func(r chi.Router) {
				// We want to redirect to login if they are not authenticated.
				r.Use(apiKeyMiddlewareRedirect)

				// This is a GET request as it's redirected to by the subdomain app
				// handler and the login page.
				r.Get("/", api.workspaceApplicationAuth)
			})
		})
	})

	r.NotFound(compressHandler(http.HandlerFunc(api.siteHandler.ServeHTTP)).ServeHTTP)
	return api
}

type API struct {
	*Options
	// ID is a uniquely generated ID on initialization.
	// This is used to associate objects with a specific
	// Coder API instance, like workspace agents to a
	// specific replica.
	ID                                uuid.UUID
	Auditor                           atomic.Pointer[audit.Auditor]
	WorkspaceClientCoordinateOverride atomic.Pointer[func(rw http.ResponseWriter) bool]
	TailnetCoordinator                atomic.Pointer[tailnet.Coordinator]
	QuotaCommitter                    atomic.Pointer[proto.QuotaCommitter]
	HTTPAuth                          *HTTPAuthorizer

	// APIHandler serves "/api/v2"
	APIHandler chi.Router
	// RootHandler serves "/"
	RootHandler chi.Router

	metricsCache *metricscache.Cache
	siteHandler  http.Handler

	WebsocketWaitMutex sync.Mutex
	WebsocketWaitGroup sync.WaitGroup

	workspaceAgentCache *wsconncache.Cache
}

// Close waits for all WebSocket connections to drain before returning.
func (api *API) Close() error {
	api.WebsocketWaitMutex.Lock()
	api.WebsocketWaitGroup.Wait()
	api.WebsocketWaitMutex.Unlock()

	api.metricsCache.Close()
	coordinator := api.TailnetCoordinator.Load()
	if coordinator != nil {
		_ = (*coordinator).Close()
	}
	return api.workspaceAgentCache.Close()
}

func compressHandler(h http.Handler) http.Handler {
	cmp := middleware.NewCompressor(5,
		"text/*",
		"application/*",
		"image/*",
	)
	cmp.SetEncoder("br", func(w io.Writer, level int) io.Writer {
		return brotli.NewWriterLevel(w, level)
	})
	cmp.SetEncoder("zstd", func(w io.Writer, level int) io.Writer {
		zw, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
		if err != nil {
			panic("invalid zstd compressor: " + err.Error())
		}
		return zw
	})

	return cmp.Handler(h)
}

// CreateInMemoryProvisionerDaemon is an in-memory connection to a provisionerd.  Useful when starting coderd and provisionerd
// in the same process.
func (api *API) CreateInMemoryProvisionerDaemon(ctx context.Context, debounce time.Duration) (client proto.DRPCProvisionerDaemonClient, err error) {
	clientSession, serverSession := provisionersdk.TransportPipe()
	defer func() {
		if err != nil {
			_ = clientSession.Close()
			_ = serverSession.Close()
		}
	}()

	name := namesgenerator.GetRandomName(1)
	daemon, err := api.Database.InsertProvisionerDaemon(ctx, database.InsertProvisionerDaemonParams{
		ID:           uuid.New(),
		CreatedAt:    database.Now(),
		Name:         name,
		Provisioners: []database.ProvisionerType{database.ProvisionerTypeEcho, database.ProvisionerTypeTerraform},
		Tags: dbtype.Map{
			provisionerdserver.TagScope: provisionerdserver.ScopeOrganization,
		},
	})
	if err != nil {
		return nil, xerrors.Errorf("insert provisioner daemon %q: %w", name, err)
	}

	tags, err := json.Marshal(daemon.Tags)
	if err != nil {
		return nil, xerrors.Errorf("marshal tags: %w", err)
	}

	mux := drpcmux.New()
	err = proto.DRPCRegisterProvisionerDaemon(mux, &provisionerdserver.Server{
		AccessURL:          api.AccessURL,
		ID:                 daemon.ID,
		Database:           api.Database,
		Pubsub:             api.Pubsub,
		Provisioners:       daemon.Provisioners,
		Telemetry:          api.Telemetry,
		Tags:               tags,
		QuotaCommitter:     &api.QuotaCommitter,
		AcquireJobDebounce: debounce,
		Logger:             api.Logger.Named(fmt.Sprintf("provisionerd-%s", daemon.Name)),
	})
	if err != nil {
		return nil, err
	}
	server := drpcserver.NewWithOptions(mux, drpcserver.Options{
		Log: func(err error) {
			if xerrors.Is(err, io.EOF) {
				return
			}
			api.Logger.Debug(ctx, "drpc server error", slog.Error(err))
		},
	})
	go func() {
		err := server.Serve(ctx, serverSession)
		if err != nil && !xerrors.Is(err, io.EOF) {
			api.Logger.Debug(ctx, "provisioner daemon disconnected", slog.Error(err))
		}
		// close the sessions so we don't leak goroutines serving them.
		_ = clientSession.Close()
		_ = serverSession.Close()
	}()

	return proto.NewDRPCProvisionerDaemonClient(provisionersdk.Conn(clientSession)), nil
}
