package review

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
)

func TestReviewRegisterRoutes_UsesOpenAPIPathParamNames(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1/course/review")
	h := &Handler{}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp, noOp)

	routes := r.Routes()
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/courses/:courseID/reviews")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/courses/:courseID/rating-stats")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/course/review/reviews/:reviewID")
	routeassert.Exists(t, routes, http.MethodDelete, "/api/v1/course/review/replies/:replyID")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/teachers/:teacherID/stats")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/course/review/admin/reports/:reportID")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/course/review/admin/sensitive-words/:sensitiveWordID")
}
