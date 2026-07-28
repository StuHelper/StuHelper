package review

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetCourseReviews 获取课程测评列表
func (h *Handler) GetCourseReviews(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
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
	userHash, ok := h.resolveOptionalUserHash(c)
	if !ok {
		return
	}

	result, err := h.service.GetCourseReviews(c.Request.Context(), GetCourseReviewsParams{
		CourseID:  courseID,
		Page:      page,
		PageSize:  pageSize,
		Sort:      sort,
		TermID:    termID,
		TeacherID: teacherID,
		UserHash:  userHash,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to get course reviews", zap.Error(err))
		response.InternalError(c, "failed to load reviews")
		return
	}

	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	stripped := stripReviewsForResponse(result.List, facts)
	response.Success(c, buildPaginatedReviewListData(stripped, result.Total, page, pageSize))
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	sort := c.DefaultQuery("sort", "time")
	if !isValidSort(sort) {
		sort = SortTime
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
	userHash, ok := h.resolveOptionalUserHash(c)
	if !ok {
		return
	}

	result, err := h.service.GetLatestReviews(c.Request.Context(), GetLatestReviewsParams{
		Page:      page,
		PageSize:  pageSize,
		Sort:      sort,
		TeacherID: teacherID,
		UserHash:  userHash,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to get latest reviews", zap.Error(err))
		response.InternalError(c, "failed to load latest reviews")
		return
	}

	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	stripped := stripReviewsForResponse(result.List, facts)
	response.Success(c, buildPaginatedReviewListData(stripped, result.Total, page, pageSize))
}

// SearchReviews 搜索测评（支持课程名/课号、院系、教师、学期等条件）
func (h *Handler) SearchReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	sort := c.DefaultQuery("sort", "time")
	if !isValidSort(sort) {
		sort = SortTime
	}

	q := strings.TrimSpace(c.Query("q"))
	teacherName := strings.TrimSpace(c.Query("teacherName"))
	termID := strings.TrimSpace(c.Query("termID"))
	if termID != "" && !validTermIDFormat.MatchString(termID) {
		response.BadRequest(c, "invalid term_id format, expected YYYY-S (e.g. 2024-1)")
		return
	}

	var departmentID int64
	if dept := strings.TrimSpace(c.Query("departmentID")); dept != "" {
		id, err := strconv.ParseInt(dept, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "invalid department_id parameter")
			return
		}
		departmentID = id
	}

	if q == "" && teacherName == "" && termID == "" && departmentID == 0 {
		response.BadRequest(c, "at least one search condition is required")
		return
	}
	userHash, ok := h.resolveOptionalUserHash(c)
	if !ok {
		return
	}

	result, err := h.service.SearchReviews(c.Request.Context(), SearchReviewsParams{
		Query:        q,
		DepartmentID: departmentID,
		TeacherName:  teacherName,
		TermID:       termID,
		Page:         page,
		PageSize:     pageSize,
		Sort:         sort,
		UserHash:     userHash,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to search reviews", zap.Error(err))
		response.InternalError(c, "failed to search reviews")
		return
	}

	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	stripped := stripReviewsForResponse(result.List, facts)
	response.Success(c, buildPaginatedReviewListData(stripped, result.Total, page, pageSize))
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
	userHash, ok := h.resolveOptionalUserHash(c)
	if !ok {
		return
	}

	result, err := h.service.GetBatchCourseReviews(c.Request.Context(), GetBatchCourseReviewsParams{
		CourseIDs: courseIDs,
		PageSize:  pageSize,
		Sort:      sort,
		UserHash:  userHash,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to get batch reviews", zap.Error(err))
		response.InternalError(c, "failed to get batch reviews")
		return
	}

	facts, ok := h.resolveReviewAccessFactsForRequest(c)
	if !ok {
		return
	}
	response.Success(c, buildGroupedReviewListData(courseIDs, result, facts, pageSize))
}

// GetStats 获取评课统计数据
func (h *Handler) GetStats(c *gin.Context) {
	respondWithCachedData(h, c, "review:stats", "all", h.service.GetStats, func(result *StatsResult) any {
		return gin.H{
			"courseCount":     result.CourseCount,
			"reviewCount":     result.ReviewCount,
			"departmentCount": result.DepartmentCount,
			"userCount":       result.UserCount,
		}
	}, "failed to load stats", "failed to load stats", nil)
}

// GetRatingTrend 获取课程评分趋势
func (h *Handler) GetRatingTrend(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	respondWithCachedData(h, c, "review:rating_trend", strconv.FormatInt(courseID, 10), func(ctx context.Context) ([]RatingTrendItem, error) {
		return h.service.GetRatingTrend(ctx, courseID)
	}, func(trend []RatingTrendItem) any {
		return gin.H{"trend": trend}
	}, "failed to load rating trend", "failed to load rating trend", nil)
}

// GetHotCourses 获取热门课程排行
func (h *Handler) GetHotCourses(c *gin.Context) {
	period := c.DefaultQuery("period", "all")
	if period != "week" && period != "month" && period != "all" {
		period = "all"
	}
	limit, ok := httputil.ParseOptionalIntQuery(c, "limit")
	if !ok {
		response.BadRequest(c, "invalid limit parameter")
		return
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	respondWithCachedData(h, c, "review:hot", "period="+period+":limit="+strconv.Itoa(limit), func(ctx context.Context) ([]HotCourse, error) {
		return h.service.GetHotCourses(ctx, period, limit)
	}, func(list []HotCourse) any {
		return gin.H{"list": list}
	}, "failed to load hot courses", "failed to load hot courses", nil)
}

// GetCourseTeachers 获取课程的授课教师列表
func (h *Handler) GetCourseTeachers(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	respondWithCachedData(h, c, "review:course_teachers", strconv.FormatInt(courseID, 10), func(ctx context.Context) ([]CourseTeacherStats, error) {
		return h.service.GetCourseTeachers(ctx, courseID)
	}, func(list []CourseTeacherStats) any {
		if list == nil {
			return []CourseTeacherStats{}
		}
		return list
	}, "failed to load course teachers", "failed to load course teachers", nil)
}

// GetTeacherRatingStats 获取教师评分统计
func (h *Handler) GetTeacherRatingStats(c *gin.Context) {
	teacherID, err := httputil.ParseIDParam(c, "teacherID")
	if err != nil {
		response.BadRequest(c, "invalid teacher id")
		return
	}

	respondWithCachedData(h, c, "review:teacher_stats", "teacherID="+c.Param("teacherID"), func(ctx context.Context) (*TeacherRatingStatsResponse, error) {
		return h.service.GetTeacherRatingStats(ctx, teacherID)
	}, nil, "failed to load teacher stats", "failed to load teacher stats", respondTeacherLookupError)
}
