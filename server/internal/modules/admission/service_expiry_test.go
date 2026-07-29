package admission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestExpiredFreshmanCredentialRevokesOnlyFreshmanProjection(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	operatorID := seedAdmissionUser(t, fixture, "expiry-operator")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "91001"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}
	app := seedReviewableFreshmanApplication(t, fixture, svc)

	approved, err := svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInput(app.ID, "91001"),
	)
	require.NoError(t, err)
	freshmanCredentialID := credentialIDByKind(t, fixture, approved.UserID, CredentialFreshmanMaterialManual)
	insertSchoolSSOCredential(t, fixture, approved.UserID)
	expireFreshmanCredential(t, fixture, freshmanCredentialID, svc.now().Add(-time.Minute))
	resetProjectionCalls(t, svc)

	processed, err := svc.ProcessExpiredFreshmanCredentials(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assertProjectionEnqueued(t, svc, approved.UserID, false)
	assertCredentialExpiryProcessed(t, fixture, freshmanCredentialID)
	assertCredentialNotRevoked(t, fixture, "school-sso-expiry")

	processed, err = svc.ProcessExpiredFreshmanCredentials(context.Background())
	require.NoError(t, err)
	assert.Zero(t, processed)
	assert.Len(t, svc.projection.(*testFreshmanProjectionGateway).calls, 1)
}

func resetProjectionCalls(t *testing.T, svc *Service) {
	t.Helper()
	gateway, ok := svc.projection.(*testFreshmanProjectionGateway)
	require.True(t, ok)
	gateway.calls = nil
}

func credentialIDByKind(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	kind VerificationCredentialKind,
) string {
	t.Helper()
	var id string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT id
		FROM user_verification_credentials
		WHERE user_id = $1 AND kind = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, kind).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertSchoolSSOCredential(t *testing.T, fixture *postgresfixture.Fixture, userID int64) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display, expires_at
		)
		VALUES ('school-sso-expiry', $1, 4111010006, $2, 'school-sso-hash', 'official student', $3)
	`, userID, CredentialSchoolSSO, time.Now().Add(-time.Hour))
	require.NoError(t, err)
}

func expireFreshmanCredential(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	credentialID string,
	expiresAt time.Time,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE user_verification_credentials
		SET expires_at = $2, revoked_at = NULL, expiry_processed_at = NULL
		WHERE id = $1
	`, credentialID, expiresAt)
	require.NoError(t, err)
}

func assertCredentialExpiryProcessed(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	credentialID string,
) {
	t.Helper()
	var revokedAt *time.Time
	var processedAt *time.Time
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT revoked_at, expiry_processed_at
		FROM user_verification_credentials
		WHERE id = $1
	`, credentialID).Scan(&revokedAt, &processedAt)
	require.NoError(t, err)
	assert.NotNil(t, revokedAt)
	assert.NotNil(t, processedAt)
}

func assertCredentialNotRevoked(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	credentialID string,
) {
	t.Helper()
	var revokedAt *time.Time
	var processedAt *time.Time
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT revoked_at, expiry_processed_at
		FROM user_verification_credentials
		WHERE id = $1
	`, credentialID).Scan(&revokedAt, &processedAt)
	require.NoError(t, err)
	assert.Nil(t, revokedAt)
	assert.Nil(t, processedAt)
}
