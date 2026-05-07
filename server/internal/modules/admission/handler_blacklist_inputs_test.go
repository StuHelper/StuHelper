package admission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBotMemberBlacklistCreateInputUsesTopLevelCreatedFrom(t *testing.T) {
	req := memberBlacklistCreateHTTPRequest{
		Platform:    "qq",
		SubjectType: BlacklistSubjectQQUser,
		SubjectID:   "10001",
		ScopeType:   BlacklistScopeGuild,
		GuildID:     stringPtr("guild-1"),
		Source:      BlacklistSourceManualAdmin,
		ReasonCode:  BlacklistReasonManualBlacklist,
		ReasonText:  "manual test",
		CreatedFrom: BlacklistCreatedFromKoishiConsole,
		Metadata: map[string]any{
			"createdFrom":  string(BlacklistCreatedFromQQCommand),
			"operatorQQID": "90001",
		},
	}

	input := botMemberBlacklistCreateInput(req)

	assert.Equal(t, BlacklistCreatedFromKoishiConsole, input.CreatedFrom)
	assert.Equal(t, BlacklistActorServiceAccount, input.CreatedByType)
	assert.Equal(t, "koishi-runtime", input.CreatedByID)
}
