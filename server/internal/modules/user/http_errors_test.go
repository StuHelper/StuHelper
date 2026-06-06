package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

func TestRespondVerifyStudentError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		status   int
		code     errs.ErrorCode
		contains string
	}{
		{name: "school missing", err: ErrSchoolNotFound, status: http.StatusNotFound, code: errs.ErrProfileSchoolNotFound, contains: "school not found"},
		{name: "ldap failed", err: ErrLDAPFailed, status: http.StatusBadRequest, code: errs.ErrProfileLDAPFailed, contains: "LDAP verification failed"},
		{name: "student id invalid", err: ErrStudentIDInvalid, status: http.StatusBadRequest, code: errs.ErrBadRequest, contains: "student ID is invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ok := respondVerifyStudentError(c, tt.err)

			assert.True(t, ok)
			assert.Equal(t, tt.status, w.Code)
			assert.Contains(t, w.Body.String(), string(tt.code))
			assert.Contains(t, w.Body.String(), tt.contains)
		})
	}
}

func TestRespondAdminUpdateSystemConfigError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := respondAdminUpdateSystemConfigError(c, ErrSystemConfigNotFound)

	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrSystemConfigNotFound))
}
