package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CacheHitsTotal 缓存命中总数
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache"},
	)

	// CacheMissesTotal 缓存未命中总数
	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache"},
	)

	// CacheInvalidationFailuresTotal 缓存失效失败总数（用于监控告警）
	CacheInvalidationFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_invalidation_failures_total",
			Help: "Total number of cache invalidation failures",
		},
		[]string{"cache_key"},
	)

	// CacheOperationDuration 缓存操作延迟
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Cache operation duration in seconds",
			Buckets: []float64{.0005, .001, .005, .01, .025, .05, .1},
		},
		[]string{"operation", "cache"},
	)
)
