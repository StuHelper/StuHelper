package review

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// AdminEditReviewRequest 管理员编辑评论内容请求
type AdminEditReviewRequest struct {
	Title   string `json:"title" binding:"max=200"`
	Content string `json:"content" binding:"required,min=1,max=5000"`
	Reason  string `json:"reason" binding:"max=500"`
}

// AdminEditReviewContent 管理员编辑评论内容
func (h *Handler) AdminEditReviewContent(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review ID")
		return
	}

	var req AdminEditReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	userID := middleware.GetUserID(c)
	err = h.service.AdminEditReview(c.Request.Context(), AdminEditReviewParams{
		ReviewID: reviewID,
		Title:    req.Title,
		Content:  req.Content,
		Reason:   req.Reason,
		AdminID:  userID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found", errs.ErrReviewNotFound)
		case errors.Is(err, ErrTitleEmpty):
			response.BadRequest(c, "title cannot be empty")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		case errors.Is(err, ErrSensitiveContent):
			response.BadRequest(c, "content contains sensitive words", errs.ErrSensitiveContent)
		case errors.Is(err, ErrContentEmpty):
			response.BadRequest(c, "content cannot be empty", errs.ErrContentEmpty)
		default:
			logger.FromGin(c).Error("failed to edit review", zap.Error(err))
			response.InternalError(c, "failed to edit review")
		}
		return
	}

	h.logAdminOp(c, "edit_content", "review", reviewID, nil, map[string]string{
		"title":  req.Title,
		"reason": req.Reason,
	})
	h.invalidateReviewCaches(c, 0)
	response.Success(c, gin.H{"message": "review content updated successfully"})
}
