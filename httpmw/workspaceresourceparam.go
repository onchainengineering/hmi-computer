package httpmw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/coder/coder/database"
	"github.com/coder/coder/httpapi"
)

type workspaceResourceParamContextKey struct{}

// ProvisionerJobParam returns the project from the ExtractProjectParam handler.
func WorkspaceResourceParam(r *http.Request) database.WorkspaceResource {
	resource, ok := r.Context().Value(workspaceResourceParamContextKey{}).(database.WorkspaceResource)
	if !ok {
		panic("developer error: workspace resource param middleware not provided")
	}
	return resource
}

// ExtractWorkspaceResourceParam grabs a workspace resource from the "provisionerjob" URL parameter.
func ExtractWorkspaceResourceParam(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			resourceUUID, parsed := parseUUID(rw, r, "workspaceresource")
			if !parsed {
				return
			}
			resource, err := db.GetWorkspaceResourceByID(r.Context(), resourceUUID)
			if errors.Is(err, sql.ErrNoRows) {
				httpapi.Write(rw, http.StatusNotFound, httpapi.Response{
					Message: "resource doesn't exist with that id",
				})
				return
			}
			if err != nil {
				httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
					Message: fmt.Sprintf("get provisioner resource: %s", err),
				})
				return
			}

			job, err := db.GetProvisionerJobByID(r.Context(), resource.JobID)
			if err != nil {
				httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
					Message: fmt.Sprintf("get provisioner job: %s", err),
				})
				return
			}
			if job.Type != database.ProvisionerJobTypeWorkspaceBuild {
				httpapi.Write(rw, http.StatusBadRequest, httpapi.Response{
					Message: "Workspace resources can only be fetched for builds.",
				})
				return
			}
			build, err := db.GetWorkspaceBuildByJobID(r.Context(), job.ID)
			if err != nil {
				httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
					Message: fmt.Sprintf("get workspace build: %s", err),
				})
				return
			}

			ctx := context.WithValue(r.Context(), workspaceResourceParamContextKey{}, resource)
			ctx = context.WithValue(ctx, workspaceBuildParamContextKey{}, build)
			chi.RouteContext(ctx).URLParams.Add("workspace", build.WorkspaceID.String())
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
