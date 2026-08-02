package review

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/rbac"
	cachepkg "github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/routeassert"
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
	(&Handler{adminAuthorizers: reviewAdminAuthorizers()}).RegisterRoutes(api, authMW, authMW)

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
	(&Handler{adminAuthorizers: reviewAdminAuthorizers()}).RegisterRoutes(api, authMW, authMW)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/admin/teachers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReviewModerationRoutesUseScopedCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		method         string
		capabilityName string
		expectedStatus int
	}{
		{
			name:           "report management accepts scoped report capability",
			path:           "/api/v1/course/review/admin/reports/not-a-uuid",
			method:         http.MethodPut,
			capabilityName: capability.AdminReportsManage,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "review moderation accepts scoped review capability",
			path:           "/api/v1/course/review/admin/reviews/not-a-uuid",
			method:         http.MethodPut,
			capabilityName: capability.AdminReviewsManage,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "content editing accepts dedicated scoped capability",
			path:           "/api/v1/course/review/admin/reviews/not-a-uuid/edit",
			method:         http.MethodPost,
			capabilityName: capability.AdminReviewsEditContent,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "review moderation capability cannot edit content",
			path:           "/api/v1/course/review/admin/reviews/not-a-uuid/edit",
			method:         http.MethodPost,
			capabilityName: capability.AdminReviewsManage,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			api := r.Group("/api/v1/course/review")
			authMW := scopedCapabilityAuth(tc.capabilityName)
			(&Handler{adminAuthorizers: reviewAdminAuthorizers()}).RegisterRoutes(api, authMW, authMW)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestReviewAdminStatsDoesNotUseGroupStepUpGate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixture := redisfixture.Start(t)
	h := &Handler{
		cache:            cachepkg.NewHelper(fixture.Client),
		adminAuthorizers: reviewAdminAuthorizers(),
	}
	ctx := context.Background()
	cacheKey := h.cache.BuildVersionedKey(ctx, "review:admin:stats", "all")
	require.NoError(t, h.cache.Set(ctx, cacheKey, gin.H{"totalReviews": 1}, time.Minute))

	r := gin.New()
	api := r.Group("/api/v1/course/review")
	authMW := globalAdminCapabilityAuth(capability.AdminDashboardView, capability.AdminLogsView)
	blockingStepUpGate := func(c *gin.Context) {
		response.Error(c, http.StatusPreconditionFailed, errs.ErrStepUpRequired, "step-up required")
	}
	h.RegisterRoutes(api, authMW, authMW, blockingStepUpGate)

	stats := httptest.NewRecorder()
	statsReq := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/admin/stats", nil)
	r.ServeHTTP(stats, statsReq)

	assert.Equal(t, http.StatusOK, stats.Code)
	assert.Contains(t, stats.Body.String(), "totalReviews")
	assert.NotContains(t, stats.Body.String(), string(errs.ErrStepUpRequired))

	logs := httptest.NewRecorder()
	logsReq := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/admin/logs", nil)
	r.ServeHTTP(logs, logsReq)

	assert.Equal(t, http.StatusPreconditionFailed, logs.Code)
	assert.Contains(t, logs.Body.String(), string(errs.ErrStepUpRequired))
}

func reviewAdminAuthorizers() AdminAuthorizers {
	return AdminAuthorizers{
		Entry:                rbac.RequireAnyCapability(capability.AdminEntryCapabilities...),
		DashboardView:        rbac.RequireGlobalCapability(capability.AdminDashboardView),
		LogsView:             rbac.RequireGlobalCapability(capability.AdminLogsView),
		ReviewsManage:        rbac.RequireGlobalCapability(capability.AdminReviewsManage),
		ReviewsModerate:      rbac.RequireCapability(capability.AdminReviewsManage),
		ReviewsEditContent:   rbac.RequireCapability(capability.AdminReviewsEditContent),
		ReportsManage:        rbac.RequireCapability(capability.AdminReportsManage),
		TeachersManage:       rbac.RequireGlobalCapability(capability.AdminTeachersManage),
		SensitiveWordsManage: rbac.RequireGlobalCapability(capability.AdminSensitiveWordsManage),
		StepUpVerified:       rbac.EnsureStepUpMFA,
	}
}

func globalAdminCapabilityAuth(capabilityNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		grants := make([]capability.Grant, 0, len(capabilityNames))
		for _, capabilityName := range capabilityNames {
			grants = append(grants, capability.Grant{Name: capabilityName})
		}
		snapshot := capability.BuildUserAccessSnapshot(grants)
		c.Set(middleware.CtxKeyUserID, "global-admin-1")
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Next()
	}
}

func scopedSectionModeratorAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := []string{"section_moderator"}
		scopes := map[string][]string{"section_moderator": {reviewModerationSectionID(4111010006)}}
		snapshot := capability.BuildUserAccessSnapshot(capability.ExpandRoleGrants(roles, scopes))
		c.Set(middleware.CtxKeyUserID, "section-moderator-1")
		c.Set(middleware.CtxKeyRoles, roles)
		c.Set(middleware.CtxKeyScopedRoleGrants, scopes)
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
			ScopeSchoolIDs: []string{"4111010006"},
		}})
		c.Set(middleware.CtxKeyUserID, "scoped-admin-1")
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Next()
	}
}
