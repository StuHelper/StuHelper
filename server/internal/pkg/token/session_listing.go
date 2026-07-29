package token

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

// ListUserSessions 列出用户的所有活跃 session（用于"管理登录设备"等 UI）。
func (s *SessionStore) ListUserSessions(ctx context.Context, userID string) ([]SessionData, error) {
	sessionIDs, err := s.rdb.SMembers(ctx, userSessionsPrefix+userID).Result()
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	if len(sessionIDs) == 0 {
		return []SessionData{}, nil
	}

	sessions, staleSessionIDs, err := s.loadUserSessions(ctx, userID, sessionIDs)
	if err != nil {
		return nil, err
	}
	s.removeStaleSessionRefs(ctx, userID, staleSessionIDs)
	return sessions, nil
}

func (s *SessionStore) loadUserSessions(ctx context.Context, userID string, sessionIDs []string) ([]SessionData, []string, error) {
	values, err := s.rdb.MGet(ctx, sessionKeys(sessionIDs)...).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("list user sessions mget: %w", err)
	}

	sessions := make([]SessionData, 0, len(sessionIDs))
	staleSessionIDs := make([]string, 0)
	for i, raw := range values {
		session, ok := decodeListedSession(sessionIDs[i], raw)
		if !ok {
			staleSessionIDs = append(staleSessionIDs, sessionIDs[i])
			continue
		}
		if session.UserID != userID {
			logger.L().Warn("list user sessions: stale cross-user session reference",
				zap.String("user_id", userID),
				zap.Bool("session_reference_present", sessionIDs[i] != ""),
				zap.String("session_user_id", session.UserID),
			)
			staleSessionIDs = append(staleSessionIDs, sessionIDs[i])
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, staleSessionIDs, nil
}

func sessionKeys(sessionIDs []string) []string {
	keys := make([]string, 0, len(sessionIDs))
	for _, sid := range sessionIDs {
		keys = append(keys, sessionPrefix+sid)
	}
	return keys
}

func decodeListedSession(sessionID string, raw any) (SessionData, bool) {
	payload, ok := sessionPayloadString(sessionID, raw)
	if !ok {
		return SessionData{}, false
	}

	var data SessionData
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		logger.L().Warn("list user sessions: failed to decode session payload",
			zap.Bool("session_reference_present", sessionID != ""),
			zap.Error(err),
		)
		return SessionData{}, false
	}
	return data, true
}

func sessionPayloadString(sessionID string, raw any) (string, bool) {
	switch value := raw.(type) {
	case nil:
		return "", false
	case string:
		return value, true
	case []byte:
		return string(value), true
	default:
		logger.L().Warn("list user sessions: unexpected session payload type",
			zap.Bool("session_reference_present", sessionID != ""),
			zap.String("type", fmt.Sprintf("%T", raw)),
		)
		return "", false
	}
}

func (s *SessionStore) removeStaleSessionRefs(ctx context.Context, userID string, sessionIDs []string) {
	if len(sessionIDs) == 0 {
		return
	}

	staleMembers := make([]interface{}, 0, len(sessionIDs))
	for _, sid := range sessionIDs {
		staleMembers = append(staleMembers, sid)
	}
	if err := s.rdb.SRem(ctx, userSessionsPrefix+userID, staleMembers...).Err(); err != nil {
		logger.L().Warn("session index cleanup: failed to remove stale session references",
			zap.String("user_id", userID),
			zap.Int("stale_count", len(sessionIDs)),
			zap.Error(err),
		)
	}
}
