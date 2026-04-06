package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRolesFromRaw(t *testing.T) {
	roles, err := ParseRolesFromRaw([]byte(`{"urn:zitadel:iam:org:project:test-project:roles":{"school_admin":{"1":"example.com"},"verified_student":{"1":"example.com"}}}`), "test-project")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"school_admin", "verified_student"}, roles)
}

func TestParseRolesFromRaw_InvalidJSON(t *testing.T) {
	_, err := ParseRolesFromRaw([]byte(`{"broken"`), "test-project")
	require.Error(t, err)
}

func TestParseRolesFromRaw_InvalidRolesClaim(t *testing.T) {
	_, err := ParseRolesFromRaw([]byte(`{"urn:zitadel:iam:org:project:test-project:roles":"bad"}`), "test-project")
	require.Error(t, err)
}
