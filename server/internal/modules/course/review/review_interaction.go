package review

import (
	"errors"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// AddFavorite 添加收藏
func (h *Handler) AddFavorite(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err = h.service.AddFavorite(c.Request.Context(), AddFavoriteParams{
		UserHash: userHash,
		CourseID: courseID,
	})
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			response.NotFound(c, "course not found")
			return
		}
		response.InternalError(c, "failed to add favorite")
		return
	}

	response.Success(c, gin.H{"message": "course favorited successfully"})
}

// RemoveFavorite 取消收藏
func (h *Handler) RemoveFavorite(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err = h.service.RemoveFavorite(c.Request.Context(), userHash, courseID)
	if err != nil {
		response.InternalError(c, "failed to remove favorite")
		return
	}

	response.Success(c, gin.H{"message": "favorite removed successfully"})
}

// GetUserFavorites 获取用户收藏列表
func (h *Handler) GetUserFavorites(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	result, err := h.service.GetUserFavorites(c.Request.Context(), GetUserFavoritesParams{
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load favorites")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// GetUserReviews 获取用户评论列表
func (h *Handler) GetUserReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	result, err := h.service.GetUserReviews(c.Request.Context(), GetUserReviewsParams{
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load reviews")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// GetUserVotes 获取用户点赞列表
func (h *Handler) GetUserVotes(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	voteType := c.DefaultQuery("vote_type", "like")
	if voteType != "like" && voteType != "dislike" {
		response.BadRequest(c, "vote_type must be 'like' or 'dislike'")
		return
	}
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	result, err := h.service.GetUserVotes(c.Request.Context(), GetUserVotesParams{
		UserHash: userHash,
		VoteType: voteType,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load votes")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// SaveDraftRequest 保存草稿请求
type SaveDraftRequest struct {
	CourseID  int64         `json:"courseID" binding:"required,gt=0"`
	TeacherID *int64        `json:"teacherID" binding:"omitempty,gt=0"`
	TermID    string        `json:"termID" binding:"omitempty,max=20"`
	Title     string        `json:"title" binding:"max=200"`
	Content   string        `json:"content" binding:"max=5000"`
	Grade     string        `json:"grade" binding:"omitempty,oneof=A+ A A- B+ B B- C+ C C- D F"`
	Ratings   ReviewRatings `json:"ratings"`
}

// SaveDraft 保存草稿
func (h *Handler) SaveDraft(c *gin.Context) {
	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	draft, err := h.service.SaveDraft(c.Request.Context(), SaveDraftParams{
		UserHash:  userHash,
		CourseID:  req.CourseID,
		TeacherID: req.TeacherID,
		TermID:    req.TermID,
		Title:     req.Title,
		Content:   req.Content,
		Grade:     req.Grade,
		Ratings:   req.Ratings,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrCourseNotFound):
			response.NotFound(c, "course not found")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		default:
			response.InternalError(c, "failed to save draft")
		}
		return
	}

	response.Success(c, draft)
}

// GetDraft 获取草稿
func (h *Handler) GetDraft(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseId")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	draft, err := h.service.GetDraft(c.Request.Context(), userHash, courseID)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.NotFound(c, "draft not found")
		} else {
			response.InternalError(c, "failed to get draft")
		}
		return
	}

	response.Success(c, draft)
}

// DeleteDraft 删除草稿
func (h *Handler) DeleteDraft(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseId")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err = h.service.DeleteDraft(c.Request.Context(), userHash, courseID)
	if err != nil {
		response.InternalError(c, "failed to delete draft")
		return
	}

	response.Success(c, gin.H{"message": "draft deleted successfully"})
}

// GetReplies 获取回复列表
func (h *Handler) GetReplies(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}
	page, pageSize := httputil.ParsePage(c)

	var userHash string
	if userID := middleware.GetUserID(c); userID != "" {
		userHash = httputil.HashUserID(userID)
	}

	result, err := h.service.GetReplies(c.Request.Context(), GetRepliesParams{
		ReviewID: reviewID,
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load replies")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// CreateReplyRequest 创建回复请求
type CreateReplyRequest struct {
	ParentID *string `json:"parentID" binding:"omitempty"`
	Content  string `json:"content" binding:"required,min=1,max=1000"`
}

// CreateReply 创建回复
func (h *Handler) CreateReply(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid review id")
		return
	}

	var req CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	result, err := h.service.CreateReply(c.Request.Context(), CreateReplyParams{
		ReviewID: reviewID,
		ParentID: req.ParentID,
		UserHash: userHash,
		Content:  req.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains dangerous elements")
		case errors.Is(err, ErrSensitiveContent):
			response.BadRequest(c, "content contains sensitive words")
		default:
			response.InternalError(c, "failed to create reply")
		}
		return
	}

	// 失效相关缓存
	h.invalidateReviewCaches(c)

	response.Created(c, result.Reply)
}

// DeleteReply 删除回复
func (h *Handler) DeleteReply(c *gin.Context) {
	replyID, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid reply id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err = h.service.DeleteReply(c.Request.Context(), DeleteReplyParams{
		ReplyID:  replyID,
		UserHash: userHash,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReplyNotFound):
			response.NotFound(c, "reply not found")
		case errors.Is(err, ErrNotReplyOwner):
			response.Forbidden(c, "you can only delete your own reply")
		default:
			response.InternalError(c, "failed to delete reply")
		}
		return
	}

	// 失效相关缓存
	h.invalidateReviewCaches(c)

	response.Success(c, gin.H{"message": "reply deleted"})
}

// GetNotifications 获取通知列表
func (h *Handler) GetNotifications(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	result, err := h.service.GetNotifications(c.Request.Context(), GetNotificationsParams{
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
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
	userHash := httputil.HashUserID(userID)

	count, err := h.service.GetUnreadNotificationCount(c.Request.Context(), userHash)
	if err != nil {
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
	userHash := httputil.HashUserID(userID)

	err = h.service.MarkNotificationRead(c.Request.Context(), notifID, userHash)
	if err != nil {
		response.InternalError(c, "failed to mark notification as read")
		return
	}

	response.Success(c, gin.H{"message": "notification marked as read"})
}

// MarkAllNotificationsRead 标记所有通知已读
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err := h.service.MarkAllNotificationsRead(c.Request.Context(), userHash)
	if err != nil {
		response.InternalError(c, "failed to mark all notifications as read")
		return
	}

	response.Success(c, gin.H{"message": "all notifications marked as read"})
}
