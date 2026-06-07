package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/tokenprobe"
	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

func TestOpenPlatformDisclosureRateLimitConfigMapsRuntimeConfig(t *testing.T) {
	cfg := openPlatformDisclosureRateLimitConfig(config.OpenPlatformDisclosureRateLimitConfig{
		AppLimit:                   700,
		AppUserLimit:               140,
		EndpointLimit:              1400,
		ConsentLimit:               30,
		ReplayLimit:                9,
		WindowSeconds:              90,
		ReplayWindowSeconds:        600,
		ReplayAuditCooldownSeconds: 1200,
	})

	require.Equal(t, 700, cfg.AppLimit)
	require.Equal(t, 140, cfg.AppUserLimit)
	require.Equal(t, 1400, cfg.EndpointLimit)
	require.Equal(t, 30, cfg.ConsentLimit)
	require.Equal(t, 9, cfg.ReplayLimit)
	require.Equal(t, 90*time.Second, cfg.Window)
	require.Equal(t, 600*time.Second, cfg.ReplayWindow)
	require.Equal(t, 1200*time.Second, cfg.ReplayAuditCooldown)
}

func TestOpenPlatformDisclosureRateLimitConfigLeavesZeroDefaultsToService(t *testing.T) {
	cfg := openPlatformDisclosureRateLimitConfig(config.OpenPlatformDisclosureRateLimitConfig{})

	require.Zero(t, cfg.AppLimit)
	require.Zero(t, cfg.AppUserLimit)
	require.Zero(t, cfg.EndpointLimit)
	require.Zero(t, cfg.ConsentLimit)
	require.Zero(t, cfg.ReplayLimit)
	require.Zero(t, cfg.Window)
	require.Zero(t, cfg.ReplayWindow)
	require.Zero(t, cfg.ReplayAuditCooldown)
}

func TestRuntimeOpenPlatformBaseURLsPreferExplicitConfig(t *testing.T) {
	rt := &Runtime{cfg: &config.Config{
		App: config.AppConfig{
			CORSOrigins: []string{"https://cors.example.com"},
		},
		OpenPlatform: config.OpenPlatformConfig{
			ConsentBaseURL: "https://consent.example.com",
			AccountBaseURL: "https://account.example.com",
		},
	}}

	consentBaseURL := rt.openPlatformConsentBaseURL()

	assert.Equal(t, "https://consent.example.com", consentBaseURL)
	assert.Equal(t, "https://account.example.com", rt.openPlatformAccountBaseURL(consentBaseURL))
}

func TestRuntimeOpenPlatformBaseURLsFallbackToCORSOriginInDevelopment(t *testing.T) {
	rt := &Runtime{cfg: &config.Config{
		App: config.AppConfig{
			Env:         config.EnvDevelopment,
			CORSOrigins: []string{"https://web.example.com", "https://admin.example.com"},
		},
	}}

	consentBaseURL := rt.openPlatformConsentBaseURL()

	assert.Equal(t, "https://web.example.com", consentBaseURL)
	assert.Equal(t, "https://web.example.com", rt.openPlatformAccountBaseURL(consentBaseURL))
}

func TestRuntimeOpenPlatformBaseURLsDoNotFallbackToCORSOriginInProductionLike(t *testing.T) {
	for _, env := range []string{config.EnvProduction, config.EnvProdParity} {
		t.Run(env, func(t *testing.T) {
			rt := &Runtime{cfg: &config.Config{
				App: config.AppConfig{
					Env:         env,
					CORSOrigins: []string{"https://web.example.com", "https://admin.example.com"},
				},
			}}

			consentBaseURL := rt.openPlatformConsentBaseURL()

			assert.Empty(t, consentBaseURL)
			assert.Empty(t, rt.openPlatformAccountBaseURL(consentBaseURL))
		})
	}
}

func TestRuntimeOpenPlatformAccountBaseURLDefaultsToConsentBaseURL(t *testing.T) {
	rt := &Runtime{cfg: &config.Config{
		App: config.AppConfig{
			CORSOrigins: []string{"https://web.example.com"},
		},
		OpenPlatform: config.OpenPlatformConfig{
			ConsentBaseURL: "https://consent.example.com",
		},
	}}

	consentBaseURL := rt.openPlatformConsentBaseURL()

	assert.Equal(t, "https://consent.example.com", consentBaseURL)
	assert.Equal(t, "https://consent.example.com", rt.openPlatformAccountBaseURL(consentBaseURL))
}

func TestRuntimeOpenPlatformTokenProbeTreatsBlankCommandAsNotConfigured(t *testing.T) {
	rt := &Runtime{cfg: &config.Config{
		OpenPlatform: config.OpenPlatformConfig{
			TokenProbe: config.OpenPlatformTokenProbeConfig{
				RuntimeCommand: "  ",
			},
		},
	}}

	prober, err := rt.newOpenPlatformRuntimeTokenProber()

	require.NoError(t, err)
	assert.Nil(t, prober)
}

func TestOpenPlatformCasdoorSpecMappingPreservesExplicitEmptyTokenFields(t *testing.T) {
	fromOpenPlatform := casdoorApplicationSpecFromOpenPlatform(openplatform.ProvisionedApplicationSpec{
		TokenFormat: "JWT-Custom",
		TokenFields: []string{},
	})

	require.NotNil(t, fromOpenPlatform.TokenFields)
	assert.Empty(t, fromOpenPlatform.TokenFields)
	_, err := platformcasdoor.ProbeApplicationSpecTokenMinimization(fromOpenPlatform)
	require.NoError(t, err)

	fromCasdoor := openPlatformApplicationSpecFromCasdoor(platformcasdoor.ApplicationSpec{
		TokenFormat: "JWT-Custom",
		TokenFields: []string{},
	})

	require.NotNil(t, fromCasdoor.TokenFields)
	assert.Empty(t, fromCasdoor.TokenFields)
	_, err = tokenprobe.ProbeApplicationTokenFieldsMinimized(fromCasdoor.TokenFormat, fromCasdoor.TokenFields)
	require.NoError(t, err)
}
