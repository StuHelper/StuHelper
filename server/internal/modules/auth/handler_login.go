package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

const (
	// stateMaxAge OIDC state 最大有效期
	stateMaxAge = 10 * time.Minute

	oidcStateCookieName  = "oidc_state"
	oidcStateCookiePath  = "/api/v1/auth/callback"
	oidcStateRedisPrefix = "auth:oidc:state:"
)

// oidcStatePayload 存储在 Redis 中的 OIDC state 关联数据
type oidcStatePayload struct {
	RedirectURL  string `json:"redirectURL"`
	CodeVerifier string `json:"codeVerifier"`
}

// GetLoginURL 生成 OIDC 授权 URL
// 支持 ?redirect=<前端路径> 参数，登录成功后重定向回去
func (h *Handler) GetLoginURL(c *gin.Context) {
	h.respondWithAuthURL(c)
}

// GetSignupURL 生成 OIDC 注册 URL。
// 当前 Zitadel Login UI 内置注册入口，因此这里与登录 URL 复用同一授权地址。
func (h *Handler) GetSignupURL(c *gin.Context) {
	h.respondWithAuthURL(c)
}

func (h *Handler) respondWithAuthURL(c *gin.Context) {
	redirect := strings.TrimSpace(c.Query("redirect"))

	state, err := generateNonce()
	if err != nil {
		logger.FromGin(c).Error("failed to generate state", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}

	authURL, verifier := h.oidcClient.GetAuthURL(state)

	if err := h.storeOIDCState(c.Request.Context(), state, redirect, verifier); err != nil {
		logger.FromGin(c).Error("failed to persist oidc state", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}

	h.setOIDCStateCookie(c, state)
	response.Success(c, gin.H{"url": authURL, "state": state})
}

// HandleCallback 处理 OIDC 授权回调
// Zitadel 登录后浏览器 302 到这里，Go 处理完后重定向回前端
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	if code == "" {
		response.BadRequest(c, "missing authorization code")
		return
	}

	// 服务端一次性 state 校验（Redis + HttpOnly cookie 绑定浏览器）
	redirect, codeVerifier, err := h.consumeOIDCState(c, state)
	if err != nil {
		logger.FromGin(c).Warn("OIDC state verification failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid state")
		response.BadRequest(c, "invalid or expired state parameter")
		return
	}

	// 用授权码 + PKCE code_verifier 交换 Token
	oauthToken, err := h.oidcClient.ExchangeCode(ctx, code, codeVerifier)
	if err != nil {
		logger.FromGin(c).Error("OIDC code exchange failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "code exchange error")
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return
	}

	// 提取并验证 ID Token
	rawIDToken := oidc.ExtractIDToken(oauthToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("no id_token in OAuth response")
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "missing id_token")
		response.InternalError(c, "authentication failed")
		return
	}

	claims, err := h.oidcClient.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("ID token verification failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "id_token verification error")
		response.InternalError(c, "authentication failed")
		return
	}

	// 同步本地 shadow user；登录成功必须意味着内部主体已经就绪。
	if h.userSyncRepo == nil {
		logger.FromGin(c).Error("user sync repository is not configured")
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "user sync repository not configured")
		response.InternalError(c, "authentication failed")
		return
	}
	if syncErr := h.userSyncRepo.UpsertUser(ctx, UserSyncInput{
		ExternalID: claims.GetUserID(),
		Username:   claims.GetUsername(),
		Email:      claims.GetEmail(),
		AvatarURL:  claims.GetAvatar(),
	}); syncErr != nil {
		logger.FromGin(c).Error("user sync failed",
			zap.String("user_id", claims.GetUserID()),
			zap.Error(syncErr),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "user sync failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 将 ID Token 作为 access_token 写入 Cookie
	if err := h.setTokenCookies(c, rawIDToken, oauthToken.RefreshToken); err != nil {
		response.InternalError(c, "authentication failed")
		return
	}

	// 跟踪用户 Token，支持 LogoutAll 批量撤销
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
		ctx, claims.GetUserID(), rawIDToken, token.TokenTypeAccess, time.Now().Add(h.tokenService.GetAccessTokenTTL()),
	); trackErr != nil {
		logger.FromGin(c).Warn("failed to track user token",
			zap.String("user_id", claims.GetUserID()),
			zap.Error(trackErr),
		)
	}
	if oauthToken.RefreshToken != "" {
		if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
			ctx, claims.GetUserID(), oauthToken.RefreshToken, token.TokenTypeRefresh, time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
		); trackErr != nil {
			logger.FromGin(c).Warn("failed to track refresh token",
				zap.String("user_id", claims.GetUserID()),
				zap.Error(trackErr),
			)
		}
	}

	audit.LogSuccess(audit.EventUserLogin, claims.GetUserID(), claims.GetUsername(), c.ClientIP(), c.Request.UserAgent(), requestID)

	// 302 回前端
	redirectTarget := h.resolveRedirectTarget(redirect)
	c.Redirect(http.StatusFound, redirectTarget)
}

func (h *Handler) storeOIDCState(ctx context.Context, state, redirect, codeVerifier string) error {
	if state == "" {
		return fmt.Errorf("empty state")
	}
	if h.redisClient == nil {
		return fmt.Errorf("redis client not configured")
	}
	payload := oidcStatePayload{
		RedirectURL:  redirect,
		CodeVerifier: codeVerifier,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal state payload: %w", err)
	}
	return h.redisClient.Set(ctx, oidcStateRedisPrefix+state, data, stateMaxAge).Err()
}

func (h *Handler) consumeOIDCState(c *gin.Context, state string) (string, string, error) {
	if state == "" {
		h.clearOIDCStateCookie(c)
		return "", "", fmt.Errorf("empty state")
	}
	if h.redisClient == nil {
		h.clearOIDCStateCookie(c)
		return "", "", fmt.Errorf("redis client not configured")
	}

	cookieState, err := c.Cookie(oidcStateCookieName)
	if err != nil || cookieState == "" {
		h.clearOIDCStateCookie(c)
		return "", "", fmt.Errorf("state cookie missing")
	}
	if subtle.ConstantTimeCompare([]byte(cookieState), []byte(state)) != 1 {
		h.clearOIDCStateCookie(c)
		return "", "", fmt.Errorf("state cookie mismatch")
	}

	raw, err := h.redisClient.GetDel(c.Request.Context(), oidcStateRedisPrefix+state).Result()
	if err != nil {
		h.clearOIDCStateCookie(c)
		if errors.Is(err, redis.Nil) {
			return "", "", fmt.Errorf("state expired or already used")
		}
		return "", "", err
	}

	h.clearOIDCStateCookie(c)

	var payload oidcStatePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		// 兼容：旧格式纯 redirect 字符串
		return raw, "", nil
	}
	return payload.RedirectURL, payload.CodeVerifier, nil
}

func (h *Handler) setOIDCStateCookie(c *gin.Context, state string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
		MaxAge:   int(stateMaxAge.Seconds()),
		Path:     oidcStateCookiePath,
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearOIDCStateCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     oidcStateCookiePath,
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// resolveRedirectTarget 根据验证后的 redirect 参数决定最终跳转地址
func (h *Handler) resolveRedirectTarget(redirect string) string {
	defaultRedirect := h.defaultRedirectURL

	if redirect == "" {
		return defaultRedirect
	}

	// 相对路径：解析到默认前端地址，拒绝 scheme-relative 输入
	if strings.HasPrefix(redirect, "/") {
		if strings.HasPrefix(redirect, "//") {
			return defaultRedirect
		}
		base, err := url.Parse(defaultRedirect)
		if err != nil {
			return defaultRedirect
		}
		ref, err := url.Parse(redirect)
		if err != nil {
			return defaultRedirect
		}
		return base.ResolveReference(ref).String()
	}

	// 绝对 URL：验证协议和 host 白名单
	parsed, err := url.Parse(redirect)
	if err != nil {
		return defaultRedirect
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return defaultRedirect
	}
	if _, ok := h.allowedRedirectHosts[parsed.Host]; !ok {
		return defaultRedirect
	}

	return parsed.String()
}

// generateNonce 生成 16 字节随机 nonce
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
