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

func TestGetTokenWithSource_RejectsMalformedAuthorizationOverCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		headers []string
	}{
		{
			name:    "unsupported scheme",
			headers: []string{"Basic Y2xpZW50OnNlY3JldA=="},
		},
		{
			name:    "blank bearer",
			headers: []string{"Bearer   "},
		},
		{
			name:    "repeated authorization",
			headers: []string{"Bearer bearer-token", "Bearer other-token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "cookie-token"})
			for _, header := range tt.headers {
				req.Header.Add("Authorization", header)
			}
			c.Request = req

			token, source := getTokenWithSource(c)

			assert.Empty(t, token)
			assert.Equal(t, tokenSourceBearer, source)
		})
	}
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
