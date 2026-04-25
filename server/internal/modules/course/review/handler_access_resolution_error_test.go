package review

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewHandler_AccessResolutionFailurePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-access-error-secret-32!", false))

	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, fakeAccessReader{err: errors.New("policy unavailable")})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 10006, "统计学院")
	teacherID := seedTeacher(t, fixture, 10006, "冯老师", departmentID)
	courseID := seedCourse(t, fixture, 10006, departmentID, "数理统计")
	reviewID := "550e8400-e29b-41d4-a716-446655441301"
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "u-access-fail", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "统计评论", "统计评论内容")
	_, err := fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
	require.NoError(t, err)

	setUser := func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "u-access-fail")
		c.Set(middleware.CtxKeyCapabilities, []string{
			capability.ReviewListFull,
			capability.ReviewCreate,
			capability.ReviewEditOwn,
			capability.ReviewDeleteOwn,
		})
	}

	cases := []struct {
		name string
		run  func(*gin.Context)
	}{
		{
			name: "GetCourseReviews",
			run: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
				h.GetCourseReviews(c)
			},
		},
		{
			name: "GetLatestReviews",
			run:  func(c *gin.Context) { h.GetLatestReviews(c) },
		},
		{
			name: "SearchReviews",
			run: func(c *gin.Context) {
				c.Request.URL.RawQuery = "q=统计"
				h.SearchReviews(c)
			},
		},
		{
			name: "GetBatchCourseReviews",
			run: func(c *gin.Context) {
				c.Request.URL.RawQuery = "courseIDs=" + strconv.FormatInt(courseID, 10)
				h.GetBatchCourseReviews(c)
			},
		},
		{
			name: "PostReview",
			run: func(c *gin.Context) {
				c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"courseID":`+strconv.FormatInt(courseID, 10)+`,"teacherID":`+strconv.FormatInt(teacherID, 10)+`,"termID":"2025-2","title":"新评论","content":"足够长的评课内容用于触发 access policy 分支","grade":"A","ratings":{"teaching":5}}`))
				c.Request.Header.Set("Content-Type", "application/json")
				h.PostReview(c)
			},
		},
		{
			name: "UpdateReview",
			run: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
				c.Request = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"title":"更新评论","content":"足够长的更新内容用于触发 access policy 分支","grade":"A","ratings":{"teaching":4}}`))
				c.Request.Header.Set("Content-Type", "application/json")
				h.UpdateReview(c)
			},
		},
		{
			name: "DeleteReview",
			run: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
				c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
				h.DeleteReview(c)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, c := withUserContext(http.MethodGet, "/", "", "u-access-fail")
			setUser(c)
			tc.run(c)
			assert.Equal(t, http.StatusServiceUnavailable, w.Code)
			assert.Contains(t, w.Body.String(), "review access policy temporarily unavailable")
		})
	}
}
