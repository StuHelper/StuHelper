package review

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetNotifications 获取通知列表
func (h *Handler) GetNotifications(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	result, err := h.service.GetNotifications(c.Request.Context(), GetNotificationsParams{
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
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

// GetUnreadCount 获取未读通知数量
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	count, err := h.service.GetUnreadNotificationCount(c.Request.Context(), userHash)
	if err != nil {
		logger.FromGin(c).Error("failed to get unread count", zap.Error(err))
		response.InternalError(c, "failed to get unread count")
		return
	}

	response.Success(c, gin.H{"count": count})
}

// MarkNotificationRead 标记通知已读
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	notifID, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	err = h.service.MarkNotificationRead(c.Request.Context(), notifID, userHash)
	if err != nil {
		logger.FromGin(c).Error("failed to mark notification as read", zap.Error(err))
		response.InternalError(c, "failed to mark notification as read")
		return
	}

	response.Success(c, gin.H{"message": "notification marked as read"})
}

// MarkAllNotificationsRead 标记所有通知已读
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	err = h.service.MarkAllNotificationsRead(c.Request.Context(), userHash)
	if err != nil {
		logger.FromGin(c).Error("failed to mark all notifications as read", zap.Error(err))
		response.InternalError(c, "failed to mark all notifications as read")
		return
	}

	response.Success(c, gin.H{"message": "all notifications marked as read"})
}
