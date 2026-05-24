package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	CacheBackendRedis     = "redis"
	CacheNamespaceGeneric = "generic"
	CacheNamespaceCourse  = "course"
	CacheNamespaceReview  = "review"
)

var (
	// CacheHitsTotal 缓存命中总数
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"backend", "namespace"},
	)

	// CacheMissesTotal 缓存未命中总数
	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"backend", "namespace"},
	)

	// CacheInvalidationFailuresTotal 缓存失效失败总数（用于监控告警）
	CacheInvalidationFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_invalidation_failures_total",
			Help: "Total number of cache invalidation failures",
		},
		[]string{"namespace"},
	)

	// CacheOperationDuration 缓存操作延迟
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Cache operation duration in seconds",
			Buckets: []float64{.0005, .001, .005, .01, .025, .05, .1},
		},
		[]string{"operation", "backend", "namespace"},
	)
)

func NormalizeCacheNamespace(namespace string) string {
	switch namespace {
	case CacheNamespaceCourse, CacheNamespaceReview:
		return namespace
	default:
		return CacheNamespaceGeneric
	}
}

func NormalizeCacheBackend(backend string) string {
	switch backend {
	case CacheBackendRedis:
		return backend
	default:
		return "unknown"
	}
}

func ObserveCacheHit(backend, namespace string) {
	CacheHitsTotal.WithLabelValues(
		NormalizeCacheBackend(backend),
		NormalizeCacheNamespace(namespace),
	).Inc()
}

func ObserveCacheMiss(backend, namespace string) {
	CacheMissesTotal.WithLabelValues(
		NormalizeCacheBackend(backend),
		NormalizeCacheNamespace(namespace),
	).Inc()
}

func ObserveCacheOperation(operation, backend, namespace string, seconds float64) {
	CacheOperationDuration.WithLabelValues(
		operation,
		NormalizeCacheBackend(backend),
		NormalizeCacheNamespace(namespace),
	).Observe(seconds)
}

func ObserveCacheInvalidationFailure(namespace string) {
	CacheInvalidationFailuresTotal.WithLabelValues(NormalizeCacheNamespace(namespace)).Inc()
}
