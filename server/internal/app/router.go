package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	gozap "go.uber.org/zap"

	apidocs "git.stuhelper.com/StuHelper/StuHelper/api"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func (rt *Runtime) buildRouter() (*gin.Engine, error) {
	if rt.isProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := rt.configureTrustedProxies(r); err != nil {
		return nil, err
	}

	rt.registerGlobalMiddleware(r)
	if err := rt.registerPlatformRoutes(r); err != nil {
		return nil, err
	}

	bgCtx, bgCancel := context.WithCancel(context.Background()) //nolint:gosec // managed by Runtime lifecycle
	rt.bgCancel = bgCancel
	if err := rt.registerAPIRoutes(r, bgCtx); err != nil {
		return nil, err
	}

	return r, nil
}

func (rt *Runtime) configureTrustedProxies(r *gin.Engine) error {
	if len(rt.cfg.App.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(rt.cfg.App.TrustedProxies); err != nil {
			return fmt.Errorf("failed to set trusted proxies: %w", err)
		}
		return nil
	}

	if err := r.SetTrustedProxies(nil); err != nil {
		logger.L().Warn("failed to set trusted proxies to nil", gozap.Error(err))
	}
	return nil
}

func (rt *Runtime) registerGlobalMiddleware(r *gin.Engine) {
	r.Use(middleware.RequestIDMiddleware())
	r.Use(otelgin.Middleware(rt.cfg.Observability.ServiceName))
	r.Use(middleware.Recovery())
	r.Use(metrics.Middleware())
	r.Use(middleware.RequestLogger())
	if rt.isProduction {
		r.Use(middleware.SecurityHeadersWithHSTS())
	} else {
		r.Use(middleware.SecurityHeadersMiddleware())
	}
	r.Use(middleware.MaxBodySize(rt.cfg.App.MaxBodySize))
}

func (rt *Runtime) registerPlatformRoutes(r *gin.Engine) error {
	corsOrigins := rt.cfg.App.CORSOrigins
	if err := config.ValidateCORSOrigins(corsOrigins); err != nil {
		return err
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", middleware.CSRFHeaderName, middleware.HeaderRequestID},
		ExposeHeaders:    []string{"Content-Length", middleware.CSRFHeaderName, middleware.HeaderRequestID},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	health.NewHandler(rt.pgPool, rt.redisClient.GetClient(), health.BuildInfo{
		Version:   rt.build.Version,
		GitCommit: rt.build.GitCommit,
		BuildTime: rt.build.BuildTime,
	}, rt.isProduction, time.Duration(rt.cfg.App.HealthCheckTimeout)*time.Second).RegisterRoutes(r)

	apidocs.RegisterDocs(r, rt.isProduction)

	metricsGroup := r.Group("/metrics")
	if rt.isProduction {
		if configStringMissing(rt.cfg.App.MetricsUser) {
			return fmt.Errorf("METRICS_USER must be set in production to protect the metrics endpoint")
		}
		if configStringMissing(rt.cfg.App.MetricsPassword) {
			return fmt.Errorf("METRICS_PASSWORD must be set in production to protect the metrics endpoint")
		}
		metricsGroup.Use(gin.BasicAuth(gin.Accounts{
			rt.cfg.App.MetricsUser: rt.cfg.App.MetricsPassword,
		}))
	}
	metricsGroup.GET("", gin.WrapH(promhttp.Handler()))

	return nil
}
