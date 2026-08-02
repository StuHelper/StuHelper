package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
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
		{name: "invalid rating", err: ErrInvalidRating, status: http.StatusBadRequest, code: errs.ErrRatingInvalid, contains: "rating must be between 1 and 5"},
		{name: "missing rating", err: ErrRatingRequired, status: http.StatusBadRequest, code: errs.ErrRatingDimensionMissing, contains: "at least one rating dimension"},
		{name: "content too short", err: ErrContentTooShort, status: http.StatusBadRequest, code: errs.ErrReviewContentTooShort, contains: "content is too short"},
		{name: "content too long", err: ErrContentTooLong, status: http.StatusBadRequest, code: errs.ErrReviewContentTooLong, contains: "content is too long"},
		{name: "dangerous content", err: ErrDangerousContent, status: http.StatusBadRequest, code: errs.ErrDangerousContent, contains: "potentially dangerous"},
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

func TestRespondReviewWriteAccessDeniedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		run  func(*gin.Context) bool
	}{
		{name: "post review", run: func(c *gin.Context) bool { return respondPostReviewError(c, ErrReviewWriteAccessDenied) }},
		{name: "update review", run: func(c *gin.Context) bool { return respondUpdateReviewError(c, ErrReviewWriteAccessDenied) }},
		{name: "delete review", run: func(c *gin.Context) bool { return respondDeleteReviewError(c, ErrReviewWriteAccessDenied) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := tt.run(c)

			assert.True(t, ok)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), string(errs.ErrAccessDenied))
			assert.Contains(t, w.Body.String(), "review write access denied")
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

func TestRespondProcessReportErrorMapsMissingReview(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := respondProcessReportError(c, ErrReviewNotFound)

	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrReviewNotFound))
	assert.Contains(t, w.Body.String(), "review not found")
}

func TestRespondAdminErrorsMapMissingAdminIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		run  func(*gin.Context) bool
	}{
		{name: "process report", run: func(c *gin.Context) bool { return respondProcessReportError(c, ErrAdminIdentityRequired) }},
		{name: "update review", run: func(c *gin.Context) bool { return respondAdminUpdateReviewError(c, ErrAdminIdentityRequired) }},
		{name: "edit review", run: func(c *gin.Context) bool { return respondAdminEditReviewError(c, ErrAdminIdentityRequired) }},
		{name: "clear content flag", run: func(c *gin.Context) bool { return respondClearContentFlagError(c, ErrAdminIdentityRequired) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := tt.run(c)

			assert.True(t, ok)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), string(errs.ErrTokenMissing))
			assert.Contains(t, w.Body.String(), "missing authentication token")
		})
	}
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
		{name: "description too long", err: ErrReportDescriptionTooLong, contains: "report description is too long"},
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

func TestNonReviewContentLimitsKeepGenericCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		err  error
		run  func(*gin.Context, error) bool
	}{
		{name: "reply", err: ErrReplyContentTooLong, run: respondCreateReplyError},
		{name: "report", err: ErrReportDescriptionTooLong, run: respondReportReviewError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := tt.run(c, tt.err)

			assert.True(t, ok)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), string(errs.ErrParamOutOfRange))
			assert.NotContains(t, w.Body.String(), string(errs.ErrReviewContentTooLong))
		})
	}
}
