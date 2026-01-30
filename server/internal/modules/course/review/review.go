package review

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetCourseReviews 获取课程测评列表
func (h *Handler) GetCourseReviews(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}
	page, pageSize := parsePage(c)

	// 检查缓存
	cacheKey := h.buildCacheKey(c.Request.Context(), "review:course", strconv.FormatInt(courseID, 10)+":page="+strconv.Itoa(page)+":size="+strconv.Itoa(pageSize))
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	result, err := h.service.GetCourseReviews(c.Request.Context(), GetCourseReviewsParams{
		CourseID: courseID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load reviews")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	_ = h.setCache(c.Request.Context(), cacheKey, data, cacheTTL)
	response.Success(c, data)
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	page, pageSize := parsePage(c)

	// 检查缓存
	cacheKey := h.buildCacheKey(c.Request.Context(), "review:latest", "page="+strconv.Itoa(page)+":size="+strconv.Itoa(pageSize))
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	result, err := h.service.GetLatestReviews(c.Request.Context(), GetLatestReviewsParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load latest reviews")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	_ = h.setCache(c.Request.Context(), cacheKey, data, cacheTTL)
	response.Success(c, data)
}

// PostReviewRequest 发布测评请求
type PostReviewRequest struct {
	CourseID  int64         `json:"course_id" binding:"required,gt=0"`
	TeacherID *int64        `json:"teacher_id" binding:"omitempty,gt=0"`
	TermID    string        `json:"term_id" binding:"omitempty,max=20"`
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
	userHash := hashUserID(userID)

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
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrRatingRequired):
			response.BadRequest(c, "at least one rating dimension is required")
		case errors.Is(err, ErrInvalidRating):
			response.BadRequest(c, "rating must be between 1 and 5")
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		case errors.Is(err, ErrCourseNotFound):
			response.NotFound(c, "course not found")
		default:
			response.InternalError(c, "failed to create review")
		}
		return
	}

	// 失效相关缓存
	ctx := c.Request.Context()
	_ = h.invalidateCache(ctx, "review:course")
	_ = h.invalidateCache(ctx, "review:latest")
	_ = h.invalidateCache(ctx, "review:stats")

	response.Created(c, gin.H{"message": "review published successfully", "id": result.ID})
}

// VoteReview 投票
func (h *Handler) VoteReview(c *gin.Context) {
	var req struct {
		VoteType string `json:"vote_type" binding:"required,oneof=like dislike"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	reviewID := c.Param("id")
	userID := middleware.GetUserID(c)
	userHash := hashUserID(userID)

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
		case errors.Is(err, ErrAlreadyVoted):
			response.Conflict(c, "already voted")
		default:
			response.InternalError(c, "failed to vote")
		}
		return
	}

	// 失效相关缓存
	ctx := c.Request.Context()
	_ = h.invalidateCache(ctx, "review:course")
	_ = h.invalidateCache(ctx, "review:latest")

	response.Success(c, gin.H{"message": "vote submitted successfully"})
}

// GetStats 获取评课统计数据
func (h *Handler) GetStats(c *gin.Context) {
	// 检查缓存
	cacheKey := h.buildCacheKey(c.Request.Context(), "review:stats", "all")
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	result, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to load stats")
		return
	}

	data := gin.H{"reviewCount": result.ReviewCount}
	_ = h.setCache(c.Request.Context(), cacheKey, data, cacheTTL)
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
			&item.LikeCount, &item.DislikeCount, &item.Status, &item.CreatedAt,
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
