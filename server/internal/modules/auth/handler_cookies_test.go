package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestSetTokenCookies(t *testing.T) {
	h, _ := newTestHandler(t)
	h.tokenConfig = config.TokenConfig{
		AccessTokenTTL:  300,
		RefreshTokenTTL: 600,
		CookieSecure:    true,
		CookieDomain:    "example.com",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := h.setTokenCookies(c, "access-token", "refresh-token")
	require.NoError(t, err)

	cookies := cookieMap(w.Result().Cookies())
	require.Contains(t, cookies, middleware.CookieAccessToken)
	require.Contains(t, cookies, middleware.CookieRefreshToken)
	require.Contains(t, cookies, middleware.CSRFCookieName)

	assert.Equal(t, "access-token", cookies[middleware.CookieAccessToken].Value)
	assert.Equal(t, "/", cookies[middleware.CookieAccessToken].Path)
	assert.True(t, cookies[middleware.CookieAccessToken].HttpOnly)
	assert.True(t, cookies[middleware.CookieAccessToken].Secure)
	assert.Equal(t, "example.com", cookies[middleware.CookieAccessToken].Domain)
	assert.Equal(t, 300, cookies[middleware.CookieAccessToken].MaxAge)

	assert.Equal(t, "refresh-token", cookies[middleware.CookieRefreshToken].Value)
	assert.Equal(t, middleware.CookieRefreshTokenPath, cookies[middleware.CookieRefreshToken].Path)
	assert.True(t, cookies[middleware.CookieRefreshToken].HttpOnly)
	assert.Equal(t, 600, cookies[middleware.CookieRefreshToken].MaxAge)

	assert.False(t, cookies[middleware.CSRFCookieName].HttpOnly)
	assert.NotEmpty(t, cookies[middleware.CSRFCookieName].Value)
	assert.Equal(t, cookies[middleware.CSRFCookieName].Value, w.Header().Get(middleware.CSRFHeaderName))
}

func TestClearTokenCookies(t *testing.T) {
	h, _ := newTestHandler(t)
	h.tokenConfig = config.TokenConfig{CookieSecure: true, CookieDomain: "example.com"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.clearTokenCookies(c)

	cookies := cookieMap(w.Result().Cookies())
	for _, name := range []string{middleware.CookieAccessToken, middleware.CookieRefreshToken, middleware.CSRFCookieName, sessionCookieName} {
		require.Contains(t, cookies, name)
		assert.Empty(t, cookies[name].Value)
		assert.Less(t, cookies[name].MaxAge, 0)
	}
	assert.Empty(t, w.Header().Get(middleware.CSRFHeaderName))
}

func TestGetSessionID_FallsBackToCookie(t *testing.T) {
	h, _ := newTestHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-session"})
	c.Request = req

	assert.Equal(t, "cookie-session", h.getSessionID(c, "not-a-jwt"))
}

func TestGetSessionID_FallsBackToHeaderWhenCookieMissing(t *testing.T) {
	h, _ := newTestHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(nativeSessionIDHeader, "header-session")
	c.Request = req

	assert.Equal(t, "header-session", h.getSessionID(c, "not-a-jwt"))
}

func TestGetSessionID_RejectsAmbiguousHeaderAndCookie(t *testing.T) {
	h, _ := newTestHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-session"})
	req.Header.Set(nativeSessionIDHeader, "header-session")
	c.Request = req

	assert.Empty(t, h.getSessionID(c, "not-a-jwt"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))
}

func TestGetSessionID_RejectsRepeatedNativeSessionHeader(t *testing.T) {
	h, _ := newTestHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(nativeSessionIDHeader, "header-session-a")
	req.Header.Add(nativeSessionIDHeader, "header-session-b")
	c.Request = req

	assert.Empty(t, h.getSessionID(c, "not-a-jwt"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))
}

func TestSetSessionCookie(t *testing.T) {
	h, _ := newTestHandler(t)
	h.tokenConfig = config.TokenConfig{RefreshTokenTTL: 900, CookieSecure: true, CookieDomain: "example.com"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.setSessionCookie(c, "session-123")

	cookies := cookieMap(w.Result().Cookies())
	require.Contains(t, cookies, sessionCookieName)
	assert.Equal(t, "session-123", cookies[sessionCookieName].Value)
	assert.True(t, cookies[sessionCookieName].HttpOnly)
	assert.Equal(t, "/", cookies[sessionCookieName].Path)
	assert.Equal(t, 900, cookies[sessionCookieName].MaxAge)
}

func cookieMap(cookies []*http.Cookie) map[string]*http.Cookie {
	result := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		result[cookie.Name] = cookie
	}
	return result
}
