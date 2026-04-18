package review

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type allowAllAuthorizationProvider struct{}

func (allowAllAuthorizationProvider) Check(context.Context, string, string, string) (bool, error) {
	return true, nil
}
func (allowAllAuthorizationProvider) WriteReviewRelations(context.Context, string, string, string, string) error {
	return nil
}
func (allowAllAuthorizationProvider) WriteReportRelations(context.Context, string, string, string, string) error {
	return nil
}

func newReviewAdminHandler(t *testing.T, svc *Service) *Handler {
	t.Helper()
	fixture := redisfixture.Start(t)

	cacheHelper := cache.NewHelper(fixture.Client)
	return NewHandler(cacheHelper, svc, fixture.Client, config.ReviewRateLimitConfig{}, allowAllAuthorizationProvider{})
}

func withAdminContext(method, target, body string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	c.Request = httptest.NewRequest(method, target, reqBody)
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Set(middleware.CtxKeyUserID, "admin-user-1")
	c.Set(middleware.CtxKeyUsername, "admin-root")
	return w, c
}

func TestReviewHandler_AdminSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	h := newReviewAdminHandler(t, svc)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 10006, "计算学院")
	teacherID := seedTeacher(t, fixture, 10006, "孙老师", departmentID)
	courseID := seedCourse(t, fixture, 10006, departmentID, "人工智能")
	reviewID := "550e8400-e29b-41d4-a716-446655440201"
	reviewID2 := "550e8400-e29b-41d4-a716-446655440202"
	flaggedID := "550e8400-e29b-41d4-a716-446655440203"
	reportableID := "550e8400-e29b-41d4-a716-446655440204"
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "u-handler-1", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "标题一", "内容一")
	seedReviewWithRatings(t, fixture, reviewID2, courseID, teacherID, "u-handler-2", 4.0, StatusPublished, ReviewRatings{"teaching": 4}, "标题二", "内容二")
	seedReviewWithRatings(t, fixture, flaggedID, courseID, teacherID, "u-handler-3", 3.8, StatusHidden, ReviewRatings{"teaching": 3}, "标题三", "内容三")
	seedReviewWithRatings(t, fixture, reportableID, courseID, teacherID, "u-handler-4", 4.1, StatusPublished, ReviewRatings{"teaching": 4}, "标题四", "内容四")
	_, err := fixture.Pool.Exec(ctx, `UPDATE reviews SET content_flag = $2 WHERE id = $1`, flaggedID, ContentFlagWarn)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 2 WHERE id = $1`, courseID)
	require.NoError(t, err)

	// ListAllReviews
	w, c := withAdminContext(http.MethodGet, "/admin/reviews?status=all", "")
	h.ListAllReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reviewID)

	// AdminUpdateReview success
	w, c = withAdminContext(http.MethodPut, "/admin/reviews/"+reviewID, `{"action":"hide","reason":"spam"}`)
	c.Params = gin.Params{{Key: "reviewID", Value: reviewID}}
	h.AdminUpdateReview(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var status string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, reviewID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, StatusHidden, status)

	// BatchUpdateReviews success
	w, c = withAdminContext(http.MethodPatch, "/admin/reviews/batch", `{"ids":["`+reviewID2+`"],"action":"hide"}`)
	h.BatchUpdateReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"affected":1`)

	// Sensitive word CRUD handlers
	w, c = withAdminContext(http.MethodPost, "/admin/sensitive-words", `{"word":"测试敏感词","category":"custom","level":"warn"}`)
	h.CreateSensitiveWord(c)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "测试敏感词")

	var sensitiveWordID string
	err = fixture.Pool.QueryRow(ctx, `SELECT id FROM sensitive_words WHERE word = $1`, "测试敏感词").Scan(&sensitiveWordID)
	require.NoError(t, err)

	w, c = withAdminContext(http.MethodGet, "/admin/sensitive-words?category=custom", "")
	h.ListSensitiveWords(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), sensitiveWordID)

	w, c = withAdminContext(http.MethodPut, "/admin/sensitive-words/"+sensitiveWordID, `{"word":"更新敏感词","level":"review","isActive":true}`)
	c.Params = gin.Params{{Key: "sensitiveWordID", Value: sensitiveWordID}}
	h.UpdateSensitiveWord(c)
	assert.Equal(t, http.StatusOK, w.Code)

	w, c = withAdminContext(http.MethodDelete, "/admin/sensitive-words/"+sensitiveWordID, "")
	c.Params = gin.Params{{Key: "sensitiveWordID", Value: sensitiveWordID}}
	h.DeleteSensitiveWord(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Teacher admin handlers
	w, c = withAdminContext(http.MethodGet, "/admin/teachers?search=孙", "")
	h.ListAdminTeachers(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "孙老师")

	w, c = withAdminContext(http.MethodPost, "/admin/teachers", `{"name":"新建教师","departmentID":`+strconv.FormatInt(departmentID, 10)+`}`)
	h.CreateTeacher(c)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "新建教师")

	var createdTeacherID int64
	err = fixture.Pool.QueryRow(ctx, `SELECT id FROM teachers WHERE name = $1`, "新建教师").Scan(&createdTeacherID)
	require.NoError(t, err)

	w, c = withAdminContext(http.MethodPut, "/admin/teachers/1", `{"name":"已更新教师","departmentID":`+strconv.FormatInt(departmentID, 10)+`}`)
	c.Params = gin.Params{{Key: "teacherID", Value: ""}}
	c.Params = gin.Params{{Key: "teacherID", Value: strconv.FormatInt(createdTeacherID, 10)}}
	h.UpdateTeacher(c)
	assert.Equal(t, http.StatusOK, w.Code)

	w, c = withAdminContext(http.MethodDelete, "/admin/teachers/1", "")
	c.Params = gin.Params{{Key: "teacherID", Value: strconv.FormatInt(createdTeacherID, 10)}}
	h.DeleteTeacher(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Content flag handlers
	w, c = withAdminContext(http.MethodGet, "/admin/content-flags", "")
	h.ListFlaggedReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), flaggedID)

	w, c = withAdminContext(http.MethodPut, "/admin/content-flags/"+flaggedID+"/clear", "")
	c.Params = gin.Params{{Key: "reviewID", Value: flaggedID}}
	h.ClearContentFlag(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Process report through handler
	var handlerReportID string
	handlerReportID, err = svc.ReportReview(ctx, ReportReviewParams{ReviewID: reportableID, UserHash: "u-handler-reporter", ReporterExternalUserID: "ext-handler-reporter", Reason: "spam", Description: "handler report"})
	require.NoError(t, err)
	w, c = withAdminContext(http.MethodPut, "/admin/reports/"+handlerReportID, `{"action":"reject","note":"handled"}`)
	c.Params = gin.Params{{Key: "reportID", Value: handlerReportID}}
	h.ProcessReport(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Admin edit review content through handler
	w, c = withAdminContext(http.MethodPost, "/admin/reviews/"+flaggedID+"/edit", `{"title":"处理后标题","content":"处理后内容","reason":"规范化"}`)
	c.Params = gin.Params{{Key: "reviewID", Value: flaggedID}}
	h.AdminEditReviewContent(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Operation log handler should now have entries from admin ops above
	w, c = withAdminContext(http.MethodGet, "/admin/logs", "")
	h.GetOperationLogs(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "create_teacher")
}
