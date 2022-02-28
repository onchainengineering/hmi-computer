package httpmw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/coder/coder/database"
	"github.com/coder/coder/httpapi"
	"github.com/google/uuid"
)

type workspaceAgentContextKey struct{}

// WorkspaceAgent returns the workspace agent from the ExtractWorkspaceAgent handler.
func WorkspaceAgent(r *http.Request) database.ProvisionerJobAgent {
	user, ok := r.Context().Value(workspaceAgentContextKey{}).(database.ProvisionerJobAgent)
	if !ok {
		panic("developer error: workspace agent middleware not provided")
	}
	return user
}

// ExtractWorkspaceAgent requires authentication using a valid agent token.
func ExtractWorkspaceAgent(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AuthCookie)
			if err != nil {
				httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
					Message: fmt.Sprintf("%q cookie must be provided", AuthCookie),
				})
				return
			}
			token, err := uuid.Parse(cookie.Value)
			if err != nil {
				httpapi.Write(rw, http.StatusBadRequest, httpapi.Response{
					Message: fmt.Sprintf("parse token: %s", err),
				})
				return
			}
			workspaceAgent, err := db.GetProvisionerJobAgentByAuthToken(r.Context(), token)
			if errors.Is(err, sql.ErrNoRows) {
				if errors.Is(err, sql.ErrNoRows) {
					httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
						Message: "agent token is invalid",
					})
					return
				}
			}
			if err != nil {
				httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
					Message: fmt.Sprintf("get workspace agent: %s", err),
				})
				return
			}

			ctx := context.WithValue(r.Context(), workspaceAgentContextKey{}, workspaceAgent)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
