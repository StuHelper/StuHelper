package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
)

// 业务错误定义
var (
	ErrIdentityAlreadyExists      = errors.New("identity already exists")
	ErrIdentityAlreadyVerified    = errors.New("identity already verified")
	ErrProfileAlreadyVerified     = errors.New("profile already verified")
	ErrProfilePendingReview       = errors.New("profile is pending review, please wait for admin approval")
	ErrSchoolNotFound             = errors.New("school not found")
	ErrSchoolDisabled             = errors.New("school verification disabled")
	ErrConsentRequired            = errors.New("consent is required")
	ErrPhotoRequired              = errors.New("photo upload required for non-mainland documents")
	ErrLDAPFailed                 = errors.New("LDAP verification failed")
	ErrIdentityRequired           = errors.New("identity verification required before student verification")
	ErrStudentIDRequired          = errors.New("student ID is required for LDAP verification")
	ErrPasswordRequired           = errors.New("password is required for LDAP verification")
	ErrPhoneAlreadyBound          = errors.New("phone already bound to another user")
	ErrStudentNotFound            = errors.New("student record not found in academic database")
	ErrProfileNotFound            = errors.New("student profile not found")
	ErrIdentityNotFound           = errors.New("identity not found")
	ErrManualFieldRequired        = errors.New("required manual form field is missing")
	ErrManualFieldInvalid         = errors.New("manual form field value is invalid")
	ErrInvalidManualFieldConfig   = errors.New("manual form field config is invalid")
	ErrInvalidAcademicDBTable     = errors.New("academic table config is invalid")
	ErrAcademicTableNotConfigured = errors.New("academic table is not configured for the school")
	ErrSchoolLDAPConfigMissing    = errors.New("LDAP configuration is not provided for the school")
	ErrLDAPConfigInvalid          = errors.New("LDAP configuration is invalid")
	ErrSystemConfigNotFound       = errors.New("system config not found")
	ErrInvalidSystemConfigValue   = errors.New("system config value is invalid")
	ErrIdentityPhotoStoreDisabled = errors.New("identity photo storage is not configured")
	ErrIdentityPhotoTooLarge      = errors.New("identity photo too large")
	ErrIdentityPhotoInvalidType   = errors.New("identity photo content type is invalid")
	ErrIdentityPhotoInvalidData   = errors.New("identity photo data is invalid")
	ErrIdentityPhotoInvalidRef    = errors.New("identity photo reference is invalid")
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
	VerifyMethodAcademicDB = "academic_db_match"
	VerifyMethodTencent    = "tencent_cloud"
	VerifyMethodManual     = "manual"
	VerifyMethodLDAP       = "ldap"
)

const (
	IdentityPhotoSlotFront  = "front"
	IdentityPhotoSlotBack   = "back"
	IdentityPhotoSlotSelfie = "selfie"
)

// Repo defines the data access methods required by Service.
type Repo interface {
	GetIdentityStatusByUserID(ctx context.Context, userID int64) (*IdentityStatus, error)
	CreateIdentity(ctx context.Context, identity *IdentityRecord) error
	ListIdentityReviewItems(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error)
	UpdateIdentityReviewStatus(ctx context.Context, userID int64, approved bool, verifyMethod *string, reviewedAt *time.Time, verifiedAt *time.Time, rejectionReason *string) error

	GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error)
	CreateProfile(ctx context.Context, profile *Profile) error
	UpdateProfile(ctx context.Context, profile *Profile) error
	ListProfilesByStatus(ctx context.Context, status string, schoolID string, page, pageSize int) ([]Profile, int, error)
	SetUserPhone(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error

	GetSchoolConfig(ctx context.Context, schoolID string) (*SchoolConfig, error)
	ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error)
	ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error)
	UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error

	ListSystemConfigs(ctx context.Context) ([]SystemConfig, error)
	UpdateSystemConfig(ctx context.Context, key, value string) error

	GetInternalUserID(ctx context.Context, externalID string) (int64, error)
	GetExternalID(ctx context.Context, userID int64) (string, error)
}

type ldapAuthClient interface {
	Login(ctx context.Context, uid, password string) (*ldap.LoginResult, error)
	QueryUserByUID(ctx context.Context, uid string) (*ldap.UserInfo, error)
}

type ldapClientFactory func(cfg ldap.Config) (ldapAuthClient, error)

type identityPhotoStore interface {
	Upload(ctx context.Context, key string, content []byte, contentType string) error
	PresignGetURL(ctx context.Context, key string) (string, error)
}

// RoleSyncFunc 角色同步回调。
// 当用户认证状态变化时调用：approved=true 添加角色，approved=false 移除角色。
// userID 是内部用户 ID（用于查询 external_id），role 是 Zitadel Project Role 名称。
type RoleSyncFunc func(ctx context.Context, userID int64, role string, approved bool) error

// Service 用户服务层
type Service struct {
	repo              Repo
	ldapClient        ldapAuthClient
	ldapClientFactory ldapClientFactory
	hmacKey           []byte
	docCipher         pii.Encryptor
	onRoleSync        RoleSyncFunc
	photoStore        identityPhotoStore
}

// NewService 创建用户服务（构造期校验关键依赖）
func NewService(repo Repo, ldapClient *ldap.Client, hmacKey []byte, docCipher pii.Encryptor) (*Service, error) {
	if repo == nil {
		return nil, errors.New("user.NewService: repo must not be nil")
	}
	if len(hmacKey) == 0 {
		return nil, errors.New("user.NewService: hmacKey must not be empty")
	}
	if docCipher == nil {
		return nil, errors.New("user.NewService: docCipher must not be nil")
	}
	return &Service{
		repo:              repo,
		ldapClient:        ldapClient,
		ldapClientFactory: func(cfg ldap.Config) (ldapAuthClient, error) { return ldap.NewClient(cfg) },
		hmacKey:           hmacKey,
		docCipher:         docCipher,
	}, nil
}

// SetRoleSyncFunc 注册角色同步回调（认证状态变化时异步通知 Zitadel）
func (s *Service) SetRoleSyncFunc(fn RoleSyncFunc) {
	s.onRoleSync = fn
}

// SetIdentityPhotoStore 注册实名认证照片对象存储。
func (s *Service) SetIdentityPhotoStore(store identityPhotoStore) {
	s.photoStore = store
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
	SchoolID       string         `json:"schoolID"`
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
	InsecureSkipVerify *bool   `json:"insecureSkipVerify,omitempty"`
}

// SchoolLDAPConfigView 管理端学校配置返回体。
// 为避免泄漏密钥，仅返回是否已配置系统绑定密码。
type SchoolLDAPConfigView struct {
	URL                   *string `json:"url,omitempty"`
	BaseDN                *string `json:"baseDN,omitempty"`
	SystemBindDN          *string `json:"systemBindDN,omitempty"`
	UseTLS                bool    `json:"useTLS"`
	InsecureSkipVerify    bool    `json:"insecureSkipVerify"`
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
