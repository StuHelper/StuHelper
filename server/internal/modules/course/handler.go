package course

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
)

// Handler 学习中心处理器
type Handler struct {
	db            *db.DB
	cache         *cache.Helper
	service       *Service
	reviewHandler *review.Handler
}

// NewHandler 创建处理器
func NewHandler(database *db.DB, rdb *redis.Client, ssoClient *sso.Client) *Handler {
	repo := NewRepository(database)
	svc := NewService(database, repo)
	return &Handler{
		db:            database,
		cache:         cache.NewHelper(rdb),
		service:       svc,
		reviewHandler: review.NewHandler(database, rdb, ssoClient),
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
