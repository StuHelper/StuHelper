package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newValidationService(words []SensitiveWord) *Service {
	svc := &Service{filter: seededFilter(words)}
	svc.dimensionCache.Store(map[string]string{
		"teaching":   "教学质量",
		"difficulty": "课程难度",
	})
	return svc
}

func TestReviewService_ValidateAndSanitizeReview(t *testing.T) {
	ctx := context.Background()
	svc := newValidationService([]SensitiveWord{
		{Word: "warnword", Level: "warn"},
		{Word: "reviewword", Level: "review"},
		{Word: "blockword", Level: "block"},
	})

	_, _, _, _, err := svc.validateAndSanitizeReview(ctx, nil, "title", "content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRatingRequired)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"bad-key!": 5}, "title", "content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRating)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"unknown": 5}, "title", "content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRating)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 6}, "title", "content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRating)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "content", "2025-3")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTermID)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "<b></b>", "content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTitleEmpty)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "javascript:alert(1)", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "<b></b>", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	title, content, status, flag, err := svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "  <b>title</b>  ", "warnword content", "2025-2")
	require.NoError(t, err)
	assert.Equal(t, "title", title)
	assert.Equal(t, "warnword content", content)
	assert.Equal(t, StatusPublished, status)
	require.NotNil(t, flag)
	assert.Equal(t, ContentFlagWarn, *flag)

	_, _, status, flag, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "reviewword content", "2025-2")
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, status)
	require.NotNil(t, flag)
	assert.Equal(t, ContentFlagReview, *flag)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "blockword content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSensitiveContent)
}

func TestReviewService_ValidateDraftRatingValues(t *testing.T) {
	ctx := context.Background()
	svc := newValidationService(nil)

	require.NoError(t, svc.validateRatingValues(ctx, nil, false))
	require.NoError(t, svc.validateRatingValues(ctx, ReviewRatings{}, false))

	err := svc.validateRatingValues(ctx, nil, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRatingRequired)

	err = svc.validateRatingValues(ctx, ReviewRatings{"teaching": 6}, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRating)

	err = svc.validateRatingValues(ctx, ReviewRatings{"unknown": 4}, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRating)
}
