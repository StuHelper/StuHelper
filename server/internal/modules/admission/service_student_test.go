package admission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
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
		SchoolID: 4111010006,
		Email:    "student@other.example",
	})
	require.ErrorIs(t, err, ErrAdmissionEmailDomainNotAllowed)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionEmailDomainNotAllowed)
	assert.Empty(t, sender.email)

	resp, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    " Student@BUAA.edu.cn ",
	})
	require.NoError(t, err)
	assert.Equal(t, admissionEmailOTPCooldownSeconds, resp.CooldownSeconds)
	assert.Equal(t, "student@buaa.edu.cn", sender.email)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
		Code:     sender.code,
	})
	require.NoError(t, err)
	assertCredentialStored(t, pg, userID, CredentialSchoolEmailOTP, "s*****t@buaa.edu.cn")
	assertUserSessionVerified(t, pg, userID)
	assertUserProfileVerified(t, pg, userID, "school_email_otp")
}

func TestAdmissionStudentVerificationRejectsInvalidUserAndSchoolBeforeDependencies(t *testing.T) {
	svc, err := NewService(&Repository{}, &testQQBindingGateway{}, []byte("test-admission-hmac-key-32-bytes!"))
	require.NoError(t, err)

	_, err = svc.MatchSchoolEmailAcademicStudent(context.Background(), SchoolEmailAcademicMatchInput{
		UserID:   0,
		SchoolID: 4111010006,
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.MatchSchoolEmailAcademicStudent(context.Background(), SchoolEmailAcademicMatchInput{
		UserID:   42,
		SchoolID: 0,
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   -1,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   42,
		SchoolID: -1,
		Email:    "student@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   0,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
		Code:     "123456",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   42,
		SchoolID: 0,
		Email:    "student@buaa.edu.cn",
		Code:     "123456",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    -1,
		SchoolID:  4111010006,
		ReturnURL: "https://join.stuhelper.com/verify/token",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    42,
		SchoolID:  -1,
		ReturnURL: "https://join.stuhelper.com/verify/token",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		UserID:   0,
		SchoolID: 4111010006,
		State:    "state",
		Code:     "oidc-code",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		UserID:   42,
		SchoolID: 0,
		State:    "state",
		Code:     "oidc-code",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.storeStudentCredential(context.Background(), studentCredentialInput{
		UserID:   -1,
		SchoolID: 4111010006,
		Kind:     CredentialSchoolEmailOTP,
		Subject:  "student@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)

	_, err = svc.storeStudentCredential(context.Background(), studentCredentialInput{
		UserID:   42,
		SchoolID: -1,
		Kind:     CredentialSchoolEmailOTP,
		Subject:  "student@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidInput)
}

func TestVerifySchoolEmailOTPRejectsMalformedCodeWithoutConsumingAttempt(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-code-format")
	record := admissionEmailOTPRecord{
		Email: "student@buaa.edu.cn",
		Code:  "123456",
	}
	require.NoError(t, svc.storeEmailOTPRecord(context.Background(), emailOTPStoreInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Record:   record,
	}))

	for _, code := range []string{"123", "1234567890123"} {
		_, err := svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
			UserID:   userID,
			SchoolID: 4111010006,
			Email:    record.Email,
			Code:     code,
		})

		require.ErrorIs(t, err, ErrAdmissionInvalidInput)
	}
	attempts, err := redis.Client.Exists(
		context.Background(),
		admissionEmailOTPAttemptsKey(userID, 4111010006),
	).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), attempts)
	loaded, err := svc.loadEmailOTPRecord(context.Background(), userID, 4111010006)
	require.NoError(t, err)
	assert.Equal(t, record, loaded)
}

func TestRequestSchoolEmailOTPSendFailureKeepsCooldown(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{err: errors.New("smtp unavailable")}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-send-fail")
	input := SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
	}

	_, err := svc.RequestSchoolEmailOTP(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp unavailable")
	assert.Equal(t, 1, sender.calls)

	_, err = svc.loadEmailOTPRecord(context.Background(), input.UserID, input.SchoolID)
	require.ErrorIs(t, err, ErrAdmissionOTPExpired)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), input)
	require.ErrorIs(t, err, ErrAdmissionOTPCooldown)
	assert.Equal(t, 1, sender.calls)
}

func TestRequestSchoolEmailOTPSendFailureCleanupSurvivesRequestCancellation(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{err: errors.New("smtp unavailable")}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-send-cancelled")
	input := SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender.onSend = cancel

	_, err := svc.RequestSchoolEmailOTP(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp unavailable")
	assert.Equal(t, 1, sender.calls)
	_, err = svc.loadEmailOTPRecord(context.Background(), input.UserID, input.SchoolID)
	require.ErrorIs(t, err, ErrAdmissionOTPExpired)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), input)
	require.ErrorIs(t, err, ErrAdmissionOTPCooldown)
	assert.Equal(t, 1, sender.calls)
}

func TestVerifySchoolEmailOTPKeepsCodeWhenCredentialCommitFails(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-commit-fail")
	missingSchoolID := int64(999999)
	record := admissionEmailOTPRecord{
		Email: "student@example.edu",
		Code:  "123456",
	}
	require.NoError(t, svc.storeEmailOTPRecord(context.Background(), emailOTPStoreInput{
		UserID:   userID,
		SchoolID: missingSchoolID,
		Record:   record,
	}))

	_, err := svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: missingSchoolID,
		Email:    record.Email,
		Code:     record.Code,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CreateVerificationCredentialTx")

	loaded, err := svc.loadEmailOTPRecord(context.Background(), userID, missingSchoolID)
	require.NoError(t, err)
	assert.Equal(t, record, loaded)
	assertNoCredentialStored(t, pg, userID, CredentialSchoolEmailOTP)
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
		SchoolID:           4111010006,
		AdmissionSessionID: first.ID,
		Email:              "student@buaa.edu.cn",
	})
	require.NoError(t, err)
	verified, err := svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:             userID,
		SchoolID:           4111010006,
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
		student: &AcademicStudent{StudentID: "20250001", Name: stringPtr("张三")},
	}
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-academic")

	_, err := pg.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true,
		    manual_form_fields = '{"admission":{"emailDomains":["buaa.edu.cn"],"emailIdentityPolicy":{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}}}'::jsonb
		WHERE school_id = 4111010006
	`)
	require.NoError(t, err)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "student/id",
		StudentName: "张三",
	})
	require.ErrorIs(t, err, ErrAdmissionStudentIDInvalid)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张\u200b三",
	})
	require.ErrorIs(t, err, ErrAdmissionStudentNameInvalid)

	resp, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: " 张 三 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "20250001@buaa.edu.cn", resp.Email)
	assert.Equal(t, "20250001@buaa.edu.cn", sender.email)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "李四",
	})
	require.ErrorIs(t, err, ErrAdmissionStudentNameMismatch)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    4111010006,
		Email:       "alias@buaa.edu.cn",
		StudentID:   "20250001",
		StudentName: "张三",
	})
	require.ErrorIs(t, err, ErrAdmissionEmailDomainNotAllowed)
}

func TestSchoolEmailOTPAcademicProjectionIncludesStudentProfileFields(t *testing.T) {
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	sender := &testSchoolEmailSender{}
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = sender
	svc.academicLookup = testAcademicLookupGateway{
		student: &AcademicStudent{StudentID: "20250001", Name: stringPtr("张三")},
	}
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-otp-academic-profile")

	_, err := pg.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true,
		    manual_form_fields = '{"admission":{"emailDomains":["buaa.edu.cn"],"emailIdentityPolicy":{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}}}'::jsonb
		WHERE school_id = 4111010006
	`)
	require.NoError(t, err)

	resp, err := svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: " 张 三 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "20250001@buaa.edu.cn", resp.Email)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "20250001@buaa.edu.cn",
		Code:     sender.code,
	})
	require.NoError(t, err)

	assertUserProfileStudentFields(
		t,
		pg,
		userID,
		`["20250001"]`,
		"20250001",
		`{
			"source":"admission_credential",
			"credentialKind":"school_email_otp",
			"subjectDisplay":"2******1@buaa.edu.cn",
			"schoolEmail":"20250001@buaa.edu.cn",
			"studentID":"20250001",
			"studentName":"张三"
		}`,
	)
}

func TestSchoolEmailAcademicMatchReturnsImmediateResult(t *testing.T) {
	pg := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.academicLookup = testAcademicLookupGateway{
		student: &AcademicStudent{StudentID: "20250001", Name: stringPtr("张三")},
	}
	userID := seedLinkedAdmissionUser(t, pg, svc, "email-academic-match")

	_, err := pg.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true,
		    manual_form_fields = '{"admission":{"emailDomains":["buaa.edu.cn"],"emailIdentityPolicy":{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}}}'::jsonb
		WHERE school_id = 4111010006
	`)
	require.NoError(t, err)

	resp, err := svc.MatchSchoolEmailAcademicStudent(context.Background(), SchoolEmailAcademicMatchInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: " 张 三 ",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Matched)
	assert.Equal(t, "20250001@buaa.edu.cn", resp.Email)
	assert.Equal(t, "20250001", resp.StudentID)

	resp, err = svc.MatchSchoolEmailAcademicStudent(context.Background(), SchoolEmailAcademicMatchInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "李四",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Matched)
	assert.Empty(t, resp.Email)
}

func TestSchoolEmailAcademicMatchRequiresLinkedSession(t *testing.T) {
	pg := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	userID := seedAdmissionUser(t, pg, "email-academic-match-unlinked")

	_, err := svc.MatchSchoolEmailAcademicStudent(context.Background(), SchoolEmailAcademicMatchInput{
		UserID:      userID,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张三",
	})
	require.ErrorIs(t, err, ErrAdmissionLinkedSessionRequired)
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
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
	})
	require.NoError(t, err)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
	})
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)

	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 4111010006,
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
		WHERE school_id = 4111010006
	`)
	require.NoError(t, err)

	schoolID, err := svc.ResolveSchoolIDByCode(context.Background(), "4111010006")
	require.NoError(t, err)
	assert.Equal(t, int64(4111010006), schoolID)

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
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token", created.AuthURL)

	userID := seedAdmissionUser(t, pg, "mvp-email-otp")
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	_, err = svc.RequestSchoolEmailOTP(context.Background(), SchoolEmailOTPInput{
		UserID:   userID,
		SchoolID: 4111010006,
		Email:    "student@buaa.edu.cn",
	})
	require.NoError(t, err)
	_, err = svc.VerifySchoolEmailOTP(context.Background(), SchoolEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: 4111010006,
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
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  4111010006,
		ReturnURL: "https://join.stuhelper.com/verify/test-admission-token",
	})
	require.NoError(t, err)
	_, err = svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 4111010006,
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
		SchoolID:  4111010006,
		ReturnURL: "https://evil.example/verify/token",
	})
	require.ErrorIs(t, err, ErrAdmissionReturnURLNotAllowed)

	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  4111010006,
		ReturnURL: "https://join.stuhelper.com/verify/token",
	})
	require.NoError(t, err)
	assert.Contains(t, start.RedirectURL, "https://sso.school.example/login")
	assert.NotEmpty(t, start.State)

	complete, err := svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 4111010006,
		State:    start.State,
		UserID:   userID,
		Code:     "oidc-code",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://join.stuhelper.com/verify/token", complete.ReturnURL)
	assertCredentialStored(t, pg, userID, CredentialSchoolSSO, "official student")
	assertUserSessionVerified(t, pg, userID)
	assertUserProfileVerified(t, pg, userID, "school_sso")

	_, err = svc.CompleteSchoolSSO(context.Background(), SchoolSSOCompleteInput{
		SchoolID: 4111010006,
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
		SchoolID: 4111010006,
		State:    start.State,
		UserID:   userID,
		Code:     "attacker-controlled-code",
	})

	require.ErrorIs(t, err, ErrAdmissionSSONotConfigured)
	assertNoCredentialStored(t, pg, userID, CredentialSchoolSSO)
	assertSchoolSSOStateAvailable(t, svc, userID, start.State)
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
		SchoolID: 4111010006,
		State:    start.State,
		UserID:   userID,
		Code:     "oidc-code",
	})

	require.ErrorIs(t, err, ErrAdmissionSSOIdentityInvalid)
	assertNoCredentialStored(t, pg, userID, CredentialSchoolSSO)
	assertSchoolSSOStateAvailable(t, svc, userID, start.State)
}

func startSchoolSSOForTest(t *testing.T, svc *Service, userID int64) *SchoolSSOStartResult {
	t.Helper()
	start, err := svc.StartSchoolSSO(context.Background(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  4111010006,
		ReturnURL: "https://join.stuhelper.com/verify/token",
	})
	require.NoError(t, err)
	return start
}

type testSchoolEmailSender struct {
	email  string
	code   string
	err    error
	calls  int
	onSend func()
}

type testAcademicLookupGateway struct {
	student *AcademicStudent
	err     error
}

func (g testAcademicLookupGateway) GetAcademicInfo(
	context.Context,
	int64,
	string,
) (*AcademicStudent, error) {
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
	s.calls++
	s.email = email
	s.code = code
	if s.onSend != nil {
		s.onSend()
	}
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

func assertUserProfileVerified(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	method string,
) {
	t.Helper()
	var status string
	var gotMethod string
	var schoolID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT verification_status, verification_method, school_id
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(&status, &gotMethod, &schoolID)
	require.NoError(t, err)
	assert.Equal(t, "verified", status)
	assert.Equal(t, method, gotMethod)
	assert.Equal(t, int64(4111010006), schoolID)
}

func assertUserProfileStudentFields(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
	expectedStudentIDs string,
	expectedActiveStudentID string,
	expectedManualFormData string,
) {
	t.Helper()
	var studentIDs string
	var activeStudentID *string
	var manualFormData string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT student_ids::text, active_student_id, manual_form_data::text
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(&studentIDs, &activeStudentID, &manualFormData)
	require.NoError(t, err)
	assert.JSONEq(t, expectedStudentIDs, studentIDs)
	require.NotNil(t, activeStudentID)
	assert.Equal(t, expectedActiveStudentID, *activeStudentID)
	assert.JSONEq(t, expectedManualFormData, manualFormData)
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

func assertSchoolSSOStateAvailable(t *testing.T, svc *Service, userID int64, state string) {
	t.Helper()
	loaded, err := svc.loadSchoolSSOState(context.Background(), SchoolSSOCompleteInput{
		UserID:   userID,
		SchoolID: 4111010006,
		State:    state,
	})
	require.NoError(t, err)
	assert.Equal(t, userID, loaded.UserID)
	assert.Equal(t, int64(4111010006), loaded.SchoolID)
	assert.NotEmpty(t, loaded.CodeVerifier)
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
