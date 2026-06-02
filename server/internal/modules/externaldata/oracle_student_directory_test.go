package externaldata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeOracleStudentDirectoryConfig(t *testing.T) {
	cfg, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              1521,
		ServiceName:       "ORCLPDB1",
		Username:          "SYSTEM",
		Password:          "secret",
		Schema:            "usr_jwbiz",
		Table:             "t_xs_jbxx",
		StudentIDColumn:   "xh",
		StudentNameColumn: "xm",
	})
	require.NoError(t, err)

	assert.Equal(t, "USR_JWBIZ", cfg.Schema)
	assert.Equal(t, "T_XS_JBXX", cfg.Table)
	assert.Equal(t, "XH", cfg.StudentIDColumn)
	assert.Equal(t, "XM", cfg.StudentNameColumn)
	assert.Equal(t, defaultOracleQueryTimeout, cfg.QueryTimeout)
}

func TestNormalizeOracleStudentDirectoryConfigRejectsUnsafeIdentifiers(t *testing.T) {
	_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              1521,
		ServiceName:       "ORCLPDB1",
		Username:          "SYSTEM",
		Password:          "secret",
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX;DROP TABLE USERS",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
		QueryTimeout:      time.Second,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oracle table identifier is invalid")
}

func TestBuildOracleStudentLookupQuery(t *testing.T) {
	query := buildOracleStudentLookupQuery(OracleStudentDirectoryConfig{
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
	})

	assert.Equal(
		t,
		"SELECT XH, XM FROM USR_JWBIZ.T_XS_JBXX WHERE XH = :1 FETCH FIRST 1 ROWS ONLY",
		query,
	)
}

func TestBuildOracleStudentProbeQuery(t *testing.T) {
	query := buildOracleStudentProbeQuery(OracleStudentDirectoryConfig{
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
	})

	assert.Equal(
		t,
		"SELECT 1 FROM USR_JWBIZ.T_XS_JBXX WHERE XH IS NOT NULL AND XM IS NOT NULL FETCH FIRST 1 ROWS ONLY",
		query,
	)
}
