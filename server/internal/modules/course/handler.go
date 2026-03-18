package course

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// Handler 学习中心处理器
type Handler struct {
	db            *db.DB
	cache         *cache.Helper
	service       *Service
	reviewHandler *review.Handler
}

// NewHandler 创建处理器
func NewHandler(database *db.DB, rdb *redis.Client, permissionSvc rbac.PermissionService, cfg *config.Config) *Handler {
	repo := NewRepository(database)
	svc := NewService(database, repo)
	return &Handler{
		db:            database,
		cache:         cache.NewHelper(rdb),
		service:       svc,
		reviewHandler: review.NewHandler(database, rdb, permissionSvc, cfg.RateLimit),
	}
}

// RegisterRoutes 注册学习中心路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware, optionalAuthMiddleware gin.HandlerFunc) {
	course := r.Group("/course")
	{
		// 通用实体接口（课程、院系等）
		course.GET("/departments", h.GetDepartments)
		course.GET("/terms", h.GetTerms)
		course.GET("/categories", h.GetCourseCategories)
		course.GET("/courses", h.GetCourses)
		course.GET("/courses/search", h.SearchCourses)
		course.GET("/courses/:id", h.GetCourse)
		course.GET("/stats", h.GetStats)

		// 评课社区子模块
		reviewGroup := course.Group("/review")
		h.reviewHandler.RegisterRoutes(reviewGroup, authMiddleware, optionalAuthMiddleware)
	}
}

// StartBackgroundJobs 启动后台定时任务（日志清理等）
// 调用方需传入可取消的 context，用于优雅关闭时停止后台任务
func (h *Handler) StartBackgroundJobs(ctx context.Context) {
	go func() {
		const cleanupInterval = 24 * time.Hour
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		// 启动时立即执行一次清理
		h.runLogCleanup(ctx)

		for {
			select {
			case <-ticker.C:
				h.runLogCleanup(ctx)
			case <-ctx.Done():
				logger.L().Info("Background jobs stopped")
				return
			}
		}
	}()
}

// runLogCleanup 执行操作日志清理
func (h *Handler) runLogCleanup(ctx context.Context) {
	deleted, err := h.reviewHandler.CleanupOldLogs(ctx)
	if err != nil {
		logger.L().Error("Failed to cleanup old operation logs",
			zap.Error(err),
		)
		return
	}
	logger.L().Info("Operation logs cleanup completed",
		zap.Int64("deleted_count", deleted),
	)
}
