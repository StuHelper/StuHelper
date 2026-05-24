package auth

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	oidcpkg "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

type fakeOIDCProvider struct {
	server   *httptest.Server
	client   *oidcpkg.Client
	clientID string
}

func newFakeOIDCProvider(t *testing.T) *fakeOIDCProvider {
	return newFakeOIDCProviderWithTokenPayload(t, nil)
}

func newFakeOIDCProviderWithTokenPayload(t *testing.T, payloadFn func(issueIDToken func() string) map[string]any) *fakeOIDCProvider {
	t.Helper()
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)

	const clientID = "test-client-id"
	const clientSecret = "test-client-secret"
	jwk := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "test-key", Algorithm: string(jose.RS256), Use: "sig"}

	var issuer string
	issueIDToken := func(audience string) string {
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: privateKey, KeyID: jwk.KeyID}}, nil)
		require.NoError(t, err)
		raw, err := josejwt.Signed(signer).Claims(map[string]any{
			"iss":                issuer,
			"sub":                "oidc-user-1",
			"aud":                audience,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
			"name":               "OIDC Tester",
			"preferred_username": "oidc-tester",
			"email":              "oidc@example.com",
			"email_verified":     true,
			"picture":            "https://cdn.example.com/oidc.png",
			"roles":              []string{"school_admin"},
		}).Serialize()
		require.NoError(t, err)
		return raw
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, issuer+"/callback?code=code-1&state=state-1", http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		issueForRequestClient := func() string {
			return issueIDToken(requestOIDCClientID(r, clientID))
		}
		w.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
			"id_token":      issueForRequestClient(),
		}
		if payloadFn != nil {
			payload = payloadFn(issueForRequestClient)
		}
		_ = json.NewEncoder(w).Encode(payload)
	})

	srv := httptest.NewServer(mux)
	issuer = srv.URL
	t.Cleanup(srv.Close)

	client, err := oidcpkg.NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              clientSecret,
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		UniappClientID:            "uniapp-client",
		UniappClientSecret:        "uniapp-secret",
		UniappRedirectURI:         "stuhelper://auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)

	return &fakeOIDCProvider{server: srv, client: client, clientID: clientID}
}

func requestOIDCClientID(r *http.Request, fallback string) string {
	clientID, _, ok := r.BasicAuth()
	if ok && clientID != "" {
		return clientID
	}
	if formClientID := r.FormValue("client_id"); formClientID != "" {
		return formClientID
	}
	return fallback
}

func newOIDCTestHandler(t *testing.T, repo UserSyncRepo) (*Handler, *recordingUserSyncRepo) {
	return newOIDCTestHandlerWithProvider(t, repo, newFakeOIDCProvider(t))
}

func newOIDCTestHandlerWithProvider(t *testing.T, repo UserSyncRepo, provider *fakeOIDCProvider) (*Handler, *recordingUserSyncRepo) {
	t.Helper()
	var recordingRepo *recordingUserSyncRepo
	if r, ok := repo.(*recordingUserSyncRepo); ok {
		recordingRepo = r
	}
	if repo == nil {
		recordingRepo = &recordingUserSyncRepo{}
		repo = recordingRepo
	}
	h, _ := newRefreshTestHandler(t, repo)
	h.oidcClient = provider.client
	h.defaultRedirectURL = "https://web.example.com"
	h.allowedRedirectHosts = map[string]struct{}{"web.example.com": {}}
	h.tokenConfig = config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}
	return h, recordingRepo
}

type failingOIDCUserSyncRepo struct{}

func (failingOIDCUserSyncRepo) UpsertUser(context.Context, UserSyncInput) error {
	return errors.New("sync failed")
}
func (failingOIDCUserSyncRepo) ExistsByCasdoorSubject(context.Context, string) (bool, error) {
	return true, nil
}

func TestHandleWebCallback_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repo := newOIDCTestHandler(t, &recordingUserSyncRepo{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)

	h.handleWebCallback(c, c.Request.Context(), webCallbackInput{
		code:         "code-1",
		redirect:     "/dashboard",
		codeVerifier: "verifier-1",
		application:  oidcpkg.ApplicationWeb,
		requestID:    "request-1",
	})

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://web.example.com/dashboard", w.Header().Get("Location"))
	assert.Equal(t, "oidc-user-1", repo.upsertInput.CasdoorSubject)
	assert.Equal(t, "oidc-tester", repo.upsertInput.Username)
	assert.Equal(t, "oidc@example.com", repo.upsertInput.Email)
	require.NotNil(t, repo.upsertInput.AvatarURL)
	assert.Equal(t, "https://cdn.example.com/oidc.png", *repo.upsertInput.AvatarURL)

	cookies := w.Result().Cookies()
	var hasAccess, hasRefresh, hasSession bool
	for _, cookie := range cookies {
		switch cookie.Name {
		case "access_token":
			hasAccess = true
		case "refresh_token":
			hasRefresh = true
		case sessionCookieName:
			hasSession = true
		}
	}
	assert.True(t, hasAccess)
	assert.True(t, hasRefresh)
	assert.True(t, hasSession)
}

func TestRefreshOIDCToken_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newOIDCTestHandler(t, &recordingUserSyncRepo{})
	ctx := context.Background()

	_, err := h.svc.CreateSession(ctx, "sid-oidc-refresh", "oidc-user-1", "old-access-token", "old-refresh-token", "oidc", "browser")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-oidc-refresh"})
	c.Request = req

	ok := h.refreshOIDCToken(c, "old-refresh-token")
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "token refreshed successfully")
	var hasAccessCookie, hasRefreshCookie, hasSessionCookie bool
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "access_token" {
			hasAccessCookie = true
		}
		if cookie.Name == "refresh_token" {
			hasRefreshCookie = true
		}
		if cookie.Name == sessionCookieName {
			hasSessionCookie = true
			assert.Equal(t, "sid-oidc-refresh", cookie.Value)
		}
	}
	assert.True(t, hasAccessCookie)
	assert.True(t, hasRefreshCookie)
	assert.True(t, hasSessionCookie)

	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(ctx, "old-refresh-token")
	require.NoError(t, err)
	assert.True(t, blacklisted)

	session, err := h.tokenService.GetSessionStore().Get(ctx, "sid-oidc-refresh")
	require.NoError(t, err)
	assert.Equal(t, "sid-oidc-refresh", session.SessionID)
	assert.NotEmpty(t, session.RefreshTokenHash)
}

func TestRefreshOIDCToken_RotationFailureDoesNotIssueCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newOIDCTestHandler(t, &recordingUserSyncRepo{})
	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-oidc-refresh-mismatch",
		"other-user",
		"old-access-token",
		"old-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-oidc-refresh-mismatch"})
	c.Request = req

	ok := h.refreshOIDCToken(c, "old-refresh-token")

	require.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assertNoIssuedTokenCookies(t, w)
}

func TestRefreshOIDCToken_RejectsMissingProviderRefreshRotation(t *testing.T) {
	assertOIDCRefreshRotationUnavailable(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token": "provider-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     issueIDToken(),
		}
	})
}

func TestRefreshOIDCToken_RejectsSameProviderRefreshToken(t *testing.T) {
	assertOIDCRefreshRotationUnavailable(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "old-refresh-token",
			"expires_in":    3600,
			"id_token":      issueIDToken(),
		}
	})
}

func TestRefreshToken_NativeOIDCUsesSessionHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newOIDCTestHandler(t, &recordingUserSyncRepo{})
	ctx := context.Background()

	_, err := h.svc.CreateSession(ctx, "sid-native-oidc-refresh", "oidc-user-1", "old-access-token", "old-refresh-token", "oidc-native", "ios")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	body := bytes.NewBufferString(`{"refreshToken":"old-refresh-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nativeSessionIDHeader, "sid-native-oidc-refresh")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "accessToken")
	assert.Contains(t, w.Body.String(), "refreshToken")

	session, err := h.tokenService.GetSessionStore().Get(ctx, "sid-native-oidc-refresh")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, "sid-native-oidc-refresh", session.SessionID)
	assert.NotEmpty(t, session.RefreshTokenHash)
}

func TestRefreshToken_NativeOIDCRequiresTrackedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newOIDCTestHandler(t, &recordingUserSyncRepo{})
	ctx := context.Background()

	_, err := h.svc.CreateSession(ctx, "sid-native-oidc-refresh", "oidc-user-1", "old-access-token", "old-refresh-token", "oidc-native", "ios")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	t.Run("missing session header is rejected before rotation", func(t *testing.T) {
		body := bytes.NewBufferString(`{"refreshToken":"old-refresh-token"}`)
		req := httptest.NewRequest(http.MethodPost, "/refresh", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), "missing native session id")

		blacklisted, blErr := h.tokenService.GetBlacklist().IsBlacklisted(ctx, "old-refresh-token")
		require.NoError(t, blErr)
		assert.False(t, blacklisted)
	})

	t.Run("unknown session header is rejected before oidc refresh", func(t *testing.T) {
		body := bytes.NewBufferString(`{"refreshToken":"old-refresh-token"}`)
		req := httptest.NewRequest(http.MethodPost, "/refresh", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(nativeSessionIDHeader, "sid-does-not-exist")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), "invalid native session id")

		blacklisted, blErr := h.tokenService.GetBlacklist().IsBlacklisted(ctx, "old-refresh-token")
		require.NoError(t, blErr)
		assert.False(t, blacklisted)
	})
}

func TestExchangeNative_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repo := newOIDCTestHandler(t, &recordingUserSyncRepo{})
	require.NoError(t, h.storeNativeCodeVerifier(context.Background(), "native-state-1", nativeCodeVerifierPayload{
		CodeVerifier: "native-verifier",
		Application:  oidcpkg.ApplicationUniapp,
	}))

	body := bytes.NewBufferString(`{"code":"code-1","state":"native-state-1"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/exchange-native", body)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExchangeNative(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "accessToken")
	assert.Contains(t, w.Body.String(), "refreshToken")
	assert.Contains(t, w.Body.String(), "sessionID")
	assert.Equal(t, "oidc-user-1", repo.upsertInput.CasdoorSubject)
}

func TestHandleWebCallback_MissingIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
		}
	})
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)

	h.handleWebCallback(c, c.Request.Context(), webCallbackInput{
		code:         "code-1",
		redirect:     "/dashboard",
		codeVerifier: "verifier-1",
		application:  oidcpkg.ApplicationWeb,
		requestID:    "request-1",
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication failed")
	assert.Empty(t, w.Header().Get("Location"))
}

func TestHandleWebCallback_UserSyncFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newOIDCTestHandlerWithProvider(t, failingOIDCUserSyncRepo{}, newFakeOIDCProvider(t))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)

	h.handleWebCallback(c, c.Request.Context(), webCallbackInput{
		code:         "code-1",
		redirect:     "/dashboard",
		codeVerifier: "verifier-1",
		application:  oidcpkg.ApplicationWeb,
		requestID:    "request-1",
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication failed")
	assert.Empty(t, w.Header().Get("Location"))
}

func TestRefreshOIDCToken_MissingIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
		}
	})
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)
	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-oidc-missing-id-token",
		"oidc-user-1",
		"old-access-token",
		"old-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-oidc-missing-id-token"})
	c.Request = req

	ok := h.refreshOIDCToken(c, "old-refresh-token")
	assert.False(t, ok)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to refresh token")
}

func TestHandleWebCallback_InvalidIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
			"id_token":      "bad-id-token",
		}
	})
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)

	h.handleWebCallback(c, c.Request.Context(), webCallbackInput{
		code:         "code-1",
		redirect:     "/dashboard",
		codeVerifier: "verifier-1",
		application:  oidcpkg.ApplicationWeb,
		requestID:    "request-1",
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication failed")
}

func assertOIDCRefreshRotationUnavailable(
	t *testing.T,
	payloadFn func(issueIDToken func() string) map[string]any,
) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, payloadFn)
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)

	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-oidc-unrotated-refresh",
		"oidc-user-1",
		"old-access-token",
		"old-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-oidc-unrotated-refresh"})
	c.Request = req

	ok := h.refreshOIDCToken(c, "old-refresh-token")
	assert.False(t, ok)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "refresh token rotation unavailable")
}

func TestRefreshOIDCToken_InvalidIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
			"id_token":      "bad-id-token",
		}
	})
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)
	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-oidc-invalid-id-token",
		"oidc-user-1",
		"old-access-token",
		"old-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-oidc-invalid-id-token"})
	c.Request = req

	ok := h.refreshOIDCToken(c, "old-refresh-token")
	assert.False(t, ok)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to refresh token")
}

func TestExchangeNative_MissingIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
		}
	})
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)
	require.NoError(t, h.storeNativeCodeVerifier(context.Background(), "native-state-missing-id", nativeCodeVerifierPayload{
		CodeVerifier: "native-verifier",
		Application:  oidcpkg.ApplicationUniapp,
	}))

	body := bytes.NewBufferString(`{"code":"code-1","state":"native-state-missing-id"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/exchange-native", body)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExchangeNative(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication failed")
}

func TestExchangeNative_UserSyncFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newOIDCTestHandlerWithProvider(t, failingOIDCUserSyncRepo{}, newFakeOIDCProvider(t))
	require.NoError(t, h.storeNativeCodeVerifier(context.Background(), "native-state-sync-fail", nativeCodeVerifierPayload{
		CodeVerifier: "native-verifier",
		Application:  oidcpkg.ApplicationUniapp,
	}))

	body := bytes.NewBufferString(`{"code":"code-1","state":"native-state-sync-fail"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/exchange-native", body)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExchangeNative(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication failed")
}

func TestExchangeNative_InvalidIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newFakeOIDCProviderWithTokenPayload(t, func(issueIDToken func() string) map[string]any {
		return map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
			"id_token":      "bad-id-token",
		}
	})
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)
	require.NoError(t, h.storeNativeCodeVerifier(context.Background(), "native-state-bad-id-token", nativeCodeVerifierPayload{
		CodeVerifier: "native-verifier",
		Application:  oidcpkg.ApplicationUniapp,
	}))

	body := bytes.NewBufferString(`{"code":"code-1","state":"native-state-bad-id-token"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/exchange-native", body)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExchangeNative(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication failed")
}
