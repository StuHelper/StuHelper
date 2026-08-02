package app

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/modules/course"
	"github.com/StuHelper/StuHelper/server/internal/modules/course/review"
	"github.com/StuHelper/StuHelper/server/internal/modules/notification"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

type courseModule struct {
	courseHandler *course.Handler
	reviewHandler *review.Handler
}

func (rt *Runtime) registerMetricsRoutes(api *gin.RouterGroup) {
	metricsGroup := api.Group("/metrics")
	metricsGroup.Use(metrics.OriginValidationMiddleware(rt.metricsAllowedOrigins()))
	metricsGroup.POST("/vitals", metrics.VitalsHandler())
	metricsGroup.POST("/frontend-errors", metrics.FrontendErrorHandler())
}

func (rt *Runtime) initCourseModule(
	ctx context.Context,
	authorizer review.AuthorizationProvider,
	notifSender notification.Sender,
	userRepo *user.Repository,
) courseModule {
	courseCache := cache.NewHelperWithNamespace(rt.redisClient.GetClient(), cache.NamespaceCourse)
	reviewCache := cache.NewHelperWithNamespace(rt.redisClient.GetClient(), cache.NamespaceReview)
	reviewRepo := review.NewRepository(
		rt.database,
		review.WithTeacherStatsRefreshTimeout(
			time.Duration(rt.cfg.Review.TeacherStatsRefreshTimeoutSeconds)*time.Second,
		),
	)
	reviewService := review.NewService(
		rt.database,
		reviewRepo,
		newReviewNotificationAdapter(notifSender),
		authorizer,
		userRepo,
		review.WithInitialCacheContext(ctx),
	)
	reviewHandler := review.NewHandler(review.HandlerConfig{
		CacheHelper:            reviewCache,
		CourseCacheHelper:      courseCache,
		Service:                reviewService,
		Redis:                  rt.redisClient.GetClient(),
		RateLimit:              rt.cfg.RateLimit,
		Authorizer:             authorizer,
		InternalUserIDResolver: userRepo.GetInternalUserID,
		AdminAuthorizers:       reviewAdminAuthorizers(),
	})

	courseRepo := course.NewRepository(rt.database)
	courseService := course.NewService(courseRepo, logger.L().Named("course_service"))
	return courseModule{
		courseHandler: course.NewHandler(courseCache, courseService),
		reviewHandler: reviewHandler,
	}
}

func (m courseModule) RegisterRoutes(
	api *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
	optionalAuthMiddleware gin.HandlerFunc,
	adminRouteSecurity review.AdminRouteSecurity,
) {
	m.courseHandler.RegisterRoutes(api, authMiddleware, optionalAuthMiddleware)
	m.reviewHandler.RegisterRoutes(api.Group("/course/review"), authMiddleware, optionalAuthMiddleware, adminRouteSecurity)
}

func (m courseModule) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	m.reviewHandler.StartBackgroundJobs(ctx, start)
	start("course operation log cleanup", m.runLogCleanupLoop)
	start("course teacher public stats refresh", m.runTeacherPublicStatsRefreshLoop)
}

func (m courseModule) runLogCleanupLoop(ctx context.Context) {
	const cleanupInterval = 24 * time.Hour
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	m.runLogCleanup(ctx)

	for {
		select {
		case <-ticker.C:
			m.runLogCleanup(ctx)
		case <-ctx.Done():
			logger.L().Info("Background jobs stopped")
			return
		}
	}
}

func (m courseModule) runTeacherPublicStatsRefreshLoop(ctx context.Context) {
	const refreshInterval = 10 * time.Minute
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	m.runTeacherPublicStatsRefresh(ctx)

	for {
		select {
		case <-ticker.C:
			m.runTeacherPublicStatsRefresh(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (m courseModule) runLogCleanup(ctx context.Context) {
	deleted, err := m.reviewHandler.CleanupOldLogs(ctx)
	if err != nil {
		logger.L().Error("Failed to cleanup old operation logs", zap.Error(err))
		return
	}
	logger.L().Info("Operation logs cleanup completed", zap.Int64("deleted_count", deleted))
}

func (m courseModule) runTeacherPublicStatsRefresh(ctx context.Context) {
	if err := m.reviewHandler.EnqueueTeacherPublicStatsRefresh(ctx); err != nil {
		logger.L().Warn("Failed to enqueue teacher public stats projection refresh", zap.Error(err))
		return
	}
	logger.L().Debug("Teacher public stats projection refresh enqueued")
}

func (rt *Runtime) metricsAllowedOrigins() []string {
	if len(rt.cfg.Observability.FrontendMetricsAllowedOrigins) > 0 {
		return append([]string(nil), rt.cfg.Observability.FrontendMetricsAllowedOrigins...)
	}
	if len(rt.cfg.App.CORSOrigins) > 0 {
		return append([]string(nil), rt.cfg.App.CORSOrigins...)
	}
	if rt.isProduction {
		return nil
	}
	return []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:4173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:4173",
	}
}
