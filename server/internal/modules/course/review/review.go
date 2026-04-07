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
	facts := h.resolveReviewAccessFactsForRequest(c)
	if !facts.CanPostReview {
		response.Forbidden(c, "student and identity verification are required to post reviews", errs.ErrAccessDenied)
		return
	}

	var req PostReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	requestID, _ := c.Get(middleware.CtxKeyRequestID)
	requestIDStr, _ := requestID.(string) //nolint:errcheck // gin context value, empty string is safe fallback

	result, err := h.service.PostReview(c.Request.Context(), PostReviewParams{
		CourseID:  req.CourseID,
		TeacherID: req.TeacherID,
		TermID:    req.TermID,
		Title:     req.Title,
		Content:   req.Content,
		Grade:     req.Grade,
		Ratings:   req.Ratings,
		UserHash:  userHash,
		IPAddress: c.ClientIP(),
		RequestID: requestIDStr,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrRatingRequired):
			response.BadRequest(c, "at least one rating dimension is required")
		case errors.Is(err, ErrInvalidRating):
			response.BadRequest(c, "rating must be between 1 and 5")
		case errors.Is(err, ErrTitleEmpty):
			response.BadRequest(c, "title cannot be empty")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		case errors.Is(err, ErrSensitiveContent):
			response.BadRequest(c, "content contains sensitive words", errs.ErrSensitiveContent)
		case errors.Is(err, ErrContentEmpty):
			response.BadRequest(c, "content cannot be empty", errs.ErrContentEmpty)
		case errors.Is(err, ErrAlreadyReviewed):
			response.Conflict(c, "you have already reviewed this course", errs.ErrReviewExists)
		case errors.Is(err, ErrCourseNotFound):
			response.NotFound(c, "course not found", errs.ErrCourseNotFound)
		default:
			logger.FromGin(c).Error("failed to create review", zap.Error(err))
			response.InternalError(c, "failed to create review")
		}
		return
	}

	h.invalidateReviewAggregateCaches(c)

	// 异步写入 OpenFGA 关系 tuple（非阻塞，失败仅记日志）
	h.writeFGAReviewTuples(c, result.Review.ID, userID, req.CourseID)

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
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	courseID, err := h.service.VoteReview(c.Request.Context(), VoteReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
		VoteType: req.VoteType,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found", errs.ErrReviewNotFound)
		default:
			logger.FromGin(c).Error("failed to vote", zap.Error(err))
			response.InternalError(c, "failed to vote")
		}
		return
	}

	h.invalidateReviewCaches(c, courseID)
	response.Success(c, gin.H{"message": "vote submitted successfully"})
}

// UpdateReviewRequest 更新评论请求
type UpdateReviewRequest struct {
	Title   string        `json:"title" binding:"max=200"`
	Content string        `json:"content" binding:"required,min=10,max=5000"`
	Grade   string        `json:"grade" binding:"omitempty,oneof=A+ A A- B+ B B- C+ C C- D F"`
	Ratings ReviewRatings `json:"ratings" binding:"required"`
}

// UpdateReview 更新评论
func (h *Handler) UpdateReview(c *gin.Context) {
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

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
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
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found", errs.ErrReviewNotFound)
		case errors.Is(err, ErrNotReviewOwner):
			response.Forbidden(c, "you can only edit your own review", errs.ErrNotReviewOwner)
		case errors.Is(err, ErrRatingRequired):
			response.BadRequest(c, "at least one rating dimension is required")
		case errors.Is(err, ErrInvalidRating):
			response.BadRequest(c, "rating must be between 1 and 5")
		case errors.Is(err, ErrTitleEmpty):
			response.BadRequest(c, "title cannot be empty")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		case errors.Is(err, ErrSensitiveContent):
			response.BadRequest(c, "content contains sensitive words", errs.ErrSensitiveContent)
		case errors.Is(err, ErrContentEmpty):
			response.BadRequest(c, "content cannot be empty", errs.ErrContentEmpty)
		default:
			logger.FromGin(c).Error("failed to update review", zap.Error(err))
			response.InternalError(c, "failed to update review")
		}
		return
	}

	h.invalidateReviewAggregateCaches(c)
	response.Success(c, gin.H{"message": "review updated successfully"})
}

// DeleteReview 删除评论
func (h *Handler) DeleteReview(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	err = h.service.DeleteReview(c.Request.Context(), DeleteReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found", errs.ErrReviewNotFound)
		case errors.Is(err, ErrNotReviewOwner):
			response.Forbidden(c, "you can only delete your own review", errs.ErrNotReviewOwner)
		default:
			logger.FromGin(c).Error("failed to delete review", zap.Error(err))
			response.InternalError(c, "failed to delete review")
		}
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

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	reportID, err := h.service.ReportReview(c.Request.Context(), ReportReviewParams{
		ReviewID:    reviewID,
		UserHash:    userHash,
		Reason:      req.Reason,
		Description: req.Description,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found", errs.ErrReviewNotFound)
		case errors.Is(err, ErrAlreadyReported):
			response.Conflict(c, "you have already reported this review", errs.ErrAlreadyReported)
		default:
			logger.FromGin(c).Error("failed to submit report", zap.Error(err))
			response.InternalError(c, "failed to submit report")
		}
		return
	}

	// 异步写入 FGA 举报关系 tuple
	h.writeFGAReportTuplesForReview(c, reportID, userID, reviewID)

	response.Success(c, gin.H{"message": "report submitted successfully"})
}

// CheckContentRequest 内容检查请求
type CheckContentRequest struct {
	Content string `json:"content" binding:"required"`
}

// CheckContent 检查内容是否包含敏感词
func (h *Handler) CheckContent(c *gin.Context) {
	var req CheckContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	result := h.service.CheckContent(c.Request.Context(), req.Content)
	response.Success(c, result)
}
