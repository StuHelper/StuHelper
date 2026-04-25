package app

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	redisclient "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
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
