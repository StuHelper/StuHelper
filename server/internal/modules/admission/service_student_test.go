package admission

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestSchoolEmailOTPRequiresLinkedSessionAndVerifiesCredential(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp")

	_, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@other.example",
	})
	require.ErrorIs(t, err, ErrAdmissionEmailDomainNotAllowed)

	resp, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    " Student@BUAA.edu.cn ",
	})
	require.NoError(t, err)
	assert.Equal(t, admissionEmailOTPCooldownSeconds, resp.CooldownSeconds)
	assert.Equal(t, "student@buaa.edu.cn", sender.email)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@buaa.edu.cn",
		Code:     sender.code,
	})
	require.NoError(t, err)
	assertCredentialStored(t, pg, userID, CredentialSchoolEmailOTP, "s*****t@buaa.edu.cn")
	assertUserSessionVerified(t, pg, userID)
}

func TestSchoolEmailOTPSendFailureClearsCooldown(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{err: errors.New("smtp rejected")}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-send-fail")

	_, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID: userID, SchoolID: 1, Email: "student@buaa.edu.cn",
	})
	require.Error(t, err)

	sender.err = nil
	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID: userID, SchoolID: 1, Email: "student@buaa.edu.cn",
	})
	require.NoError(t, err)
}

func TestSchoolSSOStartAndCallback(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.schoolSSO = &testSchoolSSOExchanger{
		identity: SchoolSSOIdentity{Subject: "student-official-id", SubjectDisplay: "official student"},
	}
	userID := seedLinkedAdmissionUser(t, pg, svc, "school-sso")

	_, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  1,
		ReturnURL: "https://evil.example/admission/a/token",
	})
	require.ErrorIs(t, err, ErrAdmissionReturnURLNotAllowed)

	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  1,
		ReturnURL: "https://auth.stuhelper.com/admission/a/token?qq=10001",
	})
	require.NoError(t, err)
	assert.Contains(t, start.RedirectURL, "https://sso.school.example/login")
	assert.NotEmpty(t, start.State)

	complete, err := svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 1,
		State:    start.State,
		UserID:   userID,
		Code:     "oidc-code",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://auth.stuhelper.com/admission/a/token?qq=10001", complete.ReturnURL)
	assertCredentialStored(t, pg, userID, CredentialSchoolSSO, "official student")
	assertUserSessionVerified(t, pg, userID)

	_, err = svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 1,
		State:    start.State,
		UserID:   userID,
		Code:     "oidc-code",
	})
	require.ErrorIs(t, err, ErrAdmissionSSOStateInvalid)
}

func TestSchoolSSOCallbackRequiresConfiguredExchanger(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	userID := seedLinkedAdmissionUser(t, pg, svc, "school-sso-no-exchanger")
	start := startSchoolSSOForTest(t, svc, userID)

	_, err := svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 1,
		State:    start.State,
		UserID:   userID,
		Code:     "attacker-controlled-code",
	})

	require.ErrorIs(t, err, ErrAdmissionSSONotConfigured)
	assertNoCredentialStored(t, pg, userID, CredentialSchoolSSO)
}

func TestSchoolSSOCallbackRejectsInvalidProviderIdentity(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.schoolSSO = &testSchoolSSOExchanger{identity: SchoolSSOIdentity{Subject: " "}}
	userID := seedLinkedAdmissionUser(t, pg, svc, "school-sso-invalid-identity")
	start := startSchoolSSOForTest(t, svc, userID)

	_, err := svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 1,
		State:    start.State,
		UserID:   userID,
		Code:     "oidc-code",
	})

	require.ErrorIs(t, err, ErrAdmissionSSOIdentityInvalid)
	assertNoCredentialStored(t, pg, userID, CredentialSchoolSSO)
}

func startSchoolSSOForTest(t *testing.T, svc *Service, userID int64) *SchoolSSOStartResult {
	t.Helper()
	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  1,
		ReturnURL: "https://auth.stuhelper.com/admission/a/token?qq=10001",
	})
	require.NoError(t, err)
	return start
}

type testSchoolEmailSender struct {
	email string
	code  string
	err   error
}

type testSchoolSSOExchanger struct {
	identity SchoolSSOIdentity
	err      error
	code     string
}

func (e *testSchoolSSOExchanger) ExchangeSchoolSSO(
	_ context.Context,
	input SchoolSSOExchangeInput,
) (SchoolSSOIdentity, error) {
	e.code = input.Code
	return e.identity, e.err
}

func (s *testSchoolEmailSender) SendAdmissionOTP(_ context.Context, email string, code string) error {
	s.email = email
	s.code = code
	return s.err
}

func assertCredentialStored(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	kind VerificationCredentialKind,
	display string,
) {
	t.Helper()
	var count int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM user_verification_credentials
		WHERE user_id = $1 AND kind = $2 AND subject_display = $3
	`, userID, kind, display).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func assertUserSessionVerified(t *testing.T, fixture *postgresfixture.Fixture, userID int64) {
	t.Helper()
	var status string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT status
		FROM group_admission_sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, userID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, string(StatusVerified), status)
}

func assertNoCredentialStored(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	kind VerificationCredentialKind,
) {
	t.Helper()
	var count int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM user_verification_credentials
		WHERE user_id = $1 AND kind = $2
	`, userID, kind).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func assertMaskedEmailShape(t *testing.T, masked string) {
	t.Helper()
	assert.True(t, strings.Contains(masked, "@"))
	assert.NotContains(t, masked, "student@")
}
