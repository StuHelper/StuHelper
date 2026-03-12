package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetUserVotesRejectsInvalidVoteTypeFromVoteTypeQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/user/votes?voteType=meh", nil)
	c.Request = req

	h := &Handler{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetUserVotes should reject invalid voteType before reaching service call, got panic: %v", r)
		}
	}()

	h.GetUserVotes(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", w.Code, w.Body.String())
	}
}

func TestGetUserVotesRejectsLegacyVoteTypeQueryName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/user/votes?vote_type=dislike", nil)
	c.Request = req

	h := &Handler{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetUserVotes should reject legacy vote_type before reaching service call, got panic: %v", r)
		}
	}()

	h.GetUserVotes(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", w.Code, w.Body.String())
	}
}
