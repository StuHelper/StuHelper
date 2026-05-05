package admission

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestMemberBlacklistAccessPrefersGlobal(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()

	_, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1")))
	require.NoError(t, err)
	global, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGlobal, nil))
	require.NoError(t, err)

	decision, err := svc.GetMemberBlacklistAccess(ctx, memberBlacklistAccessQuery("guild-1"))
	require.NoError(t, err)
	require.NotNil(t, decision.MatchedBlacklist)
	assert.False(t, decision.CanJoin)
	assert.Equal(t, "blocked", decision.Decision)
	assert.Equal(t, global.ID, decision.MatchedBlacklist.ID)
}

func TestMemberBlacklistGuildScopeDoesNotBlockOtherGuild(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()

	_, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1")))
	require.NoError(t, err)

	decision, err := svc.GetMemberBlacklistAccess(ctx, memberBlacklistAccessQuery("guild-2"))
	require.NoError(t, err)
	assert.True(t, decision.CanJoin)
	assert.Equal(t, "allowed", decision.Decision)
	assert.Nil(t, decision.MatchedBlacklist)
}

func TestMemberBlacklistRejectsSourceForWrongEntryPoint(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.Source = BlacklistSourceAdmissionFailure
	input.CreatedFrom = BlacklistCreatedFromAdmissionWorker

	_, err := svc.CreateMemberBlacklistFromBot(context.Background(), input)
	require.ErrorIs(t, err, ErrMemberBlacklistSourceForbidden)
}

func TestMemberBlacklistAllowsBotManualAdminFromQQCommand(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.CreatedByType = BlacklistActorQQOperator
	input.CreatedByID = "90001"
	input.CreatedFrom = BlacklistCreatedFromQQCommand

	entry, err := svc.CreateMemberBlacklistFromBot(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, BlacklistCreatedFromQQCommand, entry.CreatedFrom)
}

func TestMemberBlacklistReleaseBySubjectRequiresScope(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistReleaseBySubjectInput(MemberBlacklistScopeType(""), nil)

	_, err := svc.ReleaseMemberBlacklistBySubjectFromAdmin(context.Background(), input)
	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func TestMemberBlacklistExpiredRowsDoNotBlockAccess(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	past := fixedAdmissionNow().Add(-time.Hour)
	input.ExpiresAt = &past

	_, err := svc.CreateMemberBlacklistFromAdmin(context.Background(), input)
	require.NoError(t, err)

	decision, err := svc.GetMemberBlacklistAccess(context.Background(), memberBlacklistAccessQuery("guild-1"))
	require.NoError(t, err)
	assert.True(t, decision.CanJoin)
	assert.Nil(t, decision.MatchedBlacklist)
}

func TestMemberBlacklistCreateReleasesExpiredRowBeforeInsert(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()
	expired := memberBlacklistTestCreateInput(BlacklistScopeGlobal, nil)
	past := fixedAdmissionNow().Add(-time.Hour)
	expired.ExpiresAt = &past

	oldEntry, err := svc.CreateMemberBlacklistFromAdmin(ctx, expired)
	require.NoError(t, err)
	newEntry, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGlobal, nil))
	require.NoError(t, err)

	assert.NotEqual(t, oldEntry.ID, newEntry.ID)
	assertMemberBlacklistReleased(t, fixture, oldEntry.ID)
	assert.Equal(t, 2, countMemberBlacklistEntries(t, fixture))
}

func TestMemberBlacklistCreateSweepResetsExpiredAdmissionFailureCount(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()
	past := fixedAdmissionNow().Add(-time.Hour)
	insertAdmissionFailureCount(t, fixture, DefaultFailedJoinLimit)
	oldEntry := createAdmissionFailureBlacklistForTest(t, svc, &past)

	newEntry, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1")))
	require.NoError(t, err)

	assert.NotEqual(t, oldEntry.ID, newEntry.ID)
	assertMemberBlacklistReleased(t, fixture, oldEntry.ID)
	assertAdmissionFailureCount(t, fixture, "10001", 0)
}

func TestMemberBlacklistGlobalUniqueDoesNotDuplicateOnNullGuild(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()

	first, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGlobal, nil))
	require.NoError(t, err)
	second, err := svc.CreateMemberBlacklistFromAdmin(ctx, memberBlacklistTestCreateInput(BlacklistScopeGlobal, nil))
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 1, countMemberBlacklistEntries(t, fixture))
}

func TestAdmissionBlacklistManualPardonResetsFailureCount(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()
	insertAdmissionFailureCount(t, fixture, DefaultFailedJoinLimit)
	entry := createAdmissionFailureBlacklistForTest(t, svc, nil)

	_, err := svc.ReleaseMemberBlacklistFromAdmin(ctx, memberBlacklistReleaseInput(entry.ID, BlacklistReleaseManualPardon))
	require.NoError(t, err)

	assertAdmissionFailureCount(t, fixture, "10001", 0)
}

func TestAdmissionBlacklistReleaseOnlyKeepsFailureCount(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()
	insertAdmissionFailureCount(t, fixture, DefaultFailedJoinLimit)
	entry := createAdmissionFailureBlacklistForTest(t, svc, nil)

	_, err := svc.ReleaseMemberBlacklistFromAdmin(ctx, memberBlacklistReleaseInput(entry.ID, BlacklistReleaseOnly))
	require.NoError(t, err)

	assertAdmissionFailureCount(t, fixture, "10001", DefaultFailedJoinLimit)
}

func TestExpiredAdmissionBlacklistSweepResetsFailureCount(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	past := fixedAdmissionNow().Add(-time.Hour)
	insertAdmissionFailureCount(t, fixture, DefaultFailedJoinLimit)
	entry := createAdmissionFailureBlacklistForTest(t, svc, &past)

	processed, err := svc.ProcessExpiredMemberBlacklists(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, processed)
	assertMemberBlacklistReleased(t, fixture, entry.ID)
	assertAdmissionFailureCount(t, fixture, "10001", 0)
}

func newBlacklistTestService(t *testing.T, fixture *postgresfixture.Fixture) *Service {
	t.Helper()
	svc := newSessionTestService(t, fixture)
	svc.now = fixedAdmissionNow
	return svc
}

func createAdmissionFailureBlacklistForTest(
	t *testing.T,
	svc *Service,
	expiresAt *time.Time,
) *MemberBlacklistEntry {
	t.Helper()
	var entry *MemberBlacklistEntry
	err := svc.repo.WithTx(context.Background(), func(ctx context.Context, tx pgx.Tx) error {
		created, err := svc.createAdmissionFailureBlacklistTx(ctx, tx, admissionFailureBlacklistTestInput(expiresAt))
		entry = created
		return err
	})
	require.NoError(t, err)
	return entry
}

func admissionFailureBlacklistTestInput(expiresAt *time.Time) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform: "qq", SubjectType: BlacklistSubjectQQUser, SubjectID: "10001",
		ScopeType: BlacklistScopeGuild, GuildID: stringPtr("guild-1"),
		Source: BlacklistSourceAdmissionFailure, ReasonCode: BlacklistReasonAdmissionTimeoutLimit,
		ReasonText: "admission failure limit reached", CreatedByType: BlacklistActorSystem,
		CreatedByID: "system", CreatedFrom: BlacklistCreatedFromAdmissionWorker,
		ExpiresAt: expiresAt, Metadata: map[string]any{"failureCount": DefaultFailedJoinLimit},
	}
}

func memberBlacklistTestCreateInput(scope MemberBlacklistScopeType, guildID *string) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform: "qq", SubjectType: BlacklistSubjectQQUser, SubjectID: "10001",
		ScopeType: scope, GuildID: guildID, Source: BlacklistSourceManualAdmin,
		ReasonCode: BlacklistReasonManualBlacklist, ReasonText: "manual test",
		CreatedByType: BlacklistActorAdminUser, CreatedByID: "admin-1",
		CreatedFrom: BlacklistCreatedFromAdminConsole,
		Metadata:    map[string]any{"operatorInput": "10001"},
	}
}

func memberBlacklistAccessQuery(guildID string) MemberBlacklistAccessQuery {
	return MemberBlacklistAccessQuery{
		Platform: "qq", SubjectType: BlacklistSubjectQQUser, SubjectID: "10001", GuildID: stringPtr(guildID),
	}
}

func memberBlacklistReleaseBySubjectInput(
	scope MemberBlacklistScopeType,
	guildID *string,
) MemberBlacklistReleaseBySubjectInput {
	return MemberBlacklistReleaseBySubjectInput{
		Platform: "qq", SubjectType: BlacklistSubjectQQUser, SubjectID: "10001",
		ScopeType: scope, GuildID: guildID, ReleasedByType: BlacklistActorAdminUser,
		ReleasedByID: "admin-1", ReleaseReasonCode: BlacklistReleaseManualPardon,
	}
}

func memberBlacklistReleaseInput(id string, reason MemberBlacklistReleaseReasonCode) MemberBlacklistReleaseInput {
	return MemberBlacklistReleaseInput{
		ID: id, ReleasedByType: BlacklistActorAdminUser, ReleasedByID: "admin-1", ReleaseReasonCode: reason,
	}
}

func assertMemberBlacklistReleased(t *testing.T, fixture *postgresfixture.Fixture, id string) {
	t.Helper()
	var releasedAt *time.Time
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT released_at FROM member_blacklist_entries WHERE id = $1
	`, id).Scan(&releasedAt)
	require.NoError(t, err)
	assert.NotNil(t, releasedAt)
}

func countMemberBlacklistEntries(t *testing.T, fixture *postgresfixture.Fixture) int {
	t.Helper()
	var count int
	err := fixture.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM member_blacklist_entries").Scan(&count)
	require.NoError(t, err)
	return count
}

func stringPtr(value string) *string {
	return &value
}
