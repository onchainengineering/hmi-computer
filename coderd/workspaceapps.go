package coderd

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/xerrors"

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

	if !api.Authorize(r, rbac.ActionCreate, workspace.ExecutionRBAC()) {
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

	appName, port := AppNameOrPort(chi.URLParam(r, "workspaceapp"))
	api.proxyWorkspaceApplication(proxyApplication{
		Workspace:        workspace,
		Agent:            agent,
		AppName:          appName,
		Port:             port,
		Path:             chiPath,
		DashboardOnError: true,
	}, rw, r)
}

func (api *API) handleSubdomainApplications(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			host := httpapi.RequestHost(r)
			if host == "" {
				httpapi.Write(rw, http.StatusBadRequest, codersdk.Response{
					Message: "Could not determine request Host.",
				})
				return
			}

			app, err := ParseSubdomainAppURL(host)
			if err != nil {
				// Subdomain is not a valid application url. Pass through to the
				// rest of the app.
				// TODO: @emyrk we should probably catch invalid subdomains. Meaning
				// 	an invalid application should not route to the coderd.
				//	To do this we would need to know the list of valid access urls
				//	though?
				next.ServeHTTP(rw, r)
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
	if !api.Authorize(r, rbac.ActionCreate, proxyApp.Workspace.ExecutionRBAC()) {
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
		app, err := api.Database.GetWorkspaceAppByAgentIDAndName(r.Context(), database.GetWorkspaceAppByAgentIDAndNameParams{
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
			r = r.WithContext(site.WithAPIResponse(r.Context(), site.APIResponse{
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
	tracing.EndHTTPSpan(r, 200)

	proxy.ServeHTTP(rw, r)
}

var (
	// Remove the "starts with" and "ends with" regex components.
	nameRegex = strings.Trim(httpapi.UsernameValidRegex.String(), "^$")
	appURL    = regexp.MustCompile(fmt.Sprintf(
		// {PORT/APP_NAME}--{AGENT_NAME}--{WORKSPACE_NAME}--{USERNAME}
		`^(?P<AppName>%[1]s)--(?P<AgentName>%[1]s)--(?P<WorkspaceName>%[1]s)--(?P<Username>%[1]s)$`,
		nameRegex))
)

// ApplicationURL is a parsed application URL hostname.
type ApplicationURL struct {
	// Only one of AppName or Port will be set.
	AppName       string
	Port          uint16
	AgentName     string
	WorkspaceName string
	Username      string
	// BaseHostname is the rest of the hostname minus the application URL part
	// and the first dot.
	BaseHostname string
}

// String returns the application URL hostname without scheme.
func (a ApplicationURL) String() string {
	appNameOrPort := a.AppName
	if a.Port != 0 {
		appNameOrPort = strconv.Itoa(int(a.Port))
	}

	return fmt.Sprintf("%s--%s--%s--%s.%s", appNameOrPort, a.AgentName, a.WorkspaceName, a.Username, a.BaseHostname)
}

// ParseSubdomainAppURL parses an application from the subdomain of r's Host
// header. If the subdomain is not a valid application URL hostname, returns a
// non-nil error.
//
// Subdomains should be in the form:
//
//	{PORT/APP_NAME}--{AGENT_NAME}--{WORKSPACE_NAME}--{USERNAME}
//	(eg. http://8080--main--dev--dean.hi.c8s.io)
func ParseSubdomainAppURL(hostname string) (ApplicationURL, error) {
	subdomain, rest, err := SplitSubdomain(hostname)
	if err != nil {
		return ApplicationURL{}, xerrors.Errorf("split host domain %q: %w", hostname, err)
	}

	matches := appURL.FindAllStringSubmatch(subdomain, -1)
	if len(matches) == 0 {
		return ApplicationURL{}, xerrors.Errorf("invalid application url format: %q", subdomain)
	}
	matchGroup := matches[0]

	appName, port := AppNameOrPort(matchGroup[appURL.SubexpIndex("AppName")])
	return ApplicationURL{
		AppName:       appName,
		Port:          port,
		AgentName:     matchGroup[appURL.SubexpIndex("AgentName")],
		WorkspaceName: matchGroup[appURL.SubexpIndex("WorkspaceName")],
		Username:      matchGroup[appURL.SubexpIndex("Username")],
		BaseHostname:  rest,
	}, nil
}

// SplitSubdomain splits a subdomain from the rest of the hostname. E.g.:
//   - "foo.bar.com" becomes "foo", "bar.com"
//   - "foo.bar.baz.com" becomes "foo", "bar.baz.com"
//
// An error is returned if the string doesn't contain a period.
func SplitSubdomain(hostname string) (subdomain string, rest string, err error) {
	toks := strings.SplitN(hostname, ".", 2)
	if len(toks) < 2 {
		return "", "", xerrors.New("no subdomain")
	}

	return toks[0], toks[1], nil
}

// AppNameOrPort takes a string and returns either the input string or a port
// number.
func AppNameOrPort(val string) (string, uint16) {
	port, err := strconv.ParseUint(val, 10, 16)
	if err != nil || port == 0 {
		port = 0
	} else {
		val = ""
	}

	return val, uint16(port)
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
