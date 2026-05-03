package admission

import "errors"

var (
	ErrAdmissionTokenNotFound  = errors.New("admission token not found")
	ErrAdmissionTokenConsumed  = errors.New("admission token already consumed")
	ErrAdmissionTokenExpired   = errors.New("admission token expired")
	ErrAdmissionQQMismatch     = errors.New("admission qq mismatch")
	ErrAdmissionInvalidStatus  = errors.New("admission session status invalid")
	ErrAdmissionPolicyNotFound = errors.New("admission policy not found")
)
