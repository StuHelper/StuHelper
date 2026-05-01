package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProviderRolesFromRaw_CasdoorFlatRoles(t *testing.T) {
	roles, scoped, err := ParseProviderRolesFromRaw([]byte(`{"roles":["school_admin","verified_student","school_admin"]}`), "roles")
	require.NoError(t, err)
	assert.Equal(t, []string{"school_admin", "verified_student"}, roles)
	assert.Nil(t, scoped)
}

func TestParseProviderRolesFromRaw_CustomRolesClaim(t *testing.T) {
	raw := []byte(`{"stuhelper_roles":["super_admin","user"]}`)

	roles, scoped, err := ParseProviderRolesFromRaw(raw, "stuhelper_roles")

	require.NoError(t, err)
	assert.Equal(t, []string{"super_admin", "user"}, roles)
	assert.Nil(t, scoped)
}

func TestParseProviderRolesFromRaw_InvalidJSON(t *testing.T) {
	_, _, err := ParseProviderRolesFromRaw([]byte(`{"broken"`), "roles")
	require.Error(t, err)
}

func TestParseProviderRolesFromRaw_InvalidRolesClaim(t *testing.T) {
	_, _, err := ParseProviderRolesFromRaw([]byte(`{"roles":"bad"}`), "roles")
	require.Error(t, err)
}

func TestParseProviderRolesFromRaw_MissingRolesClaim(t *testing.T) {
	roles, scoped, err := ParseProviderRolesFromRaw([]byte(`{"sub":"user-1"}`), "roles")
	require.NoError(t, err)
	assert.Empty(t, roles)
	assert.Nil(t, scoped)
}

func TestClaims_HasRoleInOrg(t *testing.T) {
	c := &Claims{
		OrgScopedRoles: map[string][]string{
			"school_admin":     {"1001", "1002"},
			"verified_student": {"1001"},
		},
	}
	assert.True(t, c.HasRoleInOrg("school_admin", "1001"))
	assert.True(t, c.HasRoleInOrg("school_admin", "1002"))
	assert.False(t, c.HasRoleInOrg("school_admin", "9999"))
	assert.True(t, c.HasRoleInOrg("school_admin", ""), "empty orgID means any scope")
	assert.False(t, c.HasRoleInOrg("platform_admin", "1001"))
}

func TestClaims_HasRoleInOrg_NilSafe(t *testing.T) {
	var c *Claims
	assert.False(t, c.HasRoleInOrg("school_admin", "1001"))

	c2 := &Claims{}
	assert.False(t, c2.HasRoleInOrg("school_admin", "1001"))
}
