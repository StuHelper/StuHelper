package openplatform

import "errors"

var (
	ErrAppNotFound           = errors.New("open platform app not found")
	ErrAppNotApproved        = errors.New("open platform app is not approved")
	ErrAppNotActive          = errors.New("open platform app is not active")
	ErrInvalidScope          = errors.New("open platform scope is invalid")
	ErrScopeNotApproved      = errors.New("open platform scope is not approved")
	ErrConsentRequired       = errors.New("open platform consent is required")
	ErrConsentTokenInvalid   = errors.New("open platform consent token is invalid")
	ErrRedirectURINotAllowed = errors.New("open platform redirect URI is not allowed")
	ErrDisclosureUnavailable = errors.New("open platform disclosure unavailable")
)
