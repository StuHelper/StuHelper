package user

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestRepositoryGetQQBindingCodeByHashTxLocksCodeRow(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	userID := insertQQBindingCodeLockUser(t, fixture)
	codeHash := "qq-binding-code-lock-hash"
	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO user_qq_binding_codes (user_id, code_hash, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '10 minutes')
	`, userID, codeHash)
	require.NoError(t, err)

	tx1, err := fixture.Pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { rollbackTxIfOpen(t, tx1) }()

	code, err := repo.GetQQBindingCodeByHashTx(ctx, tx1, codeHash)
	require.NoError(t, err)
	require.NotNil(t, code)

	tx2, err := fixture.Pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { rollbackTxIfOpen(t, tx2) }()

	_, err = tx2.Exec(ctx, `SET LOCAL lock_timeout = '150ms'`)
	require.NoError(t, err)
	_, err = repo.GetQQBindingCodeByHashTx(ctx, tx2, codeHash)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "55P03", pgErr.Code)

	require.NoError(t, tx1.Rollback(ctx))
	tx1 = nil

	tx3, err := fixture.Pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { rollbackTxIfOpen(t, tx3) }()

	code, err = repo.GetQQBindingCodeByHashTx(ctx, tx3, codeHash)
	require.NoError(t, err)
	require.NotNil(t, code)
	assert.Equal(t, userID, code.UserID)
}

func insertQQBindingCodeLockUser(t *testing.T, fixture *postgresfixture.Fixture) int64 {
	t.Helper()
	var userID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ('qq-binding-code-lock-subject', 'qq-binding-code-lock-user', 'qq-binding-code-lock@example.test')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)
	return userID
}

func rollbackTxIfOpen(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if tx == nil {
		return
	}
	err := tx.Rollback(context.Background())
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		require.NoError(t, err)
	}
}
