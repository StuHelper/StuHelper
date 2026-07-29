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
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

// touchSessionScript 原子地读取、合并 lastActiveAt / tokenHash 字段并重写 session + 续期 TTL。
// 作为 Lua 脚本执行避免两个并发 refresh 之间的 read-modify-write 竞态——
// 否则较晚完成的请求会用过期的 access/refresh hash 覆盖已经轮换过的值，
// 黑名单就失去了对旧 token 的追踪能力（安全退化）。
//
// KEYS[1] = session key (session:<sessionID>)
// ARGV[1] = new lastActiveAt unix timestamp (string)
// ARGV[2] = new access token hash (empty string == keep current)
// ARGV[3] = new refresh token hash (empty string == keep current)
// ARGV[4] = provider refresh token ciphertext (empty string == keep current)
// ARGV[5] = provider app key (empty string == keep current)
// ARGV[6] = TTL seconds
// ARGV[7] = user session set prefix
//
// KEYS[2] = refresh token ref key for ARGV[3] (ignored when ARGV[3] is empty)
//
// 返回 1 表示成功，0 表示 session 不存在。
const touchSessionScript = `
local raw = redis.call('GET', KEYS[1])
if not raw then
    return 0
end
local data = cjson.decode(raw)
data['lastActiveAt'] = tonumber(ARGV[1])
if ARGV[2] ~= '' then
    data['accessTokenHash'] = ARGV[2]
end
if ARGV[3] ~= '' then
    data['refreshTokenHash'] = ARGV[3]
    local ref = {sessionID=data['sessionID'], userID=data['userID']}
	    redis.call('SET', KEYS[2], cjson.encode(ref), 'EX', tonumber(ARGV[6]))
end
if ARGV[4] ~= '' then
    data['providerRefreshTokenEnc'] = ARGV[4]
end
if ARGV[5] ~= '' then
    data['providerAppKey'] = ARGV[5]
end
if data['userID'] and data['sessionID'] then
    local userKey = ARGV[7] .. data['userID']
    redis.call('SADD', userKey, data['sessionID'])
    redis.call('EXPIRE', userKey, tonumber(ARGV[6]))
end
redis.call('SET', KEYS[1], cjson.encode(data), 'EX', tonumber(ARGV[6]))
return 1
`

var touchSessionScriptObj = redis.NewScript(touchSessionScript)

// ErrSessionNotFound session 不存在或已过期。
var ErrSessionNotFound = errors.New("session not found")

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
	// ProviderRefreshTokenEnc stores an encrypted provider refresh token.
	ProviderRefreshTokenEnc string `json:"providerRefreshTokenEnc,omitempty"`
	// ProviderAppKey records which OIDC application owns provider token ops.
	ProviderAppKey string `json:"providerAppKey,omitempty"`
	// DeviceInfo 可选的设备信息（UA / IP / 平台标识）
	DeviceInfo string `json:"deviceInfo,omitempty"`
	// LoginMethod 登录方式（oidc / phone）
	LoginMethod string `json:"loginMethod,omitempty"`
}

type SessionTouchUpdate struct {
	AccessTokenHash         string
	RefreshTokenHash        string
	ProviderRefreshTokenEnc string
	ProviderAppKey          string
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

	var refreshRef []byte
	if data.RefreshTokenHash != "" {
		refreshRef, err = marshalRefreshTokenRef(data)
		if err != nil {
			return fmt.Errorf("session create: %w", err)
		}
	}

	pipe := s.rdb.Pipeline()
	sessionKey := sessionPrefix + data.SessionID
	userKey := userSessionsPrefix + data.UserID

	pipe.Set(ctx, sessionKey, encoded, s.sessionTTL)
	pipe.SAdd(ctx, userKey, data.SessionID)
	pipe.Expire(ctx, userKey, s.sessionTTL)
	if len(refreshRef) > 0 {
		pipe.Set(ctx, refreshTokenRefKey(data.RefreshTokenHash), refreshRef, s.sessionTTL)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("session create: redis pipeline: %w", err)
	}
	return nil
}

// Get 获取 session 数据。session 不存在或已过期时返回 nil。
func (s *SessionStore) Get(ctx context.Context, sessionID string) (*SessionData, error) {
	raw, err := s.rdb.Get(ctx, sessionPrefix+sessionID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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

// Touch 原子地更新 session 的 lastActiveAt 和 token 哈希（用于 refresh 轮换）。
// 使用 Redis Lua 脚本在一次 RTT 内完成 GET+merge+SET+EXPIRE，避免并发 refresh
// 造成的 read-modify-write 竞态——两个 refresh 同时命中时互相覆盖，会让已
// 轮换的 token hash 失去追踪，使旧 token 绕过黑名单。session 不存在时返回
// ErrSessionNotFound。
func (s *SessionStore) Touch(ctx context.Context, sessionID string, update SessionTouchUpdate) error {
	now := time.Now().Unix()
	res, err := touchSessionScriptObj.Run(
		ctx, s.rdb,
		[]string{sessionPrefix + sessionID, refreshTokenRefKey(update.RefreshTokenHash)},
		now,
		update.AccessTokenHash,
		update.RefreshTokenHash,
		update.ProviderRefreshTokenEnc,
		update.ProviderAppKey,
		int(s.sessionTTL.Seconds()),
		userSessionsPrefix,
	).Int64()
	if err != nil {
		return fmt.Errorf("session touch: run script: %w", err)
	}
	if res == 0 {
		return ErrSessionNotFound
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

	return s.revokeLoadedSession(ctx, data, blacklist, accessTTL, refreshTTL)
}

func (s *SessionStore) revokeLoadedSession(ctx context.Context, data *SessionData, blacklist *Blacklist, accessTTL, refreshTTL time.Duration) (*SessionData, error) {
	// 将 session 内的 token hash 加入黑名单
	if data.AccessTokenHash != "" {
		if blErr := blacklist.AddByHash(ctx, data.AccessTokenHash, accessTTL); blErr != nil {
			return nil, fmt.Errorf("session revoke: blacklist access token: %w", blErr)
		}
	}
	if data.RefreshTokenHash != "" {
		if err := s.RememberRefreshTokenHash(ctx, data.RefreshTokenHash, RefreshTokenRef{
			SessionID: data.SessionID,
			UserID:    data.UserID,
		}, refreshTTL); err != nil {
			return nil, fmt.Errorf("session revoke: remember refresh token ref: %w", err)
		}
		if blErr := blacklist.AddByHash(ctx, data.RefreshTokenHash, refreshTTL); blErr != nil {
			return nil, fmt.Errorf("session revoke: blacklist refresh token: %w", blErr)
		}
	}

	// 删除 session key 和从用户集合中移除
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, sessionPrefix+data.SessionID)
	pipe.SRem(ctx, userSessionsPrefix+data.UserID, data.SessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.L().Warn("session revoke: cleanup failed (tokens already blacklisted)",
			zap.String("session_id", data.SessionID),
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

	var revokeErr error
	staleSessionIDs := make([]string, 0)
	for _, sid := range sessionIDs {
		data, err := s.Get(ctx, sid)
		if err != nil {
			logger.L().Warn("session revoke all: failed to revoke session",
				zap.String("user_id", userID),
				zap.String("session_id", sid),
				zap.Error(err),
			)
			revokeErr = errors.Join(revokeErr, fmt.Errorf("session %s: %w", sid, err))
			continue
		}
		if data == nil {
			staleSessionIDs = append(staleSessionIDs, sid)
			continue
		}
		if data.UserID != userID {
			logger.L().Warn("session revoke all: stale cross-user session reference",
				zap.String("user_id", userID),
				zap.String("session_id", sid),
				zap.String("session_user_id", data.UserID),
			)
			staleSessionIDs = append(staleSessionIDs, sid)
			continue
		}
		if _, rErr := s.revokeLoadedSession(ctx, data, blacklist, accessTTL, refreshTTL); rErr != nil {
			logger.L().Warn("session revoke all: failed to revoke session",
				zap.String("user_id", userID),
				zap.String("session_id", sid),
				zap.Error(rErr),
			)
			revokeErr = errors.Join(revokeErr, fmt.Errorf("session %s: %w", sid, rErr))
		}
	}
	s.removeStaleSessionRefs(ctx, userID, staleSessionIDs)
	if revokeErr != nil {
		return fmt.Errorf("session revoke all: revoke sessions: %w", revokeErr)
	}

	// 清理用户 session 集合
	if err := s.rdb.Del(ctx, userKey).Err(); err != nil {
		return fmt.Errorf("session revoke all: delete user sessions set: %w", err)
	}
	return nil
}
