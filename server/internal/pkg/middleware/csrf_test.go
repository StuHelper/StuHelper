package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateCSRFToken(t *testing.T) {
	token1, err := GenerateCSRFToken()
	assert.NoError(t, err)
	assert.Len(t, token1, 43) // base64 编码的 32 字节

	token2, err := GenerateCSRFToken()
	assert.NoError(t, err)
	assert.NotEqual(t, token1, token2, "每次生成的 token 应该不同")
}

func TestCSRFMiddleware_AllowsSafeMethod(t *testing.T) {
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(CSRFMiddleware())
			r.Handle(method, "/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request = httptest.NewRequest(method, "/test", nil)
			r.ServeHTTP(w, c.Request)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestCSRFMiddleware_BlocksWithoutToken(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
