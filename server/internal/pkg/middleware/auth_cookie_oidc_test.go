package middleware

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
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
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

func TestAuthMiddleware_CookieOIDCRequiresSessionApplicationAudience(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	oidcClient, server, issueIDToken := newCookieOIDCClient(t)
	defer server.Close()

	rawIDToken := issueIDToken("web-client")
	storeOIDCCookieSession(t, tokenSvc, "sid-admin", "admin", rawIDToken)

	r := gin.New()
	r.Use(AuthMiddleware(oidcClient, tokenSvc))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: rawIDToken})
	req.AddCookie(&http.Cookie{Name: CookieSessionID, Value: "sid-admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_CookieOIDCAcceptsMatchingSessionApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	oidcClient, server, issueIDToken := newCookieOIDCClient(t)
	defer server.Close()

	rawIDToken := issueIDToken("admin-client")
	storeOIDCCookieSession(t, tokenSvc, "sid-admin", "admin", rawIDToken)

	r := gin.New()
	r.Use(AuthMiddleware(oidcClient, tokenSvc))
	r.GET("/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"userID": GetUserID(c)}) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: rawIDToken})
	req.AddCookie(&http.Cookie{Name: CookieSessionID, Value: "sid-admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"userID":"user-oidc-cookie"`)
}

func storeOIDCCookieSession(t *testing.T, tokenSvc *token.Service, sessionID, appKey, rawIDToken string) {
	t.Helper()
	accessHash, err := crypto.HMACHash(rawIDToken)
	require.NoError(t, err)
	err = tokenSvc.GetSessionStore().Create(context.Background(), token.SessionData{
		SessionID:       sessionID,
		UserID:          "user-oidc-cookie",
		AccessTokenHash: accessHash,
		ProviderAppKey:  appKey,
		LoginMethod:     "oidc",
	})
	require.NoError(t, err)
}

func newCookieOIDCClient(t *testing.T) (*oidc.Client, *httptest.Server, func(string) string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	jwk := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "kid-cookie", Algorithm: string(jose.RS256), Use: "sig"}
	var issuer string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	server := httptest.NewServer(mux)
	issuer = server.URL

	client, err := oidc.NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  "web-client",
		ClientSecret:              "web-secret",
		RedirectURI:               "https://web.example.com/callback",
		AdminClientID:             "admin-client",
		AdminClientSecret:         "admin-secret",
		AdminRedirectURI:          "https://admin.example.com/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	return client, server, func(audience string) string {
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: privateKey, KeyID: jwk.KeyID}}, nil)
		require.NoError(t, err)
		raw, err := josejwt.Signed(signer).Claims(map[string]any{
			"iss":                issuer,
			"sub":                "user-oidc-cookie",
			"aud":                audience,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
			"preferred_username": "cookie-user",
			"roles":              []string{"school_admin"},
		}).Serialize()
		require.NoError(t, err)
		return raw
	}
}
