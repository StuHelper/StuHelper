package identityserver

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignerVerifyAccessTokenBindsAudienceAndAuthorizedPartyToClientID(t *testing.T) {
	signer, err := NewSigner("https://id.example.com", "identity-test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name    string
		aud     any
		azp     any
		wantErr string
	}{
		{
			name: "matching string audience",
			aud:  "client-1",
			azp:  "client-1",
		},
		{
			name: "matching array audience",
			aud:  []string{"resource-api", "client-1"},
			azp:  "client-1",
		},
		{
			name:    "missing audience",
			azp:     "client-1",
			wantErr: "identity token audience is invalid",
		},
		{
			name:    "mismatched audience",
			aud:     "client-2",
			azp:     "client-1",
			wantErr: "identity token audience is invalid",
		},
		{
			name:    "missing authorized party",
			aud:     "client-1",
			wantErr: "identity token authorized party is invalid",
		},
		{
			name:    "mismatched authorized party",
			aud:     "client-1",
			azp:     "client-2",
			wantErr: "identity token authorized party is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := signIdentityTestClaims(t, signer, jwt.MapClaims{
				"iss":          "https://id.example.com",
				"sub":          "stuhelper:42",
				"client_id":    "client-1",
				"scope":        "openid profile",
				"stuhelper_id": int64(42),
				"typ":          "access",
				"jti":          "jti-1",
			}, tt.aud, tt.azp)

			claims, err := signer.VerifyAccessToken(raw)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, claims)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "stuhelper:42", claims.Subject)
			assert.Equal(t, "client-1", claims.ClientID)
			assert.Equal(t, int64(42), claims.UserID)
			assert.Equal(t, []string{"openid", "profile"}, claims.Scopes)
			assert.Equal(t, "jti-1", claims.JTI)
		})
	}
}

func TestSignerVerifyIDTokenRejectsAuthorizedPartyOutsideAudience(t *testing.T) {
	signer, err := NewSigner("https://id.example.com", "identity-test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name    string
		aud     any
		azp     any
		wantErr string
	}{
		{
			name: "matching authorized party",
			aud:  "client-1",
			azp:  "client-1",
		},
		{
			name: "matching array audience returns authorized party client",
			aud:  []string{"resource-api", "client-1"},
			azp:  "client-1",
		},
		{
			name:    "multiple audience requires authorized party",
			aud:     []string{"resource-api", "client-1"},
			wantErr: "identity id token authorized party is invalid",
		},
		{
			name:    "missing audience",
			azp:     "client-1",
			wantErr: "identity id token required claim is missing",
		},
		{
			name:    "authorized party outside audience",
			aud:     "client-1",
			azp:     "client-2",
			wantErr: "identity id token authorized party is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := signIdentityTestClaims(t, signer, jwt.MapClaims{
				"iss":       "https://id.example.com",
				"sub":       "stuhelper:42",
				"auth_time": time.Now().Add(-time.Minute).UTC().Unix(),
			}, tt.aud, tt.azp)

			claims, err := signer.VerifyIDToken(raw)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, claims)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "stuhelper:42", claims.Subject)
			assert.Equal(t, "client-1", claims.ClientID)
		})
	}
}

func TestSignerIDTokenTypePreventsAccessTokenConfusion(t *testing.T) {
	signer, err := NewSigner("https://id.example.com", "identity-test-key", "")
	require.NoError(t, err)

	idToken, err := signer.SignIDToken(IDTokenInput{
		Subject:  "stuhelper:42",
		ClientID: "client-1",
		TTL:      time.Hour,
	})
	require.NoError(t, err)
	parsed, err := jwt.Parse(idToken, func(token *jwt.Token) (any, error) {
		return &signer.privateKey.PublicKey, nil
	})
	require.NoError(t, err)
	idClaims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, "id_token", idClaims["typ"])

	verified, err := signer.VerifyIDToken(idToken)
	require.NoError(t, err)
	assert.Equal(t, "client-1", verified.ClientID)

	legacyIDToken := signIdentityTestClaims(t, signer, jwt.MapClaims{
		"iss":       "https://id.example.com",
		"sub":       "stuhelper:42",
		"auth_time": time.Now().Add(-time.Minute).UTC().Unix(),
	}, "client-1", "client-1")
	legacy, err := signer.VerifyIDToken(legacyIDToken)
	require.NoError(t, err)
	assert.Equal(t, "client-1", legacy.ClientID)

	accessToken, _, err := signer.SignAccessToken(AccessTokenInput{
		Subject:  "stuhelper:42",
		ClientID: "client-1",
		UserID:   42,
		Scopes:   []string{"openid", "profile"},
		TTL:      time.Hour,
	})
	require.NoError(t, err)
	_, err = signer.VerifyIDToken(accessToken)
	require.ErrorContains(t, err, "identity id token type is invalid")
}

func signIdentityTestClaims(t *testing.T, signer *Signer, claims jwt.MapClaims, aud any, azp any) string {
	t.Helper()
	now := time.Now().UTC()
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(time.Hour).Unix()
	if aud != nil {
		claims["aud"] = aud
	}
	if azp != nil {
		claims["azp"] = azp
	}
	raw, err := signer.signClaims(claims)
	require.NoError(t, err)
	return raw
}
