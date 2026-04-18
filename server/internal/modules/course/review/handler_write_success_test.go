package review

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewHandler_WriteAndStatsSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-write-secret-32-bytes!", false))
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	schoolID := int64(10006)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		subject: &reviewaccess.Subject{SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, schoolID, "管理学院")
	teacherID := seedTeacher(t, fixture, schoolID, "吴老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "管理学原理")
	viewerID := "writer-user-1"
	viewerHash, err := httputil.HashUserID(viewerID)
	require.NoError(t, err)
	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655440771", courseID, teacherID, "another-user-hash", 4.3, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "既有评论", "既有内容")
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
	require.NoError(t, err)

	setWriteAccess := func(c *gin.Context, userID string) {
		c.Set(middleware.CtxKeyUserID, userID)
		c.Set(middleware.CtxKeyCapabilities, []string{
			capability.ReviewListFull,
			capability.ReviewCreate,
			capability.ReviewEditOwn,
			capability.ReviewDeleteOwn,
		})
	}

	w, c := withUserContext(http.MethodPost, "/reviews", `{"courseID":`+strconv.FormatInt(courseID, 10)+`,"teacherID":`+strconv.FormatInt(teacherID, 10)+`,"termID":"2025-2","title":"新评论标题","content":"新评论内容足够长用于通过校验","grade":"A","ratings":{"teaching":5,"difficulty":4}}`, viewerID)
	setWriteAccess(c, viewerID)
	h.PostReview(c)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "新评论标题")

	var createdReviewID string
	err = fixture.Pool.QueryRow(ctx, `SELECT id FROM reviews WHERE user_hash = $1 AND title = $2 LIMIT 1`, viewerHash, "新评论标题").Scan(&createdReviewID)
	require.NoError(t, err)

	w, c = withUserContext(http.MethodPut, "/reviews/"+createdReviewID, `{"title":"更新评论标题","content":"更新后的评论内容也足够长用于通过校验","grade":"A+","ratings":{"teaching":4,"difficulty":5}}`, viewerID)
	c.Params = gin.Params{{Key: "reviewID", Value: createdReviewID}}
	setWriteAccess(c, viewerID)
	h.UpdateReview(c)
	assert.Equal(t, http.StatusOK, w.Code)

	w, c = withUserContext(http.MethodDelete, "/reviews/"+createdReviewID, "", viewerID)
	c.Params = gin.Params{{Key: "reviewID", Value: createdReviewID}}
	setWriteAccess(c, viewerID)
	h.DeleteReview(c)
	assert.Equal(t, http.StatusOK, w.Code)

	w, c = withUserContext(http.MethodGet, "/courses/1/rating-stats", "", viewerID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.GetCourseRatingStats(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "allDimensionKeys")
	assert.Contains(t, w.Body.String(), "teaching")

	w, c = withUserContext(http.MethodGet, "/stats", "", viewerID)
	h.GetStats(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "reviewCount")

	require.NoError(t, svc.RefreshTeacherPublicStats(ctx))
	w, c = withUserContext(http.MethodGet, "/teachers/hot", "", viewerID)
	h.ListHotTeachers(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "吴老师")
}
