package auth

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Service 认证业务逻辑层，封装 session 生命周期管理和登录编排。
// Handler 层只负责 HTTP 协议关切（cookie、redirect、response 格式化）。
type Service struct {
	tokenService *token.Service
	tokenConfig  config.TokenConfig
	userSyncRepo UserSyncRepo
}

// NewService 创建认证业务逻辑服务
func NewService(cfg *config.Config, tokenService *token.Service, userSyncRepo UserSyncRepo) *Service {
	return &Service{
		tokenService: tokenService,
		tokenConfig:  cfg.Token,
		userSyncRepo: userSyncRepo,
	}
}

// SessionInfo 登录成功后返回的 session 信息
type SessionInfo struct {
	SessionID string
}

// CreateSession 创建新 session 并跟踪 token 对。
// 必须在 token 签发后、写入 cookie/response 前调用。
func (s *Service) CreateSession(ctx context.Context, userID, accessToken, refreshToken, loginMethod, deviceInfo string) (*SessionInfo, error) {
	sessionID, err := token.GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	accessHash, err := hashTokenForSession(accessToken)
	if err != nil {
		return nil, fmt.Errorf("create session: hash access token: %w", err)
	}

	var refreshHash string
	if refreshToken != "" {
		refreshHash, err = hashTokenForSession(refreshToken)
		if err != nil {
			return nil, fmt.Errorf("create session: hash refresh token: %w", err)
		}
	}

	sessionData := token.SessionData{
		SessionID:        sessionID,
		UserID:           userID,
		AccessTokenHash:  accessHash,
		RefreshTokenHash: refreshHash,
		LoginMethod:      loginMethod,
		DeviceInfo:       deviceInfo,
	}

	if err := s.tokenService.GetSessionStore().Create(ctx, sessionData); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// 兼容：同时注册到旧 token 跟踪系统（支持渐进迁移）
	if trackErr := s.trackTokenPair(ctx, userID, accessToken, refreshToken); trackErr != nil {
		logger.L().Warn("create session: legacy token tracking failed (session already created)",
			zap.String("session_id", sessionID),
			zap.Error(trackErr),
		)
	}

	return &SessionInfo{SessionID: sessionID}, nil
}

// RotateSession 在 refresh 流程中轮换 session 内的 token 对。
// 旧 token 加入黑名单，新 token 更新到 session（Token Family 模式）。
func (s *Service) RotateSession(ctx context.Context, sessionID, userID, oldRefreshToken, newAccessToken, newRefreshToken string) {
	// 黑名单旧 refresh token
	if blErr := s.tokenService.GetBlacklist().Add(ctx, oldRefreshToken, s.tokenService.GetRefreshTokenTTL()); blErr != nil {
		logger.L().Warn("rotate session: failed to blacklist old refresh token", zap.Error(blErr))
	}

	// 更新 session 中的 token hash
	newAccessHash, _ := hashTokenForSession(newAccessToken)
	var newRefreshHash string
	if newRefreshToken != "" {
		newRefreshHash, _ = hashTokenForSession(newRefreshToken)
	}

	if touchErr := s.tokenService.GetSessionStore().Touch(ctx, sessionID, newAccessHash, newRefreshHash); touchErr != nil {
		logger.L().Warn("rotate session: failed to touch session",
			zap.String("session_id", sessionID),
			zap.Error(touchErr),
		)
	}

	// 兼容：同时更新旧 token 跟踪系统
	s.legacyRotateTokenPair(ctx, userID, oldRefreshToken, newAccessToken, newRefreshToken)
}

// RevokeSession 撤销单个 session（单设备登出）。
// 权威撤销操作——失败时返回 error，调用方不得向客户端承诺"登出成功"。
func (s *Service) RevokeSession(ctx context.Context, sessionID, userID, accessToken, refreshToken string) error {
	// 优先通过 session store 撤销（将 session 内 token hash 加入黑名单）
	if sessionID != "" {
		_, err := s.tokenService.GetSessionStore().Revoke(
			ctx, sessionID,
			s.tokenService.GetBlacklist(),
			s.tokenService.GetAccessTokenTTL(),
			s.tokenService.GetRefreshTokenTTL(),
		)
		if err != nil {
			return fmt.Errorf("revoke session: %w", err)
		}
	}

	// 兼容回退：如果有 token 原文，也直接加入黑名单（处理无 session ID 的旧 token）
	if accessToken != "" {
		if blErr := s.tokenService.GetBlacklist().Add(ctx, accessToken, s.tokenService.GetAccessTokenTTL()); blErr != nil {
			logger.L().Error("revoke session: failed to blacklist access token directly",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
			if sessionID == "" {
				return fmt.Errorf("revoke access token: %w", blErr)
			}
		}
	}
	if refreshToken != "" {
		if blErr := s.tokenService.GetBlacklist().Add(ctx, refreshToken, s.tokenService.GetRefreshTokenTTL()); blErr != nil {
			logger.L().Error("revoke session: failed to blacklist refresh token directly",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
			if sessionID == "" {
				return fmt.Errorf("revoke refresh token: %w", blErr)
			}
		}
	}

	// 清理旧 token 跟踪
	s.legacyUntrackTokens(ctx, userID, accessToken, refreshToken)
	return nil
}

// RevokeAllSessions 撤销用户的全部 session（全设备登出）
func (s *Service) RevokeAllSessions(ctx context.Context, userID string) error {
	// 通过 session store 撤销所有 session
	if err := s.tokenService.GetSessionStore().RevokeAll(
		ctx, userID,
		s.tokenService.GetBlacklist(),
		s.tokenService.GetAccessTokenTTL(),
		s.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		logger.L().Warn("revoke all sessions: session store revoke failed, falling back to legacy",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	// 兼容：同时撤销旧 token 跟踪系统中的所有 token
	return s.tokenService.GetBlacklist().RevokeAllUserTokens(
		ctx, userID, s.tokenService.GetRefreshTokenTTL(),
	)
}

// SignPhoneTokenPair 为手机登录用户签发自签名 JWT 对（含 session ID）
func (s *Service) SignPhoneTokenPair(user *PhoneUser, roles []string, sessionID string) (accessToken, refreshToken string, err error) {
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		return "", "", fmt.Errorf("HMAC key not initialized")
	}

	accessTTL := time.Duration(s.tokenConfig.AccessTokenTTL) * time.Second
	refreshTTL := time.Duration(s.tokenConfig.RefreshTokenTTL) * time.Second

	accessClaims := token.JWTClaims{
		Sub:         user.ExternalID,
		Name:        user.Username,
		Email:       user.Email,
		DisplayName: user.Username,
		Roles:       roles,
		Typ:         token.JWTTokenTypeAccess,
		Sid:         sessionID,
	}
	if user.AvatarURL != nil {
		accessClaims.Avatar = *user.AvatarURL
	}

	accessToken, err = token.SignJWT(hmacKey, accessClaims, accessTTL)
	if err != nil {
		return "", "", fmt.Errorf("sign access JWT: %w", err)
	}

	refreshClaims := token.JWTClaims{
		Sub:   user.ExternalID,
		Name:  user.Username,
		Roles: roles,
		Typ:   token.JWTTokenTypeRefresh,
		Sid:   sessionID,
	}
	refreshToken, err = token.SignJWT(hmacKey, refreshClaims, refreshTTL)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh JWT: %w", err)
	}

	return accessToken, refreshToken, nil
}

// SyncOIDCUser 同步 OIDC 登录的用户到本地 shadow user 表。
// 登录成功必须意味着内部主体已就绪。
func (s *Service) SyncOIDCUser(ctx context.Context, input UserSyncInput) error {
	if s.userSyncRepo == nil {
		return fmt.Errorf("user sync repository is not configured")
	}
	return s.userSyncRepo.UpsertUser(ctx, input)
}

// SyncPhoneUser 通过手机号查找或创建用户。
func (s *Service) SyncPhoneUser(ctx context.Context, phone string) (*PhoneUser, error) {
	if s.userSyncRepo == nil {
		return nil, fmt.Errorf("user sync repository is not configured")
	}
	return s.userSyncRepo.UpsertByPhone(ctx, phone)
}

// UserExistsByExternalID 检查用户是否存在（用于 refresh token 校验）。
func (s *Service) UserExistsByExternalID(ctx context.Context, externalID string) (bool, error) {
	if s.userSyncRepo == nil {
		return false, fmt.Errorf("user sync repository is not configured")
	}
	return s.userSyncRepo.ExistsByExternalID(ctx, externalID)
}

// ---- 兼容旧 token 跟踪系统的内部方法 ----

func (s *Service) trackTokenPair(ctx context.Context, userID, accessToken, refreshToken string) error {
	if err := s.tokenService.GetBlacklist().TrackUserToken(
		ctx, userID, accessToken, token.TokenTypeAccess, time.Now().Add(s.tokenService.GetAccessTokenTTL()),
	); err != nil {
		return fmt.Errorf("track access token: %w", err)
	}
	if refreshToken != "" {
		if err := s.tokenService.GetBlacklist().TrackUserToken(
			ctx, userID, refreshToken, token.TokenTypeRefresh, time.Now().Add(s.tokenService.GetRefreshTokenTTL()),
		); err != nil {
			return fmt.Errorf("track refresh token: %w", err)
		}
	}
	return nil
}

func (s *Service) legacyRotateTokenPair(ctx context.Context, userID, oldRefreshToken, newAccessToken, newRefreshToken string) {
	if untrackErr := s.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, oldRefreshToken, token.TokenTypeRefresh); untrackErr != nil {
		logger.L().Warn("legacy rotate: failed to untrack old refresh token",
			zap.String("user_id", userID),
			zap.Error(untrackErr),
		)
	}

	if trackErr := s.tokenService.GetBlacklist().TrackUserToken(
		ctx, userID, newAccessToken, token.TokenTypeAccess, time.Now().Add(s.tokenService.GetAccessTokenTTL()),
	); trackErr != nil {
		logger.L().Warn("legacy rotate: failed to track new access token",
			zap.String("user_id", userID),
			zap.Error(trackErr),
		)
	}
	if newRefreshToken != "" {
		if trackErr := s.tokenService.GetBlacklist().TrackUserToken(
			ctx, userID, newRefreshToken, token.TokenTypeRefresh, time.Now().Add(s.tokenService.GetRefreshTokenTTL()),
		); trackErr != nil {
			logger.L().Warn("legacy rotate: failed to track new refresh token",
				zap.String("user_id", userID),
				zap.Error(trackErr),
			)
		}
	}
}

func (s *Service) legacyUntrackTokens(ctx context.Context, userID, accessToken, refreshToken string) {
	if accessToken != "" {
		if err := s.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, accessToken, token.TokenTypeAccess); err != nil {
			logger.L().Warn("legacy untrack: access token", zap.String("user_id", userID), zap.Error(err))
		}
	}
	if refreshToken != "" {
		if err := s.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, refreshToken, token.TokenTypeRefresh); err != nil {
			logger.L().Warn("legacy untrack: refresh token", zap.String("user_id", userID), zap.Error(err))
		}
	}
}

// hashTokenForSession 生成 token 的 HMAC hash（用于 session 内存储）
func hashTokenForSession(tokenStr string) (string, error) {
	return crypto.HMACHash(tokenStr)
}
