package admission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

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

func expireAdmissionLinkWait(t *testing.T, fixture *postgresfixture.Fixture, sessionID string) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_sessions
		SET link_wait_deadline_at = $2
		WHERE id = $1
	`, sessionID, fixedAdmissionNow().Add(-time.Minute))
	require.NoError(t, err)
}

func insertAdmissionFailureCount(t *testing.T, fixture *postgresfixture.Fixture, count int) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_failures (platform, guild_id, qq_id, failure_count)
		VALUES ('qq', 'guild-1', '10001', $1)
	`, count)
	require.NoError(t, err)
}
