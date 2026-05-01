package user

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeMFARecoveryCode(t *testing.T) {
	assert.Equal(t, "ABCDEFGH", normalizeMFARecoveryCode(" abcd-efgh "))
}

func TestGenerateMFARecoveryCodeFormat(t *testing.T) {
	code, err := generateMFARecoveryCode()
	require.NoError(t, err)

	compact := strings.ReplaceAll(code, "-", "")
	assert.Len(t, compact, 32)
	assert.Equal(t, strings.ToUpper(compact), compact)
	assert.NotContains(t, compact, "=")
}

func TestMFARecoveryHashNormalizesCode(t *testing.T) {
	manager, err := NewMFARecoveryManager(&mfaRecoveryFakeRepo{}, []byte("test-mfa-recovery-hmac-material-32!"))
	require.NoError(t, err)

	left, err := manager.hashRecoveryCode(42, "ABCD-EFGH")
	require.NoError(t, err)
	right, err := manager.hashRecoveryCode(42, " abcd efgh ")
	require.NoError(t, err)

	assert.Equal(t, left, right)
}

type mfaRecoveryFakeRepo struct{}

func (m *mfaRecoveryFakeRepo) WithTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	return fn(ctx, nil)
}

func (m *mfaRecoveryFakeRepo) ReplaceMFARecoveryCodesTx(context.Context, pgx.Tx, MFARecoveryCodeReplace) error {
	return nil
}

func (m *mfaRecoveryFakeRepo) ConsumeMFARecoveryCodeTx(context.Context, pgx.Tx, MFARecoveryCodeConsume) (bool, error) {
	return true, nil
}
