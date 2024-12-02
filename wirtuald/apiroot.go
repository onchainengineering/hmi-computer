package coderd

import (
	"net/http"

	"github.com/onchainengineering/hmi-computer/v2/coderd/httpapi"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/codersdk"
)

// @Summary API root handler
// @ID api-root-handler
// @Produce json
// @Tags General
// @Success 200 {object} codersdk.Response
// @Router / [get]
func apiRoot(w http.ResponseWriter, r *http.Request) {
	httpapi.Write(r.Context(), w, http.StatusOK, codersdk.Response{
		//nolint:gocritic
		Message: "👋",
	})
}
