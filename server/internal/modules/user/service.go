package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
)

// 业务错误定义
var (
	ErrIdentityAlreadyExists                    = errors.New("identity already exists")
	ErrIdentityAlreadyVerified                  = errors.New("identity already verified")
	ErrProfileAlreadyVerified                   = errors.New("profile already verified")
	ErrProfilePendingReview                     = errors.New("profile is pending review, please wait for admin approval")
	ErrSchoolNotFound                           = errors.New("school not found")
	ErrSchoolDisabled                           = errors.New("school verification disabled")
	ErrConsentRequired                          = errors.New("consent is required")
	ErrPhotoRequired                            = errors.New("photo upload required for non-mainland documents")
	ErrLDAPFailed                               = errors.New("LDAP verification failed")
	ErrStudentIDRequired                        = errors.New("student ID is required for LDAP verification")
	ErrPasswordRequired                         = errors.New("password is required for LDAP verification")
	ErrPhoneAlreadyBound                        = errors.New("phone already bound to another user")
	ErrStudentNotFound                          = errors.New("student record not found in academic database")
	ErrProfileNotFound                          = errors.New("student profile not found")
	ErrIdentityNotFound                         = errors.New("identity not found")
	ErrManualFieldRequired                      = errors.New("required manual form field is missing")
	ErrManualFieldInvalid                       = errors.New("manual form field value is invalid")
	ErrInvalidManualFieldConfig                 = errors.New("manual form field config is invalid")
	ErrInvalidAcademicDBTable                   = errors.New("academic table config is invalid")
	ErrAcademicTableNotConfigured               = errors.New("academic table is not configured for the school")
	ErrSchoolLDAPConfigMissing                  = errors.New("LDAP configuration is not provided for the school")
	ErrLDAPConfigInvalid                        = errors.New("LDAP configuration is invalid")
	ErrSystemConfigNotFound                     = errors.New("system config not found")
	ErrInvalidSystemConfigValue                 = errors.New("system config value is invalid")
	ErrIdentityPhotoStoreDisabled               = errors.New("identity photo storage is not configured")
	ErrIdentityPhotoTooLarge                    = errors.New("identity photo too large")
	ErrIdentityPhotoInvalidType                 = errors.New("identity photo content type is invalid")
	ErrIdentityPhotoInvalidData                 = errors.New("identity photo data is invalid")
	ErrIdentityPhotoInvalidRef                  = errors.New("identity photo reference is invalid")
	ErrIdentityPhotoStorageUnavailable          = errors.New("identity photo storage unavailable")
	ErrIdentityPhotoStorageTemporaryUnavailable = errors.New("identity photo storage temporarily unavailable")
	ErrIdentityDocNumberInvalid                 = errors.New("identity document number is invalid")
	ErrIdentityRealNameInvalid                  = errors.New("identity real name is invalid")
	ErrQQBindingAlreadyExists                   = errors.New("qq binding already exists")
	ErrQQBindingCodeInvalid                     = errors.New("qq binding code is invalid")
	ErrQQBindingCodeExpired                     = errors.New("qq binding code has expired")
	ErrQQBindingQQAlreadyBound                  = errors.New("qq account already bound to another user")
	ErrQQBindingUserConflict                    = errors.New("user already bound to another qq account")
	ErrQQIDRequired                             = errors.New("qq id is required")
	ErrProfileIdentitySyncMissing               = errors.New("profile identity sync gateway is not configured")
	ErrStudentEmailDomainNotAllowed             = errors.New("student email domain not allowed")
	ErrStudentEmailSenderUnavailable            = errors.New("student email sender unavailable")
	ErrStudentEmailRedisUnavailable             = errors.New("student email redis unavailable")
	ErrStudentEmailOTPCooldown                  = errors.New("student email otp cooldown")
	ErrStudentEmailOTPExpired                   = errors.New("student email otp expired")
	ErrStudentEmailOTPInvalid                   = errors.New("student email otp invalid")
	ErrStudentEmailOTPMaxAttempts               = errors.New("student email otp max attempts exceeded")
	ErrStudentNameRequired                      = errors.New("student name is required")
	ErrStudentNameMismatch                      = errors.New("student name does not match academic database")
)

// DocType 证件类型常量
const (
	DocTypeMainlandID = "MAINLAND_ID"
	DocTypeHKMacau    = "HK_MACAU"
	DocTypeTW         = "TW"
	DocTypePassport   = "PASSPORT"
)

// VerificationStatus 认证状态常量
const (
	StatusUnverified = "unverified"
	StatusPending    = "pending"
	StatusVerified   = "verified"
	StatusRejected   = "rejected"
)

// VerifyMethod 认证方式常量
const (
	VerifyMethodAcademicDB     = "academic_db_match"
	VerifyMethodTencent        = "tencent_cloud"
	VerifyMethodManual         = "manual"
	VerifyMethodLDAP           = "ldap"
	VerifyMethodSchoolEmailOTP = "school_email_otp"
)

const (
	IdentityPhotoSlotFront  = "front"
	IdentityPhotoSlotBack   = "back"
	IdentityPhotoSlotSelfie = "selfie"
)

const (
	qqBindingCodeLength = 8
)

// Repo 定义 Service 所需的数据访问能力。
type Repo interface {
	GetIdentityStatusByUserID(ctx context.Context, userID int64) (*IdentityStatus, error)
	CreateIdentity(ctx context.Context, identity *IdentityRecord) error
	UpdateIdentitySubmission(ctx context.Context, identity *IdentityRecord) error
	ListIdentityReviewItems(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error)
	UpdateIdentityReviewStatus(ctx context.Context, userID int64, approved bool, verifyMethod *string, reviewedAt *time.Time, verifiedAt *time.Time, rejectionReason *string) error

	GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error)
	CreateProfile(ctx context.Context, profile *Profile) error
	UpdateProfile(ctx context.Context, profile *Profile) error
	ListProfilesByStatus(ctx context.Context, status string, schoolID *int64, page, pageSize int) ([]Profile, int, error)
	GetCasdoorSubject(ctx context.Context, userID int64) (string, error)
	EnsureUserPhoneAvailable(ctx context.Context, userID int64, phoneHash string) error
	SetUserPhone(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error
	GetQQBindingByUserID(ctx context.Context, userID int64) (*QQBinding, error)
	GetQQBindingByQQID(ctx context.Context, qqID string) (*QQBinding, error)
	UpsertQQBindingCode(ctx context.Context, code *QQBindingCode) error

	GetSchoolConfig(ctx context.Context, schoolID int64) (*SchoolConfig, error)
	ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error)
	ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error)
	UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error

	ListSystemConfigs(ctx context.Context) ([]SystemConfig, error)
	UpdateSystemConfig(ctx context.Context, key, value string) error

	GetInternalUserID(ctx context.Context, casdoorSubject string) (int64, error)
	GetAcademicStudentByXHFromTable(ctx context.Context, xh string, tableName string) (*AcademicStudent, error)
	FindAcademicStudentsByPersonUIDFromTable(ctx context.Context, sfzjlxdm, sfzjh string, tableName string) ([]AcademicStudent, error)
	ValidateAcademicDBTable(ctx context.Context, tableName string) error
	WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
	GetProfileByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*Profile, error)
	CreateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error
	UpdateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error
	EnsureVerificationCredentialTx(ctx context.Context, tx pgx.Tx, credential VerificationCredentialProjection) error
	GetQQBindingCodeByHashTx(ctx context.Context, tx pgx.Tx, codeHash string) (*QQBindingCode, error)
	GetQQBindingByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*QQBinding, error)
	GetQQBindingByQQIDTx(ctx context.Context, tx pgx.Tx, qqID string) (*QQBinding, error)
	CreateQQBindingTx(ctx context.Context, tx pgx.Tx, binding *QQBinding) error
	MarkQQBindingCodeConsumedTx(ctx context.Context, tx pgx.Tx, userID int64, consumedAt time.Time) error
	UpsertExternalSyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error
	ClaimExternalSyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error)
	MarkExternalSyncJobDone(ctx context.Context, jobID int64) error
	MarkExternalSyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error
	MarkExternalSyncJobFailure(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string, terminal bool) error
	ListStudentRoleProjectionStates(ctx context.Context, limit int) ([]StudentRoleProjectionState, error)
}

type LDAPConfig struct {
	URL                string
	BaseDN             string
	SystemBindDN       string
	SystemBindPassword string
	UseTLS             bool
}

type LDAPLoginResult struct {
	Authenticated bool
}

type LDAPUserInfo struct {
	UID              string
	CN               string
	SN               string
	EmployeeNumber   string
	DepartmentNumber string
	Mail             string
	Mobile           string
	EmployeeType     string
}

type LDAPAuthClient interface {
	Login(ctx context.Context, uid, password string) (*LDAPLoginResult, error)
	QueryUserByUID(ctx context.Context, uid string) (*LDAPUserInfo, error)
}

type LDAPClientFactory func(cfg LDAPConfig) (LDAPAuthClient, error)

type identityPhotoStore interface {
	Upload(ctx context.Context, key string, content []byte, contentType string) error
	PresignGetURL(ctx context.Context, key string) (string, error)
}

type profileFGAClient interface {
	WriteTuples(ctx context.Context, tuples []fga.Tuple) error
	DeleteTuples(ctx context.Context, tuples []fga.Tuple) error
	ReadTuples(ctx context.Context, object, relation string) ([]fga.Tuple, error)
}

type profileIdentitySyncGateway interface {
	UpdatePhone(ctx context.Context, subject, phone string) error
}

type admissionVerificationProjectionGateway interface {
	ProjectStudentVerification(ctx context.Context, userID int64, schoolID int64, approved bool) error
}

type StudentEmailSender interface {
	SendStudentVerificationOTP(ctx context.Context, email string, code string) error
}

type ExternalStudentRecord struct {
	SchoolCode  string
	StudentID   string
	StudentName string
}

type studentDirectoryLookup interface {
	LookupStudent(ctx context.Context, schoolCode string, studentID string) (*ExternalStudentRecord, bool, error)
}

// Service 用户服务层
type Service struct {
	repo                Repo
	ldapClientFactory   LDAPClientFactory
	hmacKey             []byte
	docCipher           pii.EncryptDecryptor
	redisClient         *redis.Client
	studentEmailSender  StudentEmailSender
	generateOTP         func() (string, error)
	onRoleSync          RoleSyncFunc
	profileFGA          profileFGAClient
	photoStore          identityPhotoStore
	profileIdentitySync profileIdentitySyncGateway
	admissionProjection admissionVerificationProjectionGateway
	studentDirectory    studentDirectoryLookup
}

// RoleSyncFunc 角色同步回调。
// 当用户认证状态变化时调用：approved=true 添加角色，approved=false 移除角色。
// userID 是内部 users.id，role 是 Casdoor 扁平角色名称。
type RoleSyncFunc func(ctx context.Context, userID int64, role string, approved bool) error

type ServiceOption func(*Service)

func WithRoleSyncFunc(fn RoleSyncFunc) ServiceOption {
	return func(s *Service) {
		s.onRoleSync = fn
	}
}

func WithProfileFGAClient(client profileFGAClient) ServiceOption {
	return func(s *Service) {
		s.profileFGA = client
	}
}

func WithIdentityPhotoStore(store identityPhotoStore) ServiceOption {
	return func(s *Service) {
		s.photoStore = store
	}
}

func WithProfileIdentitySyncGateway(gateway profileIdentitySyncGateway) ServiceOption {
	return func(s *Service) {
		s.profileIdentitySync = gateway
	}
}

func WithAdmissionVerificationProjectionGateway(gateway admissionVerificationProjectionGateway) ServiceOption {
	return func(s *Service) {
		s.admissionProjection = gateway
	}
}

func WithStudentEmailOTP(client *redis.Client, sender StudentEmailSender) ServiceOption {
	return func(s *Service) {
		s.redisClient = client
		s.studentEmailSender = sender
	}
}

func WithExternalStudentDirectory(directory studentDirectoryLookup) ServiceOption {
	return func(s *Service) {
		s.studentDirectory = directory
	}
}

func (s *Service) SetProfileIdentitySyncGateway(gateway profileIdentitySyncGateway) {
	s.profileIdentitySync = gateway
}

func (s *Service) SetAdmissionVerificationProjectionGateway(gateway admissionVerificationProjectionGateway) {
	s.admissionProjection = gateway
}

func WithLDAPClientFactory(factory LDAPClientFactory) ServiceOption {
	return func(s *Service) {
		if factory != nil {
			s.ldapClientFactory = factory
		}
	}
}

// NewService 创建用户服务（构造期校验关键依赖）
func NewService(repo Repo, hmacKey []byte, docCipher pii.EncryptDecryptor, opts ...ServiceOption) (*Service, error) {
	if repo == nil {
		return nil, errors.New("user.NewService: repo must not be nil")
	}
	if len(hmacKey) == 0 {
		return nil, errors.New("user.NewService: hmacKey must not be empty")
	}
	if docCipher == nil {
		return nil, errors.New("user.NewService: docCipher must not be nil")
	}
	svc := &Service{
		repo:        repo,
		hmacKey:     hmacKey,
		docCipher:   docCipher,
		generateOTP: generateStudentEmailOTPCode,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc, nil
}

// SubmitIdentityRequest 提交实名认证请求
type SubmitIdentityRequest struct {
	DocType        string  `json:"docType"`
	DocNumber      string  `json:"docNumber"`
	RealName       string  `json:"realName"`
	DocPhotoFront  *string `json:"docPhotoFront"`
	DocPhotoBack   *string `json:"docPhotoBack"`
	DocPhotoSelfie *string `json:"docPhotoSelfie"`
}

// UploadIdentityPhotoRequest 上传实名认证照片请求。
type UploadIdentityPhotoRequest struct {
	Slot        string `json:"slot"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	DataBase64  string `json:"dataBase64"`
}

// VerifyStudentRequest 学生认证请求
type VerifyStudentRequest struct {
	SchoolID       int64          `json:"schoolID"`
	StudentID      string         `json:"studentID"`
	Password       string         `json:"password"`
	ManualFormData map[string]any `json:"manualFormData"`
	Consent        bool           `json:"consent"`
}

// SchoolLDAPConfigInput 学校 LDAP 配置更新输入。
// 指针用于保留省略与显式传值的区别，避免部分更新时误清空未提交字段。
type SchoolLDAPConfigInput struct {
	URL                *string `json:"url,omitempty"`
	BaseDN             *string `json:"baseDN,omitempty"`
	SystemBindDN       *string `json:"systemBindDN,omitempty"`
	SystemBindPassword *string `json:"systemBindPassword,omitempty"`
	UseTLS             *bool   `json:"useTLS,omitempty"`
}

// SchoolLDAPConfigView 管理端学校配置返回体。
// 为避免泄漏密钥，仅返回是否已配置系统绑定密码。
type SchoolLDAPConfigView struct {
	URL                   *string `json:"url,omitempty"`
	BaseDN                *string `json:"baseDN,omitempty"`
	SystemBindDN          *string `json:"systemBindDN,omitempty"`
	UseTLS                bool    `json:"useTLS"`
	HasSystemBindPassword bool    `json:"hasSystemBindPassword"`
}

// UpdateSchoolConfigInput 学校配置更新请求（管理端）
// 使用可选字段做合并更新，避免未提供字段被误清空。
type UpdateSchoolConfigInput struct {
	SchoolName         *string
	VerificationMethod *string
	ApprovalPolicy     *string
	LDAPConfig         *SchoolLDAPConfigInput
	AcademicDBTable    *string
	ConsentText        *string
	ManualFormFields   *[]ManualFieldDescriptor
	Enabled            *bool
}

// computePersonUID 计算 person_uid: HMAC-SHA256(doc_type + ':' + doc_number)
func (s *Service) computePersonUID(docType, docNumber string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(docType + ":" + docNumber))
	return hex.EncodeToString(mac.Sum(nil))
}
