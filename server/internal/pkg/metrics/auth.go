package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var AuthRefreshTokenReuseTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_refresh_token_reuse_total",
		Help: "Total number of refresh token reuse detections",
	},
	[]string{"token_family"},
)

func ObserveRefreshTokenReuse(tokenFamily string) {
	AuthRefreshTokenReuseTotal.WithLabelValues(tokenFamily).Inc()
}
