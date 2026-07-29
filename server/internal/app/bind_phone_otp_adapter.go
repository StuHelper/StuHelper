package app

import (
	"context"
	"errors"

	"github.com/StuHelper/StuHelper/server/internal/modules/auth"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

type bindPhoneOTPAdapter struct {
	service *auth.OTPService
}

func newBindPhoneOTPAdapter(service *auth.OTPService) bindPhoneOTPAdapter {
	return bindPhoneOTPAdapter{service: service}
}

func (a bindPhoneOTPAdapter) IssueCode(ctx context.Context, phone string, smsSender user.SMSSender) error {
	return normalizeBindPhoneOTPError(a.service.IssueCode(ctx, phone, smsSender))
}

func (a bindPhoneOTPAdapter) CooldownSeconds() int {
	return a.service.CooldownSeconds()
}

func (a bindPhoneOTPAdapter) Check(ctx context.Context, phone, code string) error {
	return normalizeBindPhoneOTPError(a.service.Check(ctx, phone, code))
}

func (a bindPhoneOTPAdapter) Consume(ctx context.Context, phone, code string) error {
	return normalizeBindPhoneOTPError(a.service.Consume(ctx, phone, code))
}

func normalizeBindPhoneOTPError(err error) error {
	switch {
	case errors.Is(err, auth.ErrOTPPhoneRateLimited):
		return user.ErrBindPhoneOTPPhoneRateLimited
	case errors.Is(err, auth.ErrOTPCooldown):
		return user.ErrBindPhoneOTPCooldown
	case errors.Is(err, auth.ErrOTPExpired):
		return user.ErrBindPhoneOTPExpired
	case errors.Is(err, auth.ErrOTPMaxAttempts):
		return user.ErrBindPhoneOTPMaxAttempts
	case errors.Is(err, auth.ErrOTPInvalidCode):
		return user.ErrBindPhoneOTPInvalidCode
	default:
		return err
	}
}
