package httpmw

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/rbac"
)

// AuthObject wraps the rbac object type for middleware to customize this value
// before being passed to Authorize().
type AuthObject struct {
	// Object is that base static object the above functions can modify.
	Object rbac.Object
}

// Authorize will enforce if the user roles can complete the action on the AuthObject.
// The organization and owner are found using the ExtractOrganization and
// ExtractUser middleware if present.
func Authorize(logger slog.Logger, auth *rbac.RegoAuthorizer, action rbac.Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			roles := UserRoles(r)
			args := GetAuthObject(r)

			object := args.Object
			if object.Type == "" {
				panic("developer error: auth object has no type")
			}

			// First extract the object's owner and organization if present.
			unknownOrg := r.Context().Value(organizationParamContextKey{})
			if organization, castOK := unknownOrg.(database.Organization); unknownOrg != nil {
				if !castOK {
					panic("developer error: organization param middleware not provided for authorize")
				}
				object = object.InOrg(organization.ID)
			}

			unknownOwner := r.Context().Value(userParamContextKey{})
			if owner, castOK := unknownOwner.(database.User); unknownOwner != nil {
				if !castOK {
					panic("developer error: user param middleware not provided for authorize")
				}
				object = object.WithOwner(owner.ID.String())
			}

			err := auth.AuthorizeByRoleName(r.Context(), roles.ID.String(), roles.Roles, action, object)
			if err != nil {
				internalError := new(rbac.UnauthorizedError)
				if xerrors.As(err, internalError) {
					logger = logger.With(slog.F("internal", internalError.Internal()))
				}
				// Log information for debugging. This will be very helpful
				// in the early days if we over secure endpoints.
				logger.Warn(r.Context(), "unauthorized",
					slog.F("roles", roles.Roles),
					slog.F("user_id", roles.ID),
					slog.F("username", roles.Username),
					slog.F("route", r.URL.Path),
					slog.F("action", action),
					slog.F("object", object),
				)
				httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
					Message: err.Error(),
				})
				return
			}
			next.ServeHTTP(rw, r)
		})
	}
}

type authObjectKey struct{}

// APIKey returns the API key from the ExtractAPIKey handler.
func GetAuthObject(r *http.Request) AuthObject {
	obj, ok := r.Context().Value(authObjectKey{}).(AuthObject)
	if !ok {
		panic("developer error: auth object middleware not provided")
	}
	return obj
}

// RBACObject sets the object for 'Authorize()' for all routes handled
// by this middleware. The important field to set is 'Type'
func RBACObject(object rbac.Object) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ao := GetAuthObject(r)
			ao.Object = object

			ctx := context.WithValue(r.Context(), authObjectKey{}, ao)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

// User roles are the 'subject' field of Authorize()
type userRolesKey struct{}

// APIKey returns the API key from the ExtractAPIKey handler.
func UserRoles(r *http.Request) database.GetAllUserRolesRow {
	apiKey, ok := r.Context().Value(userRolesKey{}).(database.GetAllUserRolesRow)
	if !ok {
		panic("developer error: user roles middleware not provided")
	}
	return apiKey
}

// ExtractUserRoles requires authentication using a valid API key.
func ExtractUserRoles(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			apiKey := APIKey(r)
			role, err := db.GetAllUserRoles(r.Context(), apiKey.UserID)
			if err != nil {
				httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
					Message: fmt.Sprintf("roles not found", AuthCookie),
				})
				return
			}

			ctx := context.WithValue(r.Context(), userRolesKey{}, role)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
