package main

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

func TestEnvInt32RejectsOutOfRangeValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(math.MaxInt32), envInt32(
		func(string) string { return strconv.FormatInt(math.MaxInt32, 10) },
		"DB_MAX_CONNS",
		4,
	))
	assert.Equal(t, int32(4), envInt32(
		func(string) string { return strconv.FormatInt(int64(math.MaxInt32)+1, 10) },
		"DB_MAX_CONNS",
		4,
	))
}

func TestLoadSettingsUsesExplicitCutoverEndpointsAndSafeDefaults(t *testing.T) {
	env := map[string]string{
		"DATABASE_URL":                    "postgres://app:secret@postgres/stuhelper",
		"OPENFGA_API_URL":                 "http://openfga:8080",
		"OPENFGA_STORE_ID":                "store-id",
		"OPENFGA_MODEL_ID":                "model-id",
		"CASDOOR_CUTOVER_ENDPOINT":        "http://casdoor:8000",
		"CASDOOR_BOOTSTRAP_CLIENT_ID":     "client-id",
		"CASDOOR_BOOTSTRAP_CLIENT_SECRET": "client-secret",
		"CASDOOR_BOOTSTRAP_APPLICATION":   "app-built-in",
		"CASDOOR_BOOTSTRAP_ORGANIZATION":  "stuhelper",
	}

	settings, err := loadSettings(func(key string) string { return env[key] })
	require.NoError(t, err)
	assert.Equal(t, int32(4), settings.database.MaxConns)
	assert.Equal(t, 15, settings.database.QueryTimeout)
	assert.Equal(t, "store-id", settings.openFGA.StoreID)
	assert.Equal(t, platformcasdoor.PurposeAuthorityCutover, settings.casdoor.Purpose)
	assert.Equal(t, "http://casdoor:8000", settings.casdoor.Endpoint)
	assert.Equal(t, "stuhelper", settings.casdoor.Organization)
}

func TestLoadSettingsFallsBackToRuntimeCasdoorValues(t *testing.T) {
	env := map[string]string{
		"DATABASE_URL":                    "postgres://app:secret@postgres/stuhelper",
		"OPENFGA_API_URL":                 "http://openfga:8080",
		"OPENFGA_STORE_ID":                "store-id",
		"OPENFGA_MODEL_ID":                "model-id",
		"CASDOOR_ISSUER":                  "https://sso.stuhelper.com",
		"CASDOOR_BOOTSTRAP_CLIENT_ID":     "client-id",
		"CASDOOR_BOOTSTRAP_CLIENT_SECRET": "client-secret",
		"CASDOOR_BOOTSTRAP_APPLICATION":   "app-built-in",
		"CASDOOR_ORGANIZATION":            "stuhelper",
	}

	settings, err := loadSettings(func(key string) string { return env[key] })
	require.NoError(t, err)
	assert.Equal(t, "https://sso.stuhelper.com", settings.casdoor.Endpoint)
	assert.Equal(t, "stuhelper", settings.casdoor.Organization)
}

func TestLoadSettingsReportsOnlyMissingKeyNames(t *testing.T) {
	_, err := loadSettings(func(string) string { return "" })
	require.Error(t, err)
	assert.Equal(t, "DATABASE_URL is required", err.Error())
	assert.False(t, strings.Contains(err.Error(), "secret"))
}
