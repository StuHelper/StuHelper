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

	cachepkg "github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func setupReviewCacheHandler(t *testing.T) (*Handler, context.Context) {
	t.Helper()
	fixture := redisfixture.Start(t)

	return &Handler{cache: cachepkg.NewHelper(fixture.Client)}, context.Background()
}

func TestReviewHandlers_ServeFromCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, ctx := setupReviewCacheHandler(t)
	require.NoError(t, crypto.InitHMACKey("test-review-cache-hit-secret-32-bytes", false))

	mustSet := func(prefix, key string, value any) {
		t.Helper()
		cacheKey := h.cache.BuildVersionedKey(ctx, prefix, key)
		require.NoError(t, h.cache.Set(ctx, cacheKey, value, time.Minute))
	}

	mustSet("review:stats", "all", gin.H{"courseCount": 10, "reviewCount": 20, "departmentCount": 3, "userCount": 4})
	mustSet("review:rating_dimensions", "all", []RatingDimension{{ID: "dim-1", SchoolID: 4111010006, Key: "teaching", Name: "教学", IsActive: true}})
	mustSet(courseRatingStatsCacheKey, "1", CourseRatingStatsResponse{CourseID: 1, Overall: TermRatingStats{TermID: "overall", TermName: "Overall", Dimensions: []DimensionStats{{Key: "teaching", Name: "教学", AvgRating: 4.5, RatingCount: 1, Distribution: map[int]int{5: 1}}}}, AllDimensionKeys: []string{"teaching"}})
	mustSet("review:rating_trend", "1", gin.H{"trend": []RatingTrendItem{{TermID: "2024-1", AvgRating: 4.5}}})
	mustSet("review:hot", "period=all:limit=20", gin.H{"list": []HotCourse{{CourseID: 1, CourseName: "高数"}}})
	mustSet("review:course_teachers", "1", []CourseTeacherStats{{TeacherID: 1, TeacherName: "张老师"}})
	mustSet("review:teacher_stats", "teacherID=1", TeacherRatingStatsResponse{TeacherID: 1, TeacherName: "张老师", Overall: TermRatingStats{TermID: "overall", TermName: "Overall", Dimensions: []DimensionStats{{Key: "overall", Name: "综合", AvgRating: 4.5, RatingCount: 1, Distribution: map[int]int{5: 1}}}}})
	mustSet(teacherPublicListCacheKey, "q="+httputil.SanitizeCacheKey("")+":dept=0:sort=reviews:p=1:ps=20", gin.H{"list": []TeacherSummary{{TeacherID: 1, TeacherName: "张老师"}}, "total": 1})
	mustSet(teacherPublicHotCacheKey, "limit=10", gin.H{"list": []TeacherSummary{{TeacherID: 1, TeacherName: "张老师"}}})
	mustSet("review:admin:reports", "status=pending:page=1:size=20:scope=none", gin.H{"list": []ReviewReport{{ID: "550e8400-e29b-41d4-a716-446655440000", ReviewID: "660e8400-e29b-41d4-a716-446655440000", Status: ReportStatusPending}}, "total": 1})
	mustSet("review:admin:stats", "all", gin.H{"totalReviews": 1})

	cases := []struct {
		name string
		url  string
		prep func(*gin.Context)
		run  func(*Handler, *gin.Context)
		want string
	}{
		{name: "GetStats", url: "/", run: func(h *Handler, c *gin.Context) { h.GetStats(c) }, want: "reviewCount"},
		{name: "GetRatingDimensions", url: "/", run: func(h *Handler, c *gin.Context) { h.GetRatingDimensions(c) }, want: "教学"},
		{name: "GetCourseRatingStats", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourseRatingStats(c) }, want: "teaching"},
		{name: "GetRatingTrend", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetRatingTrend(c) }, want: "2024-1"},
		{name: "GetHotCourses", url: "/?period=all", run: func(h *Handler, c *gin.Context) { h.GetHotCourses(c) }, want: "高数"},
		{name: "GetCourseTeachers", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourseTeachers(c) }, want: "张老师"},
		{name: "GetTeacherRatingStats", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "teacherID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetTeacherRatingStats(c) }, want: "overall"},
		{name: "ListTeachers", url: "/", run: func(h *Handler, c *gin.Context) { h.ListTeachers(c) }, want: "张老师"},
		{name: "ListHotTeachers", url: "/", run: func(h *Handler, c *gin.Context) { h.ListHotTeachers(c) }, want: "张老师"},
		{name: "ListReports", url: "/", run: func(h *Handler, c *gin.Context) { h.ListReports(c) }, want: ReportStatusPending},
		{name: "GetAdminStats", url: "/", run: func(h *Handler, c *gin.Context) { h.GetAdminStats(c) }, want: "totalReviews"},
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
