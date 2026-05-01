package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/authorization"
)

const (
	mfaAuditRoutePattern = "/admin/identities/:userID"
	mfaAuditRequestPath  = "/admin/identities/42"
)

func TestMFAGateAuditEventRecordsStepUpFailure(t *testing.T) {
	decision := authorization.Decision{
		Allow:  false,
		Reason: "step-up required",
		Error:  authorization.ErrStepUpRequired,
	}

	event := mfaAuditEventFromRoute(t, http.MethodPut, decision)

	assert.Equal(t, mfaStepUpAuditType, event.Type)
	assert.Equal(t, "audit", event.Category)
	assert.Equal(t, "iam.mfa", event.ResourceType)
	assert.Equal(t, string(authorization.ActionStepUpMFARequire), event.ResourceID)
	assert.Equal(t, "step_up", event.Action)
	assert.Equal(t, "failure", event.Result)
	assert.Equal(t, authorization.ErrStepUpRequired.Error(), event.Reason)
	assert.Equal(t, "PUT", event.Details["http_method"])
	assert.Equal(t, mfaAuditRoutePattern, event.Details["route"])
}

func TestMFAGateAuditEventRecordsStepUpSuccess(t *testing.T) {
	decision := authorization.Decision{Allow: true, Reason: "mfa proof fresh"}

	event := mfaAuditEventFromRoute(t, http.MethodPost, decision)

	assert.Equal(t, "success", event.Result)
	assert.Equal(t, "mfa proof fresh", event.Reason)
	assert.Equal(t, "POST", event.Details["http_method"])
	assert.Equal(t, mfaAuditRoutePattern, event.Details["route"])
}

func TestShouldAuditMFAGateScope(t *testing.T) {
	assert.True(t, shouldAuditMFAGate(
		authorization.Subject{UserID: "reviewer-1"},
		authorization.ActionStepUpMFARequire,
	))
	assert.True(t, shouldAuditMFAGate(
		authorization.Subject{UserID: "admin-1", Roles: []string{"school_admin"}},
		authorization.ActionPrivilegedMFARequire,
	))
	assert.False(t, shouldAuditMFAGate(
		authorization.Subject{UserID: "reviewer-1", Roles: []string{"section_reviewer"}},
		authorization.ActionPrivilegedMFARequire,
	))
}

func mfaAuditEventFromRoute(
	t *testing.T,
	method string,
	decision authorization.Decision,
) audit.Event {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var event audit.Event
	engine := gin.New()
	engine.Handle(method, mfaAuditRoutePattern, func(c *gin.Context) {
		event = mfaGateAuditEvent(c, authorization.ActionStepUpMFARequire, decision)
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, mfaAuditRequestPath, nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotEmpty(t, event.Type)
	return event
}
