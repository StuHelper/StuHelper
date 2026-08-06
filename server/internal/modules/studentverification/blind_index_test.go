package studentverification

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeRosterBlindIndexSeparatesSchoolAndFieldDomains(t *testing.T) {
	key := []byte("student-verification-test-hmac-key")

	studentHash, err := ComputeRosterBlindIndex(key, 4111010006, BlindIndexStudentID, "20990001")
	require.NoError(t, err)
	studentHashAgain, err := ComputeRosterBlindIndex(key, 4111010006, BlindIndexStudentID, "20990001")
	require.NoError(t, err)
	nameHash, err := ComputeRosterBlindIndex(key, 4111010006, BlindIndexName, "20990001")
	require.NoError(t, err)
	otherSchoolHash, err := ComputeRosterBlindIndex(key, 4111010007, BlindIndexStudentID, "20990001")
	require.NoError(t, err)

	assert.Equal(t, studentHash, studentHashAgain)
	assert.Len(t, studentHash, 64)
	assert.NotEqual(t, studentHash, nameHash)
	assert.NotEqual(t, studentHash, otherSchoolHash)
}
