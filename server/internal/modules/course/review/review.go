package review

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// GetCourseReviews 获取课程测评列表
func (h *Handler) GetCourseReviews(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	page, pageSize := parsePage(c)
	cacheKey := "review:course:" + strconv.FormatInt(courseID, 10) + ":page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()
	countCacheKey := "review:course:" + strconv.FormatInt(courseID, 10) + ":count"
	total, err := h.countWithCache(ctx, countCacheKey, `SELECT COUNT(*) FROM reviews WHERE course_id = $1`, courseID)
	if err != nil {
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

	list, err := scanReviews(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse reviews"})
		return
	}

	resp := gin.H{"data": gin.H{"list": list, "total": total}}
	_ = h.setCache(ctx, cacheKey, resp, cacheTTL)
	c.JSON(http.StatusOK, resp)
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := parsePage(c)
	cacheKey := "review:latest:page=" + strconv.Itoa(page) + ":size=" + strconv.Itoa(pageSize)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()
	total, err := h.countWithCache(ctx, "review:total:count", `SELECT COUNT(*) FROM reviews`)
	if err != nil {
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

	list, err := scanReviews(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse reviews"})
		return
	}

	resp := gin.H{"data": gin.H{"list": list, "total": total}}
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

	if len(req.Ratings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one rating dimension is required", "code": "RATING_REQUIRED"})
		return
	}
	for _, v := range req.Ratings {
		if v < 1 || v > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5", "code": "INVALID_RATING"})
			return
		}
	}

	ctx := c.Request.Context()

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
	`, reviewID, req.CourseID, req.TeacherID, req.TermID, req.Title,
		req.Content, req.Grade, data, userHash, "published")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	_, err = tx.Exec(ctx, `UPDATE courses SET review_count = review_count + 1 WHERE id = $1`, req.CourseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update review count"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	// 精确失效相关缓存，而非清除所有 review: 前缀
	_ = h.invalidateCache(ctx, "review:course:"+strconv.FormatInt(req.CourseID, 10))
	_ = h.invalidateCache(ctx, "review:latest:")
	_ = h.invalidateCache(ctx, "review:stats")

	c.JSON(http.StatusOK, gin.H{"message": "review published successfully", "id": reviewID})
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

	result, err := tx.Exec(ctx, `
		INSERT INTO review_votes (review_id, user_hash, vote_type, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT DO NOTHING
	`, reviewID, userHash, req.VoteType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to vote"})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "already voted"})
		return
	}

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

	// 投票只影响具体评论的展示，精确失效相关缓存
	// 由于评论列表包含 like_count/dislike_count，需要失效所有评论列表缓存
	_ = h.invalidateCache(ctx, "review:course:")
	_ = h.invalidateCache(ctx, "review:latest:")

	c.JSON(http.StatusOK, gin.H{"message": "vote submitted successfully"})
}

// GetStats 获取评课统计数据
func (h *Handler) GetStats(c *gin.Context) {
	cacheKey := "review:stats"
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	ctx := c.Request.Context()
	reviewCount, err := h.countWithCache(ctx, "review:total:count", "SELECT COUNT(*) FROM reviews")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stats"})
		return
	}

	data := gin.H{"reviewCount": reviewCount}
	_ = h.setCache(ctx, cacheKey, data, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// scanReviews 扫描评论行
func scanReviews(rows interface{ Next() bool; Scan(...interface{}) error }) ([]Review, error) {
	list := make([]Review, 0)
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.Status, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}
