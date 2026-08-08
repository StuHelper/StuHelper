package admission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type mutableStudentEligibilityGateway struct {
	decision StudentEligibilityDecision
	err      error
}

func (g *mutableStudentEligibilityGateway) EvaluateStudentEligibility(
	context.Context,
	int64,
	int64,
) (StudentEligibilityDecision, error) {
	return g.decision, g.err
}

func TestEligibilityRevisionSupersedesClaimedReleaseBeforeAck(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "eligibility-fence")
	require.NotNil(t, created.Session.UserID)

	gateway := &mutableStudentEligibilityGateway{
		decision: StudentEligibilityDecision{
			Eligible:        true,
			CredentialClass: "formal_student",
			Revision:        7,
		},
	}
	svc.studentEligibility = gateway

	err := svc.ReevaluateStudentEligibility(
		context.Background(),
		*created.Session.UserID,
		4111010006,
		7,
	)
	require.NoError(t, err)

	claimed, err := svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, BotActionRelease, claimed[0].Action)

	gateway.decision = StudentEligibilityDecision{
		Eligible:        false,
		CredentialClass: "formal_student",
		Revision:        8,
	}
	err = svc.ReevaluateStudentEligibility(
		context.Background(),
		*created.Session.UserID,
		4111010006,
		8,
	)
	require.NoError(t, err)

	// A delayed success from the superseded dispatch attempt is absorbed and
	// cannot promote the session to admitted.
	err = svc.RecordBotActionEvent(context.Background(), claimed[0].ActionID, BotEventInput{
		Action:          BotActionRelease,
		Success:         true,
		DispatchAttempt: claimed[0].DispatchAttempt,
	})
	require.NoError(t, err)

	var status AdmissionSessionStatus
	var revision int64
	var actionStatus AdmissionBotActionStatus
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT session.status, session.eligibility_revision, action.status
		FROM group_admission_sessions AS session
		JOIN admission_bot_action_outbox AS action ON action.session_id = session.id
		WHERE session.id = $1 AND action.action = 'release'
	`, created.Session.ID).Scan(&status, &revision, &actionStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusLinked, status)
	assert.Equal(t, int64(8), revision)
	assert.Equal(t, AdmissionBotActionStale, actionStatus)

	claimed, err = svc.ClaimQueuedAdmissionActions(
		context.Background(),
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "514"},
	)
	require.NoError(t, err)
	assert.Empty(t, claimed)
}

func TestEligibilityConsumerRecomputesCurrentRevisionForOldEvent(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "eligibility-latest")
	require.NotNil(t, created.Session.UserID)

	gateway := &mutableStudentEligibilityGateway{
		decision: StudentEligibilityDecision{
			Eligible:        true,
			CredentialClass: "formal_student",
			Revision:        12,
		},
	}
	svc.studentEligibility = gateway

	// The delivered event is revision 9, but the consumer must use the current
	// revision 12 decision returned by the authoritative service boundary.
	err := svc.ReevaluateStudentEligibility(
		context.Background(),
		*created.Session.UserID,
		4111010006,
		9,
	)
	require.NoError(t, err)

	var sessionRevision int64
	var actionRevision int64
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT session.eligibility_revision, action.eligibility_revision
		FROM group_admission_sessions AS session
		JOIN admission_bot_action_outbox AS action ON action.session_id = session.id
		WHERE session.id = $1 AND action.action = 'release' AND action.status = 'pending'
	`, created.Session.ID).Scan(&sessionRevision, &actionRevision)
	require.NoError(t, err)
	assert.Equal(t, int64(12), sessionRevision)
	assert.Equal(t, int64(12), actionRevision)
}

func TestTemporaryFreshmanEligibilityRequiresExplicitGroupPolicy(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createBotLinkedSessionForPendingActions(t, svc, fixture, "temporary-policy")
	require.NotNil(t, created.Session.UserID)

	gateway := &mutableStudentEligibilityGateway{
		decision: StudentEligibilityDecision{
			Eligible:        true,
			CredentialClass: "temporary_freshman",
			Revision:        20,
		},
	}
	svc.studentEligibility = gateway

	// Policies default to fail-closed for temporary freshman credentials.
	require.NoError(t, svc.ReevaluateStudentEligibility(
		context.Background(),
		*created.Session.UserID,
		4111010006,
		20,
	))
	assertAdmissionEligibilityState(t, fixture, created.Session.ID, StatusLinked, 20, 0)

	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_policies
		SET allow_temporary_freshman = TRUE
		WHERE id = 'adm-policy-1'
	`)
	require.NoError(t, err)
	require.NoError(t, svc.ReevaluateStudentEligibility(
		context.Background(),
		*created.Session.UserID,
		4111010006,
		20,
	))
	assertAdmissionEligibilityState(t, fixture, created.Session.ID, StatusVerified, 20, 1)

	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_policies
		SET allow_temporary_freshman = FALSE
		WHERE id = 'adm-policy-1'
	`)
	require.NoError(t, err)
	gateway.decision.Revision = 21
	require.NoError(t, svc.ReevaluateStudentEligibility(
		context.Background(),
		*created.Session.UserID,
		4111010006,
		21,
	))
	assertAdmissionEligibilityState(t, fixture, created.Session.ID, StatusLinked, 21, 0)
}

func assertAdmissionEligibilityState(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	sessionID string,
	wantStatus AdmissionSessionStatus,
	wantRevision int64,
	wantPendingReleaseActions int,
) {
	t.Helper()
	var status AdmissionSessionStatus
	var revision int64
	var pendingReleaseActions int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT session.status,
		       session.eligibility_revision,
		       COUNT(action.id) FILTER (
		           WHERE action.action = 'release'
		             AND action.status IN ('pending', 'failed', 'dispatched', 'dead_letter')
		       )
		FROM group_admission_sessions AS session
		LEFT JOIN admission_bot_action_outbox AS action ON action.session_id = session.id
		WHERE session.id = $1
		GROUP BY session.id
	`, sessionID).Scan(&status, &revision, &pendingReleaseActions)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, status)
	assert.Equal(t, wantRevision, revision)
	assert.Equal(t, wantPendingReleaseActions, pendingReleaseActions)
}
