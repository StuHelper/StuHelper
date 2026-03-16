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

	sort := c.DefaultQuery("sort", "time")
	if !isValidSort(sort) {
		sort = SortTime
	}
	termID := c.Query("termID")
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

	facts := h.resolveReviewAccessFactsForRequest(c)
	stripped := stripReviewsForResponse(result.List, facts)
	response.Success(c, gin.H{"list": stripped, "total": result.Total, "authenticated": facts.Authenticated})
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	sort := c.DefaultQuery("sort", "time")
	if !isValidSort(sort) {
		sort = SortTime
	}

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

	facts := h.resolveReviewAccessFactsForRequest(c)
	stripped := stripReviewsForResponse(result.List, facts)
	response.Success(c, gin.H{"list": stripped, "total": result.Total, "authenticated": facts.Authenticated})
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

	facts := h.resolveReviewAccessFactsForRequest(c)
	data := make(map[string]interface{})
	for _, cid := range courseIDs {
		reviews := result.Reviews[cid]
		if reviews == nil {
			reviews = []Review{}
		}
		stripped := stripReviewsForResponse(reviews, facts)
		data[strconv.FormatInt(cid, 10)] = map[string]interface{}{
			"list":  stripped,
			"total": result.Totals[cid],
		}
	}

	response.Success(c, gin.H{"data": data, "authenticated": facts.Authenticated})
}

// GetStats 获取评课统计数据
func (h *Handler) GetStats(c *gin.Context) {
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:stats", "all")
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

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
	if period != "week" && period != "month" && period != "all" {
		period = "all"
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

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
