package user

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestExternalSyncOutbox_UpsertClaimRetryLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	err := repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repo.UpsertExternalSyncJobTx(
			ctx,
			tx,
			externalSyncJobTypeVerifiedStudentRole,
			verifiedStudentRoleSyncKey(42),
			[]byte(`{"userID":42,"approved":true}`),
		)
	})
	require.NoError(t, err)
	assertExternalSyncStream(t, fixture, verifiedStudentRoleSyncKey(42), outbox.StreamIAMCasdoorRoleSync)

	jobs, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, externalSyncJobTypeVerifiedStudentRole, jobs[0].JobType)
	assert.JSONEq(t, `{"userID":42,"approved":true}`, string(jobs[0].Payload))
	assert.Equal(t, 0, jobs[0].AttemptCount)

	err = repo.MarkExternalSyncJobRetry(ctx, jobs[0].ID, time.Now().Add(-time.Second), "boom")
	require.NoError(t, err)

	reclaimed, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 1, reclaimed[0].AttemptCount)
}

func TestExternalSyncOutbox_UpsertConflictResetsCompletedJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()
	dedupeKey := userProfileProjectionKey(7)

	upsert := func(payload string) {
		t.Helper()
		err := repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			return repo.UpsertExternalSyncJobTx(
				ctx,
				tx,
				externalSyncJobTypeUserProfileProjection,
				dedupeKey,
				[]byte(payload),
			)
		})
		require.NoError(t, err)
	}

	upsert(`{"userID":7,"approved":false}`)
	assertExternalSyncStream(t, fixture, dedupeKey, outbox.StreamIAMOpenFGATupleSync)
	jobs, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.NoError(t, repo.MarkExternalSyncJobDone(ctx, jobs[0].ID))

	upsert(`{"userID":7,"approved":true}`)
	reclaimed, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
	assert.JSONEq(t, `{"userID":7,"approved":true}`, string(reclaimed[0].Payload))
}

func TestExternalSyncOutbox_ClaimSplitsAcrossIAMStreams(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	for userID := int64(1); userID <= 4; userID++ {
		upsertExternalSyncJob(t, repo, ctx, externalSyncJobTypeVerifiedStudentRole, verifiedStudentRoleSyncKey(userID))
		upsertExternalSyncJob(t, repo, ctx, externalSyncJobTypeUserProfileProjection, userProfileProjectionKey(userID))
	}

	jobs, err := repo.ClaimExternalSyncJobs(ctx, 4, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 4)

	counts := map[string]int{}
	for _, job := range jobs {
		counts[job.JobType]++
	}
	assert.Equal(t, 2, counts[externalSyncJobTypeVerifiedStudentRole])
	assert.Equal(t, 2, counts[externalSyncJobTypeUserProfileProjection])
}

func TestListStudentRoleProjectionStates(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	verifiedID := insertExternalSyncUser(t, fixture, "verified")
	rejectedID := insertExternalSyncUser(t, fixture, "rejected")
	insertExternalSyncProfile(t, fixture, verifiedID, StatusVerified)
	insertExternalSyncProfile(t, fixture, rejectedID, StatusRejected)

	states, err := repo.ListStudentRoleProjectionStates(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, []StudentRoleProjectionState{
		{UserID: verifiedID, Approved: true},
		{UserID: rejectedID, Approved: false},
	}, states)

	limited, err := repo.ListStudentRoleProjectionStates(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, []StudentRoleProjectionState{{UserID: verifiedID, Approved: true}}, limited)
}

func upsertExternalSyncJob(t *testing.T, repo *Repository, ctx context.Context, jobType, dedupeKey string) {
	t.Helper()
	err := repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repo.UpsertExternalSyncJobTx(ctx, tx, jobType, dedupeKey, []byte(`{"userID":1,"approved":true}`))
	})
	require.NoError(t, err)
}

func assertExternalSyncStream(t *testing.T, fixture *postgresfixture.Fixture, dedupeKey, want string) {
	t.Helper()
	var got string
	err := fixture.Pool.QueryRow(context.Background(), `SELECT stream FROM domain_event_outbox WHERE dedupe_key = $1`, dedupeKey).Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func insertExternalSyncUser(t *testing.T, fixture *postgresfixture.Fixture, suffix string) int64 {
	t.Helper()
	var userID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "sync-"+suffix, "sync-"+suffix, "sync-"+suffix+"@example.test").Scan(&userID)
	require.NoError(t, err)
	return userID
}

func insertExternalSyncProfile(t *testing.T, fixture *postgresfixture.Fixture, userID int64, status string) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_profiles (user_id, verification_status)
		VALUES ($1, $2)
	`, userID, status)
	require.NoError(t, err)
}
