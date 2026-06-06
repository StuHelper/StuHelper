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
		{name: "invalid term", err: ErrInvalidTermID, status: http.StatusBadRequest, code: errs.ErrBadRequest, contains: "invalid term_id format"},
		{name: "invalid rating", err: ErrInvalidRating, status: http.StatusBadRequest, code: errs.ErrBadRequest, contains: "rating must be between 1 and 5"},
		{name: "invalid grade", err: ErrInvalidGrade, status: http.StatusBadRequest, code: errs.ErrBadRequest, contains: "invalid grade"},
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

func TestRespondSaveDraftError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		status   int
		contains string
	}{
		{name: "invalid term", err: ErrInvalidTermID, status: http.StatusBadRequest, contains: "invalid term_id format"},
		{name: "invalid rating", err: ErrInvalidRating, status: http.StatusBadRequest, contains: "rating must be between 1 and 5"},
		{name: "invalid grade", err: ErrInvalidGrade, status: http.StatusBadRequest, contains: "invalid grade"},
		{name: "content empty", err: ErrContentEmpty, status: http.StatusBadRequest, contains: "content cannot be empty"},
		{name: "content too long", err: ErrContentTooLong, status: http.StatusBadRequest, contains: "content is too long"},
		{name: "teacher not found", err: ErrTeacherNotFound, status: http.StatusNotFound, contains: "teacher not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := respondSaveDraftError(c, tt.err)

			assert.True(t, ok)
			assert.Equal(t, tt.status, w.Code)
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

func TestRespondTeacherAdminError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		status   int
		contains string
	}{
		{name: "missing department", err: ErrTeacherDepartmentRequired, status: http.StatusBadRequest, contains: "teacher department is required"},
		{name: "department not found", err: ErrTeacherDepartmentNotFound, status: http.StatusNotFound, contains: "teacher department not found"},
		{name: "has reviews", err: ErrTeacherHasReviews, status: http.StatusConflict, contains: "teacher has associated reviews"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := respondTeacherAdminError(c, tt.err)

			assert.True(t, ok)
			assert.Equal(t, tt.status, w.Code)
			assert.Contains(t, w.Body.String(), tt.contains)
		})
	}
}

func TestRespondSensitiveWordAdminErrorValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := respondSensitiveWordAdminError(c, ErrSensitiveWordInvalid)

	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "sensitive word input is invalid")
}

func TestRespondVoteReviewErrorInvalidVoteType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := respondVoteReviewError(c, ErrInvalidAction)

	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid vote type")
}

func TestRespondReportReviewErrorValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{name: "invalid reason", err: ErrInvalidAction, contains: "invalid report reason"},
		{name: "dangerous description", err: ErrDangerousContent, contains: "potentially dangerous"},
		{name: "description too long", err: ErrContentTooLong, contains: "content is too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := respondReportReviewError(c, tt.err)

			assert.True(t, ok)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.contains)
		})
	}
}
