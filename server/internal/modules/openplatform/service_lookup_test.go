package openplatform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceLookupMethodsRejectInvalidIDsBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	for _, appID := range []int64{0, -1} {
		app, err := service.AppByID(ctx, appID)
		require.ErrorIs(t, err, ErrAppNotFound)
		assert.Nil(t, app)
	}

	for _, userID := range []int64{0, -1} {
		projection, err := service.UserProjection(ctx, userID)
		require.ErrorIs(t, err, ErrDisclosureUnavailable)
		assert.Nil(t, projection)
	}
}

func TestVerifyClientSecretRejectsBlankCredentialsBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	tests := []struct {
		name         string
		clientID     string
		clientSecret string
	}{
		{name: "blank client id", clientID: " \t\n ", clientSecret: "secret"},
		{name: "blank client secret", clientID: "client-id", clientSecret: " \t\n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := service.VerifyClientSecret(ctx, tt.clientID, tt.clientSecret)
			require.ErrorIs(t, err, ErrAppNotFound)
			assert.Nil(t, app)
		})
	}
}

func TestAuthorizeAppByClientIDRejectsBlankClientIDBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	app, err := service.AuthorizeAppByClientID(ctx, " \t\n ")
	require.ErrorIs(t, err, ErrAppNotFound)
	assert.Nil(t, app)
}

func TestIdentityClientCredentialsTokenActiveRejectsBlankClientIDBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	active, err := service.IdentityClientCredentialsTokenActive(ctx, " \t\n ", []string{ScopeResourceRead})
	require.ErrorIs(t, err, ErrAppNotFound)
	assert.False(t, active)
}

func TestIdentityTokenActivityRejectsBlankClientIDBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	active, err := service.IdentityAccessTokenActive(ctx, " \t\n ", 42, []string{"openid"})
	require.ErrorIs(t, err, ErrAppNotFound)
	assert.False(t, active)

	fingerprint, active, err := service.IdentityAuthorizationFingerprint(ctx, " \t\n ", 42, []string{"openid"})
	require.ErrorIs(t, err, ErrAppNotFound)
	assert.False(t, active)
	assert.Empty(t, fingerprint)
}

func TestIdentityTokenActivityRejectsInvalidUserBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	for _, userID := range []int64{0, -1} {
		active, err := service.IdentityAccessTokenActive(ctx, "client-id", userID, []string{"openid"})
		require.NoError(t, err)
		assert.False(t, active)

		fingerprint, active, err := service.IdentityAuthorizationFingerprint(ctx, "client-id", userID, []string{"openid"})
		require.NoError(t, err)
		assert.False(t, active)
		assert.Empty(t, fingerprint)
	}
}
