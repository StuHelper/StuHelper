package review

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewHandler_RestorePendingReviewUsesModerationRelation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	reviewID := seedAuthorizationReview(t, fixture, StatusPendingReview, "pending-auth")
	authorizer := &selectiveAuthorizationProvider{
		allowed: map[checkedRelation]bool{
			{User: "user:777", Relation: reviewRelationCanHide, Object: "review:" + reviewID}: true,
		},
	}
	h := newReviewAdminHandlerWithAuthorizer(t, svc, authorizer)

	w, c := withAdminContext(http.MethodPut, "/admin/reviews/"+reviewID, `{"action":"restore","reason":"approve"}`)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.AdminUpdateReview(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, authorizer.checks, 1)
	assert.Equal(t, reviewRelationCanHide, authorizer.checks[0].Relation)
	assertReviewStatus(t, fixture, reviewID, StatusPublished)
}

func TestReviewHandler_RestoreHiddenReviewKeepsRestoreRelation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	reviewID := seedAuthorizationReview(t, fixture, StatusHidden, "hidden-auth")
	authorizer := &selectiveAuthorizationProvider{
		allowed: map[checkedRelation]bool{
			{User: "user:777", Relation: reviewRelationCanRestore, Object: "review:" + reviewID}: true,
		},
	}
	h := newReviewAdminHandlerWithAuthorizer(t, svc, authorizer)

	w, c := withAdminContext(http.MethodPut, "/admin/reviews/"+reviewID, `{"action":"restore","reason":"show again"}`)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.AdminUpdateReview(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, authorizer.checks, 1)
	assert.Equal(t, reviewRelationCanRestore, authorizer.checks[0].Relation)
	assertReviewStatus(t, fixture, reviewID, StatusPublished)
}

func TestReviewHandler_BatchRestoreUsesStatusAwareRelations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	pendingID := seedAuthorizationReview(t, fixture, StatusPendingReview, "batch-pending-auth")
	hiddenID := seedAuthorizationReview(t, fixture, StatusHidden, "batch-hidden-auth")
	authorizer := &selectiveAuthorizationProvider{
		allowed: map[checkedRelation]bool{
			{User: "user:777", Relation: reviewRelationCanHide, Object: "review:" + pendingID}:   true,
			{User: "user:777", Relation: reviewRelationCanRestore, Object: "review:" + hiddenID}: true,
		},
	}
	h := newReviewAdminHandlerWithAuthorizer(t, svc, authorizer)
	body := `{"ids":["` + pendingID + `","` + hiddenID + `"],"action":"restore"}`

	w, c := withAdminContext(http.MethodPatch, "/admin/reviews/batch", body)
	h.BatchUpdateReviews(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.ElementsMatch(t, []checkedRelation{
		{User: "user:777", Relation: reviewRelationCanHide, Object: "review:" + pendingID},
		{User: "user:777", Relation: reviewRelationCanRestore, Object: "review:" + hiddenID},
	}, authorizer.checks)
	assertReviewStatus(t, fixture, pendingID, StatusPublished)
	assertReviewStatus(t, fixture, hiddenID, StatusPublished)
}

func seedAuthorizationReview(t *testing.T, fixture *postgresfixture.Fixture, status, label string) string {
	t.Helper()
	departmentID := seedDepartment(t, fixture, 4111010006, "授权测试学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "授权测试教师"+label, departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "授权测试课程"+label)
	reviewID := deterministicReviewID(label)
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "u-"+label, 4.5, status, ReviewRatings{"teaching": 5}, "授权测试", "内容")
	return reviewID
}

func deterministicReviewID(label string) string {
	switch label {
	case "pending-auth":
		return "550e8400-e29b-41d4-a716-446655440241"
	case "hidden-auth":
		return "550e8400-e29b-41d4-a716-446655440242"
	case "batch-pending-auth":
		return "550e8400-e29b-41d4-a716-446655440243"
	default:
		return "550e8400-e29b-41d4-a716-446655440244"
	}
}

func assertReviewStatus(t *testing.T, fixture *postgresfixture.Fixture, reviewID, expected string) {
	t.Helper()
	var status string
	err := fixture.Pool.QueryRow(context.Background(), `SELECT status FROM reviews WHERE id = $1`, reviewID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, expected, status)
}
