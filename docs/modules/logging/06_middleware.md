# 中间件实现

## Request ID 中间件

```go
// internal/pkg/middleware/request_id.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

const (
    HeaderRequestID = "X-Request-ID"
    CtxRequestID    = "request_id"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader(HeaderRequestID)
        if requestID == "" {
            requestID = uuid.New().String()
        }
        c.Set(CtxRequestID, requestID)
        c.Header(HeaderRequestID, requestID)
        c.Next()
    }
}

func GetRequestID(c *gin.Context) string {
    if id, exists := c.Get(CtxRequestID); exists {
        return id.(string)
    }
    return ""
}
```

## 请求日志中间件

```go
// internal/pkg/middleware/request_logger.go
package middleware

import (
    "time"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "your-project/internal/pkg/logger"
)

const SlowRequestThreshold = 500 * time.Millisecond

func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        requestID := GetRequestID(c)

        reqLogger := logger.L().With(zap.String("request_id", requestID))
        logger.SetGinLogger(c, reqLogger)

        c.Next()

        latency := time.Since(start)
        statusCode := c.Writer.Status()

        fields := []zap.Field{
            zap.String("method", c.Request.Method),
            zap.String("path", path),
            zap.Int("status", statusCode),
            zap.Int64("latency_ms", latency.Milliseconds()),
            zap.String("client_ip", c.ClientIP()),
        }

        switch {
        case statusCode >= 500:
            reqLogger.Error("request failed", fields...)
        case statusCode >= 400:
            reqLogger.Warn("request error", fields...)
        case latency > SlowRequestThreshold:
            reqLogger.Warn("slow request", fields...)
        default:
            reqLogger.Info("request completed", fields...)
        }
    }
}
```

## Recovery 中间件

```go
// internal/pkg/middleware/recovery.go
package middleware

import (
    "net/http"
    "runtime/debug"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "your-project/internal/pkg/logger"
)

func Recovery() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                stack := string(debug.Stack())
                logger.ForGin(c).Error("panic recovered",
                    zap.Any("error", err),
                    zap.String("stack", stack),
                )
                c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
                    "error":      "internal server error",
                    "request_id": GetRequestID(c),
                })
            }
        }()
        c.Next()
    }
}
```
