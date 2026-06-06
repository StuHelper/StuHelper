package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

const maxSchoolConfigNameRunes = 100

const (
	approvalPolicyAuto   = "auto"
	approvalPolicyManual = "manual"
)

// GetInternalUserID 根据外部ID获取内部用户ID
func (s *Service) GetInternalUserID(ctx context.Context, casdoorSubject string) (int64, error) {
	return s.repo.GetInternalUserID(ctx, casdoorSubject)
}

// ListIdentities 分页查询实名认证审核列表（管理端，不含敏感字段）
func (s *Service) ListIdentities(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error) {
	return s.repo.ListIdentityReviewItems(ctx, status, page, pageSize)
}

// ReviewIdentity 管理员审核实名认证（通过/驳回）
// 使用精准更新，不读取也不回写敏感字段
func (s *Service) ReviewIdentity(ctx context.Context, userID int64, approved bool, reason string) error {
	identityStatus, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ReviewIdentity get: %w", err)
	}
	if identityStatus == nil {
		return ErrIdentityNotFound
	}

	var (
		verifyMethod    *string
		reviewedAt      *time.Time
		verifiedAt      *time.Time
		rejectionReason *string
	)

	trimmedReason := strings.TrimSpace(reason)
	now := time.Now()
	reviewedAt = &now
	if approved {
		method := VerifyMethodManual
		verifyMethod = &method
		verifiedAt = &now
		// 显式清除旧拒绝原因：批准后 rejection_reason 必须为 NULL
		rejectionReason = nil
	} else {
		if trimmedReason != "" {
			rejectionReason = &trimmedReason
		} else {
			rejectionReason = nil
		}
		// 显式清除：拒绝时 verify_method 和 verified_at 不应保留
		verifyMethod = nil
		verifiedAt = nil
	}

	return s.repo.UpdateIdentityReviewStatus(ctx, userID, approved, verifyMethod, reviewedAt, verifiedAt, rejectionReason)
}

// ListProfiles 分页查询学生认证档案（管理端）
func (s *Service) ListProfiles(ctx context.Context, status string, schoolID *int64, page, pageSize int) ([]Profile, int, error) {
	list, total, err := s.repo.ListProfilesByStatus(ctx, status, schoolID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		if err := s.hydrateProfilePhone(&list[i]); err != nil {
			return nil, 0, fmt.Errorf("ListProfiles hydrate profile phone: %w", err)
		}
	}
	return list, total, nil
}

func (s *Service) GetProfileSchoolID(ctx context.Context, userID int64) (*int64, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetProfileSchoolID get: %w", err)
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile.SchoolID, nil
}

// ReviewStudentVerification 管理员审核学生认证（通过/驳回）
func (s *Service) ReviewStudentVerification(ctx context.Context, userID int64, approved bool, reason string) error {
	trimmedReason := strings.TrimSpace(reason)

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ReviewStudentVerification get: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	now := time.Now()
	profile.ReviewedAt = &now
	if approved {
		profile.VerificationStatus = StatusVerified
		profile.VerifiedAt = &now
		profile.RejectionReason = nil
	} else {
		profile.VerificationStatus = StatusRejected
		profile.VerifiedAt = nil
		if trimmedReason != "" {
			profile.RejectionReason = &trimmedReason
		} else {
			profile.RejectionReason = nil
		}
	}

	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.UpdateProfileTx(ctx, tx, profile); err != nil {
			return fmt.Errorf("ReviewStudentVerification update profile tx: %w", err)
		}
		if err := s.ensureProfileVerificationCredentialTx(ctx, tx, profile); err != nil {
			return fmt.Errorf("ReviewStudentVerification ensure credential: %w", err)
		}
		if err := s.enqueueVerificationProjectionTx(ctx, tx, userID, profile.VerificationStatus); err != nil {
			return fmt.Errorf("ReviewStudentVerification enqueue projections: %w", err)
		}
		return nil
	})
}

// ListAllSchoolConfigs 获取所有学校配置（含禁用，管理端用）
func (s *Service) ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListAllSchoolConfigs(ctx)
}

// UpdateSchoolConfig 更新学校认证配置
// 使用合并更新语义，保持未提供字段的现有值。
func (s *Service) UpdateSchoolConfig(ctx context.Context, schoolID int64, input UpdateSchoolConfigInput) error {
	config, err := s.repo.GetSchoolConfig(ctx, schoolID)
	if err != nil {
		return fmt.Errorf("UpdateSchoolConfig get existing: %w", err)
	}
	if config == nil {
		return ErrSchoolNotFound
	}

	if input.SchoolName != nil {
		normalizedName, err := normalizeSchoolConfigName(*input.SchoolName)
		if err != nil {
			return err
		}
		config.SchoolName = normalizedName
	}
	if input.VerificationMethod != nil {
		normalizedMethod, err := normalizeSchoolVerificationMethod(*input.VerificationMethod)
		if err != nil {
			return err
		}
		config.VerificationMethod = normalizedMethod
	}
	if input.ApprovalPolicy != nil {
		normalizedPolicy, err := normalizeSchoolApprovalPolicy(*input.ApprovalPolicy)
		if err != nil {
			return err
		}
		config.ApprovalPolicy = normalizedPolicy
	}
	if input.AcademicDBTable != nil {
		normalizedAcademicDBTable, err := normalizeConfiguredAcademicDBTable(input.AcademicDBTable)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAcademicDBTable, err)
		}
		config.AcademicDBTable = normalizedAcademicDBTable
	}
	if input.ConsentText != nil {
		value := strings.TrimSpace(*input.ConsentText)
		config.ConsentText = &value
	}
	if input.Enabled != nil {
		config.Enabled = *input.Enabled
	}
	if input.LDAPConfig != nil {
		raw, err := mergeSchoolLDAPConfig(config.LDAPConfig, input.LDAPConfig)
		if err != nil {
			return fmt.Errorf("UpdateSchoolConfig merge ldapConfig: %w", err)
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

	if err := s.validateSchoolConfig(ctx, config); err != nil {
		return err
	}

	if err := s.repo.UpdateSchoolConfig(ctx, config); err != nil {
		return err
	}

	systemconfig.InvalidateReviewAccessPolicySnapshot()
	return nil
}

func mergeSchoolLDAPConfig(existing json.RawMessage, input *SchoolLDAPConfigInput) (json.RawMessage, error) {
	if input == nil {
		return existing, nil
	}

	settings, err := decodeSchoolLDAPSettings(existing)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLDAPConfigInvalid, err)
	}

	if input.URL != nil {
		settings.URL = strings.TrimSpace(*input.URL)
	}
	if input.BaseDN != nil {
		settings.BaseDN = strings.TrimSpace(*input.BaseDN)
	}
	if input.SystemBindDN != nil {
		settings.SystemBindDN = strings.TrimSpace(*input.SystemBindDN)
	}
	if input.SystemBindPassword != nil {
		settings.SystemBindPassword = strings.TrimSpace(*input.SystemBindPassword)
	}
	if input.UseTLS != nil {
		settings.UseTLS = *input.UseTLS
	}

	settings = normalizeSchoolLDAPSettings(settings)
	if isEmptySchoolLDAPSettings(settings) {
		return nil, nil
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) validateSchoolConfig(ctx context.Context, config *SchoolConfig) error {
	if config == nil {
		return ErrSchoolNotFound
	}
	if err := validateSchoolConfigScalarFields(config); err != nil {
		return err
	}

	if config.AcademicDBTable != nil {
		normalizedTable, err := normalizeAcademicDBTableName(config.AcademicDBTable)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAcademicDBTable, err)
		}
		if err := s.repo.ValidateAcademicDBTable(ctx, normalizedTable); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAcademicDBTable, err)
		}
	}

	if !config.Enabled || config.VerificationMethod != VerifyMethodLDAP {
		return nil
	}

	if _, err := s.ensureAcademicTableConfigured(config); err != nil {
		return err
	}
	if err := validateSchoolLDAPConfig(config.LDAPConfig); err != nil {
		return err
	}
	return nil
}

func validateSchoolConfigScalarFields(config *SchoolConfig) error {
	normalizedName, err := normalizeSchoolConfigName(config.SchoolName)
	if err != nil {
		return err
	}
	config.SchoolName = normalizedName

	normalizedMethod, err := normalizeSchoolVerificationMethod(config.VerificationMethod)
	if err != nil {
		if strings.TrimSpace(config.VerificationMethod) != "" {
			return err
		}
		normalizedMethod = VerifyMethodManual
	}
	config.VerificationMethod = normalizedMethod

	normalizedPolicy, err := normalizeSchoolApprovalPolicy(config.ApprovalPolicy)
	if err != nil {
		if strings.TrimSpace(config.ApprovalPolicy) != "" {
			return err
		}
		normalizedPolicy = approvalPolicyAuto
	}
	config.ApprovalPolicy = normalizedPolicy
	return nil
}

func normalizeSchoolConfigName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: schoolName is required", ErrInvalidSchoolConfigValue)
	}
	if utf8.RuneCountInString(name) > maxSchoolConfigNameRunes {
		return "", fmt.Errorf("%w: schoolName exceeds maximum length of %d", ErrInvalidSchoolConfigValue, maxSchoolConfigNameRunes)
	}
	return name, nil
}

func normalizeSchoolVerificationMethod(method string) (string, error) {
	method = strings.TrimSpace(method)
	switch method {
	case VerifyMethodLDAP, VerifyMethodManual:
		return method, nil
	default:
		return "", fmt.Errorf("%w: verificationMethod must be one of: ldap, manual", ErrInvalidSchoolConfigValue)
	}
}

func normalizeSchoolApprovalPolicy(policy string) (string, error) {
	policy = strings.TrimSpace(policy)
	switch policy {
	case approvalPolicyAuto, approvalPolicyManual:
		return policy, nil
	default:
		return "", fmt.Errorf("%w: approvalPolicy must be one of: auto, manual", ErrInvalidSchoolConfigValue)
	}
}

func validateSchoolLDAPConfig(raw json.RawMessage) error {
	ldapCfg, err := parseSchoolLDAPConfig(raw)
	if err != nil {
		return err
	}
	return validateLDAPConfig(ldapCfg)
}

// ListSystemConfigs 获取所有系统配置项
func (s *Service) ListSystemConfigs(ctx context.Context) ([]SystemConfig, error) {
	return s.repo.ListSystemConfigs(ctx)
}

// UpdateSystemConfig 更新系统配置项
func (s *Service) UpdateSystemConfig(ctx context.Context, key, value string) error {
	if err := s.validateSystemConfigValue(ctx, key, value); err != nil {
		return err
	}

	if err := s.repo.UpdateSystemConfig(ctx, key, value); err != nil {
		return err
	}

	if systemconfig.AffectsAuthTokenPolicy(key) {
		if err := applyAuthTokenPolicySnapshotValue(value); err != nil {
			return err
		}
	}
	if systemconfig.AffectsReviewAccessPolicy(key) {
		systemconfig.InvalidateReviewAccessPolicySnapshot()
	}
	return nil
}

func (s *Service) validateSystemConfigValue(ctx context.Context, key, value string) error {
	if len(value) > 10240 {
		return fmt.Errorf("%w: %s value exceeds maximum length of 10KB", ErrInvalidSystemConfigValue, key)
	}
	switch key {
	case systemconfig.AuthAccessTokenTTLSecondsKey:
		if _, err := systemconfig.ParseAuthAccessTokenTTL(value); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
	case systemconfig.ReviewAccessSchoolIDsKey:
		schoolIDs, err := systemconfig.ParseStringList(value)
		if err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
		if err := s.validateReviewAccessSchoolIDs(ctx, schoolIDs); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
	case systemconfig.ReviewPreviewTitleCharsKey:
		if _, err := systemconfig.ParseBoundedInt(value, 1, 200); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
	case systemconfig.ReviewPreviewContentCharsKey:
		if _, err := systemconfig.ParseBoundedInt(value, 1, 5000); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
	case systemconfig.ReviewPreviewContentPercentKey:
		if _, err := systemconfig.ParseBoundedInt(value, 1, 100); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
	case systemconfig.EmailDeliveryPolicyKey:
		if err := systemconfig.ValidateEmailDeliveryPolicy(value); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalidSystemConfigValue, key, err)
		}
	}
	return nil
}

func (s *Service) validateReviewAccessSchoolIDs(ctx context.Context, schoolIDs []string) error {
	if len(schoolIDs) == 0 {
		return nil
	}

	schools, err := s.repo.ListAllSchoolConfigs(ctx)
	if err != nil {
		return fmt.Errorf("load school configs: %w", err)
	}

	allowed := make(map[string]struct{}, len(schools))
	for _, school := range schools {
		allowed[strconv.FormatInt(school.SchoolID, 10)] = struct{}{}
	}

	invalid := make([]string, 0)
	for _, schoolID := range schoolIDs {
		if _, ok := allowed[schoolID]; ok {
			continue
		}
		invalid = append(invalid, schoolID)
	}
	if len(invalid) > 0 {
		return fmt.Errorf("unknown school IDs: %s", strings.Join(invalid, ", "))
	}
	return nil
}
