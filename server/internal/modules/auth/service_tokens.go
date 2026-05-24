package auth

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

// SyncOIDCUser 同步 OIDC 登录的用户到本地 shadow user 表。
// 登录成功必须意味着内部主体已就绪。
func (s *Service) SyncOIDCUser(ctx context.Context, input UserSyncInput) error {
	return s.userSyncRepo.UpsertUser(ctx, input)
}

// UserExistsByCasdoorSubject 检查用户是否存在（用于 refresh token 校验）。
func (s *Service) UserExistsByCasdoorSubject(ctx context.Context, casdoorSubject string) (bool, error) {
	return s.userSyncRepo.ExistsByCasdoorSubject(ctx, casdoorSubject)
}

// hashTokenForSession 生成 token 的 HMAC hash（用于 session 内存储）
func hashTokenForSession(tokenStr string) (string, error) {
	return crypto.HMACHash(tokenStr)
}
