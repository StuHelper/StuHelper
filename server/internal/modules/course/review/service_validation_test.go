package review

import (
	"context"
	"strings"
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

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "<script>alert(1)</script> enough text", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "<iframe src=x></iframe>", "content", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "<b></b>", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, strings.Repeat("题", maxReviewTitleRunes+1), "content long enough", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTitleTooLong)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", "short", "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooShort)

	_, _, _, _, err = svc.validateAndSanitizeReview(ctx, ReviewRatings{"teaching": 5}, "title", strings.Repeat("内", maxReviewContentRunes+1), "2025-2")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)

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

func TestReviewService_AdminEditReviewRejectsDangerousContentBeforeDependencies(t *testing.T) {
	svc := &Service{}

	err := svc.AdminEditReview(context.Background(), AdminEditReviewParams{
		ReviewID: "review-dangerous-admin-edit",
		Title:    "title",
		Content:  "<script>alert(1)</script> enough text",
		Reason:   "reject dangerous content",
		AdminID:  "admin-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)
}

func TestReviewService_AdminEditReviewPreflightLengthValidation(t *testing.T) {
	svc := &Service{}

	err := svc.AdminEditReview(context.Background(), AdminEditReviewParams{
		ReviewID: "review-admin-edit-title-too-long",
		Title:    strings.Repeat("题", maxReviewTitleRunes+1),
		Content:  "管理员编辑内容",
		Reason:   "reject title",
		AdminID:  "admin-1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTitleTooLong)

	err = svc.AdminEditReview(context.Background(), AdminEditReviewParams{
		ReviewID: "review-admin-edit-content-too-long",
		Title:    "title",
		Content:  strings.Repeat("内", maxReviewContentRunes+1),
		Reason:   "reject content",
		AdminID:  "admin-1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)

	err = svc.AdminEditReview(context.Background(), AdminEditReviewParams{
		ReviewID: "review-admin-edit-reason-too-long",
		Title:    "title",
		Content:  "管理员编辑内容",
		Reason:   strings.Repeat("因", maxAdminEditReasonRunes+1),
		AdminID:  "admin-1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReasonTooLong)
}

func TestReviewService_SensitiveWordPreflightValidation(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	_, err := svc.CreateSensitiveWord(ctx, "   ", "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSensitiveWordInvalid)

	_, err = svc.CreateSensitiveWord(ctx, strings.Repeat("敏", maxSensitiveWordRunes+1), "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSensitiveWordInvalid)

	_, err = svc.CreateSensitiveWord(ctx, "敏感词", "Invalid Category", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSensitiveWordInvalid)

	err = svc.UpdateSensitiveWord(ctx, "missing", nil, nil, strPtr("fatal"), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSensitiveWordInvalid)
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
