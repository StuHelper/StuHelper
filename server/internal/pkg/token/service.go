package token

import (
	"time"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

// Service Token 服务
type Service struct {
	blacklist       *Blacklist
	jwtValidator    *jwt.Validator
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// ServiceConfig Token 服务配置
type ServiceConfig struct {
	RedisClient     *redis.Client
	AccessTTL       int    // 秒
	RefreshTTL      int    // 秒
	JWTIssuer       string // Casdoor endpoint
	JWTAudience     string // Client ID
	JWTCertificate  string // PEM 格式公钥
}

// NewService 创建 Token 服务
func NewService(cfg ServiceConfig) (*Service, error) {
	// 创建 JWT 验证器
	jwtValidator, err := jwt.NewValidator(jwt.ValidatorConfig{
		Issuer:      cfg.JWTIssuer,
		Audience:    cfg.JWTAudience,
		Certificate: cfg.JWTCertificate,
		ClockSkew:   30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		blacklist:       NewBlacklist(cfg.RedisClient),
		jwtValidator:    jwtValidator,
		accessTokenTTL:  time.Duration(cfg.AccessTTL) * time.Second,
		refreshTokenTTL: time.Duration(cfg.RefreshTTL) * time.Second,
	}, nil
}

// ValidateToken 验证 JWT Token
func (s *Service) ValidateToken(tokenString string) (*jwt.Claims, error) {
	return s.jwtValidator.Validate(tokenString)
}

// GetBlacklist 获取黑名单服务
func (s *Service) GetBlacklist() *Blacklist {
	return s.blacklist
}

// GetAccessTokenTTL 获取 Access Token TTL
func (s *Service) GetAccessTokenTTL() time.Duration {
	return s.accessTokenTTL
}

// GetRefreshTokenTTL 获取 Refresh Token TTL
func (s *Service) GetRefreshTokenTTL() time.Duration {
	return s.refreshTokenTTL
}
