package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

const authBlacklistLookupTimeout = 50 * time.Millisecond

// OptionalAuthConfig 可选认证中间件的配置。
// Cookie 无效时中间件需要清理浏览器端 cookie，为此需要 cookie 的 domain/secure 属性。
type OptionalAuthConfig struct {
	CookieDomain string
	CookieSecure bool
}

type RoleScopeResolver interface {
	ResolveRoleScopes(ctx context.Context, casdoorSubject string, roles []string) (map[string][]string, error)
}

// resolveToken 认证哨兵错误
var (
	errNoToken              = errors.New("missing token")
	errTokenRevoked         = errors.New("token revoked")
	errBlacklistFail        = errors.New("blacklist unavailable")
	errRoleScopeUnavailable = errors.New("role scope unavailable")
)

// resolveToken 从请求中提取、验证并解析 Token。
// Cookie Token → 本地 JWKS 验证（高性能，适合浏览器客户端）
// Bearer Token → OIDC provider introspection 验证（即时吊销能力，适合 API 客户端）
func resolveToken(c *gin.Context, oidcClient *oidc.Client, tokenService *token.Service) (*authResult, error) {
	tokenString, source := getTokenWithSource(c)
	if tokenString == "" {
		return nil, errNoToken
	}

	// 检查 token 是否在应用级黑名单中（紧急吊销，两种来源都检查）
	blacklistCtx, cancel := context.WithTimeout(c.Request.Context(), authBlacklistLookupTimeout)
	defer cancel()
	isBlacklisted, err := tokenService.GetBlacklist().IsBlacklisted(blacklistCtx, tokenString)
	if err != nil {
		reason := "unavailable"
		if errors.Is(blacklistCtx.Err(), context.DeadlineExceeded) {
			reason = "timeout"
		}
		metrics.ObserveAuthBlacklistFailure(reason)
		return nil, errBlacklistFail
	}
	if isBlacklisted {
		return nil, errTokenRevoked
	}

	switch source {
	case tokenSourceCookie:
		// 自签名 JWT（HS256，手机验证码登录签发）优先检查：
		// HMAC 验证是纯内存操作，比远程 JWKS/introspection 快几个数量级
		if token.IsSelfSignedToken(tokenString) {
			hmacKey := crypto.GetHMACKey()
			if len(hmacKey) == 0 {
				return nil, fmt.Errorf("self-signed token present but HMAC key not initialized")
			}
			claims, verifyErr := token.VerifyJWTWithType(hmacKey, tokenString, token.JWTTokenTypeAccess)
			if verifyErr != nil {
				logger.L().Debug("self-signed JWT verification failed", zap.Error(verifyErr))
				return nil, fmt.Errorf("invalid token: %w", verifyErr)
			}
			var avatarPtr *string
			if claims.Avatar != "" {
				avatarPtr = &claims.Avatar
			}
			return &authResult{
				userID:      claims.Sub,
				username:    claims.Name,
				email:       claims.Email,
				displayName: claims.DisplayName,
				avatar:      avatarPtr,
				roles:       claims.Roles,
			}, nil
		}

		// OIDC ID Token（RS256/ES256 由 provider 配置决定）— 通过 JWKS 本地验证
		claims, verifyErr := oidcClient.VerifyIDToken(c.Request.Context(), tokenString)
		if verifyErr != nil {
			logger.L().Debug("OIDC token verification failed", zap.Error(verifyErr))
			return nil, fmt.Errorf("invalid token: %w", verifyErr)
		}
		return &authResult{
			userID:         claims.GetUserID(),
			appID:          claims.GetAppID(),
			username:       claims.GetUsername(),
			email:          claims.GetEmail(),
			displayName:    claims.GetDisplayName(),
			avatar:         claims.GetAvatar(),
			roles:          claims.Roles,
			orgScopedRoles: claims.OrgScopedRoles,
			mfaProofAt:     claims.MFAProofVerifiedAt(),
		}, nil

	case tokenSourceBearer:
		result, introErr := oidcClient.IntrospectToken(c.Request.Context(), tokenString)
		if introErr != nil {
			logger.L().Debug("token introspection failed", zap.Error(introErr))
			return nil, fmt.Errorf("invalid token: %w", introErr)
		}
		if !result.Active {
			return nil, errTokenRevoked
		}
		return &authResult{
			userID:         result.Sub,
			appID:          result.GetAppID(),
			username:       result.Username,
			email:          result.Email,
			displayName:    result.Name,
			roles:          result.Roles,
			orgScopedRoles: result.OrgScopedRoles,
			mfaProofAt:     result.MFAProofVerifiedAt(),
		}, nil
	}

	return nil, errNoToken
}

func AuthMiddleware(oidcClient *oidc.Client, tokenService *token.Service) gin.HandlerFunc {
	return AuthMiddlewareWithRoleScopeResolver(oidcClient, tokenService, nil)
}

// AuthMiddlewareWithRoleScopeResolver 强制认证并补全 DB/OpenFGA role scope。
func AuthMiddlewareWithRoleScopeResolver(
	oidcClient *oidc.Client,
	tokenService *token.Service,
	scopeResolver RoleScopeResolver,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := resolveToken(c, oidcClient, tokenService)
		if err == nil {
			result, err = withResolvedRoleScopes(c.Request.Context(), result, scopeResolver)
		}
		if err != nil {
			switch {
			case errors.Is(err, errNoToken):
				response.Unauthorized(c, "missing authentication token", errs.ErrTokenMissing)
			case authBackendUnavailable(err):
				response.ServiceUnavailable(c, "service temporarily unavailable", errs.ErrServiceUnavailable)
			case errors.Is(err, errTokenRevoked):
				response.Unauthorized(c, "token has been revoked", errs.ErrTokenRevoked)
			default:
				response.Unauthorized(c, "invalid or expired token", errs.ErrTokenInvalid)
			}
			c.Abort()
			return
		}
		setClaimsToContext(c, result)
		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件。
// 四分支处理：
//  1. 无 token → 匿名继续。
//  2. Cookie token 无效/已撤销 → 清 cookie + 记录日志 + 匿名继续
//     （cookie 由浏览器自动附带，不应因过期 cookie 把公开页面打成 401）。
//  3. Bearer token 无效/已撤销 → 返回 401（bearer 是主动认证请求）。
//  4. 后端故障（黑名单/Redis 不可用）→ 注入诊断标记到 context 供
//     handler 按路由敏感度决定是否拒绝，匿名继续。
func OptionalAuthMiddleware(oidcClient *oidc.Client, tokenService *token.Service, cfg OptionalAuthConfig) gin.HandlerFunc {
	return OptionalAuthMiddlewareWithRoleScopeResolver(oidcClient, tokenService, cfg, nil)
}

func OptionalAuthMiddlewareWithRoleScopeResolver(
	oidcClient *oidc.Client,
	tokenService *token.Service,
	cfg OptionalAuthConfig,
	scopeResolver RoleScopeResolver,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, source := getTokenWithSource(c)
		if source == tokenSourceNone {
			c.Next()
			return
		}

		result, err := resolveToken(c, oidcClient, tokenService)
		if err == nil {
			result, err = withResolvedRoleScopes(c.Request.Context(), result, scopeResolver)
		}
		if err == nil {
			setClaimsToContext(c, result)
			c.Next()
			return
		}

		switch {
		case authBackendUnavailable(err):
			// 后端故障：不应把故障降级为匿名；注入标记由路由决定
			logger.FromGin(c).Warn("optional auth: backend unavailable", zap.Error(err))
			c.Set(CtxKeyAuthBackendFailure, true)
			c.Next()

		case errors.Is(err, errTokenRevoked):
			if source == tokenSourceBearer {
				response.Unauthorized(c, "token has been revoked", errs.ErrTokenRevoked)
				c.Abort()
				return
			}
			// Cookie 被撤销：清除 + 匿名继续
			logger.FromGin(c).Debug("optional auth: clearing revoked cookie token")
			clearAuthCookies(c, cfg)
			c.Next()

		default:
			// token 无效或已过期
			if source == tokenSourceBearer {
				response.Unauthorized(c, "invalid or expired token", errs.ErrTokenInvalid)
				c.Abort()
				return
			}
			logger.FromGin(c).Debug("optional auth: clearing invalid cookie token", zap.Error(err))
			clearAuthCookies(c, cfg)
			c.Next()
		}
	}
}

func authBackendUnavailable(err error) bool {
	return errors.Is(err, errBlacklistFail) ||
		errors.Is(err, errRoleScopeUnavailable) ||
		errors.Is(err, oidc.ErrProviderUnavailable)
}

// RequireHealthyOptionalAuth 在 optional auth 之后执行。
// 如果请求携带了认证态，但认证后端故障导致无法可靠判定身份，则返回 503，
// 避免业务路由把真实故障静默降级成匿名访问。
func RequireHealthyOptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool(CtxKeyAuthBackendFailure) {
			c.Next()
			return
		}

		response.ServiceUnavailable(c, "authentication service temporarily unavailable", errs.ErrServiceUnavailable)
		c.Abort()
	}
}

// clearAuthCookies 把 access / refresh cookie 清除（MaxAge=-1）。
// 浏览器匹配通过 (name, domain, path) 三元组——这里用 handler_cookies.go
// 写入时使用的同一路径组合。
func clearAuthCookies(c *gin.Context, cfg OptionalAuthConfig) {
	c.SetCookie(CookieAccessToken, "", -1, CookieAccessTokenPath, cfg.CookieDomain, cfg.CookieSecure, true)
	c.SetCookie(CookieRefreshToken, "", -1, CookieRefreshTokenPath, cfg.CookieDomain, cfg.CookieSecure, true)
}
