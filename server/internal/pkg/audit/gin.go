package audit

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// EventFromGin 用当前 gin 上下文补齐审计日志中的公共请求字段。
func EventFromGin(c *gin.Context, event Event) Event {
	event.UserID = middleware.GetUserID(c)
	event.Username = middleware.GetUsername(c)
	event.IP = c.ClientIP()
	event.UserAgent = httputil.TruncateUserAgent(c.GetHeader("User-Agent"))
	event.RequestID = middleware.GetRequestID(c)
	return event
}

// LogFromGin 使用 gin 上下文中的公共字段记录审计日志。
func LogFromGin(c *gin.Context, event Event) {
	Log(EventFromGin(c, event))
}
