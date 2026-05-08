package app

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

func (rt *Runtime) registerMetricsRoutes(api *gin.RouterGroup) {
	metricsGroup := api.Group("/metrics")
	metricsGroup.Use(metrics.OriginValidationMiddleware(rt.metricsAllowedOrigins()))
	metricsGroup.POST("/vitals", metrics.VitalsHandler())
	metricsGroup.POST("/frontend-errors", metrics.FrontendErrorHandler())
}

func (rt *Runtime) initCourseModule(
	authorizer review.AuthorizationProvider,
	notifSender notification.Sender,
	userRepo *user.Repository,
) *course.Handler {
	courseCache := cache.NewHelper(rt.redisClient.GetClient())
	reviewRepo := review.NewRepository(rt.database)
	reviewService := review.NewService(rt.database, reviewRepo, notifSender, authorizer, userRepo)
	reviewHandler := review.NewHandler(review.HandlerConfig{
		CacheHelper:            courseCache,
		Service:                reviewService,
		Redis:                  rt.redisClient.GetClient(),
		RateLimit:              rt.cfg.RateLimit,
		Authorizer:             authorizer,
		InternalUserIDResolver: userRepo.GetInternalUserID,
	})

	courseRepo := course.NewRepository(rt.database)
	courseService := course.NewService(courseRepo, logger.L().Named("course_service"))
	return course.NewHandler(courseCache, courseService, reviewHandler)
}

func (rt *Runtime) metricsAllowedOrigins() []string {
	if len(rt.cfg.App.CORSOrigins) > 0 {
		return rt.cfg.App.CORSOrigins
	}
	if rt.isProduction {
		return nil
	}
	return []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:4173",
	}
}
