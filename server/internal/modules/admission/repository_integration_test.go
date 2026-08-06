package admission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestAdmissionTargetSchemaRejectsRetiredVerificationWriters(t *testing.T) {
	fixture := postgresfixture.Start(t)
	userID := seedAdmissionUser(t, fixture, "admission-migration")
	ctx := context.Background()

	insertAdmissionPolicy(t, fixture)
	sessionID := insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-session-1",
		QQID:      "10001",
		TokenHash: "token-hash-1",
		Status:    StatusJoinedMuted,
	})
	assertTokenHashUnique(t, fixture)
	assertActiveSessionPartialUnique(t, fixture)

	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO user_profiles (user_id, verification_status)
		VALUES ($1, 'verified')
	`, userID)
	require.Error(t, err)

	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO freshman_verification_applications (
			id, user_id, school_id, admission_session_id, applicant_name,
			applicant_name_masked, material_type, status
		)
		VALUES ('retired-freshman-app', $1, 4111010006, $2, 'Retired', 'R***',
			'admission_notice', 'pending')
	`, userID, sessionID)
	require.Error(t, err)

	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display,
			status, credential_class, roster_dependency, assurance,
			verified_at, activated_at
		)
		VALUES (
			'00000000-0000-4000-8000-000000000090', $1, 4111010006,
			'school_sso', repeat('9', 64), 'retired credential',
			'active', 'formal_student', 'independent', 'standard', NOW(), NOW()
		)
	`, userID)
	require.Error(t, err)

	applicationID := "00000000-0000-4000-8000-000000000091"
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO student_verification_applications (
			id, user_id, school_id, status, current_method,
			privacy_notice_version, consented_at, expires_at, completed_at
		)
		VALUES ($1, $2, 4111010006, 'approved', 'school_sso',
			'privacy-v1', NOW(), NOW() + INTERVAL '1 hour', NOW())
	`, applicationID, userID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display,
			verification_application_id, status, credential_class,
			roster_dependency, assurance, verified_at, activated_at
		)
		VALUES (
			'00000000-0000-4000-8000-000000000092', $1, 4111010006,
			'school_sso', repeat('a', 64), 'target credential', $2,
			'active', 'formal_student', 'independent', 'standard', NOW(), NOW()
		)
	`, userID, applicationID)
	require.NoError(t, err)

	insertAdmissionFailure(t, fixture)

	var policyCount, qualifyingCredentialCount int
	err = fixture.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM group_admission_policies").Scan(&policyCount)
	require.NoError(t, err)
	require.Equal(t, 1, policyCount)
	err = fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM current_student_qualifying_credentials WHERE user_id = $1
	`, userID).Scan(&qualifyingCredentialCount)
	require.NoError(t, err)
	require.Equal(t, 1, qualifyingCredentialCount)
}

func TestListAdmissionSessionsFiltersByRuntimeSubject(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	insertAdmissionSessionForFilter(t, fixture, admissionSessionFilterSeed{
		ID:        "adm-filter-1",
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		QQID:      "10001",
		TokenHash: "token-hash-filter-1",
		Status:    StatusLinked,
	})
	insertAdmissionSessionForFilter(t, fixture, admissionSessionFilterSeed{
		ID:        "adm-filter-2",
		Platform:  "qq",
		BotSelfID: "999",
		GuildID:   "guild-1",
		QQID:      "10001",
		TokenHash: "token-hash-filter-2",
		Status:    StatusCancelled,
	})
	insertAdmissionSessionForFilter(t, fixture, admissionSessionFilterSeed{
		ID:        "adm-filter-3",
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-2",
		QQID:      "10002",
		TokenHash: "token-hash-filter-3",
		Status:    StatusVerified,
	})

	items, total, err := repo.ListSessions(ctx, AdmissionSessionListFilter{
		Status:    StatusLinked,
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		QQID:      "10001",
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "adm-filter-1", items[0].ID)

	items, total, err = repo.ListSessions(ctx, AdmissionSessionListFilter{
		Platform: "qq", QQID: "10001", PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, items, 2)
}

func TestCreateAdmissionPolicyFromSourceForNewTargetGuild(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	insertAdmissionPolicy(t, fixture)

	created, err := repo.CreatePolicyFromSource(ctx, AdmissionPolicyCreateRequest{
		SourcePolicyID: "adm-policy-1",
		Platform:       "qq",
		GuildID:        "guild-2",
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "qq", created.Platform)
	require.Equal(t, "guild-2", created.GuildID)
	require.True(t, created.GuardEnabled)
	require.Equal(t, int64(4111010006), created.SchoolID)
	require.Equal(t, 2592000, created.InitialMuteDurationSeconds)
	require.Empty(t, created.ManagementGuildIDs)

	targets, err := repo.ListPolicyTargets(ctx)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	require.Equal(t, []string{"mgmt-1"}, targets[0].ManagementGuildIDs)
	require.Equal(t, AdmissionPolicyTarget{
		PolicyID:             created.ID,
		Platform:             "qq",
		GuildID:              "guild-2",
		GuardEnabled:         true,
		JoinHandlingStrategy: AdmissionJoinHandlingPostJoinGuard,
		LinkWaitSeconds:      DefaultLinkWaitSeconds,
		ManagementGuildIDs:   []string{},
	}, targets[1])

	_, err = repo.CreatePolicyFromSource(ctx, AdmissionPolicyCreateRequest{
		SourcePolicyID: "adm-policy-1",
		Platform:       "qq",
		GuildID:        "guild-2",
	})
	require.ErrorIs(t, err, ErrAdmissionPolicyAlreadyExists)
}

func seedAdmissionUser(t *testing.T, fixture *postgresfixture.Fixture, suffix string) int64 {
	t.Helper()

	var id int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "casdoor-admission-"+suffix, "admission_"+suffix, suffix+"@example.test").Scan(&id)
	require.NoError(t, err)
	return id
}

func insertAdmissionPolicy(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_policies (
			id, platform, guild_id, school_id, management_guild_ids, freshman_channel_closes_at, freshman_default_expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "adm-policy-1", "qq", "guild-1", 4111010006, []string{"mgmt-1"}, futureTime(30), futureTime(180))
	require.NoError(t, err)
}

type admissionSessionSeed struct {
	ID        string
	QQID      string
	TokenHash string
	Status    AdmissionSessionStatus
}

func insertAdmissionSession(t *testing.T, fixture *postgresfixture.Fixture, seed admissionSessionSeed) string {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, token_hash, token_expires_at,
			status, link_wait_deadline_at, submission_wait_deadline_at, initial_mute_until,
			eligibility_revision
		)
		VALUES ($1, 'qq', '514', 'guild-1', 'channel-1', $2, $3, $4, $5, $6, $7, $8,
		        CASE WHEN $5 = 'action_pending' THEN 1 END)
	`, seed.ID, seed.QQID, seed.TokenHash, futureTime(1), seed.Status, futureTime(1), futureTime(2), futureTime(30))
	require.NoError(t, err)
	return seed.ID
}

type admissionSessionFilterSeed struct {
	ID        string
	Platform  string
	BotSelfID string
	GuildID   string
	QQID      string
	TokenHash string
	Status    AdmissionSessionStatus
}

func insertAdmissionSessionForFilter(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	seed admissionSessionFilterSeed,
) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, token_hash, token_expires_at,
			status, link_wait_deadline_at, submission_wait_deadline_at, initial_mute_until,
			eligibility_revision
		)
		VALUES ($1, $2, $3, $4, 'channel-1', $5, $6, $7, $8, $9, $10, $11,
		        CASE WHEN $8 = 'action_pending' THEN 1 END)
	`,
		seed.ID, seed.Platform, seed.BotSelfID, seed.GuildID, seed.QQID,
		seed.TokenHash, futureTime(1), seed.Status, futureTime(1),
		futureTime(2), futureTime(30),
	)
	require.NoError(t, err)
}

func assertTokenHashUnique(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, token_hash, token_expires_at,
			status, link_wait_deadline_at, submission_wait_deadline_at, initial_mute_until
		)
		VALUES ('adm-token-dup', 'qq', '514', 'guild-1', 'channel-1', '10002', 'token-hash-1', $1, $2, $3, $4, $5)
	`, futureTime(1), StatusJoinedMuted, futureTime(1), futureTime(2), futureTime(30))
	require.Error(t, err)
}

func assertActiveSessionPartialUnique(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, token_hash, token_expires_at,
			status, link_wait_deadline_at, submission_wait_deadline_at, initial_mute_until
		)
		VALUES ('adm-active-dup', 'qq', '514', 'guild-1', 'channel-1', '10001', 'token-hash-2', $1, $2, $3, $4, $5)
	`, futureTime(1), StatusLinked, futureTime(1), futureTime(2), futureTime(30))
	require.Error(t, err)

	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_sessions SET status = $1 WHERE id = 'adm-session-1'
	`, StatusExpiredKicked)
	require.NoError(t, err)

	insertAdmissionSession(t, fixture, admissionSessionSeed{
		ID:        "adm-session-2",
		QQID:      "10001",
		TokenHash: "token-hash-3",
		Status:    StatusJoinedMuted,
	})
}

func insertAdmissionFailure(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_failures (platform, guild_id, qq_id, failure_count)
		VALUES ('qq', 'guild-1', '10001', 3)
	`)
	require.NoError(t, err)
}

func futureTime(days int) time.Time {
	return time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)
}
