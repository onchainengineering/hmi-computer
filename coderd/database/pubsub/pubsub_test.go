package pubsub_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/database/postgres"
	"github.com/coder/coder/v2/coderd/database/pubsub"
	"github.com/coder/coder/v2/testutil"
)

func TestPGPubsub_Metrics(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}

	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	connectionURL, closePg, err := postgres.Open()
	require.NoError(t, err)
	defer closePg()
	db, err := sql.Open("postgres", connectionURL)
	require.NoError(t, err)
	defer db.Close()
	registry := prometheus.NewRegistry()
	ctx := testutil.Context(t, testutil.WaitLong)

	uut, err := pubsub.New(ctx, logger, db, connectionURL)
	require.NoError(t, err)
	defer uut.Close()

	err = registry.Register(uut)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)
	require.True(t, gaugeHasValue(t, metrics, 0, "coder_pubsub_current_events"))
	require.True(t, gaugeHasValue(t, metrics, 0, "coder_pubsub_current_subscribers"))

	event := "test"
	data := "testing"
	messageChannel := make(chan []byte)
	unsub0, err := uut.Subscribe(event, func(ctx context.Context, message []byte) {
		messageChannel <- message
	})
	require.NoError(t, err)
	defer unsub0()
	go func() {
		err = uut.Publish(event, []byte(data))
		assert.NoError(t, err)
	}()
	_ = testutil.RequireRecvCtx(ctx, t, messageChannel)

	require.Eventually(t, func() bool {
		metrics, err = registry.Gather()
		assert.NoError(t, err)
		return gaugeHasValue(t, metrics, 1, "coder_pubsub_current_events") &&
			gaugeHasValue(t, metrics, 1, "coder_pubsub_current_subscribers") &&
			gaugeHasValue(t, metrics, 1, "coder_pubsub_connected") &&
			counterHasValue(t, metrics, 1, "coder_pubsub_publishes_total", "true") &&
			counterHasValue(t, metrics, 1, "coder_pubsub_subscribes_total", "true") &&
			counterHasValue(t, metrics, 1, "coder_pubsub_messages_total", "normal") &&
			counterHasValue(t, metrics, 7, "coder_pubsub_received_bytes_total") &&
			counterHasValue(t, metrics, 7, "coder_pubsub_published_bytes_total")
	}, testutil.WaitShort, testutil.IntervalFast)

	colossalData := make([]byte, 7600)
	for i := range colossalData {
		colossalData[i] = 'q'
	}
	unsub1, err := uut.Subscribe(event, func(ctx context.Context, message []byte) {
		messageChannel <- message
	})
	require.NoError(t, err)
	defer unsub1()
	go func() {
		err = uut.Publish(event, colossalData)
		assert.NoError(t, err)
	}()
	// should get 2 messages because we have 2 subs
	_ = testutil.RequireRecvCtx(ctx, t, messageChannel)
	_ = testutil.RequireRecvCtx(ctx, t, messageChannel)

	require.Eventually(t, func() bool {
		metrics, err = registry.Gather()
		assert.NoError(t, err)
		return gaugeHasValue(t, metrics, 1, "coder_pubsub_current_events") &&
			gaugeHasValue(t, metrics, 2, "coder_pubsub_current_subscribers") &&
			gaugeHasValue(t, metrics, 1, "coder_pubsub_connected") &&
			counterHasValue(t, metrics, 2, "coder_pubsub_publishes_total", "true") &&
			counterHasValue(t, metrics, 2, "coder_pubsub_subscribes_total", "true") &&
			counterHasValue(t, metrics, 1, "coder_pubsub_messages_total", "normal") &&
			counterHasValue(t, metrics, 1, "coder_pubsub_messages_total", "colossal") &&
			counterHasValue(t, metrics, 7607, "coder_pubsub_received_bytes_total") &&
			counterHasValue(t, metrics, 7607, "coder_pubsub_published_bytes_total")
	}, testutil.WaitShort, testutil.IntervalFast)
}

func gaugeHasValue(t testing.TB, metrics []*dto.MetricFamily, value float64, name string, label ...string) bool {
	t.Helper()
	for _, family := range metrics {
		if family.GetName() != name {
			continue
		}
		ms := family.GetMetric()
		for _, m := range ms {
			require.Equal(t, len(label), len(m.GetLabel()))
			for i, lv := range label {
				if lv != m.GetLabel()[i].GetValue() {
					continue
				}
			}
			return value == m.GetGauge().GetValue()
		}
	}
	return false
}

func counterHasValue(t testing.TB, metrics []*dto.MetricFamily, value float64, name string, label ...string) bool {
	t.Helper()
	for _, family := range metrics {
		if family.GetName() != name {
			continue
		}
		ms := family.GetMetric()
		for _, m := range ms {
			require.Equal(t, len(label), len(m.GetLabel()))
			for i, lv := range label {
				if lv != m.GetLabel()[i].GetValue() {
					continue
				}
			}
			return value == m.GetCounter().GetValue()
		}
	}
	return false
}
