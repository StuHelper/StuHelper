package identityserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
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
	assert.Contains(t, metadata["response_types_supported"], "code")
	assert.Contains(t, metadata["scopes_supported"], "openid")
	assert.Contains(t, metadata["scopes_supported"], "profile.basic.read")

	jwks := performIdentityRequest(router, http.MethodGet, "/.well-known/jwks.json", nil)
	require.Equal(t, http.StatusOK, jwks.Code)
	var keySet map[string][]map[string]any
	require.NoError(t, json.Unmarshal(jwks.Body.Bytes(), &keySet))
	require.Len(t, keySet["keys"], 1)
	assert.Equal(t, "identity-test-key", keySet["keys"][0]["kid"])
	assert.Equal(t, "sig", keySet["keys"][0]["use"])
}

func TestHandlerAuthorizeUnauthenticatedRedirectsToFrontendLogin(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, false)
	authorizePath := "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-1"

	response := performIdentityRequest(router, http.MethodGet, authorizePath, nil)

	require.Equal(t, http.StatusFound, response.Code)
	location := response.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com", parsed.Scheme+"://"+parsed.Host)
	assert.Equal(t, "/login", parsed.Path)
	assert.Equal(t, authorizePath, parsed.Query().Get("redirect"))
}

func TestHandlerAuthorizeAuthenticatedIssuesCodeRedirect(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	response := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile&state=state-2", nil)

	require.Equal(t, http.StatusFound, response.Code)
	callback, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.NotEmpty(t, callback.Query().Get("code"))
	assert.Equal(t, "state-2", callback.Query().Get("state"))
}

func TestHandlerTokenUserInfoIntrospectAndRevoke(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)
	authorizeResponse := performIdentityRequest(router, http.MethodGet, "/oauth2/authorize?response_type=code&client_id=client-1&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=openid+profile+email&state=state-3", nil)
	require.Equal(t, http.StatusFound, authorizeResponse.Code)
	callback, err := url.Parse(authorizeResponse.Header().Get("Location"))
	require.NoError(t, err)
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	tokenForm := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {"https://client.example.com/callback"},
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
	assert.NotEmpty(t, tokenPayload["id_token"])
	assert.Equal(t, "Bearer", tokenPayload["token_type"])
	assert.Equal(t, "openid profile email", tokenPayload["scope"])

	missingBearer := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo", nil)
	require.Equal(t, http.StatusUnauthorized, missingBearer.Code)
	assert.JSONEq(t, `{"error":"invalid_token"}`, missingBearer.Body.String())

	userInfo := performIdentityRequest(router, http.MethodGet, "/oidc/userinfo", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	})
	require.Equal(t, http.StatusOK, userInfo.Code)
	var userInfoPayload map[string]any
	require.NoError(t, json.Unmarshal(userInfo.Body.Bytes(), &userInfoPayload))
	assert.Equal(t, "alice", userInfoPayload["preferred_username"])
	assert.Equal(t, "alice@example.com", userInfoPayload["email"])
	assert.Equal(t, true, userInfoPayload["email_verified"])

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

	revokeForm := url.Values{"token": {accessToken}}
	revoke := performIdentityRequest(router, http.MethodPost, "/oauth2/revoke", strings.NewReader(revokeForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, revoke.Code)

	introspectionAfterRevoke := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(introspectionForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusOK, introspectionAfterRevoke.Code)
	var revokedPayload map[string]any
	require.NoError(t, json.Unmarshal(introspectionAfterRevoke.Body.Bytes(), &revokedPayload))
	assert.Equal(t, false, revokedPayload["active"])
}

func TestHandlerOAuthErrors(t *testing.T) {
	router, _, _ := newIdentityHandlerTestRouter(t, true)

	unsupportedGrant := url.Values{
		"grant_type": {"client_credentials"},
	}
	unsupported := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(unsupportedGrant.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "secret-1")
	})
	require.Equal(t, http.StatusBadRequest, unsupported.Code)
	assert.JSONEq(t, `{"error":"unsupported_grant_type"}`, unsupported.Body.String())

	invalidClientForm := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"missing-code"},
	}
	invalidClient := performIdentityRequest(router, http.MethodPost, "/oauth2/token", strings.NewReader(invalidClientForm.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "wrong-secret")
	})
	require.Equal(t, http.StatusUnauthorized, invalidClient.Code)
	assert.JSONEq(t, `{"error":"invalid_client"}`, invalidClient.Body.String())

	badIntrospectClient := performIdentityRequest(router, http.MethodPost, "/oauth2/introspect", strings.NewReader(url.Values{"token": {"token"}}.Encode()), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("client-1", "wrong-secret")
	})
	require.Equal(t, http.StatusUnauthorized, badIntrospectClient.Code)
	assert.JSONEq(t, `{"error":"invalid_client"}`, badIntrospectClient.Body.String())
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

func newIdentityHandlerTestRouter(t *testing.T, authenticated bool) (*gin.Engine, *Service, *fakeIdentityOpenPlatform) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	handler := NewHandler(service, gateway, "https://id.example.com", "https://id.example.com",
		func(_ context.Context, externalSubject string) (int64, error) {
			if externalSubject == "casdoor-subject-1" {
				return 42, nil
			}
			return 0, openplatform.ErrDisclosureUnavailable
		},
	)
	router := gin.New()
	optionalAuth := func(c *gin.Context) {
		if authenticated {
			c.Set(middleware.CtxKeyUserID, "casdoor-subject-1")
		}
		c.Next()
	}
	handler.RegisterRoutes(router, optionalAuth)
	return router, service, gateway
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
