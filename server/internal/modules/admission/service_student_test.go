package admission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
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

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionEmailDomainNotAllowed)
	assert.Empty(t, sender.email)

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

func TestSchoolEmailOTPUsesRequestedAdmissionSession(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	userID := seedAdmissionUser(t, pg, "email-otp-current-session")
	first := linkAdmissionSessionForQQ(t, svc, userID, "10021", "email-otp-current-session-first")
	second := linkAdmissionSessionForQQ(t, svc, userID, "10022", "email-otp-current-session-second")
	setAdmissionSessionUpdatedAt(t, pg, first.ID, fixedAdmissionNow())
	setAdmissionSessionUpdatedAt(t, pg, second.ID, fixedAdmissionNow().Add(time.Minute))

	_, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:             userID,
		SchoolID:           1,
		AdmissionSessionID: first.ID,
		Email:              "student@buaa.edu.cn",
	})
	require.NoError(t, err)
	verified, err := svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:             userID,
		SchoolID:           1,
		AdmissionSessionID: first.ID,
		Email:              "student@buaa.edu.cn",
		Code:               sender.code,
	})

	require.NoError(t, err)
	require.NotNil(t, verified)
	assert.Equal(t, first.ID, verified.ID)
	assert.NotEqual(t, second.ID, verified.ID)
}

func TestSchoolEmailOTPDerivesBUAAEmailAfterAcademicNameMatch(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	svc.academicLookup = testAcademicLookupGateway{
		student: &user.AcademicStudent{XH: "20250001", XM: stringPtr("张三")},
	}
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-academic")

	_, err := pg.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true,
		    manual_form_fields = '{"admission":{"emailDomains":["buaa.edu.cn"],"emailIdentityPolicy":{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}}}'::jsonb
		WHERE school_id = 10006
	`)
	require.NoError(t, err)

	resp, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    10006,
		StudentID:   "20250001",
		StudentName: " 张 三 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "20250001@buaa.edu.cn", resp.Email)
	assert.Equal(t, "20250001@buaa.edu.cn", sender.email)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    10006,
		StudentID:   "20250001",
		StudentName: "李四",
	})
	require.ErrorIs(t, err, ErrAdmissionStudentNameMismatch)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    10006,
		Email:       "alias@buaa.edu.cn",
		StudentID:   "20250001",
		StudentName: "张三",
	})
	require.ErrorIs(t, err, ErrAdmissionEmailDomainNotAllowed)
}

func TestSchoolEmailOTPRejectsExpiredLinkedSession(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-expired")

	_, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@buaa.edu.cn",
	})
	require.NoError(t, err)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@buaa.edu.cn",
		Code:     sender.code,
	})
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)
	assertNoCredentialStored(t, pg, userID, CredentialSchoolEmailOTP)
}

func TestResolveSchoolIDByCodeUsesEnabledAdmissionSchoolConfig(t *testing.T) {
	pg := postgresfixture.Start(t)
	svc := newSessionTestService(t, pg)
	_, err := pg.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true
		WHERE school_id = 10006
	`)
	require.NoError(t, err)

	schoolID, err := svc.ResolveSchoolIDByCode(context.Background(), "4111010006")
	require.NoError(t, err)
	assert.Equal(t, int64(10006), schoolID)

	_, err = svc.ResolveSchoolIDByCode(context.Background(), "4111010001")
	require.ErrorIs(t, err, ErrAdmissionSchoolNotFound)
}

func TestAdmissionMVPEmailOTPFlowReleasesVerifiedMember(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token?qq=10001", created.AuthURL)

	userID := seedAdmissionUser(t, pg, "mvp-email-otp")
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:   created.Token,
		QQQuery: "10001",
		UserID:  userID,
	})
	require.NoError(t, err)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@buaa.edu.cn",
	})
	require.NoError(t, err)
	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 1,
		Email:    "student@buaa.edu.cn",
		Code:     sender.code,
	})
	require.NoError(t, err)

	actions, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.AuthURL, actions[0].AuthURL)

	err = svc.RecordBotEvent(context.Background(), actions[0].SessionID, BotEventInput{
		Action:  actions[0].Action,
		Success: true,
	})
	require.NoError(t, err)
	assertAdmissionSessionCancelled(t, pg, created.Session.ID)
}

func TestAdmissionMVPSchoolSSOFlowReleasesVerifiedMember(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.schoolSSO = &testSchoolSSOExchanger{
		identity: SchoolSSOIdentity{
			Subject:        "school-sso-student-id",
			SubjectDisplay: "school-sso-student",
		},
	}

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	userID := seedAdmissionUser(t, pg, "mvp-school-sso")
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:   created.Token,
		QQQuery: "10001",
		UserID:  userID,
	})
	require.NoError(t, err)

	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  1,
		ReturnURL: "https://join.stuhelper.com/verify/test-admission-token?qq=10001",
	})
	require.NoError(t, err)
	_, err = svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 1,
		State:    start.State,
		UserID:   userID,
		Code:     "oidc-code",
	})
	require.NoError(t, err)

	actions, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.AuthURL, actions[0].AuthURL)

	err = svc.RecordBotEvent(context.Background(), actions[0].SessionID, BotEventInput{
		Action:  actions[0].Action,
		Success: true,
	})
	require.NoError(t, err)
	assertAdmissionSessionCancelled(t, pg, created.Session.ID)
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
		ReturnURL: "https://evil.example/verify/token",
	})
	require.ErrorIs(t, err, ErrAdmissionReturnURLNotAllowed)

	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  1,
		ReturnURL: "https://join.stuhelper.com/verify/token?qq=10001",
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
	assert.Equal(t, "https://join.stuhelper.com/verify/token?qq=10001", complete.ReturnURL)
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
		ReturnURL: "https://join.stuhelper.com/verify/token?qq=10001",
	})
	require.NoError(t, err)
	return start
}

type testSchoolEmailSender struct {
	email string
	code  string
	err   error
}

type testAcademicLookupGateway struct {
	student *user.AcademicStudent
	err     error
}

func (g testAcademicLookupGateway) GetAcademicInfo(
	context.Context,
	int64,
	string,
) (*user.AcademicStudent, error) {
	return g.student, g.err
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

func assertAdmissionSessionCancelled(t *testing.T, fixture *postgresfixture.Fixture, sessionID string) {
	t.Helper()
	var cancelled bool
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT cancelled_at IS NOT NULL
		FROM group_admission_sessions
		WHERE id = $1
	`, sessionID).Scan(&cancelled)
	require.NoError(t, err)
	assert.True(t, cancelled)
}
