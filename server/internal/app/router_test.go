package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	redisclient "github.com/StuHelper/StuHelper/server/internal/pkg/redis"
)

func TestRegisterPlatformRoutes_RequiresCORSOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				Env:                "development",
				HealthCheckTimeout: 3,
			},
			Observability: config.ObservabilityConfig{
				ServiceName: "stuhelper-test",
			},
		},
		redisClient: &redisclient.Client{},
	}

	err := rt.registerPlatformRoutes(gin.New())

	require.Error(t, err)
	require.Contains(t, err.Error(), "CORS_ORIGINS")
}

func TestRegisterPlatformRoutes_RejectsBlankMetricsPasswordInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				Env:                "production",
				CORSOrigins:        []string{"https://stuhelper.example.com"},
				MetricsUser:        "prometheus",
				MetricsPassword:    "  ",
				HealthCheckTimeout: 3,
			},
			Observability: config.ObservabilityConfig{
				ServiceName: "stuhelper-test",
			},
		},
		isProduction: true,
		redisClient:  &redisclient.Client{},
	}

	err := rt.registerPlatformRoutes(gin.New())

	require.Error(t, err)
	require.Contains(t, err.Error(), "METRICS_PASSWORD")
}

func TestRegisterPlatformRoutes_RejectsBlankMetricsUserInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				Env:                "production",
				CORSOrigins:        []string{"https://stuhelper.example.com"},
				MetricsUser:        "  ",
				MetricsPassword:    "metrics-password",
				HealthCheckTimeout: 3,
			},
			Observability: config.ObservabilityConfig{
				ServiceName: "stuhelper-test",
			},
		},
		isProduction: true,
		redisClient:  &redisclient.Client{},
	}

	err := rt.registerPlatformRoutes(gin.New())

	require.Error(t, err)
	require.Contains(t, err.Error(), "METRICS_USER")
}

func TestRegisterPlatformRoutes_CORSUsesSecurityHeaderConstants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				Env:                "development",
				CORSOrigins:        []string{"https://stuhelper.example.com"},
				HealthCheckTimeout: 3,
			},
			Observability: config.ObservabilityConfig{
				ServiceName: "stuhelper-test",
			},
		},
		redisClient: &redisclient.Client{},
	}
	router := gin.New()
	require.NoError(t, rt.registerPlatformRoutes(router))
	router.POST("/probe", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	preflight := httptest.NewRequest(http.MethodOptions, "/probe", nil)
	preflight.Header.Set("Origin", "https://stuhelper.example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.Header.Set("Access-Control-Request-Headers", middleware.CSRFHeaderName+","+middleware.HeaderRequestID)
	preflightRecorder := httptest.NewRecorder()
	router.ServeHTTP(preflightRecorder, preflight)

	require.Equal(t, http.StatusNoContent, preflightRecorder.Code)
	allowedHeaders := preflightRecorder.Header().Get("Access-Control-Allow-Headers")
	requireCommaHeaderContains(t, allowedHeaders, middleware.CSRFHeaderName)
	requireCommaHeaderContains(t, allowedHeaders, middleware.HeaderRequestID)

	request := httptest.NewRequest(http.MethodPost, "/probe", nil)
	request.Header.Set("Origin", "https://stuhelper.example.com")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	exposedHeaders := recorder.Header().Get("Access-Control-Expose-Headers")
	requireCommaHeaderContains(t, exposedHeaders, middleware.CSRFHeaderName)
	requireCommaHeaderContains(t, exposedHeaders, middleware.HeaderRequestID)
}

func requireCommaHeaderContains(t *testing.T, headerValue, want string) {
	t.Helper()
	canonicalWant := http.CanonicalHeaderKey(want)
	for _, part := range strings.Split(headerValue, ",") {
		if strings.TrimSpace(part) == canonicalWant {
			return
		}
	}
	require.Contains(t, headerValue, canonicalWant)
}
