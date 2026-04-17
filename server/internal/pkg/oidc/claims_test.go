package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRolesFromRaw(t *testing.T) {
	roles, scoped, err := ParseRolesFromRaw([]byte(`{"urn:zitadel:iam:org:project:test-project:roles":{"school_admin":{"1":"example.com"},"verified_student":{"1":"example.com"}}}`), "test-project")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"school_admin", "verified_student"}, roles)
	assert.ElementsMatch(t, []string{"1"}, scoped["school_admin"])
	assert.ElementsMatch(t, []string{"1"}, scoped["verified_student"])
}

func TestParseRolesFromRaw_ScopedMultipleOrgs(t *testing.T) {
	roles, scoped, err := ParseRolesFromRaw([]byte(`{"urn:zitadel:iam:org:project:test-project:roles":{"school_admin":{"1001":"a.example.com","1002":"b.example.com"}}}`), "test-project")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"school_admin"}, roles)
	assert.ElementsMatch(t, []string{"1001", "1002"}, scoped["school_admin"])
}

func TestParseRolesFromRaw_InvalidJSON(t *testing.T) {
	_, _, err := ParseRolesFromRaw([]byte(`{"broken"`), "test-project")
	require.Error(t, err)
}

func TestParseRolesFromRaw_InvalidRolesClaim(t *testing.T) {
	_, _, err := ParseRolesFromRaw([]byte(`{"urn:zitadel:iam:org:project:test-project:roles":"bad"}`), "test-project")
	require.Error(t, err)
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
