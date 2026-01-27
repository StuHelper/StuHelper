package course

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	maxSearchLength = 100
	cacheTTL        = 5 * time.Minute
)

// Handler 课程评价处理器
type Handler struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewHandler 创建处理器
func NewHandler(db *pgxpool.Pool, cache *redis.Client) *Handler {
	return &Handler{db: db, cache: cache}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	course := r.Group("/course-review")
	{
		// 评分维度配置
		course.GET("/rating-dimensions", h.GetRatingDimensions)

		// 院系和课程
		course.GET("/departments", h.GetDepartments)
		course.GET("/courses", h.GetCourses)
		course.GET("/courses/search", h.SearchCourses)
		course.GET("/courses/:id", h.GetCourse)
		course.GET("/courses/:id/rating-stats", h.GetCourseRatingStats)

		// 测评
		course.GET("/courses/:id/reviews", h.GetCourseReviews)
		course.GET("/reviews/latest", h.GetLatestReviews)
		course.POST("/reviews", authMiddleware, h.PostReview)
		course.POST("/reviews/:id/vote", authMiddleware, h.VoteReview)

		// 统计
		course.GET("/stats", h.GetStats)
	}
}

// GetRatingDimensions 获取评分维度配置
func (h *Handler) GetRatingDimensions(c *gin.Context) {
	cacheKey := "course:rating_dimensions"
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT id, school_id, key, name, description, sort_order, is_active, created_at, updated_at
		FROM rating_dimensions
		WHERE is_active = true
		ORDER BY sort_order ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rating dimensions"})
		return
	}
	defer rows.Close()

	dimensions := make([]RatingDimension, 0)
	for rows.Next() {
		var d RatingDimension
		if err := rows.Scan(
			&d.ID,
			&d.SchoolID,
			&d.Key,
			&d.Name,
			&d.Description,
			&d.SortOrder,
			&d.IsActive,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse rating dimensions"})
			return
		}
		dimensions = append(dimensions, d)
	}

	_ = h.setCache(c.Request.Context(), cacheKey, dimensions, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": dimensions})
}

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
			&item.ID,
			&item.SchoolID,
			&item.DepartmentID,
			&item.DepartmentName,
			&item.Code,
			&item.Name,
			&item.Credits,
			&item.ReviewCount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse courses"})
			return
		}
		list = append(list, item)
	}

	resp := gin.H{
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	}
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
		SELECT COUNT(*)
		FROM courses
		WHERE name ILIKE $1 OR code ILIKE $1
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
			&item.ID,
			&item.SchoolID,
			&item.DepartmentID,
			&item.DepartmentName,
			&item.Code,
			&item.Name,
			&item.Credits,
			&item.ReviewCount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse courses"})
			return
		}
		list = append(list, item)
	}

	resp := gin.H{
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	}
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
		&item.ID,
		&item.SchoolID,
		&item.DepartmentID,
		&item.DepartmentName,
		&item.Code,
		&item.Name,
		&item.Credits,
		&item.ReviewCount,
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

// GetCourseRatingStats 获取课程评分统计（雷达图数据）
func (h *Handler) GetCourseRatingStats(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	cacheKey := "course:rating_stats:" + strconv.FormatInt(courseID, 10)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	ctx := c.Request.Context()
	dimensionNames, err := h.loadDimensionNames(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dimension names"})
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT term_id, dimension_key, avg_rating, rating_count, rating_dist
		FROM course_rating_stats
		WHERE course_id = $1
		ORDER BY term_id NULLS FIRST
	`, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rating stats"})
		return
	}
	defer rows.Close()

	byTerm := make(map[string]*TermRatingStats)
	var overall TermRatingStats
	allKeysSet := make(map[string]bool)

	for rows.Next() {
		var termID *string
		var key string
		var avg float64
		var count int
		var distJSON []byte
		if err := rows.Scan(&termID, &key, &avg, &count, &distJSON); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse rating stats"})
			return
		}

		allKeysSet[key] = true
		dist := map[int]int{}
		if err := json.Unmarshal(distJSON, &dist); err != nil {
			logger.L().Warn("failed to unmarshal rating distribution",
				zap.String("dimension_key", key),
				zap.Error(err),
			)
		}

		ds := DimensionStats{
			Key:          key,
			Name:         dimensionNames[key],
			AvgRating:    avg,
			RatingCount:  count,
			Distribution: dist,
		}

		if termID == nil {
			overall.Dimensions = append(overall.Dimensions, ds)
			continue
		}

		term, ok := byTerm[*termID]
		if !ok {
			term = &TermRatingStats{TermID: *termID, TermName: *termID}
			byTerm[*termID] = term
		}
		term.Dimensions = append(term.Dimensions, ds)
	}

	allKeys := make([]string, 0, len(allKeysSet))
	for k := range allKeysSet {
		allKeys = append(allKeys, k)
	}

	byTermList := make([]TermRatingStats, 0, len(byTerm))
	for _, v := range byTerm {
		byTermList = append(byTermList, *v)
	}

	response := CourseRatingStatsResponse{
		CourseID:         courseID,
		Overall:          overall,
		ByTerm:           byTermList,
		AllDimensionKeys: allKeys,
		RadarChart:       buildRadarChart(allKeys, dimensionNames, overall),
	}

	_ = h.setCache(ctx, cacheKey, response, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetCourseReviews 获取课程测评列表
func (h *Handler) GetCourseReviews(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	page, pageSize := parsePage(c)
	cacheKey := "course:reviews:" + strconv.FormatInt(courseID, 10) + ":page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE course_id = $1`, courseID).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load reviews"})
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id, r.title, r.content, r.grade,
		       r.ratings, r.like_count, r.dislike_count, r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.course_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`, courseID, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load reviews"})
		return
	}
	defer rows.Close()

	list := make([]Review, 0)
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID,
			&item.CourseID,
			&item.CourseName,
			&item.TeacherID,
			&item.TeacherName,
			&item.TermID,
			&item.Title,
			&item.Content,
			&item.Grade,
			&item.Ratings,
			&item.LikeCount,
			&item.DislikeCount,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse reviews"})
			return
		}
		list = append(list, item)
	}

	resp := gin.H{
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	}
	_ = h.setCache(ctx, cacheKey, resp, cacheTTL)
	c.JSON(http.StatusOK, resp)
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := parsePage(c)
	cacheKey := "course:reviews:latest:page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews`).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load latest reviews"})
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id, r.title, r.content, r.grade,
		       r.ratings, r.like_count, r.dislike_count, r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load latest reviews"})
		return
	}
	defer rows.Close()

	list := make([]Review, 0)
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID,
			&item.CourseID,
			&item.CourseName,
			&item.TeacherID,
			&item.TeacherName,
			&item.TermID,
			&item.Title,
			&item.Content,
			&item.Grade,
			&item.Ratings,
			&item.LikeCount,
			&item.DislikeCount,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse reviews"})
			return
		}
		list = append(list, item)
	}

	resp := gin.H{
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	}
	_ = h.setCache(ctx, cacheKey, resp, cacheTTL)
	c.JSON(http.StatusOK, resp)
}

// PostReviewRequest 发布测评请求
type PostReviewRequest struct {
	CourseID  int64         `json:"course_id" binding:"required"`
	TeacherID *int64        `json:"teacher_id"`
	TermID    string        `json:"term_id"`
	Title     string        `json:"title" binding:"max=200"`
	Content   string        `json:"content" binding:"required,min=10,max=5000"`
	Grade     string        `json:"grade"`
	Ratings   ReviewRatings `json:"ratings" binding:"required"`
}

// PostReview 发布测评
func (h *Handler) PostReview(c *gin.Context) {
	var req PostReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证评分维度
	if len(req.Ratings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一个评分维度"})
		return
	}
	for _, v := range req.Ratings {
		if v < 1 || v > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "评分必须在1-5之间"})
			return
		}
	}

	ctx := c.Request.Context()

	// 验证课程是否存在
	var courseExists bool
	err := h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, req.CourseID).Scan(&courseExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify course"})
		return
	}
	if !courseExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}

	userID := middleware.GetUserID(c)
	userHash := hashUserID(userID)

	data, err := json.Marshal(req.Ratings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode ratings"})
		return
	}

	reviewID := uuid.NewString()

	// 使用事务确保插入评论和更新计数的原子性
	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO reviews (
			id, course_id, teacher_id, term_id, title, content, grade,
			ratings, user_hash, status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
	`,
		reviewID,
		req.CourseID,
		req.TeacherID,
		req.TermID,
		req.Title,
		req.Content,
		req.Grade,
		data,
		userHash,
		"published",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	// 更新课程评论计数
	_, err = tx.Exec(ctx, `UPDATE courses SET review_count = review_count + 1 WHERE id = $1`, req.CourseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update review count"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	_ = h.invalidateCache(c.Request.Context(), "course:reviews:")
	_ = h.invalidateCache(c.Request.Context(), "course:reviews:latest")
	_ = h.invalidateCache(c.Request.Context(), "course:rating_stats:")
	_ = h.invalidateCache(c.Request.Context(), "course:stats")

	c.JSON(http.StatusOK, gin.H{
		"message": "发布成功",
		"id":      reviewID,
	})
}

// VoteReview 投票
func (h *Handler) VoteReview(c *gin.Context) {
	var req struct {
		VoteType string `json:"vote_type" binding:"required,oneof=like dislike"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewID := c.Param("id")
	userID := middleware.GetUserID(c)
	userHash := hashUserID(userID)

	ctx := c.Request.Context()

	// 验证评论是否存在
	var reviewExists bool
	err := h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE id = $1)`, reviewID).Scan(&reviewExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify review"})
		return
	}
	if !reviewExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to vote"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 插入投票记录，检查是否实际插入
	result, err := tx.Exec(ctx, `
		INSERT INTO review_votes (review_id, user_hash, vote_type, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT DO NOTHING
	`, reviewID, userHash, req.VoteType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to vote"})
		return
	}

	// 检查是否实际插入了记录（防止重复投票）
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "already voted"})
		return
	}

	// 只有实际插入了记录才更新计数
	if req.VoteType == "like" {
		_, err = tx.Exec(ctx, `UPDATE reviews SET like_count = like_count + 1 WHERE id = $1`, reviewID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE reviews SET dislike_count = dislike_count + 1 WHERE id = $1`, reviewID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vote count"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to vote"})
		return
	}

	_ = h.invalidateCache(ctx, "course:reviews:")
	_ = h.invalidateCache(ctx, "course:reviews:latest")

	c.JSON(http.StatusOK, gin.H{"message": "投票成功"})
}

// GetStats 获取统计数据
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
	reviewCount, err := h.count(ctx, "SELECT COUNT(*) FROM reviews")
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
		"reviewCount":     reviewCount,
		"departmentCount": departmentCount,
	}
	_ = h.setCache(ctx, cacheKey, data, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getCache(ctx context.Context, key string) (interface{}, bool) {
	if h.cache == nil {
		return nil, false
	}
	data, err := h.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, false
	}
	return v, true
}

func (h *Handler) setCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if h.cache == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		logger.L().Warn("failed to marshal cache value",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	if err := h.cache.Set(ctx, key, data, ttl).Err(); err != nil {
		logger.L().Warn("failed to set cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (h *Handler) invalidateCache(ctx context.Context, prefix string) error {
	if h.cache == nil {
		return nil
	}

	// 限制扫描数量，避免在大规模数据时阻塞 Redis
	const maxKeysToDelete = 1000
	var keys []string
	var cursor uint64
	for {
		var batch []string
		var err error
		batch, cursor, err = h.cache.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			logger.L().Warn("failed to scan cache keys",
				zap.String("prefix", prefix),
				zap.Error(err),
			)
			return err
		}
		keys = append(keys, batch...)
		if cursor == 0 || len(keys) >= maxKeysToDelete {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	// 使用 Pipeline 批量删除
	pipe := h.cache.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.L().Warn("failed to invalidate cache",
			zap.String("prefix", prefix),
			zap.Int("key_count", len(keys)),
			zap.Error(err),
		)
	}
	return err
}

func (h *Handler) count(ctx context.Context, query string) (int, error) {
	var total int
	if err := h.db.QueryRow(ctx, query).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func parseIDParam(c *gin.Context, name string) (int64, error) {
	idStr := c.Param(name)
	return strconv.ParseInt(idStr, 10, 64)
}

func hashUserID(userID string) string {
	if userID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(userID))
	return hex.EncodeToString(sum[:])
}

// escapeLikePattern 转义 LIKE 查询中的特殊字符
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// sanitizeCacheKey 清理缓存 key 中的特殊字符，防止缓存 key 注入
func sanitizeCacheKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8]) // 使用前8字节的哈希作为 key
}

func (h *Handler) loadDimensionNames(ctx context.Context) (map[string]string, error) {
	// 尝试从缓存获取
	cacheKey := "course:dimension_names"
	if cached, ok := h.getCache(ctx, cacheKey); ok {
		if m, ok := cached.(map[string]interface{}); ok {
			result := make(map[string]string, len(m))
			for k, v := range m {
				if s, ok := v.(string); ok {
					result[k] = s
				}
			}
			return result, nil
		}
	}

	rows, err := h.db.Query(ctx, `SELECT key, name FROM rating_dimensions WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var key, name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, err
		}
		result[key] = name
	}

	// 缓存结果（维度名称变化不频繁，可以缓存较长时间）
	_ = h.setCache(ctx, cacheKey, result, 30*time.Minute)

	return result, nil
}

func buildRadarChart(keys []string, names map[string]string, overall TermRatingStats) RadarChartData {
	labels := make([]string, 0, len(keys))
	data := make([]float64, 0, len(keys))
	statMap := make(map[string]DimensionStats)
	for _, d := range overall.Dimensions {
		statMap[d.Key] = d
	}
	for _, k := range keys {
		labels = append(labels, names[k])
		data = append(data, statMap[k].AvgRating)
	}

	return RadarChartData{
		Labels: labels,
		Datasets: []RadarChartDataset{
			{
				Label:           "总体",
				Data:            data,
				BackgroundColor: "rgba(64, 158, 255, 0.2)",
				BorderColor:     "#409EFF",
			},
		},
	}
}
