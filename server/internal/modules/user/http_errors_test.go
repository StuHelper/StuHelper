package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
)

func TestRespondAdminUpdateSystemConfigError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := respondAdminUpdateSystemConfigError(c, ErrSystemConfigNotFound)

	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrSystemConfigNotFound))
}
