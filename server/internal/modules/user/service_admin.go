package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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
		profile.VerifiedAt = &now
	} else {
		profile.VerificationStatus = StatusRejected
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
