package review

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestReviewRepliesRouteUsesOptionalAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, crypto.InitHMACKey("test-review-reply-route-secret-32b", false))

	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(
		fixture.DB,
		repo,
		noopNotificationSender{},
		noopReviewFGAWriter{},
		failClosedReviewAccessReader{},
	)

	departmentID := seedDepartment(t, fixture, 4111010006, "回复路由测试学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "回复路由测试教师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "回复路由测试课程")
	reviewID := "550e8400-e29b-41d4-a716-446655440731"
	seedReviewWithRatings(
		t,
		fixture,
		reviewID,
		courseID,
		teacherID,
		"review-reply-route-author",
		4.5,
		StatusPublished,
		ReviewRatings{"teaching": 5},
		"回复路由测试",
		"验证真实路由会解析可选身份",
	)

	ownerID := "reply-route-owner"
	ownerHash, err := httputil.HashUserID(ownerID)
	require.NoError(t, err)
	otherHash, err := httputil.HashUserID("reply-route-other-author")
	require.NoError(t, err)
	ownerReply, err := svc.CreateReply(context.Background(), CreateReplyParams{
		ReviewID: reviewID,
		UserHash: ownerHash,
		Content:  "当前用户自己的回复",
	})
	require.NoError(t, err)
	otherReply, err := svc.CreateReply(context.Background(), CreateReplyParams{
		ReviewID: reviewID,
		UserHash: otherHash,
		Content:  "其他用户的回复",
	})
	require.NoError(t, err)

	optionalAuthCalls := 0
	optionalAuth := func(c *gin.Context) {
		optionalAuthCalls++
		if c.GetHeader("X-Test-Auth-Backend-Failure") == "true" {
			c.Set(middleware.CtxKeyAuthBackendFailure, true)
		}
		if userID := c.GetHeader("X-Test-User-ID"); userID != "" {
			c.Set(middleware.CtxKeyUserID, userID)
		}
		c.Next()
	}
	noOpAuth := func(c *gin.Context) { c.Next() }
	handler := &Handler{service: svc}
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1/course/review"), noOpAuth, optionalAuth)

	assertOwners := func(t *testing.T, userID string, want map[string]bool) {
		t.Helper()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/course/review/reviews/"+reviewID+"/replies",
			nil,
		)
		if userID != "" {
			request.Header.Set("X-Test-User-ID", userID)
		}
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

		var payload struct {
			Data struct {
				List []struct {
					ID      string `json:"id"`
					IsOwner bool   `json:"isOwner"`
				} `json:"list"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
		require.Len(t, payload.Data.List, len(want))
		for _, reply := range payload.Data.List {
			wantOwner, exists := want[reply.ID]
			require.True(t, exists, "unexpected reply %s", reply.ID)
			assert.Equal(t, wantOwner, reply.IsOwner, reply.ID)
		}
	}

	t.Run("owner sees only their reply as owned", func(t *testing.T) {
		assertOwners(t, ownerID, map[string]bool{
			ownerReply.Reply.ID: true,
			otherReply.Reply.ID: false,
		})
	})
	t.Run("different authenticated user owns neither reply", func(t *testing.T) {
		assertOwners(t, "reply-route-non-owner", map[string]bool{
			ownerReply.Reply.ID: false,
			otherReply.Reply.ID: false,
		})
	})
	t.Run("anonymous user owns neither reply", func(t *testing.T) {
		assertOwners(t, "", map[string]bool{
			ownerReply.Reply.ID: false,
			otherReply.Reply.ID: false,
		})
	})
	t.Run("optional authentication backend failure fails closed", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/course/review/reviews/"+reviewID+"/replies",
			nil,
		)
		request.Header.Set("X-Test-Auth-Backend-Failure", "true")
		router.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "authentication service temporarily unavailable")
	})

	assert.Equal(t, 4, optionalAuthCalls)
}
