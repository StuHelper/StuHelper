package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

func TestLoadSettingsBuildsBootstrapPlan(t *testing.T) {
	settings, err := loadSettings(testEnv(completeEnv()))

	require.NoError(t, err)
	assert.Equal(t, casdoor.PurposeBootstrap, settings.credential.Purpose)
	assert.Equal(t, "https://sso.example.com", settings.credential.Endpoint)
	assert.Equal(t, "stuhelper", settings.plan.Organization.Name)
	require.Len(t, settings.plan.Applications, 3)
	assert.Equal(t, "stuhelper-web", settings.plan.Applications[0].Name)
	assert.Equal(t, "stuhelper-admin", settings.plan.Applications[1].Name)
	assert.Equal(t, "stuhelper-uniapp", settings.plan.Applications[2].Name)
	require.Len(t, settings.plan.Roles, 7)
	assert.Equal(t, "super_admin", settings.plan.Roles[0].Name)
	require.Len(t, settings.plan.Providers, 1)
	assert.Equal(t, "stuhelper-sms", settings.plan.Providers[0].Name)
}

func TestLoadSettingsRequiresDedicatedBootstrapCredential(t *testing.T) {
	env := completeEnv()
	delete(env, "CASDOOR_BOOTSTRAP_CLIENT_SECRET")

	settings, err := loadSettings(testEnv(env))

	require.Error(t, err)
	assert.Equal(t, bootstrapSettings{}, settings)
	assert.Contains(t, err.Error(), "CASDOOR_BOOTSTRAP_CLIENT_SECRET is required")
}

func TestLoadSettingsIgnoresDisabledProviders(t *testing.T) {
	env := completeEnv()
	env["CASDOOR_SMS_PROVIDER_ENABLED"] = "false"

	settings, err := loadSettings(testEnv(env))

	require.NoError(t, err)
	assert.Empty(t, settings.plan.Providers)
}

func TestLoadSettingsRejectsInvalidProviderFlag(t *testing.T) {
	env := completeEnv()
	env["CASDOOR_SMS_PROVIDER_ENABLED"] = "maybe"

	_, err := loadSettings(testEnv(env))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_SMS_PROVIDER_ENABLED must be true or false")
}

func completeEnv() map[string]string {
	return map[string]string{
		"CASDOOR_ISSUER":                    "https://sso.example.com",
		"CASDOOR_BOOTSTRAP_CLIENT_ID":       "bootstrap-client",
		"CASDOOR_BOOTSTRAP_CLIENT_SECRET":   "bootstrap-secret",
		"CASDOOR_BOOTSTRAP_APPLICATION":     "casdoor-admin-bootstrap",
		"CASDOOR_ORGANIZATION":              "stuhelper",
		"CASDOOR_CLIENT_ID":                 "stuhelper-web",
		"CASDOOR_CLIENT_SECRET":             "web-secret",
		"CASDOOR_REDIRECT_URI":              "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_ADMIN_CLIENT_ID":           "stuhelper-admin",
		"CASDOOR_ADMIN_CLIENT_SECRET":       "admin-secret",
		"CASDOOR_ADMIN_REDIRECT_URI":        "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_UNIAPP_CLIENT_ID":          "stuhelper-uniapp",
		"CASDOOR_UNIAPP_CLIENT_SECRET":      "uniapp-secret",
		"CASDOOR_UNIAPP_REDIRECT_URI":       "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_SMS_PROVIDER_ENABLED":      "true",
		"CASDOOR_SMS_PROVIDER_NAME":         "stuhelper-sms",
		"CASDOOR_SMS_PROVIDER_DISPLAY_NAME": "StuHelper SMS",
		"CASDOOR_SMS_PROVIDER_CATEGORY":     "SMS",
		"CASDOOR_SMS_PROVIDER_TYPE":         "CustomHTTP",
		"CASDOOR_SMS_PROVIDER_METHOD":       "POST",
		"CASDOOR_SMS_PROVIDER_ENDPOINT":     "https://api.example.com/internal/sms/send",
	}
}

func testEnv(values map[string]string) envReader {
	return func(key string) string {
		return values[key]
	}
}
