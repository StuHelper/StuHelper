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
	RedirectURL         string `json:"redirectURL"`
	CodeVerifier        string `json:"codeVerifier"`
	Application         string `json:"application"`
	CallbackRedirectURI string `json:"callbackRedirectURI,omitempty"`
	// Native 标记该 state 来自原生 App（deep link 回调流程）
	Native bool `json:"native,omitempty"`
}

// GetLoginURL 生成 OIDC 授权 URL
// 支持 ?redirect=<前端路径> 参数，登录成功后重定向回去
func (h *Handler) GetLoginURL(c *gin.Context) {
	if rejectRepeatedAuthQueryParameters(c, "prompt", "max_age") {
		return
	}
	if forceReauthRequested(c) {
		h.respondWithAuthURLProvider(c, h.oidcClient.GetReauthURLForApplicationWithRedirectURI)
		return
	}
	h.respondWithAuthURL(c)
}

func forceReauthRequested(c *gin.Context) bool {
	return strings.TrimSpace(c.Query("prompt")) == "login" || strings.TrimSpace(c.Query("max_age")) == "0"
}

// GetSignupURL 生成 OIDC 注册 URL。
func (h *Handler) GetSignupURL(c *gin.Context) {
	h.respondWithAuthURLProvider(c, h.oidcClient.GetSignupURLForApplicationWithRedirectURI)
}

func (h *Handler) respondWithAuthURL(c *gin.Context) {
	h.respondWithAuthURLProvider(c, h.oidcClient.GetAuthURLForApplicationWithRedirectURI)
}

// HandleCallback 处理 OIDC 授权回调
// OIDC 登录后浏览器 302 到这里，Go 处理完后重定向回前端
func (h *Handler) HandleCallback(c *gin.Context) {
	if rejectRepeatedAuthQueryParameters(c, "code", "state") {
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	if code == "" {
		response.BadRequest(c, "missing authorization code")
		return
	}

	// 服务端一次性 state 校验（Redis + HttpOnly cookie 绑定浏览器）
	stateResult, err := h.consumeOIDCState(c, state)
	if err != nil {
		logger.FromGin(c).Warn("OIDC state verification failed", zap.Error(err))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid state")
		response.BadRequest(c, "invalid or expired state parameter")
		return
	}

	// 原生 App 流程：将 code + state 通过 deep link 回传给 App，
	// App 再调用 /exchange-native 完成 token 交换。
	// code_verifier 留在 Redis（exchange-native 消费），避免暴露给客户端。
	if stateResult.native {
		h.handleNativeCallbackRedirect(c, code, state)
		return
	}

	h.handleWebCallback(c, ctx, webCallbackInput{
		code:         code,
		redirect:     stateResult.redirectURL,
		codeVerifier: stateResult.codeVerifier,
		application:  stateResult.appKey,
		callbackURI:  stateResult.callbackRedirectURI,
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
	callbackURI  string
	requestID    string
}

func (h *Handler) handleWebCallback(c *gin.Context, ctx context.Context, input webCallbackInput) {

	// 用授权码 + PKCE code_verifier 交换 Token
	oauthToken, err := h.oidcClient.ExchangeCodeForApplicationWithRedirectURI(
		ctx,
		input.application,
		input.code,
		input.codeVerifier,
		input.callbackURI,
	)
	if err != nil {
		logger.FromGin(c).Error("OIDC code exchange failed", zap.Error(err))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "code exchange error")
		if errors.Is(err, oidc.ErrProviderUnavailable) {
			response.ServiceUnavailable(c, "authentication service temporarily unavailable")
			return
		}
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return
	}

	// 提取并验证 ID Token
	rawIDToken := oidc.ExtractIDToken(oauthToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("no id_token in OAuth response")
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "missing id_token")
		response.InternalError(c, "authentication failed")
		return
	}

	claims, err := h.oidcClient.VerifyIDTokenForApplication(ctx, input.application, rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("ID token verification failed", zap.Error(err))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "id_token verification error")
		response.InternalError(c, "authentication failed")
		return
	}
	if !h.validateOIDCSubjectForLogin(c, ctx, claims, input.requestID, "oidc subject organization mismatch") {
		return
	}

	// 同步本地 shadow user；登录成功必须意味着内部主体已经就绪。
	if syncErr := h.svc.SyncOIDCUser(ctx, UserSyncInput{
		CasdoorSubject: claims.GetUserID(),
		Username:       claims.GetUsername(),
		Email:          claims.GetEmail(),
		AvatarURL:      claims.GetAvatar(),
		Roles:          claims.Roles,
	}); syncErr != nil {
		logger.FromGin(c).Error("user sync failed",
			zap.String("user_id", claims.GetUserID()),
			zap.Error(syncErr),
		)
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "user sync failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 创建服务端 Session（Token Family），跟踪 token 对
	// 必须在写入 Cookie 之前完成：未跟踪的 token 无法被 LogoutAll 撤销
	sessionID, sidErr := token.GenerateSessionID()
	if sidErr != nil {
		logger.FromGin(c).Error("failed to generate session id", zap.Error(sidErr))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "session id generation failed")
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
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), input.requestID, "session creation failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 将 ID Token 作为 access_token 写入 Cookie
	if err := h.setTokenCookies(c, rawIDToken, oauthToken.RefreshToken, sessInfo.SessionID); err != nil {
		response.InternalError(c, "authentication failed")
		return
	}
	// OIDC ID Token 无法携带自定义 sid claim，通过独立 cookie 传递 session ID
	h.setSessionCookie(c, sessInfo.SessionID)

	audit.LogSuccessContext(ctx, audit.EventUserLogin, claims.GetUserID(), claims.GetUsername(), c.ClientIP(), c.Request.UserAgent(), input.requestID)

	// 302 回前端
	redirectTarget := h.resolveRedirectTarget(input.redirect)
	c.Redirect(http.StatusFound, redirectTarget)
}

type oidcStateInput struct {
	state        string
	redirect     string
	codeVerifier string
	application  string
	callbackURI  string
	native       bool
}

type oidcStateConsumeResult struct {
	redirectURL         string
	codeVerifier        string
	appKey              string
	callbackRedirectURI string
	native              bool
}

func (h *Handler) storeOIDCState(ctx context.Context, input oidcStateInput) error {
	normalized, err := normalizeOIDCStateInput(input)
	if err != nil {
		return err
	}
	payload := oidcStatePayload{
		RedirectURL:         normalized.redirect,
		CodeVerifier:        normalized.codeVerifier,
		Application:         normalized.application,
		CallbackRedirectURI: normalized.callbackURI,
		Native:              normalized.native,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal state payload: %w", err)
	}
	return h.redisClient.Set(ctx, oidcStateRedisPrefix+normalized.state, data, stateMaxAge).Err()
}

func normalizeOIDCStateInput(input oidcStateInput) (oidcStateInput, error) {
	input.state = strings.TrimSpace(input.state)
	if input.state == "" {
		return input, fmt.Errorf("empty state")
	}
	input.redirect = strings.TrimSpace(input.redirect)
	input.codeVerifier = strings.TrimSpace(input.codeVerifier)
	if input.codeVerifier == "" {
		return input, fmt.Errorf("oidc state code_verifier missing")
	}
	callbackURI, err := normalizeOIDCCallbackRedirectURI(input.callbackURI)
	if err != nil {
		return input, err
	}
	input.callbackURI = callbackURI
	appKey, err := normalizeOIDCStateApplication(input.application, input.native)
	if err != nil {
		return input, err
	}
	input.application = appKey
	return input, nil
}

func normalizeRequiredAuthState(state string) (string, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return "", fmt.Errorf("empty state")
	}
	return state, nil
}

func normalizeOIDCCallbackRedirectURI(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid oidc callback redirect uri")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("invalid oidc callback redirect uri")
	}
	host := strings.ToLower(parsed.Host)
	if host == "" || parsed.User != nil || parsed.Path != oidcStateCookiePath || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid oidc callback redirect uri")
	}
	parsed.Scheme = scheme
	parsed.Host = host
	return parsed.String(), nil
}

func (h *Handler) consumeOIDCState(c *gin.Context, state string) (oidcStateConsumeResult, error) {
	var err error
	state, err = normalizeRequiredAuthState(state)
	if err != nil {
		h.clearOIDCStateCookie(c)
		return oidcStateConsumeResult{}, err
	}

	raw, err := h.redisClient.Get(c.Request.Context(), oidcStateRedisPrefix+state).Result()
	if err != nil {
		h.clearOIDCStateCookie(c)
		if errors.Is(err, redis.Nil) {
			return oidcStateConsumeResult{}, fmt.Errorf("state expired or already used")
		}
		return oidcStateConsumeResult{}, err
	}

	result, err := decodeOIDCStatePayload(raw)
	if err != nil {
		h.clearOIDCStateCookie(c)
		return oidcStateConsumeResult{}, err
	}

	if result.native {
		payload := nativeCodeVerifierPayload{CodeVerifier: result.codeVerifier, Application: result.appKey}
		if err := h.promoteOIDCStateToNativeVerifier(c.Request.Context(), state, raw, payload); err != nil {
			return result, fmt.Errorf("failed to persist code_verifier for native exchange: %w", err)
		}
		return result, nil
	}

	if err := h.validateOIDCStateCookie(c, state); err != nil {
		h.clearOIDCStateCookie(c)
		return oidcStateConsumeResult{}, err
	}
	if err := h.deleteOIDCState(c.Request.Context(), state); err != nil {
		h.clearOIDCStateCookie(c)
		return oidcStateConsumeResult{}, err
	}

	h.clearOIDCStateCookie(c)
	return result, nil
}

const nativeCodeVerifierPrefix = "auth:native:verifier:"

const promoteOIDCStateToNativeVerifierScript = `
local current = redis.call("GET", KEYS[1])
if not current then
	return 0
end
if current ~= ARGV[1] then
	return -1
end
redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[3])
redis.call("DEL", KEYS[1])
return 1
`

type nativeCodeVerifierPayload struct {
	CodeVerifier string `json:"codeVerifier"`
	Application  string `json:"application"`
}

// storeNativeCodeVerifier 将 code_verifier 暂存到 Redis，供 ExchangeNative 端点消费
func (h *Handler) storeNativeCodeVerifier(ctx context.Context, state string, payload nativeCodeVerifierPayload) error {
	var err error
	state, err = normalizeRequiredAuthState(state)
	if err != nil {
		return err
	}
	codeVerifier, appKey, err := normalizeNativeCodeVerifierPayload(payload)
	if err != nil {
		return err
	}
	payload = nativeCodeVerifierPayload{
		CodeVerifier: codeVerifier,
		Application:  appKey,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal native code_verifier payload: %w", err)
	}
	return h.redisClient.Set(ctx, nativeCodeVerifierPrefix+state, data, stateMaxAge).Err()
}

func (h *Handler) promoteOIDCStateToNativeVerifier(
	ctx context.Context,
	state string,
	expectedStateRaw string,
	payload nativeCodeVerifierPayload,
) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal native code_verifier payload: %w", err)
	}
	result, err := h.redisClient.Eval(ctx, promoteOIDCStateToNativeVerifierScript, []string{
		oidcStateRedisPrefix + state,
		nativeCodeVerifierPrefix + state,
	}, expectedStateRaw, string(data), int64(stateMaxAge/time.Second)).Int()
	if err != nil {
		return err
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("state expired or already used")
	default:
		return fmt.Errorf("state changed during native verifier promotion")
	}
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
	codeVerifier, appKey, err := normalizeNativeCodeVerifierPayload(payload)
	if err != nil {
		return "", "", err
	}
	return codeVerifier, appKey, nil
}

func decodeOIDCStatePayload(raw string) (oidcStateConsumeResult, error) {
	var payload oidcStatePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return oidcStateConsumeResult{}, fmt.Errorf("invalid oidc state payload: %w", err)
	}
	codeVerifier := strings.TrimSpace(payload.CodeVerifier)
	if codeVerifier == "" {
		return oidcStateConsumeResult{}, fmt.Errorf("oidc state code_verifier missing")
	}
	appKey, err := normalizeOIDCStateApplication(payload.Application, payload.Native)
	if err != nil {
		return oidcStateConsumeResult{}, err
	}
	callbackURI, err := normalizeOIDCCallbackRedirectURI(payload.CallbackRedirectURI)
	if err != nil {
		return oidcStateConsumeResult{}, err
	}
	return oidcStateConsumeResult{
		redirectURL:         strings.TrimSpace(payload.RedirectURL),
		codeVerifier:        codeVerifier,
		appKey:              appKey,
		callbackRedirectURI: callbackURI,
		native:              payload.Native,
	}, nil
}

func normalizeNativeCodeVerifierPayload(payload nativeCodeVerifierPayload) (string, string, error) {
	codeVerifier := strings.TrimSpace(payload.CodeVerifier)
	if codeVerifier == "" {
		return "", "", fmt.Errorf("native code_verifier missing")
	}
	appKey, err := normalizeOIDCStateApplication(payload.Application, true)
	if err != nil {
		return "", "", err
	}
	return codeVerifier, appKey, nil
}

func normalizeOIDCStateApplication(rawApp string, native bool) (string, error) {
	appKey := strings.TrimSpace(rawApp)
	if appKey == "" && native {
		return oidc.ApplicationUniapp, nil
	}
	if appKey == "" {
		return oidc.ApplicationWeb, nil
	}
	if native && appKey != oidc.ApplicationUniapp {
		return "", fmt.Errorf("native oidc state application mismatch")
	}
	switch appKey {
	case oidc.ApplicationWeb, oidc.ApplicationAdmin, oidc.ApplicationUniapp:
		return appKey, nil
	default:
		return "", fmt.Errorf("unknown oidc state application")
	}
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
	http.SetCookie(c.Writer, &http.Cookie{ //nolint:gosec // cookie Secure flag is environment-configured through TokenConfig.
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
	http.SetCookie(c.Writer, &http.Cookie{ //nolint:gosec // cookie Secure flag is environment-configured through TokenConfig.
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
	req.Code = strings.TrimSpace(req.Code)
	req.State = strings.TrimSpace(req.State)
	if req.Code == "" || req.State == "" {
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
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid or expired native state")
		response.BadRequest(c, "invalid or expired state parameter")
		return
	}

	// 用授权码 + PKCE code_verifier 交换 Token
	oauthToken, err := h.oidcClient.ExchangeCodeForApplication(ctx, appKey, req.Code, codeVerifier)
	if err != nil {
		logger.FromGin(c).Error("native exchange: OIDC code exchange failed", zap.Error(err))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "code exchange error")
		if errors.Is(err, oidc.ErrProviderUnavailable) {
			response.ServiceUnavailable(c, "authentication service temporarily unavailable")
			return
		}
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return
	}

	// 提取并验证 ID Token
	rawIDToken := oidc.ExtractIDToken(oauthToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("native exchange: no id_token in OAuth response")
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "missing id_token")
		response.InternalError(c, "authentication failed")
		return
	}

	claims, err := h.oidcClient.VerifyIDTokenForApplication(ctx, appKey, rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("native exchange: ID token verification failed", zap.Error(err))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "id_token verification error")
		response.InternalError(c, "authentication failed")
		return
	}
	if !h.validateOIDCSubjectForLogin(c, ctx, claims, requestID, "native oidc subject organization mismatch") {
		return
	}

	// 同步本地 shadow user
	if syncErr := h.svc.SyncOIDCUser(ctx, UserSyncInput{
		CasdoorSubject: claims.GetUserID(),
		Username:       claims.GetUsername(),
		Email:          claims.GetEmail(),
		AvatarURL:      claims.GetAvatar(),
		Roles:          claims.Roles,
	}); syncErr != nil {
		logger.FromGin(c).Error("native exchange: user sync failed",
			zap.String("user_id", claims.GetUserID()), zap.Error(syncErr))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "user sync failed")
		response.InternalError(c, "authentication failed")
		return
	}

	// 创建服务端 Session（Token Family）。原生客户端没有 cookie，
	// 下面把 sessionID 写入响应体，客户端须在 refresh / logout 时通过
	// X-Stuhelper-Session-ID header 回传，以保留 Token Family 追踪语义。
	nativeSessionID, nativeSidErr := token.GenerateSessionID()
	if nativeSidErr != nil {
		logger.FromGin(c).Error("native exchange: failed to generate session id", zap.Error(nativeSidErr))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "session id generation failed")
		response.InternalError(c, "authentication failed")
		return
	}
	deviceInfo := c.Request.UserAgent()
	if _, sessErr := h.svc.CreateSessionForApplication(
		ctx, nativeSessionID, claims.GetUserID(), rawIDToken, oauthToken.RefreshToken, "oidc-native", appKey, deviceInfo,
	); sessErr != nil {
		logger.FromGin(c).Error("native exchange: failed to create session",
			zap.String("user_id", claims.GetUserID()), zap.Error(sessErr))
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "session creation failed")
		response.InternalError(c, "authentication failed")
		return
	}

	audit.LogSuccessContext(ctx, audit.EventUserLogin, claims.GetUserID(), claims.GetUsername(), c.ClientIP(), c.Request.UserAgent(), requestID)

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
	redirect = strings.TrimSpace(redirect)

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
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return defaultRedirect
	}
	if _, ok := h.allowedRedirectHosts[strings.ToLower(parsed.Host)]; !ok {
		return defaultRedirect
	}

	parsed.Scheme = scheme
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String()
}

func (h *Handler) resolveOIDCCallbackRedirectURI(req authURLRequest) string {
	if req.native || strings.TrimSpace(h.tokenConfig.CookieDomain) != "" {
		return ""
	}
	switch strings.TrimSpace(req.appKey) {
	case oidc.ApplicationWeb, oidc.ApplicationAdmin:
	default:
		return ""
	}
	origin, ok := h.allowedAbsoluteRedirectOrigin(req.redirect)
	if !ok {
		return ""
	}
	return origin + oidcStateCookiePath
}

func (h *Handler) allowedAbsoluteRedirectOrigin(redirect string) (string, bool) {
	redirect = strings.TrimSpace(redirect)
	if redirect == "" || strings.HasPrefix(redirect, "/") {
		return "", false
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	host := strings.ToLower(parsed.Host)
	if host == "" {
		return "", false
	}
	if _, ok := h.allowedRedirectHosts[host]; !ok {
		return "", false
	}
	return scheme + "://" + host, true
}

// generateNonce 生成 16 字节随机 nonce
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
