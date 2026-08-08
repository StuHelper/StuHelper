package studentverification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type fakePhoneAuthority struct {
	phone       string
	setCalls    int
	clearCalls  int
	setFailures int
}

func (a *fakePhoneAuthority) GetPhone(context.Context, int64) (string, error) {
	return a.phone, nil
}

func (a *fakePhoneAuthority) SetPhone(_ context.Context, _ int64, phone string) error {
	a.setCalls++
	if a.setFailures > 0 {
		a.setFailures--
		return errors.New("authority temporarily unavailable")
	}
	a.phone = phone
	return nil
}

func (a *fakePhoneAuthority) ClearPhone(context.Context, int64) error {
	a.clearCalls++
	a.phone = ""
	return nil
}

type fakePhoneOTP struct {
	issuedPhone   string
	checkedPhone  string
	checkedCode   string
	consumedPhone string
	consumedCode  string
}

func (o *fakePhoneOTP) Issue(_ context.Context, phone string) error {
	o.issuedPhone = phone
	return nil
}

func (o *fakePhoneOTP) Check(_ context.Context, phone, code string) error {
	o.checkedPhone = phone
	o.checkedCode = code
	return nil
}

func (o *fakePhoneOTP) Consume(_ context.Context, phone, code string) error {
	o.consumedPhone = phone
	o.consumedCode = code
	return nil
}

func (*fakePhoneOTP) CooldownSeconds() int { return 60 }

func newPhoneTestCipher(t *testing.T) *pii.Cipher {
	t.Helper()
	cipher, err := pii.NewCipher(1, map[uint8][]byte{1: []byte("0123456789abcdef0123456789abcdef")})
	require.NoError(t, err)
	return cipher
}

func TestPhoneSMSBindingUsesCasdoorReadbackBeforeActivatingCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-phone-integration-key")
	userID := seedVerificationUser(t, fixture, "phone-sms")
	authority := &fakePhoneAuthority{}
	otp := &fakePhoneOTP{}
	cipher := newPhoneTestCipher(t)
	service, err := NewService(
		NewRepository(fixture.DB),
		key,
		WithClock(func() time.Time { return now }),
		WithPhoneAuthority(authority),
		WithPhoneOTPService(otp),
		WithPhoneProjectionCipher(cipher, 1),
	)
	require.NoError(t, err)

	operation, err := service.CreatePhoneOperation(ctx, CreatePhoneOperationInput{
		UserID: userID, Kind: PhoneOperationBind, Phone: "13800138000",
	})
	require.NoError(t, err)
	assert.Equal(t, PhoneOperationPendingVerification, operation.Status)
	assert.Equal(t, "sms_otp", operation.VerificationStep)
	assert.Equal(t, "+86 138****8000", *operation.TargetPhoneMasked)
	assert.Empty(t, authority.phone)

	operation, err = service.SendPhoneSMS(ctx, userID, operation.ID)
	require.NoError(t, err)
	assert.Equal(t, "13800138000", otp.issuedPhone)
	assert.NotNil(t, operation.SMSResendAvailableAt)

	operation, err = service.VerifyPhoneSMS(ctx, VerifyPhoneSMSInput{
		UserID: userID, OperationID: operation.ID, Code: "123456",
	})
	require.NoError(t, err)
	assert.Equal(t, PhoneOperationCompleted, operation.Status)
	assert.Equal(t, "+8613800138000", authority.phone)
	assert.Equal(t, "13800138000", otp.checkedPhone)
	assert.Equal(t, "123456", otp.consumedCode)

	status, err := service.GetPhoneStatus(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "verified", status.State)
	assert.True(t, status.PublishingRequirementSatisfied)
	require.NotNil(t, status.Method)
	assert.Equal(t, PhoneMethodSMS, *status.Method)
	assert.Equal(t, "+86 138****8000", *status.MaskedPhone)

	var projectionEnc []byte
	var credentialCount, studentCredentialCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `SELECT phone_enc FROM users WHERE id = $1`, userID).Scan(&projectionEnc))
	projection, err := cipher.Decrypt(projectionEnc)
	require.NoError(t, err)
	assert.Equal(t, "+8613800138000", projection)
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM phone_verification_credentials
		WHERE user_id = $1 AND status = 'active' AND method = 'sms_possession'
	`, userID).Scan(&credentialCount))
	assert.Equal(t, 1, credentialCount)
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_verification_credentials WHERE user_id = $1
	`, userID).Scan(&studentCredentialCount))
	assert.Zero(t, studentCredentialCount, "phone verification must never create a student credential")
}

func TestPhoneRosterMatchRequiresAnActiveStudentSubjectAndSkipsSMS(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-phone-roster-key")
	userID := seedVerificationUser(t, fixture, "phone-roster")
	configureRealNameMethod(t, fixture, now)
	seedActiveRosterRecord(t, fixture, key, now, "20990001", "张三", "11010519491231002X")
	cipher := newPhoneTestCipher(t)
	phoneHash, err := ComputeRosterBlindIndex(key, testSchoolID, BlindIndexPhone, "13800138000")
	require.NoError(t, err)
	phoneEnc, err := cipher.Encrypt("13800138000")
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `
		UPDATE academic.student_roster_records
		SET phone_enc = $1, phone_hash = $2
		WHERE school_id = $3
	`, phoneEnc, phoneHash, testSchoolID)
	require.NoError(t, err)
	authority := &fakePhoneAuthority{}
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithPhoneAuthority(authority),
		WithPhoneProjectionCipher(cipher, 1),
	)
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: userID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	_, err = service.VerifyRealName(ctx, VerifyRealNameInput{
		UserID: userID, ApplicationID: application.ID,
		StudentID: "20990001", Name: "张三", DocumentNumber: "11010519491231002X",
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)

	operation, err := service.CreatePhoneOperation(ctx, CreatePhoneOperationInput{
		UserID: userID, Kind: PhoneOperationBind, Phone: "13800138000",
		SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	assert.Equal(t, PhoneOperationCompleted, operation.Status)
	assert.Equal(t, "+8613800138000", authority.phone)
	status, err := service.GetPhoneStatus(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, status.Method)
	assert.Equal(t, PhoneMethodRosterMatch, *status.Method)

	var schoolID *int64
	var subjectID, snapshotID *string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT school_id, enrollment_subject_id, roster_snapshot_id
		FROM phone_verification_credentials
		WHERE user_id = $1 AND status = 'active'
	`, userID).Scan(&schoolID, &subjectID, &snapshotID))
	assert.Equal(t, testSchoolID, *schoolID)
	assert.NotEmpty(t, *subjectID)
	assert.NotEmpty(t, *snapshotID)
}

func TestPhoneBindingFailureLeavesGateClosedAndCanBeRetried(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-phone-retry-key")
	userID := seedVerificationUser(t, fixture, "phone-retry")
	authority := &fakePhoneAuthority{setFailures: 1}
	otp := &fakePhoneOTP{}
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithPhoneAuthority(authority),
		WithPhoneOTPService(otp),
		WithPhoneProjectionCipher(newPhoneTestCipher(t), 1),
	)
	require.NoError(t, err)
	operation, err := service.CreatePhoneOperation(ctx, CreatePhoneOperationInput{
		UserID: userID, Kind: PhoneOperationBind, Phone: "13800138000",
	})
	require.NoError(t, err)
	_, err = service.SendPhoneSMS(ctx, userID, operation.ID)
	require.NoError(t, err)
	operation, err = service.VerifyPhoneSMS(ctx, VerifyPhoneSMSInput{
		UserID: userID, OperationID: operation.ID, Code: "123456",
	})
	require.NoError(t, err)
	assert.Equal(t, PhoneOperationCasdoorUpdatePending, operation.Status)
	gate, err := service.GetPhoneGateEligibility(ctx, userID)
	require.NoError(t, err)
	assert.False(t, gate.Eligible)

	require.NoError(t, service.ProcessPhoneOperation(ctx, userID, operation.ID))
	gate, err = service.GetPhoneGateEligibility(ctx, userID)
	require.NoError(t, err)
	assert.True(t, gate.Eligible)
}

func TestPhoneChangeAndUnbindKeepCasdoorAuthoritative(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-phone-lifecycle-key")
	userID := seedVerificationUser(t, fixture, "phone-lifecycle")
	authority := &fakePhoneAuthority{}
	otp := &fakePhoneOTP{}
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithPhoneAuthority(authority),
		WithPhoneOTPService(otp),
		WithPhoneProjectionCipher(newPhoneTestCipher(t), 1),
	)
	require.NoError(t, err)

	bind, err := service.CreatePhoneOperation(ctx, CreatePhoneOperationInput{
		UserID: userID, Kind: PhoneOperationBind, Phone: "13800138000",
	})
	require.NoError(t, err)
	_, err = service.SendPhoneSMS(ctx, userID, bind.ID)
	require.NoError(t, err)
	_, err = service.VerifyPhoneSMS(ctx, VerifyPhoneSMSInput{
		UserID: userID, OperationID: bind.ID, Code: "123456",
	})
	require.NoError(t, err)

	change, err := service.CreatePhoneOperation(ctx, CreatePhoneOperationInput{
		UserID: userID, Kind: PhoneOperationChange, Phone: "13900139000",
	})
	require.NoError(t, err)
	_, err = service.SendPhoneSMS(ctx, userID, change.ID)
	require.NoError(t, err)
	change, err = service.VerifyPhoneSMS(ctx, VerifyPhoneSMSInput{
		UserID: userID, OperationID: change.ID, Code: "654321",
	})
	require.NoError(t, err)
	assert.Equal(t, PhoneOperationCompleted, change.Status)
	assert.Equal(t, "+8613900139000", authority.phone)

	var activeCredentials, revokedCredentials, activeClaims int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'active'),
		       COUNT(*) FILTER (WHERE status = 'revoked')
		FROM phone_verification_credentials WHERE user_id = $1
	`, userID).Scan(&activeCredentials, &revokedCredentials))
	assert.Equal(t, 1, activeCredentials)
	assert.Equal(t, 1, revokedCredentials)
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM phone_number_claims
		WHERE user_id = $1 AND claim_status = 'active'
	`, userID).Scan(&activeClaims))
	assert.Equal(t, 1, activeClaims)

	unbind, err := service.CreatePhoneUnbindOperation(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PhoneOperationCompleted, unbind.Status)
	assert.Empty(t, authority.phone)
	assert.Equal(t, 1, authority.clearCalls)
	status, err := service.GetPhoneStatus(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "unbound", status.State)
	assert.False(t, status.PublishingRequirementSatisfied)
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM phone_number_claims WHERE user_id = $1
	`, userID).Scan(&activeClaims))
	assert.Zero(t, activeClaims)
}
