package oidc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProviderRolesFromRaw_CasdoorFlatRoles(t *testing.T) {
	roles, err := ParseProviderRolesFromRaw([]byte(`{"roles":["school_admin","verified_student","school_admin"]}`), "roles")
	require.NoError(t, err)
	assert.Equal(t, []string{"school_admin", "verified_student"}, roles)
}

func TestParseProviderRolesFromRaw_CustomRolesClaim(t *testing.T) {
	raw := []byte(`{"stuhelper_roles":["super_admin","user"]}`)

	roles, err := ParseProviderRolesFromRaw(raw, "stuhelper_roles")

	require.NoError(t, err)
	assert.Equal(t, []string{"super_admin", "user"}, roles)
}

func TestParseProviderRolesFromRaw_CasdoorRoleObjects(t *testing.T) {
	raw := []byte(`{"roles":[
		{"owner":"stuhelper","name":"super_admin"},
		{"owner":"stuhelper","name":"verified_student"},
		{"owner":"stuhelper","name":"super_admin"}
	]}`)

	roles, err := ParseProviderRolesFromRaw(raw, "roles")

	require.NoError(t, err)
	assert.Equal(t, []string{"super_admin", "verified_student"}, roles)
}

func TestParseProviderRolesFromRaw_InvalidJSON(t *testing.T) {
	_, err := ParseProviderRolesFromRaw([]byte(`{"broken"`), "roles")
	require.Error(t, err)
}

func TestParseProviderRolesFromRaw_InvalidRolesClaim(t *testing.T) {
	_, err := ParseProviderRolesFromRaw([]byte(`{"roles":"bad"}`), "roles")
	require.Error(t, err)
}

func TestParseProviderRolesFromRaw_RejectsObjectRolesClaim(t *testing.T) {
	_, err := ParseProviderRolesFromRaw([]byte(`{"roles":{"school_admin":["4111010001"]}}`), "roles")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string array or role object array")
}

func TestParseProviderRolesFromRaw_RejectsRoleObjectWithoutName(t *testing.T) {
	_, err := ParseProviderRolesFromRaw([]byte(`{"roles":[{"displayName":"Super Admin"}]}`), "roles")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "without string name")
}

func TestParseProviderRolesFromRaw_MissingRolesClaim(t *testing.T) {
	roles, err := ParseProviderRolesFromRaw([]byte(`{"sub":"user-1"}`), "roles")
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestClaimsUnmarshalIgnoresInternalRoleFields(t *testing.T) {
	raw := []byte(`{
		"sub": "user-1",
		"name": "OIDC User",
		"roles": {"school_admin": ["4111010001"]},
		"orgScopedRoles": ["legacy-provider-shape"]
	}`)
	var claims Claims

	err := json.Unmarshal(raw, &claims)

	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Sub)
	assert.Equal(t, "OIDC User", claims.Name)
	assert.Empty(t, claims.Roles)
	assert.Nil(t, claims.OrgScopedRoles)
}

func TestClaims_MFAProofVerifiedAtRequiresMFAAMRAndAuthTime(t *testing.T) {
	authTime := time.Date(2026, 5, 2, 10, 30, 0, 0, time.UTC)
	claims := &Claims{AMR: []string{"pwd", "totp"}, AuthTime: authTime.Unix()}

	assert.Equal(t, authTime, claims.MFAProofVerifiedAt())
	assert.True(t, HasMFAAMR([]string{" PWD ", "WebAuthn"}))
	assert.True(t, HasMFAAMR([]string{"pwd", "sms"}))
	assert.True(t, HasMFAAMR([]string{"pwd", "app"}))
	assert.True(t, MFAProofVerifiedAt([]string{"mfa"}, authTime.Unix()).Equal(authTime))
	assert.True(t, MFAProofVerifiedAt([]string{"pwd"}, authTime.Unix()).IsZero())
	assert.True(t, MFAProofVerifiedAt([]string{"totp"}, 0).IsZero())
}

func TestClaims_MFAProofVerifiedAtNilSafe(t *testing.T) {
	var claims *Claims
	assert.True(t, claims.MFAProofVerifiedAt().IsZero())
}

func TestAppIDFromRawClaims(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "aud string",
			raw:      `{"aud":"stuhelper-web"}`,
			expected: "stuhelper-web",
		},
		{
			name:     "authorized party wins for multi audience",
			raw:      `{"aud":["stuhelper-web","stuhelper-admin"],"azp":"stuhelper-admin"}`,
			expected: "stuhelper-admin",
		},
		{
			name:     "client id introspection fallback",
			raw:      `{"client_id":"third-party-client"}`,
			expected: "third-party-client",
		},
		{
			name:     "ambiguous multi audience without authorized party",
			raw:      `{"aud":["stuhelper-web","stuhelper-admin"]}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, appIDFromRawClaims([]byte(tt.raw)))
		})
	}
}
