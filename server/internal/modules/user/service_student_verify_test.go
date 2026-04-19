package user

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStudentVerificationTransition(t *testing.T) {
	t.Run("nil profile allowed", func(t *testing.T) {
		require.NoError(t, validateStudentVerificationTransition(nil))
	})

	t.Run("verified blocked", func(t *testing.T) {
		err := validateStudentVerificationTransition(&Profile{VerificationStatus: StatusVerified})
		require.ErrorIs(t, err, ErrProfileAlreadyVerified)
	})

	t.Run("pending blocked", func(t *testing.T) {
		err := validateStudentVerificationTransition(&Profile{VerificationStatus: StatusPending})
		require.ErrorIs(t, err, ErrProfilePendingReview)
	})

	t.Run("unverified allowed", func(t *testing.T) {
		require.NoError(t, validateStudentVerificationTransition(&Profile{VerificationStatus: StatusUnverified}))
	})
}

func TestVerifyStudent_ManualAllowsEmptyCredentialsAndPersistsManualData(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(10006), schoolID)
			return &SchoolConfig{
				SchoolID:           10006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "manual",
				ManualFormFields: json.RawMessage(`[
					{"key":"studentID","label":"学号","type":"text","required":true},
					{"key":"department","label":"院系","type":"text","required":false}
				]`),
				Enabled: true,
			}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			if captured == nil {
				return nil, nil
			}
			return captured, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	profile, err := svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 10006,
		ManualFormData: map[string]any{
			"studentID":  "20240001",
			"department": "计算机学院",
		},
		Consent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.NotNil(t, captured)

	require.NotNil(t, captured.VerificationMethod)
	assert.Equal(t, VerifyMethodManual, *captured.VerificationMethod)
	assert.Equal(t, StatusPending, captured.VerificationStatus)
	assert.Equal(t, []string{"20240001"}, captured.StudentIDs)
	require.NotNil(t, captured.ActiveStudentID)
	assert.Equal(t, "20240001", *captured.ActiveStudentID)
	assert.JSONEq(t, `{"department":"计算机学院","studentID":"20240001"}`, string(captured.ManualFormData))
}

func TestVerifyStudent_ManualWithoutStudentIDDoesNotPersistBlankIdentifiers(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           30001,
				SchoolName:         "人工审核学校",
				VerificationMethod: VerifyMethodManual,
				Enabled:            true,
			}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 30001,
		Consent:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Nil(t, captured.StudentIDs)
	assert.Nil(t, captured.ActiveStudentID)
	assert.Nil(t, captured.ManualFormData)
}

func TestVerifyStudent_LDAPRequiresStudentID(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           30002,
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 30002,
		Password: "secret",
		Consent:  true,
	})
	assert.ErrorIs(t, err, ErrStudentIDRequired)
}

func TestVerifyStudent_LDAPRequiresPassword(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           30002,
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID:  30002,
		StudentID: "20240001",
		Consent:   true,
	})
	assert.ErrorIs(t, err, ErrPasswordRequired)
}

// ---------------------------------------------------------------------------
// ReviewStudentVerification
// ---------------------------------------------------------------------------

func TestReviewStudentVerification_RejectionReasonIsOptional(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{UserID: 1, VerificationStatus: StatusPending}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewStudentVerification(context.Background(), 1, false, "")
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, StatusRejected, captured.VerificationStatus)
	assert.Nil(t, captured.RejectionReason)

	err = svc.ReviewStudentVerification(context.Background(), 1, false, " ")
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Nil(t, captured.RejectionReason)

	reason := "材料不全"
	err = svc.ReviewStudentVerification(context.Background(), 1, false, reason)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.NotNil(t, captured.RejectionReason)
	assert.Equal(t, reason, *captured.RejectionReason)
}

func TestReviewStudentVerification_ApproveFlow(t *testing.T) {
	var capturedProfile *Profile

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{UserID: 1, VerificationStatus: StatusPending}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			capturedProfile = profile
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewStudentVerification(context.Background(), 1, true, "")
	require.NoError(t, err)
	require.NotNil(t, capturedProfile)
	assert.Equal(t, StatusVerified, capturedProfile.VerificationStatus)
	assert.NotNil(t, capturedProfile.VerifiedAt)
}

// ---------------------------------------------------------------------------
// UpdateSchoolConfig
// ---------------------------------------------------------------------------
