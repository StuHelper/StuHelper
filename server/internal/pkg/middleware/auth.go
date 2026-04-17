package middleware

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// 上下文键名常量
const (
	CtxKeyUserID             = "user_id"
	CtxKeyUsername           = "username"
	CtxKeyEmail              = "email"
	CtxKeyDisplayName        = "display_name"
	CtxKeyAvatar             = "avatar"
	CtxKeyRoles              = "roles"
	CtxKeyOrgScopedRoles     = "org_scoped_roles"     // map[string][]string — Zitadel 多租户作用域
	CtxKeyCapabilities       = "capabilities"
	CtxKeyCapabilitySet      = "capability_set"       // map[string]struct{} — O(1) 查找
	CtxKeyAuthBackendFailure = "auth_backend_failure" // OptionalAuth 后端故障诊断标记
)

// OptionalAuthConfig 可选认证中间件的配置。
// Cookie 无效时中间件需要清理浏览器端 cookie，为此需要 cookie 的 domain/secure 属性。
type OptionalAuthConfig struct {
	CookieDomain string
	CookieSecure bool
}

// Cookie 名称常量
const (
	CookieAccessToken  = "access_token"
	CookieRefreshToken = "refresh_token"
)

// tokenSource 标记 Token 来源
type tokenSource int

const (
	tokenSourceNone   tokenSource = iota
	tokenSourceCookie             // Cookie: 本地 JWKS 验证（快速）
	tokenSourceBearer             // Bearer: introspection 验证（即时吊销）
)

// resolveToken 认证哨兵错误
var (
	errNoToken       = errors.New("missing token")
	errTokenRevoked  = errors.New("token revoked")
	errBlacklistFail = errors.New("blacklist unavailable")
)

// authResult 认证解析结果
type authResult struct {
	userID, username, email, displayName string
	avatar                               *string
	roles                                []string
	orgScopedRoles                       map[string][]string
}

// resolveToken 从请求中提取、验证并解析 Token。
// Cookie Token → 本地 JWKS 验证（高性能，适合浏览器客户端）
// Bearer Token → Zitadel introspection 验证（即时吊销能力，适合 API 客户端）
func resolveToken(c *gin.Context, oidcClient *oidc.Client, tokenService *token.Service) (*authResult, error) {
	tokenString, source := getTokenWithSource(c)
	if tokenString == "" {
		return nil, errNoToken
	}

	// 检查 token 是否在应用级黑名单中（紧急吊销，两种来源都检查）
	isBlacklisted, err := tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), tokenString)
	if err != nil {
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

		// Zitadel OIDC ID Token（RS256）— 通过 JWKS 本地验证
		claims, verifyErr := oidcClient.VerifyIDToken(c.Request.Context(), tokenString)
		if verifyErr != nil {
			logger.L().Debug("OIDC token verification failed", zap.Error(verifyErr))
			return nil, fmt.Errorf("invalid token: %w", verifyErr)
		}
		return &authResult{
			userID:         claims.GetUserID(),
			username:       claims.GetUsername(),
			email:          claims.GetEmail(),
			displayName:    claims.GetDisplayName(),
			avatar:         claims.GetAvatar(),
			roles:          claims.Roles,
			orgScopedRoles: claims.OrgScopedRoles,
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
			userID:      result.Sub,
			username:    result.Username,
			email:       result.Email,
			displayName: result.Name,
			roles:       result.Roles,
		}, nil
	}

	return nil, errNoToken
}

// AuthMiddleware 强制认证中间件：Token 缺失或无效时返回 401/503。
func AuthMiddleware(oidcClient *oidc.Client, tokenService *token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := resolveToken(c, oidcClient, tokenService)
		if err != nil {
			switch {
			case errors.Is(err, errNoToken):
				response.Unauthorized(c, "missing authentication token", errs.ErrTokenMissing)
			case errors.Is(err, errBlacklistFail):
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
	return func(c *gin.Context) {
		_, source := getTokenWithSource(c)
		if source == tokenSourceNone {
			c.Next()
			return
		}

		result, err := resolveToken(c, oidcClient, tokenService)
		if err == nil {
			setClaimsToContext(c, result)
			c.Next()
			return
		}

		switch {
		case errors.Is(err, errBlacklistFail):
			// 后端故障：不应把故障降级为匿名；注入标记由路由决定
			logger.FromGin(c).Warn("optional auth: blacklist backend unavailable", zap.Error(err))
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
			// invalid/expired token
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

// clearAuthCookies 把 access / refresh cookie 清除（MaxAge=-1）。
// 浏览器匹配通过 (name, domain, path) 三元组——这里用 handler_cookies.go
// 写入时使用的同一路径组合。
func clearAuthCookies(c *gin.Context, cfg OptionalAuthConfig) {
	c.SetCookie(CookieAccessToken, "", -1, "/", cfg.CookieDomain, cfg.CookieSecure, true)
	c.SetCookie(CookieRefreshToken, "", -1, "/api/v1/auth", cfg.CookieDomain, cfg.CookieSecure, true)
}

// setClaimsToContext 将用户信息、角色和能力集合注入 Gin context。
// 同时构建 capability set（map）供 HasCapability 进行 O(1) 查找。
func setClaimsToContext(c *gin.Context, auth *authResult) {
	capabilities := capability.ExpandRoles(auth.roles)
	capSet := make(map[string]struct{}, len(capabilities))
	for _, cap := range capabilities {
		capSet[cap] = struct{}{}
	}

	c.Set(CtxKeyUserID, auth.userID)
	c.Set(CtxKeyUsername, auth.username)
	c.Set(CtxKeyEmail, auth.email)
	c.Set(CtxKeyDisplayName, auth.displayName)
	if auth.avatar != nil {
		c.Set(CtxKeyAvatar, *auth.avatar)
	} else {
		c.Set(CtxKeyAvatar, "")
	}
	c.Set(CtxKeyRoles, auth.roles)
	if auth.orgScopedRoles != nil {
		c.Set(CtxKeyOrgScopedRoles, auth.orgScopedRoles)
	}
	c.Set(CtxKeyCapabilities, capabilities)
	c.Set(CtxKeyCapabilitySet, capSet)
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) string {
	return getContextString(c, CtxKeyUserID)
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	return getContextString(c, CtxKeyUsername)
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) string {
	return getContextString(c, CtxKeyEmail)
}

// GetDisplayName 从上下文获取显示名称
func GetDisplayName(c *gin.Context) string {
	return getContextString(c, CtxKeyDisplayName)
}

// GetAvatar 从上下文获取头像地址
func GetAvatar(c *gin.Context) string {
	return getContextString(c, CtxKeyAvatar)
}

// GetRoles 从上下文获取角色列表
func GetRoles(c *gin.Context) []string {
	if val, exists := c.Get(CtxKeyRoles); exists {
		if roles, ok := val.([]string); ok {
			return roles
		}
	}
	return nil
}

// GetCapabilities 从上下文获取能力列表（slice 形式，用于序列化）
func GetCapabilities(c *gin.Context) []string {
	if val, exists := c.Get(CtxKeyCapabilities); exists {
		if caps, ok := val.([]string); ok {
			return caps
		}
	}
	return nil
}

// HasCapability 检查当前用户是否具有指定能力（O(1) map 查找）
func HasCapability(c *gin.Context, capabilityName string) bool {
	if val, exists := c.Get(CtxKeyCapabilitySet); exists {
		if set, ok := val.(map[string]struct{}); ok {
			_, found := set[capabilityName]
			return found
		}
	}
	return false
}

// HasRoleInOrg 检查当前用户是否在指定 orgID 上拥有指定角色（Zitadel 多租户作用域）。
// 仅 cookie-OIDC 登录路径填充 scope；手机登录与 Bearer introspection 返回
// false。orgID 为空时判定"是否在任意 org 拥有此角色"。
func HasRoleInOrg(c *gin.Context, role, orgID string) bool {
	if val, exists := c.Get(CtxKeyOrgScopedRoles); exists {
		if scoped, ok := val.(map[string][]string); ok {
			orgs, has := scoped[role]
			if !has {
				return false
			}
			if orgID == "" {
				return len(orgs) > 0
			}
			for _, o := range orgs {
				if o == orgID {
					return true
				}
			}
		}
	}
	return false
}

// IsAuthenticated 检查当前请求是否已认证
func IsAuthenticated(c *gin.Context) bool {
	return GetUserID(c) != ""
}

// GetAccessToken 从请求中提取 access token（优先级：Authorization Header > Cookie）。
// 用于 Logout 等需要获取当前 token 原始值的场景。
func GetAccessToken(c *gin.Context) string {
	token, _ := getTokenWithSource(c)
	return token
}

// getContextString 从上下文获取字符串值
func getContextString(c *gin.Context, key string) string {
	if val, exists := c.Get(key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getTokenWithSource 从请求中获取 access token 及其来源。
// 优先级：Authorization Header > Cookie。
// 当客户端显式携带 Bearer token 时，应优先按 API 客户端语义处理，
// 避免浏览器 cookie 覆盖 bearerAuth 契约。
func getTokenWithSource(c *gin.Context) (string, tokenSource) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			accessToken := strings.TrimSpace(parts[1])
			if accessToken != "" {
				return accessToken, tokenSourceBearer
			}
		}
	}

	if accessToken, err := c.Cookie(CookieAccessToken); err == nil && accessToken != "" {
		return accessToken, tokenSourceCookie
	}

	return "", tokenSourceNone
}
