package auth

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/testutil/routeassert"
)

func TestRegisterRoutes_UsesOpenAPIAuthPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &Handler{}
	h.RegisterPublicRoutes(api)
	h.RegisterRoutes(api, nil, nil)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/auth/exchange-native")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/auth/login")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/auth/signup")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/auth/step-up")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/auth/callback")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/auth/refresh")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/auth/me")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/auth/logout")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/auth/logout-all")
}

func TestRegisterAdminRoutes_UsesOpenAPIAuthAdminPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	admin := r.Group("/api/v1/admin")
	h := &Handler{}
	h.RegisterAdminRoutes(admin)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/auth/account-locks/unlock")
}
