package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewStudentVerification_RejectClearsVerifiedAtAndStoresReviewMeta(t *testing.T) {
	previousVerifiedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	var capturedProfile *Profile

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			method := VerifyMethodManual
			return &Profile{
				UserID:             1001,
				VerificationStatus: StatusVerified,
				VerificationMethod: &method,
				VerifiedAt:         &previousVerifiedAt,
			}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			copied := *profile
			capturedProfile = &copied
			return nil
		},
	}

	service, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewStudentVerification(context.Background(), 1001, false, "  材料信息不一致  ")
	require.NoError(t, err)
	require.NotNil(t, capturedProfile)

	assert.Equal(t, StatusRejected, capturedProfile.VerificationStatus)
	assert.Nil(t, capturedProfile.VerifiedAt)
	require.NotNil(t, capturedProfile.RejectionReason)
	assert.Equal(t, "材料信息不一致", *capturedProfile.RejectionReason)
	assert.NotNil(t, capturedProfile.ReviewedAt)
}

func TestReviewStudentVerification_ApproveClearsRejectionReasonAndSetsReviewMeta(t *testing.T) {
	var capturedProfile *Profile

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			method := VerifyMethodManual
			rejectionReason := "证件不清晰"
			return &Profile{
				UserID:             1002,
				VerificationStatus: StatusRejected,
				VerificationMethod: &method,
				RejectionReason:    &rejectionReason,
			}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			copied := *profile
			capturedProfile = &copied
			return nil
		},
	}

	service, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewStudentVerification(context.Background(), 1002, true, "")
	require.NoError(t, err)
	require.NotNil(t, capturedProfile)

	assert.Equal(t, StatusVerified, capturedProfile.VerificationStatus)
	assert.NotNil(t, capturedProfile.VerifiedAt)
	assert.Nil(t, capturedProfile.RejectionReason)
	assert.NotNil(t, capturedProfile.ReviewedAt)
}
