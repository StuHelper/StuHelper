package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
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

func countAuditEvents(t *testing.T, fixture *postgresfixture.Fixture) int64 {
	t.Helper()
	var count int64
	err := fixture.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM audit_events`).Scan(&count)
	require.NoError(t, err)
	return count
}
