package coderd

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/coderd/tracing"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/site"
)

// workspaceAppsProxyPath proxies requests to a workspace application
// through a relative URL path.
func (api *API) workspaceAppsProxyPath(rw http.ResponseWriter, r *http.Request) {
	workspace := httpmw.WorkspaceParam(r)
	agent := httpmw.WorkspaceAgentParam(r)

	if !api.Authorize(r, rbac.ActionCreate, workspace.ApplicationConnectRBAC()) {
		httpapi.ResourceNotFound(rw)
		return
	}

	// Determine the real path that was hit. The * URL parameter in Chi will not
	// include the leading slash if it was present, so we need to add it back.
	chiPath := chi.URLParam(r, "*")
	basePath := strings.TrimSuffix(r.URL.Path, chiPath)
	if strings.HasSuffix(basePath, "/") {
		chiPath = "/" + chiPath
	}

	api.proxyWorkspaceApplication(proxyApplication{
		Workspace: workspace,
		Agent:     agent,
		// We do not support port proxying for paths.
		AppName:          chi.URLParam(r, "workspaceapp"),
		Port:             0,
		Path:             chiPath,
		DashboardOnError: true,
	}, rw, r)
}

// handleSubdomainApplications handles subdomain-based application proxy
// requests (aka. DevURLs in Coder V1).
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
func (api *API) handleSubdomainApplications(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if api.AppHostname == "" {
				next.ServeHTTP(rw, r)
				return
			}

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

				httpapi.Write(rw, http.StatusBadRequest, codersdk.Response{
					Message: "Could not determine request Host.",
				})
				return
			}

			// Check if the hostname matches the access URL. If it does, the
			// user was definitely trying to connect to the dashboard/API.
			if httpapi.HostnamesMatch(api.AccessURL.Hostname(), host) {
				next.ServeHTTP(rw, r)
				return
			}

			// Split the subdomain so we can parse the application details and
			// verify it matches the configured app hostname later.
			subdomain, rest := httpapi.SplitSubdomain(host)
			if rest == "" {
				// If there are no periods in the hostname, then it can't be a
				// valid application URL.
				next.ServeHTTP(rw, r)
				return
			}
			matchingBaseHostname := httpapi.HostnamesMatch(api.AppHostname, rest)

			// Parse the application URL from the subdomain.
			app, err := httpapi.ParseSubdomainAppURL(subdomain)
			if err != nil {
				// If it isn't a valid app URL and the base domain doesn't match
				// the configured app hostname, this request was probably
				// destined for the dashboard/API router.
				if !matchingBaseHostname {
					next.ServeHTTP(rw, r)
					return
				}

				httpapi.Write(rw, http.StatusBadRequest, codersdk.Response{
					Message: "Could not parse subdomain application URL.",
					Detail:  err.Error(),
				})
				return
			}

			// At this point we've verified that the subdomain looks like a
			// valid application URL, so the base hostname should match the
			// configured app hostname.
			if !matchingBaseHostname {
				httpapi.Write(rw, http.StatusNotFound, codersdk.Response{
					Message: "The server does not accept application requests on this hostname.",
				})
				return
			}

			workspaceAgentKey := fmt.Sprintf("%s.%s", app.WorkspaceName, app.AgentName)
			chiCtx := chi.RouteContext(ctx)
			chiCtx.URLParams.Add("workspace_and_agent", workspaceAgentKey)
			chiCtx.URLParams.Add("user", app.Username)

			// Use the passed in app middlewares before passing to the proxy app.
			mws := chi.Middlewares(middlewares)
			mws.Handler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				workspace := httpmw.WorkspaceParam(r)
				agent := httpmw.WorkspaceAgentParam(r)

				api.proxyWorkspaceApplication(proxyApplication{
					Workspace:        workspace,
					Agent:            agent,
					AppName:          app.AppName,
					Port:             app.Port,
					Path:             r.URL.Path,
					DashboardOnError: false,
				}, rw, r)
			})).ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

// proxyApplication are the required fields to proxy a workspace application.
type proxyApplication struct {
	Workspace database.Workspace
	Agent     database.WorkspaceAgent

	// Either AppName or Port must be set, but not both.
	AppName string
	Port    uint16
	// Path must either be empty or have a leading slash.
	Path string

	// DashboardOnError determines whether or not the dashboard should be
	// rendered on error. This should be set for proxy path URLs but not
	// hostname based URLs.
	DashboardOnError bool
}

func (api *API) proxyWorkspaceApplication(proxyApp proxyApplication, rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.Authorize(r, rbac.ActionCreate, proxyApp.Workspace.ApplicationConnectRBAC()) {
		httpapi.ResourceNotFound(rw)
		return
	}

	// If the app does not exist, but the app name is a port number, then
	// route to the port as an "anonymous app". We only support HTTP for
	// port-based URLs.
	internalURL := fmt.Sprintf("http://127.0.0.1:%d", proxyApp.Port)

	// If the app name was used instead, fetch the app from the database so we
	// can get the internal URL.
	if proxyApp.AppName != "" {
		app, err := api.Database.GetWorkspaceAppByAgentIDAndName(ctx, database.GetWorkspaceAppByAgentIDAndNameParams{
			AgentID: proxyApp.Agent.ID,
			Name:    proxyApp.AppName,
		})
		if err != nil {
			httpapi.Write(rw, http.StatusInternalServerError, codersdk.Response{
				Message: "Internal error fetching workspace application.",
				Detail:  err.Error(),
			})
			return
		}

		if !app.Url.Valid {
			httpapi.Write(rw, http.StatusBadRequest, codersdk.Response{
				Message: fmt.Sprintf("Application %s does not have a url.", app.Name),
			})
			return
		}
		internalURL = app.Url.String
	}

	appURL, err := url.Parse(internalURL)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, codersdk.Response{
			Message: fmt.Sprintf("App URL %q is invalid.", internalURL),
			Detail:  err.Error(),
		})
		return
	}

	// Ensure path and query parameter correctness.
	if proxyApp.Path == "" {
		// Web applications typically request paths relative to the
		// root URL. This allows for routing behind a proxy or subpath.
		// See https://github.com/coder/code-server/issues/241 for examples.
		http.Redirect(rw, r, r.URL.Path+"/", http.StatusTemporaryRedirect)
		return
	}
	if proxyApp.Path == "/" && r.URL.RawQuery == "" && appURL.RawQuery != "" {
		// If the application defines a default set of query parameters,
		// we should always respect them. The reverse proxy will merge
		// query parameters for server-side requests, but sometimes
		// client-side applications require the query parameters to render
		// properly. With code-server, this is the "folder" param.
		r.URL.RawQuery = appURL.RawQuery
		http.Redirect(rw, r, r.URL.String(), http.StatusTemporaryRedirect)
		return
	}

	r.URL.Path = proxyApp.Path
	appURL.RawQuery = ""

	proxy := httputil.NewSingleHostReverseProxy(appURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if proxyApp.DashboardOnError {
			// To pass friendly errors to the frontend, special meta tags are
			// overridden in the index.html with the content passed here.
			r = r.WithContext(site.WithAPIResponse(ctx, site.APIResponse{
				StatusCode: http.StatusBadGateway,
				Message:    err.Error(),
			}))
			api.siteHandler.ServeHTTP(w, r)
			return
		}

		httpapi.Write(w, http.StatusBadGateway, codersdk.Response{
			Message: "Failed to proxy request to application.",
			Detail:  err.Error(),
		})
	}

	conn, release, err := api.workspaceAgentCache.Acquire(r, proxyApp.Agent.ID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to dial workspace agent.",
			Detail:  err.Error(),
		})
		return
	}
	defer release()

	// This strips the session token from a workspace app request.
	cookieHeaders := r.Header.Values("Cookie")[:]
	r.Header.Del("Cookie")
	for _, cookieHeader := range cookieHeaders {
		r.Header.Add("Cookie", httpapi.StripCoderCookies(cookieHeader))
	}
	proxy.Transport = conn.HTTPTransport()

	// end span so we don't get long lived trace data
	tracing.EndHTTPSpan(r, http.StatusOK, trace.SpanFromContext(ctx))

	proxy.ServeHTTP(rw, r)
}

// applicationCookie is a helper function to copy the auth cookie to also
// support subdomains. Until we support creating authentication cookies that can
// only do application authentication, we will just reuse the original token.
// This code should be temporary and be replaced with something that creates
// a unique session_token.
//
// Returns nil if the access URL isn't a hostname.
func (api *API) applicationCookie(authCookie *http.Cookie) *http.Cookie {
	if net.ParseIP(api.AccessURL.Hostname()) != nil {
		return nil
	}

	appCookie := *authCookie
	// We only support setting this cookie on the access URL subdomains. This is
	// to ensure we don't accidentally leak the auth cookie to subdomains on
	// another hostname.
	appCookie.Domain = "." + api.AccessURL.Hostname()
	return &appCookie
}
