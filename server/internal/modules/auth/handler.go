package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
)

var ErrProviderRefreshTokenRevocationUnavailable = errors.New(
	"auth: provider refresh token revocation unavailable",
)

// Handler 认证处理器
type Handler struct {
	svc                    *Service
	oidcClient             *oidc.Client
	oidcSubjectValidator   OIDCSubjectValidator
	tokenService           *token.Service
	tokenConfig            config.TokenConfig
	redisClient            *redis.Client
	refreshLimiter         *middleware.RedisRateLimiter
	authFailureGuard       *AuthFailureGuard
	adminAuthorizers       AdminAuthorizers
	allowedRedirectHosts   map[string]struct{}
	defaultRedirectURL     string
	oidcIssuer             string
	accountSettingsBaseURL string
}

type HandlerConfig struct {
	Token                  config.TokenConfig
	CORSOrigins            []string
	OIDCIssuer             string
	AccountSettingsBaseURL string
	ProviderTokenCipher    pii.EncryptDecryptor
	OIDCSubjectValidator   OIDCSubjectValidator
}

type OIDCSubjectValidator interface {
	ValidateOIDCSubject(ctx context.Context, subject string) error
}

type AdminAuthorizers struct {
	AccountLockUpdate gin.HandlerFunc
	StepUpMFA         gin.HandlerFunc
}

type HandlerOption func(*Handler)

func WithAdminAuthorizers(authorizers AdminAuthorizers) HandlerOption {
	return func(h *Handler) {
		h.adminAuthorizers = authorizers
	}
}

// NewHandler 创建认证处理器
func NewHandler(
	cfg HandlerConfig,
	tokenService *token.Service,
	rdb *redis.Client,
	oidcClient *oidc.Client,
	userSyncRepo UserSyncRepo,
	opts ...HandlerOption,
) (*Handler, error) {
	providerOptions, err := providerTokenRevocationOptions(oidcClient, cfg)
	if err != nil {
		return nil, err
	}
	svc := NewService(cfg.Token, tokenService, userSyncRepo, providerOptions...)

	// 从 CORS_ORIGINS 构建允许的重定向地址白名单
	redirectHosts := buildAllowedRedirectHosts(cfg.CORSOrigins)
	defaultRedirect := buildDefaultRedirectURL(cfg.CORSOrigins)

	h := &Handler{
		svc:                    svc,
		oidcClient:             oidcClient,
		oidcSubjectValidator:   cfg.OIDCSubjectValidator,
		tokenService:           tokenService,
		tokenConfig:            cfg.Token,
		redisClient:            rdb,
		refreshLimiter:         middleware.NewRedisRateLimiter(rdb, 10, time.Minute),
		authFailureGuard:       NewAuthFailureGuard(rdb),
		allowedRedirectHosts:   redirectHosts,
		defaultRedirectURL:     defaultRedirect,
		oidcIssuer:             cfg.OIDCIssuer,
		accountSettingsBaseURL: cfg.AccountSettingsBaseURL,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h, nil
}

func (h *Handler) SessionRevoker() *Service {
	if h == nil {
		return nil
	}
	return h.svc
}

func providerTokenRevocationOptions(oidcClient *oidc.Client, cfg HandlerConfig) ([]ServiceOption, error) {
	if oidcClient == nil {
		return nil, fmt.Errorf("%w: OIDC client is required", ErrProviderRefreshTokenRevocationUnavailable)
	}
	if cfg.ProviderTokenCipher == nil {
		return nil, fmt.Errorf("%w: provider token cipher is required", ErrProviderRefreshTokenRevocationUnavailable)
	}
	supported, err := oidcClient.SupportsRefreshTokenRevocation()
	if err != nil {
		return nil, fmt.Errorf(
			"%w: metadata lookup failed: %w",
			ErrProviderRefreshTokenRevocationUnavailable,
			err,
		)
	}
	if !supported {
		return nil, fmt.Errorf(
			"%w: revocation endpoint is not advertised",
			ErrProviderRefreshTokenRevocationUnavailable,
		)
	}
	return []ServiceOption{WithProviderRefreshTokenRevocation(oidcClient, cfg.ProviderTokenCipher)}, nil
}

// buildAllowedRedirectHosts 从 CORS_ORIGINS 提取允许的重定向 host
func buildAllowedRedirectHosts(corsOrigins []string) map[string]struct{} {
	hosts := make(map[string]struct{}, len(corsOrigins))
	for _, origin := range corsOrigins {
		_, host, ok := normalizeRedirectOrigin(origin)
		if !ok {
			continue
		}
		hosts[host] = struct{}{}
	}
	return hosts
}

// buildDefaultRedirectURL 从 CORS_ORIGINS 取第一个作为默认重定向地址。
// CORS_ORIGINS 由 config.validate() 强制要求在所有环境（含 dev）非空，
// 因此此处无需 localhost 兜底——若调用时仍为空即配置异常，立刻 panic 以便
// fail-fast 暴露。
func buildDefaultRedirectURL(corsOrigins []string) string {
	for _, raw := range corsOrigins {
		if origin, _, ok := normalizeRedirectOrigin(raw); ok {
			return origin
		}
	}
	panic("auth.buildDefaultRedirectURL: CORS_ORIGINS must be configured (config.validate() should have caught this)")
}

func normalizeRedirectOrigin(raw string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", "", false
	}
	host := strings.ToLower(parsed.Host)
	if host == "" {
		return "", "", false
	}
	return scheme + "://" + host, host, true
}

// RegisterPublicRoutes 注册不需要 CSRF 保护的公开路由。
// 必须在 CSRF 中间件挂载之前调用，否则匿名 POST 会被拦截。
func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	// 原生 App 令牌交换：无 cookie / 无 CSRF，用一次性 state 做防重放
	r.POST("/auth/exchange-native", middleware.EndpointRateLimitMiddleware(h.refreshLimiter, "auth-exchange-native"), h.ExchangeNative)
}

// RegisterRoutes 注册认证路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, oidcClient *oidc.Client, tokenService *token.Service) {
	h.RegisterRoutesWithAuthMiddleware(r, middleware.AuthMiddleware(oidcClient, tokenService))
}

func (h *Handler) RegisterRoutesWithAuthMiddleware(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	auth := r.Group("/auth")
	{
		auth.GET("/login", h.GetLoginURL)
		auth.GET("/signup", h.GetSignupURL)
		auth.GET("/step-up", authMW, h.GetStepUpURL)
		auth.GET("/callback", h.HandleCallback)
		auth.POST("/refresh", middleware.EndpointRateLimitMiddleware(h.refreshLimiter, "auth-refresh"), h.RefreshToken)
		auth.GET("/me", authMW, h.GetCurrentUser)
		auth.POST("/logout", authMW, h.Logout)
		auth.POST("/logout-all", authMW, h.LogoutAll)
	}
}
