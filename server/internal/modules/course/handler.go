package course

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// Handler 学习中心处理器
type Handler struct {
	cache         *cache.Helper
	service       *Service
	reviewHandler *review.Handler
}

// NewHandler 创建处理器
func NewHandler(cacheHelper *cache.Helper, service *Service, reviewHandler *review.Handler) *Handler {
	if cacheHelper == nil {
		panic("course.NewHandler: cacheHelper must not be nil")
	}
	if service == nil {
		panic("course.NewHandler: service must not be nil")
	}
	if reviewHandler == nil {
		panic("course.NewHandler: reviewHandler must not be nil")
	}
	return &Handler{
		cache:         cacheHelper,
		service:       service,
		reviewHandler: reviewHandler,
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
		course.GET("/courses", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourses)
		course.GET("/courses/grouped", h.GetCoursesGrouped)
		course.GET("/courses/search", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.SearchCourses)
		course.GET("/courses/:courseID", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourse)
		course.GET("/stats", h.GetStats)

		// 评课社区子模块
		reviewGroup := course.Group("/review")
		h.reviewHandler.RegisterRoutes(reviewGroup, authMiddleware, optionalAuthMiddleware)
	}
}

// StartBackgroundJobs 启动后台定时任务（日志清理等）。
// 调用方可传入 start 统一托管 goroutine 生命周期。
func (h *Handler) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	h.reviewHandler.StartBackgroundJobs(ctx, start)

	launch := start
	if launch == nil {
		launch = func(name string, run func(context.Context)) {
			go run(ctx)
		}
	}

	launch("course operation log cleanup", h.runLogCleanupLoop)
	launch("course teacher public stats refresh", h.runTeacherPublicStatsRefreshLoop)
}

func (h *Handler) runLogCleanupLoop(ctx context.Context) {
	const cleanupInterval = 24 * time.Hour
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

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
}

func (h *Handler) runTeacherPublicStatsRefreshLoop(ctx context.Context) {
	const refreshInterval = 10 * time.Minute
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	h.runTeacherPublicStatsRefresh(ctx)

	for {
		select {
		case <-ticker.C:
			h.runTeacherPublicStatsRefresh(ctx)
		case <-ctx.Done():
			return
		}
	}
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

func (h *Handler) runTeacherPublicStatsRefresh(ctx context.Context) {
	if err := h.reviewHandler.RefreshTeacherPublicStats(ctx); err != nil {
		logger.L().Warn("Failed to refresh teacher public stats materialized view", zap.Error(err))
		return
	}
	logger.L().Debug("Teacher public stats materialized view refreshed")
}
