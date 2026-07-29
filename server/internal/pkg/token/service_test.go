package token

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

// ---------------------------------------------------------------------------
// Service constructor & accessors
// ---------------------------------------------------------------------------

func TestNewService_Valid(t *testing.T) {
	fixture := redisfixture.Start(t)

	svc, err := NewService(ServiceConfig{
		RedisClient: fixture.Client,
		AccessTTL:   3600,
		RefreshTTL:  86400,
	})
	require.NoError(t, err)
	defer svc.Close()

	assert.Equal(t, 3600*time.Second, svc.GetAccessTokenTTL())
	assert.Equal(t, 86400*time.Second, svc.GetRefreshTokenTTL())
	assert.NotNil(t, svc.GetBlacklist())
}

func TestNewService_MissingRedis(t *testing.T) {
	_, err := NewService(ServiceConfig{
		RedisClient: nil,
		AccessTTL:   3600,
		RefreshTTL:  86400,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RedisClient is required")
}

func TestNewService_InvalidTTL(t *testing.T) {
	fixture := redisfixture.Start(t)

	tests := []struct {
		name       string
		accessTTL  int
		refreshTTL int
		wantErr    string
	}{
		{"zero AccessTTL", 0, 86400, "AccessTTL must be > 0"},
		{"negative AccessTTL", -1, 86400, "AccessTTL must be > 0"},
		{"zero RefreshTTL", 3600, 0, "RefreshTTL must be > 0"},
		{"negative RefreshTTL", 3600, -1, "RefreshTTL must be > 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(ServiceConfig{
				RedisClient: fixture.Client,
				AccessTTL:   tt.accessTTL,
				RefreshTTL:  tt.refreshTTL,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestService_Close_Nil(t *testing.T) {
	// Close on nil service must not panic.
	var svc *Service
	assert.NotPanics(t, func() { svc.Close() })
}

func TestService_Close_Idempotent(t *testing.T) {
	fixture := redisfixture.Start(t)

	svc, err := NewService(ServiceConfig{
		RedisClient: fixture.Client,
		AccessTTL:   3600,
		RefreshTTL:  86400,
	})
	require.NoError(t, err)

	// Calling Close multiple times must not panic.
	assert.NotPanics(t, func() {
		svc.Close()
		svc.Close()
	})
}

// ---------------------------------------------------------------------------
// JWT: SignJWT
// ---------------------------------------------------------------------------

func TestSignJWT_ValidToken(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	claims := JWTClaims{
		Sub:         "user-123",
		Name:        "Test User",
		DisplayName: "Test",
		Roles:       []string{"user"},
		Typ:         JWTTokenTypeAccess,
	}

	tok, err := SignJWT(key, claims, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	// Token must have 3 dot-separated parts.
	parts := splitJWT(tok)
	assert.Len(t, parts, 3)
}

func TestSignJWT_EmptyKey(t *testing.T) {
	claims := JWTClaims{Sub: "u1", Name: "n"}
	_, err := SignJWT(nil, claims, time.Hour)
	assert.ErrorIs(t, err, ErrJWTKeyMissing)
}

func TestSignJWT_MissingSubject(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	_, err := SignJWT(key, JWTClaims{Name: "n"}, time.Hour)
	assert.ErrorIs(t, err, ErrJWTClaimMissing)
}

func TestSignJWT_MissingName(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	_, err := SignJWT(key, JWTClaims{Sub: "u1"}, time.Hour)
	assert.ErrorIs(t, err, ErrJWTClaimMissing)
}

func TestSignJWT_SetsIssuerAndTimestamps(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	before := time.Now().Unix()

	tok, err := SignJWT(key, JWTClaims{
		Sub:  "user-1",
		Name: "User",
		Typ:  JWTTokenTypeAccess,
	}, 2*time.Hour)
	require.NoError(t, err)

	after := time.Now().Unix()

	claims, err := VerifyJWT(key, tok)
	require.NoError(t, err)
	assert.Equal(t, SelfSignedIssuer, claims.Iss)
	assert.GreaterOrEqual(t, claims.Iat, before)
	assert.LessOrEqual(t, claims.Iat, after)
	// Exp should be approximately Iat + 2h (7200 seconds).
	assert.InDelta(t, float64(claims.Iat+7200), float64(claims.Exp), 2)
}

// ---------------------------------------------------------------------------
// JWT: VerifyJWT
// ---------------------------------------------------------------------------

func TestVerifyJWT_Valid(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	original := JWTClaims{
		Sub:         "user-42",
		Name:        "Alice",
		DisplayName: "Alice W",
		Email:       "alice@example.com",
		Roles:       []string{"admin", "user"},
		Typ:         JWTTokenTypeAccess,
	}

	tok, err := SignJWT(key, original, time.Hour)
	require.NoError(t, err)

	parsed, err := VerifyJWT(key, tok)
	require.NoError(t, err)

	assert.Equal(t, original.Sub, parsed.Sub)
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.DisplayName, parsed.DisplayName)
	assert.Equal(t, original.Email, parsed.Email)
	assert.Equal(t, original.Roles, parsed.Roles)
	assert.Equal(t, original.Typ, parsed.Typ)
}

func TestVerifyJWT_EmptyKey(t *testing.T) {
	_, err := VerifyJWT(nil, "a.b.c")
	assert.ErrorIs(t, err, ErrJWTKeyMissing)
}

func TestVerifyJWT_Malformed(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"one part", "abc"},
		{"two parts", "abc.def"},
		{"bad base64 header", "!!!.abc.def"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyJWT(key, tt.token)
			assert.ErrorIs(t, err, ErrJWTMalformed)
		})
	}
}

func TestVerifyJWT_UnsupportedAlgorithm(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")

	// Construct a token with RS256 header but signed with HMAC.
	header := base64URLEncode([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64URLEncode([]byte(`{"sub":"u1","name":"n","iss":"stuhelper","exp":9999999999,"iat":1000000000}`))
	sigInput := header + "." + payload
	sig := base64URLEncode(hmacSHA256(key, []byte(sigInput)))

	tok := sigInput + "." + sig
	_, err := VerifyJWT(key, tok)
	assert.ErrorIs(t, err, ErrJWTUnsupported)
}

func TestVerifyJWT_WrongSigningKey(t *testing.T) {
	key1 := []byte("correct-key-at-least-32-characters")
	key2 := []byte("wrong-key---at-least-32-characters")

	tok, err := SignJWT(key1, JWTClaims{Sub: "u1", Name: "n"}, time.Hour)
	require.NoError(t, err)

	_, err = VerifyJWT(key2, tok)
	assert.ErrorIs(t, err, ErrJWTSignature)
}

func TestVerifyJWT_Expired(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")

	// A freshly signed 1s-TTL token should still be valid (within 30s clock skew).
	tok, err := SignJWT(key, JWTClaims{Sub: "u1", Name: "n"}, 1*time.Second)
	require.NoError(t, err)
	_, err = VerifyJWT(key, tok)
	assert.NoError(t, err)

	// Construct a token that is 1 hour past expiry (well beyond 30s skew).
	expiredTok := forceExpiredToken(t, key)
	_, err = VerifyJWT(key, expiredTok)
	assert.ErrorIs(t, err, ErrJWTExpired)
}

func TestVerifyJWT_WrongIssuer(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")

	// Construct a valid-signature token but with wrong issuer.
	claims := JWTClaims{
		Sub:  "u1",
		Name: "n",
		Iss:  "wrong-issuer",
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(time.Hour).Unix(),
	}
	payloadJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	payloadB64 := base64URLEncode(payloadJSON)
	sigInput := jwtHeaderB64 + "." + payloadB64
	sig := base64URLEncode(hmacSHA256(key, []byte(sigInput)))

	tok := sigInput + "." + sig
	_, err = VerifyJWT(key, tok)
	// Wrong issuer returns ErrJWTSignature per implementation.
	assert.ErrorIs(t, err, ErrJWTSignature)
}

// ---------------------------------------------------------------------------
// JWT: VerifyJWTWithType
// ---------------------------------------------------------------------------

func TestVerifyJWTWithType_Correct(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")

	tok, err := SignJWT(key, JWTClaims{
		Sub:  "u1",
		Name: "n",
		Typ:  JWTTokenTypeRefresh,
	}, time.Hour)
	require.NoError(t, err)

	claims, err := VerifyJWTWithType(key, tok, JWTTokenTypeRefresh)
	require.NoError(t, err)
	assert.Equal(t, JWTTokenTypeRefresh, claims.Typ)
}

func TestVerifyJWTWithType_Mismatch(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")

	tok, err := SignJWT(key, JWTClaims{
		Sub:  "u1",
		Name: "n",
		Typ:  JWTTokenTypeRefresh,
	}, time.Hour)
	require.NoError(t, err)

	_, err = VerifyJWTWithType(key, tok, JWTTokenTypeAccess)
	assert.ErrorIs(t, err, ErrJWTTokenType)
}

func TestVerifyJWTWithType_PropagatesVerifyError(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	_, err := VerifyJWTWithType(key, "malformed", JWTTokenTypeAccess)
	assert.ErrorIs(t, err, ErrJWTMalformed)
}

// ---------------------------------------------------------------------------
// IsSelfSignedToken
// ---------------------------------------------------------------------------

func TestIsSelfSignedToken(t *testing.T) {
	key := []byte("test-signing-key-at-least-32-chars")
	tok, err := SignJWT(key, JWTClaims{Sub: "u1", Name: "n"}, time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid HS256 token", tok, true},
		{"not a JWT", "not-a-jwt", false},
		{"two parts only", "a.b", false},
		{"empty", "", false},
		{"RS256 header", base64URLEncode([]byte(`{"alg":"RS256","typ":"JWT"}`)) + ".payload.sig", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSelfSignedToken(tt.token))
		})
	}
}

// ---------------------------------------------------------------------------
// Blacklist: additional coverage beyond blacklist_test.go
// ---------------------------------------------------------------------------

func TestBlacklist_Add_TTLBounds(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret-ttl-bounds", false))

	fixture := setupTestRedis(t)

	bl := NewBlacklist(fixture.Client)
	defer bl.Close()

	ctx := t.Context()

	// TTL below minimum
	err := bl.Add(ctx, "tok1", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of valid range")

	// TTL above maximum (31 days > 30 day max)
	err = bl.Add(ctx, "tok2", 31*24*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of valid range")
}

func TestBlacklist_TryConsumeRefreshToken_TTLBounds(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret-consume-ttl", false))

	fixture := setupTestRedis(t)

	bl := NewBlacklist(fixture.Client)
	defer bl.Close()

	ctx := t.Context()

	_, err := bl.TryConsumeRefreshToken(ctx, "tok", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of valid range")
}

func TestBlacklist_CircuitBreakerMetrics(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-cb-metrics-secret", false))

	fixture := setupTestRedis(t)

	bl := NewBlacklist(fixture.Client)
	defer bl.Close()

	m := bl.CircuitBreakerMetrics()
	assert.NotNil(t, m)
	assert.Contains(t, m, "state")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// splitJWT splits a token by '.' for structural checks.
func splitJWT(tok string) []string {
	var parts []string
	start := 0
	for i := range tok {
		if tok[i] == '.' {
			parts = append(parts, tok[start:i])
			start = i + 1
		}
	}
	parts = append(parts, tok[start:])
	return parts
}

// forceExpiredToken creates a token whose exp is 1 hour in the past (well beyond
// the 30-second clock skew tolerance).
func forceExpiredToken(t *testing.T, key []byte) string {
	t.Helper()

	claims := JWTClaims{
		Sub:  "expired-user",
		Name: "Expired",
		Iss:  SelfSignedIssuer,
		Iat:  time.Now().Add(-2 * time.Hour).Unix(),
		Exp:  time.Now().Add(-1 * time.Hour).Unix(),
	}

	payloadJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	payloadB64 := base64URLEncode(payloadJSON)
	sigInput := jwtHeaderB64 + "." + payloadB64
	sig := hmacSHA256(key, []byte(sigInput))
	sigB64 := base64URLEncode(sig)

	return sigInput + "." + sigB64
}
