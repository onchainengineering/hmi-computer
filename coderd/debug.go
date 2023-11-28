package coderd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/coder/v2/coderd/healthcheck"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/codersdk"
	"golang.org/x/exp/slices"
	"golang.org/x/xerrors"
)

// @Summary Debug Info Wireguard Coordinator
// @ID debug-info-wireguard-coordinator
// @Security CoderSessionToken
// @Produce text/html
// @Tags Debug
// @Success 200
// @Router /debug/coordinator [get]
func (api *API) debugCoordinator(rw http.ResponseWriter, r *http.Request) {
	(*api.TailnetCoordinator.Load()).ServeHTTPDebug(rw, r)
}

// @Summary Debug Info Tailnet
// @ID debug-info-tailnet
// @Security CoderSessionToken
// @Produce text/html
// @Tags Debug
// @Success 200
// @Router /debug/tailnet [get]
func (api *API) debugTailnet(rw http.ResponseWriter, r *http.Request) {
	api.agentProvider.ServeHTTPDebug(rw, r)
}

// @Summary Debug Info Deployment Health
// @ID debug-info-deployment-health
// @Security CoderSessionToken
// @Produce json
// @Tags Debug
// @Success 200 {object} healthcheck.Report
// @Router /debug/health [get]
// @Param force query boolean false "Force a healthcheck to run"
func (api *API) debugDeploymentHealth(rw http.ResponseWriter, r *http.Request) {
	apiKey := httpmw.APITokenFromRequest(r)
	ctx, cancel := context.WithTimeout(r.Context(), api.Options.HealthcheckTimeout)
	defer cancel()

	// Check if the forced query parameter is set.
	forced := r.URL.Query().Get("force") == "true"

	// Get cached report if it exists and the requester did not force a refresh.
	if !forced {
		if report := api.healthCheckCache.Load(); report != nil {
			if time.Since(report.Time) < api.Options.HealthcheckRefresh {
				formatHealthcheck(ctx, rw, r, report)
				return
			}
		}
	}

	resChan := api.healthCheckGroup.DoChan("", func() (*healthcheck.Report, error) {
		// Create a new context not tied to the request.
		ctx, cancel := context.WithTimeout(context.Background(), api.Options.HealthcheckTimeout)
		defer cancel()

		report := api.HealthcheckFunc(ctx, apiKey)
		api.healthCheckCache.Store(report)
		return report, nil
	})

	select {
	case <-ctx.Done():
		httpapi.Write(ctx, rw, http.StatusNotFound, codersdk.Response{
			Message: "Healthcheck is in progress and did not complete in time. Try again in a few seconds.",
		})
		return
	case res := <-resChan:
		formatHealthcheck(ctx, rw, r, res.Val)
		return
	}
}

func formatHealthcheck(ctx context.Context, rw http.ResponseWriter, r *http.Request, hc *healthcheck.Report) {
	format := r.URL.Query().Get("format")
	switch format {
	case "text":
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprintln(rw, "time:", hc.Time.Format(time.RFC3339))
		_, _ = fmt.Fprintln(rw, "healthy:", hc.Healthy)
		_, _ = fmt.Fprintln(rw, "derp:", hc.DERP.Healthy)
		_, _ = fmt.Fprintln(rw, "access_url:", hc.AccessURL.Healthy)
		_, _ = fmt.Fprintln(rw, "websocket:", hc.Websocket.Healthy)
		_, _ = fmt.Fprintln(rw, "database:", hc.Database.Healthy)

	case "", "json":
		httpapi.WriteIndent(ctx, rw, http.StatusOK, hc)

	default:
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: fmt.Sprintf("Invalid format option %q.", format),
			Detail:  "Allowed values are: \"json\", \"simple\".",
		})
	}
}

// @Summary Get health settings
// @ID get-health-settings
// @Security CoderSessionToken
// @Produce json
// @Tags Debug
// @Success 200 {object} codersdk.HealthSettings
// @Router /debug/health/settings [get]
func (api *API) deploymentHealthSettings(rw http.ResponseWriter, r *http.Request) {
	settingsJSON, err := api.Database.GetHealthSettings(r.Context())
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to fetch health settings.",
			Detail:  err.Error(),
		})
		return
	}

	var settings codersdk.HealthSettings
	err = json.Unmarshal([]byte(settingsJSON), &settings)
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to unmarshal health settings.",
			Detail:  err.Error(),
		})
		return
	}

	if len(settings.DismissedHealthchecks) == 0 {
		settings.DismissedHealthchecks = []string{}
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, settings)
}

// @Summary Update health settings
// @ID update-health-settings
// @Security CoderSessionToken
// @Accept json
// @Produce json
// @Tags Debug
// @Param request body codersdk.UpdateHealthSettings true "Update health settings"
// @Success 200 {object} codersdk.UpdateHealthSettings
// @Router /debug/health/settings [put]
func (api *API) putDeploymentHealthSettings(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !api.Authorize(r, rbac.ActionUpdate, rbac.ResourceDeploymentValues) {
		httpapi.Write(ctx, rw, http.StatusForbidden, codersdk.Response{
			Message: "Insufficient permissions to update health settings.",
		})
		return
	}

	var settings codersdk.HealthSettings
	if !httpapi.Read(ctx, rw, r, &settings) {
		return
	}

	err := validateHealthSettings(settings)
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to validate health settings.",
			Detail:  err.Error(),
		})
		return
	}

	settingsJSON, err := json.Marshal(&settings)
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to marshal health settings.",
			Detail:  err.Error(),
		})
		return
	}

	err = api.Database.UpsertHealthSettings(ctx, string(settingsJSON))
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to update health settings.",
			Detail:  err.Error(),
		})
		return
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, settings)
}

func validateHealthSettings(settings codersdk.HealthSettings) error {
	for _, dismissed := range settings.DismissedHealthchecks {
		ok := slices.Contains(healthcheck.Sections, dismissed)
		if !ok {
			return xerrors.Errorf("unknown healthcheck section: %s", dismissed)
		}
	}
	return nil
}

// For some reason the swagger docs need to be attached to a function.
//
// @Summary Debug Info Websocket Test
// @ID debug-info-websocket-test
// @Security CoderSessionToken
// @Produce json
// @Tags Debug
// @Success 201 {object} codersdk.Response
// @Router /debug/ws [get]
// @x-apidocgen {"skip": true}
func _debugws(http.ResponseWriter, *http.Request) {} //nolint:unused
