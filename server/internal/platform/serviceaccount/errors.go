package serviceaccount

import "github.com/StuHelper/StuHelper/server/internal/pkg/botcredential"

var (
	ErrCredentialNotConfigured    = botcredential.ErrCredentialNotConfigured
	ErrCredentialInvalid          = botcredential.ErrCredentialInvalid
	ErrCredentialForbidden        = botcredential.ErrCredentialForbidden
	ErrCredentialStoreUnavailable = botcredential.ErrCredentialStoreUnavailable
)
