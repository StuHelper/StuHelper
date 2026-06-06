package review

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// contractResponse 用于解析统一响应格式
type contractResponse struct {
	Success bool               `json:"success"`
	Error   *response.APIError `json:"error,omitempty"`
}

// setupContractRedis 创建共享 Redis fixture。
func setupContractRedis(t *testing.T) *redisfixture.Fixture {
	t.Helper()
	return redisfixture.Start(t)
}

// setupContractRouter 创建带中间件的测试路由
// 注意：不注入真实 DB 和 SSO client，仅测试中间件层的 HTTP 契约
func setupContractRouter(t *testing.T, rdb *redisfixture.Fixture, tokenSvc *token.Service) *gin.Engine {
	t.Helper()
	r := gin.New()
	reviewID := "review-123"

	// 测试用 auth 中间件：只检查 Cookie 中有 token 且不在黑名单，不做 OIDC 验证
	authMW := func(c *gin.Context) {
		tokenString, err := c.Cookie(middleware.CookieAccessToken)
		if err != nil || tokenString == "" {
			response.Unauthorized(c, "missing authentication token", errs.ErrTokenMissing)
			c.Abort()
			return
		}
		isBlacklisted, blErr := tokenSvc.GetBlacklist().IsBlacklisted(c.Request.Context(), tokenString)
		if blErr != nil || isBlacklisted {
			response.Unauthorized(c, "token revoked", errs.ErrTokenRevoked)
			c.Abort()
			return
		}
		c.Set(middleware.CtxKeyUserID, "test-user-id")
		c.Set(middleware.CtxKeyUsername, "testuser")
		c.Next()
	}
	csrfMW := middleware.CSRFMiddleware()

	limiter := middleware.NewRedisRateLimiter(rdb.Client, 2, time.Minute) // PostLimit=2

	api := r.Group("/api/v1/course/review")
	{
		// 公开端点（无中间件）
		api.GET("/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"total": 0}})
		})

		// 需要认证的端点
		api.POST("/reviews", authMW, csrfMW,
			middleware.EndpointRateLimitMiddleware(limiter, "post-review"),
			func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"success": true})
			})

		api.POST("/reviews/"+reviewID+"/votes", authMW,
			middleware.EndpointRateLimitMiddleware(limiter, "vote"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})
	}

	return r
}

// TestContract_401_NoToken 未携带 token 访问受保护端点返回 401
func TestContract_401_NoToken(t *testing.T) {
	redisFixture := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(redisFixture.Client)
	r := setupContractRouter(t, redisFixture, tokenSvc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/course/review/reviews"},
		{http.MethodPost, fmt.Sprintf("/api/v1/course/review/reviews/%s/votes", "review-123")},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(ep.method, ep.path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var resp contractResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.False(t, resp.Success)
			assert.NotNil(t, resp.Error)
		})
	}
}

// TestContract_403_NoCSRF 缺少 CSRF token 的写请求返回 403
func TestContract_403_NoCSRF(t *testing.T) {
	redisFixture := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(redisFixture.Client)
	r := setupContractRouter(t, redisFixture, tokenSvc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/course/review/reviews", nil)
	// 设置有效的 Cookie token（绕过 auth 中间件的 token 检查）
	req.AddCookie(&http.Cookie{Name: middleware.CookieAccessToken, Value: "test-valid-token"})
	// 不设置 CSRF token → 应返回 403
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp contractResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// TestContract_429_RateLimited 超过限流阈值返回 429
func TestContract_429_RateLimited(t *testing.T) {
	redisFixture := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(redisFixture.Client)
	r := setupContractRouter(t, redisFixture, tokenSvc)

	validToken := "test-valid-token"
	sessionID := "sid-contract-rate-limited"
	require.NoError(t, crypto.InitHMACKey("test-review-contract-csrf-secret-32!", false))
	csrfToken, err := middleware.GenerateCSRFToken(sessionID)
	require.NoError(t, err)

	// 发送请求直到触发限流（PostLimit=2）
	for i := range 3 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/course/review/reviews", nil)
		req.AddCookie(&http.Cookie{Name: middleware.CookieAccessToken, Value: validToken})
		req.AddCookie(&http.Cookie{Name: middleware.CookieSessionID, Value: sessionID})
		req.Header.Set(middleware.CSRFHeaderName, csrfToken)
		req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: csrfToken})
		r.ServeHTTP(w, req)

		if i < 2 {
			// 前两次应该成功（201）
			assert.Equal(t, http.StatusCreated, w.Code, "request %d should succeed", i+1)
		} else {
			// 第三次应该被限流（429）
			assert.Equal(t, http.StatusTooManyRequests, w.Code, "request %d should be rate limited", i+1)

			var resp contractResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.False(t, resp.Success)
		}
	}
}

// TestContract_200_PublicEndpoint 公开端点无需认证返回 200
func TestContract_200_PublicEndpoint(t *testing.T) {
	redisFixture := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(redisFixture.Client)
	r := setupContractRouter(t, redisFixture, tokenSvc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp contractResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}
