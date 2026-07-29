package rbac

import "github.com/StuHelper/StuHelper/server/internal/platform/authorization"

func SetDefaultAuthorizer(authorizer authorization.AuthorizationService) {
	if authorizer == nil {
		defaultAuthorizer = authorization.NewService()
		return
	}
	defaultAuthorizer = authorizer
}
