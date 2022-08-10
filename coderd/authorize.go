package coderd

import (
	"net/http"

	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/rbac"
)

func AuthorizeFilter[O rbac.Objecter](api *API, r *http.Request, action rbac.Action, objects []O) ([]O, error) {
	roles := httpmw.AuthorizationUserRoles(r)

	if len(objects) == 0 {
		return objects, nil
	}
	objectType := objects[0].RBACObject().Type
	objects, err := rbac.Filter(r.Context(), api.Authorizer, roles.ID.String(), roles.Roles, action, objectType, objects)
	if err != nil {
		api.Logger.Error(r.Context(), "filter failed",
			slog.Error(err),
			slog.F("object_type", objectType),
			slog.F("user_id", roles.ID),
			slog.F("username", roles.Username),
			slog.F("route", r.URL.Path),
			slog.F("action", action),
		)
		// Hide the underlying error in case it has sensitive information
		return nil, xerrors.Errorf("failed to filter requested objects")
	}
	return objects, nil
}

// Authorize will return false if the user is not authorized to do the action.
// This function will log appropriately, but the caller must return an
// error to the api client.
// Eg:
//	if !api.Authorize(...) {
//		httpapi.Forbidden(rw)
//		return
//	}
func (api *API) Authorize(r *http.Request, action rbac.Action, object rbac.Objecter) bool {
	roles := httpmw.AuthorizationUserRoles(r)
	err := api.Authorizer.ByRoleName(r.Context(), roles.ID.String(), roles.Roles, action, object.RBACObject())
	if err != nil {
		// Log the errors for debugging
		internalError := new(rbac.UnauthorizedError)
		logger := api.Logger
		if xerrors.As(err, internalError) {
			logger = api.Logger.With(slog.F("internal", internalError.Internal()))
		}
		// Log information for debugging. This will be very helpful
		// in the early days
		logger.Warn(r.Context(), "unauthorized",
			slog.F("roles", roles.Roles),
			slog.F("user_id", roles.ID),
			slog.F("username", roles.Username),
			slog.F("route", r.URL.Path),
			slog.F("action", action),
			slog.F("object", object),
		)

		return false
	}
	return true
}
