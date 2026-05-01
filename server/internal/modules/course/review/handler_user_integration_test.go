package review

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func withUserContext(method, target, body, userID string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, httptest.NewRecorder().Body)
	if body != "" {
		c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Set(middleware.CtxKeyUserID, userID)
	return w, c
}

func TestReviewHandler_UserInteractionSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-user-handler-secret-32!!", false))

	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	schoolID := int64(10006)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		configs: nil,
		subject: &reviewaccess.Subject{InternalUserID: 42, SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 10006, "电信学院")
	teacherID := seedTeacher(t, fixture, 10006, "周老师", departmentID)
	courseID := seedCourse(t, fixture, 10006, departmentID, "数字信号处理")
	selfUserID := "user-self-1"
	selfHash, err := httputil.HashUserID(selfUserID)
	require.NoError(t, err)
	seedUser(t, fixture, seedUserParams{CasdoorSubject: selfUserID, UserHash: selfHash})
	reviewID := "550e8400-e29b-41d4-a716-446655440301"
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, selfHash, 4.4, StatusPublished, ReviewRatings{"teaching": 5}, "我的评论", "我自己的评论内容")
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
	require.NoError(t, err)

	// favorite add/status/list/remove
	w, c := withUserContext(http.MethodPost, "/courses/1/favorites", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.AddFavorite(c)
	assert.Equal(t, http.StatusOK, w.Code)

	w, c = withUserContext(http.MethodGet, "/courses/1/favorites", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.GetFavoriteStatus(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"favorited":true`)

	w, c = withUserContext(http.MethodGet, "/user/favorites", "", selfUserID)
	h.GetUserFavorites(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "数字信号处理")

	w, c = withUserContext(http.MethodDelete, "/courses/1/favorites", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.RemoveFavorite(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// user reviews list
	w, c = withUserContext(http.MethodGet, "/user/reviews", "", selfUserID)
	h.GetUserReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reviewID)

	// latest reviews
	w, c = withUserContext(http.MethodGet, "/reviews/latest", "", selfUserID)
	c.Set(middleware.CtxKeyCapabilities, []string{"review.list.full"})
	h.GetLatestReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reviewID)

	// report review
	w, c = withUserContext(http.MethodPost, "/reviews/report", `{"reason":"spam","description":"需要处理"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.ReportReview(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "report submitted successfully")

	// content check
	w, c = withUserContext(http.MethodPost, "/content/check", `{"content":"普通内容"}`, selfUserID)
	h.CheckContent(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"isValid":true`)

	// vote + user votes list
	w, c = withUserContext(http.MethodPost, "/reviews/vote", `{"voteType":"like"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.VoteReview(c)
	assert.Equal(t, http.StatusOK, w.Code)

	w, c = withUserContext(http.MethodGet, "/user/votes?voteType=like", "", selfUserID)
	h.GetUserVotes(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reviewID)

	// draft save/get/delete
	w, c = withUserContext(http.MethodPost, "/drafts", `{"courseID":`+strconv.FormatInt(courseID, 10)+`,"teacherID":`+strconv.FormatInt(teacherID, 10)+`,"termID":"2025-2","title":"草稿","content":"草稿内容","grade":"A","ratings":{"teaching":4}}`, selfUserID)
	h.SaveDraft(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "草稿")

	w, c = withUserContext(http.MethodGet, "/drafts/1", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.GetDraft(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "草稿内容")

	w, c = withUserContext(http.MethodDelete, "/drafts/1", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.DeleteDraft(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// reply create/list/delete
	w, c = withUserContext(http.MethodPost, "/reviews/replies", `{"content":"谢谢分享"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.CreateReply(c)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "谢谢分享")

	var replyID string
	err = fixture.Pool.QueryRow(ctx, `SELECT id FROM review_replies WHERE review_id = $1 ORDER BY created_at DESC LIMIT 1`, reviewID).Scan(&replyID)
	require.NoError(t, err)

	w, c = withUserContext(http.MethodGet, "/reviews/replies", "", selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.GetReplies(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), replyID)
	assert.Contains(t, w.Body.String(), `"isOwner":true`)

	w, c = withUserContext(http.MethodDelete, "/replies/1", "", selfUserID)
	c.Params = gin.Params{{Key: "replyID", Value: replyID}}
	h.DeleteReply(c)
	assert.Equal(t, http.StatusOK, w.Code)
}
