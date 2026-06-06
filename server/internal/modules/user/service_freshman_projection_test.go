package user

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnqueueFreshmanProvisionalRoleSyncTx(t *testing.T) {
	var enqueued []externalSyncTestJob
	repo := &mockRepo{
		onUpsertExternalSyncJobTx: func(_ context.Context, _ pgx.Tx, jobType, dedupeKey string, payload []byte) error {
			enqueued = append(enqueued, externalSyncTestJob{jobType: jobType, dedupeKey: dedupeKey, payload: payload})
			return nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.EnqueueFreshmanProvisionalRoleSyncTx(context.Background(), nil, 42, true)

	require.NoError(t, err)
	require.Len(t, enqueued, 1)
	assert.Equal(t, externalSyncJobTypeFreshmanProvisionalRole, enqueued[0].jobType)
	assert.Equal(t, freshmanProvisionalRoleSyncKey(42), enqueued[0].dedupeKey)
	assertFreshmanRolePayload(t, enqueued[0].payload, 42, true)
}

func TestProcessExternalSyncJob_SyncsFreshmanProvisionalRole(t *testing.T) {
	var synced externalSyncRoleCall
	svc, err := NewService(
		&mockRepo{
			onGetProfileByUserID: func(context.Context, int64) (*Profile, error) {
				t.Fatal("freshman provisional role sync must not read user profile state")
				return nil, nil
			},
		},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithRoleSyncFunc(func(_ context.Context, userID int64, role string, approved bool) error {
			synced = externalSyncRoleCall{UserID: userID, Role: role, Approved: approved}
			return nil
		}),
	)
	require.NoError(t, err)
	payload := []byte(`{"userID":42,"approved":true}`)

	err = svc.processExternalSyncJob(context.Background(), ExternalSyncJob{
		JobType: externalSyncJobTypeFreshmanProvisionalRole,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.Equal(t, externalSyncRoleCall{UserID: 42, Role: freshmanProvisionalRoleName, Approved: true}, synced)
}

type externalSyncRoleCall struct {
	UserID   int64
	Role     string
	Approved bool
}

func assertFreshmanRolePayload(t *testing.T, payload []byte, userID int64, approved bool) {
	t.Helper()
	var decoded verifiedStudentRoleSyncPayload
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, userID, decoded.UserID)
	assert.Equal(t, freshmanProvisionalRoleName, decoded.Role)
	assert.Equal(t, approved, decoded.Approved)
}
