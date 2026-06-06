package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	assert.True(t, HasCapability(c, capability.UserStudentRead))
	assert.False(t, HasCapability(c, capability.UserIdentityRead))

	grants := GetCapabilityGrants(c)
	require.NotEmpty(t, grants)
}

func TestAuthContextGettersAreNilSafe(t *testing.T) {
	assert.Empty(t, GetUserID(nil))
	assert.Empty(t, GetAppID(nil))
	assert.Empty(t, GetUsername(nil))
	assert.Empty(t, GetEmail(nil))
	assert.Empty(t, GetDisplayName(nil))
	assert.Empty(t, GetAvatar(nil))
	assert.Nil(t, GetTokenScopes(nil))
	assert.Nil(t, GetOrgScopedRoles(nil))
	assert.Nil(t, GetRoles(nil))
	assert.Nil(t, GetCapabilities(nil))
	assert.Nil(t, GetGlobalCapabilities(nil))
	assert.Nil(t, GetCapabilityGrants(nil))
	assert.False(t, HasCapability(nil, capability.UserStudentRead))
	assert.False(t, HasGlobalCapability(nil, capability.UserStudentRead))
	assert.False(t, HasCapabilityInSchool(nil, capability.UserSchoolRead, "school-1"))
	assert.False(t, IsAuthenticated(nil))
	assert.True(t, GetAuthenticationTime(nil).IsZero())
	assert.False(t, GetMFAEnrollmentActive(nil))
	assert.True(t, GetMFAProofVerifiedAt(nil).IsZero())
	assert.Empty(t, GetRequestID(nil))
	assert.Empty(t, GetAccessToken(nil))

	tokenValue, source := getTokenWithSource(nil)
	assert.Empty(t, tokenValue)
	assert.Equal(t, tokenSourceNone, source)

	tokenValue, source = getTokenWithSource(&gin.Context{})
	assert.Empty(t, tokenValue)
	assert.Equal(t, tokenSourceNone, source)
}

func TestAuthContextSliceGettersReturnCopies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxKeyTokenScopes, []string{"scope:read"})
	c.Set(CtxKeyOrgScopedRoles, map[string][]string{
		" school_admin ": {" school-2 ", "school-1", "school-2", ""},
		" ":              {"ignored"},
	})
	c.Set(CtxKeyRoles, []string{"student"})
	c.Set(CtxKeyCapabilities, []string{capability.UserStudentRead})
	c.Set(CtxKeyGlobalCapabilities, []string{capability.UserSystemRead})
	c.Set(CtxKeyCapabilityGrants, []capability.Grant{{
		Name:            capability.UserSchoolRead,
		ScopeSchoolIDs:  []string{"school-1"},
		ScopeSectionIDs: []string{"section-1"},
		ScopeRoles:      []string{"school_admin"},
	}})

	tokenScopes := GetTokenScopes(c)
	tokenScopes[0] = "changed"
	assert.Equal(t, []string{"scope:read"}, GetTokenScopes(c))

	orgScopedRoles := GetOrgScopedRoles(c)
	require.Equal(t, map[string][]string{
		"school_admin": {"school-1", "school-2"},
	}, orgScopedRoles)
	orgScopedRoles["school_admin"][0] = "changed"
	orgScopedRoles["other"] = []string{"changed"}
	assert.Equal(t, map[string][]string{
		"school_admin": {"school-1", "school-2"},
	}, GetOrgScopedRoles(c))

	roles := GetRoles(c)
	roles[0] = "changed"
	assert.Equal(t, []string{"student"}, GetRoles(c))

	caps := GetCapabilities(c)
	caps[0] = "changed"
	assert.Equal(t, []string{capability.UserStudentRead}, GetCapabilities(c))

	globalCaps := GetGlobalCapabilities(c)
	globalCaps[0] = "changed"
	assert.Equal(t, []string{capability.UserSystemRead}, GetGlobalCapabilities(c))

	grants := GetCapabilityGrants(c)
	grants[0].Name = "changed"
	grants[0].ScopeSchoolIDs[0] = "changed"
	grants[0].ScopeSectionIDs[0] = "changed"
	grants[0].ScopeRoles[0] = "changed"

	stored := GetCapabilityGrants(c)
	require.Len(t, stored, 1)
	assert.Equal(t, capability.UserSchoolRead, stored[0].Name)
	assert.Equal(t, []string{"school-1"}, stored[0].ScopeSchoolIDs)
	assert.Equal(t, []string{"section-1"}, stored[0].ScopeSectionIDs)
	assert.Equal(t, []string{"school_admin"}, stored[0].ScopeRoles)
}

func TestAuthenticationAndMFAGetters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	authTime := time.Date(2026, 6, 6, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	mfaTime := authTime.Add(time.Minute)
	SetAuthenticationTime(c, authTime)
	SetMFAContext(c, MFAContext{EnrollmentActive: true, ProofVerifiedAt: mfaTime})

	assert.Equal(t, authTime.UTC(), GetAuthenticationTime(c))
	assert.True(t, GetMFAEnrollmentActive(c))
	assert.Equal(t, mfaTime, GetMFAProofVerifiedAt(c))
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
	require.Len(t, cookies, 4)
	for _, cookie := range cookies {
		assert.Empty(t, cookie.Value)
		assert.Less(t, cookie.MaxAge, 0)
		assert.Equal(t, "example.com", cookie.Domain)
		assert.True(t, cookie.Secure)
	}

	paths := map[string]string{}
	for _, cookie := range cookies {
		paths[cookie.Name] = cookie.Path
	}
	assert.Equal(t, CookieAccessTokenPath, paths[CookieAccessToken])
	assert.Equal(t, CookieRefreshTokenPath, paths[CookieRefreshToken])
	assert.Equal(t, "/", paths[CookieSessionID])
	assert.Equal(t, "/", paths[CSRFCookieName])
}

func TestDefaultRateLimitHelpers(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	assert.Equal(t, 10000, cfg.GlobalLimit)
	assert.Equal(t, 100, cfg.IPLimit)
	assert.Equal(t, 200, cfg.UserLimit)
	assert.Equal(t, "anonymous", rateLimitScope(""))
	assert.Equal(t, "user", rateLimitScope("user-1"))
}
