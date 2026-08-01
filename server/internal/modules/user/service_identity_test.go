package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
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

func TestBindPhone_WritesCasdoorAndUserPhoneProjection(t *testing.T) {
	var (
		syncedSubject string
		syncedPhone   string
		phoneEnc      []byte
		phoneHash     string
	)
	hmacKey := []byte("test-hmac-key-at-least-32-chars!")
	expectedHash, err := phoneutil.HashLookupWithKey("13800138000", hmacKey)
	require.NoError(t, err)

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			t.Fatalf("BindPhone must not require student profile, got userID %d", userID)
			return nil, nil
		},
		onGetCasdoorSubject: func(_ context.Context, userID int64) (string, error) {
			assert.Equal(t, int64(42), userID)
			return "casdoor-subject-42", nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			t.Fatal("BindPhone must not update user_profiles; phone projection is stored on users")
			return nil
		},
		onSetUserPhone: func(_ context.Context, userID int64, gotPhoneEnc []byte, gotPhoneHash string) error {
			assert.Equal(t, int64(42), userID)
			phoneEnc = append([]byte(nil), gotPhoneEnc...)
			phoneHash = gotPhoneHash
			return nil
		},
	}
	gateway := profileIdentitySyncFunc(func(_ context.Context, subject, phone string) error {
		syncedSubject = subject
		syncedPhone = phone
		return nil
	})

	svc, err := NewService(
		repo,
		hmacKey,
		&fakeEncryptor{},
		WithProfileIdentitySyncGateway(gateway),
	)
	require.NoError(t, err)

	err = svc.BindPhone(context.Background(), 42, "13800138000")
	require.NoError(t, err)
	assert.Equal(t, "casdoor-subject-42", syncedSubject)
	assert.Equal(t, "+8613800138000", syncedPhone)
	assert.Equal(t, []byte("encrypted:138****8000"), phoneEnc)
	assert.Equal(t, expectedHash, phoneHash)
}

func TestBindPhone_DuplicateProjectionDoesNotUpdateCasdoor(t *testing.T) {
	hmacKey := []byte("test-hmac-key-at-least-32-chars!")
	expectedHash, err := phoneutil.HashLookupWithKey("13800138000", hmacKey)
	require.NoError(t, err)
	casdoorCalls := 0

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			t.Fatalf("BindPhone must reject duplicate projection before loading profile, got userID %d", userID)
			return nil, nil
		},
		onEnsureUserPhoneAvailable: func(_ context.Context, userID int64, phoneHash string) error {
			assert.Equal(t, int64(42), userID)
			assert.Equal(t, expectedHash, phoneHash)
			return ErrPhoneAlreadyBound
		},
		onGetCasdoorSubject: func(context.Context, int64) (string, error) {
			t.Fatal("duplicate phone must be rejected before loading Casdoor subject")
			return "", nil
		},
		onSetUserPhone: func(context.Context, int64, []byte, string) error {
			t.Fatal("duplicate phone must not update local projection")
			return nil
		},
		onUpdateProfile: func(context.Context, *Profile) error {
			t.Fatal("duplicate phone must not update profile projection")
			return nil
		},
	}
	gateway := profileIdentitySyncFunc(func(context.Context, string, string) error {
		casdoorCalls++
		return nil
	})

	svc, err := NewService(
		repo,
		hmacKey,
		&fakeEncryptor{},
		WithProfileIdentitySyncGateway(gateway),
	)
	require.NoError(t, err)

	err = svc.BindPhone(context.Background(), 42, "13800138000")
	require.ErrorIs(t, err, ErrPhoneAlreadyBound)
	assert.Zero(t, casdoorCalls)
}

func TestBindPhone_RequiresIdentitySyncGateway(t *testing.T) {
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			t.Fatalf("BindPhone must reject missing identity sync gateway before loading profile, got userID %d", userID)
			return nil, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.BindPhone(context.Background(), 7, "13800138000")
	require.ErrorIs(t, err, ErrProfileIdentitySyncMissing)
}

type profileIdentitySyncFunc func(ctx context.Context, subject, phone string) error

func (f profileIdentitySyncFunc) UpdatePhone(ctx context.Context, subject, phone string) error {
	return f(ctx, subject, phone)
}

// ---------------------------------------------------------------------------
// computePersonUID
// ---------------------------------------------------------------------------

func TestComputePersonUID_Consistency(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	uid1 := svc.computePersonUID("MAINLAND_ID", "110101199001011237")
	uid2 := svc.computePersonUID("MAINLAND_ID", "110101199001011237")
	assert.Equal(t, uid1, uid2)

	uid3 := svc.computePersonUID("PASSPORT", "110101199001011237")
	assert.NotEqual(t, uid1, uid3)
}

// ---------------------------------------------------------------------------
// SubmitIdentity — 加密回归测试（调用真实 Service 方法）
// ---------------------------------------------------------------------------

func TestIdentityService_RejectsInvalidUserIDBeforeRepositoryAccess(t *testing.T) {
	enc := &fakeEncryptor{}
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(context.Context, int64) (*IdentityStatus, error) {
			t.Fatal("invalid user ID must be rejected before repository access")
			return nil, nil
		},
		onCreateIdentity: func(context.Context, *IdentityRecord) error {
			t.Fatal("invalid user ID must not create identity records")
			return nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), enc)
	require.NoError(t, err)

	_, err = svc.GetIdentity(context.Background(), 0)
	assert.ErrorIs(t, err, ErrUserIDInvalid)

	_, err = svc.SubmitIdentity(context.Background(), -1, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "11010519491231002X",
		RealName:  "张三",
	})
	assert.ErrorIs(t, err, ErrUserIDInvalid)
	assert.False(t, enc.called)
}

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

	store := &fakeIdentityPhotoStore{presignURL: "https://storage.example.test/identity/photo.png"}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		enc,
		WithIdentityPhotoStore(store),
	)
	require.NoError(t, err)

	docNumber := "110101199001011237"
	front := "identities/42/2026/04/1777777777777777001-front.png"
	selfie := "identities/42/2026/04/1777777777777777002-selfie.png"
	result, err := svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:        DocTypeMainlandID,
		DocNumber:      docNumber,
		RealName:       "张三",
		DocPhotoFront:  &front,
		DocPhotoSelfie: &selfie,
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

func TestSubmitIdentity_MainlandIDRequiresEvidenceWhenAcademicMatchUnavailable(t *testing.T) {
	repo := &mockRepo{
		onCreateIdentity: func(context.Context, *IdentityRecord) error {
			t.Fatal("identity without automatic proof or manual evidence must not be persisted")
			return nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
	)
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "110101199001011237",
		RealName:  "张三",
	})
	assert.ErrorIs(t, err, ErrPhotoRequired)
}

func TestSubmitIdentity_RejectsInvalidMainlandIDNumber(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "not-an-id-card",
		RealName:  "张三",
	})
	assert.ErrorIs(t, err, ErrIdentityDocNumberInvalid)

	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "110101200001010011",
		RealName:  "张三",
	})
	assert.ErrorIs(t, err, ErrIdentityDocNumberInvalid)
}

func TestSubmitIdentity_RejectsInvalidRealName(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "11010519491231002X",
		RealName:  "   ",
	})
	assert.ErrorIs(t, err, ErrIdentityRealNameInvalid)

	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "11010519491231002X",
		RealName:  strings.Repeat("名", maxIdentityRealNameRunes+1),
	})
	assert.ErrorIs(t, err, ErrIdentityRealNameInvalid)

	for _, unsafeName := range []string{
		"张\u200b三",
		"张\n三",
		"张" + string(rune(0xE000)) + "三",
		string([]byte{0xff}),
	} {
		_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
			DocType:   DocTypeMainlandID,
			DocNumber: "11010519491231002X",
			RealName:  unsafeName,
		})
		assert.ErrorIs(t, err, ErrIdentityRealNameInvalid, "%q", unsafeName)
	}
}

func TestSubmitIdentity_RejectsUnsafeNonMainlandDocumentNumbers(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	for _, unsafeDocNumber := range []string{
		"P123 456",
		"P123\n456",
		"P123\u200b456",
		"P123" + string(rune(0xE000)) + "456",
		string([]byte{0xff}),
	} {
		_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
			DocType:   DocTypePassport,
			DocNumber: unsafeDocNumber,
			RealName:  "张三",
		})
		assert.ErrorIs(t, err, ErrIdentityDocNumberInvalid, "%q", unsafeDocNumber)
	}
}

func TestSubmitIdentity_NormalizesMainlandIDAndRealName(t *testing.T) {
	enc := &fakeEncryptor{}
	var capturedIdentity *IdentityRecord
	callCount := 0
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			callCount++
			if callCount == 1 {
				return nil, nil
			}
			return &IdentityStatus{
				UserID:   42,
				DocType:  DocTypeMainlandID,
				RealName: capturedIdentity.RealName,
				Verified: false,
			}, nil
		},
		onCreateIdentity: func(_ context.Context, identity *IdentityRecord) error {
			copied := *identity
			capturedIdentity = &copied
			return nil
		},
	}
	store := &fakeIdentityPhotoStore{presignURL: "https://storage.example.test/identity/photo.png"}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		enc,
		WithIdentityPhotoStore(store),
	)
	require.NoError(t, err)

	front := "identities/42/2026/04/1777777777777777001-front.png"
	selfie := "identities/42/2026/04/1777777777777777002-selfie.png"
	_, err = svc.SubmitIdentity(context.Background(), 42, SubmitIdentityRequest{
		DocType:        DocTypeMainlandID,
		DocNumber:      " 11010519491231002x ",
		RealName:       " 张三 ",
		DocPhotoFront:  &front,
		DocPhotoSelfie: &selfie,
	})
	require.NoError(t, err)
	require.NotNil(t, capturedIdentity)
	assert.Equal(t, "11010519491231002X", enc.lastInput)
	assert.Equal(t, "张三", capturedIdentity.RealName)
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
	now := time.Now().Add(-time.Hour)
	rejectionReason := "照片不清晰"
	rejected := &IdentityStatus{
		UserID:          42,
		DocType:         DocTypePassport,
		RealName:        "旧实名",
		ReviewedAt:      &now,
		RejectionReason: &rejectionReason,
	}
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			callCount++
			if callCount == 1 {
				return rejected, nil
			}
			return &IdentityStatus{
				UserID:   42,
				DocType:  DocTypePassport,
				RealName: "新实名",
				Verified: false,
			}, nil
		},
		onGetIdentityStatusByUserIDTx: func(_ context.Context, _ pgx.Tx, _ int64) (*IdentityStatus, error) {
			return rejected, nil
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
	store := &fakeIdentityPhotoStore{presignURL: "https://storage.example.test/identity/photo.png"}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithIdentityPhotoStore(store),
	)
	require.NoError(t, err)

	front := "identities/42/2026/04/1777777777777777001-front.png"
	back := "identities/42/2026/04/1777777777777777002-back.png"
	selfie := "identities/42/2026/04/1777777777777777003-selfie.png"
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
	assert.Equal(t, []string{front, back, selfie}, store.presignedKeys)
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

	front := "identities/1/2026/04/1777777777777777001-front.png"
	selfie := "identities/1/2026/04/1777777777777777002-selfie.png"
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: false}, nil
		},
		onGetIdentityReviewItemByUserID: func(_ context.Context, _ int64) (*IdentityReviewItem, error) {
			return &IdentityReviewItem{
				UserID:          1,
				DocPhotoFront:   &front,
				DocPhotoSelfie:  &selfie,
				DocPhotoBack:    nil,
				RejectionReason: nil,
			}, nil
		},
		onUpdateIdentityReviewStatus: func(_ context.Context, _ int64, approved bool, verifyMethod *string, reviewedAt *time.Time, verifiedAt *time.Time, _ *string) error {
			updatedApproved = approved
			updatedMethod = verifyMethod
			updatedReviewedAt = reviewedAt
			updatedVerifiedAt = verifiedAt
			return nil
		},
	}

	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithIdentityPhotoStore(&fakeIdentityPhotoStore{
			presignURL: "https://storage.example.test/identity/photo.png",
		}),
	)
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 1, true, "")
	require.NoError(t, err)
	assert.True(t, updatedApproved)
	require.NotNil(t, updatedMethod)
	assert.Equal(t, VerifyMethodManual, *updatedMethod)
	assert.NotNil(t, updatedReviewedAt)
	assert.NotNil(t, updatedVerifiedAt)
}

func TestReviewIdentity_ApproveRejectsMissingEvidence(t *testing.T) {
	updateCalled := false
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: false}, nil
		},
		onGetIdentityReviewItemByUserID: func(_ context.Context, _ int64) (*IdentityReviewItem, error) {
			return &IdentityReviewItem{UserID: 1}, nil
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
			updateCalled = true
			return nil
		},
	}

	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
	)
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 1, true, "")
	require.ErrorIs(t, err, ErrPhotoRequired)
	assert.False(t, updateCalled)
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
