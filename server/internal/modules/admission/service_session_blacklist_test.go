package admission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestCreateBotSessionRejectsActiveMemberBlacklist(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	_, err := svc.CreateMemberBlacklistFromAdmin(context.Background(), memberBlacklistTestCreateInput(BlacklistScopeGuild, stringPtr("guild-1")))
	require.NoError(t, err)

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001", BotSelfID: "514",
	})

	require.ErrorIs(t, err, ErrMemberBlacklisted)
	assert.Nil(t, created)
	assert.Equal(t, 0, countAdmissionSessions(t, fixture))
}

func countAdmissionSessions(t *testing.T, fixture *postgresfixture.Fixture) int {
	t.Helper()
	var count int
	err := fixture.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM group_admission_sessions").Scan(&count)
	require.NoError(t, err)
	return count
}
