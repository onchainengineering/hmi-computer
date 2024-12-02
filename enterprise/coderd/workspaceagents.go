package coderd

import (
	"context"
	"net/http"

	"github.com/onchainengineering/hmi-computer/v2/coderd/httpapi"
	"github.com/onchainengineering/hmi-computer/v2/codersdk"
)

func (api *API) shouldBlockNonBrowserConnections(rw http.ResponseWriter) bool {
	if api.Entitlements.Enabled(codersdk.FeatureBrowserOnly) {
		httpapi.Write(context.Background(), rw, http.StatusConflict, codersdk.Response{
			Message: "Non-browser connections are disabled for your deployment.",
		})
		return true
	}
	return false
}
