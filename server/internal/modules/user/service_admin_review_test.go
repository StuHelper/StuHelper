package user

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewStudentVerification_RejectStoresReviewMeta(t *testing.T) {
	var capturedProfile *Profile
	const schoolID = int64(4111010006)

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			method := VerifyMethodManual
			return &Profile{
				UserID:             1001,
				SchoolID:           int64Ptr(schoolID),
				VerificationStatus: StatusPending,
				VerificationMethod: &method,
			}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			copied := *profile
			capturedProfile = &copied
			return nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewStudentVerification(context.Background(), 1001, schoolID, false, "  材料信息不一致  ")
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
	const schoolID = int64(4111010006)

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			method := VerifyMethodManual
			rejectionReason := "证件不清晰"
			return &Profile{
				UserID:             1002,
				SchoolID:           int64Ptr(schoolID),
				VerificationStatus: StatusPending,
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

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewStudentVerification(context.Background(), 1002, schoolID, true, "")
	require.NoError(t, err)
	require.NotNil(t, capturedProfile)

	assert.Equal(t, StatusVerified, capturedProfile.VerificationStatus)
	assert.NotNil(t, capturedProfile.VerifiedAt)
	assert.Nil(t, capturedProfile.RejectionReason)
	assert.NotNil(t, capturedProfile.ReviewedAt)
}

func TestReviewStudentVerification_ApproveSchoolEmailOTPEnsuresCredential(t *testing.T) {
	method := VerifyMethodSchoolEmailOTP
	schoolID := int64(4111010006)
	var capturedCredential *VerificationCredentialProjection

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{
				UserID:             1003,
				SchoolID:           &schoolID,
				VerificationStatus: StatusPending,
				VerificationMethod: &method,
				ManualFormData:     json.RawMessage(`{"schoolEmail":"20250001@buaa.edu.cn"}`),
			}, nil
		},
		onEnsureVerificationCredentialTx: func(_ context.Context, _ pgx.Tx, credential VerificationCredentialProjection) error {
			copy := credential
			capturedCredential = &copy
			return nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewStudentVerification(context.Background(), 1003, schoolID, true, "")
	require.NoError(t, err)
	require.NotNil(t, capturedCredential)
	assert.Equal(t, int64(1003), capturedCredential.UserID)
	assert.Equal(t, schoolID, capturedCredential.SchoolID)
	assert.Equal(t, userVerificationCredentialKindSchoolEmailOTP, capturedCredential.Kind)
	assert.NotEmpty(t, capturedCredential.SubjectHash)
	assert.Equal(t, "2******1@buaa.edu.cn", capturedCredential.SubjectDisplay)
}

func TestReviewIdentity_RejectSetsReviewedAtEvenWithoutReason(t *testing.T) {
	var (
		capturedReviewedAt *time.Time
		capturedVerifiedAt *time.Time
	)

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 88, Verified: false}, nil
		},
		onUpdateIdentityReviewStatus: func(_ context.Context, _ int64, approved bool, _ *string, reviewedAt *time.Time, verifiedAt *time.Time, rejectionReason *string) error {
			assert.False(t, approved)
			assert.Nil(t, rejectionReason)
			capturedReviewedAt = reviewedAt
			capturedVerifiedAt = verifiedAt
			return nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewIdentity(context.Background(), 88, false, "")
	require.NoError(t, err)
	assert.NotNil(t, capturedReviewedAt)
	assert.Nil(t, capturedVerifiedAt)
}

func TestReviewIdentity_RejectsStaleOrRepeatedReview(t *testing.T) {
	reviewedAt := time.Now().Add(-time.Minute)
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{
				UserID:     88,
				Verified:   false,
				ReviewedAt: &reviewedAt,
			}, nil
		},
		onUpdateIdentityReviewStatus: func(
			context.Context,
			int64,
			bool,
			*string,
			*time.Time,
			*time.Time,
			*string,
		) error {
			t.Fatal("non-pending identity must not be reviewed")
			return nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = service.ReviewIdentity(context.Background(), 88, true, "")
	require.ErrorIs(t, err, ErrVerificationReviewStateConflict)
}

func TestReviewStudentVerification_RevalidatesPendingStateAndSchoolInsideTransaction(t *testing.T) {
	const (
		userID           = int64(1004)
		authorizedSchool = int64(4111010006)
		changedSchool    = int64(4111010007)
	)

	tests := []struct {
		name    string
		profile *Profile
	}{
		{
			name: "profile is no longer pending",
			profile: &Profile{
				UserID:             userID,
				SchoolID:           int64Ptr(authorizedSchool),
				VerificationStatus: StatusVerified,
			},
		},
		{
			name: "profile school changed after authorization",
			profile: &Profile{
				UserID:             userID,
				SchoolID:           int64Ptr(changedSchool),
				VerificationStatus: StatusPending,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				onGetProfileByUserIDForUpdateTx: func(_ context.Context, _ pgx.Tx, _ int64) (*Profile, error) {
					return tt.profile, nil
				},
				onUpdateProfileTx: func(context.Context, pgx.Tx, *Profile) error {
					t.Fatal("stale or cross-school review must not update the profile")
					return nil
				},
			}
			service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
			require.NoError(t, err)

			err = service.ReviewStudentVerification(
				context.Background(),
				userID,
				authorizedSchool,
				true,
				"",
			)
			require.ErrorIs(t, err, ErrVerificationReviewStateConflict)
		})
	}
}
