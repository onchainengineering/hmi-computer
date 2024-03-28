package healthcheck_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coder/coder/v2/coderd/healthcheck"
	"github.com/coder/coder/v2/coderd/healthcheck/derphealth"
	"github.com/coder/coder/v2/coderd/healthcheck/health"
	"github.com/coder/coder/v2/codersdk/healthsdk"
)

type testChecker struct {
	DERPReport               healthsdk.DERPHealthReport
	AccessURLReport          healthsdk.AccessURLReport
	WebsocketReport          healthsdk.WebsocketReport
	DatabaseReport           healthsdk.DatabaseReport
	WorkspaceProxyReport     healthsdk.WorkspaceProxyReport
	ProvisionerDaemonsReport healthsdk.ProvisionerDaemonsReport
}

func (c *testChecker) DERP(context.Context, *derphealth.ReportOptions) healthsdk.DERPHealthReport {
	return c.DERPReport
}

func (c *testChecker) AccessURL(context.Context, *healthcheck.AccessURLReportOptions) healthsdk.AccessURLReport {
	return c.AccessURLReport
}

func (c *testChecker) Websocket(context.Context, *healthcheck.WebsocketReportOptions) healthsdk.WebsocketReport {
	return c.WebsocketReport
}

func (c *testChecker) Database(context.Context, *healthcheck.DatabaseReportOptions) healthsdk.DatabaseReport {
	return c.DatabaseReport
}

func (c *testChecker) WorkspaceProxy(context.Context, *healthcheck.WorkspaceProxyReportOptions) healthsdk.WorkspaceProxyReport {
	return c.WorkspaceProxyReport
}

func (c *testChecker) ProvisionerDaemons(context.Context, *healthcheck.ProvisionerDaemonsReportDeps) healthsdk.ProvisionerDaemonsReport {
	return c.ProvisionerDaemonsReport
}

func TestHealthcheck(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name            string
		checker         *testChecker
		healthy         bool
		severity        health.Severity
		failingSections []healthsdk.HealthSection
	}{{
		name: "OK",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		healthy:         true,
		severity:        health.SeverityOK,
		failingSections: []healthsdk.HealthSection{},
	}, {
		name: "DERPFail",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		healthy:         false,
		severity:        health.SeverityError,
		failingSections: []healthsdk.HealthSection{healthsdk.HealthSectionDERP},
	}, {
		name: "DERPWarning",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Warnings: []health.Message{{Message: "foobar", Code: "EFOOBAR"}},
				Severity: health.SeverityWarning,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		healthy:         true,
		severity:        health.SeverityWarning,
		failingSections: []healthsdk.HealthSection{},
	}, {
		name: "AccessURLFail",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  false,
				Severity: health.SeverityWarning,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		healthy:         false,
		severity:        health.SeverityWarning,
		failingSections: []healthsdk.HealthSection{healthsdk.HealthSectionAccessURL},
	}, {
		name: "WebsocketFail",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		healthy:         false,
		severity:        health.SeverityError,
		failingSections: []healthsdk.HealthSection{healthsdk.HealthSectionWebsocket},
	}, {
		name: "DatabaseFail",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		healthy:         false,
		severity:        health.SeverityError,
		failingSections: []healthsdk.HealthSection{healthsdk.HealthSectionDatabase},
	}, {
		name: "ProxyFail",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		severity:        health.SeverityError,
		healthy:         false,
		failingSections: []healthsdk.HealthSection{healthsdk.HealthSectionWorkspaceProxy},
	}, {
		name: "ProxyWarn",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Warnings: []health.Message{{Message: "foobar", Code: "EFOOBAR"}},
				Severity: health.SeverityWarning,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityOK,
			},
		},
		severity:        health.SeverityWarning,
		healthy:         true,
		failingSections: []healthsdk.HealthSection{},
	}, {
		name: "ProvisionerDaemonsFail",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityError,
			},
		},
		severity:        health.SeverityError,
		healthy:         false,
		failingSections: []healthsdk.HealthSection{healthsdk.HealthSectionProvisionerDaemons},
	}, {
		name: "ProvisionerDaemonsWarn",
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  true,
				Severity: health.SeverityOK,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityWarning,
				Warnings: []health.Message{{Message: "foobar", Code: "EFOOBAR"}},
			},
		},
		severity:        health.SeverityWarning,
		healthy:         true,
		failingSections: []healthsdk.HealthSection{},
	}, {
		name:    "AllFail",
		healthy: false,
		checker: &testChecker{
			DERPReport: healthsdk.DERPHealthReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			AccessURLReport: healthsdk.AccessURLReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			WebsocketReport: healthsdk.WebsocketReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			DatabaseReport: healthsdk.DatabaseReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			WorkspaceProxyReport: healthsdk.WorkspaceProxyReport{
				Healthy:  false,
				Severity: health.SeverityError,
			},
			ProvisionerDaemonsReport: healthsdk.ProvisionerDaemonsReport{
				Severity: health.SeverityError,
			},
		},
		severity: health.SeverityError,
		failingSections: []healthsdk.HealthSection{
			healthsdk.HealthSectionDERP,
			healthsdk.HealthSectionAccessURL,
			healthsdk.HealthSectionWebsocket,
			healthsdk.HealthSectionDatabase,
			healthsdk.HealthSectionWorkspaceProxy,
			healthsdk.HealthSectionProvisionerDaemons,
		},
	}} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			report := healthcheck.Run(context.Background(), &healthcheck.ReportOptions{
				Checker: c.checker,
			})

			assert.Equal(t, c.healthy, report.Healthy)
			assert.Equal(t, c.severity, report.Severity)
			assert.Equal(t, c.failingSections, report.FailingSections)
			assert.Equal(t, c.checker.DERPReport.Healthy, report.DERP.Healthy)
			assert.Equal(t, c.checker.DERPReport.Severity, report.DERP.Severity)
			assert.Equal(t, c.checker.DERPReport.Warnings, report.DERP.Warnings)
			assert.Equal(t, c.checker.AccessURLReport.Healthy, report.AccessURL.Healthy)
			assert.Equal(t, c.checker.AccessURLReport.Severity, report.AccessURL.Severity)
			assert.Equal(t, c.checker.WebsocketReport.Healthy, report.Websocket.Healthy)
			assert.Equal(t, c.checker.WorkspaceProxyReport.Healthy, report.WorkspaceProxy.Healthy)
			assert.Equal(t, c.checker.WorkspaceProxyReport.Warnings, report.WorkspaceProxy.Warnings)
			assert.Equal(t, c.checker.WebsocketReport.Severity, report.Websocket.Severity)
			assert.Equal(t, c.checker.DatabaseReport.Healthy, report.Database.Healthy)
			assert.Equal(t, c.checker.DatabaseReport.Severity, report.Database.Severity)
			assert.NotZero(t, report.Time)
			assert.NotZero(t, report.CoderVersion)
		})
	}
}
