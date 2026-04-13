package auth

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
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
	phoneLimiter         *middleware.RedisRateLimiter
	userSyncRepo         UserSyncRepo
	allowedRedirectHosts map[string]struct{}
	defaultRedirectURL   string
	otpService           *OTPService
	smsService           *sms.Service
	oidcIssuer           string
}

// NewHandler 创建认证处理器
func NewHandler(
	cfg *config.Config,
	tokenService *token.Service,
	rdb *redis.Client,
	oidcClient *oidc.Client,
	userSyncRepo UserSyncRepo,
	smsService *sms.Service,
) *Handler {
	svc := NewService(cfg, tokenService)

	// 从 CORS_ORIGINS 构建允许的重定向地址白名单
	redirectHosts := buildAllowedRedirectHosts(cfg.App.CORSOrigins)
	defaultRedirect := buildDefaultRedirectURL(cfg.App.CORSOrigins)

	var otpSvc *OTPService
	if smsService != nil {
		otpSvc = NewOTPService(rdb)
	}

	return &Handler{
		svc:                  svc,
		oidcClient:           oidcClient,
		tokenService:         tokenService,
		tokenConfig:          cfg.Token,
		redisClient:          rdb,
		userSyncRepo:         userSyncRepo,
		refreshLimiter:       middleware.NewRedisRateLimiter(rdb, 10, time.Minute),
		phoneLimiter:         middleware.NewRedisRateLimiter(rdb, 5, time.Minute),
		allowedRedirectHosts: redirectHosts,
		defaultRedirectURL:   defaultRedirect,
		otpService:           otpSvc,
		smsService:           smsService,
		oidcIssuer:           cfg.Zitadel.Issuer,
	}
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
// 如果 CORS_ORIGINS 未配置或为空，回退到 http://localhost:3000。
// 此回退仅适用于本地开发；生产环境 CORS_ORIGINS 由 validate() 强制要求配置，
// 同时 NewHandler 会在配置缺失时输出 WARNING 日志以便运维排查。
func buildDefaultRedirectURL(corsOrigins []string) string {
	if len(corsOrigins) > 0 {
		origin := strings.TrimRight(strings.TrimSpace(corsOrigins[0]), "/")
		if origin != "" {
			return origin
		}
	}
	return "http://localhost:3000"
}

// RegisterPublicRoutes 注册不需要 CSRF 保护的公开路由。
// 必须在 CSRF 中间件挂载之前调用，否则匿名 POST 会被拦截。
// OTP 路由已有独立限流保护。
func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	phone := r.Group("/auth/phone")
	{
		phone.POST("/request-otp", middleware.RateLimitMiddleware(h.phoneLimiter), h.RequestPhoneOTP)
		phone.POST("/verify-otp", middleware.RateLimitMiddleware(h.phoneLimiter), h.VerifyPhoneOTP)
	}
}

// RegisterRoutes 注册认证路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, oidcClient *oidc.Client, tokenService *token.Service) {
	auth := r.Group("/auth")
	{
		auth.GET("/login", h.GetLoginURL)
		auth.GET("/signup", h.GetSignupURL)
		auth.GET("/callback", h.HandleCallback)
		auth.POST("/refresh", middleware.RateLimitMiddleware(h.refreshLimiter), h.RefreshToken)
		auth.GET("/me", middleware.AuthMiddleware(oidcClient, tokenService), h.GetCurrentUser)
		auth.POST("/logout", middleware.AuthMiddleware(oidcClient, tokenService), h.Logout)
		auth.POST("/logout-all", middleware.AuthMiddleware(oidcClient, tokenService), h.LogoutAll)
	}
}
