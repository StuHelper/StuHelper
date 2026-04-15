// Package token 的 session.go 提供基于 Redis 的服务端 Session 管理。
//
// 每次登录创建一个独立 Session（唯一 sessionID），access + refresh token 归属于该 Session。
// logout 按 sessionID 撤销（而非按单个 token 值），语义更稳定。
// refresh 轮换在同一 Session 内进行（Token Family 模式）。
package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// Redis key 前缀
const (
	// session:{sessionID} → JSON-encoded SessionData
	sessionPrefix = "session:"
	// user:sessions:{userID} → Set of sessionIDs
	userSessionsPrefix = "user:sessions:"
)

// SessionData 服务端存储的 session 元数据
type SessionData struct {
	SessionID        string `json:"sessionID"`
	UserID           string `json:"userID"`
	CreatedAt        int64  `json:"createdAt"`
	LastActiveAt     int64  `json:"lastActiveAt"`
	AccessTokenHash  string `json:"accessTokenHash"`
	RefreshTokenHash string `json:"refreshTokenHash,omitempty"`
	// DeviceInfo 可选的设备信息（UA / IP / 平台标识）
	DeviceInfo string `json:"deviceInfo,omitempty"`
	// LoginMethod 登录方式（oidc / phone）
	LoginMethod string `json:"loginMethod,omitempty"`
}

// SessionStore 管理服务端 session 生命周期
type SessionStore struct {
	rdb        *redis.Client
	sessionTTL time.Duration // session 最大存活时间（与 refresh token TTL 对齐）
}

// NewSessionStore 创建 session store
func NewSessionStore(rdb *redis.Client, sessionTTL time.Duration) *SessionStore {
	return &SessionStore{
		rdb:        rdb,
		sessionTTL: sessionTTL,
	}
}

// GenerateSessionID 生成 16 字节随机 session ID
func GenerateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create 创建新 session，同时将 sessionID 加入用户的 session 集合。
// 必须在 token 签发后、写入 cookie/response 前调用。
func (s *SessionStore) Create(ctx context.Context, data SessionData) error {
	if data.SessionID == "" || data.UserID == "" {
		return fmt.Errorf("session create: sessionID and userID are required")
	}

	now := time.Now().Unix()
	data.CreatedAt = now
	data.LastActiveAt = now

	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("session create: marshal: %w", err)
	}

	pipe := s.rdb.Pipeline()
	sessionKey := sessionPrefix + data.SessionID
	userKey := userSessionsPrefix + data.UserID

	pipe.Set(ctx, sessionKey, encoded, s.sessionTTL)
	pipe.SAdd(ctx, userKey, data.SessionID)
	pipe.Expire(ctx, userKey, s.sessionTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("session create: redis pipeline: %w", err)
	}
	return nil
}

// Get 获取 session 数据。session 不存在或已过期时返回 nil。
func (s *SessionStore) Get(ctx context.Context, sessionID string) (*SessionData, error) {
	raw, err := s.rdb.Get(ctx, sessionPrefix+sessionID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("session get: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("session get: unmarshal: %w", err)
	}
	return &data, nil
}

// Touch 更新 session 的 lastActiveAt 和 token 哈希（用于 refresh 轮换）。
func (s *SessionStore) Touch(ctx context.Context, sessionID, newAccessHash, newRefreshHash string) error {
	raw, err := s.rdb.Get(ctx, sessionPrefix+sessionID).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("session touch: session not found")
		}
		return fmt.Errorf("session touch: get: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return fmt.Errorf("session touch: unmarshal: %w", err)
	}

	data.LastActiveAt = time.Now().Unix()
	if newAccessHash != "" {
		data.AccessTokenHash = newAccessHash
	}
	if newRefreshHash != "" {
		data.RefreshTokenHash = newRefreshHash
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("session touch: marshal: %w", err)
	}

	// 续期 session TTL
	if err := s.rdb.Set(ctx, sessionPrefix+sessionID, encoded, s.sessionTTL).Err(); err != nil {
		return fmt.Errorf("session touch: set: %w", err)
	}
	return nil
}

// Revoke 撤销单个 session，同时将该 session 内的 token 加入黑名单。
// 返回被撤销的 session 数据（供调用方决定是否需要额外清理）。
func (s *SessionStore) Revoke(ctx context.Context, sessionID string, blacklist *Blacklist, accessTTL, refreshTTL time.Duration) (*SessionData, error) {
	data, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session revoke: %w", err)
	}
	if data == nil {
		// session 已过期或不存在，视为已撤销
		return nil, nil
	}

	// 将 session 内的 token hash 加入黑名单
	if data.AccessTokenHash != "" {
		if blErr := blacklist.AddByHash(ctx, data.AccessTokenHash, accessTTL); blErr != nil {
			return nil, fmt.Errorf("session revoke: blacklist access token: %w", blErr)
		}
	}
	if data.RefreshTokenHash != "" {
		if blErr := blacklist.AddByHash(ctx, data.RefreshTokenHash, refreshTTL); blErr != nil {
			return nil, fmt.Errorf("session revoke: blacklist refresh token: %w", blErr)
		}
	}

	// 删除 session key 和从用户集合中移除
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, sessionPrefix+sessionID)
	pipe.SRem(ctx, userSessionsPrefix+data.UserID, sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.L().Warn("session revoke: cleanup failed (tokens already blacklisted)",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
	}

	return data, nil
}

// RevokeAll 撤销用户的所有 session。
func (s *SessionStore) RevokeAll(ctx context.Context, userID string, blacklist *Blacklist, accessTTL, refreshTTL time.Duration) error {
	userKey := userSessionsPrefix + userID

	sessionIDs, err := s.rdb.SMembers(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("session revoke all: get sessions: %w", err)
	}

	for _, sid := range sessionIDs {
		if _, rErr := s.Revoke(ctx, sid, blacklist, accessTTL, refreshTTL); rErr != nil {
			logger.L().Warn("session revoke all: failed to revoke session",
				zap.String("user_id", userID),
				zap.String("session_id", sid),
				zap.Error(rErr),
			)
		}
	}

	// 清理用户 session 集合
	if err := s.rdb.Del(ctx, userKey).Err(); err != nil {
		logger.L().Warn("session revoke all: failed to delete user sessions set",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}
	return nil
}

// ListUserSessions 列出用户的所有活跃 session（用于"管理登录设备"等 UI）。
func (s *SessionStore) ListUserSessions(ctx context.Context, userID string) ([]SessionData, error) {
	sessionIDs, err := s.rdb.SMembers(ctx, userSessionsPrefix+userID).Result()
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}

	var sessions []SessionData
	for _, sid := range sessionIDs {
		data, getErr := s.Get(ctx, sid)
		if getErr != nil {
			logger.L().Warn("list user sessions: failed to get session",
				zap.String("session_id", sid),
				zap.Error(getErr),
			)
			continue
		}
		if data != nil {
			sessions = append(sessions, *data)
		} else {
			// session 已过期，从集合中清理
			_ = s.rdb.SRem(ctx, userSessionsPrefix+userID, sid).Err()
		}
	}
	return sessions, nil
}
