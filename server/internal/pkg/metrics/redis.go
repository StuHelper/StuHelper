package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RedisPoolHits 连接池命中次数（复用空闲连接）
	RedisPoolHits = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_hits_total",
			Help: "Number of times a free connection was found in the pool",
		},
	)

	// RedisPoolMisses 连接池未命中次数（需新建连接）
	RedisPoolMisses = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_misses_total",
			Help: "Number of times a free connection was NOT found in the pool",
		},
	)

	// RedisPoolTimeouts 连接池等待超时次数
	RedisPoolTimeouts = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_timeouts_total",
			Help: "Number of times a wait timeout occurred",
		},
	)

	// RedisPoolTotalConns 连接池当前总连接数
	RedisPoolTotalConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_total_conns",
			Help: "Number of total connections in the pool",
		},
	)

	// RedisPoolIdleConns 连接池当前空闲连接数
	RedisPoolIdleConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_idle_conns",
			Help: "Number of idle connections in the pool",
		},
	)

	// RedisPoolStaleConns 连接池过期关闭连接数
	RedisPoolStaleConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_stale_conns_total",
			Help: "Number of stale connections removed from the pool",
		},
	)
)
