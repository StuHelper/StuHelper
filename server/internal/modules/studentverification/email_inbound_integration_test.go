package studentverification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type staticInboundEmailTarget struct {
	address string
	err     error
}

func (r staticInboundEmailTarget) TargetAddress(context.Context, string) (string, error) {
	return r.address, r.err
}

func TestInboundEmailChallengeRequiresExactAuthenticatedSenderAndIsReplaySafe(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-inbound-integration-key")
	userID := seedVerificationUser(t, fixture, "inbound-email")
	configureRosterImport(t, fixture, now)
	configureInboundEmailMethod(t, fixture, now)
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
		WithInboundEmailTargetResolver(staticInboundEmailTarget{address: "verify@inbound.stuhelper.test"}),
	)
	require.NoError(t, err)

	snapshot, err := service.ImportFullRoster(ctx, validRosterImportInput(now))
	require.NoError(t, err)
	_, err = service.ActivateRosterSnapshot(ctx, RosterSnapshotSwitchInput{
		SchoolCode: testSchoolCode, SnapshotID: snapshot.ID, ActorUserID: userID,
		Reason: "enable inbound email integration roster",
	})
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: userID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	challenge, err := service.CreateInboundEmailChallenge(ctx, StudentEmailIdentityInput{
		UserID: userID, ApplicationID: application.ID, StudentID: "20990001", Name: "张三",
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "waiting", challenge.Status)
	assert.Equal(t, "verify@inbound.stuhelper.test", challenge.TargetAddress)
	assert.Equal(t, "20****01@buaa.edu.cn", challenge.ExpectedSenderMasked)
	assert.NotEmpty(t, challenge.ChallengeValue)

	var persisted string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT concat_ws(' ', expected_sender_hash, encode(challenge_value_enc, 'hex'))
		FROM student_email_inbound_challenges WHERE id = $1
	`, challenge.ID).Scan(&persisted))
	assert.NotContains(t, persisted, "20990001@buaa.edu.cn")
	assert.NotContains(t, persisted, challenge.ChallengeValue)

	err = service.ProcessInboundEmailEvent(ctx, InboundEmailEvent{
		EventReference: "provider-event-wrong-sender", EnvelopeFrom: "alias@buaa.edu.cn",
		HeaderFrom: "alias@buaa.edu.cn", Subject: challenge.Subject,
		TextBody: inboundEmailBodyPrefix + challenge.ChallengeValue,
		SPFPass:  true, DKIMPass: true, DMARCPass: true, ReceivedAt: now,
	})
	require.NoError(t, err)
	challengeState, err := service.GetInboundEmailChallenge(ctx, userID, application.ID)
	require.NoError(t, err)
	assert.Equal(t, "waiting", challengeState.Status)

	event := InboundEmailEvent{
		EventReference: "provider-event-correct", EnvelopeFrom: "20990001@buaa.edu.cn",
		HeaderFrom: "20990001@buaa.edu.cn", Subject: challenge.Subject,
		TextBody: "Please verify\n" + inboundEmailBodyPrefix + challenge.ChallengeValue + "\n",
		SPFPass:  true, DKIMPass: true, DMARCPass: true, ReceivedAt: now,
	}
	require.NoError(t, service.ProcessInboundEmailEvent(ctx, event))
	require.NoError(t, service.ProcessInboundEmailEvent(ctx, event))

	challengeState, err = service.GetInboundEmailChallenge(ctx, userID, application.ID)
	require.NoError(t, err)
	assert.Equal(t, "verified", challengeState.Status)
	assert.Empty(t, challengeState.ChallengeValue)
	application, err = service.GetApplication(ctx, userID, application.ID)
	require.NoError(t, err)
	require.NotNil(t, application.Credential)
	assert.Equal(t, MethodStudentEmailInbound, application.Credential.Method)

	var correctEventCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_email_inbound_events
		WHERE result_code = 'verified'
	`).Scan(&correctEventCount))
	assert.Equal(t, 1, correctEventCount)
}

func configureInboundEmailMethod(t *testing.T, fixture *postgresfixture.Fixture, now time.Time) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_methods
		SET enabled = true, validation_status = 'valid', validated_at = $2,
		    health_status = 'healthy', health_checked_at = $2,
		    privacy_notice_version = '2026-08-05',
		    privacy_notice = '{
		      "title":"学校邮箱发信验证",
		      "summary":"从规范学号邮箱发送一次性挑战邮件",
		      "dataCategories":["学号","姓名","学校邮箱"],
		      "retentionSummary":"挑战短期保存并在完成后失效"
		    }'::jsonb,
		    risk_policy = '{"inboundTTLSeconds":600}'::jsonb,
		    updated_at = $2
		WHERE school_id = $1 AND method = 'student_email_inbound_challenge'
	`, testSchoolID, now)
	require.NoError(t, err)
}
