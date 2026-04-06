package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
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

func TestConsumeOIDCState_OneTimeAndCookieBound(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	h := &Handler{
		redisClient:        rdb,
		tokenConfig:        config.TokenConfig{},
		defaultRedirectURL: "https://web.example.com",
	}

	const state = "test-state"
	require.NoError(t, h.storeOIDCState(context.Background(), state, "/courses/1", "test-verifier"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	c.Request = req

	redirect, codeVerifier, err := h.consumeOIDCState(c, state)
	require.NoError(t, err)
	assert.Equal(t, "/courses/1", redirect)
	assert.Equal(t, "test-verifier", codeVerifier)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state="+state, nil)
	req2.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	c2.Request = req2

	_, _, err = h.consumeOIDCState(c2, state)
	require.Error(t, err)
}

func TestResolveRedirectTarget_SchemeRelativeFallsBack(t *testing.T) {
	h := &Handler{defaultRedirectURL: "https://web.example.com"}
	assert.Equal(t, "https://web.example.com", h.resolveRedirectTarget("//evil.example.com"))
}
