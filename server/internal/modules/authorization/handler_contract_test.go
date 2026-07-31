package authorization

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/rbac"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

func TestAuthorizationAdminRoutesRequireGlobalManageCapability(t *testing.T) {
	router := authorizationRouteContractRouter([]capability.Grant{{
		Name:           capability.IAMGrantsManage,
		ScopeSchoolIDs: []string{"4111010006"},
	}})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/authorization/grants", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestAuthorizationMutationsRequireStepUpMFA(t *testing.T) {
	router := authorizationRouteContractRouter([]capability.Grant{{
		Name: capability.IAMGrantsManage,
	}})

	body := bytes.NewBufferString(`{
		"subjectUserID": 2,
		"role": "super_admin",
		"reason": "establish redundant administrator"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/authorization/grants", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusPreconditionFailed, response.Code)
}

func TestNewAuthorizationHandlerRequiresService(t *testing.T) {
	assert.PanicsWithValue(t, "authorization.NewHandler: service is required", func() {
		NewHandler(nil, AdminAuthorizers{})
	})
}

func authorizationRouteContractRouter(grants []capability.Grant) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1/admin")
	api.Use(func(c *gin.Context) {
		snapshot := capability.BuildUserAccessSnapshot(grants)
		c.Set(middleware.CtxKeyUserID, "authorization-admin")
		c.Set(middleware.CtxKeyRoles, []string{"super_admin"})
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		capabilitySet := make(map[string]struct{}, len(snapshot.Capabilities))
		for _, name := range snapshot.Capabilities {
			capabilitySet[name] = struct{}{}
		}
		c.Set(middleware.CtxKeyCapabilitySet, capabilitySet)
		middleware.SetMFAContext(c, middleware.MFAContext{EnrollmentActive: true})
		c.Next()
	})
	handler := NewHandler(
		&Service{repo: &Repository{}},
		AdminAuthorizers{
			Manage:    rbac.RequireGlobalCapability(capability.IAMGrantsManage),
			StepUpMFA: rbac.RequireStepUpMFA(),
		},
	)
	handler.RegisterAdminRoutes(api)
	return router
}
