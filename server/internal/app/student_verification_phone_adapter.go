package app

import (
	"context"
	"errors"

	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

type studentVerificationPhoneOTPAdapter struct {
	otp user.OTPGenerator
	sms user.SMSSender
}

func newStudentVerificationPhoneOTPAdapter(
	otp user.OTPGenerator,
	sms user.SMSSender,
) *studentVerificationPhoneOTPAdapter {
	if otp == nil || sms == nil {
		return nil
	}
	return &studentVerificationPhoneOTPAdapter{otp: otp, sms: sms}
}

func (a *studentVerificationPhoneOTPAdapter) Issue(ctx context.Context, phone string) error {
	return normalizeStudentVerificationPhoneOTPError(a.otp.IssueCode(ctx, phone, a.sms))
}

func (a *studentVerificationPhoneOTPAdapter) Check(ctx context.Context, phone, code string) error {
	return normalizeStudentVerificationPhoneOTPError(a.otp.Check(ctx, phone, code))
}

func (a *studentVerificationPhoneOTPAdapter) Consume(ctx context.Context, phone, code string) error {
	return normalizeStudentVerificationPhoneOTPError(a.otp.Consume(ctx, phone, code))
}

func (a *studentVerificationPhoneOTPAdapter) CooldownSeconds() int {
	return a.otp.CooldownSeconds()
}

func normalizeStudentVerificationPhoneOTPError(err error) error {
	switch {
	case errors.Is(err, user.ErrBindPhoneOTPCooldown):
		return studentverification.ErrPhoneOTPCooldown
	case errors.Is(err, user.ErrBindPhoneOTPExpired):
		return studentverification.ErrPhoneOTPExpired
	case errors.Is(err, user.ErrBindPhoneOTPInvalidCode):
		return studentverification.ErrPhoneOTPInvalid
	case errors.Is(err, user.ErrBindPhoneOTPMaxAttempts):
		return studentverification.ErrPhoneOTPMaxAttempts
	case errors.Is(err, user.ErrBindPhoneOTPPhoneRateLimited):
		return studentverification.ErrPhoneOTPRateLimited
	default:
		return err
	}
}
