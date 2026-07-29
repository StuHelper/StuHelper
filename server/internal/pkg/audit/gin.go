package audit

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

// EventFromGin 用当前 gin 上下文补齐审计日志中的公共请求字段。
func EventFromGin(c *gin.Context, event Event) Event {
	event = eventWithContextFields(requestContextFromGin(c), event)
	if c == nil {
		return normalizeEvent(event)
	}
	if userID := middleware.GetUserID(c); userID != "" {
		event.UserID = userID
	}
	if username := middleware.GetUsername(c); username != "" {
		event.Username = username
	}
	if c.Request != nil {
		event.IP = c.ClientIP()
		event.UserAgent = httputil.TruncateUserAgent(c.GetHeader("User-Agent"))
	}
	if requestID := middleware.GetRequestID(c); requestID != "" {
		event.RequestID = requestID
	}
	if event.ActorType == "" {
		event.ActorType = inferActorType(c)
	}
	return normalizeEvent(event)
}

// LogFromGin 使用 gin 上下文中的公共字段记录审计日志。
func LogFromGin(c *gin.Context, event Event) {
	LogContext(requestContextFromGin(c), EventFromGin(c, event))
}

func requestContextFromGin(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return nil
	}
	return c.Request.Context()
}

func inferActorType(c *gin.Context) string {
	if c == nil {
		return "system"
	}
	for _, capName := range capability.AdminEntryCapabilities {
		if middleware.HasCapability(c, capName) {
			return "admin"
		}
	}
	if middleware.GetUserID(c) != "" {
		return "user"
	}
	return "system"
}
