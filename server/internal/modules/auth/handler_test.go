package auth

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

func TestGetLoginURL(t *testing.T) {
	// 创建测试路由
	r := gin.New()
	r.GET("/auth/login", func(c *gin.Context) {
		// 模拟返回登录 URL
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"url": "https://sso.example.com/login?client_id=test",
			},
		})
	})

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "url")
}

func TestGetSignupURL(t *testing.T) {
	r := gin.New()
	r.GET("/auth/signup", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"url": "https://sso.example.com/signup",
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/signup", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "url")
}
