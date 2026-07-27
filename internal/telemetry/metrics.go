package telemetry

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsRegistry encapsulates our production observability instruments.
type MetricsRegistry struct {
	RequestCounter *prometheus.CounterVec
	CacheLatency   prometheus.Histogram
	FailOpenEvents *prometheus.CounterVec
}

var (
	Metrics *MetricsRegistry
	once    sync.Once
)

// InitializeMetrics sets up the global Prometheus vector metrics space idempotently.
func InitializeMetrics() {
	once.Do(func() {
		Metrics = &MetricsRegistry{
			RequestCounter: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "chronos_guard_requests_total",
					Help: "Total number of tenant requests processed by the rate limiting sidecar layer.",
				},
				[]string{"tenant_id", "status"},
			),
			CacheLatency: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Name:    "chronos_guard_cache_latency_seconds",
					Help:    "Latency profile of the atomic distributed data store queries.",
					Buckets: []float64{0.001, 0.002, 0.005, 0.010, 0.015, 0.050, 0.100},
				},
			),
			FailOpenEvents: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "chronos_guard_fail_open_total",
					Help: "Total count of emergency fail-open bypass events triggered by datastore timeouts or outages.",
				},
				[]string{"tenant_id", "reason"},
			),
		}
	})
}