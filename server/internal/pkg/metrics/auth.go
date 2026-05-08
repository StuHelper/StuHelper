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

var AuthBlacklistFailuresTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_blacklist_failures_total",
		Help: "Total number of auth blacklist lookup failures",
	},
	[]string{"reason"},
)

var AuthSessionValidationFailuresTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_session_validation_failures_total",
		Help: "Total number of tracked session validation failures",
	},
	[]string{"reason"},
)

func ObserveRefreshTokenReuse(tokenFamily string) {
	AuthRefreshTokenReuseTotal.WithLabelValues(tokenFamily).Inc()
}

func ObserveAuthBlacklistFailure(reason string) {
	AuthBlacklistFailuresTotal.WithLabelValues(reason).Inc()
}

func ObserveAuthSessionValidationFailure(reason string) {
	AuthSessionValidationFailuresTotal.WithLabelValues(reason).Inc()
}
