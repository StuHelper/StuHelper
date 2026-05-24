package identityserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestHandlerDiscoveryAndJWKS(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)

	discovery := performIdentityRequest(router, http.MethodGet, "/.well-known/openid-configuration", nil)
	require.Equal(t, http.StatusOK, discovery.Code)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(discovery.Body.Bytes(), &metadata))
	assert.Equal(t, "https://id.example.com", metadata["issuer"])
	assert.Equal(t, "https://id.example.com/oauth2/authorize", metadata["authorization_endpoint"])
	assert.Equal(t, "https://id.example.com/oauth2/token", metadata["token_endpoint"])
	assert.Equal(t, "https://id.example.com/oidc/userinfo", metadata["userinfo_endpoint"])
	assert.Equal(t, "https://id.example.com/.well-known/jwks.json", metadata["jwks_uri"])
	assert.Equal(t, "https://id.example.com/oauth2/revoke", metadata["revocation_endpoint"])
	assert.Equal(t, "https://id.example.com/oauth2/introspect", metadata["introspection_endpoint"])
	assert.Equal(t, "https://id.example.com/oauth2/logout", metadata["end_session_endpoint"])
	assert.Equal(t, true, metadata["authorization_response_iss_parameter_supported"])
	assert.Contains(t, metadata["response_types_supported"], "code")
	assert.Contains(t, metadata["response_modes_supported"], "query")
	assert.Contains(t, metadata["grant_types_supported"], "refresh_token")
	assert.Contains(t, metadata["grant_types_supported"], "client_credentials")
	assert.Contains(t, metadata["code_challenge_methods_supported"], "S256")
	assert.Contains(t, metadata["prompt_values_supported"], "none")
	assert.Contains(t, metadata["prompt_values_supported"], "login")
	assert.Contains(t, metadata["prompt_values_supported"], "consent")
	assert.Contains(t, metadata["subject_types_supported"], "public")
	assert.Contains(t, metadata["id_token_signing_alg_values_supported"], "RS256")
	assert.Contains(t, metadata["token_endpoint_auth_methods_supported"], "client_secret_basic")
	assert.Contains(t, metadata["token_endpoint_auth_methods_supported"], "client_secret_post")
	assert.Contains(t, metadata["revocation_endpoint_auth_methods_supported"], "client_secret_basic")
	assert.Contains(t, metadata["revocation_endpoint_auth_methods_supported"], "client_secret_post")
	assert.Contains(t, metadata["introspection_endpoint_auth_methods_supported"], "client_secret_basic")
	assert.Contains(t, metadata["introspection_endpoint_auth_methods_supported"], "client_secret_post")
	assert.Contains(t, metadata["scopes_supported"], "openid")
	assert.Contains(t, metadata["scopes_supported"], "profile.basic.read")
	assert.Contains(t, metadata["scopes_supported"], "resource.read")
	assert.Contains(t, metadata["scopes_supported"], "resource.write")
	assert.Contains(t, metadata["scopes_supported"], "offline_access")
	assert.Contains(t, metadata["claims_supported"], "username")
	assert.Contains(t, metadata["claims_supported"], "displayName")
	assert.Contains(t, metadata["claims_supported"], "avatar")
	assert.Contains(t, metadata["claims_supported"], "phone_number")
	assert.Contains(t, metadata["claims_supported"], "phoneMasked")
	assert.Contains(t, metadata["claims_supported"], "identityVerified")
	assert.Contains(t, metadata["claims_supported"], "studentVerified")
	assert.Contains(t, metadata["claims_supported"], "school")

	oauthMetadataResponse := performIdentityRequest(router, http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	require.Equal(t, http.StatusOK, oauthMetadataResponse.Code)
	var oauthMetadata map[string]any
	require.NoError(t, json.Unmarshal(oauthMetadataResponse.Body.Bytes(), &oauthMetadata))
	assert.Equal(t, metadata, oauthMetadata)

	jwks := performIdentityRequest(router, http.MethodGet, "/.well-known/jwks.json", nil)
	require.Equal(t, http.StatusOK, jwks.Code)
	var keySet map[string][]map[string]any
	require.NoError(t, json.Unmarshal(jwks.Body.Bytes(), &keySet))
	require.Len(t, keySet["keys"], 1)
	assert.Equal(t, "identity-test-key", keySet["keys"][0]["kid"])
	assert.Equal(t, "sig", keySet["keys"][0]["use"])
}

func TestHandlerRegistersOIDCRouteSurface(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)

	actual := make(map[string]struct{}, len(router.Routes()))
	for _, route := range router.Routes() {
		actual[route.Method+" "+route.Path] = struct{}{}
	}

	expected := []string{
		"GET /.well-known/openid-configuration",
		"GET /.well-known/oauth-authorization-server",
		"GET /.well-known/jwks.json",
		"GET /oauth2/authorize",
		"GET /oauth2/continue",
		"GET /oauth2/logout",
		"POST /oauth2/logout",
		"POST /oauth2/token",
		"POST /oauth2/introspect",
		"POST /oauth2/revoke",
		"GET /oidc/userinfo",
		"POST /oidc/userinfo",
	}
	for _, route := range expected {
		_, ok := actual[route]
		assert.Truef(t, ok, "expected route %s to be registered", route)
	}
}

func TestHandlerAuthorizeUnauthenticatedRedirectsToFrontendLogin(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)
	verifier := validTestPKCEVerifier()
	authorizePath := "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-1&code_challenge=" + url.QueryEscape(s256Challenge(verifier)) + "&code_challenge_method=S256"

	response := performIdentityRequest(router, http.MethodGet, authorizePath, nil)

	require.Equal(t, http.StatusFound, response.Code)
	location := response.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com", parsed.Scheme+"://"+parsed.Host)
	assert.Equal(t, "/login", parsed.Path)
	assert.Equal(t, authorizePath, parsed.Query().Get("redirect"))
}

func TestHandlerAuthorizePromptNoneUnauthenticatedReturnsLoginRequired(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)
	verifier := validTestPKCEVerifier()
	authorizePath := "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-silent&code_challenge=" + url.QueryEscape(s256Challenge(verifier)) + "&code_challenge_method=S256&prompt=none"

	response := performIdentityRequest(router, http.MethodGet, authorizePath, nil)

	require.Equal(t, http.StatusFound, response.Code)
	callback, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.Equal(t, "login_required", callback.Query().Get("error"))
	assert.Equal(t, "https://id.example.com", callback.Query().Get("iss"))
	assert.Equal(t, "state-silent", callback.Query().Get("state"))
	assert.Empty(t, callback.Query().Get("code"))
}

func TestHandlerAuthorizePromptNoneRejectsInvalidRequestBeforeRedirect(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)
	verifier := validTestPKCEVerifier()

	combinedPrompt := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid&state=state-silent&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&prompt=none+login", nil)
	require.Equal(t, http.StatusBadRequest, combinedPrompt.Code)
	assert.Equal(t, "invalid authorization request", combinedPrompt.Body.String())

	invalidRedirect := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fevil.example.com%2Fcallback&scope=openid&state=state-silent&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&prompt=none", nil)
	require.Equal(t, http.StatusBadRequest, invalidRedirect.Code)
	assert.Equal(t, "invalid authorization request", invalidRedirect.Body.String())
}

func TestHandlerAuthorizeAuthenticatedIssuesCodeRedirect(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-2&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&prompt=none&max_age=3600", nil)

	require.Equal(t, http.StatusFound, response.Code)
	callback, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.NotEmpty(t, callback.Query().Get("code"))
	assert.Equal(t, "https://id.example.com", callback.Query().Get("iss"))
	assert.Equal(t, "state-2", callback.Query().Get("state"))
}

func TestHandlerAuthorizePromptLoginRedirectsToReauthLogin(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-reauth&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&prompt=login&max_age=0", nil)

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com/login", location.Scheme+"://"+location.Host+location.Path)
	assert.Equal(t, "1", location.Query().Get("reauth"))
	redirect := location.Query().Get("redirect")
	require.NotEmpty(t, redirect)
	redirectURL, err := url.Parse(redirect)
	require.NoError(t, err)
	assert.Equal(t, "/oauth2/authorize", redirectURL.Path)
	assert.Empty(t, redirectURL.Query().Get("prompt"))
	assert.Empty(t, redirectURL.Query().Get("max_age"))
	assert.Equal(t, "state-reauth", redirectURL.Query().Get("state"))
}

func TestHandlerAuthorizeZeroMaxAgeVariantIsConsumedAfterReauth(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-max-age-zero&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&max_age=00", nil)

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "1", location.Query().Get("reauth"))
	redirectURL, err := url.Parse(location.Query().Get("redirect"))
	require.NoError(t, err)
	assert.Equal(t, "/oauth2/authorize", redirectURL.Path)
	assert.Empty(t, redirectURL.Query().Get("max_age"))
	assert.Equal(t, "state-max-age-zero", redirectURL.Query().Get("state"))
}

func TestHandlerAuthorizePromptLoginPreservesPromptConsentAfterReauth(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+email&state=state-login-consent&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&prompt=login+consent", nil)

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "1", location.Query().Get("reauth"))
	redirectURL, err := url.Parse(location.Query().Get("redirect"))
	require.NoError(t, err)
	assert.Equal(t, "/oauth2/authorize", redirectURL.Path)
	assert.Equal(t, "consent", redirectURL.Query().Get("prompt"))
	assert.Equal(t, "state-login-consent", redirectURL.Query().Get("state"))
}

func TestHandlerAuthorizeMaxAgeStaleSessionRedirectsToReauthLogin(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-max-age&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256&max_age=30", nil)

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "1", location.Query().Get("reauth"))
	redirectURL, err := url.Parse(location.Query().Get("redirect"))
	require.NoError(t, err)
	assert.Equal(t, "30", redirectURL.Query().Get("max_age"))
	assert.Equal(t, "state-max-age", redirectURL.Query().Get("state"))
}

func TestHandlerAuthorizeAuthenticatedRejectsMissingPKCE(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-2", nil)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid authorization request", response.Body.String())
}

func TestHandlerAuthorizeRejectsRepeatedParameters(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)
	verifier := validTestPKCEVerifier()
	base := "/oauth2/authorize?response_type=code&client_id=client-1&scope=openid&code_challenge=" + url.QueryEscape(s256Challenge(verifier)) + "&code_challenge_method=S256"

	tests := []struct {
		name   string
		target string
	}{
		{
			name: "redirect uri smuggling",
			target: base +
				"&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback" +
				"&redirect_uri=https%3A%2F%2Fevil.example.com%2Fcallback" +
				"&state=state-1",
		},
		{
			name: "state ambiguity",
			target: base +
				"&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback" +
				"&state=state-1&state=state-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodGet, tt.target, nil)

			require.Equal(t, http.StatusBadRequest, response.Code)
			assert.Equal(t, "invalid authorization request", response.Body.String())
			assert.Empty(t, response.Header().Get("Location"))
		})
	}
}

func TestHandlerAuthorizeRejectsOfflineAccessWithoutOpenID(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)
	verifier := validTestPKCEVerifier()

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=resource.read+offline_access&state=state-offline-no-openid&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256", nil)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid authorization request", response.Body.String())
}

func TestHandlerLogoutRedirectsToRegisteredPostLogoutURI(t *testing.T) {
	router, service, _ := newIdentityHandlerTestRouter(t, false)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/logout?client_id=client-1&post_logout_redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&state=logout-state", nil)

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", location.Scheme+"://"+location.Host+location.Path)
	assert.Equal(t, "logout-state", location.Query().Get("state"))

	idToken, err := service.signer.SignIDToken(IDTokenInput{
		Subject:  "stuhelper:42",
		ClientID: "client-1",
		TTL:      time.Minute,
	})
	require.NoError(t, err)
	hintResponse := performIdentityRequest(router, http.MethodGet, "/oauth2/logout?id_token_hint="+url.QueryEscape(idToken)+"&post_logout_redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&state=hint-state", nil)
	require.Equal(t, http.StatusFound, hintResponse.Code)
	hintLocation, err := url.Parse(hintResponse.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "hint-state", hintLocation.Query().Get("state"))

	postForm := url.Values{
		"client_id":                {"client-1"},
		"post_logout_redirect_uri": {"https://client.example.com/callback"},
		"state":                    {"post-logout-state"},
	}
	postResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/logout", strings.NewReader(postForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	require.Equal(t, http.StatusFound, postResponse.Code)
	postLocation, err := url.Parse(postResponse.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", postLocation.Scheme+"://"+postLocation.Host+postLocation.Path)
	assert.Equal(t, "post-logout-state", postLocation.Query().Get("state"))
}

func TestHandlerLogoutRejectsUnregisteredPostLogoutURI(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/logout?client_id=client-1&post_logout_redirect_uri=https%3A%2F%2Fevil.example.com%2Fcallback", nil)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid logout request", response.Body.String())
}

func TestHandlerLogoutRejectsAccessTokenAsIDTokenHint(t *testing.T) {
	router, service, _ := newIdentityHandlerTestRouter(t, false)
	accessToken, _, err := service.signer.SignAccessToken(AccessTokenInput{
		Subject:  "stuhelper:42",
		ClientID: "client-1",
		UserID:   42,
		Scopes:   []string{"openid", "profile"},
		TTL:      time.Minute,
	})
	require.NoError(t, err)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/logout?id_token_hint="+url.QueryEscape(accessToken)+"&post_logout_redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback", nil)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid logout request", response.Body.String())
}

func TestHandlerLogoutRejectsAccessTokenAsIDTokenHintWithoutRedirect(t *testing.T) {
	router, service, _ := newIdentityHandlerTestRouter(t, false)
	accessToken, _, err := service.signer.SignAccessToken(AccessTokenInput{
		Subject:  "stuhelper:42",
		ClientID: "client-1",
		UserID:   42,
		Scopes:   []string{"openid", "profile"},
		TTL:      time.Minute,
	})
	require.NoError(t, err)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/logout?id_token_hint="+url.QueryEscape(accessToken), nil)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid logout request", response.Body.String())
}

func TestHandlerLogoutRejectsRepeatedParameters(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)

	getResponse := performIdentityRequest(
		router,
		http.MethodGet,
		"/oauth2/logout?client_id=client-1&post_logout_redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&post_logout_redirect_uri=https%3A%2F%2Fevil.example.com%2Fcallback&state=logout-state",
		nil,
	)
	require.Equal(t, http.StatusBadRequest, getResponse.Code)
	assert.Equal(t, "invalid logout request", getResponse.Body.String())

	postForm := url.Values{
		"client_id":                {"client-1"},
		"post_logout_redirect_uri": {"https://client.example.com/callback"},
		"state":                    {"state-1", "state-2"},
	}
	postResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/logout", strings.NewReader(postForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	require.Equal(t, http.StatusBadRequest, postResponse.Code)
	assert.Equal(t, "invalid logout request", postResponse.Body.String())
}

func TestHandlerLogoutPostRejectsQueryParametersAndUnsupportedBody(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)

	queryResponse := performIdentityRequest(
		router,
		http.MethodPost,
		"/oauth2/logout?client_id=client-1&post_logout_redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback",
		nil,
	)
	require.Equal(t, http.StatusBadRequest, queryResponse.Code)
	assert.Equal(t, "invalid logout request", queryResponse.Body.String())

	jsonResponse := performIdentityRequest(
		router,
		http.MethodPost,
		"/oauth2/logout",
		strings.NewReader(`{"client_id":"client-1"}`),
		func(req *http.Request) {
			req.Header.Set("Content-Type", "application/json")
		},
	)
	require.Equal(t, http.StatusBadRequest, jsonResponse.Code)
	assert.Equal(t, "invalid logout request", jsonResponse.Body.String())

	emptyPostResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/logout", nil)
	require.Equal(t, http.StatusNoContent, emptyPostResponse.Code)
}

func TestHandlerLogoutRevokesAuthenticatedSessionAndClearsCookies(t *testing.T) {
	revoker := &fakeIdentitySessionRevoker{}
	router, _, _ := newIdentityHandlerTestRouterWithRevoker(t, true, revoker)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/logout", nil, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: middleware.CookieAccessToken, Value: "access-logout"})
		req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "refresh-logout"})
		req.AddCookie(&http.Cookie{Name: middleware.CookieSessionID, Value: "session-logout"})
		req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "csrf-logout"})
	})

	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "session-logout", revoker.sessionID)
	assert.Equal(t, "casdoor-subject-1", revoker.userID)
	assert.Equal(t, "access-logout", revoker.accessToken)
	assert.Equal(t, "refresh-logout", revoker.refreshToken)
	cookies := response.Result().Cookies()
	cleared := map[string]http.Cookie{}
	for _, cookie := range cookies {
		cleared[cookie.Name] = *cookie
	}
	for _, name := range []string{middleware.CookieAccessToken, middleware.CookieRefreshToken, middleware.CookieSessionID, middleware.CSRFCookieName} {
		cookie, ok := cleared[name]
		require.True(t, ok, "expected %s to be cleared", name)
		assert.Negative(t, cookie.MaxAge)
		assert.Equal(t, "example.com", cookie.Domain)
		assert.True(t, cookie.Secure)
	}
}

func TestHandlerTokenUserInfoIntrospectAndRevoke(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()
	authorizeResponse := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile+email&state=state-3&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256", nil)
	require.Equal(t, http.StatusFound, authorizeResponse.Code)
	callback, err := url.Parse(authorizeResponse.Header().Get("Location"))
	require.NoError(t, err)
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"https://client.example.com/callback"},
		"code_verifier": {verifier},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	assertOAuthNoStoreHeaders(t, tokenResponse)
	var tokenPayload map[string]any
	require.NoError(t, json.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	accessToken, _ := tokenPayload["access_token"].(string)
	require.NotEmpty(t, accessToken)
	assert.NotEmpty(t, tokenPayload["id_token"])
	assert.Equal(t, "Bearer", tokenPayload["token_type"])
	assert.Equal(t, "openid profile email", tokenPayload["scope"])

	missingBearer := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo", nil)
	require.Equal(t, http.StatusUnauthorized, missingBearer.Code)
	assertOAuthNoStoreHeaders(t, missingBearer)
	assertBearerInvalidTokenChallenge(t, missingBearer)
	assert.JSONEq(t, `{"error":"invalid_token"}`, missingBearer.Body.String())

	userInfo := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	})
	require.Equal(t, http.StatusOK, userInfo.Code)
	assertOAuthNoStoreHeaders(t, userInfo)
	var userInfoPayload map[string]any
	require.NoError(t, json.Unmarshal(userInfo.Body.Bytes(), &userInfoPayload))
	assert.Equal(t, "stuhelper:42", userInfoPayload["sub"])
	assert.Equal(t, "alice", userInfoPayload["preferred_username"])
	assert.Equal(t, "alice", userInfoPayload["name"])
	assert.Equal(t, "alice@example.com", userInfoPayload["email"])
	assert.Equal(t, true, userInfoPayload["email_verified"])

	repeatedAuthorizationUserInfo := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo", nil, func(req *http.Request) {
		req.Header.Add("Authorization", "Bearer "+accessToken)
		req.Header.Add("Authorization", "Bearer other-token")
	})
	require.Equal(t, http.StatusUnauthorized, repeatedAuthorizationUserInfo.Code)
	assertOAuthNoStoreHeaders(t, repeatedAuthorizationUserInfo)
	assertBearerInvalidTokenChallenge(t, repeatedAuthorizationUserInfo)
	assert.JSONEq(t, `{"error":"invalid_token"}`, repeatedAuthorizationUserInfo.Body.String())

	queryTokenUserInfo := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo?access_token="+url.QueryEscape(accessToken), nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	})
	require.Equal(t, http.StatusUnauthorized, queryTokenUserInfo.Code)
	assertOAuthNoStoreHeaders(t, queryTokenUserInfo)
	assertBearerInvalidTokenChallenge(t, queryTokenUserInfo)
	assert.JSONEq(t, `{"error":"invalid_token"}`, queryTokenUserInfo.Body.String())

	bodyTokenUserInfo := performIdentityRequest(router, http.MethodPost, "/oidc/userinfo", strings.NewReader(url.Values{"access_token": {accessToken}}.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+accessToken)
	})
	require.Equal(t, http.StatusUnauthorized, bodyTokenUserInfo.Code)
	assertOAuthNoStoreHeaders(t, bodyTokenUserInfo)
	assertBearerInvalidTokenChallenge(t, bodyTokenUserInfo)
	assert.JSONEq(t, `{"error":"invalid_token"}`, bodyTokenUserInfo.Body.String())

	postUserInfo := performIdentityRequest(router, http.MethodPost, "/oidc/userinfo", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	})
	require.Equal(t, http.StatusOK, postUserInfo.Code)
	assertOAuthNoStoreHeaders(t, postUserInfo)
	var postUserInfoPayload map[string]any
	require.NoError(t, json.Unmarshal(postUserInfo.Body.Bytes(), &postUserInfoPayload))
	assert.Equal(t, userInfoPayload["sub"], postUserInfoPayload["sub"])
	assert.Equal(t, userInfoPayload["email"], postUserInfoPayload["email"])

	introspectionForm := url.Values{"token": {accessToken}, "token_type_hint": {"access_token"}}
	introspection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, introspection.Code)
	assertOAuthNoStoreHeaders(t, introspection)
	var introspectionPayload map[string]any
	require.NoError(t, json.Unmarshal(introspection.Body.Bytes(), &introspectionPayload))
	assert.Equal(t, true, introspectionPayload["active"])
	assert.Equal(t, "client-1", introspectionPayload["client_id"])
	assert.Equal(t, "stuhelper:42", introspectionPayload["sub"])
	assert.Equal(t, "client-1", introspectionPayload["aud"])
	assert.Equal(t, "Bearer", introspectionPayload["token_type"])
	assert.Equal(t, "access_token", introspectionPayload["token_kind"])

	crossClientIntrospection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-2", "secret-2")
	})
	require.Equal(t, http.StatusOK, crossClientIntrospection.Code)
	var crossClientPayload map[string]any
	require.NoError(t, json.Unmarshal(crossClientIntrospection.Body.Bytes(), &crossClientPayload))
	assert.Equal(t, false, crossClientPayload["active"])

	crossClientRevoke := performIdentityRequest(router, http.MethodPost, "/oauth2/revoke", strings.NewReader(url.Values{"token": {accessToken}}.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-2", "secret-2")
	})
	require.Equal(t, http.StatusOK, crossClientRevoke.Code)
	assertOAuthNoStoreHeaders(t, crossClientRevoke)
	stillActive := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, stillActive.Code)
	var stillActivePayload map[string]any
	require.NoError(t, json.Unmarshal(stillActive.Body.Bytes(), &stillActivePayload))
	assert.Equal(t, true, stillActivePayload["active"])

	revokeForm := url.Values{"token": {accessToken}, "token_type_hint": {"refresh_token"}}
	revoke := performIdentityRequest(router, http.MethodPost, "/oauth2/revoke", strings.NewReader(revokeForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, revoke.Code)
	assertOAuthNoStoreHeaders(t, revoke)

	introspectionAfterRevoke := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, introspectionAfterRevoke.Code)
	var revokedPayload map[string]any
	require.NoError(t, json.Unmarshal(introspectionAfterRevoke.Body.Bytes(), &revokedPayload))
	assert.Equal(t, false, revokedPayload["active"])
}

func TestHandlerRevokeUsedRefreshTokenRevokesRotatedFamily(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()
	authorizeResponse := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile+offline_access&state=state-refresh-revoke&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256", nil)
	require.Equal(t, http.StatusFound, authorizeResponse.Code)
	callback, err := url.Parse(authorizeResponse.Header().Get("Location"))
	require.NoError(t, err)
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"https://client.example.com/callback"},
		"code_verifier": {verifier},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	assertOAuthNoStoreHeaders(t, tokenResponse)
	var tokenPayload map[string]any
	require.NoError(t, json.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	refreshToken, _ := tokenPayload["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)

	refreshForm := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	refreshResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(refreshForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, refreshResponse.Code)
	assertOAuthNoStoreHeaders(t, refreshResponse)
	var refreshPayload map[string]any
	require.NoError(t, json.Unmarshal(refreshResponse.Body.Bytes(), &refreshPayload))
	rotatedRefreshToken, _ := refreshPayload["refresh_token"].(string)
	require.NotEmpty(t, rotatedRefreshToken)
	assert.NotEqual(t, refreshToken, rotatedRefreshToken)

	rotatedIntrospectionForm := url.Values{"token": {rotatedRefreshToken}, "token_type_hint": {"refresh_token"}}
	rotatedIntrospection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(rotatedIntrospectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, rotatedIntrospection.Code)
	assertOAuthNoStoreHeaders(t, rotatedIntrospection)
	var rotatedIntrospectionPayload map[string]any
	require.NoError(t, json.Unmarshal(rotatedIntrospection.Body.Bytes(), &rotatedIntrospectionPayload))
	assert.Equal(t, true, rotatedIntrospectionPayload["active"])
	assert.Equal(t, "refresh_token", rotatedIntrospectionPayload["token_type"])
	assert.Equal(t, "refresh_token", rotatedIntrospectionPayload["token_kind"])

	revokeUsed := performIdentityRequest(router, http.MethodPost, "/oauth2/revoke", strings.NewReader(url.Values{"token": {refreshToken}, "token_type_hint": {"access_token"}}.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, revokeUsed.Code)
	assertOAuthNoStoreHeaders(t, revokeUsed)

	afterRevoke := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(rotatedIntrospectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, afterRevoke.Code)
	assertOAuthNoStoreHeaders(t, afterRevoke)
	var afterRevokePayload map[string]any
	require.NoError(t, json.Unmarshal(afterRevoke.Body.Bytes(), &afterRevokePayload))
	assert.Equal(t, false, afterRevokePayload["active"])
}

func TestHandlerTokenClientCredentialsGrantIssuesAppOnlyToken(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tokenForm := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {openplatform.ScopeResourceRead},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	var tokenPayload map[string]any
	require.NoError(t, json.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	accessToken, _ := tokenPayload["access_token"].(string)
	require.NotEmpty(t, accessToken)
	assert.Equal(t, "Bearer", tokenPayload["token_type"])
	assert.Equal(t, openplatform.ScopeResourceRead, tokenPayload["scope"])
	assert.NotContains(t, tokenPayload, "id_token")
	assert.NotContains(t, tokenPayload, "refresh_token")

	userInfo := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	})
	require.Equal(t, http.StatusUnauthorized, userInfo.Code)
	assertOAuthNoStoreHeaders(t, userInfo)
	assertBearerInvalidTokenChallenge(t, userInfo)
	assert.JSONEq(t, `{"error":"invalid_token"}`, userInfo.Body.String())

	introspectionForm := url.Values{"token": {accessToken}}
	introspection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, introspection.Code)
	var introspectionPayload map[string]any
	require.NoError(t, json.Unmarshal(introspection.Body.Bytes(), &introspectionPayload))
	assert.Equal(t, true, introspectionPayload["active"])
	assert.Equal(t, "client-1", introspectionPayload["client_id"])
	assert.Equal(t, "client:client-1", introspectionPayload["sub"])
	assert.Equal(t, openplatform.ScopeResourceRead, introspectionPayload["scope"])
	assert.Equal(t, "Bearer", introspectionPayload["token_type"])
	assert.Equal(t, "access_token", introspectionPayload["token_kind"])
	assert.Equal(t, "client_credentials", introspectionPayload["grant_type"])
}

func TestHandlerClientSecretPostAuthWorksForTokenIntrospectAndRevoke(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tokenForm := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {openplatform.ScopeResourceRead},
		"client_id":     {"client-1"},
		"client_secret": {"secret-1"},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	var tokenPayload map[string]any
	require.NoError(t, json.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	accessToken, _ := tokenPayload["access_token"].(string)
	require.NotEmpty(t, accessToken)
	assert.Equal(t, "Bearer", tokenPayload["token_type"])
	assert.Equal(t, openplatform.ScopeResourceRead, tokenPayload["scope"])

	introspectionForm := url.Values{
		"token":         {accessToken},
		"client_id":     {"client-1"},
		"client_secret": {"secret-1"},
	}
	introspection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	require.Equal(t, http.StatusOK, introspection.Code)
	var introspectionPayload map[string]any
	require.NoError(t, json.Unmarshal(introspection.Body.Bytes(), &introspectionPayload))
	assert.Equal(t, true, introspectionPayload["active"])
	assert.Equal(t, "client-1", introspectionPayload["client_id"])
	assert.Equal(t, "client_credentials", introspectionPayload["grant_type"])

	revokeForm := url.Values{
		"token":         {accessToken},
		"client_id":     {"client-1"},
		"client_secret": {"secret-1"},
	}
	revoke := performIdentityRequest(router, http.MethodPost, "/oauth2/revoke", strings.NewReader(revokeForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	require.Equal(t, http.StatusOK, revoke.Code)

	afterRevoke := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	require.Equal(t, http.StatusOK, afterRevoke.Code)
	var afterRevokePayload map[string]any
	require.NoError(t, json.Unmarshal(afterRevoke.Body.Bytes(), &afterRevokePayload))
	assert.Equal(t, false, afterRevokePayload["active"])
}

func TestHandlerRejectsMixedClientAuthenticationMethods(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name string
		path string
		form url.Values
	}{
		{
			name: "token",
			path: "/oauth2/token",
			form: url.Values{
				"grant_type":    {"client_credentials"},
				"scope":         {openplatform.ScopeResourceRead},
				"client_id":     {"client-1"},
				"client_secret": {"secret-1"},
			},
		},
		{
			name: "introspect",
			path: "/oauth2/introspect",
			form: url.Values{
				"token":         {"opaque-token"},
				"client_id":     {"client-1"},
				"client_secret": {"secret-1"},
			},
		},
		{
			name: "revoke",
			path: "/oauth2/revoke",
			form: url.Values{
				"token":         {"opaque-token"},
				"client_id":     {"client-1"},
				"client_secret": {"secret-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.path, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth("client-1", "secret-1")
			})

			require.Equal(t, http.StatusUnauthorized, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assertBasicInvalidClientChallenge(t, response)
			assert.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())
		})
	}
}

func TestHandlerTokenRefreshGrantRotatesOfflineRefreshToken(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()
	authorizeResponse := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile+offline_access&state=state-refresh&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256", nil)
	require.Equal(t, http.StatusFound, authorizeResponse.Code)
	callback, err := url.Parse(authorizeResponse.Header().Get("Location"))
	require.NoError(t, err)

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {callback.Query().Get("code")},
		"redirect_uri":  {"https://client.example.com/callback"},
		"code_verifier": {verifier},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	var tokenPayload map[string]any
	require.NoError(t, json.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	refreshToken, _ := tokenPayload["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)
	refreshIntrospectionForm := url.Values{"token": {refreshToken}}
	refreshIntrospection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(refreshIntrospectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, refreshIntrospection.Code)
	var refreshIntrospectionPayload map[string]any
	require.NoError(t, json.Unmarshal(refreshIntrospection.Body.Bytes(), &refreshIntrospectionPayload))
	assert.Equal(t, true, refreshIntrospectionPayload["active"])
	assert.Equal(t, "refresh_token", refreshIntrospectionPayload["token_type"])
	assert.Equal(t, "refresh_token", refreshIntrospectionPayload["token_kind"])
	assert.Equal(t, "client-1", refreshIntrospectionPayload["client_id"])

	refreshForm := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	refreshResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(refreshForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, refreshResponse.Code)
	var refreshPayload map[string]any
	require.NoError(t, json.Unmarshal(refreshResponse.Body.Bytes(), &refreshPayload))
	rotatedRefreshToken, _ := refreshPayload["refresh_token"].(string)
	require.NotEmpty(t, rotatedRefreshToken)
	assert.NotEqual(t, refreshToken, rotatedRefreshToken)
	assert.NotEmpty(t, refreshPayload["access_token"])
	assert.NotEmpty(t, refreshPayload["id_token"])
	assert.Equal(t, "openid profile offline_access", refreshPayload["scope"])
	rotatedRefreshIntrospectionForm := url.Values{"token": {rotatedRefreshToken}}
	rotatedRefreshIntrospection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(rotatedRefreshIntrospectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, rotatedRefreshIntrospection.Code)
	var rotatedRefreshIntrospectionPayload map[string]any
	require.NoError(t, json.Unmarshal(rotatedRefreshIntrospection.Body.Bytes(), &rotatedRefreshIntrospectionPayload))
	assert.Equal(t, true, rotatedRefreshIntrospectionPayload["active"])
	usedRefreshIntrospection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(refreshIntrospectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, usedRefreshIntrospection.Code)
	var usedRefreshIntrospectionPayload map[string]any
	require.NoError(t, json.Unmarshal(usedRefreshIntrospection.Body.Bytes(), &usedRefreshIntrospectionPayload))
	assert.Equal(t, false, usedRefreshIntrospectionPayload["active"])

	reuseResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(refreshForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusBadRequest, reuseResponse.Code)
	assert.JSONEq(t, `{"error":"invalid_grant"}`, reuseResponse.Body.String())
	afterReuseIntrospection := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(rotatedRefreshIntrospectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, afterReuseIntrospection.Code)
	var afterReuseIntrospectionPayload map[string]any
	require.NoError(t, json.Unmarshal(afterReuseIntrospection.Body.Bytes(), &afterReuseIntrospectionPayload))
	assert.Equal(t, false, afterReuseIntrospectionPayload["active"])
}

func TestHandlerTokenRequiresRedirectURI(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	verifier := validTestPKCEVerifier()
	authorizeResponse := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-redirect-required&code_challenge="+url.QueryEscape(s256Challenge(verifier))+"&code_challenge_method=S256", nil)
	require.Equal(t, http.StatusFound, authorizeResponse.Code)
	callback, err := url.Parse(authorizeResponse.Header().Get("Location"))
	require.NoError(t, err)
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})

	require.Equal(t, http.StatusBadRequest, tokenResponse.Code)
	assert.JSONEq(t, `{"error":"invalid_grant"}`, tokenResponse.Body.String())
}

func TestHandlerOAuthErrors(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	unsupportedGrant := url.Values{
		"grant_type": {"password"},
	}
	unsupported := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(unsupportedGrant.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusBadRequest, unsupported.Code)
	assertOAuthNoStoreHeaders(t, unsupported)
	assert.JSONEq(t, `{"error":"unsupported_grant_type"}`, unsupported.Body.String())

	invalidScopeGrant := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"openid"},
	}
	invalidScope := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(invalidScopeGrant.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusBadRequest, invalidScope.Code)
	assertOAuthNoStoreHeaders(t, invalidScope)
	assert.JSONEq(t, `{"error":"invalid_scope"}`, invalidScope.Body.String())

	invalidClientForm := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"missing-code"},
	}
	invalidClient := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(invalidClientForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "wrong-secret")
	})
	require.Equal(t, http.StatusUnauthorized, invalidClient.Code)
	assertOAuthNoStoreHeaders(t, invalidClient)
	assertBasicInvalidClientChallenge(t, invalidClient)
	assert.JSONEq(t, `{"error":"invalid_client"}`, invalidClient.Body.String())

	badIntrospectClient := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(url.Values{"token": {"token"}}.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "wrong-secret")
	})
	require.Equal(t, http.StatusUnauthorized, badIntrospectClient.Code)
	assertOAuthNoStoreHeaders(t, badIntrospectClient)
	assertBasicInvalidClientChallenge(t, badIntrospectClient)
	assert.JSONEq(t, `{"error":"invalid_client"}`, badIntrospectClient.Body.String())
}

func TestHandlerRejectsMissingRequiredOAuthParameters(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name     string
		endpoint string
		form     url.Values
	}{
		{
			name:     "token missing grant type",
			endpoint: "/oauth2/token",
			form:     url.Values{"scope": {openplatform.ScopeResourceRead}},
		},
		{
			name:     "token blank grant type",
			endpoint: "/oauth2/token",
			form: url.Values{
				"grant_type": {"   "},
				"scope":      {openplatform.ScopeResourceRead},
			},
		},
		{
			name:     "introspect missing token",
			endpoint: "/oauth2/introspect",
			form:     url.Values{"token_type_hint": {"access_token"}},
		},
		{
			name:     "introspect blank token",
			endpoint: "/oauth2/introspect",
			form:     url.Values{"token": {"   "}},
		},
		{
			name:     "revoke missing token",
			endpoint: "/oauth2/revoke",
			form:     url.Values{"token_type_hint": {"access_token"}},
		},
		{
			name:     "revoke blank token",
			endpoint: "/oauth2/revoke",
			form:     url.Values{"token": {"   "}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.endpoint, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth("client-1", "secret-1")
			})

			require.Equal(t, http.StatusBadRequest, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assert.JSONEq(t, `{"error":"invalid_request"}`, response.Body.String())
		})
	}
}

func TestHandlerRejectsRepeatedOAuthFormParameters(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name     string
		endpoint string
		form     url.Values
	}{
		{
			name:     "token duplicate grant type",
			endpoint: "/oauth2/token",
			form: url.Values{
				"grant_type": {"client_credentials", "refresh_token"},
				"scope":      {openplatform.ScopeResourceRead},
			},
		},
		{
			name:     "introspect duplicate token",
			endpoint: "/oauth2/introspect",
			form: url.Values{
				"token": {"opaque-token-1", "opaque-token-2"},
			},
		},
		{
			name:     "revoke duplicate token type hint",
			endpoint: "/oauth2/revoke",
			form: url.Values{
				"token":           {"opaque-token"},
				"token_type_hint": {"access_token", "refresh_token"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.endpoint, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth("client-1", "secret-1")
			})

			require.Equal(t, http.StatusBadRequest, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assert.JSONEq(t, `{"error":"invalid_request"}`, response.Body.String())
		})
	}
}

func TestHandlerRejectsOAuthQueryParametersOnSensitivePostEndpoints(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name   string
		target string
		form   url.Values
	}{
		{
			name:   "token query grant type",
			target: "/oauth2/token?grant_type=client_credentials",
			form:   url.Values{"scope": {openplatform.ScopeResourceRead}},
		},
		{
			name:   "introspect query token",
			target: "/oauth2/introspect?token=leaked-token",
			form:   url.Values{"token": {"body-token"}},
		},
		{
			name:   "revoke query client secret",
			target: "/oauth2/revoke?client_secret=leaked-secret",
			form:   url.Values{"token": {"body-token"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.target, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth("client-1", "secret-1")
			})

			require.Equal(t, http.StatusBadRequest, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assert.JSONEq(t, `{"error":"invalid_request"}`, response.Body.String())
		})
	}
}

func TestHandlerRejectsUnsupportedOAuthFormContentType(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name        string
		endpoint    string
		contentType string
		body        string
	}{
		{
			name:        "token json body",
			endpoint:    "/oauth2/token",
			contentType: "application/json",
			body:        `{"grant_type":"client_credentials","scope":"resource.read"}`,
		},
		{
			name:        "introspect missing content type",
			endpoint:    "/oauth2/introspect",
			contentType: "",
			body:        url.Values{"token": {"opaque-token"}}.Encode(),
		},
		{
			name:        "revoke text body",
			endpoint:    "/oauth2/revoke",
			contentType: "text/plain",
			body:        "token=opaque-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.endpoint, strings.NewReader(tt.body), func(req *http.Request) {
				if tt.contentType != "" {
					req.Header.Set("Content-Type", tt.contentType)
				}
				req.SetBasicAuth("client-1", "secret-1")
			})

			require.Equal(t, http.StatusBadRequest, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assert.JSONEq(t, `{"error":"invalid_request"}`, response.Body.String())
		})
	}
}

func TestHandlerRejectsMixedClientAuthenticationWithBlankBodyCredentials(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name     string
		endpoint string
		form     url.Values
	}{
		{
			name:     "token blank client id",
			endpoint: "/oauth2/token",
			form: url.Values{
				"grant_type": {"client_credentials"},
				"scope":      {openplatform.ScopeResourceRead},
				"client_id":  {""},
			},
		},
		{
			name:     "introspect blank client secret",
			endpoint: "/oauth2/introspect",
			form: url.Values{
				"token":         {"opaque-token"},
				"client_secret": {""},
			},
		},
		{
			name:     "revoke whitespace client id",
			endpoint: "/oauth2/revoke",
			form: url.Values{
				"token":     {"opaque-token"},
				"client_id": {"   "},
			},
		},
		{
			name:     "token duplicate body client id",
			endpoint: "/oauth2/token",
			form: url.Values{
				"grant_type": {"client_credentials"},
				"scope":      {openplatform.ScopeResourceRead},
				"client_id":  {"client-1", "client-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.endpoint, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth("client-1", "secret-1")
			})

			require.Equal(t, http.StatusUnauthorized, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assertBasicInvalidClientChallenge(t, response)
			assert.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())
		})
	}
}

func TestHandlerRejectsRepeatedAuthorizationHeaders(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name     string
		endpoint string
		form     url.Values
	}{
		{
			name:     "token repeated authorization",
			endpoint: "/oauth2/token",
			form: url.Values{
				"grant_type": {"client_credentials"},
				"scope":      {openplatform.ScopeResourceRead},
			},
		},
		{
			name:     "introspect repeated authorization",
			endpoint: "/oauth2/introspect",
			form:     url.Values{"token": {"opaque-token"}},
		},
		{
			name:     "revoke repeated authorization",
			endpoint: "/oauth2/revoke",
			form:     url.Values{"token": {"opaque-token"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.endpoint, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Add("Authorization", "Basic "+basicAuth("client-1", "secret-1"))
				req.Header.Add("Authorization", "Basic "+basicAuth("client-1", "secret-1"))
			})

			require.Equal(t, http.StatusUnauthorized, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assertBasicInvalidClientChallenge(t, response)
			assert.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())
		})
	}
}

func TestHandlerRejectsUnsupportedAuthorizationHeaderWithBodyClientCredentials(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	tests := []struct {
		name          string
		endpoint      string
		authorization string
		form          url.Values
	}{
		{
			name:          "token bearer plus body secret",
			endpoint:      "/oauth2/token",
			authorization: "Bearer access-token",
			form: url.Values{
				"grant_type":    {"client_credentials"},
				"scope":         {openplatform.ScopeResourceRead},
				"client_id":     {"client-1"},
				"client_secret": {"secret-1"},
			},
		},
		{
			name:          "introspect malformed basic plus body secret",
			endpoint:      "/oauth2/introspect",
			authorization: "Basic not-valid-base64",
			form: url.Values{
				"token":         {"opaque-token"},
				"client_id":     {"client-1"},
				"client_secret": {"secret-1"},
			},
		},
		{
			name:          "revoke unsupported scheme plus body secret",
			endpoint:      "/oauth2/revoke",
			authorization: "Digest abc",
			form: url.Values{
				"token":         {"opaque-token"},
				"client_id":     {"client-1"},
				"client_secret": {"secret-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performIdentityRequest(router, http.MethodPost, tt.endpoint, strings.NewReader(tt.form.Encode()), func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("Authorization", tt.authorization)
			})

			require.Equal(t, http.StatusUnauthorized, response.Code)
			assertOAuthNoStoreHeaders(t, response)
			assertBasicInvalidClientChallenge(t, response)
			assert.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())
		})
	}
}

func TestHandlerRevokeReturnsServerErrorWhenPersistenceFails(t *testing.T) {
	router, _, _, redisFixture := newIdentityHandlerTestRouterWithFixture(t, true, nil)

	tokenForm := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {openplatform.ScopeResourceRead},
	}
	tokenResponse := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(tokenForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	var tokenPayload map[string]any
	require.NoError(t, json.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	accessToken, _ := tokenPayload["access_token"].(string)
	require.NotEmpty(t, accessToken)

	redisFixture.Server.SetError("forced-failure")
	revokeForm := url.Values{"token": {accessToken}, "token_type_hint": {"access_token"}}
	revoke := performIdentityRequest(router, http.MethodPost, "/oauth2/revoke", strings.NewReader(revokeForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})

	require.Equal(t, http.StatusServiceUnavailable, revoke.Code)
	assertOAuthNoStoreHeaders(t, revoke)
	assert.JSONEq(t, `{"error":"server_error"}`, revoke.Body.String())
}

func TestHandlerContinueAuthorizeRequiresAuthentication(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/continue?token=consent-token", nil)

	require.Equal(t, http.StatusFound, response.Code)
	location := response.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	assert.Equal(t, "/login", parsed.Path)
	assert.Equal(t, "/oauth2/continue?token=consent-token", parsed.Query().Get("redirect"))
}

func TestHandlerContinueAuthorizeRejectsInvalidTokenQuery(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		target        string
	}{
		{
			name:          "missing token unauthenticated",
			authenticated: false,
			target:        "/oauth2/continue",
		},
		{
			name:          "blank token authenticated",
			authenticated: true,
			target:        "/oauth2/continue?token=%20%20%20",
		},
		{
			name:          "repeated token unauthenticated",
			authenticated: false,
			target:        "/oauth2/continue?token=consent-token&token=other-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, _ := newIdentityHandlerTestRouter(t, tt.authenticated)

			response := performIdentityRequest(router, http.MethodGet, tt.target, nil)

			require.Equal(t, http.StatusBadRequest, response.Code)
			assert.Equal(t, "invalid authorization request", response.Body.String())
			assert.Empty(t, response.Header().Get("Location"))
		})
	}
}

func newIdentityHandlerTestRouter(t *testing.T, authenticated bool) (*gin.Engine, *Service, *fakeIdentityOpenPlatform) {
	return newIdentityHandlerTestRouterWithRevoker(t, authenticated, nil)
}

func newIdentityHandlerTestRouterWithRevoker(t *testing.T, authenticated bool, revoker sessionRevoker) (*gin.Engine, *Service, *fakeIdentityOpenPlatform) {
	router, service, gateway, _ := newIdentityHandlerTestRouterWithFixture(t, authenticated, revoker)
	return router, service, gateway
}

func newIdentityHandlerTestRouterWithFixture(t *testing.T, authenticated bool, revoker sessionRevoker) (*gin.Engine, *Service, *fakeIdentityOpenPlatform, *redisfixture.Fixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	handler := NewHandler(service, gateway, "https://id.example.com", "https://id.example.com",
		func(_ context.Context, externalSubject string) (int64, error) {
			if externalSubject == "casdoor-subject-1" {
				return 42, nil
			}
			return 0, openplatform.ErrDisclosureUnavailable
		},
	)
	if revoker != nil {
		handler.SetSessionRevoker(revoker, ".example.com", true)
	}
	router := gin.New()
	optionalAuth := func(c *gin.Context) {
		if authenticated {
			c.Set(middleware.CtxKeyUserID, "casdoor-subject-1")
			middleware.SetAuthenticationTime(c, time.Now().Add(-time.Minute))
		}
		c.Next()
	}
	handler.RegisterRoutes(router, optionalAuth)
	return router, service, gateway, fixture
}

type fakeIdentitySessionRevoker struct {
	sessionID    string
	userID       string
	accessToken  string
	refreshToken string
	err          error
}

func (f *fakeIdentitySessionRevoker) RevokeSession(_ context.Context, sessionID, userID, accessToken, refreshToken string) error {
	f.sessionID = sessionID
	f.userID = userID
	f.accessToken = accessToken
	f.refreshToken = refreshToken
	return f.err
}

func performIdentityRequest(router http.Handler, method, target string, body *strings.Reader, mutators ...func(*http.Request)) *httptest.ResponseRecorder {
	var requestBody *strings.Reader
	if body == nil {
		requestBody = strings.NewReader("")
	} else {
		requestBody = body
	}
	req := httptest.NewRequest(method, target, requestBody)
	for _, mutate := range mutators {
		mutate(req)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func assertOAuthNoStoreHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	headers := response.Result().Header
	assert.Equal(t, "no-store", headers.Get("Cache-Control"))
	assert.Equal(t, "no-cache", headers.Get("Pragma"))
}

func assertBasicInvalidClientChallenge(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, `Basic realm="StuHelper Identity"`, response.Result().Header.Get("WWW-Authenticate"))
}

func assertBearerInvalidTokenChallenge(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, `Bearer realm="StuHelper Identity", error="invalid_token"`, response.Result().Header.Get("WWW-Authenticate"))
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
