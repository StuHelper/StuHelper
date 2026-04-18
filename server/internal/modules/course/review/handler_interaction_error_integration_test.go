package review

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewHandler_InteractionErrorPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-handler-error-secret-32!", false))

	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	schoolID := int64(10006)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		subject: &reviewaccess.Subject{SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, schoolID, "物理学院")
	teacherID := seedTeacher(t, fixture, schoolID, "李老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "量子力学")
	reviewID := "550e8400-e29b-41d4-a716-446655441101"
	selfUserID := "handler-error-user"
	selfHash, err := httputil.HashUserID(selfUserID)
	require.NoError(t, err)
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, selfHash, 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "已存在评论", "已存在评论内容")
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
	require.NoError(t, err)

	// AddFavorite → course not found
	w, c := withUserContext(http.MethodPost, "/courses/missing/favorites", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: "999999"}}
	h.AddFavorite(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "course not found")

	// SaveDraft → course not found
	w, c = withUserContext(http.MethodPost, "/drafts", `{"courseID":999999,"title":"draft","content":"draft content"}`, selfUserID)
	h.SaveDraft(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "course not found")

	// GetDraft → draft not found
	w, c = withUserContext(http.MethodGet, "/drafts/1", "", selfUserID)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseID, 10)}}
	h.GetDraft(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "draft not found")

	// CreateReply → review not found
	w, c = withUserContext(http.MethodPost, "/reviews/replies", `{"content":"reply content"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655449999"}}
	h.CreateReply(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "review not found")

	// DeleteReply → not owner
	replyResult, err := svc.CreateReply(ctx, CreateReplyParams{
		ReviewID: reviewID,
		UserHash: "other-user-hash",
		Content:  "别人的回复",
	})
	require.NoError(t, err)

	w, c = withUserContext(http.MethodDelete, "/replies/"+replyResult.Reply.ID, "", selfUserID)
	c.Params = gin.Params{{Key: "replyID", Value: replyResult.Reply.ID}}
	h.DeleteReply(c)
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "only delete your own reply")

	// VoteReview → review not found
	w, c = withUserContext(http.MethodPost, "/reviews/vote", `{"voteType":"like"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655449998"}}
	h.VoteReview(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "review not found")

	// ReportReview → duplicate report
	_, err = svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               reviewID,
		UserHash:               selfHash,
		ReporterExternalUserID: selfUserID,
		Reason:                 "spam",
		Description:            "首次举报",
	})
	require.NoError(t, err)

	w, c = withUserContext(http.MethodPost, "/reviews/"+reviewID+"/report", `{"reason":"spam","description":"再次举报"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.ReportReview(c)
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already reported this review")
}
