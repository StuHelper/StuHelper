package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestOIDCVerifierAcceptsES256FromTheLocalAllowList(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	require.NoError(t, err)

	const clientID = "oidc-es256-client"
	jwk := jose.JSONWebKey{
		Key:       &privateKey.PublicKey,
		KeyID:     "kid-es256",
		Algorithm: string(jose.ES256),
		Use:       "sig",
	}

	var issuer string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/keys",
			"introspection_endpoint":                issuer + "/introspect",
			"id_token_signing_alg_values_supported": []string{"ES256"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	issuer = server.URL

	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              "oidc-es256-secret",
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "oidc-es256-introspection-client",
		IntrospectionClientSecret: "oidc-es256-introspection-secret",
	})
	require.NoError(t, err)

	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.ES256,
		Key: jose.JSONWebKey{
			Key:   privateKey,
			KeyID: jwk.KeyID,
		},
	}, nil)
	require.NoError(t, err)
	raw, err := josejwt.Signed(signer).Claims(map[string]any{
		"iss": issuer,
		"sub": "user-es256",
		"aud": clientID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}).Serialize()
	require.NoError(t, err)

	claims, err := client.VerifyIDTokenForApplication(context.Background(), ApplicationWeb, raw)
	require.NoError(t, err)
	assert.Equal(t, "user-es256", claims.GetUserID())
}
