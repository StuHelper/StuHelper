package openplatform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestBootstrapIdentityPublicSmokeClientEnsuresApprovedApp(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "identity-public-smoke-owner")
	reviewerID := seedOpenPlatformUser(t, postgres, "identity-public-smoke-reviewer")

	created, err := BootstrapIdentityPublicSmokeClient(ctx, repo, IdentityPublicSmokeClientBootstrapInput{
		OwnerUserID:      ownerID,
		ReviewerUserID:   reviewerID,
		ClientID:         "identity-public-smoke-test",
		HomepageURL:      "https://stuhelper.example.com",
		PrivacyPolicyURL: "https://stuhelper.example.com/privacy",
		RedirectURI:      "https://stuhelper.example.com/open-platform/identity-public-smoke/callback",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ClientSecret)
	require.NotNil(t, created.App)
	assert.Equal(t, AppStatusApproved, created.App.Status)
	assert.Equal(t, ownerID, created.App.OwnerUserID)
	assert.Equal(t, []string{"https://stuhelper.example.com/open-platform/identity-public-smoke/callback"}, created.App.RedirectURIs)
	assert.Equal(t, []string{ScopeResourceRead}, created.ClientScopes)

	_, err = repo.VerifyClientSecret(ctx, created.App.ClientID, created.ClientSecret)
	require.NoError(t, err)
	scopes, err := repo.ListApprovedScopes(ctx, created.App.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{ScopeResourceRead}, scopes)
	assertOpenPlatformAuditCount(t, postgres, created.App.ID, reviewerID, "open_platform.app.identity_public_smoke_bootstrapped", 1)
	developerEvents, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       created.App.ID,
		EventType:   "open_platform.app.identity_public_smoke_bootstrapped",
		PageSize:    10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, developerEvents.Total)
	require.Len(t, developerEvents.List, 1)
	bootstrapEvent := developerEvents.List[0]
	assert.Equal(t, "open_platform.app.identity_public_smoke_bootstrapped", bootstrapEvent.EventType)
	require.NotNil(t, bootstrapEvent.RequestID)
	assert.Equal(t, defaultIdentityPublicSmokeRequestID, *bootstrapEvent.RequestID)
	assert.ElementsMatch(t, []string{ScopeResourceRead}, bootstrapEvent.Scopes)
	assert.Equal(t, created.App.ClientID, bootstrapEvent.Details["clientID"])
	assert.Equal(t, created.App.DisplayName, bootstrapEvent.Details["displayName"])

	updated, err := BootstrapIdentityPublicSmokeClient(ctx, repo, IdentityPublicSmokeClientBootstrapInput{
		OwnerUserID:      ownerID,
		ReviewerUserID:   reviewerID,
		ClientID:         created.App.ClientID,
		ClientSecret:     "ids_replacement_secret",
		HomepageURL:      "https://stuhelper.example.com",
		PrivacyPolicyURL: "https://stuhelper.example.com/privacy",
		RedirectURI:      "https://stuhelper.example.com/open-platform/identity-public-smoke/v2/callback",
		ClientScopes:     []string{ScopeResourceWrite},
	})
	require.NoError(t, err)
	require.Equal(t, created.App.ID, updated.App.ID)
	assert.Equal(t, []string{"https://stuhelper.example.com/open-platform/identity-public-smoke/v2/callback"}, updated.App.RedirectURIs)
	_, err = repo.VerifyClientSecret(ctx, updated.App.ClientID, created.ClientSecret)
	require.ErrorIs(t, err, ErrAppNotFound)
	_, err = repo.VerifyClientSecret(ctx, updated.App.ClientID, "ids_replacement_secret")
	require.NoError(t, err)
	scopes, err = repo.ListApprovedScopes(ctx, updated.App.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{ScopeResourceRead, ScopeResourceWrite}, scopes)
}

func TestEnsureApprovedAppAuditVisibleToDeveloperOwner(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "ensure-approved-owner")
	reviewerID := seedOpenPlatformUser(t, postgres, "ensure-approved-reviewer")
	app := &App{
		CasdoorApplicationName: "casdoor-ensure-approved-app",
		OwnerUserID:            ownerID,
		ClientID:               "ensure-approved-app",
		ClientSecretHash:       hashClientSecret("ensure-approved-secret"),
		DisplayName:            "Ensure Approved App",
		Description:            "Used by EnsureApprovedApp audit visibility tests.",
		HomepageURL:            "https://ensure-approved.example.com",
		PrivacyPolicyURL:       "https://ensure-approved.example.com/privacy",
		RedirectURIs:           []string{"https://ensure-approved.example.com/callback"},
		Status:                 AppStatusApproved,
	}

	ensured, err := repo.EnsureApprovedApp(ctx, app, []ScopeRequest{
		{
			Scope:  ScopeResourceRead,
			Reason: "production approved client smoke access",
		},
	}, EnsureApprovedAppOptions{
		ReviewerUserID: reviewerID,
		RequestID:      "ensure-approved-app",
	})
	require.NoError(t, err)

	events, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       ensured.ID,
		EventType:   "open_platform.app.approved_app_ensured",
		PageSize:    10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, events.Total)
	require.Len(t, events.List, 1)
	ensureEvent := events.List[0]
	assert.Equal(t, "open_platform.app.approved_app_ensured", ensureEvent.EventType)
	require.NotNil(t, ensureEvent.RequestID)
	assert.Equal(t, "ensure-approved-app", *ensureEvent.RequestID)
	assert.ElementsMatch(t, []string{ScopeResourceRead}, ensureEvent.Scopes)
	assert.Equal(t, ensured.ClientID, ensureEvent.Details["clientID"])
	assert.Equal(t, ensured.DisplayName, ensureEvent.Details["displayName"])

	_, err = service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: reviewerID,
		AppID:       ensured.ID,
		EventType:   "open_platform.app.approved_app_ensured",
		PageSize:    10,
	})
	require.ErrorIs(t, err, ErrAppNotFound)
}

func TestBootstrapIdentityPublicSmokeClientDoesNotRepairRevokedAppByDefault(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)

	ownerID := seedOpenPlatformUser(t, postgres, "identity-public-smoke-revoked-owner")
	reviewerID := seedOpenPlatformUser(t, postgres, "identity-public-smoke-revoked-reviewer")

	created, err := BootstrapIdentityPublicSmokeClient(ctx, repo, IdentityPublicSmokeClientBootstrapInput{
		OwnerUserID:      ownerID,
		ReviewerUserID:   reviewerID,
		ClientID:         "identity-public-smoke-revoked-test",
		ClientSecret:     "ids_initial_secret",
		HomepageURL:      "https://stuhelper.example.com",
		PrivacyPolicyURL: "https://stuhelper.example.com/privacy",
		RedirectURI:      "https://stuhelper.example.com/open-platform/identity-public-smoke/callback",
	})
	require.NoError(t, err)
	require.NoError(t, repo.SetAppStatus(ctx, created.App.ID, AppStatusRevoked))

	_, err = BootstrapIdentityPublicSmokeClient(ctx, repo, IdentityPublicSmokeClientBootstrapInput{
		OwnerUserID:      ownerID,
		ReviewerUserID:   reviewerID,
		ClientID:         created.App.ClientID,
		ClientSecret:     "ids_repair_secret",
		HomepageURL:      "https://stuhelper.example.com",
		PrivacyPolicyURL: "https://stuhelper.example.com/privacy",
		RedirectURI:      "https://stuhelper.example.com/open-platform/identity-public-smoke/callback",
	})
	require.ErrorIs(t, err, ErrInvalidAppStatus)

	repaired, err := BootstrapIdentityPublicSmokeClient(ctx, repo, IdentityPublicSmokeClientBootstrapInput{
		OwnerUserID:        ownerID,
		ReviewerUserID:     reviewerID,
		ClientID:           created.App.ClientID,
		ClientSecret:       "ids_repair_secret",
		HomepageURL:        "https://stuhelper.example.com",
		PrivacyPolicyURL:   "https://stuhelper.example.com/privacy",
		RedirectURI:        "https://stuhelper.example.com/open-platform/identity-public-smoke/callback",
		AllowRevokedRepair: true,
	})
	require.NoError(t, err)
	assert.Equal(t, AppStatusApproved, repaired.App.Status)
	_, err = repo.VerifyClientSecret(ctx, repaired.App.ClientID, "ids_repair_secret")
	require.NoError(t, err)
}
