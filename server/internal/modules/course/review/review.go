package review

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// stripReviewsForResponse 根据认证状态和管理员身份脱敏评论内容
// - hidden 评论：非管理员看不到内容（保留 moderationReason）
// - 未登录用户：看不到任何评论正文
func stripReviewsForResponse(reviews []Review, isAuthenticated, isAdmin bool) []Review {
	result := make([]Review, len(reviews))
	copy(result, reviews)
	for i := range result {
		if result[i].Status == StatusHidden && !isAdmin {
			result[i].Content = ""
			result[i].Title = ""
		} else if !isAuthenticated {
			result[i].Content = ""
			result[i].Title = ""
		}
	}
	return result
}

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
	if !isValidSort(sort) {
		sort = SortTime
	}
	termID := c.Query("termID")
	// M-84: termID 格式白名单校验，防止非法值污染缓存键
	if termID != "" && !validTermIDFormat.MatchString(termID) {
		response.BadRequest(c, "invalid term_id format, expected YYYY-S (e.g. 2024-1)")
		return
	}
	var teacherID *int64
	tid := c.Query("teacherID")
	if tid != "" {
		id, err := strconv.ParseInt(tid, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "invalid teacher_id parameter")
			return
		}
		teacherID = &id
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
		logger.FromGin(c).Error("failed to get course reviews", zap.Error(err))
		response.InternalError(c, "failed to load reviews")
		return
	}

	// 按认证状态脱敏后返回（缓存存完整数据在 service 层，脱敏在 handler 层按请求执行）
	authenticated := middleware.IsAuthenticated(c)
	isAdmin := middleware.GetIsAdmin(c)
	stripped := stripReviewsForResponse(result.List, authenticated, isAdmin)
	response.Success(c, gin.H{"list": stripped, "total": result.Total, "authenticated": authenticated})
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	sort := c.DefaultQuery("sort", "time")
	// 白名单校验 sort 参数
	if !isValidSort(sort) {
		sort = SortTime
	}

	// 调用 Service 层
	result, err := h.service.GetLatestReviews(c.Request.Context(), GetLatestReviewsParams{
		Page:     page,
		PageSize: pageSize,
		Sort:     sort,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to get latest reviews", zap.Error(err))
		response.InternalError(c, "failed to load latest reviews")
		return
	}

	// 按认证状态脱敏后返回
	authenticated := middleware.IsAuthenticated(c)
	isAdmin := middleware.GetIsAdmin(c)
	stripped := stripReviewsForResponse(result.List, authenticated, isAdmin)
	response.Success(c, gin.H{"list": stripped, "total": result.Total, "authenticated": authenticated})
}

// GetBatchCourseReviews 批量获取多个课程的测评列表
func (h *Handler) GetBatchCourseReviews(c *gin.Context) {
	idsStr := c.Query("courseIDs")
	if idsStr == "" {
		response.BadRequest(c, "courseIDs is required")
		return
	}

	parts := strings.Split(idsStr, ",")
	if len(parts) > 20 {
		response.BadRequest(c, "maximum 20 course IDs allowed")
		return
	}

	courseIDs := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "invalid course ID: "+p)
			return
		}
		courseIDs = append(courseIDs, id)
	}

	if len(courseIDs) == 0 {
		response.BadRequest(c, "at least one valid course ID is required")
		return
	}

	_, pageSize := httputil.ParsePage(c)
	sort := c.DefaultQuery("sort", "time")
	if !isValidSort(sort) {
		sort = SortTime
	}

	result, err := h.service.GetBatchCourseReviews(c.Request.Context(), GetBatchCourseReviewsParams{
		CourseIDs: courseIDs,
		PageSize:  pageSize,
		Sort:      sort,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to get batch reviews", zap.Error(err))
		response.InternalError(c, "failed to get batch reviews")
		return
	}

	// 按认证状态脱敏后返回
	authenticated := middleware.IsAuthenticated(c)
	isAdmin := middleware.GetIsAdmin(c)

	// 构建响应 map: courseID string -> {list, total}
	data := make(map[string]interface{})
	for _, cid := range courseIDs {
		reviews := result.Reviews[cid]
		if reviews == nil {
			reviews = []Review{}
		}
		stripped := stripReviewsForResponse(reviews, authenticated, isAdmin)
		total := result.Totals[cid]
		data[strconv.FormatInt(cid, 10)] = map[string]interface{}{
			"list":  stripped,
			"total": total,
		}
	}

	response.Success(c, gin.H{"data": data, "authenticated": authenticated})
}

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

	// L-35: 从 gin context 提取 request_id，传入 service 层用于审计日志
	requestID, _ := c.Get(middleware.CtxKeyRequestID)
	requestIDStr, _ := requestID.(string)

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

	// 失效相关缓存（精确失效该课程的缓存）
	h.invalidateReviewCaches(c, req.CourseID, "review:stats")

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

	reviewID, err := httputil.ParseUUIDParam(c, "id")
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

	// 调用 Service 层
	err = h.service.VoteReview(c.Request.Context(), VoteReviewParams{
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

	// 失效相关缓存（无法确定课程 ID，全局失效）
	h.invalidateReviewCaches(c, 0)

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
		logger.FromGin(c).Error("failed to load stats", zap.Error(err))
		response.InternalError(c, "failed to load stats")
		return
	}

	data := gin.H{
		"courseCount":     result.CourseCount,
		"reviewCount":     result.ReviewCount,
		"departmentCount": result.DepartmentCount,
	}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
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
	reviewID, err := httputil.ParseUUIDParam(c, "id")
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
		case errors.Is(err, ErrContentEmpty):
			response.BadRequest(c, "content cannot be empty", errs.ErrContentEmpty)
		default:
			logger.FromGin(c).Error("failed to update review", zap.Error(err))
			response.InternalError(c, "failed to update review")
		}
		return
	}

	// 失效相关缓存（无法确定课程 ID，全局失效）
	h.invalidateReviewCaches(c, 0)

	response.Success(c, gin.H{"message": "review updated successfully"})
}

// DeleteReview 删除评论
func (h *Handler) DeleteReview(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "id")
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

	h.invalidateReviewCaches(c, 0, "review:stats", "review:votes")

	response.Success(c, gin.H{"message": "review deleted successfully"})
}

// ReportReviewRequest 举报评论请求
type ReportReviewRequest struct {
	Reason      string `json:"reason" binding:"required,oneof=spam inappropriate harassment false_info other"`
	Description string `json:"description" binding:"max=500"`
}

// ReportReview 举报评论
func (h *Handler) ReportReview(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "id")
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

	err = h.service.ReportReview(c.Request.Context(), ReportReviewParams{
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

	response.Success(c, gin.H{"message": "report submitted successfully"})
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
		logger.FromGin(c).Error("failed to load rating trend", zap.Error(err))
		response.InternalError(c, "failed to load rating trend")
		return
	}

	data := gin.H{"trend": trend}
	if err := h.cache.Set(ctx, cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// GetHotCourses 获取热门课程排行
func (h *Handler) GetHotCourses(c *gin.Context) {
	period := c.DefaultQuery("period", "all")
	// 白名单校验 period 参数
	if period != "week" && period != "month" && period != "all" {
		period = "all"
	}
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
		logger.FromGin(c).Error("failed to load hot courses", zap.Error(err))
		response.InternalError(c, "failed to load hot courses")
		return
	}

	data := gin.H{"list": list}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// GetCourseTeachers 获取课程的授课教师列表
func (h *Handler) GetCourseTeachers(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:course_teachers", strconv.FormatInt(courseID, 10))
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	list, err := h.service.GetCourseTeachers(c.Request.Context(), courseID)
	if err != nil {
		logger.FromGin(c).Error("failed to load course teachers", zap.Error(err))
		response.InternalError(c, "failed to load course teachers")
		return
	}
	if list == nil {
		list = []CourseTeacherStats{}
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, list, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, list)
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
			response.NotFound(c, "teacher not found", errs.ErrTeacherNotFound)
		default:
			logger.FromGin(c).Error("failed to load teacher stats", zap.Error(err))
			response.InternalError(c, "failed to load teacher stats")
		}
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, stats, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
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
		response.BadRequest(c, "invalid request parameters")
		return
	}

	result := h.service.CheckContent(c.Request.Context(), req.Content)
	response.Success(c, result)
}
