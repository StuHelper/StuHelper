package logger

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const ctxLoggerKey = "zap_logger"

// GinContext 将 Logger 注入到 Gin Context
func GinContext(c *gin.Context, l *zap.Logger) {
	c.Set(ctxLoggerKey, l)
}

// FromGin 从 Gin Context 提取 Logger
func FromGin(c *gin.Context) *zap.Logger {
	if l, exists := c.Get(ctxLoggerKey); exists {
		if logger, ok := l.(*zap.Logger); ok {
			return logger
		}
	}
	return L()
}
