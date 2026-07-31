package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUsernamesNormalizesAndDeduplicates(t *testing.T) {
	usernames, err := parseUsernames(" Alice, bob,alice ,, ")

	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "bob"}, usernames)
}

func TestParseUsernamesRejectsMissingOrPlaceholderValue(t *testing.T) {
	for _, raw := range []string{"", " , ", "REPLACE_WITH_INITIAL_SUPER_ADMIN_USERNAMES"} {
		t.Run(raw, func(t *testing.T) {
			_, err := parseUsernames(raw)
			require.Error(t, err)
		})
	}
}

func TestRunRejectsSingleProductionBootstrapAdministratorBeforeDatabaseAccess(t *testing.T) {
	env := map[string]string{
		"APP_ENV":                        "production",
		"STUHELPER_INITIAL_SUPER_ADMINS": "alice",
		"DATABASE_URL":                   "postgresql://unused",
	}
	err := run(nil, func(key string) string { return env[key] })
	require.EqualError(
		t,
		err,
		"production bootstrap requires at least two initial super administrators",
	)
}
