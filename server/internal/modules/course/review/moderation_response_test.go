package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRespondToModerationError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "title empty", err: ErrTitleEmpty, status: http.StatusBadRequest},
		{name: "dangerous content", err: ErrDangerousContent, status: http.StatusBadRequest},
		{name: "sensitive content", err: ErrSensitiveContent, status: http.StatusBadRequest},
		{name: "moderation unavailable", err: ErrModerationUnavailable, status: http.StatusServiceUnavailable},
		{name: "content empty", err: ErrContentEmpty, status: http.StatusBadRequest},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			assert.True(t, respondToModerationError(c, tc.err))
			assert.Equal(t, tc.status, w.Code)
		})
	}

	t.Run("unhandled", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		assert.False(t, respondToModerationError(c, ErrReviewNotFound))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
