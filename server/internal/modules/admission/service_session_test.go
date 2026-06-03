package admission

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestAdmissionSessionCreatePreviewAndMismatch(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.Token)
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token", created.AuthURL)
	assert.Equal(t, StatusJoinedMuted, created.Session.Status)
	assert.Equal(t, "514", created.Session.BotSelfID)
	assert.Equal(t, svc.now().Add(time.Hour), created.Session.LinkWaitDeadlineAt)

	preview, err := svc.PreviewToken(context.Background(), created.Token, "")
	require.NoError(t, err)
	assert.Equal(t, created.Session.ID, preview.ID)
	assert.Nil(t, preview.TokenConsumedAt)

	_, err = svc.PreviewToken(context.Background(), created.Token, "10001")
	require.ErrorIs(t, err, ErrAdmissionQQMismatch)
}

func TestCreateBotSessionRequiresConfiguredPolicy(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)

	_, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})

	require.ErrorIs(t, err, ErrAdmissionPolicyNotFound)
}

func TestCreateBotSessionRequiresBotSelfID(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)

	_, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001",
	})

	require.ErrorIs(t, err, ErrAdmissionInvalidInput)
}

func TestResolveJoinRequestDecisionUsesVerifiedAndUnverifiedPolicy(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)

	decision, err := svc.ResolveJoinRequestDecision(context.Background(), AdmissionJoinRequestDecisionInput{
		Platform: "qq", GuildID: "guild-1", QQID: "10001", RequestID: "request-1",
	})
	require.NoError(t, err)
	assert.Equal(t, AdmissionJoinRequestDecisionApprove, decision.Decision)
	assert.Equal(t, AdmissionJoinRequestUnverified, decision.VerificationState)
	assert.Equal(t, "unverified_auto_approve", decision.Reason)

	setAdmissionPolicyAutoApproval(t, fixture, true, false)
	decision, err = svc.ResolveJoinRequestDecision(context.Background(), AdmissionJoinRequestDecisionInput{
		Platform: "qq", GuildID: "guild-1", QQID: "10001", RequestID: "request-1",
	})
	require.NoError(t, err)
	assert.Equal(t, AdmissionJoinRequestDecisionReject, decision.Decision)
	assert.Equal(t, AdmissionJoinRequestUnverified, decision.VerificationState)
	assert.Equal(t, "unverified_auto_approve_disabled", decision.Reason)

	userID := seedAdmissionUser(t, fixture, "verified-join-decision")
	bindVerifiedAdmissionQQ(t, fixture, userID, "10001")
	decision, err = svc.ResolveJoinRequestDecision(context.Background(), AdmissionJoinRequestDecisionInput{
		Platform: "qq", GuildID: "guild-1", QQID: "10001", RequestID: "request-1",
	})
	require.NoError(t, err)
	assert.Equal(t, AdmissionJoinRequestDecisionApprove, decision.Decision)
	assert.Equal(t, AdmissionJoinRequestVerified, decision.VerificationState)
	assert.Equal(t, "verified_auto_approve", decision.Reason)
	require.NotNil(t, decision.UserID)
	assert.Equal(t, fmt.Sprint(userID), *decision.UserID)

	setAdmissionPolicyAutoApproval(t, fixture, false, true)
	decision, err = svc.ResolveJoinRequestDecision(context.Background(), AdmissionJoinRequestDecisionInput{
		Platform: "qq", GuildID: "guild-1", QQID: "10001", RequestID: "request-1",
	})
	require.NoError(t, err)
	assert.Equal(t, AdmissionJoinRequestDecisionReject, decision.Decision)
	assert.Equal(t, AdmissionJoinRequestVerified, decision.VerificationState)
	assert.Equal(t, "verified_auto_approve_disabled", decision.Reason)
}

func TestCreateBotSessionReturnsVerifiedForAlreadyCertifiedQQ(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	userID := seedAdmissionUser(t, fixture, "verified-create-session")
	bindVerifiedAdmissionQQ(t, fixture, userID, "10001")
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_failures (platform, guild_id, qq_id, failure_count)
		VALUES ('qq', 'guild-1', '10001', 2)
	`)
	require.NoError(t, err)

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})

	require.NoError(t, err)
	assert.Equal(t, StatusVerified, created.Session.Status)
	assert.Equal(t, fmt.Sprint(userID), admissionSessionUserID(t, created.Session))
	assert.NotNil(t, created.Session.TokenConsumedAt)
	assert.NotNil(t, created.Session.VerifiedAt)
	assert.Nil(t, created.Session.CancelledAt)
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token", created.AuthURL)
	assertAdmissionFailureCount(t, fixture, "10001", 0)

	actions, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)

	err = svc.RecordBotEvent(context.Background(), actions[0].SessionID, BotEventInput{
		Action:  actions[0].Action,
		Success: true,
	})
	require.NoError(t, err)
	assertAdmissionSessionCancelled(t, fixture, created.Session.ID)
}

func TestCreateBotSessionReusesActiveSessionOnDuplicateJoin(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	first := createLinkableSession(t, svc)
	svc.generateToken = func() (string, error) { return "test-admission-token-duplicate", nil }

	second, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})

	require.NoError(t, err)
	assert.Equal(t, first.Session.ID, second.Session.ID)
	assert.Equal(t, first.AuthURL, second.AuthURL)
	assert.Equal(t, first.Token, second.Token)
	assert.Equal(t, 1, countAdmissionSessionsBySubject(t, fixture, "qq", "guild-1", "10001"))
}

func TestCreateBotSessionReusesMaterialSubmittedSessionOnDuplicateJoin(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	first := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "duplicate-material")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  first.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	_, err = svc.MarkMaterialSubmitted(context.Background(), first.Session.ID)
	require.NoError(t, err)
	svc.generateToken = func() (string, error) { return "test-admission-token-material-duplicate", nil }

	second, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})

	require.NoError(t, err)
	assert.Equal(t, first.Session.ID, second.Session.ID)
	assert.Equal(t, first.AuthURL, second.AuthURL)
	assert.Equal(t, first.Token, second.Token)
	assert.Equal(t, 1, countAdmissionSessionsBySubject(t, fixture, "qq", "guild-1", "10001"))
}

func TestAdmissionTokenLinkIsAtomicUnderConcurrency(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	firstUser := seedAdmissionUser(t, fixture, "link-first")
	secondUser := seedAdmissionUser(t, fixture, "link-second")

	results := linkTokenConcurrently(t, svc, created.Token, firstUser, secondUser)
	require.Len(t, results, 2)
	assertExactlyOneLinkSuccess(t, results)
}

func TestAdmissionTokenLinkMapsQQBindingConflictsToQQMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "user bound to other qq", err: user.ErrQQBindingUserConflict},
		{name: "qq bound to other user", err: user.ErrQQBindingQQAlreadyBound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := postgresfixture.Start(t)
			svc := newSessionTestService(t, fixture)
			svc.qqGateway = &testQQBindingGateway{err: tc.err}
			insertAdmissionPolicy(t, fixture)
			created := createLinkableSession(t, svc)
			userID := seedAdmissionUser(t, fixture, "qq-binding-conflict")

			_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
				Token:  created.Token,
				UserID: userID,
			})

			require.ErrorIs(t, err, ErrAdmissionQQMismatch)
			require.NotErrorIs(t, err, tc.err)
		})
	}
}

func TestAdmissionTokenExpiredAndConsumedErrors(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(2 * time.Hour) }
	_, err := svc.PreviewToken(context.Background(), created.Token, "")
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)

	svc = newSessionTestService(t, fixture)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: seedAdmissionUser(t, fixture, "late-link"),
	})
	require.NoError(t, err)
	assert.Equal(t, StatusLinked, linked.Status)

	_, err = svc.PreviewToken(context.Background(), created.Token, "")
	require.ErrorIs(t, err, ErrAdmissionTokenConsumed)
}

func TestAdmissionTokenLinkIsIdempotentForSameUser(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "idempotent-link")

	first, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	require.NotNil(t, first.UserID)
	assert.Equal(t, StatusLinked, first.Status)

	second, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	require.NotNil(t, second.UserID)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, StatusLinked, second.Status)
	assert.Equal(t, userID, *second.UserID)
}

func TestAdmissionTokenConsumedResumeRejectsExpiredSubmissionWindow(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "idempotent-link-expired")

	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)
}

func TestAdmissionTokenConsumedResumeAllowsSubmittedAndVerifiedSessions(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "idempotent-link-submitted")

	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	_, err = svc.MarkMaterialSubmitted(context.Background(), created.Session.ID)
	require.NoError(t, err)

	submitted, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, StatusMaterialSubmitted, submitted.Status)

	_, err = svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)
	verified, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, StatusVerified, verified.Status)
}

func TestAdmissionTokenLinkConsumedByAnotherUserStillRejected(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	firstUser := seedAdmissionUser(t, fixture, "first-link-owner")
	secondUser := seedAdmissionUser(t, fixture, "second-link-owner")

	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: firstUser,
	})
	require.NoError(t, err)

	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: secondUser,
	})
	require.ErrorIs(t, err, ErrAdmissionTokenConsumed)
}

func TestBotAdmissionSessionQueryAndResendLinkedSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "bot-resend-linked")

	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	require.Equal(t, StatusLinked, linked.Status)

	queried, err := svc.GetBotAdmissionSession(context.Background(), BotSessionSubjectInput{
		Platform: "qq",
		GuildID:  "guild-1",
		QQID:     "10001",
	})
	require.NoError(t, err)
	assert.Equal(t, created.Session.ID, queried.ID)
	assert.Equal(t, StatusLinked, queried.Status)

	resent, err := svc.ResendBotAdmissionSession(context.Background(), BotSessionSubjectInput{
		Platform: "qq",
		GuildID:  "guild-1",
		QQID:     "10001",
	})
	require.NoError(t, err)
	assert.Equal(t, created.AuthURL, resent.AuthURL)
	assert.Equal(t, StatusLinked, resent.Status)
}

func TestAdminAdmissionSessionActionsQueueBotReminderAndCancel(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "admin-session-actions")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(initialReminderGrace + time.Second) }
	actions, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
		Limit:     5,
	})
	require.NoError(t, err)
	assert.Empty(t, actions)

	queued, err := svc.ResendAdminAdmissionSession(context.Background(), AdminAdmissionSessionActionInput{
		SessionID:      created.Session.ID,
		OperatorUserID: 9001,
	})
	require.NoError(t, err)
	assert.Equal(t, StatusLinked, queued.Status)

	actions, err = svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
		Limit:     5,
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRemind, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
	assert.True(t, actions[0].DeadlineAt.Equal(queued.SubmissionWaitDeadlineAt))

	cancelled, err := svc.CancelAdminAdmissionSession(context.Background(), AdminAdmissionSessionActionInput{
		SessionID:      created.Session.ID,
		OperatorUserID: 9001,
	})
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, cancelled.Status)
	assertAdmissionSessionStatus(t, fixture, created.Session.ID, StatusCancelled)
}

func TestRegenerateBotAdmissionSessionCancelsInProgressSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("test-admission-token-%d", tokenIndex), nil
	}
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "bot-regenerate")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	regenerated, err := svc.RegenerateBotAdmissionSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})
	require.NoError(t, err)
	require.NotEqual(t, created.Session.ID, regenerated.Session.ID)
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token-2", regenerated.AuthURL)
	assert.Equal(t, StatusJoinedMuted, regenerated.Session.Status)
	assertAdmissionSessionStatus(t, fixture, created.Session.ID, StatusCancelled)

	_, err = svc.PreviewToken(context.Background(), created.Token, "")
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)

	latest, err := svc.GetBotAdmissionSession(context.Background(), BotSessionSubjectInput{
		Platform: "qq",
		GuildID:  "guild-1",
		QQID:     "10001",
	})
	require.NoError(t, err)
	assert.Equal(t, regenerated.Session.ID, latest.ID)
}

func TestRegenerateBotAdmissionSessionRejectsVerifiedSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "bot-regenerate-verified")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	_, err = svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)

	_, err = svc.RegenerateBotAdmissionSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})
	require.ErrorIs(t, err, ErrAdmissionInvalidStatus)
}

func TestRegenerateAdminAdmissionSessionCreatesImmediateReminder(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("test-admission-token-%d", tokenIndex), nil
	}
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "admin-regenerate")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	regenerated, err := svc.RegenerateAdminAdmissionSession(context.Background(), AdminAdmissionSessionActionInput{
		SessionID:      created.Session.ID,
		OperatorUserID: 9001,
	})
	require.NoError(t, err)
	require.NotEqual(t, created.Session.ID, regenerated.Session.ID)
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token-2", regenerated.AuthURL)
	assert.Equal(t, StatusJoinedMuted, regenerated.Session.Status)
	assertAdmissionSessionStatus(t, fixture, created.Session.ID, StatusCancelled)

	actions, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
		Limit:     5,
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRemind, actions[0].Action)
	assert.Equal(t, regenerated.Session.ID, actions[0].SessionID)
	assert.Equal(t, regenerated.AuthURL, actions[0].AuthURL)
	assert.True(t, actions[0].DeadlineAt.Equal(regenerated.Session.LinkWaitDeadlineAt))
}

func TestAdmissionSessionStatusTransitions(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "status-link")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	submitted, err := svc.MarkMaterialSubmitted(context.Background(), created.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusMaterialSubmitted, submitted.Status)
	require.NotNil(t, submitted.ManualReviewDeadlineAt)

	verified, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusVerified, verified.Status)
	require.NotNil(t, verified.VerifiedAt)
}

func TestLinkedSessionSubmissionDeadlineBlocksMaterialAndVerification(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "status-link-expired")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }

	_, err = svc.MarkMaterialSubmitted(context.Background(), created.Session.ID)
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)

	_, err = svc.MarkVerified(context.Background(), created.Session.ID)
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)
}

func TestMaterialSubmittedManualReviewDeadlineBlocksVerification(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "manual-review-expired")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	_, err = svc.MarkMaterialSubmitted(context.Background(), created.Session.ID)
	require.NoError(t, err)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }

	_, err = svc.MarkVerified(context.Background(), created.Session.ID)
	require.ErrorIs(t, err, ErrAdmissionInvalidStatus)
}

func TestLinkedAdmissionActionsRequirePolicyForTimeoutsButNotRelease(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "missing-policy-link")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	deleteAdmissionPolicies(t, fixture)

	_, err = svc.MarkMaterialSubmitted(context.Background(), created.Session.ID)
	require.ErrorIs(t, err, ErrAdmissionPolicyNotFound)

	verified, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusVerified, verified.Status)

	err = svc.RecordBotEvent(context.Background(), created.Session.ID, BotEventInput{
		Action:  BotActionRelease,
		Success: true,
	})
	require.NoError(t, err)
	assertAdmissionSessionCancelled(t, fixture, created.Session.ID)
}

func TestSuccessfulBotEventClearsLastBotError(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)

	remindSession := createLinkableSessionForQQ(t, svc, "10001", "test-admission-token-remind")
	err := svc.RecordBotEvent(context.Background(), remindSession.Session.ID, BotEventInput{
		Action: BotActionRemind, Success: false, Error: "send failed",
	})
	require.NoError(t, err)
	assert.Equal(t, "send failed", *admissionSessionLastBotError(t, fixture, remindSession.Session.ID))
	err = svc.RecordBotEvent(context.Background(), remindSession.Session.ID, BotEventInput{
		Action: BotActionRemind, Success: true,
	})
	require.NoError(t, err)
	assert.Nil(t, admissionSessionLastBotError(t, fixture, remindSession.Session.ID))

	releaseSession := createLinkableSessionForQQ(t, svc, "10002", "test-admission-token-release")
	userID := seedAdmissionUser(t, fixture, "release-clears-bot-error")
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token: releaseSession.Token, UserID: userID,
	})
	require.NoError(t, err)
	_, err = svc.MarkVerified(context.Background(), releaseSession.Session.ID)
	require.NoError(t, err)
	err = svc.RecordBotEvent(context.Background(), releaseSession.Session.ID, BotEventInput{
		Action: BotActionRelease, Success: false, Error: "unmute failed",
	})
	require.NoError(t, err)
	assert.Equal(t, "unmute failed", *admissionSessionLastBotError(t, fixture, releaseSession.Session.ID))
	err = svc.RecordBotEvent(context.Background(), releaseSession.Session.ID, BotEventInput{
		Action: BotActionRelease, Success: true,
	})
	require.NoError(t, err)
	assert.Nil(t, admissionSessionLastBotError(t, fixture, releaseSession.Session.ID))

	kickSession := createLinkableSessionForQQ(t, svc, "10003", "test-admission-token-kick")
	err = svc.RecordBotEvent(context.Background(), kickSession.Session.ID, BotEventInput{
		Action: BotActionKick, Success: false, Error: "kick failed",
	})
	require.NoError(t, err)
	assert.Equal(t, "kick failed", *admissionSessionLastBotError(t, fixture, kickSession.Session.ID))
	err = svc.RecordBotEvent(context.Background(), kickSession.Session.ID, BotEventInput{
		Action: BotActionKick, Success: true,
	})
	require.NoError(t, err)
	assert.Nil(t, admissionSessionLastBotError(t, fixture, kickSession.Session.ID))
}

func TestReleaseSuccessResetsAdmissionFailureCount(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	insertAdmissionFailureCount(t, fixture, DefaultFailedJoinLimit-1)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "release-resets-failures")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token: created.Token, UserID: userID,
	})
	require.NoError(t, err)
	_, err = svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)

	err = svc.RecordBotEvent(context.Background(), created.Session.ID, BotEventInput{
		Action: BotActionRelease, Success: true,
	})

	require.NoError(t, err)
	assertAdmissionFailureCount(t, fixture, "10001", 0)
}

func TestAdmissionFailureBlacklistFromKickEvent(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("test-admission-token-%d", tokenIndex), nil
	}

	for i := 0; i < DefaultFailedJoinLimit; i++ {
		created := createLinkableSession(t, svc)
		err := svc.RecordBotEvent(context.Background(), created.Session.ID, BotEventInput{
			Action: BotActionKick, Success: true,
		})
		require.NoError(t, err)
	}

	assertAdmissionFailureBlacklisted(t, fixture, "10001")
}

func TestDuplicateKickEventDoesNotDoubleCountFailure(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)

	for i := 0; i < 2; i++ {
		err := svc.RecordBotEvent(context.Background(), created.Session.ID, BotEventInput{
			Action: BotActionKick, Success: true,
		})
		require.NoError(t, err)
	}

	var failureCount int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT failure_count
		FROM group_admission_failures
		WHERE platform = 'qq' AND guild_id = 'guild-1' AND qq_id = '10001'
	`).Scan(&failureCount)
	require.NoError(t, err)
	assert.Equal(t, 1, failureCount)
	assert.Equal(t, 0, countMemberBlacklistEntries(t, fixture))
}

type linkResult struct {
	session *AdmissionSession
	err     error
}

func linkTokenConcurrently(t *testing.T, svc *Service, token string, users ...int64) []linkResult {
	t.Helper()
	var wg sync.WaitGroup
	results := make([]linkResult, len(users))
	for i, userID := range users {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()
			session, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
				Token:  token,
				UserID: id,
			})
			results[index] = linkResult{session: session, err: err}
		}(i, userID)
	}
	wg.Wait()
	return results
}

func assertExactlyOneLinkSuccess(t *testing.T, results []linkResult) {
	t.Helper()
	var success int
	var consumed int
	for _, result := range results {
		if result.err == nil {
			success++
			assert.Equal(t, StatusLinked, result.session.Status)
			continue
		}
		if errors.Is(result.err, ErrAdmissionTokenConsumed) {
			consumed++
		}
	}
	assert.Equal(t, 1, success)
	assert.Equal(t, 1, consumed)
}

func createLinkableSession(t *testing.T, svc *Service) *CreatedAdmissionSession {
	t.Helper()
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})
	require.NoError(t, err)
	return created
}

func createLinkableSessionForQQ(t *testing.T, svc *Service, qqID string, token string) *CreatedAdmissionSession {
	t.Helper()
	previousGenerateToken := svc.generateToken
	svc.generateToken = func() (string, error) { return token, nil }
	defer func() { svc.generateToken = previousGenerateToken }()
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: qqID, BotSelfID: "514",
	})
	require.NoError(t, err)
	return created
}

func linkAdmissionSessionForQQ(
	t *testing.T,
	svc *Service,
	userID int64,
	qqID string,
	token string,
) *AdmissionSession {
	t.Helper()
	created := createLinkableSessionForQQ(t, svc, qqID, token)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	return linked
}

func newSessionTestService(t *testing.T, fixture *postgresfixture.Fixture) *Service {
	t.Helper()
	svc, err := NewService(NewRepository(fixture.DB), &testQQBindingGateway{}, []byte("test-admission-hmac-key-32-bytes!"))
	require.NoError(t, err)
	svc.now = fixedAdmissionNow
	svc.generateToken = func() (string, error) { return "test-admission-token", nil }
	svc.generateJoinToken = func() (string, error) { return svc.generateToken() }
	return svc
}

func fixedAdmissionNow() time.Time {
	return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
}

type testQQBindingGateway struct {
	err error
}

func (g *testQQBindingGateway) EnsureQQBindingForUserTx(
	context.Context,
	pgx.Tx,
	int64,
	string,
) (*user.QQBinding, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &user.QQBinding{}, nil
}

func assertAdmissionFailureBlacklisted(t *testing.T, fixture *postgresfixture.Fixture, qqID string) {
	t.Helper()
	assertAdmissionFailureCount(t, fixture, qqID, DefaultFailedJoinLimit)
	var failureCount int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM member_blacklist_entries
		WHERE platform = 'qq' AND guild_id = 'guild-1' AND subject_id = $1
		  AND source = 'admission_failure' AND released_at IS NULL
	`, qqID).Scan(&failureCount)
	require.NoError(t, err)
	assert.Equal(t, 1, failureCount)
}

func setAdmissionPolicyAutoApproval(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	verified bool,
	unverified bool,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_policies
		SET auto_approve_verified_join = $1,
		    auto_approve_unverified_join = $2,
		    auto_approve_join = $1 AND $2
		WHERE platform = 'qq' AND guild_id = 'guild-1'
	`, verified, unverified)
	require.NoError(t, err)
}

func bindVerifiedAdmissionQQ(t *testing.T, fixture *postgresfixture.Fixture, userID int64, qqID string) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_qq_bindings (user_id, qq_id, bound_at)
		VALUES ($1, $2, $3)
	`, userID, qqID, fixedAdmissionNow())
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display, verified_at
		)
		VALUES ($1, $2, 4111010006, $3, $4, $5, $6)
	`, "credential-"+qqID, userID, CredentialSchoolEmailOTP, "credential-hash-"+qqID, "student "+qqID, fixedAdmissionNow())
	require.NoError(t, err)
}

func admissionSessionUserID(t *testing.T, session *AdmissionSession) string {
	t.Helper()
	require.NotNil(t, session)
	require.NotNil(t, session.UserID)
	return fmt.Sprint(*session.UserID)
}

func countAdmissionSessionsBySubject(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	platform string,
	guildID string,
	qqID string,
) int {
	t.Helper()
	var count int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM group_admission_sessions
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
	`, platform, guildID, qqID).Scan(&count)
	require.NoError(t, err)
	return count
}

func assertAdmissionFailureCount(t *testing.T, fixture *postgresfixture.Fixture, qqID string, expected int) {
	t.Helper()
	var failureCount int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT failure_count
		FROM group_admission_failures
		WHERE platform = 'qq' AND guild_id = 'guild-1' AND qq_id = $1
	`, qqID).Scan(&failureCount)
	require.NoError(t, err)
	assert.Equal(t, expected, failureCount)
}

func assertAdmissionSessionStatus(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	sessionID string,
	expected AdmissionSessionStatus,
) {
	t.Helper()
	var status AdmissionSessionStatus
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT status
		FROM group_admission_sessions
		WHERE id = $1
	`, sessionID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, expected, status)
}

func setAdmissionSessionUpdatedAt(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	sessionID string,
	updatedAt time.Time,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_sessions
		SET updated_at = $2
		WHERE id = $1
	`, sessionID, updatedAt)
	require.NoError(t, err)
}

func admissionSessionLastBotError(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	sessionID string,
) *string {
	t.Helper()
	var lastBotError *string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT last_bot_error
		FROM group_admission_sessions
		WHERE id = $1
	`, sessionID).Scan(&lastBotError)
	require.NoError(t, err)
	return lastBotError
}

func deleteAdmissionPolicies(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `DELETE FROM group_admission_policies`)
	require.NoError(t, err)
}
