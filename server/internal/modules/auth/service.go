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

// Service 认证业务逻辑层，封装 token 生命周期管理和登录编排。
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

// TrackTokenPair 注册一对 token 到会话跟踪系统（支持 LogoutAll）。
// 必须在 token 签发后、写入 Cookie 前调用。失败时应中止登录流程。
func (s *Service) TrackTokenPair(ctx context.Context, userID, accessToken, refreshToken string) error {
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

// RotateTokenPair 黑名单旧 token 并跟踪新 token（用于 refresh 流程）。
// 旧 token 的黑名单化为 best-effort（warn-only），新 token 跟踪失败也 warn-only，
// 因为 refresh 成功后 token 已签发。
func (s *Service) RotateTokenPair(ctx context.Context, userID, oldRefreshToken, newAccessToken, newRefreshToken string) {
	// 黑名单旧 refresh token
	if blErr := s.tokenService.GetBlacklist().Add(ctx, oldRefreshToken, s.tokenService.GetRefreshTokenTTL()); blErr != nil {
		logger.L().Warn("failed to blacklist old refresh token", zap.Error(blErr))
	}
	if untrackErr := s.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, oldRefreshToken, token.TokenTypeRefresh); untrackErr != nil {
		logger.L().Warn("failed to untrack old refresh token",
			zap.String("user_id", userID),
			zap.Error(untrackErr),
		)
	}

	// 跟踪新 token（best-effort：refresh 成功后 token 已签发）
	if trackErr := s.tokenService.GetBlacklist().TrackUserToken(
		ctx, userID, newAccessToken, token.TokenTypeAccess, time.Now().Add(s.tokenService.GetAccessTokenTTL()),
	); trackErr != nil {
		logger.L().Warn("failed to track refreshed access token",
			zap.String("user_id", userID),
			zap.Error(trackErr),
		)
	}
	if newRefreshToken != "" {
		if trackErr := s.tokenService.GetBlacklist().TrackUserToken(
			ctx, userID, newRefreshToken, token.TokenTypeRefresh, time.Now().Add(s.tokenService.GetRefreshTokenTTL()),
		); trackErr != nil {
			logger.L().Warn("failed to track refreshed refresh token",
				zap.String("user_id", userID),
				zap.Error(trackErr),
			)
		}
	}
}

// RevokeCurrentSession 将指定 token 加入黑名单并解除跟踪（用于 Logout 单设备）。
// 黑名单写入是权威撤销操作——失败时返回 error，调用方不得向客户端承诺"登出成功"。
// untrack 为辅助清理，失败仅 warn。
func (s *Service) RevokeCurrentSession(ctx context.Context, userID, accessToken, refreshToken string) error {
	if accessToken != "" {
		if blErr := s.tokenService.GetBlacklist().Add(ctx, accessToken, s.tokenService.GetAccessTokenTTL()); blErr != nil {
			logger.L().Error("failed to blacklist access token",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
			return fmt.Errorf("revoke access token: %w", blErr)
		}
		if untrackErr := s.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, accessToken, token.TokenTypeAccess); untrackErr != nil {
			logger.L().Warn("failed to untrack access token",
				zap.String("user_id", userID),
				zap.Error(untrackErr),
			)
		}
	}
	if refreshToken != "" {
		if blErr := s.tokenService.GetBlacklist().Add(ctx, refreshToken, s.tokenService.GetRefreshTokenTTL()); blErr != nil {
			logger.L().Error("failed to blacklist refresh token",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
			return fmt.Errorf("revoke refresh token: %w", blErr)
		}
		if untrackErr := s.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, refreshToken, token.TokenTypeRefresh); untrackErr != nil {
			logger.L().Warn("failed to untrack refresh token",
				zap.String("user_id", userID),
				zap.Error(untrackErr),
			)
		}
	}
	return nil
}

// RevokeAllSessions 撤销用户的全部 token（用于 LogoutAll）
func (s *Service) RevokeAllSessions(ctx context.Context, userID string) error {
	return s.tokenService.GetBlacklist().RevokeAllUserTokens(
		ctx, userID, s.tokenService.GetRefreshTokenTTL(),
	)
}

// SignPhoneTokenPair 为手机登录用户签发自签名 JWT 对
func (s *Service) SignPhoneTokenPair(user *PhoneUser, roles []string) (accessToken, refreshToken string, err error) {
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
