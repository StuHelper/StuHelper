package admission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestAdmissionQQAccessUsesPlatformAndGuildScope(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertScopedAdmissionFailure(t, fixture, "qq", "guild-2", "10001")

	allowed, err := svc.GetAdmissionQQAccess(context.Background(), AdmissionQQAccessQuery{
		Platform: "qq",
		GuildID:  "guild-1",
		QQID:     "10001",
	})
	require.NoError(t, err)
	assert.True(t, allowed.CanJoin)

	blocked, err := svc.GetAdmissionQQAccess(context.Background(), AdmissionQQAccessQuery{
		Platform: "qq",
		GuildID:  "guild-2",
		QQID:     "10001",
	})
	require.NoError(t, err)
	assert.False(t, blocked.CanJoin)
}

func TestMemberBlacklistLifecycleAndValidation(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)

	assertMemberBlacklistAccessUsesGuildScope(t, svc)
	assertMemberBlacklistAccessUsesGlobalScope(t, svc)
	assertMemberBlacklistRejectsGlobalScopeWithGuildID(t, svc)
	assertMemberBlacklistReleaseRestoresAccess(t, svc)
	assertMemberBlacklistCreateIsIdempotentForActiveScope(t, svc)
	assertMemberBlacklistCreateReplacesExpiredScope(t, svc)
}

func assertMemberBlacklistAccessUsesGuildScope(t *testing.T, svc *Service) {
	t.Helper()
	_, err := svc.CreateMemberBlacklist(context.Background(), memberBlacklistCreateInput(
		"10001",
		MemberBlacklistScopeGuild,
		"guild-2",
	))
	require.NoError(t, err)

	allowed, err := svc.GetMemberBlacklistAccess(context.Background(), memberBlacklistAccessQuery("10001", "guild-1"))
	require.NoError(t, err)
	assert.True(t, allowed.CanJoin)

	blocked, err := svc.GetMemberBlacklistAccess(context.Background(), memberBlacklistAccessQuery("10001", "guild-2"))
	require.NoError(t, err)
	require.NotNil(t, blocked.MatchedBlacklist)
	assert.False(t, blocked.CanJoin)
	assert.Equal(t, MemberBlacklistScopeGuild, blocked.MatchedBlacklist.ScopeType)
}

func assertMemberBlacklistAccessUsesGlobalScope(t *testing.T, svc *Service) {
	t.Helper()
	_, err := svc.CreateMemberBlacklist(context.Background(), memberBlacklistCreateInput(
		"10002",
		MemberBlacklistScopeGlobal,
		"",
	))
	require.NoError(t, err)

	blocked, err := svc.GetMemberBlacklistAccess(context.Background(), memberBlacklistAccessQuery("10002", "guild-99"))
	require.NoError(t, err)
	require.NotNil(t, blocked.MatchedBlacklist)
	assert.False(t, blocked.CanJoin)
	assert.Equal(t, MemberBlacklistScopeGlobal, blocked.MatchedBlacklist.ScopeType)
}

func assertMemberBlacklistRejectsGlobalScopeWithGuildID(t *testing.T, svc *Service) {
	t.Helper()
	_, err := svc.CreateMemberBlacklist(context.Background(), memberBlacklistCreateInput(
		"10003",
		MemberBlacklistScopeGlobal,
		"guild-1",
	))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMemberBlacklistInvalidInput))

	err = svc.ReleaseMemberBlacklistBySubject(context.Background(), MemberBlacklistReleaseBySubjectInput{
		Platform:          "qq",
		SubjectType:       MemberBlacklistSubjectQQUser,
		SubjectID:         "10003",
		ScopeType:         MemberBlacklistScopeGlobal,
		GuildID:           "guild-1",
		ReleasedByType:    MemberBlacklistActorAdminUser,
		ReleasedByID:      "admin-1",
		ReleaseReasonCode: "manual_pardon",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMemberBlacklistInvalidInput))
}

func assertMemberBlacklistReleaseRestoresAccess(t *testing.T, svc *Service) {
	t.Helper()
	ctx := context.Background()

	entry, err := svc.CreateMemberBlacklist(ctx, memberBlacklistCreateInput("10004", MemberBlacklistScopeGuild, "guild-1"))
	require.NoError(t, err)

	blocked, err := svc.GetMemberBlacklistAccess(ctx, memberBlacklistAccessQuery("10004", "guild-1"))
	require.NoError(t, err)
	assert.False(t, blocked.CanJoin)

	err = svc.ReleaseMemberBlacklist(ctx, MemberBlacklistReleaseInput{
		ID:                entry.ID,
		ReleasedByType:    MemberBlacklistActorAdminUser,
		ReleasedByID:      "admin-1",
		ReleaseReasonCode: "manual_pardon",
	})
	require.NoError(t, err)

	allowed, err := svc.GetMemberBlacklistAccess(ctx, memberBlacklistAccessQuery("10004", "guild-1"))
	require.NoError(t, err)
	assert.True(t, allowed.CanJoin)
}

func assertMemberBlacklistCreateIsIdempotentForActiveScope(t *testing.T, svc *Service) {
	t.Helper()
	ctx := context.Background()

	first, err := svc.CreateMemberBlacklist(ctx, memberBlacklistCreateInput("10005", MemberBlacklistScopeGuild, "guild-1"))
	require.NoError(t, err)
	second, err := svc.CreateMemberBlacklist(ctx, memberBlacklistCreateInput("10005", MemberBlacklistScopeGuild, "guild-1"))
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	items, total, err := svc.ListMemberBlacklist(ctx, MemberBlacklistListFilter{
		Platform:   "qq",
		SubjectID:  "10005",
		ActiveOnly: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
}

func assertMemberBlacklistCreateReplacesExpiredScope(t *testing.T, svc *Service) {
	t.Helper()
	ctx := context.Background()

	expired := memberBlacklistCreateInput("10006", MemberBlacklistScopeGuild, "guild-1")
	expired.ExpiresAt = ptrTime(svc.now().Add(-time.Hour))
	first, err := svc.CreateMemberBlacklist(ctx, expired)
	require.NoError(t, err)

	second, err := svc.CreateMemberBlacklist(ctx, memberBlacklistCreateInput("10006", MemberBlacklistScopeGuild, "guild-1"))
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID)

	items, total, err := svc.ListMemberBlacklist(ctx, MemberBlacklistListFilter{
		Platform:  "qq",
		SubjectID: "10006",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	assert.NotNil(t, memberBlacklistEntryByID(items, first.ID).ReleasedAt)
}

func memberBlacklistCreateInput(
	subjectID string,
	scope MemberBlacklistScopeType,
	guildID string,
) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform:      "qq",
		SubjectType:   MemberBlacklistSubjectQQUser,
		SubjectID:     subjectID,
		ScopeType:     scope,
		GuildID:       guildID,
		Source:        MemberBlacklistSourceManualAdmin,
		ReasonCode:    "manual_blacklist",
		ReasonText:    "manual test",
		CreatedByType: MemberBlacklistActorAdminUser,
		CreatedByID:   "admin-1",
		CreatedFrom:   MemberBlacklistFromAdminConsole,
	}
}

func memberBlacklistAccessQuery(subjectID string, guildID string) MemberBlacklistAccessQuery {
	return MemberBlacklistAccessQuery{
		Platform:    "qq",
		SubjectType: MemberBlacklistSubjectQQUser,
		SubjectID:   subjectID,
		GuildID:     guildID,
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func memberBlacklistEntryByID(items []MemberBlacklistEntry, id string) MemberBlacklistEntry {
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	return MemberBlacklistEntry{}
}

func insertScopedAdmissionFailure(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	platform string,
	guildID string,
	qqID string,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO group_admission_failures (platform, guild_id, qq_id, failure_count, blacklisted_at)
		VALUES ($1, $2, $3, 3, NOW())
	`, platform, guildID, qqID)
	require.NoError(t, err)
}
