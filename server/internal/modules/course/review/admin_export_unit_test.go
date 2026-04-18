package review

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewExportHelpers(t *testing.T) {
	assert.True(t, isDangerousCSVPrefix('='))
	assert.True(t, isDangerousCSVPrefix('＠'))
	assert.False(t, isDangerousCSVPrefix('A'))

	assert.Equal(t, "'=1+1", sanitizeCSVField("=1+1"))
	assert.Equal(t, "safe", sanitizeCSVField("safe"))
	assert.Equal(t, "foo\n'@cmd", sanitizeCSVField("foo\n@cmd"))

	formatted := formatRatingsCSV(ReviewRatings{"overall": 5, "difficulty": 2})
	parts := strings.Split(formatted, ";")
	assert.Len(t, parts, 2)
	assert.Contains(t, formatted, "overall:5")
	assert.Contains(t, formatted, "difficulty:2")

	assert.Equal(t, "", formatRatingsCSV(nil))
}
