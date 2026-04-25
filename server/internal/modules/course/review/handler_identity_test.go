package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestReviewHandler_IdentityRequiredBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	cases := []struct {
		name string
		run  func(*Handler, *gin.Context)
		prep func(*gin.Context)
		body string
	}{
		{
			name: "AddFavorite requires auth",
			prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} },
			run:  func(h *Handler, c *gin.Context) { h.AddFavorite(c) },
		},
		{
			name: "GetFavoriteStatus requires auth",
			prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} },
			run:  func(h *Handler, c *gin.Context) { h.GetFavoriteStatus(c) },
		},
		{
			name: "RemoveFavorite requires auth",
			prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} },
			run:  func(h *Handler, c *gin.Context) { h.RemoveFavorite(c) },
		},
		{
			name: "GetUserFavorites requires auth",
			run:  func(h *Handler, c *gin.Context) { h.GetUserFavorites(c) },
		},
		{
			name: "GetUserReviews requires auth",
			run:  func(h *Handler, c *gin.Context) { h.GetUserReviews(c) },
		},
		{
			name: "GetUserVotes requires auth",
			run:  func(h *Handler, c *gin.Context) { h.GetUserVotes(c) },
		},
		{
			name: "SaveDraft requires auth",
			body: `{"courseID":1,"title":"draft","content":"draft"}`,
			run:  func(h *Handler, c *gin.Context) { h.SaveDraft(c) },
		},
		{
			name: "GetDraft requires auth",
			prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} },
			run:  func(h *Handler, c *gin.Context) { h.GetDraft(c) },
		},
		{
			name: "DeleteDraft requires auth",
			prep: func(c *gin.Context) { c.Params = gin.Params{{Key: "courseID", Value: "1"}} },
			run:  func(h *Handler, c *gin.Context) { h.DeleteDraft(c) },
		},
		{
			name: "CreateReply requires auth",
			body: `{"content":"reply content"}`,
			prep: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
			},
			run: func(h *Handler, c *gin.Context) { h.CreateReply(c) },
		},
		{
			name: "DeleteReply requires auth",
			prep: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "replyID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
			},
			run: func(h *Handler, c *gin.Context) { h.DeleteReply(c) },
		},
		{
			name: "ReportReview requires auth",
			body: `{"reason":"spam","description":"dup"}`,
			prep: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
			},
			run: func(h *Handler, c *gin.Context) { h.ReportReview(c) },
		},
		{
			name: "VoteReview requires auth",
			body: `{"voteType":"like"}`,
			prep: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "reviewID", Value: "550e8400-e29b-41d4-a716-446655440000"}}
			},
			run: func(h *Handler, c *gin.Context) { h.VoteReview(c) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, c := withUserContext(http.MethodPost, "/", tc.body, "")
			if tc.body == "" {
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			}
			if tc.prep != nil {
				tc.prep(c)
			}
			tc.run(h, c)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "missing authentication token")
		})
	}
}
