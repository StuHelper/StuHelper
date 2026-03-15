package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
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

// ---------- Identity ----------

// SubmitIdentityRequest 提交实名认证请求
type SubmitIdentityRequest struct {
	DocType        string  `json:"docType"`
	DocNumber      string  `json:"docNumber"`
	RealName       string  `json:"realName"`
	DocPhotoFront  *string `json:"docPhotoFront"`
	DocPhotoBack   *string `json:"docPhotoBack"`
	DocPhotoSelfie *string `json:"docPhotoSelfie"`
}

// GetIdentity 获取实名认证状态信息（不含敏感字段）
func (s *Service) GetIdentity(ctx context.Context, userID int64) (*IdentityStatus, error) {
	return s.repo.GetIdentityStatusByUserID(ctx, userID)
}

// SubmitIdentity 提交实名认证
func (s *Service) SubmitIdentity(ctx context.Context, userID int64, req SubmitIdentityRequest) (*IdentityStatus, error) {
	// 仅查询状态字段判断是否已存在/已认证，不读取 doc_number_enc/person_uid
	existing, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity check existing: %w", err)
	}
	if existing != nil {
		if existing.Verified {
			return nil, ErrIdentityAlreadyVerified
		}
		return nil, ErrIdentityAlreadyExists
	}

	// 非大陆证件需要上传照片
	if req.DocType != DocTypeMainlandID {
		if req.DocPhotoFront == nil || *req.DocPhotoFront == "" {
			return nil, ErrPhotoRequired
		}
	}

	// 计算 person_uid: HMAC(doc_type + ':' + doc_number)
	personUID := s.computePersonUID(req.DocType, req.DocNumber)

	// 使用 PII 加密器加密证件号
	docNumberEnc, err := s.docCipher.Encrypt(req.DocNumber)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity encrypt doc number: %w", err)
	}

	identity := &IdentityRecord{
		UserID:         userID,
		DocType:        req.DocType,
		DocNumberEnc:   docNumberEnc,
		PersonUID:      personUID,
		RealName:       req.RealName,
		Verified:       false,
		DocPhotoFront:  req.DocPhotoFront,
		DocPhotoBack:   req.DocPhotoBack,
		DocPhotoSelfie: req.DocPhotoSelfie,
	}

	// 对大陆身份证：尝试匹配学籍数据库自动验证
	if req.DocType == DocTypeMainlandID {
		matched, err := s.tryAcademicDBMatch(ctx, req.DocNumber, req.RealName)
		if err != nil {
			logger.L().Warn("academic DB match failed, falling through",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
		}
		if matched {
			method := VerifyMethodAcademicDB
			now := time.Now()
			identity.Verified = true
			identity.VerifyMethod = &method
			identity.VerifiedAt = &now
		}
		// 如果学籍不匹配，大陆身份证后续可接入腾讯云身份核验（当前为预留）
	}

	if err := s.repo.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("SubmitIdentity create: %w", err)
	}

	// 重新查询状态（不含敏感字段）
	result, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity reload: %w", err)
	}
	return result, nil
}

// tryAcademicDBMatch 尝试通过学籍数据库匹配进行自动实名验证
func (s *Service) tryAcademicDBMatch(ctx context.Context, docNumber, realName string) (bool, error) {
	// 根据证件号在学籍库中查找匹配记录
	students, err := s.repo.FindAcademicStudentsByPersonUID(ctx, DocTypeMainlandID, docNumber)
	if err != nil {
		return false, err
	}

	for _, stu := range students {
		if stu.XM != nil && strings.EqualFold(strings.TrimSpace(*stu.XM), strings.TrimSpace(realName)) {
			return true, nil
		}
	}
	return false, nil
}

// ---------- Profile (Student Verification) ----------

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

// GetProfile 获取学生认证档案
func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	return s.repo.GetProfileByUserID(ctx, userID)
}

// VerifyStudent 学生认证（LDAP 方式）
func (s *Service) VerifyStudent(ctx context.Context, userID int64, req VerifyStudentRequest) (*Profile, error) {
	// 检查是否已通过实名认证（使用状态查询，不读取敏感字段）
	identityStatus, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent check identity: %w", err)
	}
	if identityStatus == nil || !identityStatus.Verified {
		return nil, ErrIdentityRequired
	}

	// 检查是否已通过学生认证
	existing, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent check existing: %w", err)
	}
	if existing != nil && existing.VerificationStatus == StatusVerified {
		return nil, ErrProfileAlreadyVerified
	}

	// 获取学校配置
	school, err := s.repo.GetSchoolConfig(ctx, req.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent get school config: %w", err)
	}
	if school == nil {
		return nil, ErrSchoolNotFound
	}
	if !school.Enabled {
		return nil, ErrSchoolDisabled
	}

	// 检查用户是否同意数据使用授权
	if !req.Consent {
		return nil, ErrConsentRequired
	}

	trimmedStudentID := strings.TrimSpace(req.StudentID)
	manualFormData := sanitizeManualFormData(req.ManualFormData)
	manualFieldDescriptors, err := decodeManualFieldDescriptors(school.ManualFormFields)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidManualFieldConfig, err)
	}

	if school.VerificationMethod == VerifyMethodLDAP {
		if trimmedStudentID == "" {
			return nil, ErrStudentIDRequired
		}
		if req.Password == "" {
			return nil, ErrPasswordRequired
		}
	} else {
		allowedFields := make(map[string]ManualFieldDescriptor, len(manualFieldDescriptors))
		for _, field := range manualFieldDescriptors {
			allowedFields[field.Key] = field
		}
		for key := range manualFormData {
			if _, ok := allowedFields[key]; !ok {
				return nil, fmt.Errorf("%w: %s", ErrManualFieldInvalid, key)
			}
		}
		for _, field := range manualFieldDescriptors {
			value := getManualFormValueAsString(manualFormData, field.Key)
			if value == "" && field.Key == "studentID" {
				value = trimmedStudentID
			}
			if field.Required && value == "" {
				return nil, fmt.Errorf("%w: %s", ErrManualFieldRequired, field.Key)
			}
			if field.Type == "select" && value != "" && !slices.Contains(field.Options, value) {
				return nil, fmt.Errorf("%w: %s", ErrManualFieldInvalid, field.Key)
			}
			if field.Type == "date" && value != "" {
				if _, err := time.Parse("2006-01-02", value); err != nil {
					return nil, fmt.Errorf("%w: %s", ErrManualFieldInvalid, field.Key)
				}
			}
		}
		if trimmedStudentID == "" {
			trimmedStudentID = getManualFormValueAsString(manualFormData, "studentID")
		}
	}

	now := time.Now()
	method := VerifyMethodLDAP
	schoolID := req.SchoolID

	profile := &Profile{
		UserID:             userID,
		SchoolID:           &schoolID,
		VerificationStatus: StatusUnverified,
		VerificationMethod: &method,
		ConsentGivenAt:     &now,
	}

	// 按学校认证方式执行验证
	if school.VerificationMethod == VerifyMethodLDAP {
		if s.ldapClient == nil {
			return nil, fmt.Errorf("VerifyStudent: LDAP client not configured")
		}

		// LDAP bind 验证
		loginResult, err := s.ldapClient.Login(ctx, trimmedStudentID, req.Password)
		if err != nil {
			if errors.Is(err, ldap.ErrInvalidUID) {
				return nil, ErrLDAPFailed
			}
			return nil, fmt.Errorf("VerifyStudent LDAP login: %w", err)
		}
		if !loginResult.Authenticated {
			return nil, ErrLDAPFailed
		}

		// LDAP 验证成功，查询学生详细信息
		ldapInfo, err := s.ldapClient.QueryUserByUID(ctx, trimmedStudentID)
		if err != nil {
			logger.L().Warn("LDAP query user info failed after successful login",
				zap.Int64("user_id", userID),
				zap.String("student_id", trimmedStudentID),
				zap.Error(err),
			)
			// 查询失败不阻断认证，但无法自动绑定手机号
		}

		// 从学籍数据库查找该学号关联的所有学籍
		studentIDs := []string{trimmedStudentID}
		academicStudent, err := s.repo.GetAcademicStudentByXH(ctx, trimmedStudentID)
		if err != nil {
			logger.L().Warn("failed to query academic student",
				zap.String("student_id", trimmedStudentID),
				zap.Error(err),
			)
		}

		// 通过 person_uid 查找同一人的所有学籍（升学场景）
		if academicStudent != nil && academicStudent.SFZJH != nil && *academicStudent.SFZJH != "" {
			allStudents, err := s.repo.FindAcademicStudentsByPersonUID(ctx, "", *academicStudent.SFZJH)
			if err != nil {
				logger.L().Warn("failed to find all student records by person uid",
					zap.Int64("user_id", userID),
					zap.Error(err),
				)
			} else if len(allStudents) > 0 {
				studentIDs = make([]string, 0, len(allStudents))
				for _, stu := range allStudents {
					studentIDs = append(studentIDs, stu.XH)
				}
				// 选择 rxnj（入学年级）最新的记录作为 active_student_id
				sort.Slice(allStudents, func(i, j int) bool {
					iRXNJ := ""
					jRXNJ := ""
					if allStudents[i].RXNJ != nil {
						iRXNJ = *allStudents[i].RXNJ
					}
					if allStudents[j].RXNJ != nil {
						jRXNJ = *allStudents[j].RXNJ
					}
					return iRXNJ > jRXNJ
				})
				activeID := allStudents[0].XH
				profile.ActiveStudentID = &activeID
			}
		}

		if profile.ActiveStudentID == nil {
			profile.ActiveStudentID = &trimmedStudentID
		}

		profile.StudentIDs = studentIDs
		profile.VerificationStatus = StatusVerified
		profile.VerifiedAt = &now

		// 如果 LDAP 查询成功，尝试绑定手机号
		if ldapInfo != nil && ldapInfo.Mobile != "" {
			phone := ldapInfo.Mobile
			profile.Phone = &phone
			// 手机号来自 LDAP 可信源，自动标记为已验证
			profile.PhoneVerified = true
		}
	} else {
		// 非 LDAP 方式（手动审核）
		manualMethod := VerifyMethodManual
		profile.VerificationMethod = &manualMethod
		profile.VerificationStatus = StatusPending
		if trimmedStudentID != "" {
			profile.StudentIDs = []string{trimmedStudentID}
			profile.ActiveStudentID = &trimmedStudentID
		}
		if len(manualFormData) > 0 {
			if trimmedStudentID != "" {
				if _, ok := manualFormData["studentID"]; !ok {
					manualFormData["studentID"] = trimmedStudentID
				}
			}
			raw, err := json.Marshal(manualFormData)
			if err != nil {
				return nil, fmt.Errorf("VerifyStudent marshal manual form data: %w", err)
			}
			profile.ManualFormData = raw
		}
	}

	// 创建或更新 profile
	if existing != nil {
		profile.CreatedAt = existing.CreatedAt
		if err := s.repo.UpdateProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("VerifyStudent update profile: %w", err)
		}
	} else {
		if err := s.repo.CreateProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("VerifyStudent create profile: %w", err)
		}
	}

	// 重新查询以获取完整记录
	result, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent reload: %w", err)
	}
	return result, nil
}

// BindPhone 绑定手机号
func (s *Service) BindPhone(ctx context.Context, userID int64, phone string) error {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("BindPhone get profile: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	profile.Phone = &phone
	profile.PhoneVerified = false // 短信验证完成后才标记为 true

	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return fmt.Errorf("BindPhone update: %w", err)
	}
	return nil
}

// ---------- Academic Info ----------

// GetAcademicInfo 获取学籍信息
func (s *Service) GetAcademicInfo(ctx context.Context, studentID string) (*AcademicStudent, error) {
	return s.repo.GetAcademicStudentByXH(ctx, studentID)
}

// ---------- School Configs ----------

// ListSchools 获取所有启用的学校列表
func (s *Service) ListSchools(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListSchoolConfigs(ctx)
}

// ---------- Admin ----------

// GetInternalUserID 根据外部ID获取内部用户ID
func (s *Service) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	return s.repo.GetInternalUserID(ctx, externalID)
}

// ListIdentities 分页查询实名认证审核列表（管理端，不含敏感字段）
func (s *Service) ListIdentities(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error) {
	return s.repo.ListIdentityReviewItems(ctx, status, page, pageSize)
}

// ReviewIdentity 管理员审核实名认证（通过/驳回）
// 使用精准更新，不读取也不回写敏感字段
func (s *Service) ReviewIdentity(ctx context.Context, userID int64, approved bool, reason string) error {
	if !approved && strings.TrimSpace(reason) == "" {
		return ErrRejectionReasonRequired
	}

	// 只查询状态，不触碰 doc_number_enc
	identityStatus, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ReviewIdentity get: %w", err)
	}
	if identityStatus == nil {
		return ErrIdentityNotFound
	}

	var (
		verifyMethod    *string
		verifiedAt      *time.Time
		rejectionReason *string
	)

	if approved {
		method := VerifyMethodManual
		now := time.Now()
		verifyMethod = &method
		verifiedAt = &now
	} else {
		rejectionReason = &reason
	}

	return s.repo.UpdateIdentityReviewStatus(ctx, userID, approved, verifyMethod, verifiedAt, rejectionReason)
}

// ListProfiles 分页查询学生认证档案（管理端）
func (s *Service) ListProfiles(ctx context.Context, status, schoolID string, page, pageSize int) ([]Profile, int, error) {
	return s.repo.ListProfilesByStatus(ctx, status, schoolID, page, pageSize)
}

// ReviewStudentVerification 管理员审核学生认证（通过/驳回）
func (s *Service) ReviewStudentVerification(ctx context.Context, userID int64, approved bool, reason string) error {
	if !approved && strings.TrimSpace(reason) == "" {
		return ErrRejectionReasonRequired
	}

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ReviewStudentVerification get: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	now := time.Now()
	if approved {
		profile.VerificationStatus = StatusVerified
	} else {
		profile.VerificationStatus = StatusRejected
	}
	if approved {
		profile.VerifiedAt = &now
	}

	return s.repo.UpdateProfile(ctx, profile)
}

// ListAllSchoolConfigs 获取所有学校配置（含禁用，管理端用）
func (s *Service) ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListAllSchoolConfigs(ctx)
}

// UpdateSchoolConfig 更新学校认证配置
// 使用合并更新语义，保持未提供字段的现有值。
func (s *Service) UpdateSchoolConfig(ctx context.Context, schoolID string, input UpdateSchoolConfigInput) error {
	config, err := s.repo.GetSchoolConfig(ctx, schoolID)
	if err != nil {
		return fmt.Errorf("UpdateSchoolConfig get existing: %w", err)
	}
	if config == nil {
		return ErrSchoolNotFound
	}

	if input.SchoolName != nil {
		config.SchoolName = *input.SchoolName
	}
	if input.VerificationMethod != nil {
		config.VerificationMethod = *input.VerificationMethod
	}
	if input.AcademicDBTable != nil {
		value := *input.AcademicDBTable
		config.AcademicDBTable = &value
	}
	if input.ConsentText != nil {
		value := *input.ConsentText
		config.ConsentText = &value
	}
	if input.Enabled != nil {
		config.Enabled = *input.Enabled
	}
	if input.LDAPConfig != nil {
		raw, err := json.Marshal(*input.LDAPConfig)
		if err != nil {
			return fmt.Errorf("UpdateSchoolConfig marshal ldapConfig: %w", err)
		}
		config.LDAPConfig = raw
	}
	if input.ManualFormFields != nil {
		if err := validateManualFieldDescriptors(*input.ManualFormFields); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidManualFieldConfig, err)
		}
		raw, err := json.Marshal(*input.ManualFormFields)
		if err != nil {
			return fmt.Errorf("UpdateSchoolConfig marshal manualFormFields: %w", err)
		}
		config.ManualFormFields = raw
	}

	return s.repo.UpdateSchoolConfig(ctx, config)
}

// ListSystemConfigs 获取所有系统配置项
func (s *Service) ListSystemConfigs(ctx context.Context) ([]SystemConfig, error) {
	return s.repo.ListSystemConfigs(ctx)
}

// UpdateSystemConfig 更新系统配置项
func (s *Service) UpdateSystemConfig(ctx context.Context, key, value string) error {
	return s.repo.UpdateSystemConfig(ctx, key, value)
}

// ---------- Helpers ----------

// computePersonUID 计算 person_uid: HMAC-SHA256(doc_type + ':' + doc_number)
func (s *Service) computePersonUID(docType, docNumber string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(docType + ":" + docNumber))
	return hex.EncodeToString(mac.Sum(nil))
}
