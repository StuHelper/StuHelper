package review

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
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
	if !h.authorizeReviewContentEdit(c, reviewID) {
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
		if respondAdminEditReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to edit review", zap.Error(err))
		response.InternalError(c, "failed to edit review")
		return
	}

	h.logAdminOp(c, "edit_content", "review", reviewID, nil, map[string]string{
		"title":  req.Title,
		"reason": req.Reason,
	})
	h.invalidateReviewCaches(c, 0)
	response.Success(c, gin.H{"message": "review content updated successfully"})
}
