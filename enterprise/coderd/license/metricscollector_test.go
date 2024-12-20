package license_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/onchainengineering/hmi-computer/v2/coderd/entitlements"
	"github.com/onchainengineering/hmi-computer/v2/codersdk"
	"github.com/onchainengineering/hmi-computer/v2/enterprise/coderd/license"
)

func TestCollectLicenseMetrics(t *testing.T) {
	t.Parallel()

	// Given
	registry := prometheus.NewRegistry()

	var sut license.MetricsCollector

	const (
		actualUsers = 4
		userLimit   = 7
	)
	sut.Entitlements = entitlements.New()
	sut.Entitlements.Modify(func(entitlements *codersdk.Entitlements) {
		entitlements.Features[codersdk.FeatureUserLimit] = codersdk.Feature{
			Enabled: true,
			Actual:  ptr.Int64(actualUsers),
			Limit:   ptr.Int64(userLimit),
		}
	})

	registry.Register(&sut)

	// When
	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Then
	goldenFile, err := os.ReadFile("testdata/license-metrics.json")
	require.NoError(t, err)
	golden := map[string]int{}
	err = json.Unmarshal(goldenFile, &golden)
	require.NoError(t, err)

	collected := map[string]int{}
	for _, metric := range metrics {
		switch metric.GetName() {
		case "coderd_license_active_users", "coderd_license_limit_users", "coderd_license_user_limit_enabled":
			for _, m := range metric.Metric {
				collected[metric.GetName()] = int(m.Gauge.GetValue())
			}
		default:
			require.FailNowf(t, "unexpected metric collected", "metric: %s", metric.GetName())
		}
	}
	require.EqualValues(t, golden, collected)
}
