package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

func TestSetClaimsToContextAndGetters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	avatar := "https://cdn.example.com/avatar.png"
	setClaimsToContext(c, &authResult{
		userID:      "user-1",
		appID:       "stuhelper-web",
		username:    "tester",
		email:       "tester@example.com",
		displayName: "Tester",
		avatar:      &avatar,
		roles:       []string{"student", "school_admin"},
		orgScopedRoles: map[string][]string{
			"school_admin": {"school-1"},
		},
	})

	assert.Equal(t, "user-1", GetUserID(c))
	assert.Equal(t, "stuhelper-web", GetAppID(c))
	assert.Equal(t, "tester", GetUsername(c))
	assert.Equal(t, "tester@example.com", GetEmail(c))
	assert.Equal(t, "Tester", GetDisplayName(c))
	assert.Equal(t, avatar, GetAvatar(c))
	assert.ElementsMatch(t, []string{"student", "school_admin"}, GetRoles(c))
	assert.True(t, IsAuthenticated(c))

	caps := GetCapabilities(c)
	assert.Contains(t, caps, capability.UserStudentRead)
	assert.Contains(t, caps, capability.UserSchoolRead)
	assert.Contains(t, caps, capability.UserSchoolUpdate)
	assert.NotContains(t, caps, capability.UserIdentityRead)

	assert.False(t, HasGlobalCapability(c, capability.UserSchoolRead))
	assert.True(t, HasCapabilityInSchool(c, capability.UserSchoolRead, "school-1"))
	assert.False(t, HasCapabilityInSchool(c, capability.UserSchoolRead, "school-2"))
	assert.True(t, HasRoleInOrg(c, "school_admin", "school-1"))
	assert.False(t, HasRoleInOrg(c, "school_admin", "school-2"))
	assert.True(t, HasCapability(c, capability.UserStudentRead))
	assert.False(t, HasCapability(c, capability.UserIdentityRead))

	grants := GetCapabilityGrants(c)
	require.NotEmpty(t, grants)
}

func TestGetTokenWithSourceAndAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "cookie-token"})
	req.Header.Set("Authorization", "Bearer header-token")
	c.Request = req

	tokenValue, source := getTokenWithSource(c)
	assert.Equal(t, "header-token", tokenValue)
	assert.Equal(t, tokenSourceBearer, source)
	assert.Equal(t, "header-token", GetAccessToken(c))
}

func TestGetTokenWithSource_Fallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "cookie-token"})
	c.Request = req

	tokenValue, source := getTokenWithSource(c)
	assert.Equal(t, "cookie-token", tokenValue)
	assert.Equal(t, tokenSourceCookie, source)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	tokenValue, source = getTokenWithSource(c2)
	assert.Empty(t, tokenValue)
	assert.Equal(t, tokenSourceNone, source)
}

func TestClearAuthCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	clearAuthCookies(c, OptionalAuthConfig{CookieDomain: "example.com", CookieSecure: true})

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, cookie := range cookies {
		assert.Empty(t, cookie.Value)
		assert.Less(t, cookie.MaxAge, 0)
		assert.Equal(t, "example.com", cookie.Domain)
		assert.True(t, cookie.Secure)
	}
}

func TestDefaultRateLimitHelpers(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	assert.Equal(t, 10000, cfg.GlobalLimit)
	assert.Equal(t, 100, cfg.IPLimit)
	assert.Equal(t, 200, cfg.UserLimit)
	assert.Equal(t, "anonymous", rateLimitScope(""))
	assert.Equal(t, "user", rateLimitScope("user-1"))
}
