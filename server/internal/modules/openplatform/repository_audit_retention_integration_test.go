package openplatform

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestRepositoryCleanupAuditEventsUsesTieredRetention(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	staleDisclosureID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "old-disclosure", "open_platform.disclosure.granted", 366)
	staleDisclosureDeniedID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "old-disclosure-denied", "open_platform.disclosure.denied", 366)
	staleResourceCheckID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "old-resource-check", "open_platform.resource_access.checked", 366)
	freshDisclosureID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "fresh-disclosure", "open_platform.disclosure.granted", 300)
	staleOperationalID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "old-operational", "open_platform.app.approved", 1096)
	freshOperationalID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "fresh-operational", "open_platform.app.secret_rotated", 1000)
	resourceGrantID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "resource-grant", "open_platform.resource_access.granted", 400)

	deleted, err := repo.CleanupAuditEvents(ctx, AuditRetentionPolicy{})
	require.NoError(t, err)

	assert.EqualValues(t, 4, deleted)
	assert.False(t, openPlatformAuditEventExists(t, fixture, staleDisclosureID))
	assert.False(t, openPlatformAuditEventExists(t, fixture, staleDisclosureDeniedID))
	assert.False(t, openPlatformAuditEventExists(t, fixture, staleResourceCheckID))
	assert.False(t, openPlatformAuditEventExists(t, fixture, staleOperationalID))
	assert.True(t, openPlatformAuditEventExists(t, fixture, freshDisclosureID))
	assert.True(t, openPlatformAuditEventExists(t, fixture, freshOperationalID))
	assert.True(t, openPlatformAuditEventExists(t, fixture, resourceGrantID))
}

func TestRepositoryCleanupAuditEventsSkipsLockedRows(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	lockedID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "locked", "open_platform.disclosure.granted", 366)
	unlockedID := writeStaleOpenPlatformAuditEvent(t, fixture, repo, ctx, "unlocked", "open_platform.disclosure.granted", 366)

	lockTx, err := fixture.Pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, lockTx.Rollback(context.Background()))
	})

	var selectedID int64
	err = lockTx.QueryRow(ctx, `SELECT id FROM open_platform_audit_events WHERE id = $1 FOR UPDATE`, lockedID).Scan(&selectedID)
	require.NoError(t, err)
	require.Equal(t, lockedID, selectedID)

	cleanupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	deleted, err := repo.CleanupAuditEvents(cleanupCtx, AuditRetentionPolicy{})
	require.NoError(t, err)

	assert.EqualValues(t, 1, deleted)
	assert.True(t, openPlatformAuditEventExists(t, fixture, lockedID))
	assert.False(t, openPlatformAuditEventExists(t, fixture, unlockedID))
}

func TestNormalizeAuditRetentionPolicyRejectsLowerThanBaseline(t *testing.T) {
	_, err := normalizeAuditRetentionPolicy(AuditRetentionPolicy{DisclosureDays: 30})
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot be lower than baseline")

	_, err = normalizeAuditRetentionPolicy(AuditRetentionPolicy{OperationalDays: 365})
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot be lower than baseline")
}

func writeStaleOpenPlatformAuditEvent(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	repo *Repository,
	ctx context.Context,
	requestSuffix string,
	eventType string,
	ageDays int,
) int64 {
	t.Helper()
	requestID := fmt.Sprintf("%s-%s", t.Name(), requestSuffix)
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		EventType: eventType,
		RequestID: requestID,
		Metadata:  map[string]any{"test": t.Name()},
	}))

	var eventID int64
	err := fixture.Pool.QueryRow(ctx, `
		UPDATE open_platform_audit_events
		SET created_at = NOW() - make_interval(days => $2)
		WHERE request_id = $1
		RETURNING id
	`, requestID, ageDays).Scan(&eventID)
	require.NoError(t, err)
	return eventID
}

func openPlatformAuditEventExists(t *testing.T, fixture *postgresfixture.Fixture, eventID int64) bool {
	t.Helper()
	var exists bool
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM open_platform_audit_events
			WHERE id = $1
		)
	`, eventID).Scan(&exists)
	require.NoError(t, err)
	return exists
}
