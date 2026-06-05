package user

import (
	"context"

	authmodule "git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
)

var (
	errBindPhoneOTPCooldown         = authmodule.ErrOTPCooldown
	errBindPhoneOTPExpired          = authmodule.ErrOTPExpired
	errBindPhoneOTPInvalidCode      = authmodule.ErrOTPInvalidCode
	errBindPhoneOTPMaxAttempts      = authmodule.ErrOTPMaxAttempts
	errBindPhoneOTPPhoneRateLimited = authmodule.ErrOTPPhoneRateLimited
)

// OTPGenerator 负责生成并校验手机绑定所需的 OTP。
type OTPGenerator interface {
	IssueCode(ctx context.Context, phone string, smsSender authmodule.PhoneSMSSender) error
	CooldownSeconds() int
	Verify(ctx context.Context, phone, code string) error
}

// SMSSender 负责向手机号发送短信，例如 OTP 验证码。
type SMSSender = authmodule.PhoneSMSSender
