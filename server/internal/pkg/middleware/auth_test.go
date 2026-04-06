package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetTokenWithSource_PrefersBearerOverCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "cookie-token"})
	req.Header.Set("Authorization", "Bearer bearer-token")
	c.Request = req

	token, source := getTokenWithSource(c)

	assert.Equal(t, "bearer-token", token)
	assert.Equal(t, tokenSourceBearer, source)
}

func TestGetTokenWithSource_FallsBackToCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "cookie-token"})
	c.Request = req

	token, source := getTokenWithSource(c)

	assert.Equal(t, "cookie-token", token)
	assert.Equal(t, tokenSourceCookie, source)
}
