package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
)

// 业务错误定义
var (
	ErrIdentityAlreadyExists    = errors.New("identity already exists")
	ErrIdentityAlreadyVerified  = errors.New("identity already verified")
	ErrProfileAlreadyVerified   = errors.New("profile already verified")
	ErrSchoolNotFound           = errors.New("school not found")
	ErrSchoolDisabled           = errors.New("school verification disabled")
	ErrConsentRequired          = errors.New("consent is required")
	ErrPhotoRequired            = errors.New("photo upload required for non-mainland documents")
	ErrLDAPFailed               = errors.New("LDAP verification failed")
	ErrIdentityRequired         = errors.New("identity verification required before student verification")
	ErrStudentIDRequired        = errors.New("student ID is required for LDAP verification")
	ErrPasswordRequired         = errors.New("password is required for LDAP verification")
	ErrStudentNotFound          = errors.New("student record not found in academic database")
	ErrProfileNotFound          = errors.New("student profile not found")
	ErrIdentityNotFound         = errors.New("identity not found")
	ErrRejectionReasonRequired  = errors.New("rejection reason is required when rejecting")
	ErrManualFieldRequired      = errors.New("required manual form field is missing")
	ErrManualFieldInvalid       = errors.New("manual form field value is invalid")
	ErrInvalidManualFieldConfig = errors.New("manual form field config is invalid")
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

// Repo defines the data access methods required by Service.
type Repo interface {
	GetIdentityStatusByUserID(ctx context.Context, userID int64) (*IdentityStatus, error)
	CreateIdentity(ctx context.Context, identity *IdentityRecord) error
	ListIdentityReviewItems(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error)
	UpdateIdentityReviewStatus(ctx context.Context, userID int64, approved bool, verifyMethod *string, verifiedAt *time.Time, rejectionReason *string) error

	GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error)
	CreateProfile(ctx context.Context, profile *Profile) error
	UpdateProfile(ctx context.Context, profile *Profile) error
	ListProfilesByStatus(ctx context.Context, status string, schoolID string, page, pageSize int) ([]Profile, int, error)

	GetSchoolConfig(ctx context.Context, schoolID string) (*SchoolConfig, error)
	ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error)
	ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error)
	UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error

	GetAcademicStudentByXH(ctx context.Context, xh string) (*AcademicStudent, error)
	FindAcademicStudentsByPersonUID(ctx context.Context, sfzjlxdm, sfzjh string) ([]AcademicStudent, error)

	ListSystemConfigs(ctx context.Context) ([]SystemConfig, error)
	UpdateSystemConfig(ctx context.Context, key, value string) error

	GetInternalUserID(ctx context.Context, externalID string) (int64, error)
}

// Service 用户服务层
type Service struct {
	repo       Repo
	ldapClient *ldap.Client
	hmacKey    []byte
	docCipher  pii.Encryptor
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
		repo:       repo,
		ldapClient: ldapClient,
		hmacKey:    hmacKey,
		docCipher:  docCipher,
	}, nil
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

// VerifyStudentRequest 学生认证请求
type VerifyStudentRequest struct {
	SchoolID       string         `json:"schoolID"`
	StudentID      string         `json:"studentID"`
	Password       string         `json:"password"`
	ManualFormData map[string]any `json:"manualFormData"`
	Consent        bool           `json:"consent"`
}

// UpdateSchoolConfigInput 学校配置更新请求（管理端）
// 使用可选字段做合并更新，避免未提供字段被误清空。
type UpdateSchoolConfigInput struct {
	SchoolName         *string
	VerificationMethod *string
	LDAPConfig         *map[string]any
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
