package user

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
)

type fakeProfileFGAClient struct {
	readCalls   []struct{ object, relation string }
	deleteCalls [][]fga.Tuple
	writeCalls  [][]fga.Tuple
	readTuples  []fga.Tuple
	readErr     error
	writeErr    error
	deleteErr   error
}

func (f *fakeProfileFGAClient) ReadTuples(_ context.Context, object, relation string) ([]fga.Tuple, error) {
	f.readCalls = append(f.readCalls, struct{ object, relation string }{object: object, relation: relation})
	if f.readErr != nil {
		return nil, f.readErr
	}
	return append([]fga.Tuple(nil), f.readTuples...), nil
}

func (f *fakeProfileFGAClient) WriteTuples(_ context.Context, tuples []fga.Tuple) error {
	copied := append([]fga.Tuple(nil), tuples...)
	f.writeCalls = append(f.writeCalls, copied)
	return f.writeErr
}

func (f *fakeProfileFGAClient) DeleteTuples(_ context.Context, tuples []fga.Tuple) error {
	copied := append([]fga.Tuple(nil), tuples...)
	f.deleteCalls = append(f.deleteCalls, copied)
	return f.deleteErr
}

func TestSyncUserProfileProjection_RebuildsOwnerAndCurrentSchool(t *testing.T) {
	schoolID := int64(10006)
	fgaClient := &fakeProfileFGAClient{
		readTuples: []fga.Tuple{{User: "school:99999", Relation: "school", Object: "user_profile:123"}},
	}
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(123), userID)
			return &Profile{UserID: userID, SchoolID: &schoolID, VerificationStatus: StatusVerified}, nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(fgaClient),
	)
	require.NoError(t, err)

	err = svc.syncUserProfileProjection(context.Background(), 123, true)
	require.NoError(t, err)
	require.Len(t, fgaClient.readCalls, 1)
	require.Len(t, fgaClient.deleteCalls, 1)
	require.Len(t, fgaClient.writeCalls, 1)

	assert.Equal(t, []fga.Tuple{{User: "school:99999", Relation: "school", Object: "user_profile:123"}}, fgaClient.deleteCalls[0])
	assert.Equal(t, []fga.Tuple{
		{User: "user:test-external-id", Relation: "owner", Object: "user_profile:123"},
		{User: "school:10006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
}

func TestProcessExternalSyncJob_RetryOnRoleSyncFailure(t *testing.T) {
	retryMarked := false
	nextRetry := time.Time{}
	var lastError string
	repo := &mockRepo{
		onClaimExternalSyncJobs: func(_ context.Context, limit int, _ time.Duration) ([]ExternalSyncJob, error) {
			require.Equal(t, externalSyncBatchSize, limit)
			payload, err := json.Marshal(verifiedStudentRoleSyncPayload{UserID: 42, Role: verifiedStudentRoleName, Approved: true})
			require.NoError(t, err)
			return []ExternalSyncJob{{ID: 1, JobType: externalSyncJobTypeVerifiedStudentRole, Payload: payload}}, nil
		},
		onMarkExternalSyncJobRetry: func(_ context.Context, jobID int64, nextAttemptAt time.Time, errMsg string) error {
			retryMarked = true
			assert.Equal(t, int64(1), jobID)
			nextRetry = nextAttemptAt
			lastError = errMsg
			return nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithRoleSyncFunc(func(_ context.Context, userID int64, role string, approved bool) error {
			assert.Equal(t, int64(42), userID)
			assert.Equal(t, verifiedStudentRoleName, role)
			assert.True(t, approved)
			return errors.New("zitadel unavailable")
		}),
	)
	require.NoError(t, err)

	err = svc.processExternalSyncBatch(context.Background())
	require.NoError(t, err)
	assert.True(t, retryMarked)
	assert.Contains(t, lastError, "zitadel unavailable")
	assert.True(t, nextRetry.After(time.Now().Add(4*time.Second)))
}

func TestVerifyStudent_EnqueuesProjectionInsideTransaction(t *testing.T) {
	var enqueued []string
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(10006), schoolID)
			return &SchoolConfig{SchoolID: schoolID, SchoolName: "北航", VerificationMethod: VerifyMethodManual, ApprovalPolicy: "manual", Enabled: true}, nil
		},
		onCreateProfileTx:    func(_ context.Context, _ pgx.Tx, _ *Profile) error { return nil },
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) { return &Profile{UserID: 1}, nil },
		onUpsertExternalSyncJobTx: func(_ context.Context, _ pgx.Tx, jobType, dedupeKey string, _ []byte) error {
			enqueued = append(enqueued, jobType+":"+dedupeKey)
			return nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{SchoolID: 10006, Consent: true})
	require.NoError(t, err)
	assert.Contains(t, enqueued, externalSyncJobTypeUserProfileProjection+":"+userProfileProjectionKey(1))
	assert.NotContains(t, enqueued, externalSyncJobTypeVerifiedStudentRole+":"+verifiedStudentRoleSyncKey(1))
}
