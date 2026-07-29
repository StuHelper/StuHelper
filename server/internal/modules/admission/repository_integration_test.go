package admission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestAdmissionMigration(t *testing.T) {
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

	applicationID := insertFreshmanApplication(t, fixture, userID, sessionID)
	insertFreshmanMaterial(t, fixture, applicationID)
	assertPendingApplicationUnique(t, fixture, userID, sessionID)
	assertFreshmanCredentialRequiresExpiry(t, fixture, userID)
	insertAdmissionFailure(t, fixture)

	var policyCount int
	err := fixture.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM group_admission_policies").Scan(&policyCount)
	require.NoError(t, err)
	require.Equal(t, 1, policyCount)
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
	require.Equal(t, AdmissionPolicyTarget{
		PolicyID:             created.ID,
		Platform:             "qq",
		GuildID:              "guild-2",
		GuardEnabled:         true,
		JoinHandlingStrategy: AdmissionJoinHandlingPostJoinGuard,
		LinkWaitSeconds:      DefaultLinkWaitSeconds,
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
			status, link_wait_deadline_at, submission_wait_deadline_at, initial_mute_until
		)
		VALUES ($1, 'qq', '514', 'guild-1', 'channel-1', $2, $3, $4, $5, $6, $7, $8)
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
			status, link_wait_deadline_at, submission_wait_deadline_at, initial_mute_until
		)
		VALUES ($1, $2, $3, $4, 'channel-1', $5, $6, $7, $8, $9, $10, $11)
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

func insertFreshmanApplication(t *testing.T, fixture *postgresfixture.Fixture, userID int64, sessionID string) string {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_verification_applications (
			id, user_id, school_id, admission_session_id, applicant_name, applicant_name_masked, material_type, status
		)
		VALUES ('freshman-app-1', $1, 4111010006, $2, 'Alice Applicant', 'A***', 'admission_notice', $3)
	`, userID, sessionID, FreshmanApplicationPending)
	require.NoError(t, err)
	return "freshman-app-1"
}

func insertFreshmanMaterial(t *testing.T, fixture *postgresfixture.Fixture, applicationID string) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_verification_materials (
			id, application_id, object_key, content_type, size_bytes, sha256
		)
		VALUES ('freshman-material-1', $1, 'admission/freshman/material-1.jpg', 'image/jpeg', 1200, repeat('a', 64))
	`, applicationID)
	require.NoError(t, err)
}

func assertPendingApplicationUnique(t *testing.T, fixture *postgresfixture.Fixture, userID int64, sessionID string) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_verification_applications (
			id, user_id, school_id, admission_session_id, applicant_name, applicant_name_masked, material_type, status
		)
		VALUES ('freshman-app-dup', $1, 4111010006, $2, 'Bob Applicant', 'B***', 'admission_certificate', $3)
	`, userID, sessionID, FreshmanApplicationPending)
	require.Error(t, err)
}

func assertFreshmanCredentialRequiresExpiry(t *testing.T, fixture *postgresfixture.Fixture, userID int64) {
	t.Helper()

	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_verification_credentials (id, user_id, school_id, kind, subject_hash, subject_display)
		VALUES ('credential-no-expiry', $1, 4111010006, $2, 'manual-hash-1', 'freshman material')
	`, userID, CredentialFreshmanMaterialManual)
	require.Error(t, err)

	_, err = fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_verification_credentials (id, user_id, school_id, kind, subject_hash, subject_display, expires_at)
		VALUES ('credential-with-expiry', $1, 4111010006, $2, 'manual-hash-2', 'freshman material', $3)
	`, userID, CredentialFreshmanMaterialManual, futureTime(90))
	require.NoError(t, err)
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
