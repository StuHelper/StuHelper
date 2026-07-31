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

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

type fakeProfileFGAClient struct {
	readCalls   []struct{ object, relation string }
	deleteCalls [][]fga.Tuple
	writeCalls  [][]fga.Tuple
	operations  []string
	readTuples  []fga.Tuple
	readByRel   map[string][]fga.Tuple
	readErr     error
	writeErr    error
	deleteErr   error
}

func (f *fakeProfileFGAClient) ReadTuples(_ context.Context, object, relation string) ([]fga.Tuple, error) {
	f.readCalls = append(f.readCalls, struct{ object, relation string }{object: object, relation: relation})
	f.operations = append(f.operations, "read:"+relation)
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
	f.operations = append(f.operations, "write")
	return f.writeErr
}

func (f *fakeProfileFGAClient) DeleteTuples(_ context.Context, tuples []fga.Tuple) error {
	copied := append([]fga.Tuple(nil), tuples...)
	f.deleteCalls = append(f.deleteCalls, copied)
	f.operations = append(f.operations, "delete")
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
		readByRel: map[string][]fga.Tuple{
			"school": {{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}},
		},
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

	err = svc.syncUserProfileProjection(context.Background(), 123)
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

	err = svc.syncUserProfileProjection(context.Background(), 123)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no school ID")
	assert.Empty(t, fgaClient.readCalls)
	assert.Empty(t, fgaClient.writeCalls)
	assert.Empty(t, fgaClient.deleteCalls)
}

func TestSyncUserProfileProjectionRevokesStaleSchoolBeforeWrite(t *testing.T) {
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

	err = svc.syncUserProfileProjection(context.Background(), 123)
	require.ErrorIs(t, err, writeErr)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Equal(t, []fga.Tuple{
		{User: "user:123", Relation: "owner", Object: "user_profile:123"},
		{User: "school:4111010006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
	require.Len(t, fgaClient.deleteCalls, 1)
	assert.Equal(t, []fga.Tuple{
		{User: "school:4111010001", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.deleteCalls[0])
	assert.Equal(t, []string{"read:owner", "read:school", "delete", "write"}, fgaClient.operations)
}

func TestSyncUserProfileProjectionSkipsExistingOwnerTuple(t *testing.T) {
	fgaClient := &fakeProfileFGAClient{
		readByRel: map[string][]fga.Tuple{
			"owner": {{User: "user:123", Relation: "owner", Object: "user_profile:123"}},
		},
	}
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			return &Profile{UserID: userID, VerificationStatus: StatusPending}, nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(fgaClient),
	)
	require.NoError(t, err)

	err = svc.syncUserProfileProjection(context.Background(), 123)
	require.NoError(t, err)
	assert.Empty(t, fgaClient.deleteCalls)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Empty(t, fgaClient.writeCalls[0])
}

func TestSyncUserProfileProjectionDeletesStaleOwnerTuple(t *testing.T) {
	fgaClient := &fakeProfileFGAClient{
		readByRel: map[string][]fga.Tuple{
			"owner": {
				{User: "user:456", Relation: "owner", Object: "user_profile:123"},
				{User: "user:123", Relation: "owner", Object: "user_profile:123"},
			},
		},
	}
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(123), userID)
			return &Profile{UserID: userID, VerificationStatus: StatusPending}, nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(fgaClient),
	)
	require.NoError(t, err)

	err = svc.syncUserProfileProjection(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Empty(t, fgaClient.writeCalls[0])
	require.Len(t, fgaClient.deleteCalls, 1)
	assert.Equal(t, []fga.Tuple{{User: "user:456", Relation: "owner", Object: "user_profile:123"}}, fgaClient.deleteCalls[0])
}

func TestProcessExternalSyncJob_UserProfileProjectionUsesCurrentVerifiedState(t *testing.T) {
	schoolID := int64(4111010006)
	fgaClient := &fakeProfileFGAClient{}
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
	payload, err := json.Marshal(userProfileProjectionPayload{UserID: 123, Approved: false})
	require.NoError(t, err)

	err = svc.processExternalSyncJob(context.Background(), ExternalSyncJob{
		JobType: externalSyncJobTypeUserProfileProjection,
		Payload: payload,
	})
	require.NoError(t, err)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Equal(t, []fga.Tuple{
		{User: "user:123", Relation: "owner", Object: "user_profile:123"},
		{User: "school:4111010006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
	assert.Empty(t, fgaClient.deleteCalls)
}

func TestProcessExternalSyncJob_UserProfileProjectionUsesCurrentRejectedState(t *testing.T) {
	fgaClient := &fakeProfileFGAClient{
		readByRel: map[string][]fga.Tuple{
			"school": {{User: "school:4111010006", Relation: "school", Object: "user_profile:123"}},
		},
	}
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(123), userID)
			return &Profile{UserID: userID, VerificationStatus: StatusRejected}, nil
		},
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(fgaClient),
	)
	require.NoError(t, err)
	payload, err := json.Marshal(userProfileProjectionPayload{UserID: 123, Approved: true})
	require.NoError(t, err)

	err = svc.processExternalSyncJob(context.Background(), ExternalSyncJob{
		JobType: externalSyncJobTypeUserProfileProjection,
		Payload: payload,
	})
	require.NoError(t, err)
	require.Len(t, fgaClient.writeCalls, 1)
	assert.Equal(t, []fga.Tuple{{User: "user:123", Relation: "owner", Object: "user_profile:123"}}, fgaClient.writeCalls[0])
	require.Len(t, fgaClient.deleteCalls, 1)
	assert.Equal(t, []fga.Tuple{{User: "school:4111010006", Relation: "school", Object: "user_profile:123"}}, fgaClient.deleteCalls[0])
}

func TestProcessExternalSyncJob_ProjectsAdmissionVerification(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	schoolID := int64(4111010006)
	svc, err := NewService(
		&mockRepo{
			onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
				require.Equal(t, int64(42), userID)
				return &Profile{
					UserID:             userID,
					SchoolID:           &schoolID,
					VerificationStatus: StatusVerified,
				}, nil
			},
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
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(42), userID)
			return &Profile{
				UserID:             userID,
				SchoolID:           &schoolID,
				VerificationStatus: StatusVerified,
			}, nil
		},
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

func TestProcessExternalSyncJob_AdmissionProjectionSkipsCurrentRejectedProfile(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	svc, err := NewService(
		&mockRepo{
			onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
				require.Equal(t, int64(42), userID)
				return &Profile{UserID: userID, VerificationStatus: StatusRejected}, nil
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
	assert.Empty(t, gateway.calls)
}

func TestProcessExternalSyncJob_AdmissionProjectionSkipsMissingCurrentProfile(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	svc, err := NewService(
		&mockRepo{
			onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
				require.Equal(t, int64(42), userID)
				return nil, nil
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
	assert.Empty(t, gateway.calls)
}

func TestProcessExternalSyncJob_AdmissionProjectionErrorsForVerifiedProfileWithoutSchool(t *testing.T) {
	gateway := &fakeAdmissionProjectionGateway{}
	svc, err := NewService(
		&mockRepo{
			onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
				require.Equal(t, int64(42), userID)
				return &Profile{UserID: userID, VerificationStatus: StatusVerified}, nil
			},
			onGetProfileByUserIDTx: func(_ context.Context, _ pgx.Tx, userID int64) (*Profile, error) {
				require.Equal(t, int64(42), userID)
				return &Profile{UserID: userID, VerificationStatus: StatusVerified}, nil
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

	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no school ID for admission projection")
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
	assert.Equal(t, 3, requeued)
	require.Len(t, enqueued, 3)
	assertProfileProjectionTestJob(t, enqueued[0], 42, true)
	assertAdmissionProjectionTestJob(t, enqueued[1], 42, true)
	assertProfileProjectionTestJob(t, enqueued[2], 43, false)
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
