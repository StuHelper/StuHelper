package course

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	reviewmodule "git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
)

func TestCourseRegisterRoutes_UsesOpenAPIPathParamNames(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	h := &Handler{
		reviewHandler: &reviewmodule.Handler{},
	}
	noOp := func(c *gin.Context) { c.Next() }

	h.RegisterRoutes(api, noOp, noOp)

	routeassert.Exists(t, r.Routes(), http.MethodGet, "/api/v1/course/courses/:courseID")
}
