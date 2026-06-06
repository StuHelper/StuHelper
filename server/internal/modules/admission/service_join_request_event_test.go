package admission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeJoinRequestEventInputValidatesDecision(t *testing.T) {
	input := validJoinRequestEventInput()
	input.Decision = AdmissionJoinRequestDecisionAction(" Reject ")

	normalized, err := normalizeJoinRequestEventInput(input)

	require.NoError(t, err)
	assert.Equal(t, AdmissionJoinRequestDecisionReject, normalized.Decision)
	assert.Equal(t, "qq", normalized.Platform)
	assert.Equal(t, "guild-1", normalized.GuildID)
	assert.Equal(t, "10001", normalized.QQID)
	assert.Equal(t, "request-1", normalized.RequestID)
}

func TestNormalizeJoinRequestEventInputRejectsInvalidDecision(t *testing.T) {
	input := validJoinRequestEventInput()
	input.Decision = AdmissionJoinRequestDecisionAction("allow")

	_, err := normalizeJoinRequestEventInput(input)

	require.ErrorIs(t, err, ErrAdmissionInvalidInput)
}

func TestNormalizeJoinRequestEventInputAllowsMissingDecision(t *testing.T) {
	input := validJoinRequestEventInput()
	input.Decision = AdmissionJoinRequestDecisionAction(" ")

	normalized, err := normalizeJoinRequestEventInput(input)

	require.NoError(t, err)
	assert.Empty(t, normalized.Decision)
	assert.Equal(t, "unknown", joinRequestAuditEvent(context.Background(), normalized).Action)
}

func TestNormalizeJoinRequestEventInputRequiresAuditIdentity(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*AdmissionJoinRequestEventInput)
	}{
		{name: "platform", mut: func(input *AdmissionJoinRequestEventInput) { input.Platform = " " }},
		{name: "guild id", mut: func(input *AdmissionJoinRequestEventInput) { input.GuildID = " " }},
		{name: "qq id", mut: func(input *AdmissionJoinRequestEventInput) { input.QQID = " " }},
		{name: "request id", mut: func(input *AdmissionJoinRequestEventInput) { input.RequestID = " " }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validJoinRequestEventInput()
			tc.mut(&input)

			_, err := normalizeJoinRequestEventInput(input)

			require.ErrorIs(t, err, ErrAdmissionInvalidInput)
		})
	}
}

func validJoinRequestEventInput() AdmissionJoinRequestEventInput {
	return AdmissionJoinRequestEventInput{
		Platform:  " qq ",
		GuildID:   " guild-1 ",
		QQID:      " 10001 ",
		RequestID: " request-1 ",
		Success:   true,
		RawEvent:  map[string]any{"source": "koishi"},
	}
}
