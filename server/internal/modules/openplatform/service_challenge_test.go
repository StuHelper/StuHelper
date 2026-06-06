package openplatform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInteractionChallengesRejectInvalidUserBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	service := &Service{}
	app := &App{ID: 1}
	req := AuthorizeRequest{RedirectURI: "https://client.example.com/callback"}

	for _, userID := range []int64{0, -1} {
		consent, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, req)
		require.ErrorIs(t, err, ErrDisclosureUnavailable)
		assert.Nil(t, consent)

		completion, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, req)
		require.ErrorIs(t, err, ErrDisclosureUnavailable)
		assert.Nil(t, completion)
	}
}
