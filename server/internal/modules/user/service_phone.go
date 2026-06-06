package user

import (
	"context"
	"fmt"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

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

func (s *Service) buildPhoneProjection(phone string) (string, []byte, string, error) {
	trimmed := strings.TrimSpace(phone)
	masked := phoneutil.Mask(trimmed)
	phoneEnc, err := s.docCipher.Encrypt(masked)
	if err != nil {
		return "", nil, "", fmt.Errorf("encrypt phone: %w", err)
	}
	phoneHash, err := phoneutil.HashLookupWithKey(trimmed, s.hmacKey)
	if err != nil {
		return "", nil, "", fmt.Errorf("hash phone projection: %w", err)
	}
	return masked, phoneEnc, phoneHash, nil
}

func (s *Service) prepareAvailablePhoneProjection(ctx context.Context, userID int64, phone string) (string, []byte, string, error) {
	if err := validateUserID(userID); err != nil {
		return "", nil, "", err
	}
	masked, phoneEnc, phoneHash, err := s.buildPhoneProjection(phone)
	if err != nil {
		return "", nil, "", err
	}
	if err := s.repo.EnsureUserPhoneAvailable(ctx, userID, phoneHash); err != nil {
		return "", nil, "", err
	}
	return masked, phoneEnc, phoneHash, nil
}

func (s *Service) hydrateProfilePhone(profile *Profile) error {
	if profile == nil {
		return nil
	}

	profile.Phone = normalizeMaskedPhone(profile.Phone)
	if len(profile.PhoneEnc) == 0 {
		profile.PhoneVerified = profile.Phone != nil
		return nil
	}

	phone, err := s.phoneProjectionFromCiphertext(profile.PhoneEnc)
	if err != nil {
		return err
	}

	profile.Phone = phone
	profile.PhoneVerified = profile.Phone != nil
	return nil
}

func (s *Service) phoneProjectionFromCiphertext(phoneEnc []byte) (*string, error) {
	if len(phoneEnc) == 0 {
		return nil, nil
	}
	plaintext, err := s.decryptPIIBytes(phoneEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt phone_enc: %w", err)
	}
	return normalizeMaskedPhone(&plaintext), nil
}

func (s *Service) getAccountPhoneProjection(ctx context.Context, userID int64, profile *Profile) (*string, error) {
	if profile != nil {
		if err := s.hydrateProfilePhone(profile); err != nil {
			return nil, err
		}
		if profile.Phone != nil {
			return profile.Phone, nil
		}
	}

	phoneEnc, err := s.repo.GetUserPhoneProjection(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user phone projection: %w", err)
	}
	return s.phoneProjectionFromCiphertext(phoneEnc)
}

// BindPhone 绑定手机号。
// Casdoor 是手机号真相源；StuHelper 只在本地 users 上保存 masked projection，
// 用于业务页面快速判断是否已绑定。
func (s *Service) BindPhone(ctx context.Context, userID int64, phone string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(phone)
	if !phoneutil.IsValidMainlandPhone(trimmed) {
		return ErrInvalidPhoneFormat
	}
	if s.profileIdentitySync == nil {
		return ErrProfileIdentitySyncMissing
	}

	_, phoneEnc, phoneHash, err := s.prepareAvailablePhoneProjection(ctx, userID, trimmed)
	if err != nil {
		return fmt.Errorf("BindPhone check phone projection: %w", err)
	}

	subject, err := s.repo.GetCasdoorSubject(ctx, userID)
	if err != nil {
		return fmt.Errorf("BindPhone get Casdoor subject: %w", err)
	}
	if err := s.profileIdentitySync.UpdatePhone(ctx, subject, "+86"+trimmed); err != nil {
		return fmt.Errorf("BindPhone update Casdoor phone: %w", err)
	}

	if err := s.repo.SetUserPhone(ctx, userID, phoneEnc, phoneHash); err != nil {
		return fmt.Errorf("BindPhone update phone projection: %w", err)
	}
	return nil
}
