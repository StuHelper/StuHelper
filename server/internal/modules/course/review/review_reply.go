package review

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetReplies 获取回复列表
func (h *Handler) GetReplies(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}
	page, pageSize := httputil.ParsePage(c)

	userHash, ok := h.resolveOptionalUserHash(c)
	if !ok {
		return
	}

	result, err := h.service.GetReplies(c.Request.Context(), GetRepliesParams{
		ReviewID: reviewID,
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to load replies", zap.Error(err))
		response.InternalError(c, "failed to load replies")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// CreateReplyRequest 创建回复请求
type CreateReplyRequest struct {
	ParentID *string `json:"parentID" binding:"omitempty"`
	Content  string  `json:"content" binding:"required,min=1,max=1000"`
}

// CreateReply 创建回复
func (h *Handler) CreateReply(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}

	var req CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	result, err := h.service.CreateReply(c.Request.Context(), CreateReplyParams{
		ReviewID: reviewID,
		ParentID: req.ParentID,
		UserHash: userHash,
		Content:  req.Content,
	})
	if err != nil {
		if respondCreateReplyError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to create reply", zap.Error(err))
		response.InternalError(c, "failed to create reply")
		return
	}

	h.invalidateReviewCaches(c, 0)
	response.Created(c, result.Reply)
}

// DeleteReply 删除回复
func (h *Handler) DeleteReply(c *gin.Context) {
	replyID, err := httputil.ParseUUIDParam(c, "replyID")
	if err != nil {
		response.BadRequest(c, "invalid reply id")
		return
	}

	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	err = h.service.DeleteReply(c.Request.Context(), DeleteReplyParams{
		ReplyID:  replyID,
		UserHash: userHash,
	})
	if err != nil {
		if respondDeleteReplyError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to delete reply", zap.Error(err))
		response.InternalError(c, "failed to delete reply")
		return
	}

	h.invalidateReviewCaches(c, 0)
	response.Success(c, gin.H{"message": "reply deleted"})
}
