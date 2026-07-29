package openplatform

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
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

func TestLoadInteractionChallengesRejectBlankTokenBeforeRedis(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	for _, token := range []string{"", " \t\n "} {
		consent, err := service.LoadConsentChallenge(ctx, token)
		require.ErrorIs(t, err, ErrConsentTokenInvalid)
		assert.Nil(t, consent)

		completion, err := service.LoadProfileCompletionChallenge(ctx, token)
		require.ErrorIs(t, err, ErrCompletionTokenInvalid)
		assert.Nil(t, completion)
	}
}

func TestDeleteInteractionChallengesRejectBlankTokenBeforeRedis(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	for _, token := range []string{"", " \t\n "} {
		require.ErrorIs(t, service.DeleteConsentChallenge(ctx, token), ErrConsentTokenInvalid)
		require.ErrorIs(t, service.deleteProfileCompletionChallenge(ctx, token), ErrCompletionTokenInvalid)
	}
}

func TestLoadInteractionChallengesNormalizeTokenBeforeRedis(t *testing.T) {
	ctx := context.Background()
	redis := redisfixture.Start(t)
	service := &Service{rdb: redis.Client}
	now := time.Now().UTC()

	consent := ConsentChallenge{
		Token:         "consent-token",
		AppID:         10,
		UserID:        20,
		Scopes:        []string{ScopeEmailRead},
		ConsentScopes: []string{ScopeEmailRead},
		RedirectURI:   "https://client.example.com/callback",
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Minute),
	}
	consentPayload, err := consent.MarshalPayload()
	require.NoError(t, err)
	require.NoError(t, redis.Client.Set(ctx, consentRedisPrefix+consent.Token, consentPayload, time.Minute).Err())

	loadedConsent, err := service.LoadConsentChallenge(ctx, " \t"+consent.Token+"\n ")
	require.NoError(t, err)
	require.NotNil(t, loadedConsent)
	assert.Equal(t, consent.Token, loadedConsent.Token)
	assert.Equal(t, consent.UserID, loadedConsent.UserID)

	completion := ProfileCompletionChallenge{
		Token:       "completion-token",
		AppID:       11,
		UserID:      21,
		Scopes:      []string{ScopeProfileBasicRead},
		RedirectURI: "https://client.example.com/callback",
		CreatedAt:   now,
		ExpiresAt:   now.Add(time.Minute),
	}
	completionPayload, err := completion.MarshalPayload()
	require.NoError(t, err)
	require.NoError(t, redis.Client.Set(ctx, completionRedisPrefix+completion.Token, completionPayload, time.Minute).Err())

	loadedCompletion, err := service.LoadProfileCompletionChallenge(ctx, " \t"+completion.Token+"\n ")
	require.NoError(t, err)
	require.NotNil(t, loadedCompletion)
	assert.Equal(t, completion.Token, loadedCompletion.Token)
	assert.Equal(t, completion.UserID, loadedCompletion.UserID)
}
