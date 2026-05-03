package admission

import "errors"

var (
	ErrAdmissionTokenNotFound            = errors.New("admission token not found")
	ErrAdmissionTokenConsumed            = errors.New("admission token already consumed")
	ErrAdmissionTokenExpired             = errors.New("admission token expired")
	ErrAdmissionQQMismatch               = errors.New("admission qq mismatch")
	ErrAdmissionInvalidStatus            = errors.New("admission session status invalid")
	ErrAdmissionPolicyNotFound           = errors.New("admission policy not found")
	ErrAdmissionLinkedSessionRequired    = errors.New("admission linked session required")
	ErrAdmissionFreshmanChannelClosed    = errors.New("freshman admission channel closed")
	ErrAdmissionFreshmanPendingExists    = errors.New("freshman application pending already exists")
	ErrAdmissionMaterialInvalidType      = errors.New("admission material content type invalid")
	ErrAdmissionMaterialInvalidData      = errors.New("admission material image data invalid")
	ErrAdmissionMaterialTooLarge         = errors.New("admission material too large")
	ErrAdmissionMaterialStoreUnavailable = errors.New("admission material store unavailable")
	ErrAdmissionEmailDomainNotAllowed    = errors.New("admission school email domain not allowed")
	ErrAdmissionEmailSenderUnavailable   = errors.New("admission email sender unavailable")
	ErrAdmissionRedisUnavailable         = errors.New("admission redis unavailable")
	ErrAdmissionOTPExpired               = errors.New("admission email otp expired")
	ErrAdmissionOTPInvalid               = errors.New("admission email otp invalid")
	ErrAdmissionOTPMaxAttempts           = errors.New("admission email otp max attempts exceeded")
	ErrAdmissionOTPCooldown              = errors.New("admission email otp cooldown")
	ErrAdmissionSSONotConfigured         = errors.New("admission school sso not configured")
	ErrAdmissionSSOStateInvalid          = errors.New("admission school sso state invalid")
	ErrAdmissionReturnURLNotAllowed      = errors.New("admission return url not allowed")
)
