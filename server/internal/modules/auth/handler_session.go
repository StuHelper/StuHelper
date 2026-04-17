package auth

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Logout 登出当前设备（基于 Session 撤销）
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)

	accessToken := middleware.GetAccessToken(c)

	var refreshToken string
	if v, err := c.Cookie(middleware.CookieRefreshToken); err == nil {
		refreshToken = v
	}

	// 从 access token 或 session cookie 中获取 session ID
	sessionID := h.getSessionID(c, accessToken)

	if err := h.svc.RevokeSession(c.Request.Context(), sessionID, userID, accessToken, refreshToken); err != nil {
		logger.FromGin(c).Error("failed to revoke session",
			zap.String("user_id", userID),
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		// 仍然清除客户端 cookie（减少攻击面），但不向客户端承诺撤销成功
		h.clearTokenCookies(c)
		audit.LogFailure(audit.EventUserLogout, c.ClientIP(), c.Request.UserAgent(), requestID, "server-side revocation failed")
		response.InternalError(c, "logout partially failed: server-side revocation unsuccessful")
		return
	}

	h.clearTokenCookies(c)
	audit.LogSuccess(audit.EventUserLogout, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{"message": "logout successful"})
}

// LogoutAll 全设备登出
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)

	if err := h.svc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		logger.FromGin(c).Error("failed to revoke all sessions",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to logout from all devices")
		return
	}

	h.clearTokenCookies(c)
	audit.LogSuccess(audit.EventUserLogoutAll, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)
	response.Success(c, gin.H{"message": "logged out from all devices"})
}

// RefreshToken 使用 refresh token 获取新的 token 对。
// 支持两种来源：
//  1. 请求体 JSON {"refreshToken": "..."}（原生 App）
//  2. Cookie（Web 浏览器）
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshTokenStr, fromBody := resolveRefreshToken(c)
	if refreshTokenStr == "" {
		response.Unauthorized(c, "missing refresh token", errs.ErrTokenMissing)
		return
	}

	// fromBody 标记请求来自原生 App（无 cookie），refresh 响应需包含 token 值
	c.Set("refresh_from_native", fromBody)

	// 纵深防御：若 refresh token 来自 cookie，必须携带有效 CSRF token。
	// 全局 CSRFMiddleware 对携带 Bearer Authorization 的请求放行，攻击者
	// 同时诱导浏览器发送 Bearer header 和 cookie 时可绕过；refresh 是最敏感的
	// 会话轮换入口，此处显式校验关闭该缺口。原生 App 通过 body 传递 refresh
	// token，无需 cookie，不触发此校验。
	if !fromBody {
		cookieCSRF, err := c.Cookie(middleware.CSRFCookieName)
		if err != nil || cookieCSRF == "" {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenMissing, "csrf token missing for cookie-based refresh")
			return
		}
		headerCSRF := c.GetHeader(middleware.CSRFHeaderName)
		if headerCSRF == "" || subtle.ConstantTimeCompare([]byte(headerCSRF), []byte(cookieCSRF)) != 1 {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenInvalid, "csrf token invalid for cookie-based refresh")
			return
		}
	}

	// 检查 refresh token 是否已被吊销或已被并发请求消费
	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), refreshTokenStr)
	if err != nil {
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if blacklisted {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return
	}

	releaseConsume, ok := h.consumeRefreshToken(c, refreshTokenStr)
	if !ok {
		return
	}
	success := false
	defer func() {
		if success {
			return
		}
		releaseConsume()
	}()

	// 自签名 JWT refresh token（手机验证码登录签发）
	if token.IsSelfSignedToken(refreshTokenStr) {
		success = h.refreshSelfSignedToken(c, refreshTokenStr)
		return
	}

	// Zitadel OIDC refresh token
	success = h.refreshZitadelToken(c, refreshTokenStr)
}

func (h *Handler) consumeRefreshToken(c *gin.Context, refreshToken string) (func(), bool) {
	consumed, err := h.tokenService.GetBlacklist().TryConsumeRefreshToken(c.Request.Context(), refreshToken, h.tokenService.GetRefreshTokenTTL())
	if err != nil {
		logger.FromGin(c).Warn("failed to reserve refresh token for rotation", zap.Error(err))
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return nil, false
	}
	if !consumed {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return nil, false
	}

	return func() {
		if err := h.tokenService.GetBlacklist().ReleaseConsumedRefreshToken(c.Request.Context(), refreshToken); err != nil {
			logger.FromGin(c).Warn("failed to release refresh token reservation", zap.Error(err))
		}
	}, true
}

// refreshSelfSignedToken 处理自签名 JWT 的 refresh 逻辑
func (h *Handler) refreshSelfSignedToken(c *gin.Context, refreshTokenStr string) bool {
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		logger.FromGin(c).Error("HMAC key not initialized for refresh")
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	oldClaims, err := token.VerifyJWTWithType(hmacKey, refreshTokenStr, token.JWTTokenTypeRefresh)
	if err != nil {
		logger.FromGin(c).Debug("self-signed refresh token verification failed", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "invalid refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}

	accessTTL := time.Duration(h.tokenConfig.AccessTokenTTL) * time.Second
	refreshTTL := time.Duration(h.tokenConfig.RefreshTokenTTL) * time.Second

	exists, err := h.svc.UserExistsByExternalID(c.Request.Context(), oldClaims.Sub)
	if err != nil {
		logger.FromGin(c).Error("failed to verify phone-login user during refresh", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}
	if !exists {
		logger.FromGin(c).Warn("phone-login refresh rejected for missing user", zap.String("user_id", oldClaims.Sub))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}

	// Token Family：refresh 在同一 session 内轮换，继承 session ID
	sessionID := oldClaims.Sid

	newAccessClaims := token.JWTClaims{
		Sub:         oldClaims.Sub,
		Name:        oldClaims.Name,
		Email:       oldClaims.Email,
		DisplayName: oldClaims.DisplayName,
		Avatar:      oldClaims.Avatar,
		Roles:       []string{"user"},
		Typ:         token.JWTTokenTypeAccess,
		Sid:         sessionID,
	}

	newAccessToken, err := token.SignJWT(hmacKey, newAccessClaims, accessTTL)
	if err != nil {
		logger.FromGin(c).Error("failed to sign new access JWT", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	newRefreshClaims := token.JWTClaims{
		Sub:   oldClaims.Sub,
		Name:  oldClaims.Name,
		Roles: []string{"user"},
		Typ:   token.JWTTokenTypeRefresh,
		Sid:   sessionID,
	}
	newRefreshToken, err := token.SignJWT(hmacKey, newRefreshClaims, refreshTTL)
	if err != nil {
		logger.FromGin(c).Error("failed to sign new refresh JWT", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	if err := h.setTokenCookies(c, newAccessToken, newRefreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return false
	}

	// 在同一 session 内轮换 token（黑名单旧 + 更新 session + 跟踪新）
	if rotErr := h.svc.RotateSession(c.Request.Context(), sessionID, oldClaims.Sub, refreshTokenStr, newAccessToken, newRefreshToken); rotErr != nil {
		logger.FromGin(c).Error("failed to rotate self-signed session",
			zap.String("session_id", sessionID),
			zap.Error(rotErr),
		)
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	response.Success(c, h.buildRefreshResponse(c, newAccessToken, newRefreshToken))
	return true
}

// refreshZitadelToken 处理 Zitadel OIDC 的 refresh 逻辑
func (h *Handler) refreshZitadelToken(c *gin.Context, refreshTokenStr string) bool {
	// 调 Zitadel token endpoint 刷新
	newToken, err := h.oidcClient.RefreshToken(c.Request.Context(), refreshTokenStr)
	if err != nil {
		logger.FromGin(c).Error("OIDC token refresh failed", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}

	rawIDToken := oidc.ExtractIDToken(newToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("no id_token in refresh response")
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	// 验证新 ID Token 并提取用户信息
	newClaims, err := h.oidcClient.VerifyIDToken(c.Request.Context(), rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("new ID token verification failed after refresh", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	if err := h.setTokenCookies(c, rawIDToken, newToken.RefreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return false
	}

	// OIDC token 没有 sid claim，从 session cookie 获取（原生客户端无 cookie，sid 为空——
	// RotateSession 会记录 warn 并仅做黑名单轮换）
	oidcSessionID := h.getSessionID(c, "")
	if rotErr := h.svc.RotateSession(c.Request.Context(), oidcSessionID, newClaims.GetUserID(), refreshTokenStr, rawIDToken, newToken.RefreshToken); rotErr != nil {
		logger.FromGin(c).Error("failed to rotate OIDC session",
			zap.String("session_id", oidcSessionID),
			zap.Error(rotErr),
		)
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	response.Success(c, h.buildRefreshResponse(c, rawIDToken, newToken.RefreshToken))
	return true
}

// extractSessionID 从 access token 中提取 session ID。
// 仅自签名 JWT（手机登录）携带 sid claim；OIDC ID Token 无此 claim，返回空字符串。
func extractSessionID(accessToken string) string {
	if accessToken == "" {
		return ""
	}
	if !token.IsSelfSignedToken(accessToken) {
		return ""
	}
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		return ""
	}
	claims, err := token.VerifyJWT(hmacKey, accessToken)
	if err != nil {
		return ""
	}
	return claims.Sid
}

// refreshTokenRequest 原生 App 通过请求体传递 refresh token
type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// buildRefreshResponse 构建 refresh 响应。
// 原生 App（通过请求体提交 refresh token）需要在响应中获取新 token；
// Web 浏览器通过 cookie 自动获取，响应只需 message + expiresIn。
func (h *Handler) buildRefreshResponse(c *gin.Context, accessToken, refreshToken string) gin.H {
	resp := gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	}
	if fromNative, _ := c.Get("refresh_from_native"); fromNative == true {
		resp["accessToken"] = accessToken
		resp["refreshToken"] = refreshToken
	}
	return resp
}

// resolveRefreshToken 按优先级从请求中获取 refresh token：
//  1. 请求体 JSON {"refreshToken": "..."}（原生 App）
//  2. Cookie（Web 浏览器）
//
// 返回 token 字符串和是否来自请求体（原生 App 标识）。
func resolveRefreshToken(c *gin.Context) (string, bool) {
	// 1. 请求体（原生 App 无 cookie，通过 JSON body 传递）
	var body refreshTokenRequest
	if err := c.ShouldBindJSON(&body); err == nil && body.RefreshToken != "" {
		return body.RefreshToken, true
	}

	// 2. Cookie（Web 浏览器标准路径）
	if v, err := c.Cookie(middleware.CookieRefreshToken); err == nil && v != "" {
		return v, false
	}

	return "", false
}
