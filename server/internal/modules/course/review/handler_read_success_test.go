package review

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestReviewHandler_ReadSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-read-secret-32-bytes!!", false))

	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	schoolID := int64(4111010006)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		subject: &reviewaccess.Subject{InternalUserID: 42, SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, schoolID, "信息学院")
	teacherID := seedTeacher(t, fixture, schoolID, "陈老师", departmentID)
	otherTeacherID := seedTeacher(t, fixture, schoolID, "王老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "计算机网络")
	otherCourseID := seedCourse(t, fixture, schoolID, departmentID, "操作系统")

	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655441001", courseID, teacherID, "u-read-1", 4.8, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "网络课好评", "网络课内容一")
	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655441002", courseID, teacherID, "u-read-2", 4.2, StatusPublished, ReviewRatings{"teaching": 4, "difficulty": 5}, "网络课二评", "网络课内容二")
	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655441003", otherCourseID, teacherID, "u-read-3", 4.0, StatusPublished, ReviewRatings{"teaching": 4, "difficulty": 4}, "系统课好评", "系统课内容三")
	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655441004", courseID, otherTeacherID, "u-read-4", 4.1, StatusPublished, ReviewRatings{"teaching": 4, "difficulty": 4}, "其他老师评价", "其他老师内容")
	_, err := fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 2 WHERE id = $1`, courseID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, otherCourseID)
	require.NoError(t, err)

	require.NoError(t, svc.RefreshTeacherPublicStats(ctx))

	setReadContext := func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "read-user-1")
		c.Set(middleware.CtxKeyCapabilities, []string{capability.ReviewListFull})
	}

	w, c := withUserContext(http.MethodGet, "/courses/1/reviews?sort=time", "", "read-user-1")
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	setReadContext(c)
	h.GetCourseReviews(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "网络课好评")

	w, c = withUserContext(http.MethodGet, "/reviews/latest", "", "read-user-1")
	setReadContext(c)
	h.GetLatestReviews(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "系统课好评")

	w, c = withUserContext(http.MethodGet, "/reviews/latest?teacherID="+strconv.FormatInt(teacherID, 10), "", "read-user-1")
	setReadContext(c)
	h.GetLatestReviews(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "网络课好评")
	assert.Contains(t, w.Body.String(), "系统课好评")
	assert.NotContains(t, w.Body.String(), "其他老师评价")

	w, c = withUserContext(http.MethodGet, "/reviews/search?q=网络", "", "read-user-1")
	c.Request.URL.RawQuery = "q=网络&teacherName=陈老师&termID=2025-2&departmentID=" + strconv.FormatInt(departmentID, 10)
	setReadContext(c)
	h.SearchReviews(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "网络课好评")

	w, c = withUserContext(http.MethodGet, "/reviews/batch", "", "read-user-1")
	c.Request.URL.RawQuery = "courseIDs=" + strconv.FormatInt(courseID, 10) + "," + strconv.FormatInt(otherCourseID, 10)
	setReadContext(c)
	h.GetBatchCourseReviews(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), strconv.FormatInt(courseID, 10))
	assert.Contains(t, w.Body.String(), strconv.FormatInt(otherCourseID, 10))

	w, c = withUserContext(http.MethodGet, "/stats/rating-trend", "", "read-user-1")
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	setReadContext(c)
	h.GetRatingTrend(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "2025-2")

	w, c = withUserContext(http.MethodGet, "/courses/hot", "", "read-user-1")
	c.Request.URL.RawQuery = "period=all&limit=5"
	setReadContext(c)
	h.GetHotCourses(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "计算机网络")

	w, c = withUserContext(http.MethodGet, "/courses/teachers", "", "read-user-1")
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	setReadContext(c)
	h.GetCourseTeachers(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "陈老师")

	w, c = withUserContext(http.MethodGet, "/teachers/stats", "", "read-user-1")
	c.Params = gin.Params{{Key: "teacherID", Value: strconv.FormatInt(teacherID, 10)}}
	setReadContext(c)
	h.GetTeacherRatingStats(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "陈老师")
}
