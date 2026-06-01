package systemconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEmailDeliveryPolicyAcceptsPriorityFallback(t *testing.T) {
	err := ValidateEmailDeliveryPolicy(`{"mode":"priority","maxAttempts":2,"providers":[{"name":"tencent_ses","enabled":true,"priority":10,"weight":100},{"name":"resend","enabled":true,"priority":20,"weight":100}]}`)

	require.NoError(t, err)
}

func TestValidateEmailDeliveryPolicyRejectsUnsupportedProvider(t *testing.T) {
	err := ValidateEmailDeliveryPolicy(`{"mode":"priority","maxAttempts":1,"providers":[{"name":"unknown","enabled":true,"priority":10,"weight":1}]}`)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `provider "unknown" is not supported`)
}
