package rbac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func init() { gin.SetMode(gin.TestMode) }

// setupCapabilityContext creates a gin context with capabilities pre-injected
// (simulating what AuthMiddleware does after token verification).
func setupCapabilityContext(capabilities []string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	capSet := make(map[string]struct{}, len(capabilities))
	for _, cap := range capabilities {
		capSet[cap] = struct{}{}
	}
	c.Set(middleware.CtxKeyCapabilitySet, capSet)
	c.Set(middleware.CtxKeyCapabilities, capabilities)
	return c, w
}

func TestRequireCapability_Allowed(t *testing.T) {
	c, w := setupCapabilityContext([]string{"admin:reviews:manage", "admin:logs:view"})
	called := false

	handler := RequireCapability("admin:reviews:manage")
	c.Set("_gin_handler_index", 0) // needed for c.Next()
	// Build a simple handler chain
	engine := gin.New()
	engine.GET("/test", handler, func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// inject capabilities into request context via middleware
	engine.Use(func(c *gin.Context) {
		capSet := make(map[string]struct{})
		capSet["admin:reviews:manage"] = struct{}{}
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Next()
	})
	// Rebuild with middleware first
	engine2 := gin.New()
	engine2.Use(func(c *gin.Context) {
		capSet := make(map[string]struct{})
		capSet["admin:reviews:manage"] = struct{}{}
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Next()
	})
	engine2.GET("/test", handler, func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w = httptest.NewRecorder()
	engine2.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func TestRequireCapability_Denied(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		capSet := make(map[string]struct{})
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
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
		capSet := make(map[string]struct{})
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
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
		capSet := make(map[string]struct{})
		capSet["review:list:brief"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
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
		capSet := make(map[string]struct{})
		capSet["admin:reviews:manage"] = struct{}{}
		capSet["admin:logs:view"] = struct{}{}
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
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
