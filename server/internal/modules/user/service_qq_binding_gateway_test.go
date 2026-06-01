package user

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureQQBindingForUserTx_CreatesBindingWithoutNestedTx(t *testing.T) {
	repo := newQQBindingMockRepo()
	var created *QQBinding

	repo.onWithTx = func(context.Context, func(context.Context, pgx.Tx) error) error {
		t.Fatal("EnsureQQBindingForUserTx must not open a nested transaction")
		return nil
	}
	repo.onCreateQQBindingTx = func(_ context.Context, _ pgx.Tx, binding *QQBinding) error {
		created = binding
		return nil
	}

	svc := newQQGatewayTestService(t, repo)
	binding, err := svc.EnsureQQBindingForUserTx(context.Background(), nil, 77, " 123456789 ")
	require.NoError(t, err)
	require.NotNil(t, binding)
	require.NotNil(t, created)

	assert.Equal(t, int64(77), binding.UserID)
	assert.Equal(t, "123456789", binding.QQID)
	assert.WithinDuration(t, time.Now(), binding.BoundAt, time.Second)
}

func TestEnsureQQBindingForUserTx_ReturnsExistingBinding(t *testing.T) {
	repo := newQQBindingMockRepo()
	existing := &QQBinding{UserID: 77, QQID: "123456789", BoundAt: time.Now()}
	repo.onGetQQBindingByUserIDTx = func(context.Context, pgx.Tx, int64) (*QQBinding, error) {
		return existing, nil
	}

	svc := newQQGatewayTestService(t, repo)
	binding, err := svc.EnsureQQBindingForUserTx(context.Background(), nil, 77, "123456789")
	require.NoError(t, err)
	assert.Same(t, existing, binding)
}

func TestEnsureQQBindingForUserTx_RejectsUserBoundToOtherQQ(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByUserIDTx = func(context.Context, pgx.Tx, int64) (*QQBinding, error) {
		return &QQBinding{UserID: 77, QQID: "other"}, nil
	}

	svc := newQQGatewayTestService(t, repo)
	binding, err := svc.EnsureQQBindingForUserTx(context.Background(), nil, 77, "123456789")
	require.ErrorIs(t, err, ErrQQBindingUserConflict)
	assert.Nil(t, binding)
}

func TestEnsureQQBindingForUserTx_RejectsQQBoundToOtherUser(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByQQIDTx = func(context.Context, pgx.Tx, string) (*QQBinding, error) {
		return &QQBinding{UserID: 88, QQID: "123456789"}, nil
	}

	svc := newQQGatewayTestService(t, repo)
	binding, err := svc.EnsureQQBindingForUserTx(context.Background(), nil, 77, "123456789")
	require.ErrorIs(t, err, ErrQQBindingQQAlreadyBound)
	assert.Nil(t, binding)
}

func newQQGatewayTestService(t *testing.T, repo Repo) *Service {
	t.Helper()
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)
	return svc
}
