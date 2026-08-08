package studentverification

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

const (
	testSchoolID   int64 = 4111010006
	testSchoolCode       = "4111010006"
)

func TestRealNameVerificationCreatesCredentialAndRealtimeEligibility(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	key := []byte("student-verification-integration-hmac-key")
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "real-name")
	configureRealNameMethod(t, fixture, now)
	seedActiveRosterRecord(t, fixture, key, now, "20990001", "张三", "11010519491231002X")

	service, err := NewService(
		NewRepository(fixture.DB),
		key,
		WithClock(func() time.Time { return now }),
	)
	require.NoError(t, err)

	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: userID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	require.Equal(t, ApplicationCreated, application.Status)

	application, err = service.VerifyRealName(ctx, VerifyRealNameInput{
		UserID: userID, ApplicationID: application.ID,
		StudentID: "20990001", Name: "张三", DocumentNumber: "11010519491231002X",
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)
	require.Equal(t, ApplicationApproved, application.Status)
	require.NotNil(t, application.Credential)
	assert.Equal(t, MethodRealNameIdentityCheck, application.Credential.Method)
	assert.Equal(t, CredentialActive, application.Credential.Status)
	assert.Equal(t, "20****01", application.Credential.SubjectDisplay)

	eligibility, err := service.GetEligibility(ctx, userID, testSchoolCode)
	require.NoError(t, err)
	assert.True(t, eligibility.Eligible)
	assert.Equal(t, int64(1), eligibility.Revision)
	assert.Equal(t, []Method{MethodRealNameIdentityCheck}, eligibility.CredentialMethods)
	current, err := service.GetCurrentStudentStatus(ctx, userID)
	require.NoError(t, err)
	assert.True(t, current.Eligible)
	require.NotNil(t, current.SchoolID)
	assert.Equal(t, testSchoolID, *current.SchoolID)

	var outboxCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_verification_event_outbox
		WHERE user_id = $1 AND school_id = $2 AND status = 'pending'
	`, userID, testSchoolID).Scan(&outboxCount))
	assert.Equal(t, 1, outboxCount)

	revoked, err := service.RevokeCredential(ctx, userID, application.Credential.ID)
	require.NoError(t, err)
	assert.Equal(t, CredentialRevoked, revoked.Status)

	eligibility, err = service.GetEligibility(ctx, userID, testSchoolCode)
	require.NoError(t, err)
	assert.False(t, eligibility.Eligible)
	assert.Equal(t, int64(2), eligibility.Revision)
	current, err = service.GetCurrentStudentStatus(ctx, userID)
	require.NoError(t, err)
	assert.False(t, current.Eligible)
	assert.Nil(t, current.SchoolID)
}

func TestRealNameVerificationFailsClosedForStaleRoster(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	key := []byte("student-verification-integration-hmac-key")
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "stale-roster")
	configureRealNameMethod(t, fixture, now)
	seedActiveRosterRecord(t, fixture, key, now.Add(-49*time.Hour), "20990001", "张三", "11010519491231002X")

	service, err := NewService(NewRepository(fixture.DB), key, WithClock(func() time.Time { return now }))
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{UserID: userID, SchoolCode: testSchoolCode})
	require.NoError(t, err)

	_, err = service.VerifyRealName(ctx, VerifyRealNameInput{
		UserID: userID, ApplicationID: application.ID,
		StudentID: "20990001", Name: "张三", DocumentNumber: "11010519491231002X",
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.ErrorIs(t, err, ErrDependencyUnavailable)

	credentials, err := service.ListCredentials(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, credentials)
}

func TestCancelApplicationIsIdempotentAndDoesNotRewriteCompletedApplication(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	key := []byte("student-verification-integration-hmac-key")
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "cancel-application")
	configureRealNameMethod(t, fixture, now)

	service, err := NewService(NewRepository(fixture.DB), key, WithClock(func() time.Time { return now }))
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{UserID: userID, SchoolCode: testSchoolCode})
	require.NoError(t, err)

	cancelled, err := service.CancelApplication(ctx, userID, application.ID)
	require.NoError(t, err)
	assert.Equal(t, ApplicationCancelled, cancelled.Status)
	assert.Equal(t, int64(2), cancelled.Revision)
	require.NotNil(t, cancelled.TerminalCode)
	assert.Equal(t, "user_cancelled", *cancelled.TerminalCode)

	repeated, err := service.CancelApplication(ctx, userID, application.ID)
	require.NoError(t, err)
	assert.Equal(t, ApplicationCancelled, repeated.Status)
	assert.Equal(t, cancelled.Revision, repeated.Revision)

	replacement, err := service.CreateApplication(ctx, CreateApplicationInput{UserID: userID, SchoolCode: testSchoolCode})
	require.NoError(t, err)
	assert.NotEqual(t, application.ID, replacement.ID)

	_, err = fixture.Pool.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'approved', completed_at = $2, terminal_code = 'verified', updated_at = $2
		WHERE id = $1
	`, replacement.ID, now)
	require.NoError(t, err)
	_, err = service.CancelApplication(ctx, userID, replacement.ID)
	require.ErrorIs(t, err, ErrApplicationState)
}

type capturingStudentEmailSender struct {
	email string
	code  string
}

func (s *capturingStudentEmailSender) SendStudentVerificationOTP(_ context.Context, email string, code string) error {
	s.email = email
	s.code = code
	return nil
}

func TestStudentEmailOTPUsesCanonicalRosterAddressAndCreatesCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	redisServer := redisfixture.Start(t)
	ctx := context.Background()
	key := []byte("student-verification-integration-hmac-key")
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "email-otp")
	configureEmailOTPMethod(t, fixture, now)
	seedActiveRosterRecord(t, fixture, key, now, "20990001", "张三", "11010519491231002X")
	sender := &capturingStudentEmailSender{}

	service, err := NewService(
		NewRepository(fixture.DB),
		key,
		WithClock(func() time.Time { return now }),
		WithRedisClient(redisServer.Client),
		WithStudentEmailSender(sender),
		WithOTPGenerator(func() (string, error) { return "123456", nil }),
	)
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{UserID: userID, SchoolCode: testSchoolCode})
	require.NoError(t, err)

	challenge, err := service.RequestStudentEmailOTP(ctx, StudentEmailIdentityInput{
		UserID: userID, ApplicationID: application.ID,
		StudentID: "20990001", Name: "张三",
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "20990001@buaa.edu.cn", sender.email)
	assert.Equal(t, "123456", sender.code)
	assert.Equal(t, "20****01@buaa.edu.cn", challenge.MaskedEmail)

	application, err = service.VerifyStudentEmailOTP(ctx, VerifyStudentEmailOTPInput{
		UserID: userID, ApplicationID: application.ID, Code: "123456",
	})
	require.NoError(t, err)
	require.NotNil(t, application.Credential)
	assert.Equal(t, MethodStudentEmailOutboundOTP, application.Credential.Method)

	var persistedEvidence string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT metadata::text
		FROM user_verification_credentials
		WHERE id = $1
	`, application.Credential.ID).Scan(&persistedEvidence))
	assert.NotContains(t, persistedEvidence, "20990001")
	assert.NotContains(t, persistedEvidence, "张三")
	assert.NotContains(t, persistedEvidence, "buaa.edu.cn")
}

type fakeSchoolAccountAuthenticator struct {
	request SchoolAccountAuthenticationRequest
	result  *SchoolAccountAuthenticationResult
	err     error
}

func (a *fakeSchoolAccountAuthenticator) Authenticate(
	_ context.Context,
	request SchoolAccountAuthenticationRequest,
) (*SchoolAccountAuthenticationResult, error) {
	a.request = request
	return a.result, a.err
}

func TestSchoolSSOAdapterAssertionCreatesConditionalCredentialWithoutPersistingPassword(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	key := []byte("student-verification-integration-hmac-key")
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "school-sso")
	configureSchoolSSOMethod(t, fixture, now)
	authenticator := &fakeSchoolAccountAuthenticator{result: &SchoolAccountAuthenticationResult{
		AccountSubject: "stable-school-account-subject",
		StudentID:      "20990001",
		Attributes:     map[string]bool{"current_student": true},
	}}
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithSchoolAccountAuthenticator("buaa_ldap_bind", authenticator),
	)
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{UserID: userID, SchoolCode: testSchoolCode})
	require.NoError(t, err)
	password := []byte("request-only-school-password")

	application, err = service.VerifySchoolSSO(ctx, VerifySchoolSSOInput{
		UserID: userID, ApplicationID: application.ID,
		StudentID: "20990001", Password: password,
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, application.Credential)
	assert.Equal(t, MethodSchoolSSO, application.Credential.Method)
	assert.Equal(t, "buaa.ldap.authenticate", authenticator.request.ConnectorOperation)
	assert.Equal(t, password, authenticator.request.Password)

	eligibility, err := service.GetEligibility(ctx, userID, testSchoolCode)
	require.NoError(t, err)
	assert.True(t, eligibility.Eligible)

	var persisted string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT concat_ws(' ', c.metadata::text, a.evidence_metadata::text)
		FROM user_verification_credentials c
		JOIN student_verification_attempts a
		  ON a.application_id = c.verification_application_id
		WHERE c.id = $1
	`, application.Credential.ID).Scan(&persisted))
	assert.NotContains(t, persisted, string(password))
	assert.NotContains(t, persisted, "stable-school-account-subject")
}

func seedVerificationUser(t *testing.T, fixture *postgresfixture.Fixture, suffix string) int64 {
	t.Helper()
	var userID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "casdoor-student-verification-"+suffix, "student_verification_"+suffix, suffix+"@example.test").Scan(&userID)
	require.NoError(t, err)
	return userID
}

func configureRealNameMethod(t *testing.T, fixture *postgresfixture.Fixture, now time.Time) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_profiles
		SET enabled = true,
		    validation_status = 'valid',
		    validated_at = $2,
		    enrollment_policy = '{"mainlandDocumentTypes":["1"]}'::jsonb,
		    snapshot_sync_interval_seconds = 21600,
		    snapshot_warning_after_seconds = 43200,
		    snapshot_hard_expiry_seconds = 172800,
		    updated_at = $2
		WHERE school_id = $1
	`, testSchoolID, now)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_methods
		SET enabled = true,
		    validation_status = 'valid',
		    validated_at = $2,
		    health_status = 'healthy',
		    health_checked_at = $2,
		    privacy_notice_version = '2026-08-05',
		    privacy_notice = '{
		      "title":"实名信息校验",
		      "summary":"用于完成本次学生认证",
		      "dataCategories":["学号","姓名","身份证件号"],
		      "retentionSummary":"本次提交原文不写入学生认证业务库"
		    }'::jsonb,
		    updated_at = $2
		WHERE school_id = $1 AND method = 'real_name_identity_check'
	`, testSchoolID, now)
	require.NoError(t, err)
}

func configureEmailOTPMethod(t *testing.T, fixture *postgresfixture.Fixture, now time.Time) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_profiles
		SET enabled = true,
		    validation_status = 'valid',
		    validated_at = $2,
		    snapshot_sync_interval_seconds = 21600,
		    snapshot_warning_after_seconds = 43200,
		    snapshot_hard_expiry_seconds = 172800,
		    updated_at = $2
		WHERE school_id = $1
	`, testSchoolID, now)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_methods
		SET enabled = true,
		    validation_status = 'valid',
		    validated_at = $2,
		    health_status = 'healthy',
		    health_checked_at = $2,
		    privacy_notice_version = '2026-08-05',
		    privacy_notice = '{
		      "title":"学校邮箱验证",
		      "summary":"向规范学校邮箱发送一次性验证码",
		      "dataCategories":["学号","姓名","学校邮箱"],
		      "retentionSummary":"验证码及地址仅在挑战有效期内处理"
		    }'::jsonb,
		    risk_policy = '{"otpTtlSeconds":300,"otpCooldownSeconds":60,"otpMaxAttempts":5}'::jsonb,
		    updated_at = $2
		WHERE school_id = $1 AND method = 'student_email_outbound_otp'
	`, testSchoolID, now)
	require.NoError(t, err)
}

func configureSchoolSSOMethod(t *testing.T, fixture *postgresfixture.Fixture, now time.Time) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_profiles
		SET enabled = true,
		    validation_status = 'valid',
		    validated_at = $2,
		    updated_at = $2
		WHERE school_id = $1
	`, testSchoolID, now)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_methods
		SET enabled = true,
		    validation_status = 'valid',
		    validated_at = $2,
		    health_status = 'healthy',
		    health_checked_at = $2,
		    privacy_notice_version = '2026-08-05',
		    privacy_notice = '{
		      "title":"统一身份认证验证",
		      "summary":"代为提交一次学校统一身份认证账号校验",
		      "dataCategories":["学号","学校账号密码"],
		      "retentionSummary":"密码只在本次受控请求内存中处理"
		    }'::jsonb,
		    updated_at = $2
		WHERE school_id = $1 AND method = 'school_sso'
	`, testSchoolID, now)
	require.NoError(t, err)
}

func seedActiveRosterRecord(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	key []byte,
	sourceCutoff time.Time,
	studentID string,
	name string,
	documentNumber string,
) {
	t.Helper()
	ctx := context.Background()
	studentHash, err := ComputeRosterBlindIndex(key, testSchoolID, BlindIndexStudentID, studentID)
	require.NoError(t, err)
	nameHash, err := ComputeRosterBlindIndex(key, testSchoolID, BlindIndexName, name)
	require.NoError(t, err)
	documentHash, err := ComputeRosterBlindIndex(key, testSchoolID, BlindIndexDocumentNumber, documentNumber)
	require.NoError(t, err)
	snapshotID := "0198c2e1-7e91-7000-8000-" + strings.Repeat("0", 11) + "1"
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO academic.student_roster_snapshots (
		    id, school_id, source_kind, source_version, import_mode,
		    schema_version, mapping_version, status, source_cutoff_at,
		    import_completed_at, activated_at, row_count, eligible_row_count,
		    checksum, encryption_key_version, hmac_key_version
		)
		VALUES ($1, $2, 'fixture', $3, 'full', 1, 'test-v1', 'active', $4,
		        $5, $5, 1, 1, $6, 1, 1)
	`, snapshotID, testSchoolID, "fixture-"+sourceCutoff.Format(time.RFC3339Nano), sourceCutoff,
		time.Now().UTC(), strings.Repeat("a", 64))
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO academic.student_roster_records (
		    snapshot_id, school_id, source_record_key_hash,
		    student_id_enc, student_id_hash, name_enc, name_hash,
		    document_type, document_number_enc, document_number_hash,
		    encryption_key_version, hmac_key_version, current_marker,
		    eligibility_status, eligibility_code, record_checksum
		)
		VALUES ($1, $2, $3, decode('0101', 'hex'), $3,
		        decode('0102', 'hex'), $4, '1', decode('0103', 'hex'), $5,
		        1, 1, true, 'eligible', 'active_student', $6)
	`, snapshotID, testSchoolID, studentHash, nameHash, documentHash, strings.Repeat("b", 64))
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO academic.student_roster_active (
		    school_id, snapshot_id, activation_revision, activated_at
		)
		VALUES ($2, $1, 1, $3)
	`, snapshotID, testSchoolID, time.Now().UTC())
	require.NoError(t, err)
}
