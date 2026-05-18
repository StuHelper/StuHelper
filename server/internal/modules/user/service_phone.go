package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appcrypto "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

const phoneProjectionHashScope = "phone_projection:"

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

func (s *Service) encryptPhoneProjection(userID int64, phone string) ([]byte, string, error) {
	phoneEnc, err := s.docCipher.Encrypt(phone)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt phone: %w", err)
	}
	phoneHash, err := appcrypto.HMACHashWithKey(phoneProjectionHashScope+strconv.FormatInt(userID, 10), s.hmacKey)
	if err != nil {
		return nil, "", fmt.Errorf("hash phone projection: %w", err)
	}
	return phoneEnc, phoneHash, nil
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

	plaintext, err := s.decryptPIIBytes(profile.PhoneEnc)
	if err != nil {
		return fmt.Errorf("decrypt phone_enc: %w", err)
	}

	profile.Phone = normalizeMaskedPhone(&plaintext)
	profile.PhoneVerified = profile.Phone != nil
	return nil
}

// BindPhone 绑定手机号。
// Casdoor 是手机号真相源；StuHelper 只在本地 profile 上保存 masked projection，
// 用于业务页面快速判断是否已绑定。
func (s *Service) BindPhone(ctx context.Context, userID int64, phone string) error {
	trimmed := strings.TrimSpace(phone)
	if !phoneutil.IsValidMainlandPhone(trimmed) {
		return ErrInvalidPhoneFormat
	}
	if s.profileIdentitySync == nil {
		return ErrProfileIdentitySyncMissing
	}

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("BindPhone get profile: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	subject, err := s.repo.GetCasdoorSubject(ctx, userID)
	if err != nil {
		return fmt.Errorf("BindPhone get Casdoor subject: %w", err)
	}
	if err := s.profileIdentitySync.UpdatePhone(ctx, subject, "+86"+trimmed); err != nil {
		return fmt.Errorf("BindPhone update Casdoor phone: %w", err)
	}

	masked := phoneutil.Mask(trimmed)
	profile.Phone = &masked
	profile.PhoneVerified = true
	phoneEnc, phoneHash, err := s.encryptPhoneProjection(userID, masked)
	if err != nil {
		return err
	}
	if err := s.repo.SetUserPhone(ctx, userID, phoneEnc, phoneHash); err != nil {
		return fmt.Errorf("BindPhone update phone projection: %w", err)
	}
	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return fmt.Errorf("BindPhone update profile projection: %w", err)
	}
	return nil
}
