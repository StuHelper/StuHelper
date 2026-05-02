package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
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
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/course/review/admin/export")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/course/review/admin/sensitive-words/:sensitiveWordID")
}

func TestReviewExportRequiresGlobalReviewManageCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1/course/review")
	authMW := scopedSectionModeratorAuth()
	(&Handler{}).RegisterRoutes(api, authMW, authMW)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/admin/export", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReviewTeacherAdminRequiresGlobalCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1/course/review")
	authMW := scopedCapabilityAuth(capability.AdminTeachersManage)
	(&Handler{}).RegisterRoutes(api, authMW, authMW)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/admin/teachers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func scopedSectionModeratorAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := []string{"section_moderator"}
		scopes := map[string][]string{"section_moderator": {reviewModerationSectionID(10006)}}
		snapshot := capability.BuildUserAccessSnapshot(capability.ExpandRoleGrants(roles, scopes))
		c.Set(middleware.CtxKeyUserID, "section-moderator-1")
		c.Set(middleware.CtxKeyRoles, roles)
		c.Set(middleware.CtxKeyOrgScopedRoles, scopes)
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Next()
	}
}

func scopedCapabilityAuth(capabilityName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		snapshot := capability.BuildUserAccessSnapshot([]capability.Grant{{
			Name:           capabilityName,
			ScopeSchoolIDs: []string{"10006"},
		}})
		c.Set(middleware.CtxKeyUserID, "scoped-admin-1")
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Next()
	}
}
