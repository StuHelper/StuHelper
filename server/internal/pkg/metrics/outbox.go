package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var OutboxJobFailuresTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "outbox_job_failures_total",
		Help: "Total number of outbox job processing failures",
	},
	[]string{"worker", "job_type", "terminal"},
)

func ObserveOutboxJobFailure(worker, jobType string, terminal bool) {
	OutboxJobFailuresTotal.WithLabelValues(worker, jobType, strconv.FormatBool(terminal)).Inc()
}
