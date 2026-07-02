package admission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

// insertOutboxRow 直接插入一行 outbox，便于精确控制 status 与 updated_at。
func insertOutboxRow(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	sessionID, actionKey, status string,
	updatedAt time.Time,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO admission_bot_action_outbox (
			action_key, session_id, action, platform, bot_self_id, guild_id, channel_id, qq_id,
			scheduled_at, status, attempt_count, next_attempt_at, created_at, updated_at
		) VALUES (
			$1, $2, 'remind', 'qq', '514', 'guild-1', 'channel-1', '10001',
			$4, $3, 0, $4, $4, $4
		)
	`, actionKey, sessionID, status, updatedAt)
	require.NoError(t, err)
}

func countOutboxRows(t *testing.T, fixture *postgresfixture.Fixture) int {
	t.Helper()
	var count int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM admission_bot_action_outbox
	`).Scan(&count)
	require.NoError(t, err)
	return count
}

func outboxStatusExists(t *testing.T, fixture *postgresfixture.Fixture, actionKey string) bool {
	t.Helper()
	var exists bool
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM admission_bot_action_outbox WHERE action_key = $1)
	`, actionKey).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func TestPruneTerminalBotActionsDeletesOnlyAgedTerminalRows(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-prune",
		QQID:      "10001",
		TokenHash: "token-hash-prune",
		Status:    StatusJoinedMuted,
	})

	now := fixedAdmissionNow()
	aged := now.Add(-admissionBotActionRetention - time.Hour)
	fresh := now.Add(-time.Hour)

	// 过期终态：应被删除。
	insertOutboxRow(t, fixture, "adm-prune", "aged-succeeded", "succeeded", aged)
	insertOutboxRow(t, fixture, "adm-prune", "aged-dead-letter", "dead_letter", aged)
	insertOutboxRow(t, fixture, "adm-prune", "aged-stale", "stale", aged)
	// 过期但非终态：必须保留（在途动作不能误删）。
	insertOutboxRow(t, fixture, "adm-prune", "aged-pending", "pending", aged)
	insertOutboxRow(t, fixture, "adm-prune", "aged-failed", "failed", aged)
	insertOutboxRow(t, fixture, "adm-prune", "aged-dispatched", "dispatched", aged)
	// 终态但未过期：必须保留（保留期内供排障）。
	insertOutboxRow(t, fixture, "adm-prune", "fresh-succeeded", "succeeded", fresh)

	deleted, err := svc.PruneTerminalBotActions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	assert.Equal(t, 4, countOutboxRows(t, fixture))
	assert.False(t, outboxStatusExists(t, fixture, "aged-succeeded"))
	assert.False(t, outboxStatusExists(t, fixture, "aged-dead-letter"))
	assert.False(t, outboxStatusExists(t, fixture, "aged-stale"))
	assert.True(t, outboxStatusExists(t, fixture, "aged-pending"))
	assert.True(t, outboxStatusExists(t, fixture, "aged-failed"))
	assert.True(t, outboxStatusExists(t, fixture, "aged-dispatched"))
	assert.True(t, outboxStatusExists(t, fixture, "fresh-succeeded"))
}

func TestPruneTerminalBotActionsNoopWhenNothingAged(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-prune-noop",
		QQID:      "10001",
		TokenHash: "token-hash-prune-noop",
		Status:    StatusJoinedMuted,
	})
	insertOutboxRow(t, fixture, "adm-prune-noop", "fresh-stale", "stale", fixedAdmissionNow().Add(-time.Minute))

	deleted, err := svc.PruneTerminalBotActions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.Equal(t, 1, countOutboxRows(t, fixture))
}
