package course

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetDepartments 获取院系列表
func (h *Handler) GetDepartments(c *gin.Context) {
	category := c.Query("category")
	cacheKey := "course:departments:" + sanitizeCacheKey(category)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	departments, err := h.service.GetDepartments(c.Request.Context(), category)
	if err != nil {
		response.InternalError(c, "failed to load departments")
		return
	}

	_ = h.setCache(c.Request.Context(), cacheKey, departments, cacheTTL)
	response.Success(c, departments)
}

// GetCourses 获取课程列表
func (h *Handler) GetCourses(c *gin.Context) {
	page, pageSize := parsePage(c)
	cacheKey := "course:courses:page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	result, err := h.service.GetCourses(c.Request.Context(), ListCoursesParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load courses")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	_ = h.setCache(c.Request.Context(), cacheKey, data, cacheTTL)
	response.Success(c, data)
}

// SearchCourses 搜索课程
func (h *Handler) SearchCourses(c *gin.Context) {
	q := c.Query("q")
	if len(q) > maxSearchLength {
		response.BadRequest(c, "search query too long")
		return
	}
	page, pageSize := parsePage(c)
	cacheKey := "course:courses:search:" + sanitizeCacheKey(q) + ":page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	result, err := h.service.SearchCourses(c.Request.Context(), SearchCoursesParams{
		Query:    q,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to search courses")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	_ = h.setCache(c.Request.Context(), cacheKey, data, cacheTTL)
	response.Success(c, data)
}

// GetCourse 获取课程详情
func (h *Handler) GetCourse(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}
	cacheKey := "course:course:" + strconv.FormatInt(courseID, 10)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), courseID)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			response.NotFound(c, "course not found")
			return
		}
		response.InternalError(c, "failed to load course")
		return
	}

	_ = h.setCache(c.Request.Context(), cacheKey, course, cacheTTL)
	response.Success(c, course)
}

// GetStats 获取学习中心统计数据
func (h *Handler) GetStats(c *gin.Context) {
	cacheKey := "course:stats"
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
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
	_ = h.setCache(c.Request.Context(), cacheKey, data, cacheTTL)
	response.Success(c, data)
}
