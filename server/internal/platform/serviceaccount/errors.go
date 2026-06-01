package serviceaccount

import "errors"

var (
	ErrCredentialNotConfigured    = errors.New("service account credential is not configured")
	ErrCredentialInvalid          = errors.New("service account credential is invalid")
	ErrCredentialForbidden        = errors.New("service account credential lacks required audience or scope")
	ErrCredentialStoreUnavailable = errors.New("service account credential store unavailable")
)
