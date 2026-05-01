package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	ref := RefreshTokenRef{SessionID: data.SessionID, UserID: data.UserID}
	encoded, err := json.Marshal(ref)
	if err != nil {
		return nil, fmt.Errorf("refresh token ref marshal: %w", err)
	}
	return encoded, nil
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
