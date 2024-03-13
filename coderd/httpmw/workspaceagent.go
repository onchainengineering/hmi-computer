package httpmw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/codersdk"
)

type workspaceAgentContextKey struct{}

func WorkspaceAgentOptional(r *http.Request) (database.WorkspaceAgent, bool) {
	user, ok := r.Context().Value(workspaceAgentContextKey{}).(database.WorkspaceAgent)
	return user, ok
}

// WorkspaceAgent returns the workspace agent from the ExtractAgent handler.
func WorkspaceAgent(r *http.Request) database.WorkspaceAgent {
	user, ok := WorkspaceAgentOptional(r)
	if !ok {
		panic("developer error: agent middleware not provided or was made optional")
	}
	return user
}

type LatestBuildContextKey struct{}

func LatestBuildOptional(r *http.Request) (database.WorkspaceBuild, bool) {
	user, ok := r.Context().Value(LatestBuildContextKey{}).(database.WorkspaceBuild)
	return user, ok
}

// LatestBuild returns the Latest Build from the ExtractLatestBuild handler.
func LatestBuild(r *http.Request) database.WorkspaceBuild {
	user, ok := LatestBuildOptional(r)
	if !ok {
		panic("developer error: agent middleware not provided or was made optional")
	}
	return user
}

type ExtractWorkspaceAgentAndLatestBuildConfig struct {
	DB database.Store
	// Optional indicates whether the middleware should be optional.  If true, any
	// requests without the a token or with an invalid token will be allowed to
	// continue and no workspace agent will be set on the request context.
	Optional bool
}

// ExtractWorkspaceAgentAndLatestBuild requires authentication using a valid agent token.
func ExtractWorkspaceAgentAndLatestBuild(opts ExtractWorkspaceAgentAndLatestBuildConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// optionalWrite wraps httpapi.Write but runs the next handler if the
			// token is optional.
			//
			// It should be used when the token is not provided or is invalid, but not
			// when there are other errors.
			optionalWrite := func(code int, response codersdk.Response) {
				if opts.Optional {
					next.ServeHTTP(rw, r)
					return
				}
				httpapi.Write(ctx, rw, code, response)
			}

			tokenValue := APITokenFromRequest(r)
			if tokenValue == "" {
				optionalWrite(http.StatusUnauthorized, codersdk.Response{
					Message: fmt.Sprintf("Cookie %q must be provided.", codersdk.SessionTokenCookie),
				})
				return
			}
			token, err := uuid.Parse(tokenValue)
			if err != nil {
				optionalWrite(http.StatusUnauthorized, codersdk.Response{
					Message: "Workspace agent token invalid.",
					Detail:  fmt.Sprintf("An agent token must be a valid UUIDv4. (len %d)", len(tokenValue)),
				})
				return
			}

			//nolint:gocritic // System needs to be able to get workspace agents.
			row, err := opts.DB.GetWorkspaceAgentAndLatestBuildByAuthToken(dbauthz.AsSystemRestricted(ctx), token)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					optionalWrite(http.StatusUnauthorized, codersdk.Response{
						Message: "Workspace agent not authorized.",
						Detail:  "The agent cannot authenticate until the workspace provision job has been completed. If the job is no longer running, this agent is invalid.",
					})
					return
				}

				httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
					Message: "Internal error checking workspace agent authorization.",
					Detail:  err.Error(),
				})
				return
			}

			//nolint:gocritic // System needs to be able to get owner roles.
			roles, err := opts.DB.GetAuthorizationUserRoles(dbauthz.AsSystemRestricted(ctx), row.Workspace.OwnerID)
			if err != nil {
				httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
					Message: "Internal error checking workspace agent authorization.",
					Detail:  err.Error(),
				})
				return
			}

			subject := rbac.Subject{
				ID:     row.Workspace.OwnerID.String(),
				Roles:  rbac.RoleNames(roles.Roles),
				Groups: roles.Groups,
				Scope: rbac.WorkspaceAgentScope(rbac.WorkspaceAgentScopeParams{
					WorkspaceID: row.Workspace.ID,
					OwnerID:     row.Workspace.OwnerID,
					TemplateID:  row.Workspace.TemplateID,
					VersionID:   row.WorkspaceBuildTable.TemplateVersionID,
				}),
			}.WithCachedASTValue()

			ctx = context.WithValue(ctx, workspaceAgentContextKey{}, row.WorkspaceAgent)
			ctx = context.WithValue(ctx, LatestBuildContextKey{}, row.WorkspaceBuildTable)
			// Also set the dbauthz actor for the request.
			ctx = dbauthz.As(ctx, subject)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

func ensureLatestBuild(ctx context.Context, db database.Store, rw http.ResponseWriter, workspaceAgent database.WorkspaceAgent) (database.WorkspaceBuild, bool) {
	resource, err := db.GetWorkspaceResourceByID(ctx, workspaceAgent.ResourceID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Internal error fetching workspace agent resource.",
			Detail:  err.Error(),
		})
		return database.WorkspaceBuild{}, false
	}

	build, err := db.GetWorkspaceBuildByJobID(ctx, resource.JobID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Internal error fetching workspace build job.",
			Detail:  err.Error(),
		})
		return database.WorkspaceBuild{}, false
	}

	// Ensure the resource is still valid!
	// We only accept agents for resources on the latest build.
	err = checkBuildIsLatest(ctx, db, build)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusForbidden, codersdk.Response{
			Message: "Agent trying to connect from non-latest build.",
			Detail:  err.Error(),
		})
		return database.WorkspaceBuild{}, false
	}

	return build, true
}

func checkBuildIsLatest(ctx context.Context, db database.Store, build database.WorkspaceBuild) error {
	latestBuild, err := db.GetLatestWorkspaceBuildByWorkspaceID(ctx, build.WorkspaceID)
	if err != nil {
		return err
	}
	if build.ID != latestBuild.ID {
		return xerrors.New("build is outdated")
	}
	return nil
}
