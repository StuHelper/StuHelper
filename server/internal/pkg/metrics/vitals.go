package metrics

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// Web Vitals 允许的指标名称白名单，防止标签注入导致高基数
var allowedVitalNames = map[string]bool{
	"CLS":  true,
	"INP":  true,
	"LCP":  true,
	"FCP":  true,
	"TTFB": true,
}

var allowedRatings = map[string]bool{
	"good":              true,
	"needs-improvement": true,
	"poor":              true,
}

var (
	// WebVitalsValue Web Vitals 指标值直方图
	WebVitalsValue = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "web_vitals_value",
			Help:    "Web Vitals metric values reported by browser clients",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"name", "rating"},
	)
)

// vitalsPayload sendBeacon 发送的 JSON 结构
type vitalsPayload struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Rating string  `json:"rating"`
}

// VitalsHandler 接收前端 Web Vitals 上报
func VitalsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// sendBeacon Content-Type 可能是 text/plain，直接读 body 手动解析
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1024))
		if err != nil || len(body) == 0 {
			c.Status(http.StatusNoContent)
			return
		}

		var p vitalsPayload
		if err := json.Unmarshal(body, &p); err != nil {
			c.Status(http.StatusNoContent)
			return
		}

		if !allowedVitalNames[p.Name] || !allowedRatings[p.Rating] {
			response.BadRequest(c, "invalid metric name or rating")
			return
		}

		WebVitalsValue.WithLabelValues(p.Name, p.Rating).Observe(p.Value)
		c.Status(http.StatusNoContent)
	}
}
