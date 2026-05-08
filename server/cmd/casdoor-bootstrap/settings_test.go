package main

import (
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

var flatRoleNamePattern = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

func TestLoadSettingsBuildsBootstrapPlan(t *testing.T) {
	settings, err := loadSettings(testEnv(completeEnv()))

	require.NoError(t, err)
	assert.Equal(t, casdoor.PurposeBootstrap, settings.credential.Purpose)
	assert.Equal(t, "https://sso.example.com", settings.credential.Endpoint)
	assert.Equal(t, "stuhelper", settings.plan.Organization.Name)
	require.Len(t, settings.plan.Applications, 7)
	assert.Equal(t, "stuhelper-web", settings.plan.Applications[0].Name)
	assert.Equal(t, "stuhelper-admin", settings.plan.Applications[1].Name)
	assert.Equal(t, "stuhelper-uniapp", settings.plan.Applications[2].Name)
	assert.Equal(t, "casdoor-admin-app-provisioning", settings.plan.Applications[3].Name)
	assert.Equal(t, "casdoor-token-introspection", settings.plan.Applications[4].Name)
	assert.Equal(t, []string{"client_credentials"}, settings.plan.Applications[3].GrantTypes)
	assert.Equal(t, []string{"client_credentials"}, settings.plan.Applications[4].GrantTypes)
	require.Len(t, settings.plan.Roles, 8)
	assert.Equal(t, "super_admin", settings.plan.Roles[0].Name)
	require.Len(t, settings.plan.Providers, 1)
	assert.Equal(t, "stuhelper-sms", settings.plan.Providers[0].Name)
	assert.Equal(t, "Custom HTTP SMS", settings.plan.Providers[0].Type)
	assert.Equal(t, "content", settings.plan.Providers[0].Title)
	assert.Equal(t, "https://api.example.com/internal/sms/send?internal_key=sms-internal-key", settings.plan.Providers[0].Endpoint)
}

func TestFlatRoleCatalogMatchesAuthorizationRoles(t *testing.T) {
	expected := []string{
		"super_admin",
		"school_admin",
		"section_admin",
		"section_moderator",
		"section_reviewer",
		"verified_student",
		"freshman_provisional",
		"user",
	}
	bootstrapRoles := roleNames(flatRoleCatalog())
	assert.Equal(t, expected, bootstrapRoles)
	assert.ElementsMatch(t, capabilityRoleNames(), bootstrapRoles)
	for _, role := range bootstrapRoles {
		assert.Truef(t, flatRoleNamePattern.MatchString(role), "role %q must stay flat and ID-free", role)
	}
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
		"CASDOOR_ISSUER":                         "https://sso.example.com",
		"CASDOOR_BOOTSTRAP_CLIENT_ID":            "bootstrap-client",
		"CASDOOR_BOOTSTRAP_CLIENT_SECRET":        "bootstrap-secret",
		"CASDOOR_BOOTSTRAP_APPLICATION":          "casdoor-admin-bootstrap",
		"CASDOOR_ORGANIZATION":                   "stuhelper",
		"CASDOOR_CLIENT_ID":                      "stuhelper-web",
		"CASDOOR_CLIENT_SECRET":                  "web-secret",
		"CASDOOR_REDIRECT_URI":                   "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_ADMIN_CLIENT_ID":                "stuhelper-admin",
		"CASDOOR_ADMIN_CLIENT_SECRET":            "admin-secret",
		"CASDOOR_ADMIN_REDIRECT_URI":             "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_UNIAPP_CLIENT_ID":               "stuhelper-uniapp",
		"CASDOOR_UNIAPP_CLIENT_SECRET":           "uniapp-secret",
		"CASDOOR_UNIAPP_REDIRECT_URI":            "https://api.example.com/api/v1/auth/callback",
		"CASDOOR_APP_PROVISIONING_CLIENT_ID":     "casdoor-admin-app-provisioning",
		"CASDOOR_APP_PROVISIONING_CLIENT_SECRET": "app-provisioning-secret",
		"CASDOOR_APP_PROVISIONING_APPLICATION":   "casdoor-admin-app-provisioning",
		"CASDOOR_INTROSPECTION_CLIENT_ID":        "casdoor-token-introspection",
		"CASDOOR_INTROSPECTION_CLIENT_SECRET":    "introspection-secret",
		"CASDOOR_INTROSPECTION_APPLICATION":      "casdoor-token-introspection",
		"CASDOOR_ROLE_SYNC_CLIENT_ID":            "casdoor-admin-role-sync",
		"CASDOOR_ROLE_SYNC_CLIENT_SECRET":        "role-sync-secret",
		"CASDOOR_ROLE_SYNC_APPLICATION":          "casdoor-admin-role-sync",
		"CASDOOR_USER_LOOKUP_CLIENT_ID":          "casdoor-admin-user-lookup",
		"CASDOOR_USER_LOOKUP_CLIENT_SECRET":      "user-lookup-secret",
		"CASDOOR_USER_LOOKUP_APPLICATION":        "casdoor-admin-user-lookup",
		"CASDOOR_SMS_PROVIDER_ENABLED":           "true",
		"CASDOOR_SMS_PROVIDER_NAME":              "stuhelper-sms",
		"CASDOOR_SMS_PROVIDER_DISPLAY_NAME":      "StuHelper SMS",
		"CASDOOR_SMS_PROVIDER_CATEGORY":          "SMS",
		"CASDOOR_SMS_PROVIDER_TYPE":              "CustomHTTP",
		"CASDOOR_SMS_PROVIDER_METHOD":            "POST",
		"CASDOOR_SMS_PROVIDER_TITLE":             "content",
		"CASDOOR_SMS_PROVIDER_ENDPOINT":          "https://api.example.com/internal/sms/send",
		"SMS_INTERNAL_KEY":                       "sms-internal-key",
	}
}

func testEnv(values map[string]string) envReader {
	return func(key string) string {
		return values[key]
	}
}

func roleNames(roles []casdoor.RoleSpec) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Name)
	}
	return names
}

func capabilityRoleNames() []string {
	roleCapabilities := capability.GetRoleCapabilities()
	roles := make([]string, 0, len(roleCapabilities))
	for role := range roleCapabilities {
		roles = append(roles, role)
	}
	slices.Sort(roles)
	return roles
}
