package review

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPaginatedReviewListData_NormalizesNilList(t *testing.T) {
	data := buildPaginatedReviewListData(nil, 0)
	assert.NotNil(t, data.List)
	assert.Empty(t, data.List)
	assert.Equal(t, 0, data.Total)
}

func TestBuildGroupedReviewListData_UsesSpecShape(t *testing.T) {
	result := &BatchCourseReviewsResult{
		Reviews: map[int64][]Review{
			1: {{
				ID:       "r1",
				CourseID: 1,
				Title:    "Title",
				Content:  "Content",
				Status:   StatusPublished,
			}},
		},
		Totals: map[int64]int{
			1: 1,
			2: 0,
		},
	}

	grouped := buildGroupedReviewListData([]int64{1, 2}, result, ReviewAccessFacts{
		Authenticated: true,
		CanViewFull:   true,
	})

	raw, err := json.Marshal(grouped)
	require.NoError(t, err)

	var decoded map[string]map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Contains(t, decoded, "1")
	assert.Contains(t, decoded, "2")
	assert.Contains(t, decoded["1"], "list")
	assert.Contains(t, decoded["1"], "total")
	assert.NotContains(t, decoded["1"], "authenticated")
	assert.NotContains(t, decoded, "data")
}

func TestRatingResponses_DoNotExposeRadarChart(t *testing.T) {
	courseRaw, err := json.Marshal(CourseRatingStatsResponse{})
	require.NoError(t, err)
	assert.NotContains(t, string(courseRaw), "radarChart")

	teacherRaw, err := json.Marshal(TeacherRatingStatsResponse{})
	require.NoError(t, err)
	assert.NotContains(t, string(teacherRaw), "radarChart")
}
