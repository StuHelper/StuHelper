package app

import (
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	platformauth "git.stuhelper.com/StuHelper/StuHelper/internal/platform/authorization"
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
