package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

const refreshReservationReleaseTimeout = 2 * time.Second

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

	// 从 native header 或浏览器 session cookie 中获取唯一 session ID
	sessionID, ok := h.resolveSessionID(c)
	if !ok {
		return
	}
	if !h.requireTrackedNativeLogoutSession(c, accessToken, refreshToken, sessionID) {
		return
	}

	if err := h.svc.RevokeSession(c.Request.Context(), sessionID, userID, accessToken, refreshToken); err != nil {
		if errors.Is(err, token.ErrSessionNotFound) ||
			errors.Is(err, errSessionUserRequired) ||
			errors.Is(err, errSessionUserMismatch) ||
			errors.Is(err, errSessionAccessTokenMismatch) ||
			errors.Is(err, errSessionRefreshTokenMismatch) {
			h.clearTokenCookies(c)
			response.Unauthorized(c, "invalid session", errs.ErrTokenInvalid)
			return
		}
		logger.FromGin(c).Error("failed to revoke session",
			zap.String("user_id", userID),
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		// 仍然清除客户端 cookie（减少攻击面），但不向客户端承诺撤销成功
		h.clearTokenCookies(c)
		audit.LogFailureContext(c.Request.Context(), audit.EventUserLogout, c.ClientIP(), c.Request.UserAgent(), requestID, "server-side revocation failed")
		response.InternalError(c, "logout partially failed: server-side revocation unsuccessful")
		return
	}

	h.clearTokenCookies(c)
	audit.LogSuccessContext(c.Request.Context(), audit.EventUserLogout, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{"message": "logout successful"})
}

// LogoutAll 全设备登出
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)

	if err := h.svc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		if errors.Is(err, errSessionUserRequired) {
			response.Unauthorized(c, "missing authentication token", errs.ErrTokenMissing)
			return
		}
		logger.FromGin(c).Error("failed to revoke all sessions",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to logout from all devices")
		return
	}

	h.clearTokenCookies(c)
	audit.LogSuccessContext(c.Request.Context(), audit.EventUserLogoutAll, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)
	response.Success(c, gin.H{"message": "logged out from all devices"})
}

// RefreshToken 使用 refresh token 获取新的 token 对。
// 支持两种来源：
//  1. 请求体 JSON {"refreshToken": "..."}（原生 App）
//  2. Cookie（Web 浏览器）
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshTokenStr, fromBody, ok := resolveRefreshToken(c)
	if !ok {
		return
	}
	if refreshTokenStr == "" {
		response.Unauthorized(c, "missing refresh token", errs.ErrTokenMissing)
		return
	}
	if token.IsSelfSignedToken(refreshTokenStr) {
		h.clearTokenCookies(c)
		response.Unauthorized(c, "unsupported refresh token", errs.ErrRefreshTokenInvalid)
		return
	}
	if _, ok := h.resolveSessionID(c); !ok {
		return
	}

	if !h.requireTrackedNativeRefreshSession(c, fromBody, refreshTokenStr) {
		return
	}

	if !h.validateCookieRefreshCSRF(c, fromBody) {
		return
	}

	// 检查 refresh token 是否已被吊销或已被并发请求消费
	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), refreshTokenStr)
	if err != nil {
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if blacklisted {
		h.rejectRefreshReuse(c, refreshTokenStr)
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

	// OIDC provider refresh token
	success = h.refreshOIDCToken(c, refreshTokenStr, fromBody)
}

func (h *Handler) validateCookieRefreshCSRF(c *gin.Context, fromBody bool) bool {
	if fromBody {
		return true
	}

	cookieCSRF, err := c.Cookie(middleware.CSRFCookieName)
	if err != nil || cookieCSRF == "" {
		response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenMissing, "csrf token missing for cookie-based refresh")
		return false
	}
	headerCSRF := c.GetHeader(middleware.CSRFHeaderName)
	if headerCSRF == "" || subtle.ConstantTimeCompare([]byte(headerCSRF), []byte(cookieCSRF)) != 1 {
		response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenInvalid, "csrf token invalid for cookie-based refresh")
		return false
	}
	return true
}

func (h *Handler) consumeRefreshToken(c *gin.Context, refreshToken string) (func(), bool) {
	consumed, err := h.tokenService.GetBlacklist().TryConsumeRefreshToken(c.Request.Context(), refreshToken, h.tokenService.GetRefreshTokenTTL())
	if err != nil {
		logger.FromGin(c).Warn("failed to reserve refresh token for rotation", zap.Error(err))
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return nil, false
	}
	if !consumed {
		h.rejectRefreshReuse(c, refreshToken)
		return nil, false
	}

	return func() {
		releaseCtx, cancel := ctxutil.DetachedTimeout(c.Request.Context(), refreshReservationReleaseTimeout)
		defer cancel()
		if err := h.tokenService.GetBlacklist().ReleaseConsumedRefreshToken(releaseCtx, refreshToken); err != nil {
			logger.FromGin(c).Warn("failed to release refresh token reservation", zap.Error(err))
		}
	}, true
}

// refreshTokenRequest 原生 App 通过请求体传递 refresh token
type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// buildRefreshResponse 构建 refresh 响应。
// 原生 App（通过请求体提交 refresh token）需要在响应中获取新 token；
// Web 浏览器通过 cookie 自动获取，响应只需 message + expiresIn。
func (h *Handler) buildRefreshResponse(accessToken, refreshToken string, includeTokens bool) gin.H {
	resp := gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.currentAccessTokenTTLSeconds(),
	}
	if includeTokens {
		resp["accessToken"] = accessToken
		resp["refreshToken"] = refreshToken
	}
	return resp
}

// resolveRefreshToken 从唯一凭据来源获取 refresh token：
//   - 请求体 JSON {"refreshToken": "..."}（原生 App）
//   - Cookie（Web 浏览器）
//
// 两种来源不能混用；请求体存在时必须是合法 JSON，避免 malformed body 静默回退到 cookie。
// 返回 token 字符串、是否来自请求体（原生 App 标识）、请求是否可继续处理。
func resolveRefreshToken(c *gin.Context) (string, bool, bool) {
	if requestHasBody(c) {
		if requestHasAuthSessionCookie(c) {
			response.BadRequest(c, "refresh token source is ambiguous", errs.ErrInvalidParam)
			return "", false, false
		}
		var body refreshTokenRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			response.BadRequest(c, "invalid refresh request body", errs.ErrInvalidParam)
			return "", false, false
		}
		return body.RefreshToken, true, true
	}

	if v, err := c.Cookie(middleware.CookieRefreshToken); err == nil && v != "" {
		return v, false, true
	}

	return "", false, true
}

func requestHasBody(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return false
	}
	return c.Request.ContentLength != 0
}

func requestHasAuthSessionCookie(c *gin.Context) bool {
	for _, name := range []string{
		middleware.CookieAccessToken,
		middleware.CookieRefreshToken,
		middleware.CSRFCookieName,
		sessionCookieName,
	} {
		if value, err := c.Cookie(name); err == nil && value != "" {
			return true
		}
	}
	return false
}
