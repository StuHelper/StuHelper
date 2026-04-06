package notification

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes_UsesOpenAPIPathForStreamOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &Handler{}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp)

	routes := r.Routes()
	assertRouteExists(t, routes, http.MethodGet, "/api/v1/course/review/user/notifications/stream")
	assertRouteNotExists(t, routes, http.MethodGet, "/api/v1/notifications/stream")
	assertRouteNotExists(t, routes, http.MethodGet, "/api/v1/course/review/user/notifications")
}

func assertRouteExists(t *testing.T, routes gin.RoutesInfo, method, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	assert.Failf(t, "missing route", "expected route %s %s to be registered", method, path)
}

func assertRouteNotExists(t *testing.T, routes gin.RoutesInfo, method, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			assert.Failf(t, "unexpected route", "did not expect route %s %s to be registered", method, path)
			return
		}
	}
}
