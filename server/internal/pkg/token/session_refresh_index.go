package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionRefreshPrefix = "session:refresh:"

type RefreshTokenRef struct {
	SessionID string `json:"sessionID"`
	UserID    string `json:"userID"`
}

func refreshTokenRefKey(refreshTokenHash string) string {
	return sessionRefreshPrefix + refreshTokenHash
}

func marshalRefreshTokenRef(data SessionData) ([]byte, error) {
	return marshalRefreshTokenRefValue(RefreshTokenRef{SessionID: data.SessionID, UserID: data.UserID})
}

func marshalRefreshTokenRefValue(ref RefreshTokenRef) ([]byte, error) {
	encoded, err := json.Marshal(ref)
	if err != nil {
		return nil, fmt.Errorf("refresh token ref marshal: %w", err)
	}
	return encoded, nil
}

// RememberRefreshTokenHash writes or extends refresh-token attribution metadata
// using the caller-provided TTL. It is used when a refresh hash is blacklisted
// so reuse detection can still identify the affected session family for the
// full blacklist lifetime.
func (s *SessionStore) RememberRefreshTokenHash(ctx context.Context, refreshTokenHash string, ref RefreshTokenRef, ttl time.Duration) error {
	if refreshTokenHash == "" {
		return fmt.Errorf("refresh token ref remember: refresh token hash is required")
	}
	if ref.SessionID == "" || ref.UserID == "" {
		return fmt.Errorf("refresh token ref remember: sessionID and userID are required")
	}
	if ttl <= 0 {
		return fmt.Errorf("refresh token ref remember: ttl must be positive")
	}
	encoded, err := marshalRefreshTokenRefValue(ref)
	if err != nil {
		return fmt.Errorf("refresh token ref remember: %w", err)
	}
	if err := s.rdb.Set(ctx, refreshTokenRefKey(refreshTokenHash), encoded, ttl).Err(); err != nil {
		return fmt.Errorf("refresh token ref remember: %w", err)
	}
	return nil
}

func (s *SessionStore) LookupRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*RefreshTokenRef, error) {
	if refreshTokenHash == "" {
		return nil, nil
	}

	raw, err := s.rdb.Get(ctx, refreshTokenRefKey(refreshTokenHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("refresh token ref lookup: %w", err)
	}

	var ref RefreshTokenRef
	if err := json.Unmarshal([]byte(raw), &ref); err != nil {
		return nil, fmt.Errorf("refresh token ref unmarshal: %w", err)
	}
	if ref.SessionID == "" || ref.UserID == "" {
		return nil, fmt.Errorf("refresh token ref invalid")
	}
	return &ref, nil
}
