package notification

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// Handler 通知 HTTP 处理器
type Handler struct {
	service *Service
	hub     *Hub
}

// NewHandler 创建通知处理器
func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

// RegisterRoutes 注册通知路由。
// 通知域对外统一挂在 /course/review/user/notifications 前缀下，避免再由 review 模块旁路暴露读路径。
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	notif := r.Group("/course/review/user/notifications")
	notif.Use(authMW)
	{
		notif.GET("", h.List)
		notif.GET("/stream", h.Stream)
		notif.GET("/unread-count", h.GetUnreadCount)
		notif.PUT("/:notificationID/read", h.MarkRead)
		notif.PUT("/read-all", h.MarkAllRead)
	}
}

// List 获取通知列表。
func (h *Handler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID, ok := h.mustResolveUserID(c)
	if !ok {
		return
	}

	result, err := h.service.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logger.FromGin(c).Error("failed to load notifications", zap.Error(err))
		response.InternalError(c, "failed to load notifications")
		return
	}

	response.Success(c, gin.H{
		"list":   result.List,
		"total":  result.Total,
		"unread": result.Unread,
	})
}

// GetUnreadCount 获取未读通知数量。
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID, ok := h.mustResolveUserID(c)
	if !ok {
		return
	}

	count, err := h.service.CountUnread(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get unread count", zap.Error(err))
		response.InternalError(c, "failed to get unread count")
		return
	}

	response.Success(c, gin.H{"count": count})
}

// MarkRead 标记单条通知已读。
func (h *Handler) MarkRead(c *gin.Context) {
	notifID, err := httputil.ParseUUIDParam(c, "notificationID")
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	userID, ok := h.mustResolveUserID(c)
	if !ok {
		return
	}

	if err := h.service.MarkRead(c.Request.Context(), notifID, userID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.NotFound(c, "notification not found")
		default:
			logger.FromGin(c).Error("failed to mark notification as read", zap.Error(err))
			response.InternalError(c, "failed to mark notification as read")
		}
		return
	}

	response.Success(c, gin.H{"message": "notification marked as read"})
}

// MarkAllRead 标记当前用户全部通知已读。
func (h *Handler) MarkAllRead(c *gin.Context) {
	userID, ok := h.mustResolveUserID(c)
	if !ok {
		return
	}

	if err := h.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		logger.FromGin(c).Error("failed to mark all notifications as read", zap.Error(err))
		response.InternalError(c, "failed to mark all notifications as read")
		return
	}

	response.Success(c, gin.H{"message": "all notifications marked as read"})
}

// Stream SSE 实时推送端点
func (h *Handler) Stream(c *gin.Context) {
	userID, ok := h.mustResolveUserID(c)
	if !ok {
		return
	}

	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // Nginx 反向代理兼容

	// 订阅 SSE 事件
	ch := h.hub.Subscribe(userID)
	defer h.hub.Unsubscribe(userID, ch)

	// 首次推送未读数
	count, err := h.service.CountUnread(c.Request.Context(), userID)
	if err == nil {
		if writeErr := writeSSE(c.Writer, "unread_count", map[string]int{"count": count}); writeErr != nil {
			logger.FromGin(c).Warn("failed to write initial unread count event", zap.Error(writeErr))
			return
		}
	}

	// 30 秒心跳
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSE(c.Writer, event.Event, event.Data); err != nil {
				logger.FromGin(c).Warn("failed to write notification SSE event", zap.Error(err))
				return
			}
		case <-heartbeat.C:
			// SSE 心跳（注释行不推送数据，仅保持连接）
			if _, err := fmt.Fprintf(c.Writer, ": heartbeat\n\n"); err != nil {
				logger.FromGin(c).Warn("failed to write SSE heartbeat", zap.Error(err))
				return
			}
			c.Writer.Flush()
		}
	}
}

// writeSSE 向 SSE 流写入事件
func writeSSE(w io.Writer, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sanitizeSSEEventName(event), payload); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}

func sanitizeSSEEventName(event string) string {
	event = strings.TrimSpace(event)
	if event == "" {
		return "message"
	}

	var builder strings.Builder
	builder.Grow(len(event))
	lastWasSeparator := false
	for _, r := range event {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastWasSeparator = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
			lastWasSeparator = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastWasSeparator = false
		case r == '_' || r == '-' || r == '.':
			if !lastWasSeparator {
				builder.WriteRune(r)
				lastWasSeparator = true
			}
		default:
			if !lastWasSeparator {
				builder.WriteByte('_')
				lastWasSeparator = true
			}
		}
	}

	sanitized := strings.Trim(builder.String(), "_.-")
	if sanitized == "" {
		return "message"
	}
	return sanitized
}

// mustResolveUserID 从 context 解析 casdoor_subject，并换算为内部 user_id。
func (h *Handler) mustResolveUserID(c *gin.Context) (int64, bool) {
	return middleware.ResolveRequiredInternalUserID(c, h.service.ResolveInternalUserID, "failed to resolve user identity")
}
