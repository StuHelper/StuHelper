package metrics

import (
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// DBTableUnknown is used when a repository has not provided explicit table
	// metadata or the provided value is not a stable metric label.
	DBTableUnknown = "unknown"
)

var dbTableLabelPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)?$`)

var (
	// DBQueryDuration 数据库查询延迟直方图
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"operation", "table"},
	)

	// DBQueryTotal 数据库查询总数
	DBQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table", "status"},
	)

	// DBConnectionsActive 活跃数据库连接数
	DBConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)
)

// NormalizeDBTable keeps DB table labels stable and intentionally refuses to
// derive labels from SQL fragments. Callers should pass explicit repository
// metadata such as "users" or "open_platform_apps".
func NormalizeDBTable(table string) string {
	table = strings.ToLower(strings.TrimSpace(table))
	if table == "" || !dbTableLabelPattern.MatchString(table) {
		return DBTableUnknown
	}
	return table
}

// ObserveDBQueryDuration records database query latency with normalized labels.
func ObserveDBQueryDuration(operation, table string, seconds float64) {
	DBQueryDuration.WithLabelValues(operation, NormalizeDBTable(table)).Observe(seconds)
}

// ObserveDBQueryTotal records database query totals with normalized table labels.
func ObserveDBQueryTotal(operation, table, status string) {
	DBQueryTotal.WithLabelValues(operation, NormalizeDBTable(table), status).Inc()
}
