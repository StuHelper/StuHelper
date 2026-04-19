package resource

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
)

func TestRegisterRoutes_UsesOpenAPIResourcePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	noOp := func(c *gin.Context) { c.Next() }

	h := &Handler{}
	h.RegisterRoutes(api, noOp, noOp)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/resources")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/resources/:resourceID")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/resources/:resourceID/download-url")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/resources")
	routeassert.Exists(t, routes, http.MethodPatch, "/api/v1/resources/:resourceID")
	routeassert.Exists(t, routes, http.MethodDelete, "/api/v1/resources/:resourceID")
}
