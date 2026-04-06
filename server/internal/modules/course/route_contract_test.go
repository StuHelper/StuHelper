package course

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	reviewmodule "git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
)

func TestRegisterRoutes_UsesOpenAPIPathParamNames(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &Handler{
		reviewHandler: &reviewmodule.Handler{},
	}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp, noOp)

	assertCourseRouteExists(t, r.Routes(), http.MethodGet, "/api/v1/course/courses/:courseID")
}

func assertCourseRouteExists(t *testing.T, routes gin.RoutesInfo, method, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	assert.Failf(t, "missing route", "expected route %s %s to be registered", method, path)
}
