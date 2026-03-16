package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Handler 认证处理器
type Handler struct {
	ssoClient        *sso.Client
	tokenService     *token.Service
	tokenConfig      config.TokenConfig
	redirectURI      string
	appName          string
	ssoEndpoint      string
	refreshLimiter   *middleware.RedisRateLimiter
	userSyncRepo     UserSyncRepo
	capabilityReader CapabilityReader
}

// NewHandler 创建认证处理器
func NewHandler(
	cfg *config.Config,
	tokenService *token.Service,
	rdb *redis.Client,
	ssoClient *sso.Client,
	userSyncRepo UserSyncRepo,
	capabilityReader CapabilityReader,
) *Handler {
	return &Handler{
		ssoClient:        ssoClient,
		tokenService:     tokenService,
		tokenConfig:      cfg.Token,
		redirectURI:      cfg.Casdoor.RedirectURI,
		appName:          cfg.Casdoor.Application,
		ssoEndpoint:      cfg.Casdoor.Endpoint,
		userSyncRepo:     userSyncRepo,
		capabilityReader: capabilityReader,
		// RefreshToken 限制: 每分钟最多 10 次
		refreshLimiter: middleware.NewRedisRateLimiter(rdb, 10, time.Minute),
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.GET("/login", h.GetLoginURL)
		auth.GET("/signup", h.GetSignupURL)
		auth.GET("/callback", h.HandleCallback)
		auth.POST("/refresh", middleware.RateLimitMiddleware(h.refreshLimiter), h.RefreshToken)
		auth.GET("/me", middleware.AuthMiddleware(h.tokenService), h.GetCurrentUser)
		auth.POST("/logout", middleware.AuthMiddleware(h.tokenService), h.Logout)
		auth.POST("/logout-all", middleware.AuthMiddleware(h.tokenService), h.LogoutAll)
	}
}
