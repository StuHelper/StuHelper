package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OpenPlatformDisclosureRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "open_platform_disclosure_requests_total",
			Help: "Total number of Open Platform disclosure requests by endpoint and result",
		},
		[]string{"endpoint", "result"},
	)

	OpenPlatformDisclosureRateLimitTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "open_platform_disclosure_rate_limit_total",
			Help: "Total number of Open Platform disclosure rate-limit decisions by dimension and outcome",
		},
		[]string{"dimension", "outcome"},
	)

	OpenPlatformDisclosureReplayTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "open_platform_disclosure_replay_total",
			Help: "Total number of Open Platform disclosure replay anomaly decisions by endpoint and outcome",
		},
		[]string{"endpoint", "outcome"},
	)
)

func ObserveOpenPlatformDisclosure(endpoint, result string) {
	OpenPlatformDisclosureRequestsTotal.WithLabelValues(endpoint, result).Inc()
}

func ObserveOpenPlatformDisclosureRateLimit(dimension, outcome string) {
	OpenPlatformDisclosureRateLimitTotal.WithLabelValues(dimension, outcome).Inc()
}

func ObserveOpenPlatformDisclosureReplay(endpoint, outcome string) {
	OpenPlatformDisclosureReplayTotal.WithLabelValues(endpoint, outcome).Inc()
}
