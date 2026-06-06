package openplatform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeScopeRequestsKeepsFirstNonEmptyReasonForDuplicateScopes(t *testing.T) {
	scopes, err := normalizeScopeRequests([]ScopeRequestInput{
		{Scope: " " + ScopeEmailRead + " ", Reason: "read contact email"},
		{Scope: ScopeEmailRead, Reason: " "},
		{Scope: ScopeProfileBasicRead, Reason: "read public profile"},
	})

	require.NoError(t, err)
	assert.Equal(t, []ScopeRequestInput{
		{Scope: ScopeEmailRead, Reason: "read contact email"},
		{Scope: ScopeProfileBasicRead, Reason: "read public profile"},
	}, scopes)
}

func TestNormalizeScopeRequestsUsesLaterNonEmptyReasonForDuplicateScopes(t *testing.T) {
	scopes, err := normalizeScopeRequests([]ScopeRequestInput{
		{Scope: ScopeEmailRead, Reason: " "},
		{Scope: " " + ScopeEmailRead + " ", Reason: "read contact email"},
	})

	require.NoError(t, err)
	assert.Equal(t, []ScopeRequestInput{
		{Scope: ScopeEmailRead, Reason: "read contact email"},
	}, scopes)
}
