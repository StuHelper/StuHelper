package user

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

type fakeProfileFGAClient struct {
	readCalls   []struct{ object, relation string }
	deleteCalls [][]fga.Tuple
	writeCalls  [][]fga.Tuple
	readTuples  []fga.Tuple
	readByRel   map[string][]fga.Tuple
	readErr     error
	writeErr    error
	deleteErr   error
}

func (f *fakeProfileFGAClient) ReadTuples(_ context.Context, object, relation string) ([]fga.Tuple, error) {
	f.readCalls = append(f.readCalls, struct{ object, relation string }{object: object, relation: relation})
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.readByRel != nil {
		return append([]fga.Tuple(nil), f.readByRel[relation]...), nil
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

type fakeAdmissionProjectionGateway struct {
	calls []admissionProjectionCall
	err   error
}

type admissionProjectionCall struct {
	userID   int64
	schoolID int64
	approved bool
}

func (f *fakeAdmissionProjectionGateway) ProjectStudentVerification(
	_ context.Context,
	userID int64,
	schoolID int64,
	approved bool,
) error {
	f.calls = append(f.calls, admissionProjectionCall{userID: userID, schoolID: schoolID, approved: approved})
	return f.err
}

func TestSyncUserProfileProjection_RebuildsOwnerAndCurrentSchool(t *testing.T) {
	schoolID := int64(4111010006)
	fgaClient := &fakeProfileFGAClient{
		readTuples: []fga.Tuple{{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}},
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
	require.Len(t, fgaClient.readCalls, 2)
	require.Len(t, fgaClient.deleteCalls, 1)
	require.Len(t, fgaClient.writeCalls, 1)

	assert.Equal(t, "owner", fgaClient.readCalls[0].relation)
	assert.Equal(t, "school", fgaClient.readCalls[1].relation)
	assert.Equal(t, []fga.Tuple{{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}}, fgaClient.deleteCalls[0])
	assert.Equal(t, []fga.Tuple{
		{User: "user:123", Relation: "owner", Object: "user_profile:123"},
		{User: "school:4111010006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
}

func TestSyncUserProfileProjectionValidatesSchoolBeforeFGAMutation(t *testing.T) {
	fgaClient := &fakeProfileFGAClient{
		readByRel: map[string][]fga.Tuple{
			"school": {{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}},
		},
	}
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(123), userID)
			return &Profile{UserID: userID, VerificationStatus: StatusVerified}, nil
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no school ID")
	assert.Empty(t, fgaClient.readCalls)
	assert.Empty(t, fgaClient.writeCalls)
	assert.Empty(t, fgaClient.deleteCalls)
}

func TestSyncUserProfileProjectionKeepsExistingSchoolTupleWhenWriteFails(t *testing.T) {
	schoolID := int64(4111010006)
	writeErr := errors.New("openfga write unavailable")
	fgaClient := &fakeProfileFGAClient{
		readByRel: map[string][]fga.Tuple{
			"school": {{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}},
		},
		writeErr: writeErr,
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
	require.ErrorIs(t, err, writeErr)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Equal(t, []fga.Tuple{
		{User: "user:123", Relation: "owner", Object: "user_profile:123"},
		{User: "school:4111010006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
	assert.Empty(t, fgaClient.deleteCalls)
}

func TestSyncUserProfileProjectionSkipsExistingOwnerTuple(t *testing.T) {
	schoolID := int64(4111010006)
	fgaClient := &fakeProfileFGAClient{
		readByRel: map[string][]fga.Tuple{
			"owner": {{User: "user:123", Relation: "owner", Object: "user_profile:123"}},
		},
	}
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
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

	err = svc.syncUserProfileProjection(context.Background(), 123, false)
	require.NoError(t, err)
	assert.Empty(t, fgaClient.deleteCalls)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Empty(t, fgaClient.writeCalls[0])
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
		onMarkExternalSyncJobFailure: func(_ context.Context, jobID int64, _ time.Time, nextAttemptAt time.Time, errMsg string, terminal bool) error {
			retryMarked = true
			assert.Equal(t, int64(1), jobID)
			nextRetry = nextAttemptAt
			lastError = errMsg
			assert.False(t, terminal)
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
			return errors.New("casdoor unavailable")
		}),
	)
	require.NoError(t, err)

	startedAt := time.Now()
	err = svc.processExternalSyncBatch(context.Background())
	require.NoError(t, err)
	assert.True(t, retryMarked)
	assert.Contains(t, lastError, "casdoor unavailable")
	assert.True(t, nextRetry.After(startedAt.Add(2*time.Second)))
	assert.True(t, nextRetry.Before(startedAt.Add(6*time.Second)))
}

func TestProcessExternalSyncJob_ProjectsAdmissionVerification(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	schoolID := int64(4111010006)
	svc, err := NewService(
		&mockRepo{
			onGetProfileByUserIDTx: func(_ context.Context, _ pgx.Tx, userID int64) (*Profile, error) {
				require.Equal(t, int64(42), userID)
				return &Profile{
					UserID:             userID,
					SchoolID:           &schoolID,
					VerificationStatus: StatusVerified,
				}, nil
			},
		},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithAdmissionVerificationProjectionGateway(gateway),
	)
	require.NoError(t, err)
	payload, err := json.Marshal(admissionVerificationProjectionPayload{UserID: 42, Approved: true})
	require.NoError(t, err)

	err = svc.processExternalSyncJob(context.Background(), ExternalSyncJob{
		ID:      1,
		JobType: externalSyncJobTypeAdmissionVerification,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.Equal(t, []admissionProjectionCall{{userID: 42, schoolID: schoolID, approved: true}}, gateway.calls)
}

func TestProcessExternalSyncJob_EnsuresSchoolEmailCredentialBeforeAdmissionProjection(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	method := VerifyMethodSchoolEmailOTP
	schoolID := int64(4111010006)
	var capturedCredential *VerificationCredentialProjection
	repo := &mockRepo{
		onGetProfileByUserIDTx: func(_ context.Context, _ pgx.Tx, userID int64) (*Profile, error) {
			require.Equal(t, int64(42), userID)
			return &Profile{
				UserID:             userID,
				SchoolID:           &schoolID,
				VerificationStatus: StatusVerified,
				VerificationMethod: &method,
				ManualFormData:     json.RawMessage(`{"schoolEmail":"20250001@buaa.edu.cn"}`),
			}, nil
		},
		onEnsureVerificationCredentialTx: func(_ context.Context, _ pgx.Tx, credential VerificationCredentialProjection) error {
			copy := credential
			capturedCredential = &copy
			return nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithAdmissionVerificationProjectionGateway(gateway),
	)
	require.NoError(t, err)
	payload, err := json.Marshal(admissionVerificationProjectionPayload{UserID: 42, Approved: true})
	require.NoError(t, err)

	err = svc.processExternalSyncJob(context.Background(), ExternalSyncJob{
		ID:      1,
		JobType: externalSyncJobTypeAdmissionVerification,
		Payload: payload,
	})

	require.NoError(t, err)
	require.NotNil(t, capturedCredential)
	assert.Equal(t, int64(42), capturedCredential.UserID)
	assert.Equal(t, schoolID, capturedCredential.SchoolID)
	assert.Equal(t, userVerificationCredentialKindSchoolEmailOTP, capturedCredential.Kind)
	assert.Equal(t, "2******1@buaa.edu.cn", capturedCredential.SubjectDisplay)
	assert.Equal(t, []admissionProjectionCall{{userID: 42, schoolID: schoolID, approved: true}}, gateway.calls)
}

func TestProcessExternalSyncJob_SkipsAdmissionProjectionWhenUnapproved(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	svc, err := NewService(
		&mockRepo{},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithAdmissionVerificationProjectionGateway(gateway),
	)
	require.NoError(t, err)
	payload, err := json.Marshal(admissionVerificationProjectionPayload{UserID: 42, Approved: false})
	require.NoError(t, err)

	err = svc.processExternalSyncJob(context.Background(), ExternalSyncJob{
		ID:      1,
		JobType: externalSyncJobTypeAdmissionVerification,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.Empty(t, gateway.calls)
}

func TestStartBackgroundJobsRequiresStarter(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Panics(t, func() {
		svc.StartBackgroundJobs(ctx, nil)
	})
}

func TestRunExternalSyncReconciliationRecoversPanic(t *testing.T) {
	repo := &mockRepo{
		onListStudentRoleProjectionStates: func(context.Context, int) ([]StudentRoleProjectionState, error) {
			panic("projection query panic")
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	require.NotPanics(t, func() {
		svc.runExternalSyncReconciliation(context.Background())
	})
}

func TestVerifyStudent_EnqueuesProjectionInsideTransaction(t *testing.T) {
	var enqueued []string
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(4111010006), schoolID)
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

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{SchoolID: 4111010006, Consent: true})
	require.NoError(t, err)
	assert.Contains(t, enqueued, externalSyncJobTypeUserProfileProjection+":"+userProfileProjectionKey(1))
	assert.NotContains(t, enqueued, externalSyncJobTypeAdmissionVerification+":"+admissionVerificationProjectionKey(1))
	assert.NotContains(t, enqueued, externalSyncJobTypeVerifiedStudentRole+":"+verifiedStudentRoleSyncKey(1))
}

func TestVerifyStudent_EnqueuesAdmissionProjectionWhenVerified(t *testing.T) {
	var enqueued []string
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) { return nil, nil },
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(4111010006), schoolID)
			return &SchoolConfig{
				SchoolID:           schoolID,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "auto",
				Enabled:            true,
			}, nil
		},
		onCreateProfileTx: func(_ context.Context, _ pgx.Tx, _ *Profile) error { return nil },
		onUpsertExternalSyncJobTx: func(_ context.Context, _ pgx.Tx, jobType, dedupeKey string, _ []byte) error {
			enqueued = append(enqueued, jobType+":"+dedupeKey)
			return nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{SchoolID: 4111010006, Consent: true})
	require.NoError(t, err)
	assert.Contains(t, enqueued, externalSyncJobTypeUserProfileProjection+":"+userProfileProjectionKey(1))
	assert.Contains(t, enqueued, externalSyncJobTypeAdmissionVerification+":"+admissionVerificationProjectionKey(1))
	assert.Contains(t, enqueued, externalSyncJobTypeVerifiedStudentRole+":"+verifiedStudentRoleSyncKey(1))
}

func TestReconcileUserProfileProjectionsRequeuesWithinLimit(t *testing.T) {
	var enqueued []externalSyncTestJob
	repo := &mockRepo{
		onListStudentRoleProjectionStates: func(_ context.Context, limit int) ([]StudentRoleProjectionState, error) {
			require.Equal(t, 101, limit)
			return []StudentRoleProjectionState{{UserID: 42, Approved: true}, {UserID: 43}}, nil
		},
		onUpsertExternalSyncJobTx: func(_ context.Context, _ pgx.Tx, jobType, dedupeKey string, payload []byte) error {
			enqueued = append(enqueued, externalSyncTestJob{jobType: jobType, dedupeKey: dedupeKey, payload: payload})
			return nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	requeued, err := svc.ReconcileUserProfileProjections(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, 5, requeued)
	require.Len(t, enqueued, 5)
	assertProfileProjectionTestJob(t, enqueued[0], 42, true)
	assertAdmissionProjectionTestJob(t, enqueued[1], 42, true)
	assertRoleSyncTestJob(t, enqueued[2], 42, true)
	assertProfileProjectionTestJob(t, enqueued[3], 43, false)
	assertRoleSyncTestJob(t, enqueued[4], 43, false)
}

func TestReconcileUserProfileProjectionsStopsAboveThreshold(t *testing.T) {
	txCalled := false
	before := testutil.ToFloat64(metrics.IAMDriftReconciliationThresholdExceededTotal.WithLabelValues("user_profile_projection"))
	repo := &mockRepo{
		onListStudentRoleProjectionStates: func(_ context.Context, limit int) ([]StudentRoleProjectionState, error) {
			require.Equal(t, 101, limit)
			return make([]StudentRoleProjectionState, 101), nil
		},
		onWithTx: func(context.Context, func(context.Context, pgx.Tx) error) error {
			txCalled = true
			return nil
		},
	}
	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	requeued, err := svc.ReconcileUserProfileProjections(context.Background(), 100)
	require.ErrorIs(t, err, ErrExternalSyncReconciliationThresholdExceeded)
	assert.Equal(t, 0, requeued)
	assert.False(t, txCalled)
	after := testutil.ToFloat64(metrics.IAMDriftReconciliationThresholdExceededTotal.WithLabelValues("user_profile_projection"))
	assert.Equal(t, before+1, after)
}

func TestNextExternalSyncReconciliationDelay(t *testing.T) {
	location := time.FixedZone("test", 8*60*60)
	cases := []struct {
		name string
		now  time.Time
		want time.Duration
	}{
		{name: "before window", now: time.Date(2026, 5, 2, 2, 30, 0, 0, location), want: 30 * time.Minute},
		{name: "at window", now: time.Date(2026, 5, 2, 3, 0, 0, 0, location), want: 24 * time.Hour},
		{name: "after window", now: time.Date(2026, 5, 2, 4, 0, 0, 0, location), want: 23 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, nextExternalSyncReconciliationDelay(tc.now))
		})
	}
}

type externalSyncTestJob struct {
	jobType   string
	dedupeKey string
	payload   []byte
}

func assertRoleSyncTestJob(t *testing.T, job externalSyncTestJob, userID int64, approved bool) {
	t.Helper()
	assert.Equal(t, externalSyncJobTypeVerifiedStudentRole, job.jobType)
	assert.Equal(t, verifiedStudentRoleSyncKey(userID), job.dedupeKey)
	var payload verifiedStudentRoleSyncPayload
	require.NoError(t, json.Unmarshal(job.payload, &payload))
	assert.Equal(t, userID, payload.UserID)
	assert.Equal(t, verifiedStudentRoleName, payload.Role)
	assert.Equal(t, approved, payload.Approved)
}

func assertProfileProjectionTestJob(t *testing.T, job externalSyncTestJob, userID int64, approved bool) {
	t.Helper()
	assert.Equal(t, externalSyncJobTypeUserProfileProjection, job.jobType)
	assert.Equal(t, userProfileProjectionKey(userID), job.dedupeKey)
	var payload userProfileProjectionPayload
	require.NoError(t, json.Unmarshal(job.payload, &payload))
	assert.Equal(t, userID, payload.UserID)
	assert.Equal(t, approved, payload.Approved)
}

func assertAdmissionProjectionTestJob(t *testing.T, job externalSyncTestJob, userID int64, approved bool) {
	t.Helper()
	assert.Equal(t, externalSyncJobTypeAdmissionVerification, job.jobType)
	assert.Equal(t, admissionVerificationProjectionKey(userID), job.dedupeKey)
	var payload admissionVerificationProjectionPayload
	require.NoError(t, json.Unmarshal(job.payload, &payload))
	assert.Equal(t, userID, payload.UserID)
	assert.Equal(t, approved, payload.Approved)
}
