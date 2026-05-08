package serviceaccount

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKoishiRuntimeScopesIncludeAdmissionScopes(t *testing.T) {
	assert.ElementsMatch(t, []string{
		ScopeBotQQBindingConsume,
		ScopeBotQQVerificationRead,
		ScopeBotAdmissionSession,
		ScopeBotAdmissionEvent,
		ScopeBotAdmissionReview,
		ScopeBotAdmissionForward,
		ScopeBotMemberBlacklist,
	}, KoishiRuntimeScopes())
}
