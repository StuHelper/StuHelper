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

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
)

// VerifyStudent 学生认证（LDAP 方式）
func (s *Service) VerifyStudent(ctx context.Context, userID int64, req VerifyStudentRequest) (*Profile, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
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
	if err := validateOptionalStudentID(trimmedStudentID); err != nil {
		return nil, err
	}
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
			if err := validateOptionalStudentID(trimmedStudentID); err != nil {
				return nil, err
			}
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
			if errors.Is(err, ErrLDAPFailed) {
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

		if academicStudent.SFZJH != nil && *academicStudent.SFZJH != "" {
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

	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txExisting, err := s.repo.GetProfileByUserIDForUpdateTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("VerifyStudent check existing tx: %w", err)
		}
		if err := validateStudentVerificationTransition(txExisting); err != nil {
			return err
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

	if verifiedPhoneRaw != "" {
		if err := s.syncVerifiedPhoneProjection(ctx, userID, verifiedPhoneRaw); err != nil {
			logger.L().Warn("failed to sync LDAP phone projection",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
		}
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

func validateOptionalStudentID(studentID string) error {
	if studentID == "" {
		return nil
	}
	if !schoolauth.IsValidStudentID(studentID) {
		return ErrStudentIDInvalid
	}
	return nil
}

func (s *Service) syncVerifiedPhoneProjection(ctx context.Context, userID int64, phone string) error {
	if s.profileIdentitySync == nil {
		return ErrProfileIdentitySyncMissing
	}
	_, phoneEnc, phoneHash, err := s.prepareAvailablePhoneProjection(ctx, userID, phone)
	if err != nil {
		return err
	}
	subject, err := s.repo.GetCasdoorSubject(ctx, userID)
	if err != nil {
		return fmt.Errorf("get Casdoor subject: %w", err)
	}
	if err := s.profileIdentitySync.UpdatePhone(ctx, subject, "+86"+phone); err != nil {
		return err
	}
	if err := s.repo.SetUserPhone(ctx, userID, phoneEnc, phoneHash); err != nil {
		return fmt.Errorf("set phone projection: %w", err)
	}
	return nil
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

	student, err := s.lookupAcademicStudentForSchool(ctx, school, studentID)
	if err != nil {
		return nil, fmt.Errorf("GetAcademicInfo lookup: %w", err)
	}
	return student, nil
}

func (s *Service) lookupAcademicStudentForSchool(
	ctx context.Context,
	school *SchoolConfig,
	studentID string,
) (*AcademicStudent, error) {
	normalizedID := schoolauth.NormalizeStudentID(studentID)
	if normalizedID == "" {
		return nil, ErrStudentIDRequired
	}
	if !schoolauth.IsValidStudentID(normalizedID) {
		return nil, ErrStudentIDInvalid
	}
	if s.studentDirectory != nil && school != nil && strings.TrimSpace(school.SchoolCode) != "" {
		record, handled, err := s.studentDirectory.LookupStudent(ctx, school.SchoolCode, normalizedID)
		if err != nil {
			return nil, fmt.Errorf("external student directory lookup: %w", err)
		}
		if handled {
			if record == nil {
				return nil, nil
			}
			return academicStudentFromExternalRecord(record), nil
		}
	}
	tableName, err := s.ensureAcademicTableConfigured(school)
	if err != nil {
		return nil, err
	}

	return s.getAcademicStudentByXH(ctx, normalizedID, tableName)
}

func academicStudentFromExternalRecord(record *ExternalStudentRecord) *AcademicStudent {
	if record == nil {
		return nil
	}
	name := strings.TrimSpace(record.StudentName)
	return &AcademicStudent{
		XH: strings.TrimSpace(record.StudentID),
		XM: &name,
	}
}

// ListSchools 获取所有启用的学校列表
func (s *Service) ListSchools(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListSchoolConfigs(ctx)
}

func (s *Service) ResolveEnabledSchoolIDByCode(ctx context.Context, schoolCode string) (int64, error) {
	code := strings.TrimSpace(schoolCode)
	if code == "" {
		return 0, ErrSchoolNotFound
	}
	schools, err := s.repo.ListSchoolConfigs(ctx)
	if err != nil {
		return 0, fmt.Errorf("ResolveEnabledSchoolIDByCode list schools: %w", err)
	}
	for i := range schools {
		if strings.EqualFold(strings.TrimSpace(schools[i].SchoolCode), code) {
			return schools[i].SchoolID, nil
		}
	}
	return 0, ErrSchoolNotFound
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
	if err := validateSchoolID(schoolID); err != nil {
		return nil, err
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

func (s *Service) ensureLDAPClientForSchool(school *SchoolConfig) (LDAPAuthClient, error) {
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
	if err := validateLDAPConfig(ldapCfg); err != nil {
		return nil, err
	}
	if s.ldapClientFactory == nil {
		return nil, ErrLDAPConfigInvalid
	}
	client, err := s.ldapClientFactory(ldapCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLDAPConfigInvalid, err)
	}
	return client, nil
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

func parseSchoolLDAPConfig(raw json.RawMessage) (LDAPConfig, error) {
	settings, err := decodeSchoolLDAPSettings(raw)
	if err != nil {
		return LDAPConfig{}, err
	}
	if isEmptySchoolLDAPSettings(settings) {
		return LDAPConfig{}, ErrSchoolLDAPConfigMissing
	}

	return LDAPConfig(settings), nil
}

func validateLDAPConfig(cfg LDAPConfig) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("%w: URL is required", ErrLDAPConfigInvalid)
	}
	if strings.TrimSpace(cfg.BaseDN) == "" {
		return fmt.Errorf("%w: BaseDN is required", ErrLDAPConfigInvalid)
	}
	if strings.TrimSpace(cfg.SystemBindDN) == "" {
		return fmt.Errorf("%w: SystemBindDN is required", ErrLDAPConfigInvalid)
	}
	if cfg.SystemBindPassword == "" {
		return fmt.Errorf("%w: SystemBindPassword is required", ErrLDAPConfigInvalid)
	}
	return nil
}
