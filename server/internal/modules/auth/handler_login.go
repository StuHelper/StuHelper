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
	Application  string `json:"application"`
	// Native 标记该 state 来自原生 App（deep link 回调流程）
	Native bool `json:"native,omitempty"`
}

// GetLoginURL 生成 OIDC 授权 URL
// 支持 ?redirect=<前端路径> 参数，登录成功后重定向回去
func (h *Handler) GetLoginURL(c *gin.Context) {
	h.respondWithAuthURL(c)
}

// GetSignupURL 生成 OIDC 注册 URL。
// 当前 Casdoor Login UI 内置注册入口，因此这里与登录 URL 复用同一授权地址。
func (h *Handler) GetSignupURL(c *gin.Context) {
	h.respondWithAuthURL(c)
}

func (h *Handler) respondWithAuthURL(c *gin.Context) {
	h.respondWithAuthURLProvider(c, h.oidcClient.GetAuthURLForApplication)
}

// HandleCallback 处理 OIDC 授权回调
// OIDC 登录后浏览器 302 到这里，Go 处理完后重定向回前端
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
	redirect, codeVerifier, appKey, isNative, err := h.consumeOIDCState(c, state)
	if err != nil {
		logger.FromGin(c).Warn("OIDC state verification failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid state")
		response.BadRequest(c, "invalid or expired state parameter")
		return
	}

	// 原生 App 流程：将 code + state 通过 deep link 回传给 App，
	// App 再调用 /exchange-native 完成 token 交换。
	// code_verifier 留在 Redis（exchange-native 消费），避免暴露给客户端。
	if isNative {
		h.handleNativeCallbackRedirect(c, code, state)
		return
	}

	h.handleWebCallback(c, ctx, webCallbackInput{
		code:         code,
		redirect:     redirect,
		codeVerifier: codeVerifier,
		application:  appKey,
		requestID:    requestID,
	})
}

// handleNativeCallbackRedirect 将授权码通过 deep link 回传给原生 App
func (h *Handler) handleNativeCallbackRedirect(c *gin.Context, code, state string) {
	deepLink := fmt.Sprintf("stuhelper://auth/callback?code=%s&state=%s",
		url.QueryEscape(code),
		url.QueryEscape(state),
	)
	c.Redirect(http.StatusFound, deepLink)
}

// handleWebCallback 处理 Web/H5 的标准 OIDC 回调流程
type webCallbackInput struct {
	code         string
	redirect     string
	codeVerifier string
	application  string
	requestID    string
}

func (h *Handler) handleWebCallback(c *gin.Context, ctx context.Context, input webCallbackInput) {

	// 用授权码 + PKCE code_verifier 交换 Token
	oauthToken, err := h.oidcClient.ExchangeCodeForApplication(ctx, input.application, input.code, input.codeVerifier)
	if err != nil {
		logger.FromGin(c).Error("OIDC code exchange failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "code exchange error")
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return
	}

	// 提取并验证 ID Token
	rawIDToken := oidc.ExtractIDToken(oauthToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("no id_token in OAuth response")
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "missing id_token")
		response.InternalError(c, "authentication failed")
		return
	}

	claims, err := h.oidcClient.VerifyIDTokenForApplication(ctx, input.application, rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("ID token verification failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "id_token verification error")
		response.InternalError(c, "authentication failed")
		return
	}

	// 同步本地 shadow user；登录成功必须意味着内部主体已经就绪。
	if syncErr := h.svc.SyncOIDCUser(ctx, UserSyncInput{
		CasdoorSubject: claims.GetUserID(),
		Username:       claims.GetUsername(),
		Email:          claims.GetEmail(),
		AvatarURL:      claims.GetAvatar(),
	}); syncErr != nil {
		logger.FromGin(c).Error("user sync failed",
			zap.String("user_id", claims.GetUserID()),
			zap.Error(syncErr),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "user sync failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 创建服务端 Session（Token Family），跟踪 token 对
	// 必须在写入 Cookie 之前完成：未跟踪的 token 无法被 LogoutAll 撤销
	sessionID, sidErr := token.GenerateSessionID()
	if sidErr != nil {
		logger.FromGin(c).Error("failed to generate session id", zap.Error(sidErr))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "session id generation failed")
		response.InternalError(c, "authentication failed")
		return
	}
	deviceInfo := c.Request.UserAgent()
	sessInfo, sessErr := h.svc.CreateSessionForApplication(
		ctx, sessionID, claims.GetUserID(), rawIDToken, oauthToken.RefreshToken, "oidc", input.application, deviceInfo,
	)
	if sessErr != nil {
		logger.FromGin(c).Error("failed to create session",
			zap.String("user_id", claims.GetUserID()),
			zap.Error(sessErr),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "session creation failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 将 ID Token 作为 access_token 写入 Cookie
	if err := h.setTokenCookies(c, rawIDToken, oauthToken.RefreshToken); err != nil {
		response.InternalError(c, "authentication failed")
		return
	}
	// OIDC ID Token 无法携带自定义 sid claim，通过独立 cookie 传递 session ID
	h.setSessionCookie(c, sessInfo.SessionID)

	audit.LogSuccess(audit.EventUserLogin, claims.GetUserID(), claims.GetUsername(), c.ClientIP(), c.Request.UserAgent(), input.requestID)

	// 302 回前端
	redirectTarget := h.resolveRedirectTarget(input.redirect)
	c.Redirect(http.StatusFound, redirectTarget)
}

type oidcStateInput struct {
	state        string
	redirect     string
	codeVerifier string
	application  string
	native       bool
}

func (h *Handler) storeOIDCState(ctx context.Context, input oidcStateInput) error {
	if input.state == "" {
		return fmt.Errorf("empty state")
	}
	payload := oidcStatePayload{
		RedirectURL:  input.redirect,
		CodeVerifier: input.codeVerifier,
		Application:  input.application,
		Native:       input.native,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal state payload: %w", err)
	}
	return h.redisClient.Set(ctx, oidcStateRedisPrefix+input.state, data, stateMaxAge).Err()
}

func (h *Handler) consumeOIDCState(c *gin.Context, state string) (string, string, string, bool, error) {
	if state == "" {
		h.clearOIDCStateCookie(c)
		return "", "", "", false, fmt.Errorf("empty state")
	}

	raw, err := h.redisClient.Get(c.Request.Context(), oidcStateRedisPrefix+state).Result()
	if err != nil {
		h.clearOIDCStateCookie(c)
		if errors.Is(err, redis.Nil) {
			return "", "", "", false, fmt.Errorf("state expired or already used")
		}
		return "", "", "", false, err
	}

	redirectURL, codeVerifier, appKey, isNative, err := decodeOIDCStatePayload(raw)
	if err != nil {
		h.clearOIDCStateCookie(c)
		return "", "", "", false, err
	}

	if isNative {
		if err := h.deleteOIDCState(c.Request.Context(), state); err != nil {
			return "", "", "", true, err
		}
		payload := nativeCodeVerifierPayload{CodeVerifier: codeVerifier, Application: appKey}
		if err := h.storeNativeCodeVerifier(c.Request.Context(), state, payload); err != nil {
			return "", "", "", true, fmt.Errorf("failed to persist code_verifier for native exchange: %w", err)
		}
		return redirectURL, codeVerifier, appKey, true, nil
	}

	if err := h.validateOIDCStateCookie(c, state); err != nil {
		h.clearOIDCStateCookie(c)
		return "", "", "", false, err
	}
	if err := h.deleteOIDCState(c.Request.Context(), state); err != nil {
		h.clearOIDCStateCookie(c)
		return "", "", "", false, err
	}

	h.clearOIDCStateCookie(c)
	return redirectURL, codeVerifier, appKey, false, nil
}

const nativeCodeVerifierPrefix = "auth:native:verifier:"

type nativeCodeVerifierPayload struct {
	CodeVerifier string `json:"codeVerifier"`
	Application  string `json:"application"`
}

// storeNativeCodeVerifier 将 code_verifier 暂存到 Redis，供 ExchangeNative 端点消费
func (h *Handler) storeNativeCodeVerifier(ctx context.Context, state string, payload nativeCodeVerifierPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal native code_verifier payload: %w", err)
	}
	return h.redisClient.Set(ctx, nativeCodeVerifierPrefix+state, data, stateMaxAge).Err()
}

// consumeNativeCodeVerifier 从 Redis 取出并删除 code_verifier（一次性消费）
func (h *Handler) consumeNativeCodeVerifier(ctx context.Context, state string) (string, string, error) {
	raw, err := h.redisClient.GetDel(ctx, nativeCodeVerifierPrefix+state).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", "", fmt.Errorf("native code_verifier expired or already used")
		}
		return "", "", err
	}
	var payload nativeCodeVerifierPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", "", fmt.Errorf("invalid native code_verifier payload: %w", err)
	}
	return payload.CodeVerifier, payload.Application, nil
}

func decodeOIDCStatePayload(raw string) (string, string, string, bool, error) {
	var payload oidcStatePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw, "", oidc.ApplicationWeb, false, nil
	}
	appKey := payload.Application
	if appKey == "" {
		appKey = oidc.ApplicationWeb
	}
	return payload.RedirectURL, payload.CodeVerifier, appKey, payload.Native, nil
}

func (h *Handler) validateOIDCStateCookie(c *gin.Context, state string) error {
	cookieState, err := c.Cookie(oidcStateCookieName)
	if err != nil || cookieState == "" {
		return fmt.Errorf("state cookie missing")
	}
	if subtle.ConstantTimeCompare([]byte(cookieState), []byte(state)) != 1 {
		return fmt.Errorf("state cookie mismatch")
	}
	return nil
}

func (h *Handler) deleteOIDCState(ctx context.Context, state string) error {
	deleted, err := h.redisClient.Del(ctx, oidcStateRedisPrefix+state).Result()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("state expired or already used")
	}
	return nil
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

// exchangeNativeRequest 原生 App 令牌交换请求体
type exchangeNativeRequest struct {
	Code  string `json:"code"  binding:"required"`
	State string `json:"state" binding:"required"`
}

// ExchangeNative 原生 App 令牌交换端点
// 原生 App 通过 deep link 收到 code + state 后，调用此端点用授权码换取 token。
// code_verifier 存储在服务端 Redis，客户端无需知晓，安全性更高。
func (h *Handler) ExchangeNative(c *gin.Context) {
	var req exchangeNativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	// 消费一次性 code_verifier
	codeVerifier, appKey, err := h.consumeNativeCodeVerifier(ctx, req.State)
	if err != nil {
		logger.FromGin(c).Warn("native exchange: code_verifier lookup failed",
			zap.String("state", req.State), zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid or expired native state")
		response.BadRequest(c, "invalid or expired state parameter")
		return
	}

	// 用授权码 + PKCE code_verifier 交换 Token
	oauthToken, err := h.oidcClient.ExchangeCodeForApplication(ctx, appKey, req.Code, codeVerifier)
	if err != nil {
		logger.FromGin(c).Error("native exchange: OIDC code exchange failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "code exchange error")
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return
	}

	// 提取并验证 ID Token
	rawIDToken := oidc.ExtractIDToken(oauthToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("native exchange: no id_token in OAuth response")
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "missing id_token")
		response.InternalError(c, "authentication failed")
		return
	}

	claims, err := h.oidcClient.VerifyIDTokenForApplication(ctx, appKey, rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("native exchange: ID token verification failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "id_token verification error")
		response.InternalError(c, "authentication failed")
		return
	}

	// 同步本地 shadow user
	if syncErr := h.svc.SyncOIDCUser(ctx, UserSyncInput{
		CasdoorSubject: claims.GetUserID(),
		Username:       claims.GetUsername(),
		Email:          claims.GetEmail(),
		AvatarURL:      claims.GetAvatar(),
	}); syncErr != nil {
		logger.FromGin(c).Error("native exchange: user sync failed",
			zap.String("user_id", claims.GetUserID()), zap.Error(syncErr))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "user sync failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 创建服务端 Session（Token Family）。原生客户端没有 cookie，
	// 下面把 sessionID 写入响应体，客户端须在 refresh / logout 时通过
	// X-Stuhelper-Session-ID header 回传，以保留 Token Family 追踪语义。
	nativeSessionID, nativeSidErr := token.GenerateSessionID()
	if nativeSidErr != nil {
		logger.FromGin(c).Error("native exchange: failed to generate session id", zap.Error(nativeSidErr))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "session id generation failed")
		response.InternalError(c, "authentication failed")
		return
	}
	deviceInfo := c.Request.UserAgent()
	if _, sessErr := h.svc.CreateSessionForApplication(
		ctx, nativeSessionID, claims.GetUserID(), rawIDToken, oauthToken.RefreshToken, "oidc-native", appKey, deviceInfo,
	); sessErr != nil {
		logger.FromGin(c).Error("native exchange: failed to create session",
			zap.String("user_id", claims.GetUserID()), zap.Error(sessErr))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "session creation failed")
		response.InternalError(c, "authentication failed")
		return
	}

	audit.LogSuccess(audit.EventUserLogin, claims.GetUserID(), claims.GetUsername(), c.ClientIP(), c.Request.UserAgent(), requestID)

	// 返回 token 给原生 App（JSON body 而非 cookie）
	response.Success(c, gin.H{
		"accessToken":  rawIDToken,
		"refreshToken": oauthToken.RefreshToken,
		"sessionID":    nativeSessionID,
		"expiresIn":    h.currentAccessTokenTTLSeconds(),
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
