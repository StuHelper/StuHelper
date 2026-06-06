package user

import (
	"context"
	"fmt"
	"strings"
)

// GetUserSurface 聚合当前用户的身份认证、学生认证、手机绑定和能力信息。
// displayName 和 avatarURL 由 auth 中间件注入，capabilities 由角色展开得到。
func (s *Service) GetUserSurface(ctx context.Context, userID int64, displayName, avatarURL string, capabilities []string) (*UserSurface, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	identity, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSurface identity: %w", err)
	}

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSurface profile: %w", err)
	}
	if err := s.hydrateProfilePhone(profile); err != nil {
		return nil, fmt.Errorf("GetUserSurface hydrate profile phone: %w", err)
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
