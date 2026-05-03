package admission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestAdmissionMeShowsProjectionPendingUntilOutboxCompletes(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "projection-me")
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "20001", Token: "projection-me-token",
	})
	_, err := svc.MarkVerified(context.Background(), latestAdmissionSessionID(t, fixture, userID))
	require.NoError(t, err)
	insertFreshmanCredential(t, fixture, freshmanCredentialSeed{
		ID: "projection-me-credential", UserID: userID, ExpiresAt: futureTime(60),
	})
	insertFreshmanProjectionOutbox(t, fixture, userID, "pending")

	me, err := svc.GetAdmissionMe(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, StatusVerified, me.Status)
	assert.True(t, me.ProjectionPending)
	require.NotNil(t, me.Session)
	assert.True(t, me.Session.ProjectionPending)
	require.NotNil(t, me.CredentialKind)
	assert.Equal(t, CredentialFreshmanMaterialManual, *me.CredentialKind)
	require.NotNil(t, me.ProvisionalExpiresAt)

	markFreshmanProjectionOutboxCompleted(t, fixture, userID)
	me, err = svc.GetAdmissionMe(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, me.ProjectionPending)
	assert.False(t, me.Session.ProjectionPending)
}

type freshmanCredentialSeed struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
}

func insertFreshmanCredential(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	seed freshmanCredentialSeed,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display, expires_at
		)
		VALUES ($1, $2, 1, $3, $4, $5, $6)
	`, seed.ID, seed.UserID, CredentialFreshmanMaterialManual, "freshman-hash-"+seed.ID,
		"freshman material A***", seed.ExpiresAt)
	require.NoError(t, err)
}

func insertFreshmanProjectionOutbox(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	status string,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO domain_event_outbox (stream, job_type, dedupe_key, payload, status)
		VALUES ($1, 'freshman_provisional_role', $2, '{}'::jsonb, $3)
	`, outbox.StreamIAMCasdoorRoleSync, freshmanProjectionDedupeKey(userID), status)
	require.NoError(t, err)
}

func markFreshmanProjectionOutboxCompleted(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE domain_event_outbox
		SET status = 'completed', updated_at = NOW()
		WHERE stream = $1 AND dedupe_key = $2
	`, outbox.StreamIAMCasdoorRoleSync, freshmanProjectionDedupeKey(userID))
	require.NoError(t, err)
}

func latestAdmissionSessionID(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
) string {
	t.Helper()
	var id string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT id
		FROM group_admission_sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, userID).Scan(&id)
	require.NoError(t, err)
	return id
}
