package user

import (
	"context"
	"fmt"
	"strings"
)

// GetUserSurface 聚合当前用户的目标学生资格、手机号门槛和能力信息。
// displayName 和 avatarURL 由 auth 中间件注入，capabilities 由角色展开得到。
func (s *Service) GetUserSurface(ctx context.Context, userID int64, displayName, avatarURL string, capabilities []string) (*UserSurface, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if s.verificationStatus == nil {
		return nil, fmt.Errorf("GetUserSurface: verification status gateway is not configured")
	}
	student, err := s.verificationStatus.GetCurrentStudentStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSurface student status: %w", err)
	}
	phone, err := s.verificationStatus.GetCurrentPhoneStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSurface phone status: %w", err)
	}
	studentVerificationStatus := "none"
	if student.Eligible {
		studentVerificationStatus = "approved"
	}

	return &UserSurface{
		DisplayName:               displayName,
		AvatarURL:                 avatarURL,
		Phone:                     phone.MaskedPhone,
		StudentVerificationStatus: studentVerificationStatus,
		PhoneBound:                phone.MaskedPhone != nil,
		Capabilities:              capabilities,
	}, nil
}

// GetProfile 获取学生认证档案
func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil || profile == nil {
		return profile, err
	}
	if err := s.hydrateProfilePhone(profile); err != nil {
		return nil, fmt.Errorf("GetProfile hydrate profile phone: %w", err)
	}
	return profile, nil
}

func (s *Service) hydrateAcademicStudent(student *AcademicStudent) error {
	if student == nil || student.SFZJH != nil || len(student.SFZJHEnc) == 0 {
		return nil
	}

	plaintext, err := s.decryptPIIBytes(student.SFZJHEnc)
	if err != nil {
		return fmt.Errorf("decrypt sfzjh_enc: %w", err)
	}
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil
	}
	student.SFZJH = &plaintext
	return nil
}

func (s *Service) decryptPIIBytes(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}

	return s.docCipher.Decrypt(ciphertext)
}
