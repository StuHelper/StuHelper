package token

import (
	"time"

	"github.com/redis/go-redis/v9"
)

// Service Token 服务
type Service struct {
	blacklist       *Blacklist
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewService 创建 Token 服务
func NewService(rdb *redis.Client, accessTTL, refreshTTL int) *Service {
	return &Service{
		blacklist:       NewBlacklist(rdb),
		accessTokenTTL:  time.Duration(accessTTL) * time.Second,
		refreshTokenTTL: time.Duration(refreshTTL) * time.Second,
	}
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
