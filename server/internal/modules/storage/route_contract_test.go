package storage

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
)

func TestRegisterAdminRoutes_UsesOpenAPIStoragePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	noOp := func(c *gin.Context) { c.Next() }

	h := &Handler{}
	h.RegisterAdminRoutes(api, noOp)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/storage/mounts")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/storage/mounts")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/storage/mounts/:mountID/health-check")
}
