package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestLogContextPersistsWithCanceledContextAndTrace(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ConfigureRepository(NewRepository(fixture.DB))
	t.Cleanup(func() { ConfigureRepository(nil) })

	traceID := trace.TraceID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	spanID := trace.SpanID{0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(logger.WithRequestID(context.Background(), "req-canceled-1"), spanContext)
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	LogContext(ctx, Event{
		Type:         EventAdminConfigChange,
		Category:     "admin_operation",
		ActorType:    "admin",
		UserID:       "admin-trace",
		Action:       "update",
		ResourceType: "school_config",
		ResourceID:   "trace-test",
		Result:       "success",
	})

	var requestID string
	var persistedTraceID string
	row := fixture.Pool.QueryRow(context.Background(), `
		SELECT request_id, trace_id
		FROM audit_events
		WHERE actor_user_id = $1 AND resource_id = $2
	`, "admin-trace", "trace-test")
	require.NoError(t, row.Scan(&requestID, &persistedTraceID))
	assert.Equal(t, "req-canceled-1", requestID)
	assert.Equal(t, traceID.String(), persistedTraceID)
}

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
		ResourceID:   "4111010006",
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

func TestRepository_ListAdminOperationsHandlesNullableRequestFields(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	require.NoError(t, repo.WriteEvent(ctx, Event{
		Type:         EventType("member_blacklist.created"),
		Category:     "admin_operation",
		ActorType:    "system",
		Action:       "create",
		ResourceType: "member_blacklist.entry",
		ResourceID:   "entry-1",
		Result:       "success",
	}))

	logs, total, err := repo.ListAdminOperations(ctx, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, logs, 1)
	assert.Equal(t, "", logs[0].ActorUserID)
	assert.Equal(t, "", logs[0].ActorUsername)
	assert.Equal(t, "", logs[0].IPAddress)
	assert.Equal(t, "", logs[0].UserAgent)
	assert.Equal(t, "member_blacklist.entry", logs[0].ResourceType)
	assert.JSONEq(t, `{}`, string(logs[0].BeforeData))
	assert.JSONEq(t, `{}`, string(logs[0].AfterData))
}

func TestRepository_CleanupAdminOperationsPreservesIAMEvents(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	require.NoError(t, repo.WriteEvent(ctx, staleAdminEvent(EventAdminConfigChange)))
	require.NoError(t, repo.WriteEvent(ctx, staleAdminEvent(EventType("iam.role.grant"))))

	deleted, err := repo.CleanupAdminOperations(ctx, 90)
	require.NoError(t, err)

	assert.EqualValues(t, 1, deleted)
	assert.EqualValues(t, 1, countEvents(t, fixture, "event_type LIKE 'iam.%'"))
}

func TestRepository_CleanupAdminOperationsSkipsLockedRows(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	const retentionDays = 90
	lockedID := writeStaleAdminEventForUser(t, fixture, repo, ctx, "locked-admin")
	unlockedID := writeStaleAdminEventForUser(t, fixture, repo, ctx, "unlocked-admin")

	lockTx, err := fixture.Pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, lockTx.Rollback(context.Background()))
	})

	var selectedID string
	err = lockTx.QueryRow(ctx, `SELECT id FROM audit_events WHERE id = $1 FOR UPDATE`, lockedID).Scan(&selectedID)
	require.NoError(t, err)
	require.Equal(t, lockedID, selectedID)

	cleanupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	deleted, err := repo.CleanupAdminOperations(cleanupCtx, retentionDays)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
	assert.True(t, eventExists(t, fixture, lockedID))
	assert.False(t, eventExists(t, fixture, unlockedID))
}

func TestRepository_CleanupIAMEventsUsesTieredRetention(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	require.NoError(t, repo.WriteEvent(ctx, staleAuditEvent(EventUserLogin, 91)))
	require.NoError(t, repo.WriteEvent(ctx, staleAuditEvent(EventUserLoginFailed, 366)))
	require.NoError(t, repo.WriteEvent(ctx, staleAdminEventWithAge(EventType("iam.casdoor_admin_api.call"), 366)))
	require.NoError(t, repo.WriteEvent(ctx, staleAdminEventWithAge(EventType("iam.role.grant"), 400)))
	require.NoError(t, repo.WriteEvent(ctx, staleAdminEventWithAge(EventType("iam.role.revoke"), 1100)))

	deleted, err := repo.CleanupIAMEvents(ctx, IAMRetentionPolicy{})
	require.NoError(t, err)

	assert.EqualValues(t, 4, deleted)
	assert.EqualValues(t, 1, countEvents(t, fixture, "event_type = 'iam.role.grant'"))
}

func staleAuditEvent(eventType EventType, ageDays int) Event {
	return Event{
		Type:      eventType,
		Category:  "audit",
		ActorType: "user",
		UserID:    "user-1",
		Action:    "test",
		Result:    "success",
		Timestamp: time.Now().AddDate(0, 0, -ageDays),
	}
}

func staleAdminEvent(eventType EventType) Event {
	return staleAdminEventWithAge(eventType, 120)
}

func staleAdminEventWithAge(eventType EventType, ageDays int) Event {
	return Event{
		Type:         eventType,
		Category:     "admin_operation",
		ActorType:    "admin",
		UserID:       "admin-1",
		Action:       "test",
		ResourceType: "iam",
		ResourceID:   "resource-1",
		Result:       "success",
		Timestamp:    time.Now().AddDate(0, 0, -ageDays),
	}
}

func writeStaleAdminEventForUser(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	repo *Repository,
	ctx context.Context,
	userID string,
) string {
	t.Helper()
	event := staleAdminEvent(EventAdminConfigChange)
	event.UserID = userID
	require.NoError(t, repo.WriteEvent(ctx, event))

	var eventID string
	row := fixture.Pool.QueryRow(ctx, `
		SELECT id
		FROM audit_events
		WHERE category = $1
		  AND event_type = $2
		  AND actor_user_id = $3
	`, adminOperationCategory, string(EventAdminConfigChange), userID)
	require.NoError(t, row.Scan(&eventID))
	return eventID
}

func eventExists(t *testing.T, fixture *postgresfixture.Fixture, eventID string) bool {
	t.Helper()
	var exists bool
	row := fixture.Pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM audit_events WHERE id = $1)`, eventID)
	require.NoError(t, row.Scan(&exists))
	return exists
}

func countEvents(t *testing.T, fixture *postgresfixture.Fixture, condition string) int64 {
	t.Helper()
	var count int64
	row := fixture.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_events WHERE "+condition)
	require.NoError(t, row.Scan(&count))
	return count
}
