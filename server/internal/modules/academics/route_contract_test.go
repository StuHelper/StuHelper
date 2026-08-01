package academics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/routeassert"
)

func TestRegisterRoutes_UsesOpenAPIAcademicsPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	noOp := func(c *gin.Context) { c.Next() }

	h := NewHandler(&Service{})
	h.RegisterRoutes(api, noOp)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/academics/terms")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/academics/offerings")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/academics/offerings/:offeringID")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/academics/me/courses")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/academics/me/schedule")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/academics/sources")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/academics/import-jobs")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/academics/import-jobs")
}

func TestRegisterRoutes_AdminRoutesUseAdminMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	noOp := func(c *gin.Context) { c.Next() }
	blockingMFA := func(c *gin.Context) {
		c.AbortWithStatus(http.StatusPreconditionFailed)
	}

	h := NewHandler(&Service{})
	h.RegisterRoutes(api, noOp, blockingMFA)

	for _, request := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/admin/academics/sources"},
		{method: http.MethodGet, path: "/api/v1/admin/academics/import-jobs"},
		{method: http.MethodPost, path: "/api/v1/admin/academics/import-jobs"},
	} {
		req := httptest.NewRequest(request.method, request.path, nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		require.Equal(t, http.StatusPreconditionFailed, resp.Code, "%s %s", request.method, request.path)
	}
}
