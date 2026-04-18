package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

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

func (s *Service) encryptPhone(phone string) ([]byte, string, error) {
	phoneEnc, err := s.docCipher.Encrypt(phone)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt phone: %w", err)
	}
	phoneHash, err := phoneutil.HashLookupWithKey(phone, s.hmacKey)
	if err != nil {
		return nil, "", fmt.Errorf("hash phone: %w", err)
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

	phoneEnc, phoneHash, err := s.encryptPhone(trimmed)
	if err != nil {
		return err
	}

	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txProfile, err := s.repo.GetProfileByUserIDTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("BindPhone get profile tx: %w", err)
		}
		if txProfile == nil {
			return ErrProfileNotFound
		}
		masked := phoneutil.Mask(trimmed)
		txProfile.Phone = &masked
		txProfile.PhoneVerified = true
		if err := s.repo.SetUserPhoneTx(ctx, tx, userID, phoneEnc, phoneHash); err != nil {
			return fmt.Errorf("BindPhone set phone tx: %w", err)
		}
		if err := s.repo.UpdateProfileTx(ctx, tx, txProfile); err != nil {
			return fmt.Errorf("BindPhone update profile tx: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrPhoneAlreadyBound) {
			return ErrPhoneAlreadyBound
		}
		return err
	}
	return nil
}
