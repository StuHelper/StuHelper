package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Service 认证业务逻辑层，封装 session 生命周期管理和登录编排。
// Handler 层只负责 HTTP 协议关切（cookie、redirect、response 格式化）。
type Service struct {
	tokenService   *token.Service
	tokenConfig    config.TokenConfig
	userSyncRepo   UserSyncRepo
	providerTokens providerTokenCoordinator
}

// NewService 创建认证业务逻辑服务。
// 开发期采用 fail-fast：关键依赖缺失直接 panic，而不是把“不可能状态”带进热路径。
func NewService(
	tokenConfig config.TokenConfig,
	tokenService *token.Service,
	userSyncRepo UserSyncRepo,
	opts ...ServiceOption,
) *Service {
	if tokenService == nil {
		panic("auth.NewService: tokenService is required")
	}
	if userSyncRepo == nil {
		panic("auth.NewService: userSyncRepo is required")
	}
	svc := &Service{
		tokenService: tokenService,
		tokenConfig:  tokenConfig,
		userSyncRepo: userSyncRepo,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

// SessionInfo 登录成功后返回的 session 信息
type SessionInfo struct {
	SessionID string
}

type sessionRotationCommittedError struct {
	err error
}

func (e *sessionRotationCommittedError) Error() string {
	return e.err.Error()
}

func (e *sessionRotationCommittedError) Unwrap() error {
	return e.err
}

func sessionRotationCommitted(err error) bool {
	var committedErr *sessionRotationCommittedError
	return errors.As(err, &committedErr)
}

// CreateSession 在 Redis session store 中注册新 session。
// 调用方必须在签发 JWT 前生成 sessionID（token.GenerateSessionID），
// 并在签发 JWT 的 Sid claim 中使用同一值——否则 JWT sid 与服务端存储的
// session key 不一致，refresh / revoke 都会失效。
func (s *Service) CreateSession(ctx context.Context, sessionID, userID, accessToken, refreshToken, loginMethod, deviceInfo string) (*SessionInfo, error) {
	return s.CreateSessionForApplication(ctx, sessionID, userID, accessToken, refreshToken, loginMethod, "", deviceInfo)
}

func (s *Service) CreateSessionForApplication(
	ctx context.Context,
	sessionID, userID, accessToken, refreshToken, loginMethod, providerAppKey, deviceInfo string,
) (*SessionInfo, error) {
	var err error
	sessionID, err = normalizeRequiredSessionID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	userID, err = normalizeRequiredSessionUserID(userID)
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
	providerRefreshTokenEnc, err := s.encryptProviderRefreshToken(loginMethod, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	sessionData := token.SessionData{
		SessionID:               sessionID,
		UserID:                  userID,
		AccessTokenHash:         accessHash,
		RefreshTokenHash:        refreshHash,
		ProviderRefreshTokenEnc: providerRefreshTokenEnc,
		ProviderAppKey:          providerAppKeyForSession(loginMethod, providerAppKey),
		LoginMethod:             loginMethod,
		DeviceInfo:              deviceInfo,
	}

	if err := s.tokenService.GetSessionStore().Create(ctx, sessionData); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &SessionInfo{SessionID: sessionID}, nil
}

func (s *Service) OIDCApplicationForRefresh(ctx context.Context, sessionID, refreshToken string) (string, error) {
	var err error
	sessionID, err = normalizeRequiredSessionID(sessionID)
	if err != nil {
		return "", fmt.Errorf("resolve oidc application: %w", err)
	}
	session, err := s.verifyTrackedSession(ctx, sessionID, trackedSessionExpectation{refreshToken: refreshToken})
	if err != nil {
		return "", fmt.Errorf("resolve oidc application: %w", err)
	}
	return providerAppKeyForSession(session.LoginMethod, session.ProviderAppKey), nil
}

// RotateSession 在 refresh 流程中轮换 session 内的 token 对（Token Family 模式）。
// 步骤：
//  1. 校验旧 refresh token 仍属于该 tracked session
//  2. 计算新 token 的 HMAC hash
//  3. 原子更新 session store
//  4. 撤销旧 provider refresh token
//  5. 将旧 refresh token 加入黑名单
//
// 任何步骤失败都返回 error。Touch 前失败表示本地 session 未提交，调用方可
// 补偿撤销刚签发但尚未写入 session 的 provider refresh token。Touch 后的
// provider token 撤销和旧 refresh token 黑名单错误会以
// sessionRotationCommitted 标记为已提交，调用方不得把新 provider refresh
// token 当作孤儿 token 撤销。
//
// sessionID 为空时直接失败。调用方必须先定位到被追踪的 session family，
// 否则 refresh 不能继续签发“不受 session store 跟踪”的新 token。
func (s *Service) RotateSession(ctx context.Context, sessionID, userID, oldRefreshToken, newAccessToken, newRefreshToken string) error {
	var err error
	userID, err = normalizeRequiredSessionUserID(userID)
	if err != nil {
		return fmt.Errorf("rotate session: %w", err)
	}

	sessionID, err = normalizeRequiredSessionID(sessionID)
	if err != nil {
		return fmt.Errorf("rotate session: %w", err)
	}
	session, err := s.verifyTrackedSession(ctx, sessionID, trackedSessionExpectation{
		userID:       userID,
		refreshToken: oldRefreshToken,
	})
	if err != nil {
		return fmt.Errorf("rotate session: %w", err)
	}

	newAccessHash, err := hashTokenForSession(newAccessToken)
	if err != nil {
		return fmt.Errorf("rotate session: hash new access token: %w", err)
	}
	var newRefreshHash string
	if newRefreshToken != "" {
		newRefreshHash, err = hashTokenForSession(newRefreshToken)
		if err != nil {
			return fmt.Errorf("rotate session: hash new refresh token: %w", err)
		}
	}
	providerRefreshTokenEnc, err := s.encryptProviderRefreshToken(session.LoginMethod, newRefreshToken)
	if err != nil {
		return fmt.Errorf("rotate session: %w", err)
	}

	update := token.SessionTouchUpdate{
		AccessTokenHash:         newAccessHash,
		RefreshTokenHash:        newRefreshHash,
		ProviderRefreshTokenEnc: providerRefreshTokenEnc,
		ProviderAppKey:          providerAppKeyForSession(session.LoginMethod, session.ProviderAppKey),
	}
	if touchErr := s.tokenService.GetSessionStore().Touch(ctx, sessionID, update); touchErr != nil {
		return fmt.Errorf("rotate session: touch session: %w", touchErr)
	}

	var cleanupErr error
	if err := s.revokeProviderRefreshTokenFromSession(ctx, session); err != nil {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("revoke old provider refresh token: %w", err))
	}

	refreshTTL := s.tokenService.GetRefreshTokenTTL()
	if session.RefreshTokenHash != "" {
		if err := s.tokenService.GetSessionStore().RememberRefreshTokenHash(ctx, session.RefreshTokenHash, token.RefreshTokenRef{
			SessionID: session.SessionID,
			UserID:    session.UserID,
		}, refreshTTL); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remember old refresh token ref: %w", err))
		}
	}
	if blErr := s.tokenService.GetBlacklist().Add(ctx, oldRefreshToken, refreshTTL); blErr != nil {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("blacklist old refresh token: %w", blErr))
	}
	if cleanupErr != nil {
		return &sessionRotationCommittedError{err: fmt.Errorf("rotate session cleanup after commit: %w", cleanupErr)}
	}
	return nil
}

// RevokeSession 撤销单个 session（单设备登出）。
// 权威撤销操作——失败时返回 error，调用方不得向客户端承诺"登出成功"。
//
// 优先通过 session store 撤销（将 session 内 token hash 加入黑名单）；
// 若调用方持有 token 原文（例如 session ID 无法解析），直接按 token 原文加黑名单作兜底。
func (s *Service) RevokeSession(ctx context.Context, sessionID, userID, accessToken, refreshToken string) error {
	var err error
	userID, err = normalizeRequiredSessionUserID(userID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" {
		session, err := s.verifyTrackedSession(ctx, sessionID, trackedSessionExpectation{
			userID:       userID,
			accessToken:  accessToken,
			refreshToken: refreshToken,
		})
		if err != nil {
			return fmt.Errorf("revoke session: %w", err)
		}
		_, err = s.tokenService.GetSessionStore().Revoke(
			ctx, sessionID,
			s.tokenService.GetBlacklist(),
			s.tokenService.GetAccessTokenTTL(),
			s.tokenService.GetRefreshTokenTTL(),
		)
		if err != nil {
			return fmt.Errorf("revoke session: %w", err)
		}
		if err := s.revokeProviderRefreshTokenFromSession(ctx, session); err != nil {
			return fmt.Errorf("revoke session: local session revoked; provider revoke failed: %w", err)
		}
	}

	// 兜底：无 session ID 或 session 已过期时，按 token 原文加黑名单
	if accessToken != "" {
		if blErr := s.tokenService.GetBlacklist().Add(ctx, accessToken, s.tokenService.GetAccessTokenTTL()); blErr != nil {
			logger.L().Error("revoke session: failed to blacklist access token",
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
			logger.L().Error("revoke session: failed to blacklist refresh token",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
			if sessionID == "" {
				return fmt.Errorf("revoke refresh token: %w", blErr)
			}
		}
	}
	return nil
}

func (s *Service) verifyTrackedSession(
	ctx context.Context,
	sessionID string,
	expectation trackedSessionExpectation,
) (*token.SessionData, error) {
	session, err := loadTrackedSession(ctx, s.tokenService.GetSessionStore(), sessionID)
	if err != nil {
		if errors.Is(err, token.ErrSessionNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("load session: %w", err)
	}
	if err := validateTrackedSession(session, expectation); err != nil {
		return nil, err
	}
	return session, nil
}

// RevokeAllSessions 撤销用户的全部 session（全设备登出）。
// 仅依赖 session store；不再回退到旧 token 跟踪系统。
func (s *Service) RevokeAllSessions(ctx context.Context, userID string) error {
	var err error
	userID, err = normalizeRequiredSessionUserID(userID)
	if err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}

	providerErr := s.revokeProviderRefreshTokensForUser(ctx, userID)
	if err := s.tokenService.GetSessionStore().RevokeAll(
		ctx, userID,
		s.tokenService.GetBlacklist(),
		s.tokenService.GetAccessTokenTTL(),
		s.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		return fmt.Errorf("revoke all sessions: %w", errors.Join(err, providerErr))
	}
	if providerErr != nil {
		return fmt.Errorf("revoke all sessions: local sessions revoked; provider revoke failed: %w", providerErr)
	}
	return nil
}

func normalizeRequiredSessionUserID(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", errSessionUserRequired
	}
	return userID, nil
}
