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
}

// F001/F007 回归：新生临时凭证过期且无其他有效凭证时，user_profiles 投影必须降级，
// join decision 不得继续返回 verified，verified_student 角色同步必须以 approved=false 入队。
func TestExpiredFreshmanCredentialDowngradesProjectionAndJoinDecision(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	operatorID := seedAdmissionUser(t, fixture, "expiry-downgrade-operator")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "92001"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}
	app := seedReviewableFreshmanApplication(t, fixture, svc)
	approved, err := svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInput(app.ID, "92001"),
	)
	require.NoError(t, err)
	setAdmissionPolicyJoinRequestReview(t, fixture, "请先完成认证后再申请入群")
	applicantQQ := "93001"
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: approved.UserID, QQID: applicantQQ})

	before := resolveJoinDecisionForQQ(t, svc, applicantQQ)
	assert.Equal(t, AdmissionJoinRequestDecisionApprove, before.Decision)
	assert.Equal(t, AdmissionJoinRequestVerified, before.VerificationState)
	assertUserProfileVerified(t, fixture, approved.UserID, "manual")

	freshmanCredentialID := credentialIDByKind(t, fixture, approved.UserID, CredentialFreshmanMaterialManual)
	expireFreshmanCredential(t, fixture, freshmanCredentialID, svc.now().Add(-time.Minute))
	resetProjectionCalls(t, svc)

	processed, err := svc.ProcessExpiredFreshmanCredentials(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	after := resolveJoinDecisionForQQ(t, svc, applicantQQ)
	assert.Equal(t, AdmissionJoinRequestDecisionReject, after.Decision)
	assert.Equal(t, AdmissionJoinRequestUnverified, after.VerificationState)
	assert.Nil(t, after.UserID)
	assertUserProfileUnverified(t, fixture, approved.UserID)
	assertProjectionEnqueued(t, svc, approved.UserID, false)
	assertVerifiedRoleSyncJobApproved(t, fixture, approved.UserID, false)
}

// F007 边界：新生临时凭证过期，但用户经主站正规流程重新认证（仍持有其他有效凭证）时，
// join decision 与 user_profiles 投影必须保持 verified，不得被过期凭证误伤。
func TestExpiredFreshmanCredentialKeepsVerifiedWithRemainingCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	operatorID := seedAdmissionUser(t, fixture, "expiry-keep-operator")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "92002"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}
	app := seedReviewableFreshmanApplication(t, fixture, svc)
	approved, err := svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInput(app.ID, "92002"),
	)
	require.NoError(t, err)
	setAdmissionPolicyJoinRequestReview(t, fixture, "请先完成认证后再申请入群")
	applicantQQ := "93002"
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: approved.UserID, QQID: applicantQQ})
	insertAdmissionVerificationCredential(t, fixture, approved.UserID, 4111010006, "expiry-keep")

	freshmanCredentialID := credentialIDByKind(t, fixture, approved.UserID, CredentialFreshmanMaterialManual)
	expireFreshmanCredential(t, fixture, freshmanCredentialID, svc.now().Add(-time.Minute))

	processed, err := svc.ProcessExpiredFreshmanCredentials(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	decision := resolveJoinDecisionForQQ(t, svc, applicantQQ)
	assert.Equal(t, AdmissionJoinRequestDecisionApprove, decision.Decision)
	assert.Equal(t, AdmissionJoinRequestVerified, decision.VerificationState)
	assertUserProfileVerified(t, fixture, approved.UserID, "school_email_otp")
	assertVerifiedRoleSyncJobApproved(t, fixture, approved.UserID, true)
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

func resolveJoinDecisionForQQ(t *testing.T, svc *Service, qqID string) *AdmissionJoinRequestDecision {
	t.Helper()
	decision, err := svc.ResolveJoinRequestDecision(context.Background(), AdmissionJoinRequestDecisionInput{
		Platform:  "qq",
		GuildID:   "guild-1",
		QQID:      qqID,
		RequestID: "request-expiry-regression",
	})
	require.NoError(t, err)
	require.NotNil(t, decision)
	return decision
}

func assertUserProfileUnverified(t *testing.T, fixture *postgresfixture.Fixture, userID int64) {
	t.Helper()
	var status string
	var method *string
	var verifiedAt *time.Time
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT verification_status, verification_method, verified_at
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(&status, &method, &verifiedAt)
	require.NoError(t, err)
	assert.Equal(t, "unverified", status)
	assert.Nil(t, method)
	assert.Nil(t, verifiedAt)
}

func assertVerifiedRoleSyncJobApproved(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	approved bool,
) {
	t.Helper()
	var payloadApproved bool
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT (payload->>'approved')::boolean
		FROM domain_event_outbox
		WHERE stream = $1 AND dedupe_key = $2
	`, outbox.StreamIAMCasdoorRoleSync, admissionVerifiedStudentRoleDedupeKey(userID)).Scan(&payloadApproved)
	require.NoError(t, err)
	assert.Equal(t, approved, payloadApproved)
}
