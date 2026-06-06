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

	_, err := svc.CreateReply(context.Background(), CreateReplyParams{UserHash: "u-reply", Content: "   "})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	_, err = svc.CreateReply(context.Background(), CreateReplyParams{UserHash: "u-reply", Content: `<script>alert(1)</script>`})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, err = svc.CreateReply(context.Background(), CreateReplyParams{
		UserHash: "u-reply",
		Content:  strings.Repeat("字", maxReplyContentRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)
}

func TestSaveDraftPreflightValidation(t *testing.T) {
	svc := &Service{}

	_, err := svc.SaveDraft(context.Background(), SaveDraftParams{
		UserHash: "u-draft",
		Title:    strings.Repeat("题", maxReviewTitleRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTitleTooLong)

	_, err = svc.SaveDraft(context.Background(), SaveDraftParams{
		UserHash: "u-draft",
		Content:  "   ",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	_, err = svc.SaveDraft(context.Background(), SaveDraftParams{
		UserHash: "u-draft",
		Content:  strings.Repeat("内", maxReviewContentRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)

	_, err = svc.SaveDraft(context.Background(), SaveDraftParams{
		UserHash: "u-draft",
		Grade:    "S",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidGrade)
}

func TestPostReviewRejectsInvalidGradeBeforeDependencies(t *testing.T) {
	svc := &Service{}

	_, err := svc.PostReview(context.Background(), PostReviewParams{
		UserHash:             "u-post",
		AuthorInternalUserID: 1,
		Access:               fullReviewWriteAccess(1),
		Grade:                "S",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidGrade)
}

func TestUserScopedReviewInteractionsRejectMissingUserHashBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	err := svc.AddFavorite(ctx, AddFavoriteParams{CourseID: 1})
	require.ErrorIs(t, err, ErrUserIdentityRequired)

	err = svc.RemoveFavorite(ctx, " ", 1)
	require.ErrorIs(t, err, ErrUserIdentityRequired)

	favorited, err := svc.GetFavoriteStatus(ctx, "", 1)
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.False(t, favorited)

	favorites, err := svc.GetUserFavorites(ctx, GetUserFavoritesParams{})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.Nil(t, favorites)

	reviews, err := svc.GetUserReviews(ctx, GetUserReviewsParams{UserHash: " "})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.Nil(t, reviews)

	votes, err := svc.GetUserVotes(ctx, GetUserVotesParams{VoteType: voteTypeLike})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.Nil(t, votes)

	draft, err := svc.SaveDraft(ctx, SaveDraftParams{Title: strings.Repeat("题", maxReviewTitleRunes+1)})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.Nil(t, draft)

	draft, err = svc.GetDraft(ctx, "")
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.Nil(t, draft)

	err = svc.DeleteDraft(ctx, " ")
	require.ErrorIs(t, err, ErrUserIdentityRequired)

	reply, err := svc.CreateReply(ctx, CreateReplyParams{Content: "这是有效回复内容"})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
	assert.Nil(t, reply)

	err = svc.DeleteReply(ctx, DeleteReplyParams{ReplyID: "reply-id"})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
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

func TestReviewWriteOperationsRejectMissingUserIdentityBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	posted, err := svc.PostReview(ctx, PostReviewParams{
		AuthorInternalUserID: 1,
		Grade:                "S",
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)
	assert.Nil(t, posted)

	posted, err = svc.PostReview(ctx, PostReviewParams{
		UserHash:             "u-post",
		AuthorInternalUserID: 0,
		Access:               fullReviewWriteAccess(1),
		Grade:                "S",
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)
	assert.Nil(t, posted)

	_, err = svc.VoteReview(ctx, VoteReviewParams{VoteType: "bookmark"})
	require.ErrorIs(t, err, ErrUserIdentityRequired)

	err = svc.UpdateReview(ctx, UpdateReviewParams{})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)

	err = svc.DeleteReview(ctx, DeleteReviewParams{})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)
}

func TestReviewWriteOperationsFailClosedOnAccessFacts(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	_, err := svc.PostReview(ctx, PostReviewParams{
		UserHash:             "u-post-denied",
		AuthorInternalUserID: 1,
		Access: ReviewWriteAccess{
			InternalUserID: 1,
			CanPostReview:  false,
		},
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)

	_, err = svc.PostReview(ctx, PostReviewParams{
		UserHash:             "u-post-mismatch",
		AuthorInternalUserID: 2,
		Access: ReviewWriteAccess{
			InternalUserID: 1,
			CanPostReview:  true,
		},
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		UserHash: "u-edit-denied",
		Access: ReviewWriteAccess{
			InternalUserID: 1,
			CanEditOwn:     false,
		},
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)

	err = svc.DeleteReview(ctx, DeleteReviewParams{
		UserHash: "u-delete-denied",
		Access: ReviewWriteAccess{
			InternalUserID: 1,
			CanDeleteOwn:   false,
		},
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		UserHash: "u-edit-no-internal-id",
		Access: ReviewWriteAccess{
			InternalUserID: 0,
			CanEditOwn:     true,
		},
	})
	require.ErrorIs(t, err, ErrReviewWriteAccessDenied)
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
