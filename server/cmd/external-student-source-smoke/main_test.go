package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmokeConfigFromEnv(t *testing.T) {
	env := map[string]string{
		"EXTERNAL_STUDENT_SOURCE_ENABLED":                    "true",
		"EXTERNAL_STUDENT_SOURCE_NAME":                       "buaa-academic-oracle",
		"EXTERNAL_STUDENT_SOURCE_PROVIDER":                   "oracle",
		"EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE":                "4111010006",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_HOST":                "oracle.example.test",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_PORT":                "65521",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME":        "ORCLPDB1",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME":            "SYSTEM",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD":            "secret",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA":              "USR_JWBIZ",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE":               "T_XS_JBXX",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN":   "XH",
		"EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN": "XM",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE":       "true",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID":           "20250001",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME":        "张三",
	}

	cfg, err := smokeConfigFromEnv(func(key string) string { return env[key] })
	require.NoError(t, err)

	assert.Equal(t, "buaa-academic-oracle", cfg.SourceName)
	assert.Equal(t, "oracle", cfg.Provider)
	assert.True(t, cfg.RequireReadableRecord)
	assert.True(t, cfg.RequireSample)
	assert.Equal(t, "4111010006", cfg.Oracle.SchoolCode)
	assert.Equal(t, 65521, cfg.Oracle.Port)
	assert.Equal(t, "verify-full", cfg.Oracle.TLSMode)
	assert.Equal(t, "/external-student-source-tls/ca.crt", cfg.Oracle.TLSCAFile)
	assert.Equal(t, 300, int(cfg.Oracle.ConnMaxLifetime.Seconds()))
	assert.Equal(t, 60, int(cfg.Oracle.ConnMaxIdleTime.Seconds()))
	assert.Equal(t, 5, cfg.Oracle.BreakerFailureThreshold)
	assert.Equal(t, 2, cfg.Oracle.BreakerSuccessThreshold)
	assert.Equal(t, 30, int(cfg.Oracle.BreakerOpenTimeout.Seconds()))
	assert.Equal(t, "20250001", cfg.SampleStudentID)
	assert.Equal(t, "张三", cfg.SampleExpectedName)
}

func TestSmokeConfigFromEnvRequiresSampleWhenRequested(t *testing.T) {
	env := map[string]string{
		"EXTERNAL_STUDENT_SOURCE_ENABLED":              "true",
		"EXTERNAL_STUDENT_SOURCE_PROVIDER":             "oracle",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE": "true",
	}

	_, err := smokeConfigFromEnv(func(key string) string { return env[key] })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID is required")
}

func TestRedactSensitiveText(t *testing.T) {
	env := map[string]string{
		"EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD":     "oracle-password",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID":    "20250001",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME": "张三",
	}

	redacted := redactSensitiveText(
		"password=oracle-password student=20250001 name=张三",
		func(key string) string { return env[key] },
	)

	assert.NotContains(t, redacted, "oracle-password")
	assert.NotContains(t, redacted, "20250001")
	assert.NotContains(t, redacted, "张三")
	assert.Contains(t, redacted, "<redacted>")
}

func TestHashPrefixDoesNotExposeRawStudentID(t *testing.T) {
	hash := hashPrefix("20250001")

	assert.Len(t, hash, 12)
	assert.NotContains(t, hash, "20250001")
}
