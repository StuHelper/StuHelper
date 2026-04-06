package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ExternalRequestDuration 外部依赖调用延迟。
	ExternalRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "external_request_duration_seconds",
			Help:    "Duration of outbound dependency requests in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"client", "operation", "status"},
	)

	// ExternalRequestsTotal 外部依赖调用次数。
	ExternalRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "external_requests_total",
			Help: "Total number of outbound dependency requests",
		},
		[]string{"client", "operation", "status"},
	)
)

// ObserveExternalRequest 记录外部依赖调用结果。
func ObserveExternalRequest(client, operation string, start time.Time, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	ExternalRequestDuration.WithLabelValues(client, operation, status).Observe(time.Since(start).Seconds())
	ExternalRequestsTotal.WithLabelValues(client, operation, status).Inc()
}
