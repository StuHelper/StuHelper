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

func TestPendingFreshmanForwardsAllowEmptyQueueWithoutMaterialStore(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)

	items, err := svc.ListPendingFreshmanForwards(context.Background())

	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestPendingFreshmanForwardsRequireMaterialStoreWhenQueueHasItems(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionSchoolConfig(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "freshman-forward-requires-store")
	appID := "freshman-forward-requires-store"
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_verification_applications (
			id, user_id, school_id, admission_session_id, status, applicant_name,
			applicant_name_masked, material_type, created_at, updated_at
		)
		VALUES ($1, $2, 4111010006, $3, 'pending', 'Alice Applicant', 'A***',
			'admission_notice', NOW(), NOW())
	`, appID, userID, created.Session.ID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_verification_materials (
			id, application_id, object_key, content_type, size_bytes, sha256, created_at
		)
		VALUES ('material-forward-requires-store', $1, 'admission/material.png',
			'image/png', 12, repeat('a', 64), NOW())
	`, appID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_policies
		SET forward_raw_material_to_qq = TRUE
		WHERE platform = 'qq' AND guild_id = 'guild-1'
	`)
	require.NoError(t, err)

	_, err = svc.ListPendingFreshmanForwards(context.Background())

	require.ErrorIs(t, err, ErrAdmissionMaterialStoreUnavailable)
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
