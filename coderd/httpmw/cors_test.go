package httpmw_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
)

func TestWorkspaceAppCors(t *testing.T) {
	t.Parallel()

	regex, err := httpapi.CompileHostnamePattern("*--apps.dev.coder.com")
	require.NoError(t, err)

	app := httpapi.ApplicationURL{
		AppSlugOrPort: "3000",
		AgentName:     "agent",
		WorkspaceName: "ws",
		Username:      "user",
	}

	handler := httpmw.WorkspaceAppCors(regex, app)(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	}))
	methods := []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	tests := []struct {
		name    string
		origin  string
		allowed bool
	}{
		{
			name:    "Self",
			origin:  "https://3000--agent--ws--user--apps.dev.coder.com",
			allowed: true,
		},
		{
			name:    "SameWorkspace",
			origin:  "https://8000--agent--ws--user--apps.dev.coder.com",
			allowed: true,
		},
		{
			name:    "SameUser",
			origin:  "https://8000--agent2--ws2--user--apps.dev.coder.com",
			allowed: true,
		},
		{
			name:    "DifferentUser",
			origin:  "https://3000--agent--ws--user2--apps.dev.coder.com",
			allowed: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			for _, method := range methods {
				r := httptest.NewRequest(method, "http://localhost", nil)
				r.Header.Set("Origin", test.origin)
				rw := httptest.NewRecorder()

				// Preflight requests need to know what method will be requested.
				if method == http.MethodOptions {
					r.Header.Set("Access-Control-Request-Method", method)
				}

				handler.ServeHTTP(rw, r)

				if test.allowed {
					require.Equal(t, test.origin, rw.Header().Get("Access-Control-Allow-Origin"))
				} else {
					require.Equal(t, "", rw.Header().Get("Access-Control-Allow-Origin"))
				}

				// For options we should never get to our handler as the middleware
				// short-circuits with a 200.
				if method == http.MethodOptions {
					require.Equal(t, http.StatusOK, rw.Code)
				} else {
					require.Equal(t, http.StatusNoContent, rw.Code)
				}
			}
		})
	}
}
