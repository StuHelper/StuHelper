package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var IAMDriftReconciliationThresholdExceededTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "iam_drift_reconciliation_threshold_exceeded_total",
		Help: "Total number of IAM drift reconciliation runs that exceeded the automatic repair threshold",
	},
	[]string{"domain"},
)

func ObserveIAMDriftReconciliationThresholdExceeded(domain string) {
	IAMDriftReconciliationThresholdExceededTotal.WithLabelValues(domain).Inc()
}
