package httpmw_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
)

func TestExtractProvisionerDaemonAuthenticated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		opts                    httpmw.ExtractProvisionerAuthConfig
		expectedStatusCode      int
		expectedResponseMessage string
		provisionerKey          string
		provisionerPSK          string
	}{
		{
			name: "NoKeyProvided_Optional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: true,
			},
			expectedStatusCode:      http.StatusOK,
			expectedResponseMessage: "",
		},
		{
			name: "NoKeyProvided_NotOptional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: false,
			},
			expectedStatusCode:      http.StatusUnauthorized,
			expectedResponseMessage: "provisioner daemon key required",
		},
		{
			name: "ProvisionerKeyAndPSKProvided_NotOptional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: false,
			},
			provisionerKey:          "key",
			provisionerPSK:          "psk",
			expectedStatusCode:      http.StatusBadRequest,
			expectedResponseMessage: "provisioner daemon key and psk provided, but only one is allowed",
		},
		{
			name: "ProvisionerKeyAndPSKProvided_Optional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: true,
			},
			provisionerKey:          "key",
			expectedStatusCode:      http.StatusOK,
			expectedResponseMessage: "",
		},
		{
			name: "InvalidProvisionerKey_NotOptional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: false,
			},
			provisionerKey:          "invalid",
			expectedStatusCode:      http.StatusBadRequest,
			expectedResponseMessage: "provisioner daemon key invalid",
		},
		{
			name: "InvalidProvisionerKey_Optional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: true,
			},
			provisionerKey:          "invalid",
			expectedStatusCode:      http.StatusOK,
			expectedResponseMessage: "",
		},
		{
			name: "InvalidProvisionerPSK_NotOptional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: false,
				PSK:      "psk",
			},
			provisionerPSK:          "invalid",
			expectedStatusCode:      http.StatusUnauthorized,
			expectedResponseMessage: "provisioner daemon psk invalid",
		},
		{
			name: "InvalidProvisionerPSK_Optional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: true,
				PSK:      "psk",
			},
			provisionerPSK:          "invalid",
			expectedStatusCode:      http.StatusOK,
			expectedResponseMessage: "",
		},
		{
			name: "ValidProvisionerPSK_NotOptional",
			opts: httpmw.ExtractProvisionerAuthConfig{
				DB:       nil,
				Optional: false,
				PSK:      "ThisIsAValidPSK",
			},
			provisionerPSK:          "ThisIsAValidPSK",
			expectedStatusCode:      http.StatusOK,
			expectedResponseMessage: "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			routeCtx := chi.NewRouteContext()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
			res := httptest.NewRecorder()

			if test.provisionerKey != "" {
				r.Header.Set(codersdk.ProvisionerDaemonKey, test.provisionerKey)
			}
			if test.provisionerPSK != "" {
				r.Header.Set(codersdk.ProvisionerDaemonPSK, test.provisionerPSK)
			}

			httpmw.ExtractProvisionerDaemonAuthenticated(test.opts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(res, r)

			require.Equal(t, test.expectedStatusCode, res.Result().StatusCode)
			if test.expectedResponseMessage != "" {
				require.Contains(t, res.Body.String(), test.expectedResponseMessage)
			}
		})
	}

}
