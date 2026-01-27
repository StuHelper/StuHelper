package course

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// GetDepartments 获取院系列表
func (h *Handler) GetDepartments(c *gin.Context) {
	category := c.Query("category")
	cacheKey := "course:departments:" + sanitizeCacheKey(category)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT id, school_id, name, short_name, category, sort_order
		FROM departments
		WHERE ($1 = '' OR category = $1)
		ORDER BY sort_order ASC
	`, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load departments"})
		return
	}
	defer rows.Close()

	departments := make([]Department, 0)
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.SchoolID, &d.Name, &d.ShortName, &d.Category, &d.SortOrder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse departments"})
			return
		}
		departments = append(departments, d)
	}

	_ = h.setCache(c.Request.Context(), cacheKey, departments, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": departments})
}

// GetCourses 获取课程列表
func (h *Handler) GetCourses(c *gin.Context) {
	page, pageSize := parsePage(c)
	cacheKey := "course:courses:page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()
	total, err := h.count(ctx, "SELECT COUNT(*) FROM courses")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load courses"})
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		ORDER BY c.name ASC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load courses"})
		return
	}
	defer rows.Close()

	list := make([]Course, 0)
	for rows.Next() {
		var item Course
		if err := rows.Scan(
			&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
			&item.Code, &item.Name, &item.Credits, &item.ReviewCount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse courses"})
			return
		}
		list = append(list, item)
	}

	resp := gin.H{"data": gin.H{"list": list, "total": total}}
	_ = h.setCache(ctx, cacheKey, resp, cacheTTL)
	c.JSON(http.StatusOK, resp)
}

// SearchCourses 搜索课程
func (h *Handler) SearchCourses(c *gin.Context) {
	q := c.Query("q")
	if len(q) > maxSearchLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query too long"})
		return
	}
	page, pageSize := parsePage(c)
	cacheKey := "course:courses:search:" + sanitizeCacheKey(q) + ":page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()
	qLike := "%" + escapeLikePattern(q) + "%"

	var total int
	err := h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM courses WHERE name ILIKE $1 OR code ILIKE $1
	`, qLike).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search courses"})
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE c.name ILIKE $1 OR c.code ILIKE $1
		ORDER BY c.name ASC
		LIMIT $2 OFFSET $3
	`, qLike, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search courses"})
		return
	}
	defer rows.Close()

	list := make([]Course, 0)
	for rows.Next() {
		var item Course
		if err := rows.Scan(
			&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
			&item.Code, &item.Name, &item.Credits, &item.ReviewCount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse courses"})
			return
		}
		list = append(list, item)
	}

	resp := gin.H{"data": gin.H{"list": list, "total": total}}
	_ = h.setCache(ctx, cacheKey, resp, cacheTTL)
	c.JSON(http.StatusOK, resp)
}

// GetCourse 获取课程详情
func (h *Handler) GetCourse(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	cacheKey := "course:course:" + strconv.FormatInt(courseID, 10)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	var item Course
	err = h.db.QueryRow(c.Request.Context(), `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE c.id = $1
	`, courseID).Scan(
		&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
		&item.Code, &item.Name, &item.Credits, &item.ReviewCount,
	)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load course"})
		return
	}

	_ = h.setCache(c.Request.Context(), cacheKey, item, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": item})
}

// GetStats 获取学习中心统计数据
func (h *Handler) GetStats(c *gin.Context) {
	cacheKey := "course:stats"
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	ctx := c.Request.Context()
	courseCount, err := h.count(ctx, "SELECT COUNT(*) FROM courses")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stats"})
		return
	}
	departmentCount, err := h.count(ctx, "SELECT COUNT(*) FROM departments")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stats"})
		return
	}

	data := gin.H{
		"courseCount":     courseCount,
		"departmentCount": departmentCount,
	}
	_ = h.setCache(ctx, cacheKey, data, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": data})
}
