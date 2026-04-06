package user

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

// fakeEncryptor 实现 pii.Encryptor 接口的测试替身
type fakeEncryptor struct {
	called    bool
	lastInput string
}

func (f *fakeEncryptor) Encrypt(plaintext string) ([]byte, error) {
	f.called = true
	f.lastInput = plaintext
	return []byte("encrypted:" + plaintext), nil
}

// ---------------------------------------------------------------------------
// mockRepo — 实现 Repo 接口的测试替身，可通过函数字段定制行为
// ---------------------------------------------------------------------------

type mockRepo struct {
	onGetIdentityStatusByUserID                func(ctx context.Context, userID int64) (*IdentityStatus, error)
	onCreateIdentity                           func(ctx context.Context, identity *IdentityRecord) error
	onListIdentityReviewItems                  func(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error)
	onFindAcademicStudentsByPersonUIDFromTable func(ctx context.Context, sfzjlxdm, sfzjh, tableName string) ([]AcademicStudent, error)
	onGetAcademicStudentByXHFromTable          func(ctx context.Context, xh, tableName string) (*AcademicStudent, error)
	onUpdateIdentityReviewStatus               func(ctx context.Context, userID int64, approved bool, verifyMethod *string, reviewedAt *time.Time, verifiedAt *time.Time, rejectionReason *string) error
	onGetProfileByUserID                       func(ctx context.Context, userID int64) (*Profile, error)
	onCreateProfile                            func(ctx context.Context, profile *Profile) error
	onUpdateProfile                            func(ctx context.Context, profile *Profile) error
	onListProfilesByStatus                     func(ctx context.Context, status, schoolID string, page, pageSize int) ([]Profile, int, error)
	onSetUserPhone                             func(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error
	onGetSchoolConfig                          func(ctx context.Context, schoolID string) (*SchoolConfig, error)
	onListSchoolConfigs                        func(ctx context.Context) ([]SchoolConfig, error)
	onListAllSchoolConfigs                     func(ctx context.Context) ([]SchoolConfig, error)
	onUpdateSchoolConfig                       func(ctx context.Context, config *SchoolConfig) error
	onValidateAcademicDBTable                  func(ctx context.Context, tableName string) error
	onListSystemConfigs                        func(ctx context.Context) ([]SystemConfig, error)
	onUpdateSystemConfig                       func(ctx context.Context, key, value string) error
	onGetInternalUserID                        func(ctx context.Context, externalID string) (int64, error)
}

func (m *mockRepo) GetIdentityStatusByUserID(ctx context.Context, userID int64) (*IdentityStatus, error) {
	if m.onGetIdentityStatusByUserID != nil {
		return m.onGetIdentityStatusByUserID(ctx, userID)
	}
	return nil, nil
}

func (m *mockRepo) CreateIdentity(ctx context.Context, identity *IdentityRecord) error {
	if m.onCreateIdentity != nil {
		return m.onCreateIdentity(ctx, identity)
	}
	return nil
}

func (m *mockRepo) FindAcademicStudentsByPersonUIDFromTable(ctx context.Context, sfzjlxdm, sfzjh, tableName string) ([]AcademicStudent, error) {
	if m.onFindAcademicStudentsByPersonUIDFromTable != nil {
		return m.onFindAcademicStudentsByPersonUIDFromTable(ctx, sfzjlxdm, sfzjh, tableName)
	}
	return nil, nil
}

func (m *mockRepo) GetAcademicStudentByXHFromTable(ctx context.Context, xh, tableName string) (*AcademicStudent, error) {
	if m.onGetAcademicStudentByXHFromTable != nil {
		return m.onGetAcademicStudentByXHFromTable(ctx, xh, tableName)
	}
	return nil, nil
}

func (m *mockRepo) UpdateIdentityReviewStatus(ctx context.Context, userID int64, approved bool, verifyMethod *string, reviewedAt *time.Time, verifiedAt *time.Time, rejectionReason *string) error {
	if m.onUpdateIdentityReviewStatus != nil {
		return m.onUpdateIdentityReviewStatus(ctx, userID, approved, verifyMethod, reviewedAt, verifiedAt, rejectionReason)
	}
	return nil
}

func (m *mockRepo) GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error) {
	if m.onGetProfileByUserID != nil {
		return m.onGetProfileByUserID(ctx, userID)
	}
	return nil, nil
}

func (m *mockRepo) UpdateProfile(ctx context.Context, profile *Profile) error {
	if m.onUpdateProfile != nil {
		return m.onUpdateProfile(ctx, profile)
	}
	return nil
}

// 以下方法在当前测试中不需要定制，提供默认空实现

func (m *mockRepo) ListIdentityReviewItems(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error) {
	if m.onListIdentityReviewItems != nil {
		return m.onListIdentityReviewItems(ctx, status, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockRepo) CreateProfile(ctx context.Context, profile *Profile) error {
	if m.onCreateProfile != nil {
		return m.onCreateProfile(ctx, profile)
	}
	return nil
}

func (m *mockRepo) ListProfilesByStatus(ctx context.Context, status string, schoolID string, page, pageSize int) ([]Profile, int, error) {
	if m.onListProfilesByStatus != nil {
		return m.onListProfilesByStatus(ctx, status, schoolID, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockRepo) SetUserPhone(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error {
	if m.onSetUserPhone != nil {
		return m.onSetUserPhone(ctx, userID, phoneEnc, phoneHash)
	}
	return nil
}

func (m *mockRepo) GetSchoolConfig(ctx context.Context, schoolID string) (*SchoolConfig, error) {
	if m.onGetSchoolConfig != nil {
		return m.onGetSchoolConfig(ctx, schoolID)
	}
	return nil, nil
}

func (m *mockRepo) ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	if m.onListSchoolConfigs != nil {
		return m.onListSchoolConfigs(ctx)
	}
	return nil, nil
}

func (m *mockRepo) ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	if m.onListAllSchoolConfigs != nil {
		return m.onListAllSchoolConfigs(ctx)
	}
	return nil, nil
}

func (m *mockRepo) UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error {
	if m.onUpdateSchoolConfig != nil {
		return m.onUpdateSchoolConfig(ctx, config)
	}
	return nil
}

func (m *mockRepo) ValidateAcademicDBTable(ctx context.Context, tableName string) error {
	if m.onValidateAcademicDBTable != nil {
		return m.onValidateAcademicDBTable(ctx, tableName)
	}
	return nil
}

func (m *mockRepo) ListSystemConfigs(ctx context.Context) ([]SystemConfig, error) {
	if m.onListSystemConfigs != nil {
		return m.onListSystemConfigs(ctx)
	}
	return nil, nil
}

func (m *mockRepo) UpdateSystemConfig(ctx context.Context, key, value string) error {
	if m.onUpdateSystemConfig != nil {
		return m.onUpdateSystemConfig(ctx, key, value)
	}
	return nil
}

func (m *mockRepo) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	if m.onGetInternalUserID != nil {
		return m.onGetInternalUserID(ctx, externalID)
	}
	return 0, nil
}

func (m *mockRepo) GetExternalID(_ context.Context, _ int64) (string, error) {
	return "test-external-id", nil
}

// ---------------------------------------------------------------------------
// NewService 构造校验
// ---------------------------------------------------------------------------

func TestNewService_NilRepo(t *testing.T) {
	_, err := NewService(nil, nil, []byte("secret-key-32-chars-long-enough!"), &fakeEncryptor{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo must not be nil")
}

func TestNewService_EmptyHMACKey(t *testing.T) {
	_, err := NewService(&mockRepo{}, nil, nil, &fakeEncryptor{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hmacKey must not be empty")
}

func TestNewService_NilDocCipher(t *testing.T) {
	_, err := NewService(&mockRepo{}, nil, []byte("secret-key-32-chars-long-enough!"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docCipher must not be nil")
}

func TestNewService_ValidConstruction(t *testing.T) {
	svc, err := NewService(&mockRepo{}, nil, []byte("secret-key-32-chars-long-enough!"), &fakeEncryptor{})
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), enc)
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.BindPhone(context.Background(), 7, "13800138000")
	require.ErrorIs(t, err, ErrPhoneAlreadyBound)
}

// ---------------------------------------------------------------------------
// computePersonUID
// ---------------------------------------------------------------------------

func TestComputePersonUID_Consistency(t *testing.T) {
	svc, err := NewService(&mockRepo{}, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), enc)
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
	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 1, SubmitIdentityRequest{
		DocType: DocTypeMainlandID, DocNumber: "123", RealName: "test",
	})
	assert.ErrorIs(t, err, ErrIdentityAlreadyExists)
}

func TestSubmitIdentity_AlreadyVerified(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{Verified: true}, nil
		},
	}
	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.SubmitIdentity(context.Background(), 1, SubmitIdentityRequest{
		DocType: DocTypeMainlandID, DocNumber: "123", RealName: "test",
	})
	assert.ErrorIs(t, err, ErrIdentityAlreadyVerified)
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
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
	svc, err := NewService(&mockRepo{}, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewIdentity(context.Background(), 999, true, "")
	assert.ErrorIs(t, err, ErrIdentityNotFound)
}

// ---------------------------------------------------------------------------
// VerifyStudent
// ---------------------------------------------------------------------------

func TestVerifyStudent_ManualAllowsEmptyCredentialsAndPersistsManualData(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID string) (*SchoolConfig, error) {
			require.Equal(t, "10006", schoolID)
			return &SchoolConfig{
				SchoolID:           "10006",
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	profile, err := svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: "10006",
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
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           "manual",
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: "manual",
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
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           "ldap",
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: "ldap",
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
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           "ldap",
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID:  "ldap",
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
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

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
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

func TestUpdateSchoolConfig_MergesPartialUpdateAndPreservesUnspecifiedFields(t *testing.T) {
	var captured *SchoolConfig

	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID string) (*SchoolConfig, error) {
			require.Equal(t, "10006", schoolID)
			academicTable := "academic.buaa_students"
			consentText := "原始授权文本"
			return &SchoolConfig{
				SchoolID:           "10006",
				SchoolName:         "旧学校名",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"url":"ldap://ldap.old:389","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":false,"insecureSkipVerify":false}`),
				AcademicDBTable:    &academicTable,
				ConsentText:        &consentText,
				ManualFormFields:   json.RawMessage(`[{"key":"idCard","label":"身份证号","type":"text","required":true}]`),
				Enabled:            true,
			}, nil
		},
		onUpdateSchoolConfig: func(_ context.Context, config *SchoolConfig) error {
			captured = config
			return nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	schoolName := "新学校名"
	enabled := false
	ldapURL := "ldaps://ldap.new:636"
	systemBindPassword := "new-secret"
	useTLS := true

	err = svc.UpdateSchoolConfig(context.Background(), "10006", UpdateSchoolConfigInput{
		SchoolName: &schoolName,
		LDAPConfig: &SchoolLDAPConfigInput{
			URL:                &ldapURL,
			SystemBindPassword: &systemBindPassword,
			UseTLS:             &useTLS,
		},
		Enabled: &enabled,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)

	assert.Equal(t, "10006", captured.SchoolID)
	assert.Equal(t, "新学校名", captured.SchoolName)
	assert.Equal(t, VerifyMethodLDAP, captured.VerificationMethod, "未提供的 verificationMethod 应保留原值")
	assert.JSONEq(t, `{"baseDN":"ou=users,dc=example,dc=com","insecureSkipVerify":false,"systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"new-secret","url":"ldaps://ldap.new:636","useTLS":true}`, string(captured.LDAPConfig))
	require.NotNil(t, captured.AcademicDBTable)
	assert.Equal(t, "academic.buaa_students", *captured.AcademicDBTable, "未提供的 academicDbTable 应保留原值")
	require.NotNil(t, captured.ConsentText)
	assert.Equal(t, "原始授权文本", *captured.ConsentText, "未提供的 consentText 应保留原值")
	assert.JSONEq(t, `[{"key":"idCard","label":"身份证号","type":"text","required":true}]`, string(captured.ManualFormFields), "未提供的 manualFormFields 应保留原值")
	assert.False(t, captured.Enabled)
}

func TestUpdateSchoolConfig_PreservesExistingLDAPPasswordWhenOmitted(t *testing.T) {
	var captured *SchoolConfig

	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID string) (*SchoolConfig, error) {
			require.Equal(t, "10006", schoolID)
			table := "academic.buaa_students"
			return &SchoolConfig{
				SchoolID:           "10006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"url":"ldap://ldap.old:389","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":false,"insecureSkipVerify":false}`),
				AcademicDBTable:    &table,
				Enabled:            true,
			}, nil
		},
		onUpdateSchoolConfig: func(_ context.Context, config *SchoolConfig) error {
			copied := *config
			captured = &copied
			return nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	ldapURL := "ldaps://ldap.new:636"
	useTLS := true
	err = svc.UpdateSchoolConfig(context.Background(), "10006", UpdateSchoolConfigInput{
		LDAPConfig: &SchoolLDAPConfigInput{
			URL:    &ldapURL,
			UseTLS: &useTLS,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.JSONEq(t, `{"url":"ldaps://ldap.new:636","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":true,"insecureSkipVerify":false}`, string(captured.LDAPConfig))
}

func TestUpdateSchoolConfig_SchoolNotFoundReturnsError(t *testing.T) {
	svc, err := NewService(&mockRepo{}, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSchoolConfig(context.Background(), "missing", UpdateSchoolConfigInput{})
	assert.ErrorIs(t, err, ErrSchoolNotFound)
}

func TestUpdateSchoolConfig_InvalidManualFieldConfigReturnsError(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           "10006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	fields := []ManualFieldDescriptor{{Key: "", Label: "空 key", Type: "text", Required: true}}
	err = svc.UpdateSchoolConfig(context.Background(), "10006", UpdateSchoolConfigInput{
		ManualFormFields: &fields,
	})
	assert.ErrorIs(t, err, ErrInvalidManualFieldConfig)
}

func TestUpdateSchoolConfig_EnabledLDAPRequiresAcademicTable(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           "10006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				Enabled:            false,
			}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	method := VerifyMethodLDAP
	enabled := true
	err = svc.UpdateSchoolConfig(context.Background(), "10006", UpdateSchoolConfigInput{
		VerificationMethod: &method,
		Enabled:            &enabled,
	})

	assert.ErrorIs(t, err, ErrAcademicTableNotConfigured)
}

func TestUpdateSchoolConfig_EnabledLDAPRequiresValidLDAPConfig(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			table := "academic.buaa_students"
			return &SchoolConfig{
				SchoolID:           "10006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodLDAP,
				AcademicDBTable:    &table,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSchoolConfig(context.Background(), "10006", UpdateSchoolConfigInput{})

	assert.ErrorIs(t, err, ErrSchoolLDAPConfigMissing)
}

func TestUpdateSchoolConfig_ValidatesAcademicTableExists(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ string) (*SchoolConfig, error) {
			table := "academic.buaa_students"
			return &SchoolConfig{
				SchoolID:           "10006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				AcademicDBTable:    &table,
				Enabled:            false,
			}, nil
		},
		onValidateAcademicDBTable: func(_ context.Context, tableName string) error {
			assert.Equal(t, "academic.buaa_students", tableName)
			return assert.AnError
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSchoolConfig(context.Background(), "10006", UpdateSchoolConfigInput{})

	assert.ErrorIs(t, err, ErrInvalidAcademicDBTable)
}

func TestUpdateSystemConfig_RejectsInvalidReviewAccessSchoolIDs(t *testing.T) {
	svc, err := NewService(&mockRepo{}, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), systemconfig.ReviewAccessSchoolIDsKey, `{"schoolID":"10006"}`)
	assert.ErrorIs(t, err, ErrInvalidSystemConfigValue)
}

func TestUpdateSystemConfig_RejectsInvalidReviewPreviewPercent(t *testing.T) {
	svc, err := NewService(&mockRepo{}, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), systemconfig.ReviewPreviewContentPercentKey, "120")
	assert.ErrorIs(t, err, ErrInvalidSystemConfigValue)
}

func TestUpdateSystemConfig_RejectsUnknownReviewAccessSchoolIDs(t *testing.T) {
	repo := &mockRepo{
		onListAllSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{
				{SchoolID: "10006"},
				{SchoolID: "10007"},
			}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), systemconfig.ReviewAccessSchoolIDsKey, `["10006","99999"]`)
	assert.ErrorIs(t, err, ErrInvalidSystemConfigValue)
}

func TestUpdateSystemConfig_ReturnsNotFoundWhenKeyMissing(t *testing.T) {
	repo := &mockRepo{
		onUpdateSystemConfig: func(_ context.Context, key, value string) error {
			assert.Equal(t, "feature.missing", key)
			assert.Equal(t, "enabled", value)
			return ErrSystemConfigNotFound
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), "feature.missing", "enabled")
	assert.ErrorIs(t, err, ErrSystemConfigNotFound)
}
