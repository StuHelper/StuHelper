package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewService_AdminMutationsRequireAdminIdentityBeforeDependencies(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	_, err := svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{
		ReviewID: "review-admin-identity",
		Action:   "hide",
		AdminID:  "   ",
	})
	require.ErrorIs(t, err, ErrAdminIdentityRequired)

	_, err = svc.AdminUpdateReply(ctx, AdminUpdateReplyParams{
		ReplyID: "reply-admin-identity",
		Action:  "restore",
	})
	require.ErrorIs(t, err, ErrAdminIdentityRequired)

	err = svc.AdminEditReview(ctx, AdminEditReviewParams{
		ReviewID: "review-admin-edit-identity",
		Title:    "title",
		Content:  "admin edited content",
		Reason:   "normalize content",
		AdminID:  " ",
	})
	require.ErrorIs(t, err, ErrAdminIdentityRequired)

	_, err = svc.BatchUpdateReviews(ctx, BatchUpdateReviewsParams{
		IDs:    []string{"review-batch-admin-identity"},
		Action: "hide",
	})
	require.ErrorIs(t, err, ErrAdminIdentityRequired)

	err = svc.ClearContentFlag(ctx, "review-flag-admin-identity", " ")
	require.ErrorIs(t, err, ErrAdminIdentityRequired)

	err = svc.LogOperation(ctx, LogOperationParams{
		Action:       "delete",
		ResourceType: "review",
		ResourceID:   "review-log-admin-identity",
	})
	require.ErrorIs(t, err, ErrAdminIdentityRequired)
}
