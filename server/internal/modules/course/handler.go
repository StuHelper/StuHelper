package course

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
)

// Handler 学习中心处理器
type Handler struct {
	db            *pgxpool.Pool
	cache         *redis.Client
	reviewHandler *review.Handler
}

// NewHandler 创建处理器
func NewHandler(db *pgxpool.Pool, cache *redis.Client) *Handler {
	return &Handler{
		db:            db,
		cache:         cache,
		reviewHandler: review.NewHandler(db, cache),
	}
}

// RegisterRoutes 注册学习中心路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	course := r.Group("/course")
	{
		// 通用实体接口（课程、院系等）
		course.GET("/departments", h.GetDepartments)
		course.GET("/courses", h.GetCourses)
		course.GET("/courses/search", h.SearchCourses)
		course.GET("/courses/:id", h.GetCourse)
		course.GET("/stats", h.GetStats)

		// 评课社区子模块
		reviewGroup := course.Group("/review")
		h.reviewHandler.RegisterRoutes(reviewGroup, authMiddleware)
	}
}
