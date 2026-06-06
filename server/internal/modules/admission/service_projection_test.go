package admission

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

	me, err := svc.GetAdmissionMe(context.Background(), userID, "")
	require.NoError(t, err)
	assert.Equal(t, StatusVerified, me.Status)
	assert.True(t, me.ProjectionPending)
	require.NotNil(t, me.Session)
	assert.True(t, me.Session.ProjectionPending)
	require.NotNil(t, me.CredentialKind)
	assert.Equal(t, CredentialFreshmanMaterialManual, *me.CredentialKind)
	require.NotNil(t, me.ProvisionalExpiresAt)

	markFreshmanProjectionOutboxCompleted(t, fixture, userID)
	me, err = svc.GetAdmissionMe(context.Background(), userID, "")
	require.NoError(t, err)
	assert.False(t, me.ProjectionPending)
	assert.False(t, me.Session.ProjectionPending)
}

func TestAdmissionMeTracksVerifiedProfileProjectionJobs(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "projection-verified-profile")
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "20002", Token: "projection-verified-profile-token",
	})
	expiresAt := futureTime(60)
	credential := VerificationCredential{
		UserID:         userID,
		SchoolID:       4111010006,
		Kind:           CredentialSchoolEmailOTP,
		SubjectHash:    "school-email-hash",
		SubjectDisplay: "s*****t@buaa.edu.cn",
		Subject:        "student@buaa.edu.cn",
		StudentID:      "20260001",
		StudentName:    "投影测试",
		ExpiresAt:      &expiresAt,
		VerifiedAt:     fixedAdmissionNow(),
	}
	err := svc.repo.WithTx(context.Background(), func(ctx context.Context, tx pgx.Tx) error {
		if err := svc.repo.CreateVerificationCredentialTx(ctx, tx, credential); err != nil {
			return err
		}
		return svc.repo.ProjectVerifiedUserProfileTx(ctx, tx, credential)
	})
	require.NoError(t, err)

	me, err := svc.GetAdmissionMe(context.Background(), userID, "")

	require.NoError(t, err)
	assert.True(t, me.ProjectionPending)
	require.NotNil(t, me.Session)
	assert.True(t, me.Session.ProjectionPending)
	require.NotNil(t, me.CredentialKind)
	assert.Equal(t, CredentialSchoolEmailOTP, *me.CredentialKind)
	assertOutboxJobStatus(t, fixture, outbox.StreamIAMOpenFGATupleSync, admissionProfileProjectionDedupeKey(userID), "pending")
	assertOutboxJobStatus(t, fixture, outbox.StreamIAMCasdoorRoleSync, admissionVerifiedStudentRoleDedupeKey(userID), "pending")

	markVerifiedProfileProjectionOutboxCompleted(t, fixture, userID)
	me, err = svc.GetAdmissionMe(context.Background(), userID, "")
	require.NoError(t, err)
	assert.False(t, me.ProjectionPending)
	assert.False(t, me.Session.ProjectionPending)
}

func TestAdmissionMeShowsStudentVerificationProjectionPending(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "projection-student-verification")
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "20003", Token: "projection-student-verification-token",
	})
	insertFreshmanCredential(t, fixture, freshmanCredentialSeed{
		ID: "projection-student-credential", UserID: userID, ExpiresAt: futureTime(60),
	})
	insertAdmissionVerificationProjectionOutbox(t, fixture, userID, "processing")

	me, err := svc.GetAdmissionMe(context.Background(), userID, "")

	require.NoError(t, err)
	assert.Equal(t, StatusLinked, me.Status)
	assert.True(t, me.ProjectionPending)
	require.NotNil(t, me.Session)
	assert.True(t, me.Session.ProjectionPending)
	require.NotNil(t, me.CredentialKind)
	assert.Equal(t, CredentialFreshmanMaterialManual, *me.CredentialKind)
}

func TestAdmissionMeIgnoresExpiredUnprocessedCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "projection-expired-credential")
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "20004", Token: "projection-expired-credential-token",
	})
	insertFreshmanCredential(t, fixture, freshmanCredentialSeed{
		ID: "projection-expired-credential", UserID: userID, ExpiresAt: fixedAdmissionNow().Add(-time.Minute),
	})

	me, err := svc.GetAdmissionMe(context.Background(), userID, "")

	require.NoError(t, err)
	assert.Equal(t, StatusLinked, me.Status)
	assert.Nil(t, me.CredentialKind)
	assert.Nil(t, me.ProvisionalExpiresAt)
}

func TestAdmissionMeIgnoresOtherSchoolCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	insertAdmissionPolicyForSchool(t, fixture, "adm-policy-projection-other-school", "guild-2", 4111010007)
	userID := seedAdmissionUser(t, fixture, "projection-other-school-credential")
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "20014", Token: "projection-other-school-credential-token",
	})
	insertAdmissionVerificationCredential(t, fixture, userID, 4111010007, "projection-other-school-credential")

	me, err := svc.GetAdmissionMe(context.Background(), userID, "")

	require.NoError(t, err)
	assert.Equal(t, StatusLinked, me.Status)
	assert.Nil(t, me.CredentialKind)
	assert.Nil(t, me.ProvisionalExpiresAt)
}

func TestAdmissionMeDoesNotTreatFailedProjectionAsPending(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "projection-failed")
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "93002", Token: "projection-failed-token",
	})
	_, err := svc.MarkVerified(context.Background(), latestAdmissionSessionID(t, fixture, userID))
	require.NoError(t, err)
	insertFreshmanProjectionOutbox(t, fixture, userID, "failed")

	me, err := svc.GetAdmissionMe(context.Background(), userID, "")

	require.NoError(t, err)
	assert.False(t, me.ProjectionPending)
	assert.False(t, me.Session.ProjectionPending)
}

func TestAdmissionMeCanTargetCurrentJoinSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "projection-current-session")
	first := linkAdmissionSessionForQQ(t, svc, userID, "20011", "projection-current-session-first")
	second := linkAdmissionSessionForQQ(t, svc, userID, "20012", "projection-current-session-second")
	setAdmissionSessionUpdatedAt(t, fixture, first.ID, fixedAdmissionNow())
	setAdmissionSessionUpdatedAt(t, fixture, second.ID, fixedAdmissionNow().Add(time.Minute))

	latest, err := svc.GetAdmissionMe(context.Background(), userID, "")
	require.NoError(t, err)
	require.NotNil(t, latest.Session)
	assert.Equal(t, second.ID, latest.Session.ID)

	current, err := svc.GetAdmissionMe(context.Background(), userID, first.ID)
	require.NoError(t, err)
	require.NotNil(t, current.Session)
	assert.Equal(t, first.ID, current.Session.ID)

	_, err = svc.GetAdmissionMe(context.Background(), userID, second.ID+"-missing")
	require.ErrorIs(t, err, ErrAdmissionSessionNotFound)
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
		VALUES ($1, $2, 4111010006, $3, $4, $5, $6)
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

func insertAdmissionVerificationProjectionOutbox(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	status string,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO domain_event_outbox (stream, job_type, dedupe_key, payload, status)
		VALUES ($1, 'admission_verification_projection', $2, '{}'::jsonb, $3)
	`, admissionVerificationProjectionStream, admissionVerificationProjectionDedupeKey(userID), status)
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

func markVerifiedProfileProjectionOutboxCompleted(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE domain_event_outbox
		SET status = 'completed', updated_at = NOW()
		WHERE (stream = $1 AND dedupe_key = $2)
		   OR (stream = $3 AND dedupe_key = $4)
	`, outbox.StreamIAMOpenFGATupleSync, admissionProfileProjectionDedupeKey(userID),
		outbox.StreamIAMCasdoorRoleSync, admissionVerifiedStudentRoleDedupeKey(userID))
	require.NoError(t, err)
}

func assertOutboxJobStatus(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	stream string,
	dedupeKey string,
	expected string,
) {
	t.Helper()
	var status string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT status
		FROM domain_event_outbox
		WHERE stream = $1 AND dedupe_key = $2
	`, stream, dedupeKey).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, expected, status)
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
