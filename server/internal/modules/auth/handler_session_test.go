package auth

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

func TestLogout_ReceivesRefreshCookieAndRevokesCurrentSession(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-logout-secret-32-chars-long!", false))

	h, _ := newTestHandler(t)
	h.tokenConfig = config.TokenConfig{
		AccessTokenTTL:  3600,
		RefreshTokenTTL: 3600,
	}

	tokenSvc, err := token.NewService(token.ServiceConfig{
		RedisClient: h.redisClient,
		AccessTTL:   3600,
		RefreshTTL:  3600,
	})
	require.NoError(t, err)
	h.tokenService = tokenSvc
	h.svc = &Service{tokenService: tokenSvc}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-123")
		c.Set(middleware.CtxKeyUsername, "tester")
		c.Next()
	})
	r.GET("/api/v1/auth/issue", func(c *gin.Context) {
		if err := h.setTokenCookies(c, "access-token", "refresh-token", "sid-logout"); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	r.POST("/api/v1/auth/logout", h.Logout)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := srv.Client()
	client.Jar = jar

	issueResp, err := client.Get(srv.URL + "/api/v1/auth/issue")
	require.NoError(t, err)
	require.NoError(t, issueResp.Body.Close())
	assert.Equal(t, http.StatusNoContent, issueResp.StatusCode)

	logoutReq, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/logout", nil)
	require.NoError(t, err)
	cookies := client.Jar.Cookies(logoutReq.URL)

	assertCookieValue(t, cookies, middleware.CookieAccessToken, "access-token")
	assertCookieValue(t, cookies, middleware.CookieRefreshToken, "refresh-token")

	resp, err := client.Do(logoutReq)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	accessBlacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(context.Background(), "access-token")
	require.NoError(t, err)
	assert.True(t, accessBlacklisted)

	refreshBlacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(context.Background(), "refresh-token")
	require.NoError(t, err)
	assert.True(t, refreshBlacklisted)
}

func assertCookieValue(t *testing.T, cookies []*http.Cookie, name, want string) {
	t.Helper()

	for _, c := range cookies {
		if c.Name == name {
			assert.Equal(t, want, c.Value)
			return
		}
	}
	t.Fatalf("cookie %q not found", name)
}
