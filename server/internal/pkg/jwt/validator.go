package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 允许的签名算法白名单
var allowedAlgorithms = map[string]bool{
	"RS256": true,
	"RS384": true,
	"RS512": true,
	"ES256": true,
	"ES384": true,
	"ES512": true,
}

// 常见错误
var (
	ErrInvalidToken       = errors.New("invalid token format")
	ErrAlgorithmNotAllowed = errors.New("algorithm not allowed")
	ErrInvalidIssuer      = errors.New("invalid issuer")
	ErrInvalidAudience    = errors.New("invalid audience")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenNotYetValid   = errors.New("token not yet valid")
	ErrInvalidSignature   = errors.New("invalid signature")
)

// ValidatorConfig JWT 验证器配置
type ValidatorConfig struct {
	Issuer       string        // 期望的 issuer (Casdoor endpoint)
	Audience     string        // 期望的 audience (client ID)
	Certificate  string        // PEM 格式的公钥证书
	ClockSkew    time.Duration // 允许的时钟偏移
}

// Validator JWT 验证器
type Validator struct {
	config    ValidatorConfig
	publicKey *rsa.PublicKey
}

// NewValidator 创建 JWT 验证器
func NewValidator(cfg ValidatorConfig) (*Validator, error) {
	if cfg.ClockSkew == 0 {
		cfg.ClockSkew = 30 * time.Second // 默认 30 秒时钟偏移
	}

	v := &Validator{config: cfg}

	// 解析公钥证书
	if cfg.Certificate != "" {
		pubKey, err := parseCertificate(cfg.Certificate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		v.publicKey = pubKey
	}

	return v, nil
}

// Claims 自定义 JWT Claims
type Claims struct {
	jwt.RegisteredClaims
	// Casdoor 特定字段
	Owner       string `json:"owner,omitempty"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	IsAdmin     bool   `json:"isAdmin,omitempty"`
}

// Validate 验证 JWT Token
func (v *Validator) Validate(tokenString string) (*Claims, error) {
	// 1. 预检查 token 格式和算法
	if err := v.preValidate(tokenString); err != nil {
		return nil, err
	}

	// 2. 解析并验证 token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, v.keyFunc,
		jwt.WithLeeway(v.config.ClockSkew),
		jwt.WithIssuer(v.config.Issuer),
		jwt.WithAudience(v.config.Audience),
	)

	if err != nil {
		return nil, v.mapError(err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// 3. 额外的时间验证
	if err := v.validateTime(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// preValidate 预检查 token 格式和算法
func (v *Validator) preValidate(tokenString string) error {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	// 解码 header 检查算法
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidToken
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return ErrInvalidToken
	}

	// 检查算法是否在白名单中
	if header.Alg == "" || header.Alg == "none" {
		return ErrAlgorithmNotAllowed
	}
	if !allowedAlgorithms[header.Alg] {
		return fmt.Errorf("%w: %s", ErrAlgorithmNotAllowed, header.Alg)
	}

	return nil
}

// keyFunc 返回用于验证签名的密钥
func (v *Validator) keyFunc(token *jwt.Token) (interface{}, error) {
	// 再次检查算法
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	}

	if v.publicKey == nil {
		return nil, errors.New("public key not configured")
	}

	return v.publicKey, nil
}

// validateTime 额外的时间验证
func (v *Validator) validateTime(claims *Claims) error {
	now := time.Now()

	// 检查 exp
	if claims.ExpiresAt != nil {
		if now.After(claims.ExpiresAt.Time.Add(v.config.ClockSkew)) {
			return ErrTokenExpired
		}
	}

	// 检查 nbf
	if claims.NotBefore != nil {
		if now.Before(claims.NotBefore.Time.Add(-v.config.ClockSkew)) {
			return ErrTokenNotYetValid
		}
	}

	// 检查 iat (不应该在未来)
	if claims.IssuedAt != nil {
		if claims.IssuedAt.Time.After(now.Add(v.config.ClockSkew)) {
			return ErrTokenNotYetValid
		}
	}

	return nil
}

// mapError 将 jwt 库错误映射为自定义错误
func (v *Validator) mapError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return ErrTokenNotYetValid
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return ErrInvalidSignature
	case errors.Is(err, jwt.ErrTokenMalformed):
		return ErrInvalidToken
	default:
		// 检查是否是 issuer 或 audience 错误
		errStr := err.Error()
		if strings.Contains(errStr, "issuer") {
			return ErrInvalidIssuer
		}
		if strings.Contains(errStr, "audience") {
			return ErrInvalidAudience
		}
		return fmt.Errorf("token validation failed: %w", err)
	}
}

// parseCertificate 解析 PEM 格式的证书或公钥
func parseCertificate(certPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate does not contain RSA public key")
		}
		return pubKey, nil

	case "PUBLIC KEY", "RSA PUBLIC KEY":
		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			// 尝试解析为 PKCS1 格式
			rsaPubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key: %w", err)
			}
			return rsaPubKey, nil
		}
		rsaPubKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("not an RSA public key")
		}
		return rsaPubKey, nil

	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

// GetUserID 从 claims 获取用户 ID
func (c *Claims) GetUserID() string {
	return c.Subject
}

// GetUsername 从 claims 获取用户名
func (c *Claims) GetUsername() string {
	return c.Name
}
