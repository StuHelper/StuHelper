package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAcademicDBTableName_RequiresExplicitValue(t *testing.T) {
	_, err := normalizeAcademicDBTableName(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be provided")

	empty := "   "
	_, err = normalizeAcademicDBTableName(&empty)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be provided")
}

func TestNormalizeAcademicDBTableName_ValidConfiguredTable(t *testing.T) {
	table := "academic.school_students_2026"
	value, err := normalizeAcademicDBTableName(&table)
	require.NoError(t, err)
	assert.Equal(t, table, value)
}

func TestNormalizeAcademicDBTableName_RejectsInvalidIdentifier(t *testing.T) {
	table := `academic.students;DROP TABLE users;`
	_, err := normalizeAcademicDBTableName(&table)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid identifier characters")

	table = `academic."students"`
	_, err = normalizeAcademicDBTableName(&table)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid identifier characters")
}

func TestNormalizeConfiguredAcademicDBTable_EmptyMeansNull(t *testing.T) {
	raw := " "
	value, err := normalizeConfiguredAcademicDBTable(&raw)
	require.NoError(t, err)
	assert.Nil(t, value)

	raw = "academic.students"
	value, err = normalizeConfiguredAcademicDBTable(&raw)
	require.NoError(t, err)
	require.NotNil(t, value)
	assert.Equal(t, "academic.students", *value)
}
