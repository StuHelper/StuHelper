package review

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// PostReviewRequest 发布测评请求
type PostReviewRequest struct {
	CourseID  int64         `json:"courseID" binding:"required,gt=0"`
	TeacherID *int64        `json:"teacherID" binding:"omitempty,gt=0"`
	TermID    string        `json:"termID" binding:"required,max=20"`
	Title     string        `json:"title" binding:"required,max=200"`
	Content   string        `json:"content" binding:"required,min=10,max=5000"`
	Grade     string        `json:"grade" binding:"omitempty,oneof=A+ A A- B+ B B- C+ C C- D F"`
	Ratings   ReviewRatings `json:"ratings" binding:"required"`
}

// PostReview 发布测评
func (h *Handler) PostReview(c *gin.Context) {
	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	if !facts.CanPostReview {
		response.Forbidden(c, "student and identity verification are required to post reviews", errs.ErrAccessDenied)
		return
	}

	var req PostReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	userID, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	requestID, _ := c.Get(middleware.CtxKeyRequestID)
	requestIDStr, _ := requestID.(string) //nolint:errcheck // gin context value, empty string is safe fallback

	result, err := h.service.PostReview(c.Request.Context(), PostReviewParams{
		CourseID:             req.CourseID,
		TeacherID:            req.TeacherID,
		TermID:               req.TermID,
		Title:                req.Title,
		Content:              req.Content,
		Grade:                req.Grade,
		Ratings:              req.Ratings,
		UserHash:             userHash,
		AuthorExternalUserID: userID,
		IPAddress:            c.ClientIP(),
		RequestID:            requestIDStr,
	})
	if err != nil {
		if respondPostReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to create review", zap.Error(err))
		response.InternalError(c, "failed to create review")
		return
	}

	h.invalidateReviewAggregateCaches(c)

	response.Created(c, result.Review)
}

// VoteReview 投票
func (h *Handler) VoteReview(c *gin.Context) {
	var req struct {
		VoteType string `json:"voteType" binding:"required,oneof=like dislike"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}
	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	courseID, err := h.service.VoteReview(c.Request.Context(), VoteReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
		VoteType: req.VoteType,
	})
	if err != nil {
		if respondVoteReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to vote", zap.Error(err))
		response.InternalError(c, "failed to vote")
		return
	}

	h.invalidateReviewCaches(c, courseID)
	response.Success(c, gin.H{"message": "vote submitted successfully"})
}

// UpdateReviewRequest 更新评论请求
type UpdateReviewRequest struct {
	Title   *string        `json:"title" binding:"omitempty,max=200"`
	Content *string        `json:"content" binding:"omitempty,min=10,max=5000"`
	Grade   *string        `json:"grade" binding:"omitempty,oneof=A+ A A- B+ B B- C+ C C- D F"`
	Ratings *ReviewRatings `json:"ratings"`
}

// UpdateReview 更新评论
func (h *Handler) UpdateReview(c *gin.Context) {
	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	if !facts.CanEditOwn {
		response.Forbidden(c, "verified student capability is required to edit your own review", errs.ErrAccessDenied)
		return
	}

	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	err = h.service.UpdateReview(c.Request.Context(), UpdateReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
		Title:    req.Title,
		Content:  req.Content,
		Grade:    req.Grade,
		Ratings:  req.Ratings,
	})
	if err != nil {
		if respondUpdateReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to update review", zap.Error(err))
		response.InternalError(c, "failed to update review")
		return
	}

	h.invalidateReviewAggregateCaches(c)
	response.Success(c, gin.H{"message": "review updated successfully"})
}

// DeleteReview 删除评论
func (h *Handler) DeleteReview(c *gin.Context) {
	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	if !facts.CanDeleteOwn {
		response.Forbidden(c, "verified student capability is required to delete your own review", errs.ErrAccessDenied)
		return
	}

	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}
	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	err = h.service.DeleteReview(c.Request.Context(), DeleteReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
	})
	if err != nil {
		if respondDeleteReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to delete review", zap.Error(err))
		response.InternalError(c, "failed to delete review")
		return
	}

	h.invalidateReviewAggregateCaches(c)
	response.Success(c, gin.H{"message": "review deleted successfully"})
}

// ReportReviewRequest 举报评论请求
type ReportReviewRequest struct {
	Reason      string `json:"reason" binding:"required,oneof=spam inappropriate harassment false_info other"`
	Description string `json:"description" binding:"max=500"`
}

// ReportReview 举报评论
func (h *Handler) ReportReview(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}

	var req ReportReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	userID, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	_, err = h.service.ReportReview(c.Request.Context(), ReportReviewParams{
		ReviewID:               reviewID,
		UserHash:               userHash,
		ReporterExternalUserID: userID,
		Reason:                 req.Reason,
		Description:            req.Description,
	})
	if err != nil {
		if respondReportReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to submit report", zap.Error(err))
		response.InternalError(c, "failed to submit report")
		return
	}

	response.Success(c, gin.H{"message": "report submitted successfully"})
}

// CheckContentRequest 内容检查请求
type CheckContentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=5000"`
}

// CheckContent 检查内容是否包含敏感词
func (h *Handler) CheckContent(c *gin.Context) {
	var req CheckContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	result, err := h.service.CheckContent(c.Request.Context(), req.Content)
	if err != nil {
		if respondCheckContentError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to check content", zap.Error(err))
		response.InternalError(c, "failed to check content")
		return
	}
	response.Success(c, result)
}
