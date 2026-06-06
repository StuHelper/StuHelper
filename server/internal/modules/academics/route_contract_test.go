package academics

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
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
