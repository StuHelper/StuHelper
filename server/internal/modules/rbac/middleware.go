package rbac

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/platform/authorization"
)

var defaultAuthorizer authorization.AuthorizationService = authorization.NewService()

const (
	mfaProofAuditType  audit.EventType = "iam.mfa.proof"
	mfaStepUpAuditType audit.EventType = "iam.mfa.step_up"
)

// RequireCapability 检查当前用户是否持有指定能力。
// 能力入口统一委托 Authorization Service，保持业务 PDP 单一。
func RequireCapability(capName string) gin.HandlerFunc {
	return RequireCapabilityWithAuthorizer(defaultAuthorizer, capName)
}

func RequireCapabilityWithAuthorizer(authorizer authorization.AuthorizationService, capName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision := authorizer.Authorize(
			c.Request.Context(),
			subjectFromGin(c),
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
			subjectFromGin(c),
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
			subjectFromGin(c),
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

// RequireMFAProof requires an active enrollment and proof that the current
// authentication session used MFA. It intentionally does not impose the
// short freshness window reserved for sensitive operations.
func RequireMFAProof() gin.HandlerFunc {
	return RequireMFAProofWithAuthorizer(defaultAuthorizer)
}

func RequireMFAProofWithAuthorizer(authorizer authorization.AuthorizationService) gin.HandlerFunc {
	return requireMFAWithAuthorizer(authorizer, authorization.ActionMFAProofRequire, authorization.MFAProofResource())
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
	return requireMFAWithAuthorizer(authorizer, authorization.ActionStepUpMFARequire, authorization.StepUpMFAResource(0))
}

func EnsureStepUpMFA(c *gin.Context) bool {
	return EnsureStepUpMFAWithAuthorizer(defaultAuthorizer, c)
}

func EnsureStepUpMFAWithAuthorizer(authorizer authorization.AuthorizationService, c *gin.Context) bool {
	return ensureMFAWithAuthorizer(authorizer, c, authorization.ActionStepUpMFARequire, authorization.StepUpMFAResource(0))
}

func requireMFAWithAuthorizer(
	authorizer authorization.AuthorizationService,
	action authorization.Action,
	resource authorization.Resource,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureMFAWithAuthorizer(authorizer, c, action, resource) {
			return
		}
		c.Next()
	}
}

func ensureMFAWithAuthorizer(
	authorizer authorization.AuthorizationService,
	c *gin.Context,
	action authorization.Action,
	resource authorization.Resource,
) bool {
	subject := subjectFromGin(c)
	decision := authorizer.Authorize(c.Request.Context(), subject, action, resource)
	logMFAGateDecision(c, subject, action, decision)
	return !abortOnDeny(c, decision)
}

func logMFAGateDecision(
	c *gin.Context,
	subject authorization.Subject,
	action authorization.Action,
	decision authorization.Decision,
) {
	if !shouldAuditMFAGate(subject, action) {
		return
	}
	audit.LogFromGin(c, mfaGateAuditEvent(c, action, decision))
}

func shouldAuditMFAGate(subject authorization.Subject, action authorization.Action) bool {
	switch action {
	case authorization.ActionStepUpMFARequire:
		return true
	case authorization.ActionMFAProofRequire:
		return subjectHasPrivilegedRole(subject)
	case authorization.ActionPrivilegedMFARequire:
		return subjectHasPrivilegedRole(subject)
	default:
		return false
	}
}

func subjectHasPrivilegedRole(subject authorization.Subject) bool {
	for _, role := range subject.Roles {
		if role == "super_admin" || role == "school_admin" {
			return true
		}
	}
	return false
}

func mfaGateAuditEvent(c *gin.Context, action authorization.Action, decision authorization.Decision) audit.Event {
	result := "failure"
	if decision.Allow {
		result = "success"
	}
	return audit.Event{
		Type:         mfaGateAuditType(action),
		Category:     "audit",
		ResourceType: "iam.mfa",
		ResourceID:   string(action),
		Action:       mfaGateAuditAction(action),
		Result:       result,
		Reason:       mfaGateReason(decision),
		Details: map[string]any{
			"authorization_action": string(action),
			"http_method":          c.Request.Method,
			"route":                c.FullPath(),
		},
	}
}

func mfaGateAuditType(action authorization.Action) audit.EventType {
	if action == authorization.ActionMFAProofRequire {
		return mfaProofAuditType
	}
	return mfaStepUpAuditType
}

func mfaGateAuditAction(action authorization.Action) string {
	if action == authorization.ActionMFAProofRequire {
		return "verify"
	}
	return "step_up"
}

func mfaGateReason(decision authorization.Decision) string {
	if decision.Error != nil {
		return decision.Error.Error()
	}
	return decision.Reason
}

func abortOnDeny(c *gin.Context, decision authorization.Decision) bool {
	if decision.Allow {
		return false
	}
	switch {
	case errors.Is(decision.Error, authorization.ErrMFAEnrollmentRequired):
		response.Forbidden(c, "mfa enrollment required", errs.ErrMFARequired)
	case errors.Is(decision.Error, authorization.ErrStepUpRequired):
		response.Error(c, http.StatusPreconditionFailed, errs.ErrStepUpRequired, "step-up required")
	case decision.Error != nil:
		response.ServiceUnavailable(c, "authorization service temporarily unavailable", errs.ErrServiceUnavailable)
	default:
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
	}
	c.Abort()
	return true
}
