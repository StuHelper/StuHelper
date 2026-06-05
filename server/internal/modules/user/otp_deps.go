package user

import (
	"context"
	"errors"
)

var (
	ErrBindPhoneOTPCooldown         = errors.New("bind phone otp cooldown")
	ErrBindPhoneOTPExpired          = errors.New("bind phone otp expired")
	ErrBindPhoneOTPInvalidCode      = errors.New("bind phone otp invalid code")
	ErrBindPhoneOTPMaxAttempts      = errors.New("bind phone otp max attempts exceeded")
	ErrBindPhoneOTPPhoneRateLimited = errors.New("bind phone otp phone rate limited")
)

// OTPGenerator 负责生成并校验手机绑定所需的 OTP。
type OTPGenerator interface {
	IssueCode(ctx context.Context, phone string, smsSender SMSSender) error
	CooldownSeconds() int
	Verify(ctx context.Context, phone, code string) error
}

// SMSSender 负责向手机号发送短信，例如 OTP 验证码。
type SMSSender interface {
	Send(ctx context.Context, phone, content string) error
}
