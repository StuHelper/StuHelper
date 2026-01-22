package token

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 黑名单 key 前缀
	blacklistPrefix = "token:blacklist:"
	// 用户 token 集合前缀
	userTokensPrefix = "token:user:"
)

// Blacklist Token 黑名单服务
type Blacklist struct {
	rdb *redis.Client
}

// NewBlacklist 创建黑名单服务
func NewBlacklist(rdb *redis.Client) *Blacklist {
	return &Blacklist{rdb: rdb}
}

// hashToken 对 token 进行 SHA256 哈希，减少 Redis 内存占用
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Add 将 token 加入黑名单
func (b *Blacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	key := blacklistPrefix + hashToken(token)
	return b.rdb.Set(ctx, key, "1", expiry).Err()
}

// IsBlacklisted 检查 token 是否在黑名单中
// 安全优先：Redis 错误时返回 true，拒绝请求
func (b *Blacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	key := blacklistPrefix + hashToken(token)
	exists, err := b.rdb.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("redis error checking blacklist: %v", err)
		return true, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return exists > 0, nil
}

// RevokeAllUserTokens 撤销用户的所有 token
func (b *Blacklist) RevokeAllUserTokens(ctx context.Context, userID string, expiry time.Duration) error {
	key := userTokensPrefix + userID
	tokenHashes, err := b.rdb.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get user tokens: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil
	}

	pipe := b.rdb.Pipeline()
	for _, tokenHash := range tokenHashes {
		pipe.Set(ctx, blacklistPrefix+tokenHash, "1", expiry)
	}
	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	return err
}

// TrackUserToken 记录用户的 token（用于批量撤销）
func (b *Blacklist) TrackUserToken(ctx context.Context, userID, token string, expiry time.Duration) error {
	key := userTokensPrefix + userID
	tokenHash := hashToken(token)
	pipe := b.rdb.Pipeline()
	pipe.SAdd(ctx, key, tokenHash)
	pipe.Expire(ctx, key, expiry)
	_, err := pipe.Exec(ctx)
	return err
}
