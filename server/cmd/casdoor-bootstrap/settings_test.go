package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

func TestLoadSettingsBuildsBootstrapPlan(t *testing.T) {
	settings, err := loadSettings(testEnv(completeEnv()))

	require.NoError(t, err)
	assert.Equal(t, casdoor.PurposeBootstrap, settings.credential.Purpose)
	assert.Equal(t, "https://sso.example.com", settings.credential.Endpoint)
	assert.Equal(t, "stuhelper", settings.credential.Organization)
	assert.Equal(t, "stuhelper", settings.plan.Organization.Name)
	require.Len(t, settings.plan.Applications, 8)
	assert.Equal(t, "stuhelper-web", settings.plan.Applications[0].Name)
	assert.Equal(t, "stuhelper-admin", settings.plan.Applications[1].Name)
	assert.Equal(t, "stuhelper-uniapp", settings.plan.Applications[2].Name)
	assert.Equal(t, "https://www.example.com/android-chrome-512x512.png", settings.plan.Applications[1].Logo)
	assert.Equal(t, "casdoor-admin-app-provisioning", settings.plan.Applications[3].Name)
	assert.Equal(t, "casdoor-token-introspection", settings.plan.Applications[4].Name)
	assert.Equal(t, "casdoor-admin-user-profile", settings.plan.Applications[5].Name)
	assert.Equal(t, "casdoor-admin-user-lookup", settings.plan.Applications[6].Name)
	assert.Equal(t, "casdoor-token-probe-smoke", settings.plan.Applications[7].Name)
	assert.Equal(t, []string{"client_credentials"}, settings.plan.Applications[3].GrantTypes)
	assert.Equal(t, []string{"client_credentials"}, settings.plan.Applications[4].GrantTypes)
	assert.Equal(t, []string{"client_credentials"}, settings.plan.Applications[5].GrantTypes)
	assert.Equal(t, []string{"https://api.example.com/api/v1/auth/callback"}, settings.plan.Applications[0].RedirectURIs)
	assert.Equal(t, []string{"authorization_code", "refresh_token"}, settings.plan.Applications[7].GrantTypes)
	assert.Equal(t, []string{"https://www.example.com/open-platform/token-probe/callback"}, settings.plan.Applications[7].RedirectURIs)
	assert.Equal(t, "JWT-Custom", settings.plan.Applications[7].TokenFormat)
	assert.Equal(t, []string{}, settings.plan.Applications[7].TokenFields)
	require.Len(t, settings.plan.Providers, 1)
	assert.Equal(t, "stuhelper-sms", settings.plan.Providers[0].Name)
	assert.Equal(t, "Custom HTTP SMS", settings.plan.Providers[0].Type)
	assert.Equal(t, "content", settings.plan.Providers[0].Title)
	assert.Equal(t, "https://api.example.com/internal/sms/send?internal_key=sms-internal-key", settings.plan.Providers[0].Endpoint)
}

func TestLoadSettingsAllowsBootstrapCredentialOrganizationOverride(t *testing.T) {
	env := completeEnv()
	env["CASDOOR_BOOTSTRAP_ORGANIZATION"] = "built-in"

	settings, err := loadSettings(testEnv(env))

	require.NoError(t, err)
	assert.Equal(t, "built-in", settings.credential.Organization)
	assert.Equal(t, "stuhelper", settings.plan.Organization.Name)
}

func TestLoadSettingsAllowsApplicationLogoOverride(t *testing.T) {
	env := completeEnv()
	env["CASDOOR_ADMIN_LOGO"] = "https://static.example.com/admin-logo.png"

	settings, err := loadSettings(testEnv(env))

	require.NoError(t, err)
	assert.Equal(t, "https://static.example.com/admin-logo.png", settings.plan.Applications[1].Logo)
}

func TestLoadSettingsAppendsAdditionalApplicationRedirectURIs(t *testing.T) {
	env := completeEnv()
	env["CASDOOR_ADDITIONAL_REDIRECT_URIS"] = "https://www.example.com/api/v1/auth/callback, https://join.example.com/api/v1/auth/callback"
	env["CASDOOR_ADMIN_ADDITIONAL_REDIRECT_URIS"] = "https://admin.example.com/api/v1/auth/callback"

	settings, err := loadSettings(testEnv(env))

	require.NoError(t, err)
	assert.Equal(t, []string{
		"https://api.example.com/api/v1/auth/callback",
		"https://www.example.com/api/v1/auth/callback",
		"https://join.example.com/api/v1/auth/callback",
	}, settings.plan.Applications[0].RedirectURIs)
	assert.Equal(t, []string{
		"https://api.example.com/api/v1/auth/callback",
		"https://admin.example.com/api/v1/auth/callback",
	}, settings.plan.Applications[1].RedirectURIs)
}

func TestLoadApplicationBootstrapSettingsUsesAppProvisioningCredential(t *testing.T) {
	settings, err := loadApplicationBootstrapSettings(testEnv(completeEnv()))

	require.NoError(t, err)
	assert.Equal(t, casdoor.PurposeAppProvisioning, settings.credential.Purpose)
	assert.Equal(t, "casdoor-admin-app-provisioning", settings.credential.ClientID)
	assert.Equal(t, "stuhelper", settings.credential.Organization)
	require.Len(t, settings.applications, 8)
	assert.Equal(t, "stuhelper-web", settings.applications[0].Name)
	assert.Equal(t, "casdoor-token-probe-smoke", settings.applications[7].Name)
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

func TestLoadSettingsRequiresSMSInternalKeyForEnabledSMSProvider(t *testing.T) {
	env := completeEnv()
	delete(env, "SMS_INTERNAL_KEY")

	_, err := loadSettings(testEnv(env))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS_INTERNAL_KEY is required")
}

func TestLoadSettingsRejectsWrongSMSType(t *testing.T) {
	env := completeEnv()
	env["CASDOOR_SMS_PROVIDER_TYPE"] = "Custom HTTP SMS"

	_, err := loadSettings(testEnv(env))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_SMS_PROVIDER_TYPE must be CustomHTTP")
}

func completeEnv() map[string]string {
	return map[string]string{
		"CASDOOR_ISSUER":                          "https://sso.example.com",
		"WEB_PUBLIC_URL":                          "https://www.example.com",
		"CASDOOR_BOOTSTRAP_CLIENT_ID":             "bootstrap-client",
		"CASDOOR_BOOTSTRAP_CLIENT_SECRET":         "bootstrap-secret",
		"CASDOOR_BOOTSTRAP_APPLICATION":           "casdoor-admin-bootstrap",
		"CASDOOR_ORGANIZATION":                    "stuhelper",
		"CASDOOR_CLIENT_ID":                       "stuhelper-web",
		"CASDOOR_CLIENT_SECRET":                   "web-secret",
		"CASDOOR_REDIRECT_URI":                    "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_ADMIN_CLIENT_ID":                 "stuhelper-admin",
		"CASDOOR_ADMIN_CLIENT_SECRET":             "admin-secret",
		"CASDOOR_ADMIN_REDIRECT_URI":              "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_UNIAPP_CLIENT_ID":                "stuhelper-uniapp",
		"CASDOOR_UNIAPP_CLIENT_SECRET":            "uniapp-secret",
		"CASDOOR_UNIAPP_REDIRECT_URI":             "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_APP_PROVISIONING_CLIENT_ID":      "casdoor-admin-app-provisioning",
		"CASDOOR_APP_PROVISIONING_CLIENT_SECRET":  "app-provisioning-secret",
		"CASDOOR_APP_PROVISIONING_APPLICATION":    "casdoor-admin-app-provisioning",
		"CASDOOR_USER_PROFILE_CLIENT_ID":          "casdoor-admin-user-profile",
		"CASDOOR_USER_PROFILE_CLIENT_SECRET":      "user-profile-secret",
		"CASDOOR_USER_PROFILE_APPLICATION":        "casdoor-admin-user-profile",
		"CASDOOR_INTROSPECTION_CLIENT_ID":         "casdoor-token-introspection",
		"CASDOOR_INTROSPECTION_CLIENT_SECRET":     "introspection-secret",
		"CASDOOR_INTROSPECTION_APPLICATION":       "casdoor-token-introspection",
		"CASDOOR_USER_LOOKUP_CLIENT_ID":           "casdoor-admin-user-lookup",
		"CASDOOR_USER_LOOKUP_CLIENT_SECRET":       "user-lookup-secret",
		"CASDOOR_USER_LOOKUP_APPLICATION":         "casdoor-admin-user-lookup",
		"CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID":     "casdoor-token-probe-smoke",
		"CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET": "token-probe-smoke-secret",
		"CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION":   "casdoor-token-probe-smoke",
		"CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI":  "https://www.example.com/open-platform/token-probe/callback",
		"CASDOOR_SMS_PROVIDER_ENABLED":            "true",
		"CASDOOR_SMS_PROVIDER_NAME":               "stuhelper-sms",
		"CASDOOR_SMS_PROVIDER_DISPLAY_NAME":       "StuHelper SMS",
		"CASDOOR_SMS_PROVIDER_CATEGORY":           "SMS",
		"CASDOOR_SMS_PROVIDER_TYPE":               "CustomHTTP",
		"CASDOOR_SMS_PROVIDER_METHOD":             "POST",
		"CASDOOR_SMS_PROVIDER_TITLE":              "content",
		"CASDOOR_SMS_PROVIDER_ENDPOINT":           "https://api.example.com/internal/sms/send",
		"SMS_INTERNAL_KEY":                        "sms-internal-key",
	}
}

func testEnv(values map[string]string) envReader {
	return func(key string) string {
		return values[key]
	}
}
