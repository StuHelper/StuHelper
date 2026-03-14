package user

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// 业务错误定义
var (
	ErrIdentityAlreadyExists  = errors.New("identity already exists")
	ErrIdentityAlreadyVerified = errors.New("identity already verified")
	ErrProfileAlreadyVerified = errors.New("profile already verified")
	ErrSchoolNotFound         = errors.New("school not found")
	ErrSchoolDisabled         = errors.New("school verification disabled")
	ErrConsentRequired        = errors.New("consent is required")
	ErrPhotoRequired          = errors.New("photo upload required for non-mainland documents")
	ErrLDAPFailed             = errors.New("LDAP verification failed")
	ErrIdentityRequired       = errors.New("identity verification required before student verification")
	ErrStudentNotFound        = errors.New("student record not found in academic database")
	ErrProfileNotFound        = errors.New("student profile not found")
	ErrIdentityNotFound       = errors.New("identity not found")
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

// Service 用户服务层
type Service struct {
	repo       *Repository
	ldapClient *ldap.Client
	hmacKey    []byte
	aesKey     []byte // AES-256 key for doc number encryption (32 bytes)
}

// NewService 创建用户服务
func NewService(repo *Repository, ldapClient *ldap.Client, hmacKey []byte) *Service {
	return &Service{
		repo:       repo,
		ldapClient: ldapClient,
		hmacKey:    hmacKey,
	}
}

// SetAESKey 设置 AES 加密密钥（可选，用于证件号加密）
func (s *Service) SetAESKey(key []byte) {
	s.aesKey = key
}

// ---------- Identity ----------

// SubmitIdentityRequest 提交实名认证请求
type SubmitIdentityRequest struct {
	DocType       string  `json:"docType"`
	DocNumber     string  `json:"docNumber"`
	RealName      string  `json:"realName"`
	DocPhotoFront *string `json:"docPhotoFront"`
	DocPhotoBack  *string `json:"docPhotoBack"`
	DocPhotoSelfie *string `json:"docPhotoSelfie"`
}

// GetIdentity 获取实名认证信息
func (s *Service) GetIdentity(ctx context.Context, userID int64) (*Identity, error) {
	return s.repo.GetIdentityByUserID(ctx, userID)
}

// SubmitIdentity 提交实名认证
func (s *Service) SubmitIdentity(ctx context.Context, userID int64, req SubmitIdentityRequest) (*Identity, error) {
	// 检查是否已存在认证记录
	existing, err := s.repo.GetIdentityByUserID(ctx, userID)
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

	// 加密证件号
	docNumberEnc, err := s.encryptDocNumber(req.DocNumber)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity encrypt doc number: %w", err)
	}

	identity := &Identity{
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

	// 重新查询以获取完整记录（含 created_at/updated_at）
	result, err := s.repo.GetIdentityByUserID(ctx, userID)
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
	SchoolID  string `json:"schoolID"`
	StudentID string `json:"studentID"`
	Password  string `json:"password"`
	Consent   bool   `json:"consent"`
}

// BindPhoneRequest 绑定手机号请求
type BindPhoneRequest struct {
	Phone string `json:"phone"`
	// Code 短信验证码（预留，当前阶段不校验）
	Code string `json:"code"`
}

// GetProfile 获取学生认证档案
func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	return s.repo.GetProfileByUserID(ctx, userID)
}

// VerifyStudent 学生认证（LDAP 方式）
func (s *Service) VerifyStudent(ctx context.Context, userID int64, req VerifyStudentRequest) (*Profile, error) {
	// 检查是否已通过实名认证
	identity, err := s.repo.GetIdentityByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent check identity: %w", err)
	}
	if identity == nil || !identity.Verified {
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
	if school.VerificationMethod == "ldap" {
		if s.ldapClient == nil {
			return nil, fmt.Errorf("VerifyStudent: LDAP client not configured")
		}

		// LDAP bind 验证
		loginResult, err := s.ldapClient.Login(ctx, req.StudentID, req.Password)
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
		ldapInfo, err := s.ldapClient.QueryUserByUID(ctx, req.StudentID)
		if err != nil {
			logger.L().Warn("LDAP query user info failed after successful login",
				zap.Int64("user_id", userID),
				zap.String("student_id", req.StudentID),
				zap.Error(err),
			)
			// 查询失败不阻断认证，但无法自动绑定手机号
		}

		// 从学籍数据库查找该学号关联的所有学籍
		studentIDs := []string{req.StudentID}
		academicStudent, err := s.repo.GetAcademicStudentByXH(ctx, req.StudentID)
		if err != nil {
			logger.L().Warn("failed to query academic student",
				zap.String("student_id", req.StudentID),
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
			profile.ActiveStudentID = &req.StudentID
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
		profile.StudentIDs = []string{req.StudentID}
		profile.ActiveStudentID = &req.StudentID
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
func (s *Service) BindPhone(ctx context.Context, userID int64, req BindPhoneRequest) error {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("BindPhone get profile: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	// TODO: 接入短信验证码校验（当前阶段预留接口，不做实际验证）
	profile.Phone = &req.Phone
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

// ListIdentities 分页查询实名认证记录（管理端）
func (s *Service) ListIdentities(ctx context.Context, status string, page, pageSize int) ([]Identity, int, error) {
	return s.repo.ListIdentitiesByStatus(ctx, status, page, pageSize)
}

// ReviewIdentity 管理员审核实名认证（通过/驳回）
func (s *Service) ReviewIdentity(ctx context.Context, userID int64, approved bool, reason string) error {
	identity, err := s.repo.GetIdentityByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ReviewIdentity get: %w", err)
	}
	if identity == nil {
		return ErrIdentityNotFound
	}

	now := time.Now()
	if approved {
		method := VerifyMethodManual
		identity.Verified = true
		identity.VerifyMethod = &method
		identity.VerifiedAt = &now
		identity.RejectionReason = nil
	} else {
		identity.Verified = false
		identity.VerifyMethod = nil
		identity.VerifiedAt = nil
		identity.RejectionReason = &reason
	}

	return s.repo.UpdateIdentity(ctx, identity)
}

// ListProfiles 分页查询学生认证档案（管理端）
func (s *Service) ListProfiles(ctx context.Context, status, schoolID string, page, pageSize int) ([]Profile, int, error) {
	return s.repo.ListProfilesByStatus(ctx, status, schoolID, page, pageSize)
}

// ReviewStudentVerification 管理员审核学生认证（通过/驳回）
func (s *Service) ReviewStudentVerification(ctx context.Context, userID int64, status, reason string) error {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ReviewStudentVerification get: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	now := time.Now()
	profile.VerificationStatus = status
	if status == StatusVerified {
		profile.VerifiedAt = &now
	}
	// Store reason in a way that we can query later if needed
	// For now, just update the status

	return s.repo.UpdateProfile(ctx, profile)
}

// ListAllSchoolConfigs 获取所有学校配置（含禁用，管理端用）
func (s *Service) ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListAllSchoolConfigs(ctx)
}

// UpdateSchoolConfig 更新学校认证配置
func (s *Service) UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error {
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

// encryptDocNumber 加密证件号（AES-256-GCM）
// 如果 AES 密钥未配置，返回明文的 hex 编码作为降级方案
func (s *Service) encryptDocNumber(docNumber string) ([]byte, error) {
	if len(s.aesKey) == 0 {
		// AES 密钥未配置，降级为明文存储（开发环境）
		return []byte(docNumber), nil
	}

	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("encryptDocNumber new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryptDocNumber new GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encryptDocNumber generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(docNumber), nil)
	return ciphertext, nil
}
