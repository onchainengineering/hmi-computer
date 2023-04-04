package workspaceapps

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"nhooyr.io/websocket"

	"cdr.dev/slog"
	"github.com/coder/coder/agent"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/tracing"
	"github.com/coder/coder/coderd/wsconncache"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/site"
)

const (
	// This needs to be a super unique query parameter because we don't want to
	// conflict with query parameters that users may use.
	//nolint:gosec
	SubdomainProxyAPIKeyParam = "coder_application_connect_api_key_35e783"
	// appLogoutHostname is the hostname to use for the logout redirect. When
	// the dashboard logs out, it will redirect to this subdomain of the app
	// hostname, and the server will remove the cookie and redirect to the main
	// login page.
	// It is important that this URL can never match a valid app hostname.
	//
	// DEPRECATED: we no longer use this, but we still redirect from it to the
	// main login page.
	appLogoutHostname = "coder-logout"
)

// nonCanonicalHeaders is a map from "canonical" headers to the actual header we
// should send to the app in the workspace. Some headers (such as the websocket
// upgrade headers from RFC 6455) are not canonical according to the HTTP/1
// spec. Golang has said that they will not add custom cases for these headers,
// so we need to do it ourselves.
//
// Some apps our customers use are sensitive to the case of these headers.
//
// https://github.com/golang/go/issues/18495
var nonCanonicalHeaders = map[string]string{
	"Sec-Websocket-Accept":     "Sec-WebSocket-Accept",
	"Sec-Websocket-Extensions": "Sec-WebSocket-Extensions",
	"Sec-Websocket-Key":        "Sec-WebSocket-Key",
	"Sec-Websocket-Protocol":   "Sec-WebSocket-Protocol",
	"Sec-Websocket-Version":    "Sec-WebSocket-Version",
}

// WorkspaceAppServer serves workspace apps endpoints, including:
// - Path-based apps
// - Subdomain app middleware
// - Workspace reconnecting-pty (aka. web terminal)
type WorkspaceAppServer struct {
	Logger slog.Logger

	// PrimaryAccessURL should be a url to the coderd dashboard. This can be the
	// same as the AccessURL if the WorkspaceAppServer is embedded.
	PrimaryAccessURL *url.URL
	AccessURL        *url.URL
	AppHostname      string
	AppHostnameRegex *regexp.Regexp
	DeploymentValues *codersdk.DeploymentValues
	RealIPConfig     *httpmw.RealIPConfig

	TokenProvider      SignedTokenProvider
	WorkspaceConnCache *wsconncache.Cache
	AppSigningKey      AppSigningKey
}

func (s *WorkspaceAppServer) Attach(r chi.Router, pathAppRateLimiter func(http.Handler) http.Handler) {
	servePathApps := func(r chi.Router) {
		r.Use(pathAppRateLimiter)
		r.HandleFunc("/*", s.workspaceAppsProxyPath)
	}

	// %40 is the encoded character of the @ symbol. VS Code Web does
	// not handle character encoding properly, so it's safe to assume
	// other applications might not as well.
	r.Route("/%40{user}/{workspace_and_agent}/apps/{workspaceapp}", servePathApps)
	r.Route("/@{user}/{workspace_and_agent}/apps/{workspaceapp}", servePathApps)

	r.Get("/api/v2/workspaceagents/{workspaceagent}/pty", s.workspaceAgentPTY)
}

// workspaceAppsProxyPath proxies requests to a workspace application
// through a relative URL path.
func (s *WorkspaceAppServer) workspaceAppsProxyPath(rw http.ResponseWriter, r *http.Request) {
	if s.DeploymentValues.DisablePathApps.Value() {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusUnauthorized,
			Title:        "Unauthorized",
			Description:  "Path-based applications are disabled on this Coder deployment by the administrator.",
			RetryEnabled: false,
			DashboardURL: s.PrimaryAccessURL.String(),
		})
		return
	}

	// We don't support @me in path apps since it requires the database to
	// lookup the username from token. We used to redirect by doing this lookup.
	if chi.URLParam(r, "user") == codersdk.Me {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusNotFound,
			Title:        "Application Not Found",
			Description:  "Applications must be accessed with the full username, not @me.",
			RetryEnabled: false,
			DashboardURL: s.PrimaryAccessURL.String(),
		})
		return
	}

	// Determine the real path that was hit. The * URL parameter in Chi will not
	// include the leading slash if it was present, so we need to add it back.
	chiPath := chi.URLParam(r, "*")
	basePath := strings.TrimSuffix(r.URL.Path, chiPath)
	if strings.HasSuffix(basePath, "/") {
		chiPath = "/" + chiPath
	}

	// ResolveRequest will only return a new signed token if the actor has the RBAC
	// permissions to connect to a workspace.
	token, ok := ResolveRequest(s.Logger, s.AccessURL, s.TokenProvider, rw, r, Request{
		AccessMethod:      AccessMethodPath,
		BasePath:          basePath,
		UsernameOrID:      chi.URLParam(r, "user"),
		WorkspaceAndAgent: chi.URLParam(r, "workspace_and_agent"),
		// We don't support port proxying on paths. The ResolveRequest method
		// won't allow port proxying on path-based apps if the app is a number.
		AppSlugOrPort: chi.URLParam(r, "workspaceapp"),
	})
	if !ok {
		return
	}

	s.proxyWorkspaceApp(rw, r, *token, chiPath)
}

// SubdomainAppMW handles subdomain-based application proxy requests (aka.
// DevURLs in Coder V1).
//
// There are a lot of paths here:
//  1. If api.AppHostname is not set then we pass on.
//  2. If we can't read the request hostname then we return a 400.
//  3. If the request hostname matches api.AccessURL then we pass on.
//  5. We split the subdomain into the subdomain and the "rest". If there are no
//     periods in the hostname then we pass on.
//  5. We parse the subdomain into a httpapi.ApplicationURL struct. If we
//     encounter an error:
//     a. If the "rest" does not match api.AppHostname then we pass on;
//     b. Otherwise, we return a 400.
//  6. Finally, we verify that the "rest" matches api.AppHostname, else we
//     return a 404.
//
// Rationales for each of the above steps:
//  1. We pass on if api.AppHostname is not set to avoid returning any errors if
//     `--app-hostname` is not configured.
//  2. Every request should have a valid Host header anyways.
//  3. We pass on if the request hostname matches api.AccessURL so we can
//     support having the access URL be at the same level as the application
//     base hostname.
//  4. We pass on if there are no periods in the hostname as application URLs
//     must be a subdomain of a hostname, which implies there must be at least
//     one period.
//  5. a. If the request subdomain is not a valid application URL, and the
//     "rest" does not match api.AppHostname, then it is very unlikely that
//     the request was intended for this handler. We pass on.
//     b. If the request subdomain is not a valid application URL, but the
//     "rest" matches api.AppHostname, then we return a 400 because the
//     request is probably a typo or something.
//  6. We finally verify that the "rest" matches api.AppHostname for security
//     purposes regarding re-authentication and application proxy session
//     tokens.
func (s *WorkspaceAppServer) SubdomainAppMW(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Step 1: Pass on if subdomain-based application proxying is not
			// configured.
			if s.AppHostname == "" || s.AppHostnameRegex == nil {
				next.ServeHTTP(rw, r)
				return
			}

			// Step 2: Get the request Host.
			host := httpapi.RequestHost(r)
			if host == "" {
				if r.URL.Path == "/derp" {
					// The /derp endpoint is used by wireguard clients to tunnel
					// through coderd. For some reason these requests don't set
					// a Host header properly sometimes in tests (no idea how),
					// which causes this path to get hit.
					next.ServeHTTP(rw, r)
					return
				}

				httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
					Message: "Could not determine request Host.",
				})
				return
			}

			// Steps 3-6: Parse application from subdomain.
			app, ok := s.parseWorkspaceApplicationHostname(rw, r, next, host)
			if !ok {
				return
			}

			// If the request has the special query param then we need to set a
			// cookie and strip that query parameter.
			if encryptedAPIKey := r.URL.Query().Get(SubdomainProxyAPIKeyParam); encryptedAPIKey != "" {
				// Exchange the encoded API key for a real one.
				token, err := s.AppSigningKey.DecryptAPIKey(encryptedAPIKey)
				if err != nil {
					s.Logger.Debug(ctx, "could not decrypt API key", slog.Error(err))
					site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
						Status:      http.StatusBadRequest,
						Title:       "Bad Request",
						Description: "Could not decrypt API key. Please remove the query parameter and try again.",
						// Retry is disabled because the user needs to remove
						// the query parameter before they try again.
						RetryEnabled: false,
						DashboardURL: s.AccessURL.String(),
					})
					return
				}

				s.setWorkspaceAppCookie(rw, r, token)

				// Strip the query parameter.
				path := r.URL.Path
				if path == "" {
					path = "/"
				}
				q := r.URL.Query()
				q.Del(SubdomainProxyAPIKeyParam)
				rawQuery := q.Encode()
				if rawQuery != "" {
					path += "?" + q.Encode()
				}

				http.Redirect(rw, r, path, http.StatusTemporaryRedirect)
				return
			}

			token, ok := ResolveRequest(s.Logger, s.AccessURL, s.TokenProvider, rw, r, Request{
				AccessMethod:      AccessMethodSubdomain,
				BasePath:          "/",
				UsernameOrID:      app.Username,
				WorkspaceNameOrID: app.WorkspaceName,
				AgentNameOrID:     app.AgentName,
				AppSlugOrPort:     app.AppSlugOrPort,
			})
			if !ok {
				return
			}

			// Use the passed in app middlewares before passing to the proxy
			// app.
			mws := chi.Middlewares(middlewares)
			mws.Handler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				s.proxyWorkspaceApp(rw, r, *token, r.URL.Path)
			})).ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

func (s *WorkspaceAppServer) parseWorkspaceApplicationHostname(rw http.ResponseWriter, r *http.Request, next http.Handler, host string) (httpapi.ApplicationURL, bool) {
	// Check if the hostname matches either of the access URLs. If it does, the
	// user was definitely trying to connect to the dashboard/API or a
	// path-based app.
	if httpapi.HostnamesMatch(s.PrimaryAccessURL.Hostname(), host) || httpapi.HostnamesMatch(s.AccessURL.Hostname(), host) {
		next.ServeHTTP(rw, r)
		return httpapi.ApplicationURL{}, false
	}

	// If there are no periods in the hostname, then it can't be a valid
	// application URL.
	if !strings.Contains(host, ".") {
		next.ServeHTTP(rw, r)
		return httpapi.ApplicationURL{}, false
	}

	// Split the subdomain so we can parse the application details and verify it
	// matches the configured app hostname later.
	subdomain, ok := httpapi.ExecuteHostnamePattern(s.AppHostnameRegex, host)
	if !ok {
		// Doesn't match the regex, so it's not a valid application URL.
		next.ServeHTTP(rw, r)
		return httpapi.ApplicationURL{}, false
	}

	// Check if the request is part of the deprecated logout flow. If so, we
	// just redirect to the main access URL.
	if subdomain == appLogoutHostname {
		http.Redirect(rw, r, s.AccessURL.String(), http.StatusTemporaryRedirect)
		return httpapi.ApplicationURL{}, false
	}

	// Parse the application URL from the subdomain.
	app, err := httpapi.ParseSubdomainAppURL(subdomain)
	if err != nil {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusBadRequest,
			Title:        "Invalid Application URL",
			Description:  fmt.Sprintf("Could not parse subdomain application URL %q: %s", subdomain, err.Error()),
			RetryEnabled: false,
			DashboardURL: s.AccessURL.String(),
		})
		return httpapi.ApplicationURL{}, false
	}

	return app, true
}

// setWorkspaceAppCookie sets a cookie on the workspace app domain. If the app
// hostname cannot be parsed properly, a static error page is rendered and false
// is returned.
func (s *WorkspaceAppServer) setWorkspaceAppCookie(rw http.ResponseWriter, r *http.Request, token string) bool {
	hostSplit := strings.SplitN(s.AppHostname, ".", 2)
	if len(hostSplit) != 2 {
		// This should be impossible as we verify the app hostname on
		// startup, but we'll check anyways.
		s.Logger.Error(r.Context(), "could not split invalid app hostname", slog.F("hostname", s.AppHostname))
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusInternalServerError,
			Title:        "Internal Server Error",
			Description:  "The app is configured with an invalid app wildcard hostname. Please contact an administrator.",
			RetryEnabled: false,
			DashboardURL: s.AccessURL.String(),
		})
		return false
	}

	// Set the app cookie for all subdomains of s.AppHostname. We don't set an
	// expiration because the key in the database already has an expiration, and
	// expired tokens don't affect the user experience (they get auto-redirected
	// to re-smuggle the API key).
	cookieHost := "." + hostSplit[1]
	http.SetCookie(rw, &http.Cookie{
		Name:     codersdk.DevURLSessionTokenCookie,
		Value:    token,
		Domain:   cookieHost,
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.DeploymentValues.SecureAuthCookie.Value(),
	})

	return true
}

func (s *WorkspaceAppServer) proxyWorkspaceApp(rw http.ResponseWriter, r *http.Request, appToken SignedToken, path string) {
	ctx := r.Context()

	// Filter IP headers from untrusted origins.
	httpmw.FilterUntrustedOriginHeaders(s.RealIPConfig, r)

	// Ensure proper IP headers get sent to the forwarded application.
	err := httpmw.EnsureXForwardedForHeader(r)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	appURL, err := url.Parse(appToken.AppURL)
	if err != nil {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusBadRequest,
			Title:        "Bad Request",
			Description:  fmt.Sprintf("Application has an invalid URL %q: %s", appToken.AppURL, err.Error()),
			RetryEnabled: true,
			DashboardURL: s.PrimaryAccessURL.String(),
		})
		return
	}

	// Verify that the port is allowed. See the docs above
	// `codersdk.MinimumListeningPort` for more details.
	port := appURL.Port()
	if port != "" {
		portInt, err := strconv.Atoi(port)
		if err != nil {
			httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
				Message: fmt.Sprintf("App URL %q has an invalid port %q.", appToken.AppURL, port),
				Detail:  err.Error(),
			})
			return
		}

		if portInt < codersdk.WorkspaceAgentMinimumListeningPort {
			httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
				Message: fmt.Sprintf("Application port %d is not permitted. Coder reserves ports less than %d for internal use.", portInt, codersdk.WorkspaceAgentMinimumListeningPort),
			})
			return
		}
	}

	// Ensure path and query parameter correctness.
	if path == "" {
		// Web applications typically request paths relative to the
		// root URL. This allows for routing behind a proxy or subpath.
		// See https://github.com/coder/code-server/issues/241 for examples.
		http.Redirect(rw, r, r.URL.Path+"/", http.StatusTemporaryRedirect)
		return
	}
	if path == "/" && r.URL.RawQuery == "" && appURL.RawQuery != "" {
		// If the application defines a default set of query parameters,
		// we should always respect them. The reverse proxy will merge
		// query parameters for server-side requests, but sometimes
		// client-side applications require the query parameters to render
		// properly. With code-server, this is the "folder" param.
		r.URL.RawQuery = appURL.RawQuery
		http.Redirect(rw, r, r.URL.String(), http.StatusTemporaryRedirect)
		return
	}

	r.URL.Path = path
	appURL.RawQuery = ""

	proxy := httputil.NewSingleHostReverseProxy(appURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusBadGateway,
			Title:        "Bad Gateway",
			Description:  "Failed to proxy request to application: " + err.Error(),
			RetryEnabled: true,
			DashboardURL: s.PrimaryAccessURL.String(),
		})
	}

	conn, release, err := s.WorkspaceConnCache.Acquire(appToken.AgentID)
	if err != nil {
		site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
			Status:       http.StatusBadGateway,
			Title:        "Bad Gateway",
			Description:  "Could not connect to workspace agent: " + err.Error(),
			RetryEnabled: true,
			DashboardURL: s.PrimaryAccessURL.String(),
		})
		return
	}
	defer release()
	proxy.Transport = conn.HTTPTransport()

	// This strips the session token from a workspace app request.
	cookieHeaders := r.Header.Values("Cookie")[:]
	r.Header.Del("Cookie")
	for _, cookieHeader := range cookieHeaders {
		r.Header.Add("Cookie", httpapi.StripCoderCookies(cookieHeader))
	}

	// Convert canonicalized headers to their non-canonicalized counterparts.
	// See the comment on `nonCanonicalHeaders` for more information on why this
	// is necessary.
	for k, v := range r.Header {
		if n, ok := nonCanonicalHeaders[k]; ok {
			r.Header.Del(k)
			r.Header[n] = v
		}
	}

	// end span so we don't get long lived trace data
	tracing.EndHTTPSpan(r, http.StatusOK, trace.SpanFromContext(ctx))

	proxy.ServeHTTP(rw, r)
}

// workspaceAgentPTY spawns a PTY and pipes it over a WebSocket.
// This is used for the web terminal.
//
// @Summary Open PTY to workspace agent
// @ID open-pty-to-workspace-agent
// @Security CoderSessionToken
// @Tags Agents
// @Param workspaceagent path string true "Workspace agent ID" format(uuid)
// @Success 101
// @Router /workspaceagents/{workspaceagent}/pty [get]
func (s *WorkspaceAppServer) workspaceAgentPTY(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Fix this later
	// s.WebsocketWaitMutex.Lock()
	// s.WebsocketWaitGroup.Add(1)
	// s.WebsocketWaitMutex.Unlock()
	// defer s.WebsocketWaitGroup.Done()

	appToken, ok := ResolveRequest(s.Logger, s.AccessURL, s.TokenProvider, rw, r, Request{
		AccessMethod:  AccessMethodTerminal,
		BasePath:      r.URL.Path,
		AgentNameOrID: chi.URLParam(r, "workspaceagent"),
	})
	if !ok {
		return
	}

	values := r.URL.Query()
	parser := httpapi.NewQueryParamParser()
	reconnect := parser.Required("reconnect").UUID(values, uuid.New(), "reconnect")
	height := parser.UInt(values, 80, "height")
	width := parser.UInt(values, 80, "width")
	if len(parser.Errors) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message:     "Invalid query parameters.",
			Validations: parser.Errors,
		})
		return
	}

	conn, err := websocket.Accept(rw, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Failed to accept websocket.",
			Detail:  err.Error(),
		})
		return
	}

	ctx, wsNetConn := WebsocketNetConn(ctx, conn, websocket.MessageBinary)
	defer wsNetConn.Close() // Also closes conn.

	go httpapi.Heartbeat(ctx, conn)

	agentConn, release, err := s.WorkspaceConnCache.Acquire(appToken.AgentID)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, httpapi.WebsocketCloseSprintf("dial workspace agent: %s", err))
		return
	}
	defer release()
	ptNetConn, err := agentConn.ReconnectingPTY(ctx, reconnect, uint16(height), uint16(width), r.URL.Query().Get("command"))
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, httpapi.WebsocketCloseSprintf("dial: %s", err))
		return
	}
	defer ptNetConn.Close()
	agent.Bicopy(ctx, wsNetConn, ptNetConn)
}

// wsNetConn wraps net.Conn created by websocket.NetConn(). Cancel func
// is called if a read or write error is encountered.
type wsNetConn struct {
	cancel context.CancelFunc
	net.Conn
}

func (c *wsNetConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		c.cancel()
	}
	return n, err
}

func (c *wsNetConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		c.cancel()
	}
	return n, err
}

func (c *wsNetConn) Close() error {
	defer c.cancel()
	return c.Conn.Close()
}

// WebsocketNetConn wraps websocket.NetConn and returns a context that
// is tied to the parent context and the lifetime of the conn. Any error
// during read or write will cancel the context, but not close the
// conn. Close should be called to release context resources.
func WebsocketNetConn(ctx context.Context, conn *websocket.Conn, msgType websocket.MessageType) (context.Context, net.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	nc := websocket.NetConn(ctx, conn, msgType)
	return ctx, &wsNetConn{
		cancel: cancel,
		Conn:   nc,
	}
}
