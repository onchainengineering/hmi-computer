package httpmw

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/coder/coder/coderd/httpapi"
)

// parseUUID consumes a url parameter and parses it as a UUID.
func parseUUID(rw http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	rawID := chi.URLParam(r, param)
	if rawID == "" {
		httpapi.Write(rw, r, http.StatusBadRequest, httpapi.Response{
			Message: fmt.Sprintf("%q must be provided", param),
		})
		return uuid.UUID{}, false
	}

	// Automatically set uuid.Nil to the acting users id.
	if param == UserKey && rawID == "me" {
		key := APIKey(r)
		return key.UserID, true
	}

	parsed, err := uuid.Parse(rawID)
	if err != nil {
		httpapi.Write(rw, r, http.StatusBadRequest, httpapi.Response{
			Message: fmt.Sprintf("%q must be a uuid", param),
		})
		return uuid.UUID{}, false
	}

	return parsed, true
}
