package app

import (
	"github.com/StuHelper/StuHelper/server/internal/modules/rbac"
	platformauth "github.com/StuHelper/StuHelper/server/internal/platform/authorization"
)

func configureRBACAuthorizer(appEnv string) {
	rbac.SetDefaultAuthorizer(newRBACAuthorizer(appEnv))
}

func newRBACAuthorizer(appEnv string) platformauth.AuthorizationService {
	if appEnv == "development" {
		return platformauth.NewService(platformauth.WithMFAGatesDisabled())
	}
	return platformauth.NewService()
}
