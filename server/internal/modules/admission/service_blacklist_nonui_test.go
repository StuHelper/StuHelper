package admission

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestMemberBlacklistRejectsPastExpiresAtOnCreate(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	past := fixedAdmissionNow().Add(-time.Nanosecond)
	input.ExpiresAt = &past

	_, err := svc.CreateMemberBlacklistFromAdmin(context.Background(), input)

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
	assert.Equal(t, 0, countMemberBlacklistEntries(t, fixture))
}

func TestMemberBlacklistCreateConflictUsesSavepointAndReturnsExisting(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	first := insertMemberBlacklistForTest(t, svc, input)
	key := memberBlacklistCreateKey(normalizeMemberBlacklistCreateInput(input))
	var recovered *MemberBlacklistEntry

	err := svc.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := svc.repo.CreateMemberBlacklistSavepointTx(ctx, tx); err != nil {
			return err
		}
		_, createErr := svc.repo.CreateMemberBlacklistTx(ctx, tx, input, svc.now())
		if createErr == nil {
			return fmt.Errorf("expected duplicate member blacklist insert to fail")
		}
		entry, err := svc.memberBlacklistCreateConflictResultTx(ctx, tx, key, svc.now(), createErr)
		recovered = entry
		return err
	})

	require.NoError(t, err)
	require.NotNil(t, recovered)
	assert.Equal(t, first.ID, recovered.ID)
	assert.Equal(t, 1, countMemberBlacklistEntries(t, fixture))
}

func TestMemberBlacklistListFiltersByCreatedByID(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	first := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	second := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-2"))
	second.SubjectID = "10002"
	second.CreatedByID = "admin-2"
	second.Metadata["operatorInput"] = "10002"
	insertMemberBlacklistForTest(t, svc, first)
	want := insertMemberBlacklistForTest(t, svc, second)

	items, total, err := svc.ListMemberBlacklist(context.Background(), MemberBlacklistListFilter{
		CreatedByID: " admin-2 ",
		Status:      BlacklistStatusActive,
		PageSize:    20,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, want.ID, items[0].ID)
}

func TestMemberBlacklistListStatusAllAndReleasedUseValidSQL(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	ctx := context.Background()
	active := insertMemberBlacklistForTest(t, svc, memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1")))
	releasedInput := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-2"))
	releasedInput.SubjectID = "10002"
	releasedInput.Metadata["operatorInput"] = "10002"
	released := insertMemberBlacklistForTest(t, svc, releasedInput)
	_, err := svc.ReleaseMemberBlacklistFromAdmin(ctx, memberBlacklistReleaseInput(released.ID, BlacklistReleaseOnly))
	require.NoError(t, err)

	items, total, err := svc.ListMemberBlacklist(ctx, MemberBlacklistListFilter{Status: BlacklistStatusAll})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	assert.ElementsMatch(t, []string{active.ID, released.ID}, []string{items[0].ID, items[1].ID})

	items, total, err = svc.ListMemberBlacklist(ctx, MemberBlacklistListFilter{Status: BlacklistStatusReleased})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, released.ID, items[0].ID)
}

func TestMemberBlacklistListRejectsInvalidSource(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)

	_, _, err := svc.ListMemberBlacklist(context.Background(), MemberBlacklistListFilter{
		Source: MemberBlacklistSource("unknown_source"),
	})

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func insertMemberBlacklistForTest(
	t *testing.T,
	svc *Service,
	input MemberBlacklistCreateInput,
) *MemberBlacklistEntry {
	t.Helper()
	var entry *MemberBlacklistEntry
	err := svc.repo.WithTx(context.Background(), func(ctx context.Context, tx pgx.Tx) error {
		created, err := svc.repo.CreateMemberBlacklistTx(ctx, tx, input, svc.now())
		entry = created
		return err
	})
	require.NoError(t, err)
	return entry
}

func assertReleaseAuditFailureReset(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	entryID string,
	previousCount int,
) {
	t.Helper()
	var reset bool
	var previous int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT
			(details->>'admission_failure_count_reset')::boolean,
			(details->>'previous_failure_count')::int
		FROM audit_events
		WHERE event_type = 'member_blacklist.released' AND resource_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, entryID).Scan(&reset, &previous)
	require.NoError(t, err)
	assert.True(t, reset)
	assert.Equal(t, previousCount, previous)
}
