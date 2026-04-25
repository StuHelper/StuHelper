package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

func TestRespondMappedError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testErr := errors.New("boom")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := RespondMappedError(c, testErr,
		MatchError(testErr, http.StatusConflict, "already exists", errs.ErrConflict),
	)

	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already exists")
	assert.Contains(t, w.Body.String(), string(errs.ErrConflict))
}

func TestRespondMappedError_NoMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := RespondMappedError(c, errors.New("boom"),
		MatchError(errors.New("other"), http.StatusConflict, "already exists", errs.ErrConflict),
	)

	assert.False(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRespondMappedErrorGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	target := errors.New("target")
	ok := RespondMappedErrorGroups(c, target,
		[]ErrorMapping{MatchError(errors.New("other"), http.StatusBadRequest, "other")},
		[]ErrorMapping{MatchError(target, http.StatusConflict, "conflict", errs.ErrConflict)},
	)

	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrConflict))
}
