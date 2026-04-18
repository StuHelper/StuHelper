package user

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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

func (f *fakeEncryptor) Decrypt(ciphertext []byte) (string, error) {
	return string(ciphertext), nil
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
	onListProfilesByStatus                     func(ctx context.Context, status string, schoolID *int64, page, pageSize int) ([]Profile, int, error)
	onSetUserPhone                             func(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error
	onGetSchoolConfig                          func(ctx context.Context, schoolID int64) (*SchoolConfig, error)
	onListSchoolConfigs                        func(ctx context.Context) ([]SchoolConfig, error)
	onListAllSchoolConfigs                     func(ctx context.Context) ([]SchoolConfig, error)
	onUpdateSchoolConfig                       func(ctx context.Context, config *SchoolConfig) error
	onValidateAcademicDBTable                  func(ctx context.Context, tableName string) error
	onListSystemConfigs                        func(ctx context.Context) ([]SystemConfig, error)
	onUpdateSystemConfig                       func(ctx context.Context, key, value string) error
	onGetInternalUserID                        func(ctx context.Context, externalID string) (int64, error)
	onWithTx                                   func(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
	onGetProfileByUserIDTx                     func(ctx context.Context, tx pgx.Tx, userID int64) (*Profile, error)
	onCreateProfileTx                          func(ctx context.Context, tx pgx.Tx, profile *Profile) error
	onUpdateProfileTx                          func(ctx context.Context, tx pgx.Tx, profile *Profile) error
	onSetUserPhoneTx                           func(ctx context.Context, tx pgx.Tx, userID int64, phoneEnc []byte, phoneHash string) error
	onUpsertExternalSyncJobTx                  func(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error
	onClaimExternalSyncJobs                    func(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error)
	onMarkExternalSyncJobDone                  func(ctx context.Context, jobID int64) error
	onMarkExternalSyncJobRetry                 func(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error
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

func (m *mockRepo) ListProfilesByStatus(ctx context.Context, status string, schoolID *int64, page, pageSize int) ([]Profile, int, error) {
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

func (m *mockRepo) GetSchoolConfig(ctx context.Context, schoolID int64) (*SchoolConfig, error) {
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

func (m *mockRepo) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if m.onWithTx != nil {
		return m.onWithTx(ctx, fn)
	}
	return fn(ctx, nil)
}

func (m *mockRepo) GetProfileByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*Profile, error) {
	if m.onGetProfileByUserIDTx != nil {
		return m.onGetProfileByUserIDTx(ctx, tx, userID)
	}
	return m.GetProfileByUserID(ctx, userID)
}

func (m *mockRepo) CreateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	if m.onCreateProfileTx != nil {
		return m.onCreateProfileTx(ctx, tx, profile)
	}
	return m.CreateProfile(ctx, profile)
}

func (m *mockRepo) UpdateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	if m.onUpdateProfileTx != nil {
		return m.onUpdateProfileTx(ctx, tx, profile)
	}
	return m.UpdateProfile(ctx, profile)
}

func (m *mockRepo) SetUserPhoneTx(ctx context.Context, tx pgx.Tx, userID int64, phoneEnc []byte, phoneHash string) error {
	if m.onSetUserPhoneTx != nil {
		return m.onSetUserPhoneTx(ctx, tx, userID, phoneEnc, phoneHash)
	}
	return m.SetUserPhone(ctx, userID, phoneEnc, phoneHash)
}

func (m *mockRepo) UpsertExternalSyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	if m.onUpsertExternalSyncJobTx != nil {
		return m.onUpsertExternalSyncJobTx(ctx, tx, jobType, dedupeKey, payload)
	}
	return nil
}

func (m *mockRepo) ClaimExternalSyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error) {
	if m.onClaimExternalSyncJobs != nil {
		return m.onClaimExternalSyncJobs(ctx, limit, staleAfter)
	}
	return nil, nil
}

func (m *mockRepo) MarkExternalSyncJobDone(ctx context.Context, jobID int64) error {
	if m.onMarkExternalSyncJobDone != nil {
		return m.onMarkExternalSyncJobDone(ctx, jobID)
	}
	return nil
}

func (m *mockRepo) MarkExternalSyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	if m.onMarkExternalSyncJobRetry != nil {
		return m.onMarkExternalSyncJobRetry(ctx, jobID, nextAttemptAt, lastError)
	}
	return nil
}

// ---------------------------------------------------------------------------
// NewService 构造校验
// ---------------------------------------------------------------------------
