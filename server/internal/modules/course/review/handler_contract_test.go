package review

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// contractResponse 用于解析统一响应格式
type contractResponse struct {
	Success bool             `json:"success"`
	Error   *response.APIError `json:"error,omitempty"`
}

// setupContractRedis 创建 miniredis 实例和 Redis 客户端
func setupContractRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	return rdb, mr
}

// setupContractRouter 创建带中间件的测试路由
// 注意：不注入真实 DB 和 SSO client，仅测试中间件层的 HTTP 契约
func setupContractRouter(t *testing.T, rdb *redis.Client, tokenSvc *token.Service) *gin.Engine {
	t.Helper()
	r := gin.New()

	authMW := middleware.AuthMiddleware(tokenSvc)
	csrfMW := middleware.CSRFMiddleware()

	limiter := middleware.NewRedisRateLimiter(rdb, 2, time.Minute) // PostLimit=2

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

		api.POST("/reviews/test-id/vote", authMW,
			middleware.EndpointRateLimitMiddleware(limiter, "vote"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})
	}

	return r
}

// TestContract_401_NoToken 未携带 token 访问受保护端点返回 401
func TestContract_401_NoToken(t *testing.T) {
	rdb, _ := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(rdb)
	r := setupContractRouter(t, rdb, tokenSvc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/course/review/reviews"},
		{http.MethodPost, "/api/v1/course/review/reviews/test-id/vote"},
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
	rdb, _ := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(rdb)
	r := setupContractRouter(t, rdb, tokenSvc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/course/review/reviews", nil)
	// 设置有效的 Bearer token（绕过 auth 中间件）
	req.Header.Set("Authorization", "Bearer "+tokenSvc.IssueTestToken(t))
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
	rdb, _ := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(rdb)
	r := setupContractRouter(t, rdb, tokenSvc)

	validToken := tokenSvc.IssueTestToken(t)
	csrfToken := "test-csrf-token"

	// 发送请求直到触发限流（PostLimit=2）
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/course/review/reviews", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("X-CSRF-Token", csrfToken)
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
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
	rdb, _ := setupContractRedis(t)
	tokenSvc := token.NewServiceForTest(rdb)
	r := setupContractRouter(t, rdb, tokenSvc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/review/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp contractResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}
