package review

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// ListFlaggedReviews 获取待复核评课列表（触发了 warn 级敏感词但未被人工复核）
func (h *Handler) ListFlaggedReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	offset := httputil.SafeOffset(page, pageSize)

	list, total, err := h.service.ListFlaggedReviews(c.Request.Context(), pageSize, offset)
	if err != nil {
		logger.FromGin(c).Error("failed to list flagged reviews", zap.Error(err))
		response.InternalError(c, "failed to list flagged reviews")
		return
	}

	response.Success(c, gin.H{"list": list, "total": total})
}

// ClearContentFlag 管理员复核通过，标记 content_flag = cleared
func (h *Handler) ClearContentFlag(c *gin.Context) {
	reviewID := c.Param("reviewID")
	if reviewID == "" {
		response.BadRequest(c, "invalid review id")
		return
	}

	adminUserID := middleware.GetUserID(c)
	if err := h.service.ClearContentFlag(c.Request.Context(), reviewID, adminUserID); err != nil {
		logger.FromGin(c).Error("failed to clear content flag", zap.Error(err))
		response.InternalError(c, "failed to clear content flag")
		return
	}

	h.logAdminOp(c, "clear_content_flag", "review", reviewID, nil, nil)
	response.Success(c, gin.H{"message": "content flag cleared"})
}
