package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestRuntimeMetricsAllowedOriginsIncludesLoopbackDevelopmentHosts(t *testing.T) {
	rt := &Runtime{
		cfg: &config.Config{App: config.AppConfig{Env: "development"}},
	}

	require.ElementsMatch(t, []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:4173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:4173",
	}, rt.metricsAllowedOrigins())
}

func TestRuntimeMetricsAllowedOriginsUsesConfiguredOrigins(t *testing.T) {
	configuredOrigins := []string{"https://stuhelper.example.com"}
	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				CORSOrigins: configuredOrigins,
				Env:         "development",
			},
		},
	}

	require.Equal(t, configuredOrigins, rt.metricsAllowedOrigins())
}

func TestRuntimeMetricsAllowedOriginsUsesFrontendMetricsOrigins(t *testing.T) {
	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				CORSOrigins: []string{"https://api-cors.example.com"},
				Env:         "development",
			},
			Observability: config.ObservabilityConfig{
				FrontendMetricsAllowedOrigins: []string{"https://web.example.com"},
			},
		},
	}

	require.Equal(t, []string{"https://web.example.com"}, rt.metricsAllowedOrigins())
}

func TestRuntimeMetricsAllowedOriginsReturnsCopy(t *testing.T) {
	rt := &Runtime{
		cfg: &config.Config{
			App: config.AppConfig{
				CORSOrigins: []string{"https://stuhelper.example.com"},
				Env:         "development",
			},
		},
	}

	origins := rt.metricsAllowedOrigins()
	origins[0] = "https://mutated.example.com"

	require.Equal(t, []string{"https://stuhelper.example.com"}, rt.cfg.App.CORSOrigins)
	require.Equal(t, []string{"https://stuhelper.example.com"}, rt.metricsAllowedOrigins())
}

func TestRuntimeMetricsAllowedOriginsEmptyInProductionByDefault(t *testing.T) {
	rt := &Runtime{
		cfg:          &config.Config{App: config.AppConfig{Env: "production"}},
		isProduction: true,
	}

	require.Nil(t, rt.metricsAllowedOrigins())
}
