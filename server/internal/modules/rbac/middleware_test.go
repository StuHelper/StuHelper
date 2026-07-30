package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/platform/authorization"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRequireCapability_Allowed(t *testing.T) {
	called := false

	handler := RequireCapability("admin:reviews:manage")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine2 := gin.New()
	engine2.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		capSet := make(map[string]struct{})
		capSet["admin:reviews:manage"] = struct{}{}
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Set(middleware.CtxKeyCapabilities, []string{"admin:reviews:manage", "admin:logs:view"})
		c.Next()
	})
	engine2.GET("/test", handler, func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	engine2.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func TestRequireCapability_Denied(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		capSet := make(map[string]struct{})
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Set(middleware.CtxKeyCapabilities, []string{"admin:logs:view"})
		c.Next()
	})
	called := false
	engine.GET("/test", RequireCapability("admin:reviews:manage"), func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
}

func TestRequireCapability_EmptyCapabilities(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		// no capabilities set at all — simulates unauthenticated or minimal user
		c.Next()
	})
	called := false
	engine.GET("/test", RequireCapability("admin:reviews:manage"), func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}

func TestRequireAnyCapability_OneMatch(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		capSet := make(map[string]struct{})
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Set(middleware.CtxKeyCapabilities, []string{"admin:logs:view"})
		c.Next()
	})
	called := false
	engine.GET("/test",
		RequireAnyCapability("admin:reviews:manage", "admin:logs:view"),
		func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func TestRequireAnyCapability_NoMatch(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		capSet := make(map[string]struct{})
		capSet["review:list:brief"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Set(middleware.CtxKeyCapabilities, []string{"review:list:brief"})
		c.Next()
	})
	called := false
	engine.GET("/test",
		RequireAnyCapability("admin:reviews:manage", "admin:logs:view"),
		func(c *gin.Context) {
			called = true
		},
	)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}

func TestRequireAnyCapability_AllMatch(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		capSet := make(map[string]struct{})
		capSet["admin:reviews:manage"] = struct{}{}
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Set(middleware.CtxKeyCapabilities, []string{"admin:reviews:manage", "admin:logs:view"})
		c.Next()
	})
	called := false
	engine.GET("/test",
		RequireAnyCapability("admin:reviews:manage", "admin:logs:view"),
		func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func TestRequireAnyCapability_EmptyCapabilities(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		c.Next()
	})
	called := false
	engine.GET("/test",
		RequireAnyCapability("admin:reviews:manage"),
		func(c *gin.Context) {
			called = true
		},
	)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}

func TestRequireCapability_AuthorizerErrorReturns503(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		c.Next()
	})
	called := false
	handler := RequireCapabilityWithAuthorizer(errorAuthorizer{}, "admin:reviews:manage")
	engine.GET("/test", handler, func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.False(t, called)
}

func TestRequirePrivilegedMFARequiresEnrollment(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "admin-1")
		c.Set(middleware.CtxKeyRoles, []string{"super_admin"})
		c.Next()
	})
	called := false
	engine.GET("/test", RequirePrivilegedMFA(), func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
	assertErrorCode(t, w, errs.ErrMFARequired)
}

func TestRequireStepUpMFARequiresFreshProof(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "reviewer-1")
		middleware.SetMFAContext(c, middleware.MFAContext{EnrollmentActive: true})
		c.Next()
	})
	called := false
	engine.GET("/test", RequireStepUpMFA(), func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
	assert.False(t, called)
	assertErrorCode(t, w, errs.ErrStepUpRequired)
}

func TestRequirePrivilegedMFAAllowsFreshProof(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "school-admin-1")
		c.Set(middleware.CtxKeyRoles, []string{"school_admin"})
		middleware.SetMFAContext(c, middleware.MFAContext{
			EnrollmentActive: true,
			ProofVerifiedAt:  time.Now(),
		})
		c.Next()
	})
	called := false
	engine.GET("/test", RequirePrivilegedMFA(), func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func TestRequireStepUpMFABypassedWhenAuthorizerDisablesMFAGates(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "reviewer-1")
		c.Next()
	})
	called := false
	engine.GET(
		"/test",
		RequireStepUpMFAWithAuthorizer(authorization.NewService(authorization.WithMFAGatesDisabled())),
		func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, code errs.ErrorCode) {
	t.Helper()
	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	if assert.NotNil(t, resp.Error) {
		assert.Equal(t, string(code), resp.Error.Code)
	}
}

type errorAuthorizer struct{}

func (errorAuthorizer) Authorize(context.Context, authorization.Subject, authorization.Action, authorization.Resource) authorization.Decision {
	return authorization.Decision{Allow: false, Reason: "dependency failed", Error: errors.New("dependency failed")}
}

func (e errorAuthorizer) BatchAuthorize(ctx context.Context, subject authorization.Subject, checks []authorization.Check) []authorization.Decision {
	decisions := make([]authorization.Decision, len(checks))
	for i, check := range checks {
		decisions[i] = e.Authorize(ctx, subject, check.Action, check.Resource)
	}
	return decisions
}
