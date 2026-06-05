package botcredential

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKoishiRuntimeScopesIncludeAllBotScopes(t *testing.T) {
	assert.ElementsMatch(t, []string{
		ScopeBotQQBindingConsume,
		ScopeBotQQVerificationRead,
		ScopeBotAdmissionSession,
		ScopeBotAdmissionEvent,
		ScopeBotAdmissionReview,
		ScopeBotAdmissionForward,
		ScopeBotMemberBlacklistRead,
		ScopeBotMemberBlacklistManage,
	}, KoishiRuntimeScopes())
}
