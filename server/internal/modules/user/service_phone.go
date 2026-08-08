package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
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
	trimmed := strings.TrimSpace(plaintext)
	mainland := trimmed
	if strings.HasPrefix(mainland, "+86") {
		mainland = strings.TrimPrefix(mainland, "+86")
	} else if strings.HasPrefix(mainland, "86") && len(mainland) == 13 {
		mainland = strings.TrimPrefix(mainland, "86")
	}
	if phoneutil.IsValidMainlandPhone(mainland) {
		masked := phoneutil.Mask(mainland)
		return &masked, nil
	}
	// Migration compatibility: historical ciphertext contained only the
	// masked display value. New writes are owned exclusively by the independent
	// phone domain and contain the full canonical Casdoor projection.
	return normalizeMaskedPhone(&trimmed), nil
}

// BindPhone is retained only as a compile-time compatibility surface for
// callers that have not yet migrated. It deliberately performs no mutation;
// all binding/change/unbind operations must go through the independent phone
// state machine, which creates a verification credential only after Casdoor
// write and authoritative readback.
func (s *Service) BindPhone(_ context.Context, _ int64, _ string) error {
	return ErrLegacyPhoneBindingRemoved
}
