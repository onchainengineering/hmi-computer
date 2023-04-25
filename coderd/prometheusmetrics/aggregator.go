package prometheusmetrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/xerrors"

	"cdr.dev/slog"

	"github.com/coder/coder/codersdk/agentsdk"
)

const (
	// MetricHelpForAgent is a help string that replaces all agent metric help
	// messages. This is because a registry cannot have conflicting
	// help messages for the same metric in a "gather". If our coder agents are
	// on different versions, this is a possible scenario.
	metricHelpForAgent = "Metric is forwarded from workspace agent connected to this instance of coderd."
)

const (
	sizeCollectCh = 10
	sizeUpdateCh  = 1024

	defaultMetricsCleanupInterval = 2 * time.Minute
)

type MetricsAggregator struct {
	queue []annotatedMetric

	log                    slog.Logger
	metricsCleanupInterval time.Duration

	collectCh chan (chan<- prometheus.Metric)
	updateCh  chan updateRequest
}

type updateRequest struct {
	username      string
	workspaceName string
	agentName     string

	metrics []agentsdk.AgentMetric

	timestamp time.Time
}

type annotatedMetric struct {
	agentsdk.AgentMetric

	username      string
	workspaceName string
	agentName     string

	expiryDate time.Time
}

var _ prometheus.Collector = new(MetricsAggregator)

func NewMetricsAggregator(logger slog.Logger, duration time.Duration) *MetricsAggregator {
	metricsCleanupInterval := defaultMetricsCleanupInterval
	if duration > 0 {
		metricsCleanupInterval = duration
	}
	return &MetricsAggregator{
		log:                    logger,
		metricsCleanupInterval: metricsCleanupInterval,

		collectCh: make(chan (chan<- prometheus.Metric), sizeCollectCh),
		updateCh:  make(chan updateRequest, sizeUpdateCh),
	}
}

func (ma *MetricsAggregator) Run(ctx context.Context) func() {
	ctx, cancelFunc := context.WithCancel(ctx)
	done := make(chan struct{})

	cleanupTicker := time.NewTicker(ma.metricsCleanupInterval)
	go func() {
		defer close(done)
		defer cleanupTicker.Stop()

		for {
			select {
			case req := <-ma.updateCh:
				ma.log.Debug(ctx, "metrics aggregator: update metrics")

			UpdateLoop:
				for _, m := range req.metrics {
					for i, q := range ma.queue {
						if q.username == req.username && q.workspaceName == req.workspaceName && q.agentName == req.agentName && q.Name == m.Name {
							ma.queue[i].AgentMetric.Value = m.Value
							ma.queue[i].expiryDate = req.timestamp.Add(ma.metricsCleanupInterval)
							continue UpdateLoop
						}
					}

					ma.queue = append(ma.queue, annotatedMetric{
						username:      req.username,
						workspaceName: req.workspaceName,
						agentName:     req.agentName,

						AgentMetric: m,

						expiryDate: req.timestamp.Add(ma.metricsCleanupInterval),
					})
				}
			case inputCh := <-ma.collectCh:
				ma.log.Debug(ctx, "metrics aggregator: collect metrics")

				for _, m := range ma.queue {
					desc := prometheus.NewDesc(m.Name, metricHelpForAgent, agentMetricsLabels, nil)
					valueType, err := asPrometheusValueType(m.Type)
					if err != nil {
						ma.log.Error(ctx, "can't convert Prometheus value type", slog.F("name", m.Name), slog.F("type", m.Type), slog.F("value", m.Value), slog.Error(err))
						continue
					}
					constMetric := prometheus.MustNewConstMetric(desc, valueType, m.Value, m.username, m.workspaceName, m.agentName)
					inputCh <- constMetric
				}
				close(inputCh)
			case <-cleanupTicker.C:
				ma.log.Debug(ctx, "metrics aggregator: clean expired metrics")

				now := time.Now()

				var hasExpiredMetrics bool
				for _, m := range ma.queue {
					if m.expiryDate.After(now) {
						hasExpiredMetrics = true
						break
					}
				}

				if !hasExpiredMetrics {
					continue
				}

				var j int
				fresh := make([]annotatedMetric, len(ma.queue))
				for _, m := range ma.queue {
					if m.expiryDate.After(now) {
						fresh[j] = m
						j++
					}
				}
				fresh = fresh[:j]
				ma.queue = fresh
			case <-ctx.Done():
				ma.log.Debug(ctx, "metrics aggregator: is stopped")
				return
			}
		}
	}()
	return func() {
		cancelFunc()
		<-done
	}
}

// Describe function does not have any knowledge about the metrics schema,
// so it does not emit anything.
func (*MetricsAggregator) Describe(_ chan<- *prometheus.Desc) {
}

var agentMetricsLabels = []string{usernameLabel, workspaceNameLabel, agentNameLabel}

func (ma *MetricsAggregator) Collect(ch chan<- prometheus.Metric) {
	collect := make(chan prometheus.Metric, 128)

	select {
	case ma.collectCh <- collect:
	default:
		ma.log.Error(context.Background(), "metrics aggregator: collect queue is full")
		return
	}

	for m := range collect {
		ch <- m
	}
}

func (ma *MetricsAggregator) Update(ctx context.Context, username, workspaceName, agentName string, metrics []agentsdk.AgentMetric) {
	select {
	case ma.updateCh <- updateRequest{
		username:      username,
		workspaceName: workspaceName,
		agentName:     agentName,
		metrics:       metrics,

		timestamp: time.Now(),
	}:
	case <-ctx.Done():
		ma.log.Debug(ctx, "metrics aggregator: update request is canceled")
	default:
		ma.log.Error(ctx, "metrics aggregator: update queue is full")
	}
}

func asPrometheusValueType(metricType agentsdk.AgentMetricType) (prometheus.ValueType, error) {
	switch metricType {
	case agentsdk.AgentMetricTypeGauge:
		return prometheus.GaugeValue, nil
	case agentsdk.AgentMetricTypeCounter:
		return prometheus.CounterValue, nil
	default:
		return -1, xerrors.Errorf("unsupported value type: %s", metricType)
	}
}
