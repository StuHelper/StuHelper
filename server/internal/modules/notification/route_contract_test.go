package notification

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
)

func TestRegisterRoutes_UsesOpenAPINotificationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &Handler{}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/user/notifications")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/user/notifications/stream")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/user/notifications/unread-count")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/course/review/user/notifications/:notificationID/read")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/course/review/user/notifications/read-all")
	routeassert.NotExists(t, routes, http.MethodGet, "/api/v1/notifications/stream")
}
