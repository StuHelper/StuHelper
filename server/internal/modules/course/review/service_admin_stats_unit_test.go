package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupRatingStatsAndBuildRatingStats(t *testing.T) {
	svc := &Service{}
	resp := &TeacherRatingStatsResponse{}

	term1 := "2025-1"
	term2 := "2025-2"
	stats := []TeacherRatingStats{
		{TermID: nil, DimensionKey: "teaching", AvgRating: 4.5, RatingCount: 10},
		{TermID: nil, DimensionKey: "difficulty", AvgRating: 3.5, RatingCount: 8},
		{TermID: &term2, DimensionKey: "teaching", AvgRating: 4.8, RatingCount: 4},
		{TermID: &term1, DimensionKey: "teaching", AvgRating: 4.0, RatingCount: 6},
	}
	dimensions := []RatingDimension{
		{Key: "difficulty", Name: "课程难度"},
		{Key: "teaching", Name: "教学质量"},
	}

	svc.groupRatingStats(stats, dimensions, resp)
	require.Len(t, resp.Overall.Dimensions, 2)
	require.Len(t, resp.ByTerm, 2)
	assert.Equal(t, "2025-1", resp.ByTerm[0].TermID)
	assert.Equal(t, "2025-2", resp.ByTerm[1].TermID)
	assert.Equal(t, "教学质量", resp.Overall.Dimensions[1].Name)

	svc.buildRatingStats(resp)
	require.NotNil(t, resp.AvgRating)
	assert.InDelta(t, 4.0, *resp.AvgRating, 0.001)
	require.Len(t, resp.RatingTrend, 2)
	assert.Equal(t, "2025-1", resp.RatingTrend[0].TermID)
	assert.InDelta(t, 4.0, resp.RatingTrend[0].AvgRating, 0.001)
	assert.Equal(t, "2025-2", resp.RatingTrend[1].TermID)
	assert.InDelta(t, 4.8, resp.RatingTrend[1].AvgRating, 0.001)
}

func TestBuildRatingStats_EmptyDimensions(t *testing.T) {
	svc := &Service{}
	resp := &TeacherRatingStatsResponse{
		Overall: TermRatingStats{TermID: "", TermName: "overall"},
		ByTerm: []TermRatingStats{
			{TermID: "2025-1", TermName: "2025-1", Dimensions: nil},
		},
	}

	svc.buildRatingStats(resp)
	assert.Nil(t, resp.AvgRating)
	assert.Empty(t, resp.RatingTrend)
}
