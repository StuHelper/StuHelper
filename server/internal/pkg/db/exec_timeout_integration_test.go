package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestExecWithTimeoutOverridesDefaultAndPreservesCallerCancellation(t *testing.T) {
	fixture := postgresfixture.Start(t)
	pool, err := pgxpool.New(context.Background(), fixture.URL)
	require.NoError(t, err)
	database := db.NewDB(pool, 20*time.Millisecond)
	t.Cleanup(database.Close)

	start := time.Now()
	_, err = database.ExecWithTimeout(
		db.WithTableHint(context.Background(), "timeout_probe"),
		250*time.Millisecond,
		`SELECT pg_sleep(0.06)`,
	)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond,
		"the ordinary 20ms DB timeout must not truncate the explicit budget")

	_, err = database.ExecWithTimeout(
		db.WithTableHint(context.Background(), "timeout_probe"),
		25*time.Millisecond,
		`SELECT pg_sleep(0.2)`,
	)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = database.ExecWithTimeout(
		db.WithTableHint(canceledCtx, "timeout_probe"),
		250*time.Millisecond,
		`SELECT pg_sleep(0.2)`,
	)
	require.ErrorIs(t, err, context.Canceled)
}
