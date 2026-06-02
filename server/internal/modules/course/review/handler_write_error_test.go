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

func TestReviewHandler_WriteErrorPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-write-error-secret-32!", false))

	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	schoolID := int64(4111010006)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		subject: &reviewaccess.Subject{SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, schoolID, "法学院")
	teacherID := seedTeacher(t, fixture, schoolID, "郑老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "法理学")
	selfUserID := "write-error-user"
	selfHash, err := httputil.HashUserID(selfUserID)
	require.NoError(t, err)
	selfInternalID := seedUser(t, fixture, seedUserParams{CasdoorSubject: selfUserID, UserHash: selfHash})
	svc.accessReader = fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		subject: &reviewaccess.Subject{InternalUserID: selfInternalID, SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	}

	duplicateReviewID := "550e8400-e29b-41d4-a716-446655440881"
	otherReviewID := "550e8400-e29b-41d4-a716-446655440882"
	reportReviewID := "550e8400-e29b-41d4-a716-446655440883"
	seedReviewWithRatings(t, fixture, duplicateReviewID, courseID, teacherID, selfHash, 4.6, StatusPublished, ReviewRatings{"teaching": 5}, "已写过", "已写过的内容")
	seedReviewWithRatings(t, fixture, otherReviewID, courseID, teacherID, "other-user-hash", 4.0, StatusPublished, ReviewRatings{"teaching": 4}, "他人评论", "他人评论内容")
	seedReviewWithRatings(t, fixture, reportReviewID, courseID, teacherID, "report-target-hash", 4.2, StatusPublished, ReviewRatings{"teaching": 4}, "被举报评论", "被举报评论内容")
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 3 WHERE id = $1`, courseID)
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

	// duplicate post -> 409
	w, c := withUserContext(http.MethodPost, "/reviews", `{"courseID":`+strconv.FormatInt(courseID, 10)+`,"teacherID":`+strconv.FormatInt(teacherID, 10)+`,"termID":"2025-2","title":"重复评论","content":"重复评论内容足够长用于通过校验","grade":"A","ratings":{"teaching":5}}`, selfUserID)
	setWriteAccess(c, selfUserID)
	h.PostReview(c)
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already reviewed this course")

	// update/delete not owner -> 403
	w, c = withUserContext(http.MethodPut, "/reviews/"+otherReviewID, `{"title":"非法修改","content":"非法修改的内容足够长用于通过校验","grade":"A","ratings":{"teaching":4}}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: otherReviewID}}
	setWriteAccess(c, selfUserID)
	h.UpdateReview(c)
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "only edit your own review")

	w, c = withUserContext(http.MethodDelete, "/reviews/"+otherReviewID, "", selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: otherReviewID}}
	setWriteAccess(c, selfUserID)
	h.DeleteReview(c)
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "only delete your own review")

	// duplicate report -> 409
	_, err = svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               reportReviewID,
		UserHash:               selfHash,
		ReporterInternalUserID: selfInternalID,
		Reason:                 "spam",
		Description:            "首次举报",
	})
	require.NoError(t, err)

	w, c = withUserContext(http.MethodPost, "/reviews/"+reportReviewID+"/report", `{"reason":"spam","description":"重复举报"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: reportReviewID}}
	setWriteAccess(c, selfUserID)
	h.ReportReview(c)
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already reported this review")

	// vote missing target -> 404
	w, c = withUserContext(http.MethodPost, "/reviews/missing/vote", `{"voteType":"like"}`, selfUserID)
	c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655449999"}}
	setWriteAccess(c, selfUserID)
	h.VoteReview(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "review not found")
}
