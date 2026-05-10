package rbac

import "git.stuhelper.com/StuHelper/StuHelper/internal/platform/authorization"

func SetDefaultAuthorizer(authorizer authorization.AuthorizationService) {
	if authorizer == nil {
		defaultAuthorizer = authorization.NewService()
		return
	}
	defaultAuthorizer = authorizer
}
