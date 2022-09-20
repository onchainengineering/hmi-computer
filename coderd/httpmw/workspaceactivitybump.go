package httpmw

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/coderd/database"
)

// ActivityBumpWorkspace automatically bumps the workspace's auto-off timer
// if it is set to expire soon.
// It must be ran after ExtractWorkspace.
func ActivityBumpWorkspace(log slog.Logger, db database.Store) func(h http.Handler) http.Handler {
	log = log.Named("activity_bump")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workspace := WorkspaceParam(r)
			log.Debug(r.Context(), "middleware called")
			// We run the bump logic asynchronously since the result doesn't
			// affect the response.
			go func() {
				// We cannot use the Request context since the goroutine
				// may be around after the request terminates.
				// We set a short timeout so if the app is under load, these
				// low priority operations fail first.
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()

				err := db.InTx(func(s database.Store) error {
					build, err := s.GetLatestWorkspaceBuildByWorkspaceID(ctx, workspace.ID)
					log.Debug(ctx, "build", slog.F("build", build))
					if errors.Is(err, sql.ErrNoRows) {
						return nil
					} else if err != nil {
						return xerrors.Errorf("get latest workspace build: %w", err)
					}

					job, err := s.GetProvisionerJobByID(ctx, build.JobID)
					if err != nil {
						return xerrors.Errorf("get provisioner job: %w", err)
					}

					if build.Transition != database.WorkspaceTransitionStart || !job.CompletedAt.Valid {
						return nil
					}

					if build.Deadline.IsZero() {
						// Workspace shutdown is manual
						return nil
					}

					// We sent bumpThreshold slightly under bumpAmount to minimize DB writes.
					const (
						bumpAmount    = time.Hour
						bumpThreshold = time.Hour - (time.Minute * 10)
					)

					if !build.Deadline.Before(time.Now().Add(bumpThreshold)) {
						return nil
					}

					newDeadline := time.Now().Add(bumpAmount)

					if err := s.UpdateWorkspaceBuildByID(ctx, database.UpdateWorkspaceBuildByIDParams{
						ID:               build.ID,
						UpdatedAt:        build.UpdatedAt,
						ProvisionerState: build.ProvisionerState,
						Deadline:         newDeadline,
					}); err != nil {
						return xerrors.Errorf("update workspace build: %w", err)
					}
					return nil
				})

				if err != nil {
					log.Error(ctx, "bump failed", slog.Error(err))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
