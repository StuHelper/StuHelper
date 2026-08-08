package user

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestExternalSyncOutbox_UpsertClaimRetryLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	err := repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repo.UpsertExternalSyncJobTx(
			ctx,
			tx,
			externalSyncJobTypeUserProfileProjection,
			userProfileProjectionKey(42),
			[]byte(`{"userID":42,"approved":true}`),
		)
	})
	require.NoError(t, err)
	assertExternalSyncStream(t, fixture, userProfileProjectionKey(42), outbox.StreamIAMOpenFGATupleSync)

	jobs, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, externalSyncJobTypeUserProfileProjection, jobs[0].JobType)
	assert.JSONEq(t, `{"userID":42,"approved":true}`, string(jobs[0].Payload))
	assert.Equal(t, 0, jobs[0].AttemptCount)

	err = repo.MarkExternalSyncJobRetry(ctx, jobs[0].ID, jobs[0].LockedAt, time.Now().Add(-time.Second), "boom")
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
	require.NoError(t, repo.MarkExternalSyncJobDone(ctx, jobs[0].ID, jobs[0].LockedAt))

	upsert(`{"userID":7,"approved":true}`)
	reclaimed, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
	assert.JSONEq(t, `{"userID":7,"approved":true}`, string(reclaimed[0].Payload))
}

func TestExternalSyncOutbox_ClaimsOnlyTargetProfileProjectionJobs(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	for userID := int64(1); userID <= 4; userID++ {
		upsertExternalSyncJob(t, repo, ctx, externalSyncJobTypeUserProfileProjection, userProfileProjectionKey(userID))
	}
	upsertForeignOpenFGAJob(t, fixture, ctx)

	jobs, err := repo.ClaimExternalSyncJobs(ctx, 6, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 4)

	counts := map[string]int{}
	for _, job := range jobs {
		counts[job.JobType]++
	}
	assert.Equal(t, 4, counts[externalSyncJobTypeUserProfileProjection])
	assertExternalSyncJobStatus(t, fixture, "review-relations:foreign", "pending")
}

func TestListStudentRoleProjectionStates(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	verifiedID := insertExternalSyncUser(t, fixture, "verified")
	rejectedID := insertExternalSyncUser(t, fixture, "rejected")
	insertExternalSyncEligibilityRevision(t, fixture, verifiedID)
	insertExternalSyncEligibilityRevision(t, fixture, rejectedID)
	insertExternalSyncTargetCredential(t, fixture, verifiedID)

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

func upsertForeignOpenFGAJob(t *testing.T, fixture *postgresfixture.Fixture, ctx context.Context) {
	t.Helper()
	err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return outbox.UpsertJobTx(
			ctx,
			tx,
			outbox.StreamIAMOpenFGATupleSync,
			"review_relations",
			"review-relations:foreign",
			[]byte(`{"reviewID":"foreign","authorUserID":"u-1","schoolID":4111010006}`),
		)
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

func assertExternalSyncJobStatus(t *testing.T, fixture *postgresfixture.Fixture, dedupeKey, want string) {
	t.Helper()
	var got string
	err := fixture.Pool.QueryRow(context.Background(), `SELECT status FROM domain_event_outbox WHERE dedupe_key = $1`, dedupeKey).Scan(&got)
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

func insertExternalSyncEligibilityRevision(t *testing.T, fixture *postgresfixture.Fixture, userID int64) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO student_eligibility_revisions (user_id, school_id, revision, reason_code)
		VALUES ($1, 4111010006, 1, 'test_projection_candidate')
	`, userID)
	require.NoError(t, err)
}

func insertExternalSyncTargetCredential(t *testing.T, fixture *postgresfixture.Fixture, userID int64) {
	t.Helper()
	applicationID := "00000000-0000-4000-8000-000000000001"
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO student_verification_applications (
		    id, user_id, school_id, status, current_method,
		    privacy_notice_version, consented_at, expires_at, completed_at
		)
		VALUES ($1, $2, 4111010006, 'approved', 'school_sso',
		    'privacy-v1', NOW(), NOW() + INTERVAL '1 hour', NOW())
	`, applicationID, userID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_verification_credentials (
		    id, user_id, school_id, kind, subject_hash, subject_display,
		    verification_application_id, status, credential_class, roster_dependency, assurance,
		    verified_at, activated_at
		)
		VALUES (
		    '00000000-0000-4000-8000-000000000001', $1, 4111010006,
		    'school_sso', repeat('a', 64), '20****01',
		    $2, 'active', 'formal_student', 'independent', 'standard', NOW(), NOW()
		)
	`, userID, applicationID)
	require.NoError(t, err)
}
