package serviceaccount

import "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/botcredential"

var (
	ErrCredentialNotConfigured    = botcredential.ErrCredentialNotConfigured
	ErrCredentialInvalid          = botcredential.ErrCredentialInvalid
	ErrCredentialForbidden        = botcredential.ErrCredentialForbidden
	ErrCredentialStoreUnavailable = botcredential.ErrCredentialStoreUnavailable
)
