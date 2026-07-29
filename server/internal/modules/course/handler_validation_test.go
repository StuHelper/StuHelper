package course

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

func TestCourseHandlerValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	t.Run("GetCourses rejects invalid departmentID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/course/courses?departmentID=bad", nil)
		h.GetCourses(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetCourses rejects too long query", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/course/courses?q="+repeatString("a", maxSearchLength+1), nil)
		h.GetCourses(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("SearchCourses rejects empty query", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/course/courses/search", nil)
		h.SearchCourses(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("SearchCourses rejects too long query", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/course/courses/search?q="+repeatString("a", maxSearchLength+1), nil)
		h.SearchCourses(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetCourse rejects invalid course id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "courseID", Value: "bad"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/course/courses/bad", nil)
		h.GetCourse(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCourseResolveUserHashAndNoopFavoriteAnnotation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Empty(t, h.resolveUserHash(c))

	require.NoError(t, crypto.InitHMACKey("test-course-hash-secret-32-bytes!!", false))
	c.Set(middleware.CtxKeyUserID, "user-123")
	assert.NotEmpty(t, h.resolveUserHash(c))

	courses := []Course{}
	assert.NotPanics(t, func() { h.annotateFavorites(c, "hash", courses) })
}

func TestNewHandlerRequiresDeps(t *testing.T) {
	assert.Panics(t, func() { NewHandler(nil, nil) })
}

func repeatString(ch string, n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = ch[0]
	}
	return string(buf)
}
