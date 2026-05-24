package auth

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Handler 认证处理器
type Handler struct {
	svc                  *Service
	oidcClient           *oidc.Client
	tokenService         *token.Service
	tokenConfig          config.TokenConfig
	redisClient          *redis.Client
	refreshLimiter       *middleware.RedisRateLimiter
	authFailureGuard     *AuthFailureGuard
	allowedRedirectHosts map[string]struct{}
	defaultRedirectURL   string
	oidcIssuer           string
}

type HandlerConfig struct {
	Token               config.TokenConfig
	CORSOrigins         []string
	OIDCIssuer          string
	ProviderTokenCipher pii.EncryptDecryptor
}

// NewHandler 创建认证处理器
func NewHandler(
	cfg HandlerConfig,
	tokenService *token.Service,
	rdb *redis.Client,
	oidcClient *oidc.Client,
	userSyncRepo UserSyncRepo,
) *Handler {
	svc := NewService(cfg.Token, tokenService, userSyncRepo, providerTokenRevocationOptions(oidcClient, cfg)...)

	// 从 CORS_ORIGINS 构建允许的重定向地址白名单
	redirectHosts := buildAllowedRedirectHosts(cfg.CORSOrigins)
	defaultRedirect := buildDefaultRedirectURL(cfg.CORSOrigins)

	return &Handler{
		svc:                  svc,
		oidcClient:           oidcClient,
		tokenService:         tokenService,
		tokenConfig:          cfg.Token,
		redisClient:          rdb,
		refreshLimiter:       middleware.NewRedisRateLimiter(rdb, 10, time.Minute),
		authFailureGuard:     NewAuthFailureGuard(rdb),
		allowedRedirectHosts: redirectHosts,
		defaultRedirectURL:   defaultRedirect,
		oidcIssuer:           cfg.OIDCIssuer,
	}
}

func (h *Handler) SessionRevoker() *Service {
	if h == nil {
		return nil
	}
	return h.svc
}

func providerTokenRevocationOptions(oidcClient *oidc.Client, cfg HandlerConfig) []ServiceOption {
	if oidcClient == nil || cfg.ProviderTokenCipher == nil {
		return nil
	}
	supported, err := oidcClient.SupportsRefreshTokenRevocation()
	if err != nil {
		logger.L().Warn("OIDC provider refresh token revocation disabled: metadata lookup failed", zap.Error(err))
		return nil
	}
	if !supported {
		logger.L().Warn("OIDC provider refresh token revocation disabled: revocation endpoint unavailable")
		return nil
	}
	return []ServiceOption{WithProviderRefreshTokenRevocation(oidcClient, cfg.ProviderTokenCipher)}
}

// buildAllowedRedirectHosts 从 CORS_ORIGINS 提取允许的重定向 host
func buildAllowedRedirectHosts(corsOrigins []string) map[string]struct{} {
	hosts := make(map[string]struct{}, len(corsOrigins))
	for _, origin := range corsOrigins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || parsed.Host == "" {
			continue
		}
		hosts[parsed.Host] = struct{}{}
	}
	return hosts
}

// buildDefaultRedirectURL 从 CORS_ORIGINS 取第一个作为默认重定向地址。
// CORS_ORIGINS 由 config.validate() 强制要求在所有环境（含 dev）非空，
// 因此此处无需 localhost 兜底——若调用时仍为空即配置异常，立刻 panic 以便
// fail-fast 暴露。
func buildDefaultRedirectURL(corsOrigins []string) string {
	for _, raw := range corsOrigins {
		origin := strings.TrimRight(strings.TrimSpace(raw), "/")
		if origin != "" {
			return origin
		}
	}
	panic("auth.buildDefaultRedirectURL: CORS_ORIGINS must be configured (config.validate() should have caught this)")
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
