package user

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/testutil/routeassert"
)

func TestRegisterRoutes_UsesOpenAPIUserPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &Handler{}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/user/me")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/user/identity")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/identity")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/identity/uploads")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/user/profile")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/profile/verify")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/profile/school-email/academic-match")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/profile/school-email/request-otp")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/profile/school-email/verify-otp")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/profile/bind-phone/otp")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/profile/bind-phone")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/user/qq-binding")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/user/qq-binding/code")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/user/profile/academic-info")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/user/schools")
}

func TestRegisterBotRoutes_UsesOpenAPIBotUserPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &BotHandler{}

	h.RegisterRoutes(api)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/qq-binding/consume")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/qq-users/:qqID/verification")
}

func TestRegisterAdminRoutes_UsesOpenAPIAdminUserPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	admin := r.Group("/api/v1/admin")
	h := &Handler{}

	h.RegisterAdminRoutes(admin)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/identities")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/identities/:userID")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/identities/:userID")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/student-verifications")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/student-verifications/:userID")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/school-configs")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/school-configs/:schoolID")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/system-configs")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/system-configs/:key")
}
