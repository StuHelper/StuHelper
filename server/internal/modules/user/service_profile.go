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

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// GetProfile 获取学生认证档案
func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	return s.repo.GetProfileByUserID(ctx, userID)
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

	if school.VerificationMethod == VerifyMethodLDAP {
		if s.ldapClient == nil {
			return nil, fmt.Errorf("VerifyStudent: LDAP client not configured")
		}

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

		ldapInfo, err := s.ldapClient.QueryUserByUID(ctx, trimmedStudentID)
		if err != nil {
			logger.L().Warn("LDAP query user info failed after successful login",
				zap.Int64("user_id", userID),
				zap.String("student_id", trimmedStudentID),
				zap.Error(err),
			)
		}

		studentIDs := []string{trimmedStudentID}
		academicStudent, err := s.repo.GetAcademicStudentByXH(ctx, trimmedStudentID)
		if err != nil {
			logger.L().Warn("failed to query academic student",
				zap.String("student_id", trimmedStudentID),
				zap.Error(err),
			)
		}

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

		if ldapInfo != nil && ldapInfo.Mobile != "" {
			phone := ldapInfo.Mobile
			profile.Phone = &phone
			profile.PhoneVerified = true
		}
	} else {
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
	profile.PhoneVerified = false

	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return fmt.Errorf("BindPhone update: %w", err)
	}
	return nil
}

// GetAcademicInfo 获取学籍信息
func (s *Service) GetAcademicInfo(ctx context.Context, studentID string) (*AcademicStudent, error) {
	return s.repo.GetAcademicStudentByXH(ctx, studentID)
}

// ListSchools 获取所有启用的学校列表
func (s *Service) ListSchools(ctx context.Context) ([]SchoolConfig, error) {
	return s.repo.ListSchoolConfigs(ctx)
}
