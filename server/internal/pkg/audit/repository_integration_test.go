package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestRepository_WriteListAndCleanupAdminOperations(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	require.NoError(t, repo.WriteEvent(ctx, Event{
		Type:         EventAdminConfigChange,
		Category:     "admin_operation",
		ActorType:    "admin",
		UserID:       "admin-1",
		Username:     "root",
		Action:       "update",
		ResourceType: "school_config",
		ResourceID:   "10006",
		RequestID:    "req-1",
		TraceID:      "0123456789abcdef0123456789abcdef",
		IP:           "127.0.0.1",
		UserAgent:    "go-test",
		Result:       "success",
		Before:       map[string]any{"enabled": false},
		After:        map[string]any{"enabled": true},
		Details:      map[string]any{"field": "enabled"},
	}))

	logs, total, err := repo.ListAdminOperations(ctx, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, logs, 1)
	assert.Equal(t, "admin-1", logs[0].ActorUserID)
	assert.Equal(t, "school_config", logs[0].ResourceType)
	assert.JSONEq(t, `{"enabled":false}`, string(logs[0].BeforeData))
	assert.JSONEq(t, `{"enabled":true}`, string(logs[0].AfterData))

	_, err = fixture.Pool.Exec(ctx, `UPDATE audit_events SET created_at = NOW() - INTERVAL '120 days' WHERE id = $1`, logs[0].ID)
	require.NoError(t, err)

	deleted, err := repo.CleanupAdminOperations(ctx, 90)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
}
