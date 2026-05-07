package admission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestMemberBlacklistRejectsMissingRequiredMetadata(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.Metadata = map[string]any{"operatorInput": "10001"}

	_, err := svc.CreateMemberBlacklistFromAdmin(context.Background(), input)

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func TestMemberBlacklistRejectsBotManualAdminWithoutCreatedFrom(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.CreatedByType = BlacklistActorServiceAccount
	input.CreatedByID = "koishi-runtime"
	input.CreatedFrom = ""

	_, err := svc.CreateMemberBlacklistFromBot(context.Background(), input)

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func TestMemberBlacklistRejectsQQCommandManualAdminWithoutOperatorQQID(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.CreatedByType = BlacklistActorQQOperator
	input.CreatedByID = "90001"
	input.CreatedFrom = BlacklistCreatedFromQQCommand

	_, err := svc.CreateMemberBlacklistFromBot(context.Background(), input)

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func TestMemberBlacklistRejectsSourceReasonMismatch(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.Source = BlacklistSourceKickBlacklist

	_, err := svc.CreateMemberBlacklistFromBot(context.Background(), input)

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func TestMemberBlacklistAllowsModerationMetadata(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newBlacklistTestService(t, fixture)
	input := memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1"))
	input.Source = BlacklistSourceModerationAction
	input.ReasonCode = BlacklistReasonViolationReviewBlacklist
	input.CreatedByType = BlacklistActorServiceAccount
	input.CreatedByID = "koishi-runtime"
	input.CreatedFrom = BlacklistCreatedFromModerationReview
	input.Metadata = map[string]any{
		"reviewID":      "review-1",
		"workItemID":    "work-item-1",
		"targetGuildID": "guild-1",
	}

	_, err := svc.CreateMemberBlacklistFromBot(context.Background(), input)

	require.NoError(t, err)
}
