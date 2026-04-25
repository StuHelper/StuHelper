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

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

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
	if err := validateStudentVerificationTransition(existing); err != nil {
		return nil, err
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
			return nil, fmt.Errorf("VerifyStudent query academic student: %w", err)
		}
		if academicStudent == nil {
			return nil, ErrStudentNotFound
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

	var (
		verifiedPhoneEnc  []byte
		verifiedPhoneHash string
	)
	if verifiedPhoneRaw != "" {
		verifiedPhoneEnc, verifiedPhoneHash, err = s.encryptPhone(verifiedPhoneRaw)
		if err != nil {
			logger.L().Warn("failed to prepare LDAP phone for secure storage",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			verifiedPhoneEnc = nil
			verifiedPhoneHash = ""
		} else {
			masked := phoneutil.Mask(verifiedPhoneRaw)
			profile.Phone = &masked
			profile.PhoneVerified = true
		}
	}

	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txExisting, err := s.repo.GetProfileByUserIDTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("VerifyStudent check existing tx: %w", err)
		}
		if err := validateStudentVerificationTransition(txExisting); err != nil {
			return err
		}

		if len(verifiedPhoneEnc) > 0 {
			if err := s.repo.SetUserPhoneTx(ctx, tx, userID, verifiedPhoneEnc, verifiedPhoneHash); err != nil {
				return fmt.Errorf("VerifyStudent set phone tx: %w", err)
			}
		}

		if txExisting != nil {
			profile.CreatedAt = txExisting.CreatedAt
			if err := s.repo.UpdateProfileTx(ctx, tx, profile); err != nil {
				return fmt.Errorf("VerifyStudent update profile tx: %w", err)
			}
		} else {
			if err := s.repo.CreateProfileTx(ctx, tx, profile); err != nil {
				return fmt.Errorf("VerifyStudent create profile tx: %w", err)
			}
		}

		if err := s.enqueueVerificationProjectionTx(ctx, tx, userID, profile.VerificationStatus); err != nil {
			return fmt.Errorf("VerifyStudent enqueue projections: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudent reload: %w", err)
	}
	if result != nil {
		if err := s.hydrateProfilePhone(result); err != nil {
			return nil, fmt.Errorf("VerifyStudent hydrate profile phone: %w", err)
		}
	}

	return result, nil
}

func validateStudentVerificationTransition(existing *Profile) error {
	if existing == nil {
		return nil
	}
	switch existing.VerificationStatus {
	case StatusVerified:
		return ErrProfileAlreadyVerified
	case StatusPending:
		return ErrProfilePendingReview
	default:
		return nil
	}
}

// ErrInvalidPhoneFormat 手机号格式无效
var ErrInvalidPhoneFormat = errors.New("invalid phone number format")

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
	student, err := s.repo.GetAcademicStudentByXHFromTable(ctx, studentID, tableName)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateAcademicStudent(student); err != nil {
		return nil, fmt.Errorf("getAcademicStudentByXH hydrate academic student: %w", err)
	}
	return student, nil
}

func (s *Service) findAcademicStudentsByPersonUID(ctx context.Context, sfzjlxdm string, sfzjh string, tableName string) ([]AcademicStudent, error) {
	if tableName == "" {
		return nil, fmt.Errorf("academic table name is required for person UID lookup")
	}
	list, err := s.repo.FindAcademicStudentsByPersonUIDFromTable(ctx, sfzjlxdm, sfzjh, tableName)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if err := s.hydrateAcademicStudent(&list[i]); err != nil {
			return nil, fmt.Errorf("findAcademicStudentsByPersonUID hydrate academic student: %w", err)
		}
	}
	return list, nil
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
	return s.ldapClientFactory(ldapCfg)
}

type schoolLDAPSettings struct {
	URL                string `json:"url"`
	BaseDN             string `json:"baseDN"`
	SystemBindDN       string `json:"systemBindDN"`
	SystemBindPassword string `json:"systemBindPassword"`
	UseTLS             bool   `json:"useTLS"`
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
		!settings.UseTLS
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
	}, nil
}
