package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/openplatform"
	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestRunAuditRetentionCleanupUsesIAMPolicy(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := audit.NewRepository(fixture.DB)
	ctx := context.Background()

	require.NoError(t, repo.WriteEvent(ctx, audit.Event{
		Type:      audit.EventUserLogin,
		Category:  "audit",
		ActorType: "user",
		UserID:    "user-1",
		Action:    "login",
		Result:    "success",
		Timestamp: time.Now().AddDate(0, 0, -91),
	}))

	runAuditRetentionCleanup(ctx, repo)

	assert.EqualValues(t, 0, countAuditEvents(t, fixture))
}

func TestRunOpenPlatformAuditRetentionCleanupUsesDefaultPolicy(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := openplatform.NewRepository(fixture.DB)
	ctx := context.Background()

	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO open_platform_audit_events (event_type, metadata, created_at)
		VALUES ($1, '{}'::jsonb, NOW() - make_interval(days => $2))
	`, "open_platform.disclosure.granted", 366)
	require.NoError(t, err)

	runOpenPlatformAuditRetentionCleanup(ctx, repo)

	assert.EqualValues(t, 0, countOpenPlatformAuditEvents(t, fixture))
}

func countAuditEvents(t *testing.T, fixture *postgresfixture.Fixture) int64 {
	t.Helper()
	var count int64
	err := fixture.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM audit_events`).Scan(&count)
	require.NoError(t, err)
	return count
}

func countOpenPlatformAuditEvents(t *testing.T, fixture *postgresfixture.Fixture) int64 {
	t.Helper()
	var count int64
	err := fixture.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM open_platform_audit_events`).Scan(&count)
	require.NoError(t, err)
	return count
}
