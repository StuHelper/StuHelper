package db_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestWithTxRollsBackWhenCallerCancelsBeforeCommit(t *testing.T) {
	fixture := postgresfixture.Start(t)
	setupCtx := context.Background()
	_, err := fixture.Pool.Exec(setupCtx, `
		CREATE TABLE tx_context_commit_guard (
			id bigint PRIMARY KEY
		)
	`)
	require.NoError(t, err)

	txCtx, cancel := context.WithCancel(context.Background())
	err = fixture.DB.WithTx(txCtx, func(ctx context.Context, tx pgx.Tx) error {
		if _, execErr := tx.Exec(ctx, `INSERT INTO tx_context_commit_guard (id) VALUES (1)`); execErr != nil {
			return execErr
		}
		cancel()
		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	var count int
	require.NoError(t, fixture.Pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM tx_context_commit_guard`,
	).Scan(&count))
	assert.Zero(t, count, "a canceled transaction must not commit writes")
}
