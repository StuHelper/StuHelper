package user

import "context"

// OTPGenerator generates and verifies OTP codes for phone binding.
type OTPGenerator interface {
	Generate(ctx context.Context, phone string) (code string, err error)
	Verify(ctx context.Context, phone, code string) error
	Cleanup(ctx context.Context, phone string) error
}

// SMSSender sends SMS messages (e.g. OTP codes) to a phone number.
type SMSSender interface {
	Send(ctx context.Context, phone, content string) error
}
