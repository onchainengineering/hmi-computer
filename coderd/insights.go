package coderd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/codersdk"
)

// @Summary Get deployment DAUs
// @ID get-deployment-daus
// @Security CoderSessionToken
// @Produce json
// @Tags Insights
// @Success 200 {object} codersdk.DAUsResponse
// @Router /insights/daus [get]
func (api *API) deploymentDAUs(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.Authorize(r, rbac.ActionRead, rbac.ResourceDeploymentValues) {
		httpapi.Forbidden(rw)
		return
	}

	vals := r.URL.Query()
	p := httpapi.NewQueryParamParser()
	tzOffset := p.Int(vals, 0, "tz_offset")
	p.ErrorExcessParams(vals)
	if len(p.Errors) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message:     "Query parameters have invalid values.",
			Validations: p.Errors,
		})
		return
	}

	_, resp, _ := api.metricsCache.DeploymentDAUs(tzOffset)
	if resp == nil || resp.Entries == nil {
		httpapi.Write(ctx, rw, http.StatusOK, &codersdk.DAUsResponse{
			Entries: []codersdk.DAUEntry{},
		})
		return
	}
	httpapi.Write(ctx, rw, http.StatusOK, resp)
}

// @Summary Get insights about user latency
// @ID get-insights-about-user-latency
// @Security CoderSessionToken
// @Produce json
// @Tags Insights
// @Success 200 {object} codersdk.UserLatencyInsightsResponse
// @Router /insights/user-latency [get]
func (api *API) insightsUserLatency(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.Authorize(r, rbac.ActionRead, rbac.ResourceDeploymentValues) {
		httpapi.Forbidden(rw)
		return
	}

	// TODO(mafredri): Client or deployment timezone?
	// Example:
	// - I want data from Monday - Friday
	// - I'm UTC+3 and the deployment is UTC+0
	// - Do we select Monday - Friday in UTC+0 or UTC+3?
	// - Considering users can be in different timezones, perhaps this should be per-user (but we don't keep track of user timezones).
	p := httpapi.NewQueryParamParser().
		Required("start_time").
		Required("end_time")
	vals := r.URL.Query()
	var (
		startTime   = p.Time3339Nano(vals, time.Time{}, "start_time")
		endTime     = p.Time3339Nano(vals, time.Time{}, "end_time")
		templateIDs = p.UUIDs(vals, []uuid.UUID{}, "template_ids")
	)
	p.ErrorExcessParams(vals)
	if len(p.Errors) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message:     "Query parameters have invalid values.",
			Validations: p.Errors,
		})
		return
	}

	if !verifyInsightsStartAndEndTime(ctx, rw, startTime, endTime) {
		return
	}

	// Should we verify all template IDs exist, or just return no rows?
	// _, err := api.Database.GetTemplatesWithFilter(ctx, database.GetTemplatesWithFilterParams{
	// 	IDs: templateIDs,
	// })

	rows, err := api.Database.GetUserLatencyInsights(ctx, database.GetUserLatencyInsightsParams{
		StartTime:   startTime,
		EndTime:     endTime,
		TemplateIDs: templateIDs,
	})
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	// Fetch all users so that we can still include users that have no
	// latency data.
	users, err := api.Database.GetUsers(ctx, database.GetUsersParams{})
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	templateIDSet := make(map[uuid.UUID]struct{})
	usersWithLatencyByID := make(map[uuid.UUID]codersdk.UserLatency)
	for _, row := range rows {
		for _, templateID := range row.TemplateIDs {
			templateIDSet[templateID] = struct{}{}
		}
		usersWithLatencyByID[row.UserID] = codersdk.UserLatency{
			TemplateIDs: row.TemplateIDs,
			UserID:      row.UserID,
			Username:    row.Username,
			LatencyMS: &codersdk.ConnectionLatency{
				P50: row.WorkspaceConnectionLatency50,
				P95: row.WorkspaceConnectionLatency95,
			},
		}
	}
	userLatencies := []codersdk.UserLatency{}
	for _, user := range users {
		userLatency, ok := usersWithLatencyByID[user.ID]
		if !ok {
			// TODO(mafredri): Other cases?
			// We only include deleted/inactive users if they were
			// active as part of the requested timeframe.
			if user.Deleted || user.Status != database.UserStatusActive {
				continue
			}

			userLatency = codersdk.UserLatency{
				TemplateIDs: []uuid.UUID{},
				UserID:      user.ID,
				Username:    user.Username,
			}
		}
		userLatencies = append(userLatencies, userLatency)
	}

	// TemplateIDs that contributed to the data.
	seenTemplateIDs := make([]uuid.UUID, 0, len(templateIDSet))
	for templateID := range templateIDSet {
		seenTemplateIDs = append(seenTemplateIDs, templateID)
	}
	slices.SortFunc(seenTemplateIDs, func(a, b uuid.UUID) bool {
		return a.String() < b.String()
	})

	resp := codersdk.UserLatencyInsightsResponse{
		Report: codersdk.UserLatencyInsightsReport{
			StartTime:   startTime,
			EndTime:     endTime,
			TemplateIDs: seenTemplateIDs,
			Users:       userLatencies,
		},
	}
	httpapi.Write(ctx, rw, http.StatusOK, resp)
}

// @Summary Get insights about templates
// @ID get-insights-about-templates
// @Security CoderSessionToken
// @Produce json
// @Tags Insights
// @Success 200 {object} codersdk.TemplateInsightsResponse
// @Router /insights/templates [get]
func (api *API) insightsTemplates(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.Authorize(r, rbac.ActionRead, rbac.ResourceDeploymentValues) {
		httpapi.Forbidden(rw)
		return
	}

	p := httpapi.NewQueryParamParser().
		Required("start_time").
		Required("end_time")
	vals := r.URL.Query()
	var (
		startTime      = p.Time3339Nano(vals, time.Time{}, "start_time")
		endTime        = p.Time3339Nano(vals, time.Time{}, "end_time")
		intervalString = p.String(vals, string(codersdk.InsightsReportIntervalNone), "interval")
		templateIDs    = p.UUIDs(vals, []uuid.UUID{}, "template_ids")
	)
	p.ErrorExcessParams(vals)
	if len(p.Errors) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message:     "Query parameters have invalid values.",
			Validations: p.Errors,
		})
		return
	}

	if !verifyInsightsStartAndEndTime(ctx, rw, startTime, endTime) {
		return
	}

	// Should we verify all template IDs exist, or just return no rows?
	// _, err := api.Database.GetTemplatesWithFilter(ctx, database.GetTemplatesWithFilterParams{
	// 	IDs: templateIDs,
	// })

	var interval codersdk.InsightsReportInterval
	switch v := codersdk.InsightsReportInterval(intervalString); v {
	case codersdk.InsightsReportIntervalDay, codersdk.InsightsReportIntervalNone:
		interval = v
	default:
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Query parameter has invalid value.",
			Validations: []codersdk.ValidationError{
				{
					Field:  "interval",
					Detail: fmt.Sprintf("must be one of %v", []codersdk.InsightsReportInterval{codersdk.InsightsReportIntervalNone, codersdk.InsightsReportIntervalDay}),
				},
			},
		})
		return
	}

	var usage database.GetTemplateInsightsRow
	var dailyUsage []database.GetTemplateDailyInsightsRow
	// Use a transaction to ensure that we get consistent data between
	// the full and interval report.
	err := api.Database.InTx(func(db database.Store) error {
		var err error

		if interval != codersdk.InsightsReportIntervalNone {
			dailyUsage, err = db.GetTemplateDailyInsights(ctx, database.GetTemplateDailyInsightsParams{
				StartTime:   startTime,
				EndTime:     endTime,
				TemplateIDs: templateIDs,
			})
			if err != nil {
				return xerrors.Errorf("get template daily insights: %w", err)
			}
		}

		usage, err = db.GetTemplateInsights(ctx, database.GetTemplateInsightsParams{
			StartTime:   startTime,
			EndTime:     endTime,
			TemplateIDs: templateIDs,
		})
		if err != nil {
			return xerrors.Errorf("get template insights: %w", err)
		}

		return nil
	}, nil)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	intervalReports := []codersdk.TemplateInsightsIntervalReport{}
	for _, row := range dailyUsage {
		intervalReports = append(intervalReports, codersdk.TemplateInsightsIntervalReport{
			StartTime:   row.StartTime,
			EndTime:     row.EndTime,
			Interval:    interval,
			TemplateIDs: row.TemplateIDs,
			ActiveUsers: row.ActiveUsers,
		})
	}

	resp := codersdk.TemplateInsightsResponse{
		Report: codersdk.TemplateInsightsReport{
			StartTime:   startTime,
			EndTime:     endTime,
			TemplateIDs: usage.TemplateIDs,
			ActiveUsers: usage.ActiveUsers,
			AppsUsage: []codersdk.TemplateAppUsage{
				{
					TemplateIDs: usage.TemplateIDs, // TODO(mafredri): Update query to return template IDs/app?
					Type:        codersdk.TemplateAppsTypeBuiltin,
					DisplayName: "Visual Studio Code",
					Slug:        "vscode",
					Icon:        "/icons/code.svg",
					Seconds:     usage.UsageVscodeSeconds,
				},
				{
					TemplateIDs: usage.TemplateIDs, // TODO(mafredri): Update query to return template IDs/app?
					Type:        codersdk.TemplateAppsTypeBuiltin,
					DisplayName: "JetBrains",
					Slug:        "jetbrains",
					Icon:        "/icons/intellij.svg",
					Seconds:     usage.UsageJetbrainsSeconds,
				},
				{
					TemplateIDs: usage.TemplateIDs, // TODO(mafredri): Update query to return template IDs/app?
					Type:        codersdk.TemplateAppsTypeBuiltin,
					DisplayName: "Web Terminal",
					Slug:        "reconnecting-pty",
					Icon:        "/icons/terminal.svg",
					Seconds:     usage.UsageReconnectingPtySeconds,
				},
				{
					TemplateIDs: usage.TemplateIDs, // TODO(mafredri): Update query to return template IDs/app?
					Type:        codersdk.TemplateAppsTypeBuiltin,
					DisplayName: "SSH",
					Slug:        "ssh",
					Icon:        "/icons/terminal.svg",
					Seconds:     usage.UsageSshSeconds,
				},
			},
		},
		IntervalReports: intervalReports,
	}
	httpapi.Write(ctx, rw, http.StatusOK, resp)
}

func verifyInsightsStartAndEndTime(ctx context.Context, rw http.ResponseWriter, startTime, endTime time.Time) bool {
	for _, v := range []struct {
		name string
		t    time.Time
	}{
		{"start_time", startTime},
		{"end_time", endTime},
	} {
		if v.t.IsZero() {
			httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
				Message: "Query parameter has invalid value.",
				Validations: []codersdk.ValidationError{
					{
						Field:  v.name,
						Detail: "must be not be zero",
					},
				},
			})
			return false
		}
		h, m, s := v.t.Clock()
		if h != 0 || m != 0 || s != 0 {
			httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
				Message: "Query parameter has invalid value.",
				Validations: []codersdk.ValidationError{
					{
						Field:  v.name,
						Detail: "clock must be 00:00:00",
					},
				},
			})
			return false
		}
	}
	if endTime.Before(startTime) {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Query parameter has invalid value.",
			Validations: []codersdk.ValidationError{
				{
					Field:  "end_time",
					Detail: "must be after start_time",
				},
			},
		})
		return false
	}

	return true
}
