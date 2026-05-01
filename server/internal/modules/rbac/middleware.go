package rbac

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/authorization"
)

var defaultAuthorizer authorization.AuthorizationService = authorization.NewService()

// RequireCapability 检查当前用户是否持有指定能力。
// 能力入口统一委托 Authorization Service，保持业务 PDP 单一。
func RequireCapability(capName string) gin.HandlerFunc {
	return RequireCapabilityWithAuthorizer(defaultAuthorizer, capName)
}

func RequireCapabilityWithAuthorizer(authorizer authorization.AuthorizationService, capName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision := authorizer.Authorize(
			c.Request.Context(),
			authorization.SubjectFromGin(c),
			authorization.ActionCapabilityRequire,
			authorization.CapabilityResource(capName),
		)
		if abortOnDeny(c, decision) {
			return
		}
		c.Next()
	}
}

// RequireAnyCapability 检查当前用户是否持有任一指定能力。
func RequireAnyCapability(capNames ...string) gin.HandlerFunc {
	return RequireAnyCapabilityWithAuthorizer(defaultAuthorizer, capNames...)
}

func RequireAnyCapabilityWithAuthorizer(authorizer authorization.AuthorizationService, capNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision := authorizer.Authorize(
			c.Request.Context(),
			authorization.SubjectFromGin(c),
			authorization.ActionCapabilityRequireAny,
			authorization.AnyCapabilityResource(capNames...),
		)
		if abortOnDeny(c, decision) {
			return
		}
		c.Next()
	}
}

// RequireGlobalCapability 要求当前用户持有全局能力授权。
// 作用域能力（例如 school_admin 的 school-scoped grant）不会通过此检查。
func RequireGlobalCapability(capName string) gin.HandlerFunc {
	return RequireGlobalCapabilityWithAuthorizer(defaultAuthorizer, capName)
}

func RequireGlobalCapabilityWithAuthorizer(authorizer authorization.AuthorizationService, capName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision := authorizer.Authorize(
			c.Request.Context(),
			authorization.SubjectFromGin(c),
			authorization.ActionCapabilityRequireGlobal,
			authorization.GlobalCapabilityResource(capName),
		)
		if abortOnDeny(c, decision) {
			return
		}
		c.Next()
	}
}

func RequirePrivilegedMFA() gin.HandlerFunc {
	return RequirePrivilegedMFAWithAuthorizer(defaultAuthorizer)
}

func RequirePrivilegedMFAWithAuthorizer(authorizer authorization.AuthorizationService) gin.HandlerFunc {
	return requireMFAWithAuthorizer(
		authorizer,
		authorization.ActionPrivilegedMFARequire,
		authorization.PrivilegedMFAResource(0),
	)
}

func RequireStepUpMFA() gin.HandlerFunc {
	return RequireStepUpMFAWithAuthorizer(defaultAuthorizer)
}

func RequireStepUpMFAWithAuthorizer(authorizer authorization.AuthorizationService) gin.HandlerFunc {
	return requireMFAWithAuthorizer(
		authorizer,
		authorization.ActionStepUpMFARequire,
		authorization.StepUpMFAResource(0),
	)
}

func requireMFAWithAuthorizer(
	authorizer authorization.AuthorizationService,
	action authorization.Action,
	resource authorization.Resource,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision := authorizer.Authorize(c.Request.Context(), authorization.SubjectFromGin(c), action, resource)
		if abortOnDeny(c, decision) {
			return
		}
		c.Next()
	}
}

func abortOnDeny(c *gin.Context, decision authorization.Decision) bool {
	if decision.Allow {
		return false
	}
	switch {
	case errors.Is(decision.Error, authorization.ErrMFAEnrollmentRequired):
		response.Forbidden(c, "mfa enrollment required", errs.ErrMFARequired)
	case errors.Is(decision.Error, authorization.ErrStepUpRequired):
		response.Error(c, http.StatusPreconditionRequired, errs.ErrStepUpRequired, "step-up required")
	case decision.Error != nil:
		response.ServiceUnavailable(c, "authorization service temporarily unavailable", errs.ErrServiceUnavailable)
	default:
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
	}
	c.Abort()
	return true
}
