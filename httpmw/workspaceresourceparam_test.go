package httpmw_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/cryptorand"
	"github.com/coder/coder/database"
	"github.com/coder/coder/database/databasefake"
	"github.com/coder/coder/httpmw"
)

func TestWorkspaceResource(t *testing.T) {
	t.Parallel()

	setup := func(db database.Store) (*http.Request, database.WorkspaceHistory) {
		var (
			id, secret = randomAPIKeyParts()
			hashed     = sha256.Sum256([]byte(secret))
		)
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  httpmw.AuthCookie,
			Value: fmt.Sprintf("%s-%s", id, secret),
		})
		userID, err := cryptorand.String(16)
		require.NoError(t, err)
		username, err := cryptorand.String(8)
		require.NoError(t, err)
		user, err := db.InsertUser(r.Context(), database.InsertUserParams{
			ID:             userID,
			Email:          "testaccount@coder.com",
			Name:           "example",
			LoginType:      database.LoginTypeBuiltIn,
			HashedPassword: hashed[:],
			Username:       username,
			CreatedAt:      database.Now(),
			UpdatedAt:      database.Now(),
		})
		require.NoError(t, err)
		_, err = db.InsertAPIKey(r.Context(), database.InsertAPIKeyParams{
			ID:           id,
			UserID:       user.ID,
			HashedSecret: hashed[:],
			LastUsed:     database.Now(),
			ExpiresAt:    database.Now().Add(time.Minute),
		})
		require.NoError(t, err)

		history, err := db.InsertWorkspaceHistory(context.Background(), database.InsertWorkspaceHistoryParams{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
		})
		require.NoError(t, err)
		workspace, err := db.InsertWorkspace(context.Background(), database.InsertWorkspaceParams{
			ID:      history.WorkspaceID,
			OwnerID: user.ID,
			Name:    "bananas",
		})
		require.NoError(t, err)

		ctx := chi.NewRouteContext()
		ctx.URLParams.Add("user", "me")
		ctx.URLParams.Add("workspace", workspace.Name)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
		return r, history
	}

	t.Run("None", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		rtr := chi.NewRouter()
		rtr.Use(
			httpmw.ExtractWorkspaceResource(db),
		)
		rtr.Get("/", nil)
		r, _ := setup(db)
		rw := httptest.NewRecorder()
		rtr.ServeHTTP(rw, r)

		res := rw.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("BadUUID", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		rtr := chi.NewRouter()
		rtr.Use(
			httpmw.ExtractWorkspaceResource(db),
		)
		rtr.Get("/", nil)
		r, _ := setup(db)
		chi.RouteContext(r.Context()).URLParams.Add("workspaceresource", "bad")
		rw := httptest.NewRecorder()
		rtr.ServeHTTP(rw, r)

		res := rw.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		rtr := chi.NewRouter()
		rtr.Use(
			httpmw.ExtractWorkspaceResource(db),
		)
		rtr.Get("/", nil)
		r, _ := setup(db)
		chi.RouteContext(r.Context()).URLParams.Add("workspaceresource", "bad")
		rw := httptest.NewRecorder()
		rtr.ServeHTTP(rw, r)

		res := rw.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("Found", func(t *testing.T) {
		t.Parallel()
		db := databasefake.New()
		r, history := setup(db)
		resource, err := db.InsertWorkspaceResource(context.Background(), database.InsertWorkspaceResourceParams{
			ID:                 uuid.New(),
			WorkspaceHistoryID: history.ID,
		})
		require.NoError(t, err)
		rtr := chi.NewRouter()
		rtr.Use(
			httpmw.ExtractAPIKey(db, nil),
			httpmw.ExtractUserParam(db),
			httpmw.ExtractWorkspaceParam(db),
			httpmw.ExtractWorkspaceResource(db),
		)
		rtr.Get("/", func(rw http.ResponseWriter, r *http.Request) {
			_ = httpmw.WorkspaceResource(r)
			rw.WriteHeader(http.StatusOK)
		})
		chi.RouteContext(r.Context()).URLParams.Add("workspaceresource", resource.ID.String())
		rw := httptest.NewRecorder()
		rtr.ServeHTTP(rw, r)

		res := rw.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	})
}
