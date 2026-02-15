package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware 返回 Prometheus 指标收集中间件
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// 必须使用 c.FullPath()（返回路由模板如 /api/v1/courses/:id），
		// 绝不能使用 c.Request.URL.Path（返回实际路径如 /api/v1/courses/123），
		// 否则每个不同的 ID 都会创建新的 Prometheus 时间序列，导致高基数爆炸。
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
