package coderd

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/coder/coder/v2/buildinfo"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/db2sdk"
	"github.com/coder/coder/v2/coderd/database/dbtime"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/cryptorand"
	"github.com/coder/coder/v2/enterprise/coderd/identityprovider"
)

func (api *API) oAuth2ProviderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !buildinfo.IsDev() {
			httpapi.Write(r.Context(), rw, http.StatusForbidden, codersdk.Response{
				Message: "OAuth2 provider is under development.",
			})
			return
		}

		api.entitlementsMu.RLock()
		entitled := api.entitlements.Features[codersdk.FeatureOAuth2Provider].Entitlement != codersdk.EntitlementNotEntitled
		api.entitlementsMu.RUnlock()

		if !entitled {
			httpapi.Write(r.Context(), rw, http.StatusForbidden, codersdk.Response{
				Message: "OAuth2 provider is an Enterprise feature. Contact sales!",
			})
			return
		}

		next.ServeHTTP(rw, r)
	})
}

// @Summary Get OAuth2 applications.
// @ID get-oauth2-applications
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Param user_id query string false "Filter by applications authorized for a user"
// @Success 200 {array} codersdk.OAuth2ProviderApp
// @Router /oauth2-provider/apps [get]
func (api *API) oAuth2ProviderApps(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawUserID := r.URL.Query().Get("user_id")
	if rawUserID == "" {
		dbApps, err := api.Database.GetOAuth2ProviderApps(ctx)
		if err != nil {
			httpapi.InternalServerError(rw, err)
			return
		}
		httpapi.Write(ctx, rw, http.StatusOK, db2sdk.OAuth2ProviderApps(api.AccessURL, dbApps))
		return
	}

	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid user UUID",
			Detail:  fmt.Sprintf("queried user_id=%q", userID),
		})
		return
	}

	userApps, err := api.Database.GetOAuth2ProviderAppsByUserID(ctx, userID)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	var dbApps []database.OAuth2ProviderApp
	for _, app := range userApps {
		dbApps = append(dbApps, database.OAuth2ProviderApp{
			ID:          app.OAuth2ProviderApp.ID,
			Name:        app.OAuth2ProviderApp.Name,
			CallbackURL: app.OAuth2ProviderApp.CallbackURL,
			Icon:        app.OAuth2ProviderApp.Icon,
		})
	}
	httpapi.Write(ctx, rw, http.StatusOK, db2sdk.OAuth2ProviderApps(api.AccessURL, dbApps))
}

// @Summary Get OAuth2 application.
// @ID get-oauth2-application
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Param app path string true "App ID"
// @Success 200 {object} codersdk.OAuth2ProviderApp
// @Router /oauth2-provider/apps/{app} [get]
func (api *API) oAuth2ProviderApp(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	app := httpmw.OAuth2ProviderApp(r)
	httpapi.Write(ctx, rw, http.StatusOK, db2sdk.OAuth2ProviderApp(api.AccessURL, app))
}

// @Summary Create OAuth2 application.
// @ID create-oauth2-application
// @Security CoderSessionToken
// @Accept json
// @Produce json
// @Tags Enterprise
// @Param request body codersdk.PostOAuth2ProviderAppRequest true "The OAuth2 application to create."
// @Success 200 {object} codersdk.OAuth2ProviderApp
// @Router /oauth2-provider/apps [post]
func (api *API) postOAuth2ProviderApp(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req codersdk.PostOAuth2ProviderAppRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}
	app, err := api.Database.InsertOAuth2ProviderApp(ctx, database.InsertOAuth2ProviderAppParams{
		ID:          uuid.New(),
		CreatedAt:   dbtime.Now(),
		UpdatedAt:   dbtime.Now(),
		Name:        req.Name,
		Icon:        req.Icon,
		CallbackURL: req.CallbackURL,
	})
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error creating OAuth2 application.",
			Detail:  err.Error(),
		})
		return
	}
	httpapi.Write(ctx, rw, http.StatusCreated, db2sdk.OAuth2ProviderApp(api.AccessURL, app))
}

// @Summary Update OAuth2 application.
// @ID update-oauth2-application
// @Security CoderSessionToken
// @Accept json
// @Produce json
// @Tags Enterprise
// @Param app path string true "App ID"
// @Param request body codersdk.PutOAuth2ProviderAppRequest true "Update an OAuth2 application."
// @Success 200 {object} codersdk.OAuth2ProviderApp
// @Router /oauth2-provider/apps/{app} [put]
func (api *API) putOAuth2ProviderApp(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	app := httpmw.OAuth2ProviderApp(r)
	var req codersdk.PutOAuth2ProviderAppRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}
	app, err := api.Database.UpdateOAuth2ProviderAppByID(ctx, database.UpdateOAuth2ProviderAppByIDParams{
		ID:          app.ID,
		UpdatedAt:   dbtime.Now(),
		Name:        req.Name,
		Icon:        req.Icon,
		CallbackURL: req.CallbackURL,
	})
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error creating OAuth2 application.",
			Detail:  err.Error(),
		})
		return
	}
	httpapi.Write(ctx, rw, http.StatusOK, db2sdk.OAuth2ProviderApp(api.AccessURL, app))
}

// @Summary Delete OAuth2 application.
// @ID delete-oauth2-application
// @Security CoderSessionToken
// @Tags Enterprise
// @Param app path string true "App ID"
// @Success 204
// @Router /oauth2-provider/apps/{app} [delete]
func (api *API) deleteOAuth2ProviderApp(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	app := httpmw.OAuth2ProviderApp(r)
	err := api.Database.DeleteOAuth2ProviderAppByID(ctx, app.ID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error deleting OAuth2 application.",
			Detail:  err.Error(),
		})
		return
	}
	httpapi.Write(ctx, rw, http.StatusNoContent, nil)
}

// @Summary Get OAuth2 application secrets.
// @ID get-oauth2-application-secrets
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Param app path string true "App ID"
// @Success 200 {array} codersdk.OAuth2ProviderAppSecret
// @Router /oauth2-provider/apps/{app}/secrets [get]
func (api *API) oAuth2ProviderAppSecrets(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	app := httpmw.OAuth2ProviderApp(r)
	dbSecrets, err := api.Database.GetOAuth2ProviderAppSecretsByAppID(ctx, app.ID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error getting OAuth2 client secrets.",
			Detail:  err.Error(),
		})
		return
	}
	secrets := []codersdk.OAuth2ProviderAppSecret{}
	for _, secret := range dbSecrets {
		secrets = append(secrets, codersdk.OAuth2ProviderAppSecret{
			ID:                    secret.ID,
			LastUsedAt:            codersdk.NullTime{NullTime: secret.LastUsedAt},
			ClientSecretTruncated: secret.DisplaySecret,
		})
	}
	httpapi.Write(ctx, rw, http.StatusOK, secrets)
}

// @Summary Create OAuth2 application secret.
// @ID create-oauth2-application-secret
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Param app path string true "App ID"
// @Success 200 {array} codersdk.OAuth2ProviderAppSecretFull
// @Router /oauth2-provider/apps/{app}/secrets [post]
func (api *API) postOAuth2ProviderAppSecret(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	app := httpmw.OAuth2ProviderApp(r)
	// 40 characters matches the length of GitHub's client secrets.
	rawSecret, err := cryptorand.String(40)
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to generate OAuth2 client secret.",
		})
		return
	}
	// TODO: Currently unused.
	prefix, _ := cryptorand.String(40)
	hashed := identityprovider.Hash(rawSecret, app.ID)
	secret, err := api.Database.InsertOAuth2ProviderAppSecret(ctx, database.InsertOAuth2ProviderAppSecretParams{
		ID:           uuid.New(),
		CreatedAt:    dbtime.Now(),
		SecretPrefix: []byte(prefix),
		HashedSecret: hashed[:],
		// DisplaySecret is the last six characters of the original unhashed secret.
		// This is done so they can be differentiated and it matches how GitHub
		// displays their client secrets.
		DisplaySecret: rawSecret[len(rawSecret)-6:],
		AppID:         app.ID,
	})
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error creating OAuth2 client secret.",
			Detail:  err.Error(),
		})
		return
	}
	httpapi.Write(ctx, rw, http.StatusOK, codersdk.OAuth2ProviderAppSecretFull{
		ID:               secret.ID,
		ClientSecretFull: rawSecret,
	})
}

// @Summary Delete OAuth2 application secret.
// @ID delete-oauth2-application-secret
// @Security CoderSessionToken
// @Tags Enterprise
// @Param app path string true "App ID"
// @Param secretID path string true "Secret ID"
// @Success 204
// @Router /oauth2-provider/apps/{app}/secrets/{secretID} [delete]
func (api *API) deleteOAuth2ProviderAppSecret(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	secret := httpmw.OAuth2ProviderAppSecret(r)
	err := api.Database.DeleteOAuth2ProviderAppSecretByID(ctx, secret.ID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error deleting OAuth2 client secret.",
			Detail:  err.Error(),
		})
		return
	}
	httpapi.Write(ctx, rw, http.StatusNoContent, nil)
}

// @Summary OAuth2 authorization request.
// @ID oauth2-authorization-request
// @Security CoderSessionToken
// @Tags Enterprise
// @Param client_id query string true "Client ID"
// @Param state query string true "A random unguessable string"
// @Param response_type query codersdk.OAuth2ProviderResponseType true "Response type"
// @Param redirect_uri query string false "Redirect here after authorization"
// @Param scope query string false "Token scopes (currently ignored)"
// @Success 302
// @Router /login/oauth2/authorize [post]
func (api *API) postOAuth2ProviderAppAuthorize() http.HandlerFunc {
	return identityprovider.Authorize(api.Database, api.AccessURL)
}

// @Summary OAuth2 token exchange.
// @ID oauth2-token-exchange
// @Produce json
// @Tags Enterprise
// @Param client_id formData string true "Client ID"
// @Param client_secret formData string true "Client secret"
// @Param code formData string true "Authorization code"
// @Param grant_type formData codersdk.OAuth2ProviderGrantType true "Grant type"
// @Success 200 {object} oauth2.Token
// @Router /login/oauth2/tokens [post]
func (api *API) postOAuth2ProviderAppToken() http.HandlerFunc {
	return identityprovider.Tokens(api.Database, api.DeploymentValues.SessionDuration.Value())
}

// @Summary Delete OAuth2 application tokens.
// @ID delete-oauth2-application-tokens
// @Security CoderSessionToken
// @Tags Enterprise
// @Param client_id query string true "Client ID"
// @Success 204
// @Router /login/oauth2/tokens [delete]
func (api *API) deleteOAuth2ProviderAppTokens() http.HandlerFunc {
	return identityprovider.RevokeApp(api.Database)
}
