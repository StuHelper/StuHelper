package token

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

// Service Token 管理服务
// Zitadel 架构下不再自行验证 JWT，验证由 OIDC 客户端的 VerifyIDToken 处理。
// 本服务管理 Token 黑名单（紧急吊销）、Session Store（Token Family）和 TTL 配置。
type Service struct {
	blacklist       *Blacklist
	sessionStore    *SessionStore
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// ServiceConfig Token 服务配置
type ServiceConfig struct {
	RedisClient *redis.Client
	AccessTTL   int // 秒
	RefreshTTL  int // 秒
}

// NewService 创建 Token 管理服务
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.RedisClient == nil {
		return nil, fmt.Errorf("token service: RedisClient is required")
	}
	if cfg.AccessTTL <= 0 {
		return nil, fmt.Errorf("token service: AccessTTL must be > 0 (got %d)", cfg.AccessTTL)
	}
	if cfg.RefreshTTL <= 0 {
		return nil, fmt.Errorf("token service: RefreshTTL must be > 0 (got %d)", cfg.RefreshTTL)
	}

	refreshTTL := time.Duration(cfg.RefreshTTL) * time.Second

	return &Service{
		blacklist:       NewBlacklist(cfg.RedisClient),
		sessionStore:    NewSessionStore(cfg.RedisClient, refreshTTL),
		accessTokenTTL:  time.Duration(cfg.AccessTTL) * time.Second,
		refreshTokenTTL: refreshTTL,
	}, nil
}

// GetBlacklist 获取黑名单服务
func (s *Service) GetBlacklist() *Blacklist {
	return s.blacklist
}

// GetSessionStore 获取 session store
func (s *Service) GetSessionStore() *SessionStore {
	return s.sessionStore
}

// GetAccessTokenTTL 获取 Access Token TTL
func (s *Service) GetAccessTokenTTL() time.Duration {
	effectiveSeconds := systemconfig.EffectiveAuthAccessTokenTTL(int(s.accessTokenTTL / time.Second))
	return time.Duration(effectiveSeconds) * time.Second
}

// GetRefreshTokenTTL 获取 Refresh Token TTL
func (s *Service) GetRefreshTokenTTL() time.Duration {
	return s.refreshTokenTTL
}

// Close 优雅关闭 token 服务的后台资源。
func (s *Service) Close() {
	if s == nil || s.blacklist == nil {
		return
	}
	s.blacklist.Close()
}
