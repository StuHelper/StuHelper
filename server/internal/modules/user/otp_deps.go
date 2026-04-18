package user

import (
	"context"

	authmodule "git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
)

// OTPGenerator generates and verifies OTP codes for phone binding.
type OTPGenerator interface {
	IssueCode(ctx context.Context, phone string, smsSender authmodule.PhoneSMSSender) error
	CooldownSeconds() int
	Verify(ctx context.Context, phone, code string) error
}

// SMSSender sends SMS messages (e.g. OTP codes) to a phone number.
type SMSSender = authmodule.PhoneSMSSender
