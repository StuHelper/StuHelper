package review

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetCourseReviews 获取课程测评列表
func (h *Handler) GetCourseReviews(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}
	page, pageSize := httputil.ParsePage(c)

	// 解析排序和筛选参数
	sort := c.DefaultQuery("sort", "time")
	// 验证 sort 参数
	if sort != "time" && sort != "likes" && sort != "rating" {
		sort = "time"
	}
	termID := c.Query("term_id")
	if termID == "" {
		termID = c.Query("termID")
	}
	var teacherID *int64
	tid := c.Query("teacher_id")
	if tid == "" {
		tid = c.Query("teacherID")
	}
	if tid != "" {
		id, err := strconv.ParseInt(tid, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "invalid teacher_id parameter")
			return
		}
		teacherID = &id
	}

	// 构建缓存键
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:course",
		strconv.FormatInt(courseID, 10)+":sort="+sort+":term="+termID+
			":teacher="+c.Query("teacher_id")+":page="+strconv.Itoa(page)+":size="+strconv.Itoa(pageSize))
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	result, err := h.service.GetCourseReviews(c.Request.Context(), GetCourseReviewsParams{
		CourseID:  courseID,
		Page:      page,
		PageSize:  pageSize,
		Sort:      sort,
		TermID:    termID,
		TeacherID: teacherID,
	})
	if err != nil {
		response.InternalError(c, "failed to load reviews")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	sort := c.Query("sort")

	// 检查缓存
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:latest", "page="+strconv.Itoa(page)+":size="+strconv.Itoa(pageSize)+":sort="+sort)
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	result, err := h.service.GetLatestReviews(c.Request.Context(), GetLatestReviewsParams{
		Page:     page,
		PageSize: pageSize,
		Sort:     sort,
	})
	if err != nil {
		response.InternalError(c, "failed to load latest reviews")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// PostReviewRequest 发布测评请求
type PostReviewRequest struct {
	CourseID  int64         `json:"courseID" binding:"required,gt=0"`
	TeacherID *int64        `json:"teacherID" binding:"omitempty,gt=0"`
	TermID    string        `json:"termID" binding:"omitempty,max=20"`
	Title     string        `json:"title" binding:"max=200"`
	Content   string        `json:"content" binding:"required,min=10,max=5000"`
	Grade     string        `json:"grade" binding:"omitempty,oneof=A+ A A- B+ B B- C+ C C- D F"`
	Ratings   ReviewRatings `json:"ratings" binding:"required"`
}

// PostReview 发布测评
func (h *Handler) PostReview(c *gin.Context) {
	var req PostReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	// 调用 Service 层
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
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrRatingRequired):
			response.BadRequest(c, "at least one rating dimension is required")
		case errors.Is(err, ErrInvalidRating):
			response.BadRequest(c, "rating must be between 1 and 5")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		case errors.Is(err, ErrSensitiveContent):
			response.BadRequest(c, "content contains sensitive words")
		case errors.Is(err, ErrContentEmpty):
			response.BadRequest(c, "content cannot be empty")
		case errors.Is(err, ErrAlreadyReviewed):
			response.Conflict(c, "you have already reviewed this course")
		case errors.Is(err, ErrCourseNotFound):
			response.NotFound(c, "course not found")
		default:
			response.InternalError(c, "failed to create review")
		}
		return
	}

	// 失效相关缓存
	ctx := c.Request.Context()
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:stats"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Created(c, result.Review)
}

// VoteReview 投票
func (h *Handler) VoteReview(c *gin.Context) {
	var req struct {
		VoteType string `json:"voteType" binding:"required,oneof=like dislike"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	reviewID := c.Param("id")
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	// 调用 Service 层
	err := h.service.VoteReview(c.Request.Context(), VoteReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
		VoteType: req.VoteType,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found")
		default:
			response.InternalError(c, "failed to vote")
		}
		return
	}

	// 失效相关缓存
	ctx := c.Request.Context()
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Success(c, gin.H{"message": "vote submitted successfully"})
}

// GetStats 获取评课统计数据
func (h *Handler) GetStats(c *gin.Context) {
	// 检查缓存
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:stats", "all")
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	result, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to load stats")
		return
	}

	data := gin.H{
		"courseCount":     result.CourseCount,
		"reviewCount":    result.ReviewCount,
		"departmentCount": result.DepartmentCount,
	}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// scanReviews 扫描评论行
func scanReviews(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]Review, error) {
	list := make([]Review, 0)
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.ReplyCount, &item.Status, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// scanReviewsWithTotal 扫描评论行（含 COUNT(*) OVER() 窗口函数总数）
func scanReviewsWithTotal(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]Review, int, error) {
	list := make([]Review, 0)
	var total int
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.ReplyCount, &item.Status, &item.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
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
	reviewID := c.Param("id")

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err := h.service.UpdateReview(c.Request.Context(), UpdateReviewParams{
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
			response.NotFound(c, "review not found")
		case errors.Is(err, ErrNotReviewOwner):
			response.Forbidden(c, "you can only edit your own review")
		case errors.Is(err, ErrRatingRequired):
			response.BadRequest(c, "at least one rating dimension is required")
		case errors.Is(err, ErrInvalidRating):
			response.BadRequest(c, "rating must be between 1 and 5")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		case errors.Is(err, ErrContentEmpty):
			response.BadRequest(c, "content cannot be empty")
		default:
			response.InternalError(c, "failed to update review")
		}
		return
	}

	// 失效相关缓存
	ctx := c.Request.Context()
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Success(c, gin.H{"message": "review updated successfully"})
}

// DeleteReview 删除评论
func (h *Handler) DeleteReview(c *gin.Context) {
	reviewID := c.Param("id")
	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err := h.service.DeleteReview(c.Request.Context(), DeleteReviewParams{
		ReviewID: reviewID,
		UserHash: userHash,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found")
		case errors.Is(err, ErrNotReviewOwner):
			response.Forbidden(c, "you can only delete your own review")
		default:
			response.InternalError(c, "failed to delete review")
		}
		return
	}

	ctx := c.Request.Context()
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:stats"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Success(c, gin.H{"message": "review deleted successfully"})
}

// ReportReviewRequest 举报评论请求
type ReportReviewRequest struct {
	Reason      string `json:"reason" binding:"required,oneof=spam inappropriate harassment false_info other"`
	Description string `json:"description" binding:"max=500"`
}

// ReportReview 举报评论
func (h *Handler) ReportReview(c *gin.Context) {
	reviewID := c.Param("id")

	var req ReportReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err := h.service.ReportReview(c.Request.Context(), ReportReviewParams{
		ReviewID:    reviewID,
		UserHash:    userHash,
		Reason:      req.Reason,
		Description: req.Description,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found")
		case errors.Is(err, ErrAlreadyReported):
			response.Conflict(c, "you have already reported this review")
		default:
			response.InternalError(c, "failed to submit report")
		}
		return
	}

	response.Success(c, gin.H{"message": "report submitted successfully"})
}

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
	reviewID := c.Param("id")
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
	ParentID *int64 `json:"parentID" binding:"omitempty,gt=0"`
	Content  string `json:"content" binding:"required,min=1,max=1000"`
}

// CreateReply 创建回复
func (h *Handler) CreateReply(c *gin.Context) {
	reviewID := c.Param("id")

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
	ctx := c.Request.Context()
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Created(c, result.Reply)
}

// DeleteReply 删除回复
func (h *Handler) DeleteReply(c *gin.Context) {
	replyID, err := httputil.ParseIDParam(c, "id")
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
	ctx := c.Request.Context()
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

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
	id, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash := httputil.HashUserID(userID)

	err = h.service.MarkNotificationRead(c.Request.Context(), id, userHash)
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

// GetRatingTrend 获取课程评分趋势
func (h *Handler) GetRatingTrend(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	ctx := c.Request.Context()

	cacheKey := h.cache.BuildVersionedKey(ctx, "review:rating_trend", strconv.FormatInt(courseID, 10))
	if cached, ok := h.cache.Get(ctx, cacheKey); ok {
		response.Success(c, cached)
		return
	}

	trend, err := h.service.GetRatingTrend(ctx, courseID)
	if err != nil {
		response.InternalError(c, "failed to load rating trend")
		return
	}

	data := gin.H{"trend": trend}
	if err := h.cache.Set(ctx, cacheKey, data, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// GetHotCourses 获取热门课程排行
func (h *Handler) GetHotCourses(c *gin.Context) {
	period := c.DefaultQuery("period", "all")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 使用缓存
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:hot", "period="+period+":limit="+strconv.Itoa(limit))
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	list, err := h.service.GetHotCourses(c.Request.Context(), period, limit)
	if err != nil {
		response.InternalError(c, "failed to load hot courses")
		return
	}

	data := gin.H{"list": list}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// GetTeacherRatingStats 获取教师评分统计
func (h *Handler) GetTeacherRatingStats(c *gin.Context) {
	teacherID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid teacher id")
		return
	}

	// 使用缓存
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:teacher_stats", "id="+c.Param("id"))
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	stats, err := h.service.GetTeacherRatingStats(c.Request.Context(), teacherID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTeacherNotFound):
			response.NotFound(c, "teacher not found")
		default:
			response.InternalError(c, "failed to load teacher stats")
		}
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, stats, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, stats)
}

// CheckContentRequest 内容检查请求
type CheckContentRequest struct {
	Content string `json:"content" binding:"required"`
}

// CheckContent 检查内容是否包含敏感词
func (h *Handler) CheckContent(c *gin.Context) {
	var req CheckContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result := h.service.CheckContent(c.Request.Context(), req.Content)
	response.Success(c, result)
}

// CheckQuality 检查内容质量
func (h *Handler) CheckQuality(c *gin.Context) {
	var req CheckContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result := h.service.CheckQuality(req.Content)
	response.Success(c, result)
}
