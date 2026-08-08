package admission

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestBotActionAttemptForDatabaseEnforcesClaimedRange(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		want    int32
		wantErr bool
	}{
		{name: "zero is not a claim", attempt: 0, wantErr: true},
		{name: "first attempt", attempt: 1, want: 1},
		{name: "last attempt", attempt: admissionBotActionMaxAttempts, want: admissionBotActionMaxAttempts},
		{name: "above retry budget", attempt: admissionBotActionMaxAttempts + 1, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := botActionAttemptForDatabase(test.attempt)
			if test.wantErr {
				require.ErrorIs(t, err, ErrAdmissionInvalidInput)
				assert.Zero(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestPendingAdmissionActionBlacklistsOnFinalFailure(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-final-failure",
		QQID:      "10001",
		TokenHash: "token-hash-final-failure",
		Status:    StatusJoinedMuted,
	})
	expireAdmissionLinkWait(t, fixture, "adm-final-failure")
	insertAdmissionFailureCount(t, fixture, DefaultFailedJoinLimit-1)

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionBlacklist, actions[0].Action)
}

func TestNewAdmissionSessionWaitsBeforeFirstPendingReminder(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	assert.Empty(t, actions)

	svc.now = func() time.Time {
		return fixedAdmissionNow().Add(initialReminderGrace + time.Second)
	}
	actions, err = svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRemind, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
}

func TestPendingAdmissionActionsRespectLimit(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-pending-limit-1",
		QQID:      "10001",
		TokenHash: "token-hash-pending-limit-1",
		Status:    StatusJoinedMuted,
	})
	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-pending-limit-2",
		QQID:      "10002",
		TokenHash: "token-hash-pending-limit-2",
		Status:    StatusJoinedMuted,
	})
	expireAdmissionLinkWait(t, fixture, "adm-pending-limit-1")
	expireAdmissionLinkWait(t, fixture, "adm-pending-limit-2")

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514", Limit: 1},
	)

	require.NoError(t, err)
	require.Len(t, actions, 1)
}

func TestPendingAdmissionActionsRequireBotIdentity(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)

	_, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{})

	require.ErrorIs(t, err, ErrAdmissionPendingActionFilterInvalid)
}

func TestNormalizePendingActionFilterRejectsOversizedFields(t *testing.T) {
	_, err := normalizePendingActionFilter(AdmissionPendingActionFilter{
		Platform:  strings.Repeat("p", maxAdmissionPendingActionFilterRunes+1),
		BotSelfID: "514",
	})
	require.ErrorIs(t, err, ErrAdmissionPendingActionFilterInvalid)

	_, err = normalizePendingActionFilter(AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: strings.Repeat("b", maxAdmissionPendingActionFilterRunes+1),
	})
	require.ErrorIs(t, err, ErrAdmissionPendingActionFilterInvalid)
}

func TestLinkedAdmissionSessionDoesNotReleaseBeforeStudentVerification(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-no-student-proof")

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)

	require.NoError(t, err)
	assert.Empty(t, actions)

	_, err = svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)

	actions, err = svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)

	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
}

func TestQueuedAdmissionActionReleaseAckCompletesSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-queued-release")

	_, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)

	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.NotEmpty(t, actions[0].ActionID)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)

	err = svc.RecordBotActionEvent(context.Background(), actions[0].ActionID, BotEventInput{
		Action:          BotAction(" release "),
		Success:         true,
		DispatchAttempt: actions[0].DispatchAttempt,
		MessageID:       " release-msg-1 ",
	})
	require.NoError(t, err)

	var status string
	var cancelled bool
	var messageID string
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT o.status, s.cancelled_at IS NOT NULL, COALESCE(o.message_id, '')
		FROM admission_bot_action_outbox AS o
		JOIN group_admission_sessions AS s ON s.id = o.session_id
		WHERE o.id = $1::bigint
	`, actions[0].ActionID).Scan(&status, &cancelled, &messageID)
	require.NoError(t, err)
	assert.Equal(t, "succeeded", status)
	assert.True(t, cancelled)
	assert.Equal(t, "release-msg-1", messageID)
}

func TestQueuedAdmissionActionUpsertPreservesActiveDispatchLease(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-active-dispatch")

	verified, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)
	firstClaim, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, firstClaim, 1)
	assert.Equal(t, 1, firstClaim[0].DispatchAttempt)

	err = svc.repo.WithTx(context.Background(), func(ctx context.Context, tx pgx.Tx) error {
		return svc.queueBotActionTx(ctx, tx, verified, BotActionRelease, svc.now(), svc.now())
	})
	require.NoError(t, err)

	secondClaim, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	assert.Empty(t, secondClaim)

	var status string
	var attemptCount int
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT status, attempt_count
		FROM admission_bot_action_outbox
		WHERE id = $1::bigint
	`, firstClaim[0].ActionID).Scan(&status, &attemptCount)
	require.NoError(t, err)
	assert.Equal(t, string(AdmissionBotActionDispatched), status)
	assert.Equal(t, 1, attemptCount)
}

func TestQueuedAdmissionActionLateAckCannotFinalizeNewDispatchAttempt(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-late-dispatch-ack")

	_, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)
	firstClaim, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, firstClaim, 1)
	assert.Equal(t, 1, firstClaim[0].DispatchAttempt)

	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE admission_bot_action_outbox
		SET next_attempt_at = $2
		WHERE id = $1::bigint
	`, firstClaim[0].ActionID, svc.now().Add(-time.Second))
	require.NoError(t, err)
	secondClaim, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, secondClaim, 1)
	assert.Equal(t, 2, secondClaim[0].DispatchAttempt)

	err = svc.RecordBotActionEvent(context.Background(), firstClaim[0].ActionID, BotEventInput{
		Action:          BotActionRelease,
		Success:         true,
		DispatchAttempt: firstClaim[0].DispatchAttempt,
	})
	require.NoError(t, err)
	assertAdmissionSessionStatus(t, fixture, created.Session.ID, StatusVerified)

	var status string
	var attemptCount int
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT status, attempt_count
		FROM admission_bot_action_outbox
		WHERE id = $1::bigint
	`, firstClaim[0].ActionID).Scan(&status, &attemptCount)
	require.NoError(t, err)
	assert.Equal(t, string(AdmissionBotActionDispatched), status)
	assert.Equal(t, secondClaim[0].DispatchAttempt, attemptCount)

	err = svc.RecordBotActionEvent(context.Background(), secondClaim[0].ActionID, BotEventInput{
		Action:          BotActionRelease,
		Success:         true,
		DispatchAttempt: secondClaim[0].DispatchAttempt,
	})
	require.NoError(t, err)
	assertAdmissionSessionCancelled(t, fixture, created.Session.ID)
}

func TestQueuedAdmissionActionDeadLettersTimedOutDispatchAfterMaxAttempts(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-queued-release-timeout")

	_, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)

	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.NotEmpty(t, actions[0].ActionID)
	actionID := actions[0].ActionID

	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE admission_bot_action_outbox
		SET attempt_count = $2,
		    next_attempt_at = $3,
		    updated_at = $3
		WHERE id = $1::bigint
	`, actionID, admissionBotActionMaxAttempts, fixedAdmissionNow().Add(-time.Second))
	require.NoError(t, err)

	actions, err = svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	assert.Empty(t, actions)

	var status string
	var attemptCount int
	var lastError string
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT status, attempt_count, COALESCE(last_error, '')
		FROM admission_bot_action_outbox
		WHERE id = $1::bigint
	`, actionID).Scan(&status, &attemptCount, &lastError)
	require.NoError(t, err)
	assert.Equal(t, "dead_letter", status)
	assert.Equal(t, admissionBotActionMaxAttempts, attemptCount)
	assert.Equal(t, "bot action dispatch timed out", lastError)
}

func TestQueuedAdmissionActionSkipsStaleKickAfterLink(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	userID := seedAdmissionUser(t, fixture, "queued-stale-kick")
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	svc.now = func() time.Time {
		return created.Session.LinkWaitDeadlineAt.Add(time.Second)
	}

	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	assert.Empty(t, actions)

	var staleCount int
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM admission_bot_action_outbox
		WHERE session_id = $1 AND action = 'kick' AND status = 'stale'
	`, created.Session.ID).Scan(&staleCount)
	require.NoError(t, err)
	assert.Equal(t, 1, staleCount)
}

func TestQueuedAdmissionActionSkipsStaleReminderAfterBotSkip(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)

	_, err = svc.ResendAdminAdmissionSession(context.Background(), AdminAdmissionSessionActionInput{
		SessionID:      created.Session.ID,
		OperatorUserID: 9001,
	})
	require.NoError(t, err)
	_, err = svc.SkipBotAdmissionSession(context.Background(), BotSessionOperatorInput{
		Platform:     "qq",
		GuildID:      "guild-1",
		QQID:         "10001",
		OperatorQQID: "90001",
	})
	require.NoError(t, err)

	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	assert.Empty(t, actions)

	var staleCount int
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM admission_bot_action_outbox
		WHERE session_id = $1 AND action = 'remind' AND status = 'stale'
	`, created.Session.ID).Scan(&staleCount)
	require.NoError(t, err)
	assert.Equal(t, 1, staleCount)
}

func TestClaimQueuedAdmissionActionsUsesFixedDatabaseRoundTrips(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)

	testCases := []struct {
		name      string
		botSelfID string
		rowCount  int
	}{
		{name: "one row", botSelfID: "fixed-query-one", rowCount: 1},
		{name: "eight rows", botSelfID: "fixed-query-many", rowCount: 8},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < tc.rowCount; i++ {
				insertQueuedBotActionForTest(t, fixture, queuedBotActionTestSeed{
					SessionID:     fmt.Sprintf("fixed-query-%s-%02d", tc.botSelfID, i),
					BotSelfID:     tc.botSelfID,
					GuildID:       "fixed-query-guild",
					QQID:          fmt.Sprintf("%s-%02d", tc.botSelfID, i),
					SessionStatus: StatusVerified,
					Action:        BotActionRelease,
					ScheduledAt:   fixedAdmissionNow().Add(-time.Minute),
				})
			}

			before := fixture.Pool.Stat().AcquireCount()
			actions, err := svc.ClaimQueuedAdmissionActions(
				context.Background(),
				AdmissionPendingActionFilter{
					Platform:  "qq",
					BotSelfID: tc.botSelfID,
					Limit:     tc.rowCount,
				},
			)
			acquisitions := fixture.Pool.Stat().AcquireCount() - before

			require.NoError(t, err)
			require.Len(t, actions, tc.rowCount)
			assert.Equal(t, int64(3), acquisitions)
		})
	}
}

func TestClaimQueuedAdmissionActionsReleasesBatchWhenContextLookupFails(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	actionIDs := make([]int64, 0, 2)
	for i := 0; i < 2; i++ {
		actionIDs = append(actionIDs, insertQueuedBotActionForTest(t, fixture, queuedBotActionTestSeed{
			SessionID:     fmt.Sprintf("context-failure-%d", i),
			BotSelfID:     "context-failure",
			GuildID:       "context-failure-guild",
			QQID:          fmt.Sprintf("context-failure-%d", i),
			SessionStatus: StatusVerified,
			Action:        BotActionRelease,
			ScheduledAt:   fixedAdmissionNow().Add(-time.Minute),
		}))
	}

	_, err := fixture.Pool.Exec(context.Background(), `
		ALTER TABLE group_admission_policies
		RENAME TO group_admission_policies_context_failure
	`)
	require.NoError(t, err)
	policyTableRenamed := true
	t.Cleanup(func() {
		if policyTableRenamed {
			_, _ = fixture.Pool.Exec(context.Background(), `
				ALTER TABLE group_admission_policies_context_failure
				RENAME TO group_admission_policies
			`)
		}
	})

	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "context-failure", Limit: 2},
	)
	require.Error(t, err)
	assert.Nil(t, actions)

	_, err = fixture.Pool.Exec(context.Background(), `
		ALTER TABLE group_admission_policies_context_failure
		RENAME TO group_admission_policies
	`)
	require.NoError(t, err)
	policyTableRenamed = false

	for _, actionID := range actionIDs {
		state := readQueuedBotActionState(t, fixture, actionID)
		assert.Equal(t, AdmissionBotActionPending, state.status)
		assert.Zero(t, state.attemptCount)
		assert.WithinDuration(
			t,
			fixedAdmissionNow().Add(botActionDispatchRetryAfter),
			state.nextAttemptAt,
			time.Millisecond,
		)
	}

	retryNow := fixedAdmissionNow().Add(botActionDispatchRetryAfter + time.Second)
	svc.now = func() time.Time { return retryNow }
	actions, err = svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "context-failure", Limit: 2},
	)
	require.NoError(t, err)
	assert.Len(t, actions, 2)
}

func TestClaimQueuedAdmissionActionsReleasesBatchAfterCallerCancellation(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	actionID := insertQueuedBotActionForTest(t, fixture, queuedBotActionTestSeed{
		SessionID:     "context-cancelled",
		BotSelfID:     "context-cancelled",
		GuildID:       "context-cancelled-guild",
		QQID:          "context-cancelled",
		SessionStatus: StatusVerified,
		Action:        BotActionRelease,
		ScheduledAt:   fixedAdmissionNow().Add(-time.Minute),
	})

	lockTx, err := fixture.Pool.Begin(context.Background())
	require.NoError(t, err)
	defer func() { _ = lockTx.Rollback(context.Background()) }()
	_, err = lockTx.Exec(context.Background(), `
		LOCK TABLE group_admission_policies IN ACCESS EXCLUSIVE MODE
	`)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type claimResult struct {
		actions []AdmissionPendingAction
		err     error
	}
	resultCh := make(chan claimResult, 1)
	go func() {
		actions, claimErr := svc.ClaimQueuedAdmissionActions(
			ctx,
			AdmissionPendingActionFilter{
				Platform:  "qq",
				BotSelfID: "context-cancelled",
				Limit:     1,
			},
		)
		resultCh <- claimResult{actions: actions, err: claimErr}
	}()

	require.Eventually(t, func() bool {
		var status AdmissionBotActionStatus
		err := fixture.Pool.QueryRow(context.Background(), `
			SELECT status
			FROM admission_bot_action_outbox
			WHERE id = $1
		`, actionID).Scan(&status)
		return err == nil && status == AdmissionBotActionDispatched
	}, 3*time.Second, 10*time.Millisecond)
	cancel()

	var result claimResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("claim did not return after caller cancellation")
	}
	require.ErrorIs(t, result.err, context.Canceled)
	assert.Nil(t, result.actions)
	state := readQueuedBotActionState(t, fixture, actionID)
	assert.Equal(t, AdmissionBotActionPending, state.status)
	assert.Zero(t, state.attemptCount)
}

func TestQueuedAdmissionActionFinalizersRespectAttemptFence(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	actionID := insertQueuedBotActionForTest(t, fixture, queuedBotActionTestSeed{
		SessionID:     "attempt-fence",
		BotSelfID:     "attempt-fence",
		GuildID:       "attempt-fence-guild",
		QQID:          "attempt-fence",
		SessionStatus: StatusVerified,
		Action:        BotActionRelease,
		ScheduledAt:   fixedAdmissionNow().Add(-time.Minute),
	})
	filter := AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "attempt-fence",
		Limit:     1,
	}

	firstClaim, err := svc.repo.ClaimDueBotActions(context.Background(), filter, fixedAdmissionNow())
	require.NoError(t, err)
	require.Len(t, firstClaim, 1)
	assert.Equal(t, 1, firstClaim[0].AttemptCount)

	reclaimNow := fixedAdmissionNow().Add(botActionDispatchRetryAfter + time.Second)
	secondClaim, err := svc.repo.ClaimDueBotActions(context.Background(), filter, reclaimNow)
	require.NoError(t, err)
	require.Len(t, secondClaim, 1)
	assert.Equal(t, 2, secondClaim[0].AttemptCount)

	err = svc.repo.MarkBotActionPreparationFailed(
		context.Background(),
		actionID,
		firstClaim[0].AttemptCount,
		"old preparation failure",
		reclaimNow,
	)
	require.ErrorIs(t, err, ErrAdmissionBotActionLeaseLost)
	err = svc.repo.MarkBotActionStale(
		context.Background(),
		actionID,
		firstClaim[0].AttemptCount,
		reclaimNow,
	)
	require.ErrorIs(t, err, ErrAdmissionBotActionLeaseLost)
	affected, err := svc.repo.abandonBotActionClaims(
		context.Background(),
		firstClaim,
		reclaimNow,
	)
	require.NoError(t, err)
	assert.Zero(t, affected)

	state := readQueuedBotActionState(t, fixture, actionID)
	assert.Equal(t, AdmissionBotActionDispatched, state.status)
	assert.Equal(t, secondClaim[0].AttemptCount, state.attemptCount)

	err = svc.repo.MarkBotActionStale(
		context.Background(),
		actionID,
		secondClaim[0].AttemptCount,
		reclaimNow,
	)
	require.NoError(t, err)
	assert.Equal(t, AdmissionBotActionStale, readQueuedBotActionState(t, fixture, actionID).status)
}

func TestClaimQueuedAdmissionActionsIsolatesPoisonRow(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	badActionID := insertQueuedBotActionForTest(t, fixture, queuedBotActionTestSeed{
		SessionID:     "poison-kick",
		BotSelfID:     "poison-row",
		GuildID:       "missing-policy-guild",
		QQID:          "poison-kick",
		SessionStatus: StatusJoinedMuted,
		Action:        BotActionKick,
		ScheduledAt:   fixedAdmissionNow().Add(-2 * time.Minute),
		LinkDeadline:  fixedAdmissionNow().Add(-time.Minute),
	})
	goodActionID := insertQueuedBotActionForTest(t, fixture, queuedBotActionTestSeed{
		SessionID:     "healthy-release",
		BotSelfID:     "poison-row",
		GuildID:       "missing-policy-guild",
		QQID:          "healthy-release",
		SessionStatus: StatusVerified,
		Action:        BotActionRelease,
		ScheduledAt:   fixedAdmissionNow().Add(-time.Minute),
	})

	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "poison-row", Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, "healthy-release", actions[0].SessionID)
	assert.Equal(t, AdmissionBotActionDispatched, readQueuedBotActionState(t, fixture, goodActionID).status)

	badState := readQueuedBotActionState(t, fixture, badActionID)
	assert.Equal(t, AdmissionBotActionFailed, badState.status)
	assert.Equal(t, 1, badState.attemptCount)
	assert.Contains(t, badState.lastError, ErrAdmissionPolicyNotFound.Error())

	err = svc.RecordBotActionEvent(context.Background(), actions[0].ActionID, BotEventInput{
		Action:          BotActionRelease,
		Success:         true,
		DispatchAttempt: actions[0].DispatchAttempt,
	})
	require.NoError(t, err)

	retryNow := fixedAdmissionNow()
	for expectedAttempt := 2; expectedAttempt <= admissionBotActionMaxAttempts; expectedAttempt++ {
		badState = readQueuedBotActionState(t, fixture, badActionID)
		retryNow = badState.nextAttemptAt.Add(time.Second)
		svc.now = func() time.Time { return retryNow }
		actions, err = svc.ClaimQueuedAdmissionActions(
			context.Background(),
			AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "poison-row", Limit: 2},
		)
		require.NoError(t, err)
		assert.Empty(t, actions)

		badState = readQueuedBotActionState(t, fixture, badActionID)
		assert.Equal(t, expectedAttempt, badState.attemptCount)
		if expectedAttempt < admissionBotActionMaxAttempts {
			assert.Equal(t, AdmissionBotActionFailed, badState.status)
		} else {
			assert.Equal(t, AdmissionBotActionDeadLetter, badState.status)
		}
	}
	assert.Equal(t, AdmissionBotActionSucceeded, readQueuedBotActionState(t, fixture, goodActionID).status)
}

func TestTruncateBotActionPreparationErrorPreservesUTF8(t *testing.T) {
	message := strings.Repeat("错误", maxBotActionPreparationErrorBytes)

	truncated := truncateBotActionPreparationError(fmt.Errorf("%s", message))

	assert.LessOrEqual(t, len(truncated), maxBotActionPreparationErrorBytes)
	assert.True(t, utf8.ValidString(truncated))
	assert.NotEmpty(t, truncated)
}

func TestClaimedAdmissionReminderAckAfterBotSkipIsStale(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)

	_, err = svc.ResendAdminAdmissionSession(context.Background(), AdminAdmissionSessionActionInput{
		SessionID:      created.Session.ID,
		OperatorUserID: 9001,
	})
	require.NoError(t, err)
	actions, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.NotEmpty(t, actions[0].ActionID)
	assert.Equal(t, BotActionRemind, actions[0].Action)

	_, err = svc.SkipBotAdmissionSession(context.Background(), BotSessionOperatorInput{
		Platform:     "qq",
		GuildID:      "guild-1",
		QQID:         "10001",
		OperatorQQID: "90001",
	})
	require.NoError(t, err)
	err = svc.RecordBotActionEvent(context.Background(), actions[0].ActionID, BotEventInput{
		Action:          BotActionRemind,
		Success:         true,
		DispatchAttempt: actions[0].DispatchAttempt,
		MessageID:       "stale-reminder-message",
	})
	require.NoError(t, err)

	var outboxStatus string
	var sessionStatus string
	var nextReminderQueued bool
	var messageID string
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT o.status, s.status, s.next_reminder_at IS NOT NULL, COALESCE(o.message_id, '')
		FROM admission_bot_action_outbox AS o
		JOIN group_admission_sessions AS s ON s.id = o.session_id
		WHERE o.id = $1::bigint
	`, actions[0].ActionID).Scan(&outboxStatus, &sessionStatus, &nextReminderQueued, &messageID)
	require.NoError(t, err)
	assert.Equal(t, "stale", outboxStatus)
	assert.Equal(t, string(StatusCancelled), sessionStatus)
	assert.False(t, nextReminderQueued)
	assert.Empty(t, messageID)
}

func TestStudentVerificationProjectionReleasesLinkedSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-main-student-proof")
	require.NotNil(t, created.Session.UserID)

	err := svc.ProjectStudentVerification(context.Background(), *created.Session.UserID, 4111010006, true)
	require.NoError(t, err)

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
}

func TestStudentVerificationProjectionDoesNotReleaseExpiredLinkedSessions(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-expired-projection")
	require.NotNil(t, created.Session.UserID)
	expiredID := insertExpiredLinkedAdmissionSessionForUser(t, fixture, *created.Session.UserID)

	err := svc.ProjectStudentVerification(context.Background(), *created.Session.UserID, 4111010006, true)
	require.NoError(t, err)

	assertAdmissionSessionStatus(t, fixture, created.Session.ID, StatusVerified)
	assertAdmissionSessionStatus(t, fixture, expiredID, StatusLinked)

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 2)
	assert.ElementsMatch(t, []BotAction{BotActionRelease, BotActionKick}, []BotAction{
		actions[0].Action,
		actions[1].Action,
	})
}

func TestStudentVerificationProjectionDoesNotReleaseOtherSchoolSessions(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	insertAdmissionPolicyForSchool(t, fixture, "adm-policy-other-school", "guild-2", 4111010007)
	userID := seedAdmissionUser(t, fixture, "linked-other-school-projection")
	sameSchool := createLinkedSessionForSubject(t, svc, userID, "guild-1", "10001", "linked-same-school-token")
	otherSchool := createLinkedSessionForSubject(t, svc, userID, "guild-2", "10002", "linked-other-school-token")

	err := svc.ProjectStudentVerification(context.Background(), userID, 4111010006, true)
	require.NoError(t, err)

	assertAdmissionSessionStatus(t, fixture, sameSchool.ID, StatusVerified)
	assertAdmissionSessionStatus(t, fixture, otherSchool.ID, StatusLinked)

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, sameSchool.ID, actions[0].SessionID)
}

func TestLinkedAdmissionSessionTimesOutInsteadOfReleaseWithoutStudentVerification(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "linked-timeout-no-student-proof")
	svc.now = func() time.Time {
		return fixedAdmissionNow().Add(25 * time.Hour)
	}

	actions, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)

	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, BotActionKick, actions[0].Action)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
}

func TestPendingAdmissionKickActionRequiresPolicy(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-missing-policy",
		QQID:      "10001",
		TokenHash: "token-hash-missing-policy",
		Status:    StatusJoinedMuted,
	})
	expireAdmissionLinkWait(t, fixture, "adm-missing-policy")
	deleteAdmissionPolicies(t, fixture)

	_, err := svc.ListPendingAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)

	require.ErrorIs(t, err, ErrAdmissionPolicyNotFound)
}

func createBotLinkedSessionForPendingActions(
	t *testing.T,
	svc *Service,
	fixture *postgresfixture.Fixture,
	userSuffix string,
) *CreatedAdmissionSession {
	t.Helper()

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	userID := seedAdmissionUser(t, fixture, userSuffix)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	created.Session = linked
	return created
}

func createLinkedSessionForSubject(
	t *testing.T,
	svc *Service,
	userID int64,
	guildID string,
	qqID string,
	token string,
) *AdmissionSession {
	t.Helper()
	previousGenerateToken := svc.generateToken
	svc.generateToken = func() (string, error) { return token, nil }
	defer func() { svc.generateToken = previousGenerateToken }()

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   guildID,
		ChannelID: "channel-1",
		QQID:      qqID,
	})
	require.NoError(t, err)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	return linked
}

func insertAdmissionPolicyForSchool(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	id string,
	guildID string,
	schoolID int64,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO schools (id, code, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, schoolID, "4111010007", "Other School")
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_policies (
			id, platform, guild_id, school_id, management_guild_ids, freshman_channel_closes_at, freshman_default_expires_at
		)
		VALUES ($1, 'qq', $2, $3, $4, $5, $6)
	`, id, guildID, schoolID, []string{"mgmt-1"}, futureTime(30), futureTime(180))
	require.NoError(t, err)
}

func expireAdmissionLinkWait(t *testing.T, fixture *postgresfixture.Fixture, sessionID string) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_sessions
		SET link_wait_deadline_at = $2
		WHERE id = $1
	`, sessionID, fixedAdmissionNow().Add(-time.Minute))
	require.NoError(t, err)
}

func insertExpiredLinkedAdmissionSessionForUser(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	userID int64,
) string {
	t.Helper()

	id := "adm-expired-linked-projection"
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, user_id, token_hash,
			token_expires_at, token_consumed_at, status, link_wait_deadline_at,
			submission_wait_deadline_at, initial_mute_until
		)
		VALUES ($1, 'qq', '514', 'guild-1', 'channel-1', '10002', $2, 'token-hash-expired-linked-projection',
			$3, $4, $5, $6, $7, $8)
	`, id, userID, futureTime(1), fixedAdmissionNow().Add(-2*time.Hour), StatusLinked,
		futureTime(1), fixedAdmissionNow().Add(-time.Minute), futureTime(30))
	require.NoError(t, err)
	return id
}

func insertAdmissionFailureCount(t *testing.T, fixture *postgresfixture.Fixture, count int) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_failures (platform, guild_id, qq_id, failure_count)
		VALUES ('qq', 'guild-1', '10001', $1)
	`, count)
	require.NoError(t, err)
}

type queuedBotActionTestSeed struct {
	SessionID     string
	BotSelfID     string
	GuildID       string
	QQID          string
	SessionStatus AdmissionSessionStatus
	Action        BotAction
	ScheduledAt   time.Time
	LinkDeadline  time.Time
}

func insertQueuedBotActionForTest(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	seed queuedBotActionTestSeed,
) int64 {
	t.Helper()
	linkDeadline := seed.LinkDeadline
	if linkDeadline.IsZero() {
		linkDeadline = fixedAdmissionNow().Add(time.Hour)
	}
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, token_hash,
			token_expires_at, status, link_wait_deadline_at,
			submission_wait_deadline_at, initial_mute_until, eligibility_revision
		)
		VALUES (
			$1, 'qq', $2, $3, 'channel-1', $4, $5,
			$6, $7, $8, $9, $10, CASE WHEN $7 = 'action_pending' THEN 1 END
		)
	`, seed.SessionID, seed.BotSelfID, seed.GuildID, seed.QQID,
		"token-hash-"+seed.SessionID, fixedAdmissionNow().Add(24*time.Hour),
		seed.SessionStatus, linkDeadline, fixedAdmissionNow().Add(2*time.Hour),
		fixedAdmissionNow().Add(30*time.Minute))
	require.NoError(t, err)

	var actionID int64
	err = fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO admission_bot_action_outbox (
			action_key, session_id, action, platform, bot_self_id, guild_id,
			channel_id, qq_id, scheduled_at, status, attempt_count,
			next_attempt_at, eligibility_revision, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, 'qq', $4, $5,
			'channel-1', $6, $7, 'pending', 0,
			$8, CASE WHEN $3 = 'release' THEN 1 END, $8, $8
		)
		RETURNING id
	`, "test:"+seed.SessionID+":"+string(seed.Action), seed.SessionID, seed.Action,
		seed.BotSelfID, seed.GuildID, seed.QQID, seed.ScheduledAt,
		fixedAdmissionNow().Add(-time.Minute)).Scan(&actionID)
	require.NoError(t, err)
	return actionID
}

type queuedBotActionState struct {
	status        AdmissionBotActionStatus
	attemptCount  int
	nextAttemptAt time.Time
	lastError     string
}

func readQueuedBotActionState(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	actionID int64,
) queuedBotActionState {
	t.Helper()
	var state queuedBotActionState
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT status, attempt_count, next_attempt_at, COALESCE(last_error, '')
		FROM admission_bot_action_outbox
		WHERE id = $1
	`, actionID).Scan(
		&state.status,
		&state.attemptCount,
		&state.nextAttemptAt,
		&state.lastError,
	)
	require.NoError(t, err)
	return state
}
