package metrics

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var allowedFrontendErrorKinds = map[string]bool{
	"error":              true,
	"unhandledrejection": true,
}

var (
	FrontendErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "frontend_errors_total",
			Help: "Total number of frontend runtime errors reported by browser clients",
		},
		[]string{"kind"},
	)
)

type frontendErrorPayload struct {
	Kind string `json:"kind"`
}

func FrontendErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1024))
		if err != nil || len(body) == 0 {
			c.Status(http.StatusNoContent)
			return
		}

		var p frontendErrorPayload
		if err := json.Unmarshal(body, &p); err != nil {
			c.Status(http.StatusNoContent)
			return
		}

		if !allowedFrontendErrorKinds[p.Kind] {
			c.Status(http.StatusNoContent)
			return
		}

		FrontendErrorsTotal.WithLabelValues(p.Kind).Inc()
		c.Status(http.StatusNoContent)
	}
}
