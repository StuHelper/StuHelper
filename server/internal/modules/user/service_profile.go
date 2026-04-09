package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

type academicTableRepo interface {
	GetAcademicStudentByXHFromTable(ctx context.Context, xh string, tableName string) (*AcademicStudent, error)
	FindAcademicStudentsByPersonUIDFromTable(ctx context.Context, sfzjlxdm, sfzjh string, tableName string) ([]AcademicStudent, error)
}

// GetUserSurface 聚合当前用户的身份认证、学生认证、手机绑定和能力信息。
// displayName 和 avatarURL 由 auth 中间件注入，capabilities 由角色展开得到。
func (s *Service) GetUserSurface(ctx context.Context, userID int64, displayName, avatarURL string, capabilities []string) (*UserSurface, error) {
	identity, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSurface identity: %w", err)
	}

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSurface profile: %w", err)
	}

	return &UserSurface{
		DisplayName:        displayName,
		AvatarURL:          avatarURL,
		IdentityStatus:     deriveIdentityStatus(identity),
		VerificationStatus: deriveVerificationStatus(profile),
		PhoneBound:         profile != nil && profile.PhoneVerified,
		Capabilities:       capabilities,
	}, nil
}

// deriveIdentityStatus 将实名认证记录映射为 API 枚举
func deriveIdentityStatus(identity *IdentityStatus) string {
	if identity == nil {
		return "none"
	}
	if identity.Verified {
		return "approved"
	}
	if identity.RejectionReason != nil {
		return "rejected"
	}
	return "pending"
}

// deriveVerificationStatus 将学生认证档案映射为 API 枚举
func deriveVerificationStatus(profile *Profile) string {
	if profile == nil {
		return "none"
	}
	switch profile.VerificationStatus {
	case StatusVerified:
		return "approved"
	case StatusPending:
		return "pending"
	case StatusRejected:
		return "rejected"
	default:
		return "none"
	}
}

// GetProfile 获取学生认证档案
func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil || profile == nil {
		return profile, err
	}
	profile.Phone = normalizeMaskedPhone(profile.Phone)
	return profile, nil
}

func normalizeMaskedPhone(phone *string) *string {
	if phone == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*phone)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, "*") {
		return &trimmed
	}
	masked := phoneutil.Mask(trimmed)
	if masked == "***" {
		return nil
	}
	return &masked
}

func (s *Service) storeEncryptedPhone(ctx context.Context, userID int64, phone string) error {
	phoneEnc, err := s.docCipher.Encrypt(phone)
	if err != nil {
		return fmt.Errorf("encrypt phone: %w", err)
	}
	phoneHash, err := phoneutil.HashLookupWithKey(phone, s.hmacKey)
	if err != nil {
		return fmt.Errorf("hash phone: %w", err)
	}
	if err := s.repo.SetUserPhone(ctx, userID, phoneEnc, phoneHash); err != nil {
		return err
	}
	return nil
}

// VerifyStudent 学生认证（LDAP 方式）
func (s *Service) VerifyStudent(ctx context.Context, userID int64, req VerifyStudentRequest) (*Profile, error) {
	identityStatus, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent check identity: %w", err)
	}
	if identityStatus == nil || !identityStatus.Verified {
		return nil, ErrIdentityRequired
	}

	existing, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent check existing: %w", err)
	}
	if existing != nil && existing.VerificationStatus == StatusVerified {
		return nil, ErrProfileAlreadyVerified
	}
	if existing != nil && existing.VerificationStatus == StatusPending {
		return nil, ErrProfilePendingReview
	}

	school, err := s.loadEnabledSchoolConfig(ctx, req.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent get school config: %w", err)
	}
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
	verifiedPhoneRaw := ""
	profile := &Profile{
		UserID:             userID,
		SchoolID:           &schoolID,
		VerificationStatus: StatusUnverified,
		VerificationMethod: &method,
		ConsentGivenAt:     &now,
	}

	if school.VerificationMethod == VerifyMethodLDAP {
		academicTableName, err := s.ensureAcademicTableConfigured(school)
		if err != nil {
			return nil, fmt.Errorf("VerifyStudent academic table: %w", err)
		}

		ldapClient, err := s.ensureLDAPClientForSchool(school)
		if err != nil {
			return nil, fmt.Errorf("VerifyStudent ldap client: %w", err)
		}

		loginResult, err := ldapClient.Login(ctx, trimmedStudentID, req.Password)
		if err != nil {
			if errors.Is(err, ldap.ErrInvalidUID) {
				return nil, ErrLDAPFailed
			}
			return nil, fmt.Errorf("VerifyStudent LDAP login: %w", err)
		}
		if !loginResult.Authenticated {
			return nil, ErrLDAPFailed
		}

		ldapInfo, err := ldapClient.QueryUserByUID(ctx, trimmedStudentID)
		if err != nil {
			logger.L().Warn("LDAP query user info failed after successful login",
				zap.Int64("user_id", userID),
				zap.String("student_id", trimmedStudentID),
				zap.Error(err),
			)
		}

		studentIDs := []string{trimmedStudentID}
		academicStudent, err := s.getAcademicStudentByXH(ctx, trimmedStudentID, academicTableName)
		if err != nil {
			logger.L().Warn("failed to query academic student",
				zap.String("student_id", trimmedStudentID),
				zap.Error(err),
			)
		}

		if academicStudent != nil && academicStudent.SFZJH != nil && *academicStudent.SFZJH != "" {
			docType := ""
			if academicStudent.SFZJLXDM != nil {
				docType = *academicStudent.SFZJLXDM
			}
			allStudents, err := s.findAcademicStudentsByPersonUID(ctx, docType, *academicStudent.SFZJH, academicTableName)
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
		// 根据学校审批策略决定状态：auto → 直接通过，manual → 需人工审核
		if school.IsAutoApprove() {
			profile.VerificationStatus = StatusVerified
			profile.VerifiedAt = &now
		} else {
			profile.VerificationStatus = StatusPending
		}

		if ldapInfo != nil && ldapInfo.Mobile != "" {
			phone := strings.TrimSpace(ldapInfo.Mobile)
			if phoneutil.IsValidMainlandPhone(phone) {
				verifiedPhoneRaw = phone
			}
		}
	} else {
		manualMethod := VerifyMethodManual
		profile.VerificationMethod = &manualMethod
		// 根据学校审批策略决定状态（manual 提交也可能自动批准）
		if school.IsAutoApprove() {
			profile.VerificationStatus = StatusVerified
			profile.VerifiedAt = &now
		} else {
			profile.VerificationStatus = StatusPending
		}
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

	if verifiedPhoneRaw != "" {
		if err := s.storeEncryptedPhone(ctx, userID, verifiedPhoneRaw); err != nil {
			logger.L().Warn("failed to persist LDAP phone to secure user storage",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			verifiedPhoneRaw = ""
		} else {
			masked := phoneutil.Mask(verifiedPhoneRaw)
			profile.Phone = &masked
			profile.PhoneVerified = true
		}
	}

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

	result, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent reload: %w", err)
	}
	if result != nil {
		result.Phone = normalizeMaskedPhone(result.Phone)
	}

	// auto-approve 路径：持久化成功后同步角色
	if profile.VerificationStatus == StatusVerified {
		s.syncRole(ctx, userID, "verified_student", true)
	}

	return result, nil
}

// ErrInvalidPhoneFormat 手机号格式无效
var ErrInvalidPhoneFormat = errors.New("invalid phone number format")

// BindPhone 绑定手机号
func (s *Service) BindPhone(ctx context.Context, userID int64, phone string) error {
	trimmed := strings.TrimSpace(phone)
	if !phoneutil.IsValidMainlandPhone(trimmed) {
		return ErrInvalidPhoneFormat
	}

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("BindPhone get profile: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	if err := s.storeEncryptedPhone(ctx, userID, trimmed); err != nil {
		if errors.Is(err, ErrPhoneAlreadyBound) {
			return ErrPhoneAlreadyBound
		}
		return fmt.Errorf("BindPhone store phone: %w", err)
	}

	masked := phoneutil.Mask(trimmed)
	profile.Phone = &masked
	profile.PhoneVerified = true

	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return fmt.Errorf("BindPhone update: %w", err)
	}
	return nil
}

// GetAcademicInfo 获取学籍信息
func (s *Service) GetAcademicInfo(ctx context.Context, schoolID int64, studentID string) (*AcademicStudent, error) {
	school, err := s.loadEnabledSchoolConfig(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("GetAcademicInfo load school config: %w", err)
	}

	tableName, err := s.ensureAcademicTableConfigured(school)
	if err != nil {
		return nil, fmt.Errorf("GetAcademicInfo academic table: %w", err)
	}

	return s.getAcademicStudentByXH(ctx, studentID, tableName)
}

// ListSchools 获取所有启用的学校列表
func (s *Service) ListSchools(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListSchoolConfigs(ctx)
}

func (s *Service) getAcademicStudentByXH(ctx context.Context, studentID string, tableName string) (*AcademicStudent, error) {
	if tableName == "" {
		return nil, fmt.Errorf("academic table name is required for student lookup")
	}
	if repoWithTable, ok := s.repo.(academicTableRepo); ok {
		return repoWithTable.GetAcademicStudentByXHFromTable(ctx, studentID, tableName)
	}
	return nil, fmt.Errorf("repository does not support school-scoped academic table queries")
}

func (s *Service) findAcademicStudentsByPersonUID(ctx context.Context, sfzjlxdm string, sfzjh string, tableName string) ([]AcademicStudent, error) {
	if tableName == "" {
		return nil, fmt.Errorf("academic table name is required for person UID lookup")
	}
	if repoWithTable, ok := s.repo.(academicTableRepo); ok {
		return repoWithTable.FindAcademicStudentsByPersonUIDFromTable(ctx, sfzjlxdm, sfzjh, tableName)
	}
	return nil, fmt.Errorf("repository does not support school-scoped academic table queries")
}

func (s *Service) loadEnabledSchoolConfig(ctx context.Context, schoolID int64) (*SchoolConfig, error) {
	if schoolID == 0 {
		return nil, ErrSchoolNotFound
	}

	school, err := s.repo.GetSchoolConfig(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("load school config: %w", err)
	}
	if school == nil {
		return nil, ErrSchoolNotFound
	}
	if !school.Enabled {
		return nil, ErrSchoolDisabled
	}
	return school, nil
}

func (s *Service) ensureAcademicTableConfigured(school *SchoolConfig) (string, error) {
	if school == nil || school.AcademicDBTable == nil {
		return "", ErrAcademicTableNotConfigured
	}
	table := strings.TrimSpace(*school.AcademicDBTable)
	if table == "" {
		return "", ErrAcademicTableNotConfigured
	}

	normalized, err := normalizeAcademicDBTableName(&table)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidAcademicDBTable, err)
	}
	return normalized, nil
}

func (s *Service) ensureLDAPClientForSchool(school *SchoolConfig) (ldapAuthClient, error) {
	if school == nil {
		return nil, ErrSchoolNotFound
	}
	if strings.TrimSpace(string(school.LDAPConfig)) == "" {
		return nil, ErrSchoolLDAPConfigMissing
	}

	ldapCfg, err := parseSchoolLDAPConfig(school.LDAPConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLDAPConfigInvalid, err)
	}
	if s.ldapClientFactory == nil {
		s.ldapClientFactory = func(cfg ldap.Config) (ldapAuthClient, error) { return ldap.NewClient(cfg) }
	}
	return s.ldapClientFactory(ldapCfg)
}

type schoolLDAPSettings struct {
	URL                string `json:"url"`
	BaseDN             string `json:"baseDN"`
	SystemBindDN       string `json:"systemBindDN"`
	SystemBindPassword string `json:"systemBindPassword"`
	UseTLS             bool   `json:"useTLS"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

func decodeSchoolLDAPSettings(raw json.RawMessage) (schoolLDAPSettings, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return schoolLDAPSettings{}, nil
	}

	var settings schoolLDAPSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return schoolLDAPSettings{}, err
	}
	return normalizeSchoolLDAPSettings(settings), nil
}

func normalizeSchoolLDAPSettings(settings schoolLDAPSettings) schoolLDAPSettings {
	settings.URL = strings.TrimSpace(settings.URL)
	settings.BaseDN = strings.TrimSpace(settings.BaseDN)
	settings.SystemBindDN = strings.TrimSpace(settings.SystemBindDN)
	settings.SystemBindPassword = strings.TrimSpace(settings.SystemBindPassword)
	return settings
}

func isEmptySchoolLDAPSettings(settings schoolLDAPSettings) bool {
	return settings.URL == "" &&
		settings.BaseDN == "" &&
		settings.SystemBindDN == "" &&
		settings.SystemBindPassword == "" &&
		!settings.UseTLS &&
		!settings.InsecureSkipVerify
}

func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseSchoolLDAPConfig(raw json.RawMessage) (ldap.Config, error) {
	settings, err := decodeSchoolLDAPSettings(raw)
	if err != nil {
		return ldap.Config{}, err
	}
	if isEmptySchoolLDAPSettings(settings) {
		return ldap.Config{}, ErrSchoolLDAPConfigMissing
	}

	return ldap.Config{
		URL:                settings.URL,
		BaseDN:             settings.BaseDN,
		SystemBindDN:       settings.SystemBindDN,
		SystemBindPassword: settings.SystemBindPassword,
		UseTLS:             settings.UseTLS,
		InsecureSkipVerify: settings.InsecureSkipVerify,
	}, nil
}
