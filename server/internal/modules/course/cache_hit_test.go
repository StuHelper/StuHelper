package course

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cachepkg "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func setupCourseCacheHandler(t *testing.T) (*Handler, context.Context) {
	t.Helper()
	fixture := redisfixture.Start(t)

	return &Handler{cache: cachepkg.NewHelper(fixture.Client)}, context.Background()
}

func TestCourseHandlers_ServeFromCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, ctx := setupCourseCacheHandler(t)
	require.NoError(t, crypto.InitHMACKey("test-course-cache-hit-secret-32-bytes", false))

	require.NoError(t, h.cache.Set(ctx, "course:categories", []CourseCategory{{ID: 1, Name: "公共课"}}, time.Minute))
	require.NoError(t, h.cache.Set(ctx, "course:departments:"+httputil.SanitizeCacheKey("science"), []Department{{ID: 1, Name: "理学院"}}, time.Minute))
	require.NoError(t, h.cache.Set(ctx, "course:terms", []Term{{ID: "2024-1", Name: "2024春", IsCurrent: true}}, time.Minute))
	require.NoError(t, h.cache.Set(ctx, "course:stats", gin.H{"courseCount": 10, "departmentCount": 2}, time.Minute))

	mustSetCourseCourses := func(key string, value any) {
		t.Helper()
		cacheKey := h.cache.BuildVersionedKey(ctx, courseCoursesCachePrefix, key)
		require.NoError(t, h.cache.Set(ctx, cacheKey, value, time.Minute))
	}
	mustSetCourse := func(key string, value any) {
		t.Helper()
		cacheKey := h.cache.BuildVersionedKey(ctx, courseCourseCachePrefix, key)
		require.NoError(t, h.cache.Set(ctx, cacheKey, value, time.Minute))
	}

	mustSetCourseCourses("course:courses:grouped", gin.H{"groups": []DepartmentGroup{{DepartmentID: 1, DepartmentName: "理学院"}}})
	mustSetCourse("course:course:1", Course{ID: 1, Code: "MATH101", Name: "高数"})
	mustSetCourseCourses("course:courses:q="+httputil.SanitizeCacheKey("math")+":dept=0:cat="+httputil.SanitizeCacheKey("")+":sort=name:page=1:size=20", gin.H{"list": []Course{{ID: 1, Code: "MATH101", Name: "高数"}}, "total": 1})
	mustSetCourseCourses("course:courses:search:"+httputil.SanitizeCacheKey("math")+":page=1:size=20", gin.H{"list": []Course{{ID: 1, Code: "MATH101", Name: "高数"}}, "total": 1})

	cases := []struct {
		name string
		url  string
		run  func(*Handler, *gin.Context)
		prep func(*gin.Context)
		want string
	}{
		{name: "GetCourseCategories", url: "/", run: func(h *Handler, c *gin.Context) { h.GetCourseCategories(c) }, want: "公共课"},
		{name: "GetDepartments", url: "/?category=science", run: func(h *Handler, c *gin.Context) { h.GetDepartments(c) }, want: "理学院"},
		{name: "GetTerms", url: "/", run: func(h *Handler, c *gin.Context) { h.GetTerms(c) }, want: "2024春"},
		{name: "GetCoursesGrouped", url: "/", run: func(h *Handler, c *gin.Context) { h.GetCoursesGrouped(c) }, want: "理学院"},
		{name: "GetStats", url: "/", run: func(h *Handler, c *gin.Context) { h.GetStats(c) }, want: "courseCount"},
		{name: "GetCourse", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourse(c) }, want: "MATH101"},
		{name: "GetCourses", url: "/?q=math", run: func(h *Handler, c *gin.Context) { h.GetCourses(c) }, want: "MATH101"},
		{name: "SearchCourses", url: "/?q=math", run: func(h *Handler, c *gin.Context) { h.SearchCourses(c) }, want: "MATH101"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tc.url, nil)
			if tc.prep != nil {
				tc.prep(c)
			}
			assert.NotPanics(t, func() { tc.run(h, c) })
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tc.want)
		})
	}
}
