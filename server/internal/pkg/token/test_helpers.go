package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	jwtpkg "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/jwt"
)

// testKeyPair 测试用 RSA 密钥对（进程级缓存，避免重复生成）
var testKeyPair struct {
	privateKey *rsa.PrivateKey
	publicPEM  string
}

func init() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate test RSA key: " + err.Error())
	}
	testKeyPair.privateKey = key

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic("failed to marshal test public key: " + err.Error())
	}
	testKeyPair.publicPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}))
}

const (
	testIssuer   = "test-issuer"
	testAudience = "test-audience"
)

// NewServiceForTest 创建用于测试的 Token Service（使用内置 RSA 密钥对）。
// 配合 IssueTestToken 使用，可生成能通过 ValidateToken 校验的 JWT。
func NewServiceForTest(rdb *redis.Client) *Service {
	// 确保 HMAC key 已初始化（Blacklist 内部依赖）
	_ = crypto.InitHMACKey("test-token-helper-secret", false)

	svc, err := NewService(ServiceConfig{
		RedisClient:    rdb,
		AccessTTL:      3600,
		RefreshTTL:     86400,
		JWTIssuer:      testIssuer,
		JWTAudience:    testAudience,
		JWTCertificate: testKeyPair.publicPEM,
	})
	if err != nil {
		panic("NewServiceForTest: " + err.Error())
	}
	return svc
}

// IssueTestToken 签发一个有效的测试 JWT，可通过 AuthMiddleware 校验。
func (s *Service) IssueTestToken(t *testing.T) string {
	t.Helper()

	now := time.Now()
	claims := jwtpkg.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			Subject:   "test-user-id",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
		ID:          "test-user-id",
		Name:        "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
		IsAdmin:     false,
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(testKeyPair.privateKey)
	require.NoError(t, err)
	return signed
}
