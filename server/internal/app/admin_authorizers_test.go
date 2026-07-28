package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestSensitiveAdminAuthorizersRequireStepUpMFA(t *testing.T) {
	configureRBACAuthorizer(config.EnvProduction)
	t.Cleanup(func() {
		configureRBACAuthorizer(config.EnvProduction)
	})

	middlewares := map[string]gin.HandlerFunc{
		"auth account unlock": authAdminAuthorizers().StepUpMFA,
		"user mutation":       userAdminAuthorizers().StepUpMFA,
	}
	for name, stepUp := range middlewares {
		t.Run(name, func(t *testing.T) {
			w, called := exerciseStepUpMiddleware(stepUp)

			assert.Equal(t, http.StatusPreconditionRequired, w.Code)
			assert.False(t, called)
		})
	}
}

func TestReviewAdminAuthorizerRequiresStepUpMFA(t *testing.T) {
	configureRBACAuthorizer(config.EnvProduction)
	t.Cleanup(func() {
		configureRBACAuthorizer(config.EnvProduction)
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(adminStepUpTestContext())
	called := false
	engine.GET("/admin/reviews", func(c *gin.Context) {
		if !reviewAdminAuthorizers().StepUpVerified(c) {
			return
		}
		called = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/reviews", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
	assert.False(t, called)
}

func exerciseStepUpMiddleware(stepUp gin.HandlerFunc) (*httptest.ResponseRecorder, bool) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(adminStepUpTestContext())
	called := false
	engine.GET("/admin/sensitive", stepUp, func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/sensitive", nil)
	engine.ServeHTTP(w, req)
	return w, called
}

func adminStepUpTestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "casdoor-admin")
		c.Set(middleware.CtxKeyRoles, []string{"super_admin"})
		middleware.SetMFAContext(c, middleware.MFAContext{EnrollmentActive: true})
		c.Next()
	}
}
