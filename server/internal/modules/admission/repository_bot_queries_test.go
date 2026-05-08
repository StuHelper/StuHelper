package admission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPendingActionSessionsQueryUsesDirectBotFilters(t *testing.T) {
	const testPendingActionLimit = 10
	filter := AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "bot-1", Limit: testPendingActionLimit}

	query, args := pendingActionSessionsQuery(filter, fixedAdmissionNow())

	assert.Contains(t, query, "platform = $1")
	assert.Contains(t, query, "bot_self_id = $2")
	assert.NotContains(t, query, "::text = '' OR")
	assert.NotContains(t, query, " OR platform")
	assert.Equal(t, "qq", args[0])
	assert.Equal(t, "bot-1", args[1])
}

func TestPendingActionSessionFilterClausesAlwaysUseBotIdentity(t *testing.T) {
	const testPendingActionLimit = 10

	query, args := pendingActionSessionsQuery(
		AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "bot-1", Limit: testPendingActionLimit},
		fixedAdmissionNow(),
	)

	assert.Contains(t, query, "platform = $1")
	assert.Contains(t, query, "bot_self_id = $2")
	assert.Equal(t, "qq", args[0])
	assert.Equal(t, "bot-1", args[1])
	assert.Equal(t, StatusJoinedMuted, args[2])
}
