package course

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// maxSearchLength 搜索关键词最大长度
const maxSearchLength = 100

const (
	CourseSortName        = "name"
	CourseSortCredits     = "credits"
	CourseSortReviewCount = "reviewCount"
)

func normalizeCourseSort(sort string) string {
	switch sort {
	case CourseSortCredits, CourseSortReviewCount:
		return sort
	default:
		return CourseSortName
	}
}

// GetCourseCategories 获取课程分类列表
func (h *Handler) GetCourseCategories(c *gin.Context) {
	cacheKey := "course:categories"
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	categories, err := h.service.GetCourseCategories(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to load categories")
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, categories, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, categories)
}

// GetDepartments 获取院系列表
func (h *Handler) GetDepartments(c *gin.Context) {
	category := c.Query("category")
	cacheKey := "course:departments:" + httputil.SanitizeCacheKey(category)
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	departments, err := h.service.GetDepartments(c.Request.Context(), category)
	if err != nil {
		response.InternalError(c, "failed to load departments")
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, departments, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, departments)
}

// GetTerms 获取学期列表
func (h *Handler) GetTerms(c *gin.Context) {
	cacheKey := "course:terms"
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	terms, err := h.service.GetTerms(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to load terms")
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, terms, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, terms)
}

// GetCourses 获取课程列表
func (h *Handler) GetCourses(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	query := strings.TrimSpace(c.Query("q"))
	departmentID, ok := httputil.ParseOptionalInt64Query(c, "departmentID")
	if !ok {
		response.BadRequest(c, "invalid departmentID")
		return
	}
	category := c.Query("category")
	sort := normalizeCourseSort(c.DefaultQuery("sort", CourseSortName))

	if len(query) > maxSearchLength {
		response.BadRequest(c, "search query too long")
		return
	}

	userHash := h.resolveUserHash(c)

	// 未登录用户：走 raw cache 快速路径
	cacheKey := "course:courses:q=" + httputil.SanitizeCacheKey(query) + ":dept=" + strconv.FormatInt(departmentID, 10) + ":cat=" + httputil.SanitizeCacheKey(category) + ":sort=" + sort + ":page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if userHash == "" {
		if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
			response.Success(c, cached)
			return
		}
	}

	// 已登录用户或缓存未命中：使用类型化缓存
	ctx := c.Request.Context()
	result, cacheHit := cache.GetAs[ListCoursesResult](h.cache, ctx, cacheKey)
	if !cacheHit {
		fetched, fetchErr := h.service.GetCourses(ctx, ListCoursesParams{
			Query:        query,
			DepartmentID: departmentID,
			Category:     category,
			Sort:         sort,
			Page:         page,
			PageSize:     pageSize,
		})
		if fetchErr != nil {
			response.InternalError(c, "failed to load courses")
			return
		}
		result = *fetched

		data := gin.H{"list": result.List, "total": result.Total}
		if err := h.cache.Set(ctx, cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
			logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
		}
	}

	if userHash != "" {
		h.annotateFavorites(c, userHash, result.List)
	}
	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// SearchCourses 搜索课程
func (h *Handler) SearchCourses(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.BadRequest(c, "search query is required")
		return
	}
	if len(q) > maxSearchLength {
		response.BadRequest(c, "search query too long")
		return
	}
	page, pageSize := httputil.ParsePage(c)

	userHash := h.resolveUserHash(c)

	cacheKey := "course:courses:search:" + httputil.SanitizeCacheKey(q) + ":page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if userHash == "" {
		if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
			response.Success(c, cached)
			return
		}
	}

	ctx := c.Request.Context()
	result, cacheHit := cache.GetAs[ListCoursesResult](h.cache, ctx, cacheKey)
	if !cacheHit {
		fetched, err := h.service.SearchCourses(ctx, SearchCoursesParams{
			Query:    q,
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			response.InternalError(c, "failed to search courses")
			return
		}
		result = *fetched

		data := gin.H{"list": result.List, "total": result.Total}
		if err := h.cache.Set(ctx, cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
			logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
		}
	}

	if userHash != "" {
		h.annotateFavorites(c, userHash, result.List)
	}
	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// GetCourse 获取课程详情
func (h *Handler) GetCourse(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userHash := h.resolveUserHash(c)

	cacheKey := "course:course:" + strconv.FormatInt(courseID, 10)

	// 未登录用户：走 raw cache 快速路径
	if userHash == "" {
		if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
			response.Success(c, cached)
			return
		}
	}

	ctx := c.Request.Context()
	course, cacheHit := cache.GetAs[Course](h.cache, ctx, cacheKey)
	if !cacheHit {
		fetched, fetchErr := h.service.GetCourse(ctx, courseID)
		if fetchErr != nil {
			if errors.Is(fetchErr, ErrCourseNotFound) {
				response.NotFound(c, "course not found", errs.ErrCourseNotFound)
				return
			}
			response.InternalError(c, "failed to load course")
			return
		}
		course = *fetched

		if err := h.cache.Set(ctx, cacheKey, course, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
			logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
		}
	}

	if userHash != "" {
		favorited, favErr := h.service.repo.FavoriteExists(ctx, userHash, courseID)
		if favErr != nil {
			logger.FromGin(c).Warn("failed to check favorite status", zap.Error(favErr))
		} else {
			course.IsFavorited = &favorited
		}
	}
	response.Success(c, course)
}

// GetStats 获取学习中心统计数据
func (h *Handler) GetStats(c *gin.Context) {
	cacheKey := "course:stats"
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to load stats")
		return
	}

	data := gin.H{
		"courseCount":     stats.CourseCount,
		"departmentCount": stats.DepartmentCount,
	}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// resolveUserHash 从请求上下文获取已登录用户的 userHash，未登录返回空字符串
func (h *Handler) resolveUserHash(c *gin.Context) string {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return ""
	}
	hash, err := httputil.HashUserID(userID)
	if err != nil {
		logger.FromGin(c).Warn("failed to hash user identity", zap.Error(err))
		return ""
	}
	return hash
}

// annotateFavorites 批量标注课程列表的收藏状态（原地修改）
func (h *Handler) annotateFavorites(c *gin.Context, userHash string, courses []Course) {
	if len(courses) == 0 {
		return
	}
	ids := make([]int64, len(courses))
	for i := range courses {
		ids[i] = courses[i].ID
	}
	favSet, err := h.service.repo.BatchFavoritedCourseIDs(c.Request.Context(), userHash, ids)
	if err != nil {
		logger.FromGin(c).Warn("failed to batch check favorites", zap.Error(err))
		return
	}
	for i := range courses {
		fav := favSet[courses[i].ID]
		courses[i].IsFavorited = &fav
	}
}
