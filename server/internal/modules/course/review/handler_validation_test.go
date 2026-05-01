package review

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

type errorAuthorizationProvider struct{}

func (errorAuthorizationProvider) Check(_ context.Context, _, _, _ string) (bool, error) {
	return false, errors.New("boom")
}
func (errorAuthorizationProvider) WriteReviewRelations(context.Context, string, string, string, string) error {
	return nil
}
func (errorAuthorizationProvider) WriteReportRelations(context.Context, string, string, string, string) error {
	return nil
}

func TestReviewHandlerValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{fga: NewFailClosedAuthorizationProvider()}

	cases := []struct {
		name string
		run  func(*Handler, *gin.Context)
		url  string
		prep func(*gin.Context)
	}{
		{name: "GetCourseReviews invalid course id", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourseReviews(c) }},
		{name: "GetCourseReviews invalid term", url: "/?termID=2024-3", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourseReviews(c) }},
		{name: "GetCourseReviews invalid teacher", url: "/?teacherID=bad", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourseReviews(c) }},
		{name: "SearchReviews empty conditions", url: "/", run: func(h *Handler, c *gin.Context) { h.SearchReviews(c) }},
		{name: "SearchReviews invalid term", url: "/?q=math&termID=2024-3", run: func(h *Handler, c *gin.Context) { h.SearchReviews(c) }},
		{name: "SearchReviews invalid department", url: "/?q=math&departmentID=bad", run: func(h *Handler, c *gin.Context) { h.SearchReviews(c) }},
		{name: "GetBatchCourseReviews missing ids", url: "/", run: func(h *Handler, c *gin.Context) { h.GetBatchCourseReviews(c) }},
		{name: "GetBatchCourseReviews too many ids", url: "/?courseIDs=" + strings.TrimSuffix(strings.Repeat("1,", 21), ","), run: func(h *Handler, c *gin.Context) { h.GetBatchCourseReviews(c) }},
		{name: "GetBatchCourseReviews invalid id", url: "/?courseIDs=1,bad", run: func(h *Handler, c *gin.Context) { h.GetBatchCourseReviews(c) }},
		{name: "AddFavorite invalid course id", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.AddFavorite(c) }},
		{name: "GetFavoriteStatus invalid course id", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.GetFavoriteStatus(c) }},
		{name: "RemoveFavorite invalid course id", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.RemoveFavorite(c) }},
		{name: "ProcessReport invalid report id", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "reportID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.ProcessReport(c) }},
		{name: "AdminUpdateReview invalid review id", url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "reviewID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.AdminUpdateReview(c) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tc.url, nil)
			if tc.prep != nil {
				tc.prep(c)
			}
			tc.run(h, c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestReviewHandler_AdminUpdateReviewForbiddenOnFailClosedFGA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{fga: NewFailClosedAuthorizationProvider()}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"action":"hide","reason":"spam"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
	c.Set("user_id", "user-1")

	h.AdminUpdateReview(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func newFactAwareHandler(canPost, canEdit, canDelete bool) *Handler {
	systemconfig.SetReviewAccessPolicySnapshot(systemconfig.ReviewAccessPolicySnapshot{
		AllowedSchoolIDs:    []string{"1001"},
		PreviewTitleRunes:   24,
		PreviewContentRunes: 120,
		PreviewContentPct:   100,
		LoadedAt:            time.Now(),
	})
	subjectSchoolID := int64(1001)
	return &Handler{
		service: &Service{accessReader: fakeAccessReader{subject: &reviewaccess.Subject{
			InternalUserID:   42,
			SchoolID:         &subjectSchoolID,
			StudentVerified:  true,
			IdentityVerified: true,
		}}},
		fga: NewFailClosedAuthorizationProvider(),
	}
}

func TestReviewHandler_MoreValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	t.Run("public read handlers reject invalid ids", func(t *testing.T) {
		h := &Handler{}
		cases := []struct {
			name string
			prep func(*gin.Context)
			run  func(*Handler, *gin.Context)
		}{
			{name: "GetRatingTrend", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.GetRatingTrend(c) }},
			{name: "GetCourseTeachers", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.GetCourseTeachers(c) }},
			{name: "GetTeacherRatingStats", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "teacherID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.GetTeacherRatingStats(c) }},
			{name: "GetReplies", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "reviewID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.GetReplies(c) }},
			{name: "DeleteReply", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "replyID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.DeleteReply(c) }},
			{name: "ClearContentFlag", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "reviewID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.ClearContentFlag(c) }},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				tc.prep(c)
				tc.run(h, c)
				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})

	t.Run("teacher handlers reject invalid params", func(t *testing.T) {
		h := &Handler{}
		cases := []struct {
			name   string
			method string
			url    string
			body   string
			prep   func(*gin.Context)
			run    func(*Handler, *gin.Context)
		}{
			{name: "ListTeachers invalid dept", method: http.MethodGet, url: "/?departmentID=bad", run: func(h *Handler, c *gin.Context) { h.ListTeachers(c) }},
			{name: "ListHotTeachers invalid limit", method: http.MethodGet, url: "/?limit=bad", run: func(h *Handler, c *gin.Context) { h.ListHotTeachers(c) }},
			{name: "ListAdminTeachers invalid dept", method: http.MethodGet, url: "/?departmentID=bad", run: func(h *Handler, c *gin.Context) { h.ListAdminTeachers(c) }},
			{name: "CreateTeacher invalid body", method: http.MethodPost, url: "/", body: `{}`, run: func(h *Handler, c *gin.Context) { h.CreateTeacher(c) }},
			{name: "UpdateTeacher invalid id", method: http.MethodPut, url: "/", body: `{"name":"x"}`, prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "teacherID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.UpdateTeacher(c) }},
			{name: "DeleteTeacher invalid id", method: http.MethodDelete, url: "/", prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "teacherID", Value: "bad"}} }, run: func(h *Handler, c *gin.Context) { h.DeleteTeacher(c) }},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				var body io.Reader
				if tc.body != "" {
					body = strings.NewReader(tc.body)
				}
				c.Request = httptest.NewRequest(tc.method, tc.url, body)
				if tc.body != "" {
					c.Request.Header.Set("Content-Type", "application/json")
				}
				if tc.prep != nil {
					tc.prep(c)
				}
				tc.run(h, c)
				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})

	t.Run("write handlers fail closed before service", func(t *testing.T) {
		hPost := newFactAwareHandler(false, false, false)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"courseID":1,"termID":"2024-1","title":"title","content":"0123456789","ratings":{"overall":5}}`))
		c.Request.Header.Set("Content-Type", "application/json")
		hPost.PostReview(c)
		assert.Equal(t, http.StatusForbidden, w.Code)

		hEdit := newFactAwareHandler(false, false, false)
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"content":"0123456789","ratings":{"overall":5}}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
		hEdit.UpdateReview(c)
		assert.Equal(t, http.StatusForbidden, w.Code)

		hDelete := newFactAwareHandler(false, false, false)
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
		c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
		hDelete.DeleteReview(c)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("bind failures still return bad request on valid gated paths", func(t *testing.T) {
		h := newFactAwareHandler(true, true, true)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reason":"bad_reason"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
		h.ReportReview(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
		h.CreateReply(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.CheckContent(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("batch update rejects malformed uuid", func(t *testing.T) {
		h := &Handler{fga: NewFailClosedAuthorizationProvider()}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ids":["bad-uuid"],"action":"hide"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.BatchUpdateReviews(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("batch hide of five reviews requires step-up proof", func(t *testing.T) {
		h := &Handler{fga: NewFailClosedAuthorizationProvider()}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(batchStepUpRequestBody("hide")))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.CtxKeyUserID, "moderator-1")
		middleware.SetMFAContext(c, middleware.MFAContext{EnrollmentActive: true})

		h.BatchUpdateReviews(c)

		assert.Equal(t, http.StatusPreconditionRequired, w.Code)
	})

	t.Run("checkFGA returns false on provider error", func(t *testing.T) {
		h := &Handler{fga: errorAuthorizationProvider{}}
		assert.False(t, h.checkFGA(context.Background(), "user:1", "can_hide", "review:1"))
	})
}

func batchStepUpRequestBody(action string) string {
	return `{"ids":[` +
		`"550e8400-e29b-41d4-a716-446655440001",` +
		`"550e8400-e29b-41d4-a716-446655440002",` +
		`"550e8400-e29b-41d4-a716-446655440003",` +
		`"550e8400-e29b-41d4-a716-446655440004",` +
		`"550e8400-e29b-41d4-a716-446655440005"` +
		`],"action":"` + action + `"}`
}

func TestBatchReviewStepUpRequired(t *testing.T) {
	assert.True(t, batchReviewStepUpRequired(BatchUpdateReviewsRequest{
		IDs:    make([]string, batchStepUpThreshold),
		Action: "hide",
	}))
	assert.True(t, batchReviewStepUpRequired(BatchUpdateReviewsRequest{
		IDs:    make([]string, batchStepUpThreshold),
		Action: "delete",
	}))
	assert.False(t, batchReviewStepUpRequired(BatchUpdateReviewsRequest{
		IDs:    make([]string, batchStepUpThreshold),
		Action: "restore",
	}))
	assert.False(t, batchReviewStepUpRequired(BatchUpdateReviewsRequest{
		IDs:    make([]string, batchStepUpThreshold-1),
		Action: "hide",
	}))
}
