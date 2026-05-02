package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

type selfSignedTokenSignInput struct {
	hmacKey   []byte
	claims    *token.JWTClaims
	tokenType token.JWTTokenType
	sessionID string
}

type selfSignedSessionRotation struct {
	sessionID       string
	userID          string
	oldRefreshToken string
	newAccessToken  string
	newRefreshToken string
}

// refreshSelfSignedToken 处理自签名 JWT 的 refresh 逻辑。
func (h *Handler) refreshSelfSignedToken(c *gin.Context, refreshTokenStr string) bool {
	oldClaims, hmacKey, ok := h.verifySelfSignedRefreshToken(c, refreshTokenStr)
	if !ok {
		return false
	}
	if !h.ensurePhoneLoginUserExists(c, oldClaims.Sub) {
		return false
	}

	sessionID := oldClaims.Sid
	newAccessToken, ok := h.signRotatedSelfSignedToken(c, selfSignedTokenSignInput{
		hmacKey: hmacKey, claims: oldClaims, tokenType: token.JWTTokenTypeAccess, sessionID: sessionID,
	})
	if !ok {
		return false
	}
	newRefreshToken, ok := h.signRotatedSelfSignedToken(c, selfSignedTokenSignInput{
		hmacKey: hmacKey, claims: oldClaims, tokenType: token.JWTTokenTypeRefresh, sessionID: sessionID,
	})
	if !ok {
		return false
	}
	rotation := selfSignedSessionRotation{
		sessionID:       sessionID,
		userID:          oldClaims.Sub,
		oldRefreshToken: refreshTokenStr,
		newAccessToken:  newAccessToken,
		newRefreshToken: newRefreshToken,
	}
	csrfToken, ok := h.prepareTokenCookies(c)
	if !ok {
		return false
	}
	if !h.rotateSelfSignedSession(c, rotation) {
		return false
	}
	h.setTokenCookiesWithCSRF(c, newAccessToken, newRefreshToken, csrfToken)

	response.Success(c, h.buildRefreshResponse(c, newAccessToken, newRefreshToken))
	return true
}

func (h *Handler) verifySelfSignedRefreshToken(c *gin.Context, refreshTokenStr string) (*token.JWTClaims, []byte, bool) {
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		logger.FromGin(c).Error("HMAC key not initialized for refresh")
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return nil, nil, false
	}

	claims, err := token.VerifyJWTWithType(hmacKey, refreshTokenStr, token.JWTTokenTypeRefresh)
	if err != nil {
		logger.FromGin(c).Debug("self-signed refresh token verification failed", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "invalid refresh token", errs.ErrRefreshTokenInvalid)
		return nil, nil, false
	}
	return claims, hmacKey, true
}

func (h *Handler) ensurePhoneLoginUserExists(c *gin.Context, casdoorSubject string) bool {
	exists, err := h.svc.UserExistsByCasdoorSubject(c.Request.Context(), casdoorSubject)
	if err != nil {
		logger.FromGin(c).Error("failed to verify phone-login user during refresh", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}
	if !exists {
		logger.FromGin(c).Warn("phone-login refresh rejected for missing user", zap.String("user_id", casdoorSubject))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}
	return true
}

func (h *Handler) signRotatedSelfSignedToken(c *gin.Context, input selfSignedTokenSignInput) (string, bool) {
	ttl := h.currentAccessTokenTTL()
	if input.tokenType == token.JWTTokenTypeRefresh {
		ttl = time.Duration(h.tokenConfig.RefreshTokenTTL) * time.Second
	}
	claims := token.JWTClaims{
		Sub: input.claims.Sub, Name: input.claims.Name, Email: input.claims.Email,
		DisplayName: input.claims.DisplayName, Avatar: input.claims.Avatar,
		Roles: input.claims.Roles, Typ: input.tokenType, Sid: input.sessionID,
	}
	signed, err := token.SignJWT(input.hmacKey, claims, ttl)
	if err != nil {
		logger.FromGin(c).Error("failed to sign rotated refresh token pair", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return "", false
	}
	return signed, true
}

func (h *Handler) rotateSelfSignedSession(c *gin.Context, rotation selfSignedSessionRotation) bool {
	err := h.svc.RotateSession(
		c.Request.Context(),
		rotation.sessionID,
		rotation.userID,
		rotation.oldRefreshToken,
		rotation.newAccessToken,
		rotation.newRefreshToken,
	)
	if err == nil {
		return true
	}
	logger.FromGin(c).Error("failed to rotate self-signed session", zap.String("session_id", rotation.sessionID), zap.Error(err))
	h.clearTokenCookies(c)
	response.InternalError(c, "failed to refresh token")
	return false
}
