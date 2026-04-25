package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService_NilRepo(t *testing.T) {
	_, err := NewService(nil, []byte("secret-key-32-chars-long-enough!"), &fakeEncryptor{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo must not be nil")
}

func TestNewService_EmptyHMACKey(t *testing.T) {
	_, err := NewService(&mockRepo{}, nil, &fakeEncryptor{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hmacKey must not be empty")
}

func TestNewService_NilDocCipher(t *testing.T) {
	_, err := NewService(&mockRepo{}, []byte("secret-key-32-chars-long-enough!"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docCipher must not be nil")
}

func TestNewService_ValidConstruction(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("secret-key-32-chars-long-enough!"), &fakeEncryptor{})
	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.docCipher)
}

func TestBindPhone_StoresEncryptedPhoneAndMaskedProfile(t *testing.T) {
	enc := &fakeEncryptor{}
	var (
		capturedPhoneHash string
		capturedPhoneEnc  []byte
		updatedProfile    *Profile
	)

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			return &Profile{UserID: userID}, nil
		},
		onSetUserPhone: func(_ context.Context, userID int64, phoneEnc []byte, phoneHash string) error {
			assert.Equal(t, int64(42), userID)
			capturedPhoneHash = phoneHash
			capturedPhoneEnc = append([]byte(nil), phoneEnc...)
			return nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			cp := *profile
			updatedProfile = &cp
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), enc)
	require.NoError(t, err)

	err = svc.BindPhone(context.Background(), 42, "13800138000")
	require.NoError(t, err)
	require.True(t, enc.called)
	assert.Equal(t, "13800138000", enc.lastInput)
	assert.Equal(t, []byte("encrypted:13800138000"), capturedPhoneEnc)
	assert.Len(t, capturedPhoneHash, 64)
	require.NotNil(t, updatedProfile)
	require.NotNil(t, updatedProfile.Phone)
	assert.Equal(t, "138****8000", *updatedProfile.Phone)
	assert.True(t, updatedProfile.PhoneVerified)
}

func TestBindPhone_ReturnsConflictWhenAlreadyBound(t *testing.T) {
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			return &Profile{UserID: userID}, nil
		},
		onSetUserPhone: func(_ context.Context, _ int64, _ []byte, _ string) error {
			return ErrPhoneAlreadyBound
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.BindPhone(context.Background(), 7, "13800138000")
	require.ErrorIs(t, err, ErrPhoneAlreadyBound)
}

// ---------------------------------------------------------------------------
// computePersonUID
// ---------------------------------------------------------------------------

func TestComputePersonUID_Consistency(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	uid1 := svc.computePersonUID("MAINLAND_ID", "110101199001011234")
	uid2 := svc.computePersonUID("MAINLAND_ID", "110101199001011234")
	assert.Equal(t, uid1, uid2)

	uid3 := svc.computePersonUID("PASSPORT", "110101199001011234")
	assert.NotEqual(t, uid1, uid3)
}

// ---------------------------------------------------------------------------
// SubmitIdentity — 加密回归测试（调用真实 Service 方法）
// ---------------------------------------------------------------------------

func TestSubmitIdentity_EncryptAndWriteCiphertext(t *testing.T) {
	enc := &fakeEncryptor{}
	var capturedIdentity *IdentityRecord
	callCount := 0

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			callCount++
			if callCount == 1 {
				return nil, nil // 首次查询：不存在
			}
			// 第二次查询（CreateIdentity 之后的 reload）
			return &IdentityStatus{
				UserID:   42,
				DocType:  "MAINLAND_ID",
				RealName: "张三",
				Verified: false,
			}, nil
		},
		onCreateIdentity: func(_ context.Context, identity *IdentityRecord) error {
			capturedIdentity = identity
			return nil
		},
		onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return nil, nil
		},
		onFindAcademicStudentsByPersonUIDFromTable: func(_ context.Context, _, _, _ string) ([]AcademicStudent, error) {
			return nil, nil // 无学籍匹配
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), enc)
	require.NoError(t, err)

	docNumber := "110101199001011234"
	result, err := svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: docNumber,
		RealName:  "张三",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// 断言 Encryptor 被 SubmitIdentity 调用
	assert.True(t, enc.called, "docCipher.Encrypt 必须被 SubmitIdentity 调用")
	assert.Equal(t, docNumber, enc.lastInput, "Encrypt 接收原始证件号")

	// 核心回归断言：写入 repo 的 DocNumberEnc 是加密结果，而非明文
	require.NotNil(t, capturedIdentity, "CreateIdentity 必须被调用")
	assert.Equal(t, []byte("encrypted:"+docNumber), capturedIdentity.DocNumberEnc,
		"写入仓库的 DocNumberEnc 必须是 Encryptor 的加密输出")
	assert.NotEqual(t, []byte(docNumber), capturedIdentity.DocNumberEnc,
		"写入仓库的 DocNumberEnc 不能是原始证件号明文")

	// PersonUID 应为 HMAC 结果，非空且非原始值
	assert.NotEmpty(t, capturedIdentity.PersonUID)
	assert.NotEqual(t, docNumber, capturedIdentity.PersonUID)
}

func TestSubmitIdentity_AlreadyExists(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{Verified: false}, nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 1, SubmitIdentityRequest{
		DocType: DocTypeMainlandID, DocNumber: "123", RealName: "test",
	})
	assert.ErrorIs(t, err, ErrIdentityAlreadyExists)
}

func TestSubmitIdentity_RejectedIdentityAllowsResubmission(t *testing.T) {
	var updated *IdentityRecord
	callCount := 0
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			callCount++
			if callCount == 1 {
				now := time.Now().Add(-time.Hour)
				rejectionReason := "照片不清晰"
				return &IdentityStatus{
					UserID:          42,
					DocType:         DocTypePassport,
					RealName:        "旧实名",
					ReviewedAt:      &now,
					RejectionReason: &rejectionReason,
				}, nil
			}
			return &IdentityStatus{
				UserID:   42,
				DocType:  DocTypePassport,
				RealName: "新实名",
				Verified: false,
			}, nil
		},
		onUpdateIdentitySubmission: func(_ context.Context, identity *IdentityRecord) error {
			copied := *identity
			updated = &copied
			return nil
		},
		onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return nil, nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	front := "identities/42/2026/04/front.png"
	back := "identities/42/2026/04/back.png"
	selfie := "identities/42/2026/04/selfie.png"
	result, err := svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:        DocTypePassport,
		DocNumber:      "P12345678",
		RealName:       "新实名",
		DocPhotoFront:  &front,
		DocPhotoBack:   &back,
		DocPhotoSelfie: &selfie,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, updated)
	assert.Equal(t, "新实名", updated.RealName)
	assert.False(t, updated.Verified)
	assert.Nil(t, updated.VerifyMethod)
	assert.Nil(t, updated.ReviewedAt)
	assert.Nil(t, updated.VerifiedAt)
	assert.Nil(t, updated.RejectionReason)
}

func TestSubmitIdentity_AlreadyVerified(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{Verified: true}, nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 1, SubmitIdentityRequest{
		DocType: DocTypeMainlandID, DocNumber: "123", RealName: "test",
	})
	assert.ErrorIs(t, err, ErrIdentityAlreadyVerified)
}

func TestSubmitIdentity_RejectsLegacyPhotoValuesForNewSubmission(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	legacyFront := "https://cdn.example.com/front.png"
	legacyBack := "data:image/png;base64,ZmFrZQ=="
	legacySelfie := "http://cdn.example.com/selfie.png"
	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:        DocTypePassport,
		DocNumber:      "P12345678",
		RealName:       "张三",
		DocPhotoFront:  &legacyFront,
		DocPhotoBack:   &legacyBack,
		DocPhotoSelfie: &legacySelfie,
	})
	assert.ErrorIs(t, err, ErrIdentityPhotoInvalidRef)
}

// ---------------------------------------------------------------------------
// ReviewIdentity — 拒绝理由可选 + 完整审批流程
// ---------------------------------------------------------------------------

func TestReviewIdentity_RejectionReasonIsOptional(t *testing.T) {
	var capturedReason *string
	var capturedReviewedAt *time.Time

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: false}, nil
		},
		onUpdateIdentityReviewStatus: func(_ context.Context, _ int64, approved bool, _ *string, reviewedAt *time.Time, _ *time.Time, rejectionReason *string) error {
			assert.False(t, approved)
			capturedReviewedAt = reviewedAt
			capturedReason = rejectionReason
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 1, false, "")
	require.NoError(t, err)
	assert.Nil(t, capturedReason)
	assert.NotNil(t, capturedReviewedAt)

	err = svc.ReviewIdentity(context.Background(), 1, false, "  ")
	require.NoError(t, err)
	assert.Nil(t, capturedReason)

	reason := "材料不合规"
	err = svc.ReviewIdentity(context.Background(), 1, false, reason)
	require.NoError(t, err)
	require.NotNil(t, capturedReason)
	assert.Equal(t, reason, *capturedReason)
}

func TestReviewIdentity_ApproveFlow(t *testing.T) {
	var updatedApproved bool
	var updatedMethod *string
	var updatedReviewedAt *time.Time
	var updatedVerifiedAt *time.Time

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: false}, nil
		},
		onUpdateIdentityReviewStatus: func(_ context.Context, _ int64, approved bool, verifyMethod *string, reviewedAt *time.Time, verifiedAt *time.Time, _ *string) error {
			updatedApproved = approved
			updatedMethod = verifyMethod
			updatedReviewedAt = reviewedAt
			updatedVerifiedAt = verifiedAt
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 1, true, "")
	require.NoError(t, err)
	assert.True(t, updatedApproved)
	require.NotNil(t, updatedMethod)
	assert.Equal(t, VerifyMethodManual, *updatedMethod)
	assert.NotNil(t, updatedReviewedAt)
	assert.NotNil(t, updatedVerifiedAt)
}

func TestReviewIdentity_RejectFlow(t *testing.T) {
	var updatedApproved bool
	var updatedReason *string
	var updatedReviewedAt *time.Time
	var updatedVerifiedAt *time.Time

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: false}, nil
		},
		onUpdateIdentityReviewStatus: func(_ context.Context, _ int64, approved bool, _ *string, reviewedAt *time.Time, verifiedAt *time.Time, rejectionReason *string) error {
			updatedApproved = approved
			updatedReviewedAt = reviewedAt
			updatedVerifiedAt = verifiedAt
			updatedReason = rejectionReason
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 1, false, "材料不清晰")
	require.NoError(t, err)
	assert.False(t, updatedApproved)
	require.NotNil(t, updatedReason)
	assert.Equal(t, "材料不清晰", *updatedReason)
	assert.NotNil(t, updatedReviewedAt)
	assert.Nil(t, updatedVerifiedAt)
}

func TestReviewIdentity_NotFoundReturnsError(t *testing.T) {
	// GetIdentityStatusByUserID 返回 nil → ErrIdentityNotFound
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 999, true, "")
	assert.ErrorIs(t, err, ErrIdentityNotFound)
}

// ---------------------------------------------------------------------------
// VerifyStudent
// ---------------------------------------------------------------------------
