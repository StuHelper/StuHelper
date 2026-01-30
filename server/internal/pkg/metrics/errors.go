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
)
