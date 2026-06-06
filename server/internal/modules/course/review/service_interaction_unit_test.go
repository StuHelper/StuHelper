package review

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReply_PreflightContentValidation(t *testing.T) {
	svc := &Service{}

	_, err := svc.CreateReply(context.Background(), CreateReplyParams{Content: "   "})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	_, err = svc.CreateReply(context.Background(), CreateReplyParams{Content: `<script>alert(1)</script>`})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, err = svc.CreateReply(context.Background(), CreateReplyParams{
		Content: strings.Repeat("字", maxReplyContentRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)
}

func TestSaveDraftPreflightValidation(t *testing.T) {
	svc := &Service{}

	_, err := svc.SaveDraft(context.Background(), SaveDraftParams{
		Title: strings.Repeat("题", maxReviewTitleRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTitleTooLong)

	_, err = svc.SaveDraft(context.Background(), SaveDraftParams{
		Content: "   ",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	_, err = svc.SaveDraft(context.Background(), SaveDraftParams{
		Content: strings.Repeat("内", maxReviewContentRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)

	_, err = svc.SaveDraft(context.Background(), SaveDraftParams{
		Grade: "S",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidGrade)
}

func TestPostReviewRejectsInvalidGradeBeforeDependencies(t *testing.T) {
	svc := &Service{}

	_, err := svc.PostReview(context.Background(), PostReviewParams{
		Grade: "S",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidGrade)
}

func TestVoteReviewRejectsInvalidVoteTypeBeforeTx(t *testing.T) {
	svc := &Service{}

	_, err := svc.VoteReview(context.Background(), VoteReviewParams{
		ReviewID: "550e8400-e29b-41d4-a716-446655440901",
		UserHash: "u-voter-invalid",
		VoteType: "bookmark",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestGetUserVotesRejectsInvalidVoteTypeBeforeQuery(t *testing.T) {
	svc := &Service{}

	_, err := svc.GetUserVotes(context.Background(), GetUserVotesParams{
		UserHash: "u-voter-invalid",
		VoteType: "bookmark",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)
}
