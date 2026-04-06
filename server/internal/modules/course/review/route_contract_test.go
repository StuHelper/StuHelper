package review

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes_UsesOpenAPIPathParamNames(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1/course/review")
	h := &Handler{}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp, noOp)

	routes := r.Routes()
	assertRouteExists(t, routes, http.MethodGet, "/api/v1/course/review/courses/:courseID/reviews")
	assertRouteExists(t, routes, http.MethodGet, "/api/v1/course/review/courses/:courseID/rating-stats")
	assertRouteExists(t, routes, http.MethodPut, "/api/v1/course/review/reviews/:reviewID")
	assertRouteExists(t, routes, http.MethodDelete, "/api/v1/course/review/replies/:replyID")
	assertRouteExists(t, routes, http.MethodGet, "/api/v1/course/review/teachers/:teacherID/stats")
	assertRouteExists(t, routes, http.MethodPut, "/api/v1/course/review/admin/reports/:reportID")
	assertRouteExists(t, routes, http.MethodPut, "/api/v1/course/review/admin/sensitive-words/:sensitiveWordID")
	assertRouteExists(t, routes, http.MethodPut, "/api/v1/course/review/user/notifications/:notificationID/read")
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
