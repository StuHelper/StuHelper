package user

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type qqBindingMockRepo struct {
	*mockRepo
	onGetQQBindingByUserID        func(ctx context.Context, userID int64) (*QQBinding, error)
	onGetQQBindingByQQID          func(ctx context.Context, qqID string) (*QQBinding, error)
	onUpsertQQBindingCode         func(ctx context.Context, code *QQBindingCode) error
	onGetQQBindingCodeByHashTx    func(ctx context.Context, tx pgx.Tx, codeHash string) (*QQBindingCode, error)
	onGetQQBindingByUserIDTx      func(ctx context.Context, tx pgx.Tx, userID int64) (*QQBinding, error)
	onGetQQBindingByQQIDTx        func(ctx context.Context, tx pgx.Tx, qqID string) (*QQBinding, error)
	onCreateQQBindingTx           func(ctx context.Context, tx pgx.Tx, binding *QQBinding) error
	onMarkQQBindingCodeConsumedTx func(ctx context.Context, tx pgx.Tx, userID int64, consumedAt time.Time) error
}

func newQQBindingMockRepo() *qqBindingMockRepo {
	return &qqBindingMockRepo{mockRepo: &mockRepo{}}
}

func (m *mockRepo) GetQQBindingByUserID(context.Context, int64) (*QQBinding, error) {
	return nil, nil
}

func (m *mockRepo) GetQQBindingByQQID(context.Context, string) (*QQBinding, error) {
	return nil, nil
}

func (m *mockRepo) UpsertQQBindingCode(context.Context, *QQBindingCode) error {
	return nil
}

func (m *mockRepo) GetQQBindingCodeByHashTx(context.Context, pgx.Tx, string) (*QQBindingCode, error) {
	return nil, nil
}

func (m *mockRepo) GetQQBindingByUserIDTx(context.Context, pgx.Tx, int64) (*QQBinding, error) {
	return nil, nil
}

func (m *mockRepo) GetQQBindingByQQIDTx(context.Context, pgx.Tx, string) (*QQBinding, error) {
	return nil, nil
}

func (m *mockRepo) CreateQQBindingTx(context.Context, pgx.Tx, *QQBinding) error {
	return nil
}

func (m *mockRepo) MarkQQBindingCodeConsumedTx(context.Context, pgx.Tx, int64, time.Time) error {
	return nil
}

func (m *qqBindingMockRepo) GetQQBindingByUserID(ctx context.Context, userID int64) (*QQBinding, error) {
	if m.onGetQQBindingByUserID != nil {
		return m.onGetQQBindingByUserID(ctx, userID)
	}
	return nil, nil
}

func (m *qqBindingMockRepo) GetQQBindingByQQID(ctx context.Context, qqID string) (*QQBinding, error) {
	if m.onGetQQBindingByQQID != nil {
		return m.onGetQQBindingByQQID(ctx, qqID)
	}
	return nil, nil
}

func (m *qqBindingMockRepo) UpsertQQBindingCode(ctx context.Context, code *QQBindingCode) error {
	if m.onUpsertQQBindingCode != nil {
		return m.onUpsertQQBindingCode(ctx, code)
	}
	return nil
}

func (m *qqBindingMockRepo) GetQQBindingCodeByHashTx(ctx context.Context, tx pgx.Tx, codeHash string) (*QQBindingCode, error) {
	if m.onGetQQBindingCodeByHashTx != nil {
		return m.onGetQQBindingCodeByHashTx(ctx, tx, codeHash)
	}
	return nil, nil
}

func (m *qqBindingMockRepo) GetQQBindingByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*QQBinding, error) {
	if m.onGetQQBindingByUserIDTx != nil {
		return m.onGetQQBindingByUserIDTx(ctx, tx, userID)
	}
	return nil, nil
}

func (m *qqBindingMockRepo) GetQQBindingByQQIDTx(ctx context.Context, tx pgx.Tx, qqID string) (*QQBinding, error) {
	if m.onGetQQBindingByQQIDTx != nil {
		return m.onGetQQBindingByQQIDTx(ctx, tx, qqID)
	}
	return nil, nil
}

func (m *qqBindingMockRepo) CreateQQBindingTx(ctx context.Context, tx pgx.Tx, binding *QQBinding) error {
	if m.onCreateQQBindingTx != nil {
		return m.onCreateQQBindingTx(ctx, tx, binding)
	}
	return nil
}

func (m *qqBindingMockRepo) MarkQQBindingCodeConsumedTx(ctx context.Context, tx pgx.Tx, userID int64, consumedAt time.Time) error {
	if m.onMarkQQBindingCodeConsumedTx != nil {
		return m.onMarkQQBindingCodeConsumedTx(ctx, tx, userID, consumedAt)
	}
	return nil
}

func TestGenerateQQBindingCode_ReturnsPlainCodeAndStoresHashedValue(t *testing.T) {
	repo := newQQBindingMockRepo()
	var stored *QQBindingCode
	repo.onUpsertQQBindingCode = func(_ context.Context, code *QQBindingCode) error {
		stored = code
		return nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	before := time.Now()
	result, err := svc.GenerateQQBindingCode(context.Background(), 42, 10*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, stored)

	assert.NotEmpty(t, result.Code)
	assert.NotEqual(t, result.Code, stored.CodeHash)
	assert.Equal(t, int64(42), stored.UserID)
	assert.Equal(t, stored.CodeHash, svc.hashQQBindingCode(result.Code))
	assert.True(t, stored.ExpiresAt.After(before))
	assert.WithinDuration(t, stored.ExpiresAt, result.ExpiresAt, time.Second)
	assert.Nil(t, stored.ConsumedAt)
}

func TestGenerateQQBindingCode_RejectsAlreadyBoundUser(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByUserID = func(_ context.Context, userID int64) (*QQBinding, error) {
		return &QQBinding{UserID: userID, QQID: "123456"}, nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := svc.GenerateQQBindingCode(context.Background(), 42, 10*time.Minute)
	require.ErrorIs(t, err, ErrQQBindingAlreadyExists)
	assert.Nil(t, result)
}

func TestGenerateQQBindingCode_RejectsNonPositiveTTLBeforeRepo(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByUserID = func(context.Context, int64) (*QQBinding, error) {
		t.Fatal("GenerateQQBindingCode must reject non-positive ttl before checking existing bindings")
		return nil, nil
	}
	repo.onUpsertQQBindingCode = func(context.Context, *QQBindingCode) error {
		t.Fatal("GenerateQQBindingCode must reject non-positive ttl before storing a binding code")
		return nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	for _, ttl := range []time.Duration{0, -time.Second} {
		result, err := svc.GenerateQQBindingCode(context.Background(), 42, ttl)
		require.ErrorIs(t, err, ErrQQBindingCodeTTLInvalid)
		assert.Nil(t, result)
	}
}

func TestConsumeQQBindingCode_BindsQQAndMarksCodeConsumed(t *testing.T) {
	repo := newQQBindingMockRepo()
	code := "ABCD1234"
	now := time.Now()

	var createdBinding *QQBinding
	var consumedUserID int64
	var consumedAt time.Time

	repo.onWithTx = func(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
		return fn(ctx, nil)
	}
	repo.onGetQQBindingCodeByHashTx = func(_ context.Context, _ pgx.Tx, codeHash string) (*QQBindingCode, error) {
		assert.Equal(t, svcHashQQBindingCode(t, code), codeHash)
		return &QQBindingCode{
			UserID:    77,
			CodeHash:  codeHash,
			ExpiresAt: now.Add(5 * time.Minute),
		}, nil
	}
	repo.onCreateQQBindingTx = func(_ context.Context, _ pgx.Tx, binding *QQBinding) error {
		createdBinding = binding
		return nil
	}
	repo.onMarkQQBindingCodeConsumedTx = func(_ context.Context, _ pgx.Tx, userID int64, at time.Time) error {
		consumedUserID = userID
		consumedAt = at
		return nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := svc.ConsumeQQBindingCode(context.Background(), code, "123456789")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, createdBinding)

	assert.Equal(t, int64(77), result.UserID)
	assert.Equal(t, "123456789", result.QQID)
	assert.Equal(t, int64(77), consumedUserID)
	assert.WithinDuration(t, time.Now(), consumedAt, time.Second)
	assert.Equal(t, createdBinding.QQID, result.QQID)
}

func TestConsumeQQBindingCode_NormalizesCodeAndQQID(t *testing.T) {
	repo := newQQBindingMockRepo()
	code := "ABCD1234"
	now := time.Now()

	var seenQQID string
	repo.onWithTx = func(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
		return fn(ctx, nil)
	}
	repo.onGetQQBindingCodeByHashTx = func(_ context.Context, _ pgx.Tx, codeHash string) (*QQBindingCode, error) {
		assert.Equal(t, svcHashQQBindingCode(t, code), codeHash)
		return &QQBindingCode{
			UserID:    77,
			CodeHash:  codeHash,
			ExpiresAt: now.Add(5 * time.Minute),
		}, nil
	}
	repo.onGetQQBindingByQQIDTx = func(_ context.Context, _ pgx.Tx, qqID string) (*QQBinding, error) {
		seenQQID = qqID
		return nil, nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := svc.ConsumeQQBindingCode(context.Background(), " "+strings.ToLower(code)+" ", " 123456789 ")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "123456789", seenQQID)
	assert.Equal(t, "123456789", result.QQID)
}

func TestConsumeQQBindingCode_ReturnsConflictWhenQQBelongsToAnotherUser(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onWithTx = func(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
		return fn(ctx, nil)
	}
	repo.onGetQQBindingCodeByHashTx = func(_ context.Context, _ pgx.Tx, _ string) (*QQBindingCode, error) {
		return &QQBindingCode{
			UserID:    77,
			CodeHash:  "hash",
			ExpiresAt: time.Now().Add(time.Minute),
		}, nil
	}
	repo.onGetQQBindingByQQIDTx = func(_ context.Context, _ pgx.Tx, qqID string) (*QQBinding, error) {
		return &QQBinding{UserID: 88, QQID: qqID}, nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := svc.ConsumeQQBindingCode(context.Background(), "ABCD1234", "123456789")
	require.ErrorIs(t, err, ErrQQBindingQQAlreadyBound)
	assert.Nil(t, result)
}

func TestConsumeQQBindingCode_ReturnsExpiredForTimedOutCode(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onWithTx = func(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
		return fn(ctx, nil)
	}
	repo.onGetQQBindingCodeByHashTx = func(_ context.Context, _ pgx.Tx, _ string) (*QQBindingCode, error) {
		return &QQBindingCode{
			UserID:    77,
			CodeHash:  "hash",
			ExpiresAt: time.Now().Add(-time.Minute),
		}, nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := svc.ConsumeQQBindingCode(context.Background(), "ABCD1234", "123456789")
	require.ErrorIs(t, err, ErrQQBindingCodeExpired)
	assert.Nil(t, result)
}

func TestGetQQVerificationStateByQQID_ReturnsUnboundWhenMissingBinding(t *testing.T) {
	repo := newQQBindingMockRepo()
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	state, err := svc.GetQQVerificationStateByQQID(context.Background(), "123456789")
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, QQVerificationStateUnbound, state.VerificationState)
	assert.False(t, state.StudentVerified)
	assert.Nil(t, state.UserID)
}

func TestGetQQVerificationStateByQQID_ReturnsVerifiedForVerifiedProfile(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByQQID = func(_ context.Context, qqID string) (*QQBinding, error) {
		return &QQBinding{UserID: 77, QQID: qqID}, nil
	}
	repo.onGetProfileByUserID = func(_ context.Context, userID int64) (*Profile, error) {
		return &Profile{UserID: userID, VerificationStatus: StatusVerified}, nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	state, err := svc.GetQQVerificationStateByQQID(context.Background(), "123456789")
	require.NoError(t, err)
	require.NotNil(t, state)

	require.NotNil(t, state.UserID)
	assert.Equal(t, int64(77), *state.UserID)
	assert.Equal(t, QQVerificationStateVerified, state.VerificationState)
	assert.True(t, state.StudentVerified)
	assert.Equal(t, StatusVerified, state.ProfileVerificationStatus)
}

func TestGetQQVerificationStateByQQID_ReturnsBoundUnverifiedForPendingProfile(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByQQID = func(_ context.Context, qqID string) (*QQBinding, error) {
		return &QQBinding{UserID: 77, QQID: qqID}, nil
	}
	repo.onGetProfileByUserID = func(_ context.Context, userID int64) (*Profile, error) {
		return &Profile{UserID: userID, VerificationStatus: StatusPending}, nil
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	state, err := svc.GetQQVerificationStateByQQID(context.Background(), "123456789")
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, QQVerificationStateBoundUnverified, state.VerificationState)
	assert.False(t, state.StudentVerified)
	assert.Equal(t, StatusPending, state.ProfileVerificationStatus)
}

func TestConsumeQQBindingCode_PropagatesTxError(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onWithTx = func(_ context.Context, _ func(ctx context.Context, tx pgx.Tx) error) error {
		return errors.New("tx failed")
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := svc.ConsumeQQBindingCode(context.Background(), "ABCD1234", "123456789")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "tx failed")
}

func svcHashQQBindingCode(t *testing.T, code string) string {
	t.Helper()

	repo := newQQBindingMockRepo()
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)
	return svc.hashQQBindingCode(code)
}
