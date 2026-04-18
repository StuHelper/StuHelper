package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

func TestRespondPostReviewError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		status   int
		code     errs.ErrorCode
		contains string
	}{
		{name: "already reviewed", err: ErrAlreadyReviewed, status: http.StatusConflict, code: errs.ErrReviewExists, contains: "already reviewed"},
		{name: "invalid rating", err: ErrInvalidRating, status: http.StatusBadRequest, code: errs.ErrBadRequest, contains: "rating must be between 1 and 5"},
		{name: "sensitive content", err: ErrSensitiveContent, status: http.StatusBadRequest, code: errs.ErrSensitiveContent, contains: "sensitive words"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := respondPostReviewError(c, tt.err)

			assert.True(t, ok)
			assert.Equal(t, tt.status, w.Code)
			assert.Contains(t, w.Body.String(), string(tt.code))
			assert.Contains(t, w.Body.String(), tt.contains)
		})
	}
}

func TestRespondAdminUpdateReviewError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := respondAdminUpdateReviewError(c, ErrInvalidTransition)

	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrInvalidTransition))
	assert.Contains(t, w.Body.String(), "invalid status transition")
}
