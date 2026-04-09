package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ErrorsTotal 错误总数
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "code"},
	)

	// PanicsTotal panic 总数
	PanicsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "panics_total",
			Help: "Total number of panics recovered",
		},
	)

	// SSEDroppedEventsTotal SSE 通道满时丢弃事件计数（监控慢客户端）
	SSEDroppedEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sse_dropped_events_total",
			Help: "Total number of SSE events dropped due to full channel buffer",
		},
	)

	// CryptoRandFailuresTotal crypto/rand 失败并降级为 math/rand 的次数
	CryptoRandFailuresTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "crypto_rand_failures_total",
			Help: "Number of crypto/rand failures fallen back to math/rand",
		},
	)
)
