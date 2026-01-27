package review

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Handler 评课社区处理器
type Handler struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewHandler 创建处理器
func NewHandler(db *pgxpool.Pool, cache *redis.Client) *Handler {
	return &Handler{db: db, cache: cache}
}

// RegisterRoutes 注册评课社区路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// 评分维度配置
	r.GET("/rating-dimensions", h.GetRatingDimensions)

	// 课程评分统计
	r.GET("/courses/:id/rating-stats", h.GetCourseRatingStats)

	// 测评
	r.GET("/courses/:id/reviews", h.GetCourseReviews)
	r.GET("/reviews/latest", h.GetLatestReviews)
	r.POST("/reviews", authMiddleware, h.PostReview)
	r.POST("/reviews/:id/vote", authMiddleware, h.VoteReview)

	// 评课统计
	r.GET("/stats", h.GetStats)
}
