// Package token 的 jwt.go 提供自签名 JWT 创建和验证能力。
// 用于手机验证码登录等不经过外部 OIDC 流程的认证场景。
// 算法：HMAC-SHA256（HS256），签名密钥来自 HMAC_SECRET。
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// JWTTokenType 自签名 JWT 的 token 用途类型
type JWTTokenType string

const (
	JWTTokenTypeAccess  JWTTokenType = "access"
	JWTTokenTypeRefresh JWTTokenType = "refresh"
)

// JWT 错误
var (
	ErrJWTMalformed    = errors.New("jwt: malformed token")
	ErrJWTExpired      = errors.New("jwt: token expired")
	ErrJWTSignature    = errors.New("jwt: invalid signature")
	ErrJWTUnsupported  = errors.New("jwt: unsupported algorithm")
	ErrJWTKeyMissing   = errors.New("jwt: signing key not provided")
	ErrJWTClaimMissing = errors.New("jwt: required claim missing")
	ErrJWTTokenType    = errors.New("jwt: unexpected token type")
)

// SelfSignedIssuer 自签名 JWT 的 issuer 标识
const SelfSignedIssuer = "stuhelper"

const jwtClockSkew = 30 * time.Second

// JWTClaims 自签名 JWT 的 Claims
type JWTClaims struct {
	Sub         string       `json:"sub"`
	Name        string       `json:"name"`
	Email       string       `json:"email,omitempty"`
	DisplayName string       `json:"display_name"`
	Avatar      string       `json:"avatar,omitempty"`
	Phone       string       `json:"phone,omitempty"`
	Roles       []string     `json:"roles,omitempty"`
	Typ         JWTTokenType `json:"typ,omitempty"`
	// Sid Session ID — 标识 token 所属的服务端 session（Token Family 模式）
	Sid string `json:"sid,omitempty"`
	Iss string `json:"iss"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

// jwtHeader HS256 固定头
var jwtHeaderB64 = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

// SignJWT 签发一个 HMAC-SHA256 JWT。
// key 为 HMAC_SECRET 密钥，ttl 为有效期。
func SignJWT(key []byte, claims JWTClaims, ttl time.Duration) (string, error) {
	if len(key) == 0 {
		return "", ErrJWTKeyMissing
	}
	if claims.Sub == "" || claims.Name == "" {
		return "", ErrJWTClaimMissing
	}

	now := time.Now()
	claims.Iss = SelfSignedIssuer
	claims.Iat = now.Unix()
	claims.Exp = now.Add(ttl).Unix()

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("jwt: marshal claims: %w", err)
	}
	payloadB64 := base64URLEncode(payloadJSON)

	signingInput := jwtHeaderB64 + "." + payloadB64
	sig := hmacSHA256(key, []byte(signingInput))
	sigB64 := base64URLEncode(sig)

	return signingInput + "." + sigB64, nil
}

// VerifyJWT 验证一个 HMAC-SHA256 JWT 并返回解析后的 Claims。
// 仅接受 issuer="stuhelper" 的自签名 token。
func VerifyJWT(key []byte, tokenString string) (*JWTClaims, error) {
	if len(key) == 0 {
		return nil, ErrJWTKeyMissing
	}

	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 3 {
		return nil, ErrJWTMalformed
	}

	// 验证 header
	headerJSON, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, ErrJWTMalformed
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrJWTMalformed
	}
	if header.Alg != "HS256" {
		return nil, ErrJWTUnsupported
	}

	// 验证签名
	signingInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256(key, []byte(signingInput))
	actualSig, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, ErrJWTMalformed
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, ErrJWTSignature
	}

	// 解析 Claims
	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, ErrJWTMalformed
	}
	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrJWTMalformed
	}

	// 验证 issuer
	if claims.Iss != SelfSignedIssuer {
		return nil, ErrJWTSignature
	}

	// 验证过期时间（允许少量时钟偏差）
	if time.Now().After(time.Unix(claims.Exp, 0).Add(jwtClockSkew)) {
		return nil, ErrJWTExpired
	}

	return &claims, nil
}

// VerifyJWTWithType 验证 JWT 并校验 token type（access / refresh）。
// 防止 refresh token 被用作 access token。
func VerifyJWTWithType(key []byte, tokenString string, expectedType JWTTokenType) (*JWTClaims, error) {
	claims, err := VerifyJWT(key, tokenString)
	if err != nil {
		return nil, err
	}
	if claims.Typ != expectedType {
		return nil, fmt.Errorf("%w: expected %q, got %q", ErrJWTTokenType, expectedType, claims.Typ)
	}
	return claims, nil
}

// IsSelfSignedToken 快速检查 token 是否为自签名 JWT（不做完整验证）。
// 通过检查 header 的 alg 字段判断：HS256 为自签名，其它算法交给 OIDC verifier。
func IsSelfSignedToken(tokenString string) bool {
	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 3 {
		return false
	}
	headerJSON, err := base64URLDecode(parts[0])
	if err != nil {
		return false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false
	}
	return header.Alg == "HS256"
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
