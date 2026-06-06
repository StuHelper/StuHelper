package openplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestUserConsentManagementListAndRevoke(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "owner")
	userID := seedOpenPlatformUser(t, postgres, "viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "grant-request",
	}, []string{ScopeProfileBasicRead, ScopeEmailRead}))

	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	assert.Equal(t, app.ID, consents[0].App.ID)
	assert.Equal(t, "Open Platform Test App", consents[0].App.DisplayName)
	require.Len(t, consents[0].Scopes, 2)
	assert.Equal(t, ScopeEmailRead, consents[0].Scopes[0].Scope)
	assert.Equal(t, ScopeProfileBasicRead, consents[0].Scopes[1].Scope)
	initialScopes := userConsentScopesByScope(t, consents[0].Scopes)
	require.Contains(t, initialScopes, ScopeEmailRead)
	require.Contains(t, initialScopes, ScopeProfileBasicRead)
	assert.Nil(t, initialScopes[ScopeEmailRead].LastUsedAt)
	assert.Nil(t, initialScopes[ScopeProfileBasicRead].LastUsedAt)
	assert.Equal(t, "integration test", initialScopes[ScopeEmailRead].Reason)
	assert.Equal(t, "integration test", initialScopes[ScopeProfileBasicRead].Reason)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.granted", 2)

	payload, err := service.UserInfo(ctx, DisclosureRequest{
		ClientID:       app.ClientID,
		UserID:         userID,
		Scopes:         []string{ScopeProfileBasicRead, ScopeEmailRead},
		RedirectURI:    app.RedirectURIs[0],
		ConsentBaseURL: "https://account.example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "viewer", payload["username"])
	assert.Equal(t, "viewer@example.com", payload["email"])

	consents, err = service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	require.Len(t, consents[0].Scopes, 2)
	usedScopes := userConsentScopesByScope(t, consents[0].Scopes)
	emailScope := usedScopes[ScopeEmailRead]
	profileScope := usedScopes[ScopeProfileBasicRead]
	require.NotNil(t, emailScope.LastUsedAt)
	require.NotNil(t, profileScope.LastUsedAt)
	assert.False(t, emailScope.LastUsedAt.IsZero())
	assert.False(t, profileScope.LastUsedAt.IsZero())

	require.NoError(t, service.RevokeUserConsent(ctx, RevokeConsentInput{
		UserID:    userID,
		AppID:     app.ID,
		Scopes:    []string{ScopeEmailRead},
		RequestID: "revoke-email",
	}))
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.revoked", 1)

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:       app.ClientID,
		UserID:         userID,
		Scopes:         []string{ScopeProfileBasicRead, ScopeEmailRead},
		RedirectURI:    app.RedirectURIs[0],
		ConsentBaseURL: "https://account.example.com",
	})
	var consentErr ConsentRequiredError
	require.Error(t, err)
	assert.True(t, errors.As(err, &consentErr))
	assert.Contains(t, consentErr.ConsentURL, "/consent?token=")

	consents, err = service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	require.Len(t, consents[0].Scopes, 1)
	assert.Equal(t, ScopeProfileBasicRead, consents[0].Scopes[0].Scope)
	require.NotNil(t, consents[0].Scopes[0].LastUsedAt)
	assert.False(t, consents[0].Scopes[0].LastUsedAt.IsZero())

	require.NoError(t, service.RevokeUserConsent(ctx, RevokeConsentInput{
		UserID:    userID,
		AppID:     app.ID,
		RequestID: "revoke-app",
	}))
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.revoked", 2)

	consents, err = service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
}

func TestOpenPlatformDisclosureRequiresRegisteredRedirectURI(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "redirect-owner")
	userID := seedOpenPlatformUser(t, postgres, "redirect-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "grant-redirect",
	}, []string{ScopeProfileBasicRead}))

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:    app.ClientID,
		UserID:      userID,
		Scopes:      []string{ScopeProfileBasicRead},
		RedirectURI: "https://attacker.example.com/callback",
		RequestID:   "bad-redirect",
	})
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, userID, "open_platform.disclosure.denied", map[string]any{
		"clientID": app.ClientID,
		"endpoint": "userinfo",
		"result":   "redirect_uri_not_allowed",
	})
}

func TestOpenPlatformDisclosureRejectsBearerClientMismatch(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "mismatch-owner")
	userID := seedOpenPlatformUser(t, postgres, "mismatch-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "grant-mismatch",
	}, []string{ScopeProfileBasicRead}))

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:              app.ClientID,
		AuthenticatedClientID: "other-client",
		AuthenticatedByBearer: true,
		UserID:                userID,
		Scopes:                []string{ScopeProfileBasicRead},
		RedirectURI:           app.RedirectURIs[0],
		RequestID:             "client-mismatch",
	})
	require.ErrorIs(t, err, ErrDisclosureClientMismatch)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, userID, "open_platform.disclosure.denied", map[string]any{
		"clientID": app.ClientID,
		"endpoint": "userinfo",
		"result":   "client_mismatch",
	})
}

func TestAdminUserConsentListAndTargetedRevoke(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "admin-consent-owner")
	adminID := seedOpenPlatformUser(t, postgres, "admin-consent-admin")
	userID := seedOpenPlatformUser(t, postgres, "admin-consent-viewer")
	otherUserID := seedOpenPlatformUser(t, postgres, "admin-consent-other")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	otherApp := seedApprovedOpenPlatformAppWithName(t, ctx, repo, ownerID, "Other Consent App", []string{
		ScopeProfileBasicRead,
	})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "admin-consent-grant",
	}, []string{ScopeProfileBasicRead, ScopeEmailRead}))
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      otherUserID,
		GrantSource: "web",
		RequestID:   "admin-consent-other-user-grant",
	}, []string{ScopeProfileBasicRead}))
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       otherApp.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "admin-consent-other-app-grant",
	}, []string{ScopeProfileBasicRead}))

	_, err = service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{})
	require.ErrorIs(t, err, ErrInvalidAuditFilter)

	byApp, err := service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{AppID: app.ID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, byApp.Total)
	require.Len(t, byApp.List, 2)
	assert.Equal(t, userID, byApp.List[0].UserID)
	assert.Equal(t, app.ID, byApp.List[0].App.ID)
	require.Len(t, byApp.List[0].Scopes, 2)
	assert.Equal(t, otherUserID, byApp.List[1].UserID)
	require.Len(t, byApp.List[1].Scopes, 1)

	byUser, err := service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{UserID: userID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, byUser.Total)
	require.Len(t, byUser.List, 2)

	targeted, err := service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{AppID: app.ID, UserID: userID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, targeted.Total)
	require.Len(t, targeted.List, 1)
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, userConsentScopeNames(targeted.List[0].Scopes))

	require.NoError(t, service.RevokeAdminUserConsent(ctx, AdminRevokeConsentInput{
		AppID:       app.ID,
		UserID:      userID,
		ActorUserID: adminID,
		Reason:      "privacy incident response",
		Scopes:      []string{ScopeEmailRead},
		RequestID:   "admin-consent-revoke-email",
	}))
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, userID, "open_platform.consent.revoked", map[string]any{
		"actor":  "admin",
		"reason": "privacy incident response",
		"source": "admin_console",
	})
	var metadataRaw []byte
	require.NoError(t, postgres.DB.QueryRow(ctx, `
		SELECT metadata
		FROM open_platform_audit_events
		WHERE app_id = $1
		  AND user_id = $2
		  AND event_type = 'open_platform.consent.revoked'
		ORDER BY id DESC
		LIMIT 1
	`, app.ID, userID).Scan(&metadataRaw))
	metadata := map[string]any{}
	require.NoError(t, json.Unmarshal(metadataRaw, &metadata))
	assert.Equal(t, float64(adminID), metadata["actorUserID"])

	targeted, err = service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{AppID: app.ID, UserID: userID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, targeted.Total)
	require.Len(t, targeted.List, 1)
	assert.Equal(t, []string{ScopeProfileBasicRead}, userConsentScopeNames(targeted.List[0].Scopes))

	require.NoError(t, service.RevokeAdminUserConsent(ctx, AdminRevokeConsentInput{
		AppID:       app.ID,
		UserID:      userID,
		ActorUserID: adminID,
		Reason:      "complete consent reset",
		RequestID:   "admin-consent-revoke-app",
	}))
	targeted, err = service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{AppID: app.ID, UserID: userID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Zero(t, targeted.Total)
	assert.Empty(t, targeted.List)

	byApp, err = service.ListAdminUserConsents(ctx, ListAdminUserConsentsInput{AppID: app.ID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, byApp.Total)
	require.Len(t, byApp.List, 1)
	assert.Equal(t, otherUserID, byApp.List[0].UserID)
}

func TestUserConsentAuditEventsAreScopedToCurrentUser(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "audit-owner")
	userID := seedOpenPlatformUser(t, postgres, "audit-viewer")
	otherUserID := seedOpenPlatformUser(t, postgres, "audit-other")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "grant-user",
	}, []string{ScopeProfileBasicRead, ScopeEmailRead}))
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      otherUserID,
		GrantSource: "web",
		RequestID:   "grant-other",
	}, []string{ScopeEmailRead}))

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:       app.ClientID,
		UserID:         userID,
		Scopes:         []string{ScopeProfileBasicRead, ScopeEmailRead},
		RedirectURI:    app.RedirectURIs[0],
		ConsentBaseURL: "https://account.example.com",
		RequestID:      "userinfo-user",
	})
	require.NoError(t, err)
	require.NoError(t, service.RevokeUserConsent(ctx, RevokeConsentInput{
		UserID:    userID,
		AppID:     app.ID,
		Scopes:    []string{ScopeEmailRead},
		RequestID: "revoke-user-email",
	}))

	events, err := service.ListUserConsentAuditEvents(ctx, ListUserConsentAuditEventsInput{
		UserID:   userID,
		PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, events.Total)
	require.Len(t, events.List, 4)
	for _, event := range events.List {
		require.NotNil(t, event.AppID)
		assert.Equal(t, app.ID, *event.AppID)
		require.NotNil(t, event.AppDisplayName)
		assert.Equal(t, "Open Platform Test App", *event.AppDisplayName)
		require.NotNil(t, event.ClientID)
		assert.Equal(t, app.ClientID, *event.ClientID)
		if event.RequestID != nil {
			assert.NotEqual(t, "grant-other", *event.RequestID)
		}
	}

	disclosure := userConsentAuditEventByRequestID(t, events.List, "userinfo-user")
	assert.Equal(t, "open_platform.disclosure.granted", disclosure.EventType)
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, disclosure.Scopes)
	require.NotNil(t, disclosure.Endpoint)
	assert.Equal(t, "userinfo", *disclosure.Endpoint)
	require.NotNil(t, disclosure.Result)
	assert.Equal(t, "ok", *disclosure.Result)
	assert.Equal(t, "userinfo", disclosure.Details["endpoint"])
	assert.Equal(t, "ok", disclosure.Details["result"])

	revoked := userConsentAuditEventByRequestID(t, events.List, "revoke-user-email")
	assert.Equal(t, "open_platform.consent.revoked", revoked.EventType)
	require.NotNil(t, revoked.Scope)
	assert.Equal(t, ScopeEmailRead, *revoked.Scope)
	assert.Equal(t, []string{ScopeEmailRead}, revoked.Scopes)
	assert.Equal(t, "user", revoked.Details["actor"])

	scoped, err := service.ListUserConsentAuditEvents(ctx, ListUserConsentAuditEventsInput{
		UserID:    userID,
		AppID:     app.ID,
		EventType: "open_platform.consent.revoked",
		Scope:     ScopeEmailRead,
		PageSize:  20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, scoped.Total)
	require.Len(t, scoped.List, 1)
	assert.Equal(t, "revoke-user-email", *scoped.List[0].RequestID)

	_, err = service.ListUserConsentAuditEvents(ctx, ListUserConsentAuditEventsInput{
		UserID:    userID,
		EventType: "open_platform.app.approved",
	})
	assert.ErrorIs(t, err, ErrInvalidAuditFilter)
}

func TestAcceptConsentKeepsChallengeAndSkipsGrantWhenOIDCRedirectFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "accept-oidc-fail-owner")
	userID := seedOpenPlatformUser(t, postgres, "accept-oidc-fail-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		State:       "accept-oidc-fail-state",
		Flow:        AuthorizeFlowCasdoor,
	})
	require.NoError(t, err)

	redirectURL, err := service.AcceptConsent(ctx, challenge.Token, "accept-oidc-fail", userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC URL builder")
	assert.Empty(t, redirectURL)

	_, err = service.LoadConsentChallenge(ctx, challenge.Token)
	require.NoError(t, err)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.granted", 0)
}

func TestAcceptConsentPersistsGrantAfterBuildingCasdoorRedirect(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	builder := &recordingOIDCAuthURLBuilder{url: "https://sso.example.com/login?state=accept-ok-state"}
	service, err := NewService(
		repo,
		redis.Client,
		WithConsentBaseURL("https://account.example.com"),
		WithOIDCAuthURLBuilder(builder),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "accept-ok-owner")
	userID := seedOpenPlatformUser(t, postgres, "accept-ok-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		State:       "accept-ok-state",
		Flow:        AuthorizeFlowCasdoor,
	})
	require.NoError(t, err)

	redirectURL, err := service.AcceptConsent(ctx, challenge.Token, "accept-ok", userID)
	require.NoError(t, err)
	assert.Equal(t, "https://sso.example.com/login?state=accept-ok-state", redirectURL)
	assert.Equal(t, app.ClientID, builder.clientID)
	assert.Equal(t, app.RedirectURIs[0], builder.redirectURI)
	assert.Equal(t, []string{"openid", "email"}, builder.scopes)
	assert.Equal(t, "accept-ok-state", builder.state)

	_, err = service.LoadConsentChallenge(ctx, challenge.Token)
	require.ErrorIs(t, err, ErrConsentTokenInvalid)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	require.Len(t, consents[0].Scopes, 1)
	assert.Equal(t, ScopeEmailRead, consents[0].Scopes[0].Scope)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.granted", 1)
}

func TestDenyConsentWritesUserDeveloperAndAdminAuditEvents(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "deny-audit-owner")
	userID := seedOpenPlatformUser(t, postgres, "deny-audit-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	}, AuthorizeRequest{
		RedirectURI: app.RedirectURIs[0],
		State:       "deny-state",
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)

	redirectURL, err := service.DenyConsent(ctx, challenge.Token, "deny-consent", userID)
	require.NoError(t, err)
	assert.Contains(t, redirectURL, "error=access_denied")
	assert.Contains(t, redirectURL, "state=deny-state")
	_, err = service.LoadConsentChallenge(ctx, challenge.Token)
	require.ErrorIs(t, err, ErrConsentTokenInvalid)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.denied", 1)

	userEvents, err := service.ListUserConsentAuditEvents(ctx, ListUserConsentAuditEventsInput{
		UserID:    userID,
		AppID:     app.ID,
		EventType: "open_platform.consent.denied",
		Scope:     ScopeEmailRead,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, userEvents.Total)
	require.Len(t, userEvents.List, 1)
	userEvent := userEvents.List[0]
	assert.Equal(t, "open_platform.consent.denied", userEvent.EventType)
	assert.Nil(t, userEvent.Scope)
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, userEvent.Scopes)
	require.NotNil(t, userEvent.Result)
	assert.Equal(t, "access_denied", *userEvent.Result)
	require.NotNil(t, userEvent.RequestID)
	assert.Equal(t, "deny-consent", *userEvent.RequestID)
	assert.Equal(t, "user", userEvent.Details["actor"])
	assert.Equal(t, "access_denied", userEvent.Details["result"])
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, userEvent.Details["scopes"])

	developerEvents, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       app.ID,
		EventType:   "open_platform.consent.denied",
		Scope:       ScopeEmailRead,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, developerEvents.Total)
	require.Len(t, developerEvents.List, 1)
	developerEvent := developerEvents.List[0]
	assert.Equal(t, "open_platform.consent.denied", developerEvent.EventType)
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, developerEvent.Scopes)
	require.NotNil(t, developerEvent.Result)
	assert.Equal(t, "access_denied", *developerEvent.Result)
	assert.Equal(t, "user", developerEvent.Details["actor"])
	assert.NotContains(t, developerEvent.Details, "userID")

	adminEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		AppID:     app.ID,
		UserID:    userID,
		EventType: "open_platform.consent.denied",
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, adminEvents.Total)
	require.Len(t, adminEvents.List, 1)
	assert.Equal(t, "access_denied", adminEvents.List[0].Metadata["result"])
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, adminEvents.List[0].Metadata["scopes"])
}

func TestDenyConsentRejectsRedirectURIDriftBeforeAuditAndDeletingChallenge(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "deny-redirect-owner")
	adminID := seedOpenPlatformUser(t, postgres, "deny-redirect-admin")
	userID := seedOpenPlatformUser(t, postgres, "deny-redirect-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)

	approveOpenPlatformRedirectURIs(t, ctx, service, app.ID, ownerID, adminID, []string{
		"https://new-client.example.com/callback",
	}, "deny-redirect-drift")

	redirectURL, err := service.DenyConsent(ctx, challenge.Token, "deny-redirect-drift", userID)
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)
	assert.Empty(t, redirectURL)

	_, err = service.LoadConsentChallenge(ctx, challenge.Token)
	require.NoError(t, err)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.denied", 0)
}

func TestConsentAndProfileCompletionPagesIncludeScopeReasons(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "reason-owner")
	userID := seedOpenPlatformUser(t, postgres, "reason-viewer")
	app := seedApprovedOpenPlatformAppWithScopeInputs(t, ctx, repo, ownerID, []ScopeRequestInput{
		{Scope: ScopeProfileBasicRead, Reason: "show avatar and name in the team roster"},
		{Scope: ScopeEmailRead, Reason: "send access receipts and incident notices"},
	})

	consentChallenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Flow:        AuthorizeFlowCasdoor,
	})
	require.NoError(t, err)

	consentPage, err := service.GetConsentPage(ctx, consentChallenge.Token, userID)
	require.NoError(t, err)
	consentScopes := scopeDefinitionsByScope(t, consentPage.Scopes)
	assert.Equal(t, "show avatar and name in the team roster", consentScopes[ScopeProfileBasicRead].Reason)
	assert.Equal(t, "send access receipts and incident notices", consentScopes[ScopeEmailRead].Reason)

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{
		ScopeProfileBasicRead,
	}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Flow:        AuthorizeFlowCasdoor,
	})
	require.NoError(t, err)

	completionPage, err := service.GetProfileCompletionPage(ctx, completionChallenge.Token, userID)
	require.NoError(t, err)
	completionScopes := scopeDefinitionsByScope(t, completionPage.Scopes)
	assert.Equal(t, "show avatar and name in the team roster", completionScopes[ScopeProfileBasicRead].Reason)
}

func TestConsentPageRejectsRedirectURIDrift(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "consent-page-redirect-owner")
	adminID := seedOpenPlatformUser(t, postgres, "consent-page-redirect-admin")
	userID := seedOpenPlatformUser(t, postgres, "consent-page-redirect-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)

	approveOpenPlatformRedirectURIs(t, ctx, service, app.ID, ownerID, adminID, []string{
		"https://new-client.example.com/callback",
	}, "consent-page-redirect-drift")

	page, err := service.GetConsentPage(ctx, challenge.Token, userID)
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)
	assert.Nil(t, page)

	_, err = service.LoadConsentChallenge(ctx, challenge.Token)
	require.NoError(t, err)
}

func TestBuildConsentChallengeRejectsStaleAppStatus(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "build-consent-stale-owner")
	adminID := seedOpenPlatformUser(t, postgres, "build-consent-stale-admin")
	userID := seedOpenPlatformUser(t, postgres, "build-consent-stale-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "integration test suspension",
		RequestID:   "build-consent-stale-suspend",
	})
	require.NoError(t, err)

	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.ErrorIs(t, err, ErrAppNotActive)
	assert.Nil(t, challenge)

	keys, err := redis.Client.Keys(ctx, consentRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestBuildProfileCompletionChallengeRejectsStaleRedirectURI(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "build-completion-redirect-owner")
	adminID := seedOpenPlatformUser(t, postgres, "build-completion-redirect-admin")
	userID := seedOpenPlatformUser(t, postgres, "build-completion-redirect-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	approveOpenPlatformRedirectURIs(t, ctx, service, app.ID, ownerID, adminID, []string{
		"https://new-client.example.com/callback",
	}, "build-completion-redirect-drift")

	challenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)
	assert.Nil(t, challenge)

	keys, err := redis.Client.Keys(ctx, completionRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestBuildConsentChallengeRejectsScopeApprovalDrift(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "build-consent-scope-owner")
	userID := seedOpenPlatformUser(t, postgres, "build-consent-scope-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	_, err = postgres.DB.Exec(ctx, `
		DELETE FROM open_platform_approved_scopes
		WHERE app_id = $1 AND scope = $2
	`, app.ID, ScopeEmailRead)
	require.NoError(t, err)

	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.ErrorIs(t, err, ErrScopeNotApproved)
	assert.Nil(t, challenge)

	keys, err := redis.Client.Keys(ctx, consentRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestUserConsentListUsageIndexesExist(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)

	activeConsentIndex := loadOpenPlatformIndexDefinition(t, ctx, postgres, "idx_open_platform_user_consents_active_user")
	assert.Contains(t, activeConsentIndex, "USING btree (user_id, app_id, scope)")
	assert.Contains(t, activeConsentIndex, "WHERE (revoked_at IS NULL)")

	activeConsentAppIndex := loadOpenPlatformIndexDefinition(t, ctx, postgres, "idx_open_platform_user_consents_active_app")
	assert.Contains(t, activeConsentAppIndex, "USING btree (app_id, user_id, scope)")
	assert.Contains(t, activeConsentAppIndex, "WHERE (revoked_at IS NULL)")

	disclosureUsageIndex := loadOpenPlatformIndexDefinition(t, ctx, postgres, "idx_open_platform_audit_events_disclosure_usage")
	assert.Contains(t, disclosureUsageIndex, "USING btree (app_id, user_id, created_at DESC, id DESC)")
	assert.Contains(t, disclosureUsageIndex, "WHERE (event_type = 'open_platform.disclosure.granted'::text)")
}

func userConsentScopeNames(scopes []UserConsentScope) []string {
	names := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		names = append(names, scope.Scope)
	}
	return names
}

func userConsentScopesByScope(t *testing.T, scopes []UserConsentScope) map[string]UserConsentScope {
	t.Helper()
	byScope := make(map[string]UserConsentScope, len(scopes))
	for _, scope := range scopes {
		require.NotContains(t, byScope, scope.Scope)
		byScope[scope.Scope] = scope
	}
	return byScope
}

func scopeDefinitionsByScope(t *testing.T, scopes []ScopeDefinition) map[string]ScopeDefinition {
	t.Helper()
	byScope := make(map[string]ScopeDefinition, len(scopes))
	for _, scope := range scopes {
		require.NotContains(t, byScope, scope.Scope)
		byScope[scope.Scope] = scope
	}
	return byScope
}

func userConsentAuditEventByRequestID(
	t *testing.T,
	events []UserConsentAuditEvent,
	requestID string,
) UserConsentAuditEvent {
	t.Helper()
	for _, event := range events {
		if event.RequestID != nil && *event.RequestID == requestID {
			return event
		}
	}
	require.Failf(t, "audit event not found", "requestID %q not found", requestID)
	return UserConsentAuditEvent{}
}

func developerAppAuditEventByRequestID(
	t *testing.T,
	events []DeveloperAppAuditEvent,
	requestID string,
) DeveloperAppAuditEvent {
	t.Helper()
	for _, event := range events {
		if event.RequestID != nil && *event.RequestID == requestID {
			return event
		}
	}
	require.Failf(t, "developer audit event not found", "requestID %q not found", requestID)
	return DeveloperAppAuditEvent{}
}

func loadOpenPlatformIndexDefinition(
	t *testing.T,
	ctx context.Context,
	fixture *postgresfixture.Fixture,
	indexName string,
) string {
	t.Helper()
	var definition string
	err := fixture.Pool.QueryRow(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND indexname = $1
	`, indexName).Scan(&definition)
	require.NoError(t, err)
	return definition
}

func TestOpenPlatformAppListIncludesScopeReviewStatus(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "app-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "other-owner")

	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Developer Portal App",
		Description:      "App submitted from developer portal.",
		HomepageURL:      "https://developer.example.com",
		PrivacyPolicyURL: "https://developer.example.com/privacy",
		RedirectURIs:     []string{"https://developer.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
			{Scope: ScopeEmailRead, Reason: "send receipts"},
		},
	})
	require.NoError(t, err)

	ownerApps, err := service.ListApps(ctx, ListAppsInput{
		OwnerUserID: ownerID,
		Status:      "all",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, ownerApps.Total)
	require.Len(t, ownerApps.List, 1)
	assert.Equal(t, registered.App.ID, ownerApps.List[0].App.ID)
	assert.Equal(t, AppStatusPending, ownerApps.List[0].App.Status)
	require.Len(t, ownerApps.List[0].Scopes, 2)
	assert.Equal(t, ScopeStatusPending, ownerApps.List[0].Scopes[0].Status)
	assert.Equal(t, ScopeStatusPending, ownerApps.List[0].Scopes[1].Status)

	otherApps, err := service.ListApps(ctx, ListAppsInput{
		OwnerUserID: otherOwnerID,
		Status:      "all",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, otherApps.Total)
	assert.Empty(t, otherApps.List)

	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeEmailRead,
		ReviewerUserID: ownerID,
		DecisionNote:   "email is required",
	}))
	approved, err := service.ApproveApp(ctx, registered.App.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, approved.ClientSecret)

	_, err = service.ApproveApp(ctx, registered.App.ID)
	require.ErrorIs(t, err, ErrInvalidAppStatus)

	adminApprovedApps, err := service.ListApps(ctx, ListAppsInput{
		Status:   AppStatusApproved,
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, adminApprovedApps.Total)
	require.Len(t, adminApprovedApps.List, 1)
	assert.Equal(t, AppStatusApproved, adminApprovedApps.List[0].App.Status)

	scopeStatuses := map[string]string{}
	for _, scope := range adminApprovedApps.List[0].Scopes {
		scopeStatuses[scope.Scope] = scope.Status
	}
	assert.Equal(t, ScopeStatusApproved, scopeStatuses[ScopeEmailRead])
	assert.Equal(t, ScopeStatusPending, scopeStatuses[ScopeProfileBasicRead])
}

func TestOpenPlatformScopeReasonRequiredForRegistrationAndImport(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "scope-reason-owner")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Missing Scope Reason App",
		Description:      "Direct API callers must explain every requested scope.",
		HomepageURL:      "https://missing-scope-reason.example.com",
		PrivacyPolicyURL: "https://missing-scope-reason.example.com/privacy",
		RedirectURIs:     []string{"https://missing-scope-reason.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: " "},
		},
	})
	require.Nil(t, registered)
	require.ErrorIs(t, err, ErrScopeReasonRequired)

	apps, err := service.ListApps(ctx, ListAppsInput{
		OwnerUserID: ownerID,
		Status:      "all",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, apps.Total)

	importService, err := NewService(repo, redis.Client, WithAppProvisioner(&fakeOpenPlatformAppProvisioner{
		existing: ProvisionedApplicationSpec{
			Name:                 "legacy-missing-scope-reason",
			DisplayName:          "Legacy Missing Scope Reason",
			HomepageURL:          "https://legacy-missing-scope-reason.example.com",
			ClientID:             "legacy-missing-scope-reason-client",
			ClientSecret:         "legacy-missing-scope-reason-secret",
			RedirectURIs:         []string{"https://legacy-missing-scope-reason.example.com/callback"},
			GrantTypes:           []string{"authorization_code"},
			TokenFormat:          "JWT-Custom",
			TokenFields:          []string{"sub"},
			ExpireInHours:        1,
			RefreshExpireInHours: 24,
		},
	}))
	require.NoError(t, err)

	imported, err := importService.ImportCasdoorApp(ctx, ImportCasdoorAppInput{
		OwnerUserID:            ownerID,
		ReviewerUserID:         ownerID,
		CasdoorApplicationName: "legacy-missing-scope-reason",
		PrivacyPolicyURL:       "https://legacy-missing-scope-reason.example.com/privacy",
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead},
		},
		RequestID: "import-missing-scope-reason",
	})
	require.Nil(t, imported)
	require.ErrorIs(t, err, ErrScopeReasonRequired)
}

func TestOpenPlatformDeveloperUpdatesAppProfile(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "profile-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "profile-other")
	adminID := seedOpenPlatformUser(t, postgres, "profile-admin")

	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Profile Editable App",
		Description:      "Original developer-facing description.",
		HomepageURL:      "https://profile-edit.example.com",
		PrivacyPolicyURL: "https://profile-edit.example.com/privacy",
		RedirectURIs:     []string{"https://profile-edit.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)

	_, err = service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      otherOwnerID,
		DisplayName:      "Wrong Owner Edit",
		HomepageURL:      "https://wrong-owner.example.com",
		PrivacyPolicyURL: "https://wrong-owner.example.com/privacy",
		Reason:           "wrong owner should not edit",
		RequestID:        "profile-wrong-owner",
	})
	require.ErrorIs(t, err, ErrAppNotFound)

	_, err = service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      ownerID,
		DisplayName:      "Profile Editable App",
		HomepageURL:      "https://profile-edit.example.com",
		PrivacyPolicyURL: "https://profile-edit.example.com/privacy",
		Reason:           " ",
		RequestID:        "profile-empty-reason",
	})
	require.ErrorIs(t, err, ErrLifecycleReasonRequired)

	_, err = service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      ownerID,
		DisplayName:      "Profile Editable App",
		HomepageURL:      "http://profile-edit.example.com",
		PrivacyPolicyURL: "https://profile-edit.example.com/privacy",
		Reason:           "invalid homepage",
		RequestID:        "profile-invalid-url",
	})
	require.ErrorIs(t, err, ErrInvalidAppProfile)

	updatedPending, err := service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      ownerID,
		DisplayName:      "Profile Editable App v2",
		Description:      "Updated before review.",
		HomepageURL:      "https://profile-v2.example.com",
		PrivacyPolicyURL: "https://profile-v2.example.com/privacy",
		Reason:           "brand name changed before review",
		RequestID:        "profile-update-pending",
	})
	require.NoError(t, err)
	assert.Equal(t, "Profile Editable App v2", updatedPending.App.DisplayName)
	assert.Equal(t, "Updated before review.", updatedPending.App.Description)
	assert.Equal(t, []string{"https://profile-edit.example.com/callback"}, updatedPending.App.RedirectURIs)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.app.profile_updated", 1)

	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "basic profile is low risk",
	}))
	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "profile-approve-app",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, approved.ClientSecret)

	updatedApproved, err := service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      ownerID,
		DisplayName:      "Profile Editable App v3",
		Description:      "Updated after approval.",
		HomepageURL:      "https://profile-v3.example.com",
		PrivacyPolicyURL: "https://profile-v3.example.com/privacy",
		Reason:           "public policy update",
		RequestID:        "profile-update-approved",
	})
	require.NoError(t, err)
	assert.Equal(t, AppStatusApproved, updatedApproved.App.Status)
	assert.Equal(t, "https://profile-v3.example.com/privacy", updatedApproved.App.PrivacyPolicyURL)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.app.profile_updated", 2)

	audits, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       registered.App.ID,
		EventType:   "open_platform.app.profile_updated",
		Page:        1,
		PageSize:    10,
	})
	require.NoError(t, err)
	require.Equal(t, 2, audits.Total)
	require.Len(t, audits.List, 2)
	assert.Equal(t, "public policy update", audits.List[0].Details["reason"])
	assert.Equal(t, "Profile Editable App v3", audits.List[0].Details["displayName"])

	_, err = service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       registered.App.ID,
		ActorUserID: adminID,
		Reason:      "retired app",
		RequestID:   "profile-revoke-app",
	})
	require.NoError(t, err)
	_, err = service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      ownerID,
		DisplayName:      "Revoked App Edit",
		HomepageURL:      "https://revoked.example.com",
		PrivacyPolicyURL: "https://revoked.example.com/privacy",
		Reason:           "revoked app should not edit",
		RequestID:        "profile-update-revoked",
	})
	require.ErrorIs(t, err, ErrInvalidAppStatus)
}

func TestApprovedAppProfileUpdateSyncsCasdoorApplication(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "profile-sync")
	registered := fixture.registered
	updated, err := service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      fixture.ownerID,
		DisplayName:      "Synced Profile App",
		Description:      "Public profile metadata synchronized to Casdoor.",
		HomepageURL:      "https://profile-sync-new.example.com",
		PrivacyPolicyURL: "https://profile-sync-new.example.com/privacy",
		Reason:           "public metadata changed",
		RequestID:        "profile-sync-update",
	})

	require.NoError(t, err)
	assert.Equal(t, "Synced Profile App", updated.App.DisplayName)
	assert.Equal(t, "https://profile-sync-new.example.com/privacy", updated.App.PrivacyPolicyURL)
	require.Len(t, provisioner.ensuredSpecs, 2)
	profileSpec := provisioner.ensuredSpecs[1]
	assert.Equal(t, registered.App.CasdoorApplicationName, profileSpec.Name)
	assert.Equal(t, "Synced Profile App", profileSpec.DisplayName)
	assert.Equal(t, "Public profile metadata synchronized to Casdoor.", profileSpec.Description)
	assert.Equal(t, "https://profile-sync-new.example.com", profileSpec.HomepageURL)
	assert.Equal(t, fixture.initialRedirectURIs, profileSpec.RedirectURIs)
	assert.Equal(t, provisioner.ensuredSpecs[0].ClientSecret, profileSpec.ClientSecret)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.ownerID, "open_platform.app.profile_updated", 1)
}

func TestApprovedAppProfileUpdateDoesNotWriteLocalWhenCasdoorSyncFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "profile-fail")
	registered := fixture.registered
	provisioner.ensureErr = errors.New("casdoor profile update unavailable")
	updated, err := service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      fixture.ownerID,
		DisplayName:      "Unsaved Profile App",
		Description:      "This update should not be persisted.",
		HomepageURL:      "https://profile-fail-new.example.com",
		PrivacyPolicyURL: "https://profile-fail-new.example.com/privacy",
		Reason:           "public metadata changed",
		RequestID:        "profile-sync-fail",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "casdoor profile update unavailable")
	assert.Nil(t, updated)
	app, err := repo.GetAppByID(ctx, registered.App.ID)
	require.NoError(t, err)
	assert.Equal(t, registered.App.DisplayName, app.DisplayName)
	assert.Equal(t, registered.App.Description, app.Description)
	assert.Equal(t, registered.App.HomepageURL, app.HomepageURL)
	assert.Equal(t, registered.App.PrivacyPolicyURL, app.PrivacyPolicyURL)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.ownerID, "open_platform.app.profile_updated", 0)
}

func TestApprovedAppProfileUpdateRollsBackCasdoorWhenLocalUpdateFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "profile-rollback")
	registered := fixture.registered
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	provisioner.onEnsure = cancel
	updated, err := service.UpdateAppProfile(requestCtx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      fixture.ownerID,
		DisplayName:      "Rolled Back Profile App",
		Description:      "Local write cancellation should roll Casdoor back.",
		HomepageURL:      "https://profile-rollback-new.example.com",
		PrivacyPolicyURL: "https://profile-rollback-new.example.com/privacy",
		Reason:           "public metadata changed",
		RequestID:        "profile-sync-rollback",
	})

	require.Error(t, err)
	assert.Nil(t, updated)
	require.Len(t, provisioner.ensuredSpecs, 3)
	assert.Equal(t, "Rolled Back Profile App", provisioner.ensuredSpecs[1].DisplayName)
	assert.Equal(t, registered.App.DisplayName, provisioner.ensuredSpecs[2].DisplayName)
	assert.Equal(t, registered.App.Description, provisioner.ensuredSpecs[2].Description)
	assert.Equal(t, registered.App.HomepageURL, provisioner.ensuredSpecs[2].HomepageURL)
	assert.Equal(t, registered.App.DisplayName, provisioner.existing.DisplayName)
	app, err := repo.GetAppByID(context.Background(), registered.App.ID)
	require.NoError(t, err)
	assert.Equal(t, registered.App.DisplayName, app.DisplayName)
	assert.Equal(t, registered.App.Description, app.Description)
	assert.Equal(t, registered.App.HomepageURL, app.HomepageURL)
	assert.Equal(t, registered.App.PrivacyPolicyURL, app.PrivacyPolicyURL)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.ownerID, "open_platform.app.profile_updated", 0)
}

func TestPrivacyOnlyProfileUpdateDoesNotRequireCasdoorSync(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "privacy-only")
	registered := fixture.registered
	provisioner.ensureErr = errors.New("casdoor should not be called")
	updated, err := service.UpdateAppProfile(ctx, UpdateAppProfileInput{
		AppID:            registered.App.ID,
		OwnerUserID:      fixture.ownerID,
		DisplayName:      registered.App.DisplayName,
		Description:      registered.App.Description,
		HomepageURL:      registered.App.HomepageURL,
		PrivacyPolicyURL: "https://privacy-only-new.example.com/privacy",
		Reason:           "privacy policy moved",
		RequestID:        "profile-privacy-only",
	})

	require.NoError(t, err)
	assert.Equal(t, "https://privacy-only-new.example.com/privacy", updated.App.PrivacyPolicyURL)
	require.Len(t, provisioner.ensuredSpecs, 1)
	assert.Equal(t, registered.App.DisplayName, provisioner.existing.DisplayName)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.ownerID, "open_platform.app.profile_updated", 1)
}

func TestOpenPlatformScopeChangeReviewAndResubmit(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "scope-change-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "scope-change-other")
	adminID := seedOpenPlatformUser(t, postgres, "scope-change-admin")

	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Scope Change App",
		Description:      "App that requests additional scopes after creation.",
		HomepageURL:      "https://scope-change.example.com",
		PrivacyPolicyURL: "https://scope-change.example.com/privacy",
		RedirectURIs:     []string{"https://scope-change.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)

	_, err = service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: otherOwnerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: "wrong owner"},
		},
		RequestID: "scope-change-wrong-owner",
	})
	require.ErrorIs(t, err, ErrAppNotFound)

	_, err = service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: " "},
		},
		RequestID: "scope-change-empty-reason",
	})
	require.ErrorIs(t, err, ErrScopeReasonRequired)

	requested, err := service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: "send security notifications"},
		},
		RequestID: "scope-change-request",
	})
	require.NoError(t, err)
	require.Len(t, requested.Scopes, 1)
	assert.Equal(t, ScopeEmailRead, requested.Scopes[0].Scope)
	assert.Equal(t, ScopeStatusPending, requested.Scopes[0].Status)
	assert.Equal(t, "send security notifications", requested.Scopes[0].Reason)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.scope.requested", 1)

	_, err = service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: "duplicate pending request"},
		},
		RequestID: "scope-change-pending-again",
	})
	require.ErrorIs(t, err, ErrScopeAlreadyPending)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.scope.requested", 1)

	withdrawn, err := service.WithdrawScopeRequest(ctx, ScopeWithdrawalInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scope:       ScopeEmailRead,
		Reason:      "need to narrow scope purpose first",
		RequestID:   "scope-change-withdraw",
	})
	require.NoError(t, err)
	assert.Equal(t, ScopeStatusWithdrawn, withdrawn.Status)
	require.NotNil(t, withdrawn.DecisionNote)
	assert.Equal(t, "need to narrow scope purpose first", *withdrawn.DecisionNote)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.scope.withdrawn", 1)

	require.ErrorIs(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeEmailRead,
		ReviewerUserID: adminID,
		RequestID:      "scope-change-approve-withdrawn",
	}), ErrInvalidScope)

	requestedAfterWithdraw, err := service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: "send security notifications after narrowing"},
		},
		RequestID: "scope-change-request-after-withdraw",
	})
	require.NoError(t, err)
	require.Len(t, requestedAfterWithdraw.Scopes, 1)
	assert.Equal(t, ScopeStatusPending, requestedAfterWithdraw.Scopes[0].Status)
	assert.Nil(t, requestedAfterWithdraw.Scopes[0].ReviewerUserID)
	assert.Nil(t, requestedAfterWithdraw.Scopes[0].ReviewedAt)
	assert.Nil(t, requestedAfterWithdraw.Scopes[0].DecisionNote)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.scope.requested", 2)

	require.NoError(t, service.RejectScope(ctx, RejectScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeEmailRead,
		ReviewerUserID: adminID,
		DecisionNote:   "purpose is too broad",
		RequestID:      "scope-change-reject",
	}))
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.scope.rejected", 1)

	reopened, err := service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: "send account security notifications only"},
		},
		RequestID: "scope-change-resubmit",
	})
	require.NoError(t, err)
	require.Len(t, reopened.Scopes, 1)
	assert.Equal(t, ScopeStatusPending, reopened.Scopes[0].Status)
	assert.Equal(t, "send account security notifications only", reopened.Scopes[0].Reason)
	assert.Nil(t, reopened.Scopes[0].ReviewerUserID)
	assert.Nil(t, reopened.Scopes[0].ReviewedAt)
	assert.Nil(t, reopened.Scopes[0].DecisionNote)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.scope.requested", 3)

	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeEmailRead,
		ReviewerUserID: adminID,
		DecisionNote:   "narrow purpose accepted",
		RequestID:      "scope-change-approve",
	}))
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.scope.approved", 1)

	approvedScopes, err := repo.ListApprovedScopes(ctx, registered.App.ID)
	require.NoError(t, err)
	assert.Contains(t, approvedScopes, ScopeEmailRead)

	_, err = service.RequestScopeChange(ctx, ScopeChangeInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Scopes: []ScopeRequestInput{
			{Scope: ScopeEmailRead, Reason: "request already approved scope again"},
		},
		RequestID: "scope-change-approved-again",
	})
	require.ErrorIs(t, err, ErrScopeAlreadyApproved)
}

func TestApproveAppProvisionsMinimizedCasdoorApplication(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "approval-provision-owner")
	adminID := seedOpenPlatformUser(t, postgres, "approval-provision-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Provisioned App",
		Description:      "Needs Casdoor app provisioning.",
		HomepageURL:      "https://provision.example.com",
		PrivacyPolicyURL: "https://provision.example.com/privacy",
		RedirectURIs:     []string{"https://provision.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))

	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-provisioned-app",
	})

	require.NoError(t, err)
	require.NotEmpty(t, approved.ClientSecret)
	require.NotNil(t, provisioner.ensured)
	assert.Equal(t, registered.App.CasdoorApplicationName, provisioner.ensured.Name)
	assert.Equal(t, registered.App.ClientID, provisioner.ensured.ClientID)
	assert.Equal(t, []string{"authorization_code"}, provisioner.ensured.GrantTypes)
	assert.Equal(t, []string{}, provisioner.ensured.TokenFields)
	assert.Equal(t, "JWT-Custom", provisioner.ensured.TokenFormat)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.token_probe.passed", 1)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.approved", 1)
}

func TestApproveAppRecordsRuntimeTokenProbeEvidence(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	runtimeProber := &fakeRuntimeTokenProber{
		result: RuntimeTokenMinimizationProbeResult{
			Method: "authorization_code",
			TokenClaims: map[string][]string{
				"id_token":     {"sub", "iss"},
				"access_token": {"sub"},
			},
			Metadata: map[string]any{
				"source":      "integration-test",
				"accessToken": "must-not-be-persisted",
			},
		},
	}
	service, err := NewService(
		repo,
		redis.Client,
		WithAppProvisioner(provisioner),
		WithRuntimeTokenProbe(runtimeProber, true),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "runtime-probe-owner")
	adminID := seedOpenPlatformUser(t, postgres, "runtime-probe-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Runtime Probe App",
		Description:      "Needs runtime token evidence.",
		HomepageURL:      "https://runtime-probe.example.com",
		PrivacyPolicyURL: "https://runtime-probe.example.com/privacy",
		RedirectURIs:     []string{"https://runtime-probe.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))

	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-runtime-probe",
	})

	require.NoError(t, err)
	require.NotEmpty(t, approved.ClientSecret)
	require.NotNil(t, runtimeProber.spec)
	assert.Equal(t, registered.App.ClientID, runtimeProber.spec.ClientID)
	assert.Equal(t, []string{"authorization_code"}, runtimeProber.spec.GrantTypes)

	evidence := latestOpenPlatformTokenProbeEvidence(t, postgres, registered.App.ID)
	assert.Equal(t, "passed", evidence.Result)
	assert.Equal(t, []string{"iss", "sub"}, evidence.InspectedClaims)
	assert.Empty(t, evidence.BusinessClaims)
	assert.Equal(t, []string{"iss", "sub"}, evidence.TokenClaims["id_token"])
	assert.Equal(t, "integration-test", evidence.Metadata["source"])
	assert.NotContains(t, evidence.Metadata, "accessToken")
	assert.Empty(t, evidence.Error)

	evidenceList, err := service.ListTokenProbeEvidence(ctx, ListTokenProbeEvidenceInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		Result:         "passed",
		ClientID:       registered.App.ClientID,
		Page:           1,
		PageSize:       20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, evidenceList.Total)
	require.Len(t, evidenceList.List, 1)
	assert.Equal(t, evidence.Result, evidenceList.List[0].Result)
	require.NotNil(t, evidenceList.List[0].ReviewerUserID)
	assert.Equal(t, adminID, *evidenceList.List[0].ReviewerUserID)
	require.NotNil(t, evidenceList.List[0].RequestID)
	assert.Equal(t, "approve-runtime-probe", *evidenceList.List[0].RequestID)

	_, err = service.ListTokenProbeEvidence(ctx, ListTokenProbeEvidenceInput{Result: "unsafe"})
	require.ErrorIs(t, err, ErrInvalidTokenProbeFilter)

	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.token_probe.runtime.passed", 1)
	assertOpenPlatformAuditMetadata(t, postgres, registered.App.ID, adminID, "open_platform.app.token_probe.runtime.passed", map[string]any{
		"probeType": "runtime_code_flow",
		"result":    "passed",
	})
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.approved", 1)
}

func TestApproveAppBlocksUnsafeRuntimeTokenProbe(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	runtimeProber := &fakeRuntimeTokenProber{
		result: RuntimeTokenMinimizationProbeResult{
			TokenClaims: map[string][]string{
				"id_token": {"sub", "phoneNumber"},
			},
		},
	}
	service, err := NewService(
		repo,
		redis.Client,
		WithAppProvisioner(provisioner),
		WithRuntimeTokenProbe(runtimeProber, true),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "runtime-unsafe-owner")
	adminID := seedOpenPlatformUser(t, postgres, "runtime-unsafe-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Unsafe Runtime Probe App",
		Description:      "Runtime probe returns unsafe claims.",
		HomepageURL:      "https://runtime-unsafe.example.com",
		PrivacyPolicyURL: "https://runtime-unsafe.example.com/privacy",
		RedirectURIs:     []string{"https://runtime-unsafe.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))

	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-runtime-unsafe",
	})

	require.Nil(t, approved)
	require.ErrorIs(t, err, ErrTokenMinimizationProbe)
	evidence := latestOpenPlatformTokenProbeEvidence(t, postgres, registered.App.ID)
	assert.Equal(t, "failed", evidence.Result)
	assert.Equal(t, []string{"phone_number"}, evidence.BusinessClaims)
	assert.Contains(t, evidence.Error, "forbidden business claims")
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.token_probe.runtime.failed", 1)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.approved", 0)
	assert.Equal(t, []string{registered.App.CasdoorApplicationName}, provisioner.deleted)
	app, err := repo.GetAppByID(ctx, registered.App.ID)
	require.NoError(t, err)
	assert.Equal(t, AppStatusPending, app.Status)
}

func TestApproveAppRequiresRuntimeTokenProbeWhenConfigured(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(
		repo,
		redis.Client,
		WithAppProvisioner(provisioner),
		WithRuntimeTokenProbe(nil, true),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "runtime-required-owner")
	adminID := seedOpenPlatformUser(t, postgres, "runtime-required-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Required Runtime Probe App",
		Description:      "Runtime probe is required.",
		HomepageURL:      "https://runtime-required.example.com",
		PrivacyPolicyURL: "https://runtime-required.example.com/privacy",
		RedirectURIs:     []string{"https://runtime-required.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))

	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-runtime-required",
	})

	require.Nil(t, approved)
	require.ErrorIs(t, err, ErrTokenMinimizationProbe)
	evidence := latestOpenPlatformTokenProbeEvidence(t, postgres, registered.App.ID)
	assert.Equal(t, "failed", evidence.Result)
	assert.Contains(t, evidence.Error, "runtime code-flow probe runner is not configured")
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.token_probe.runtime.failed", 1)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.approved", 0)
	assert.Equal(t, []string{registered.App.CasdoorApplicationName}, provisioner.deleted)
}

func TestApproveAppDeletesCasdoorApplicationWhenLocalApprovalFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "approval-local-fail-owner")
	adminID := seedOpenPlatformUser(t, postgres, "approval-local-fail-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Approval Local Failure App",
		Description:      "Local approval update fails after Casdoor provisioning.",
		HomepageURL:      "https://approval-local-fail.example.com",
		PrivacyPolicyURL: "https://approval-local-fail.example.com/privacy",
		RedirectURIs:     []string{"https://approval-local-fail.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	provisioner.onEnsure = func() {
		_, updateErr := postgres.DB.Exec(ctx, `
			UPDATE open_platform_apps
			SET status = $1, updated_at = NOW()
			WHERE id = $2
		`, AppStatusRevoked, registered.App.ID)
		require.NoError(t, updateErr)
	}

	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-local-fail",
	})

	require.Nil(t, approved)
	require.ErrorIs(t, err, ErrInvalidAppStatus)
	assert.Equal(t, []string{registered.App.CasdoorApplicationName}, provisioner.deleted)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.approved", 0)
	assertOpenPlatformAppStatus(t, ctx, repo, registered.App.ID, AppStatusRevoked)
}

func TestApproveAppCleanupSurvivesRequestCancellationAfterCasdoorProvision(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "approval-cancel-owner")
	adminID := seedOpenPlatformUser(t, postgres, "approval-cancel-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Approval Cancel App",
		Description:      "Request is cancelled after Casdoor provisioning.",
		HomepageURL:      "https://approval-cancel.example.com",
		PrivacyPolicyURL: "https://approval-cancel.example.com/privacy",
		RedirectURIs:     []string{"https://approval-cancel.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	provisioner.onEnsure = cancel

	approved, err := service.ApproveAppWithAudit(requestCtx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-cancel-after-provision",
	})

	require.Nil(t, approved)
	require.Error(t, err)
	assert.Equal(t, []string{registered.App.CasdoorApplicationName}, provisioner.deleted)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.approved", 0)
	assertOpenPlatformAppStatus(t, context.Background(), repo, registered.App.ID, AppStatusPending)
}

func TestRevokePendingAppDoesNotDeleteCasdoorApplication(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "revoke-pending-casdoor-owner")
	adminID := seedOpenPlatformUser(t, postgres, "revoke-pending-casdoor-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Pending Revoke App",
		Description:      "Pending app has no provisioned Casdoor application.",
		HomepageURL:      "https://pending-revoke.example.com",
		PrivacyPolicyURL: "https://pending-revoke.example.com/privacy",
		RedirectURIs:     []string{"https://pending-revoke.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)

	revoked, err := service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       registered.App.ID,
		ActorUserID: adminID,
		Reason:      "reject before provisioning",
		RequestID:   "revoke-pending-no-casdoor-delete",
	})

	require.NoError(t, err)
	assert.Equal(t, AppStatusRevoked, revoked.App.Status)
	assert.Empty(t, provisioner.deleted)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.revoked", 1)
}

func TestRevokeAppDeletesProvisionedCasdoorApplication(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "revoke-casdoor-owner")
	adminID := seedOpenPlatformUser(t, postgres, "revoke-casdoor-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Revoke Casdoor App",
		Description:      "Deletes provisioned Casdoor application on revoke.",
		HomepageURL:      "https://revoke-casdoor.example.com",
		PrivacyPolicyURL: "https://revoke-casdoor.example.com/privacy",
		RedirectURIs:     []string{"https://revoke-casdoor.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	_, err = service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-before-casdoor-delete",
	})
	require.NoError(t, err)

	revoked, err := service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       registered.App.ID,
		ActorUserID: adminID,
		Reason:      "delete provisioned Casdoor application",
		RequestID:   "revoke-delete-casdoor",
	})

	require.NoError(t, err)
	assert.Equal(t, AppStatusRevoked, revoked.App.Status)
	assert.Equal(t, []string{registered.App.CasdoorApplicationName}, provisioner.deleted)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.revoked", 1)
}

func TestRevokeAppCanRetryCasdoorApplicationCleanupAfterDeleteFailure(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "revoke-casdoor-retry-owner")
	adminID := seedOpenPlatformUser(t, postgres, "revoke-casdoor-retry-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Revoke Casdoor Retry App",
		Description:      "Retries Casdoor application cleanup after failure.",
		HomepageURL:      "https://revoke-casdoor-retry.example.com",
		PrivacyPolicyURL: "https://revoke-casdoor-retry.example.com/privacy",
		RedirectURIs:     []string{"https://revoke-casdoor-retry.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	_, err = service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-before-casdoor-delete-retry",
	})
	require.NoError(t, err)

	provisioner.deleteErr = errors.New("casdoor delete unavailable")
	_, err = service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       registered.App.ID,
		ActorUserID: adminID,
		Reason:      "first delete fails",
		RequestID:   "revoke-delete-casdoor-fails",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "casdoor delete unavailable")
	assert.Empty(t, provisioner.deleted)
	assertOpenPlatformAppStatus(t, ctx, repo, registered.App.ID, AppStatusRevoked)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.revoked", 1)

	provisioner.deleteErr = nil
	retried, err := service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       registered.App.ID,
		ActorUserID: adminID,
		Reason:      "retry Casdoor cleanup",
		RequestID:   "revoke-delete-casdoor-retry",
	})

	require.NoError(t, err)
	assert.Equal(t, AppStatusRevoked, retried.App.Status)
	assert.Equal(t, []string{registered.App.CasdoorApplicationName}, provisioner.deleted)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, adminID, "open_platform.app.revoked", 1)
}

func TestImportCasdoorAppRejectsBusinessTokenFields(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{
		existing: ProvisionedApplicationSpec{
			Name:                 "legacy-bad-claims",
			DisplayName:          "Legacy Bad Claims",
			HomepageURL:          "https://legacy.example.com",
			Description:          "Legacy app with unsafe token fields.",
			ClientID:             "legacy-bad-client",
			ClientSecret:         "legacy-bad-secret",
			RedirectURIs:         []string{"https://legacy.example.com/callback"},
			GrantTypes:           []string{"authorization_code"},
			TokenFormat:          "JWT",
			TokenFields:          []string{"sub", "phone_verified", "stuhelper_student_verified"},
			ExpireInHours:        1,
			RefreshExpireInHours: 24,
		},
	}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "import-bad-claims-admin")
	imported, err := service.ImportCasdoorApp(ctx, ImportCasdoorAppInput{
		OwnerUserID:            adminID,
		ReviewerUserID:         adminID,
		CasdoorApplicationName: "legacy-bad-claims",
		PrivacyPolicyURL:       "https://legacy.example.com/privacy",
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "legacy profile"},
		},
		RequestID: "import-bad-claims",
	})

	require.Nil(t, imported)
	require.ErrorIs(t, err, ErrTokenMinimizationProbe)
	assertOpenPlatformAuditMetadata(t, postgres, 0, adminID, "open_platform.app.token_probe.failed", map[string]any{
		"result": "failed",
	})
}

func TestOpenIDOnlyAuthorizationSkipsConsentAndBusinessDisclosure(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "openid-owner")
	userID := seedOpenPlatformUser(t, postgres, "openid-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})

	decision, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid"},
		Flow:        AuthorizeFlowAccount,
	}, userID)
	require.NoError(t, err)
	assert.Empty(t, decision.Scopes)
	assert.Equal(t, []string{"openid"}, decision.OAuthScopes)
	assert.Empty(t, decision.ConsentURL)
	assert.Empty(t, decision.ProfileCompletionURL)

	payload, err := service.UserInfoForIdentityToken(ctx, app.ClientID, userID, "stuhelper:openid-viewer", []string{"openid"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"sub": "stuhelper:openid-viewer"}, payload)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.granted", 1)
}

func TestOfflineAccessRequiresConsentAndControlsIdentityTokenActivity(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "offline-owner")
	userID := seedOpenPlatformUser(t, postgres, "offline-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeOfflineAccess})

	decision, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", ScopeOfflineAccess},
		Flow:        AuthorizeFlowAccount,
	}, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, decision.ConsentURL)
	assert.Empty(t, decision.ProfileCompletionURL)
	assert.Equal(t, []string{ScopeOfflineAccess}, decision.Scopes)
	assert.Equal(t, []string{"openid", ScopeOfflineAccess}, decision.OAuthScopes)

	consentToken := queryValueFromURL(t, decision.ConsentURL, "token")
	page, err := service.GetConsentPage(ctx, consentToken, userID)
	require.NoError(t, err)
	require.Len(t, page.Scopes, 1)
	assert.Equal(t, ScopeOfflineAccess, page.Scopes[0].Scope)
	assert.Equal(t, "离线访问", page.Scopes[0].DisplayName)

	challenge, err := service.GrantConsent(ctx, consentToken, "offline-access-consent")
	require.NoError(t, err)
	assert.Equal(t, []string{ScopeOfflineAccess}, challenge.ConsentScopes)
	assert.Equal(t, []string{"openid", ScopeOfflineAccess}, challenge.OAuthScopes)

	active, err := service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", ScopeOfflineAccess})
	require.NoError(t, err)
	assert.True(t, active)

	require.NoError(t, service.RevokeUserConsent(ctx, RevokeConsentInput{
		UserID:    userID,
		AppID:     app.ID,
		Scopes:    []string{ScopeOfflineAccess},
		RequestID: "offline-access-revoke",
	}))
	active, err = service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", ScopeOfflineAccess})
	require.NoError(t, err)
	assert.False(t, active)
}

func TestPromptNoneReturnsInteractionErrorsWithoutChallenges(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "prompt-none-owner")
	userID := seedOpenPlatformUser(t, postgres, "prompt-none-viewer")
	emailApp := seedApprovedOpenPlatformAppWithName(t, ctx, repo, ownerID, "Prompt None Email", []string{ScopeEmailRead})

	consentDecision, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    emailApp.ClientID,
		RedirectURI: emailApp.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
		PromptNone:  true,
	}, userID)
	require.NoError(t, err)
	assert.True(t, consentDecision.InteractionRequired)
	assert.Equal(t, "consent_required", consentDecision.InteractionError)
	assert.Empty(t, consentDecision.ConsentURL)
	assert.Empty(t, consentDecision.ProfileCompletionURL)
	assert.Equal(t, []string{ScopeEmailRead}, consentDecision.Scopes)
	assert.Equal(t, []string{"openid", "email"}, consentDecision.OAuthScopes)

	profileApp := seedApprovedOpenPlatformAppWithName(t, ctx, repo, ownerID, "Prompt None Profile", []string{ScopeProfileBasicRead})
	profileDecision, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    profileApp.ClientID,
		RedirectURI: profileApp.RedirectURIs[0],
		Scopes:      []string{"openid", "profile"},
		Flow:        AuthorizeFlowAccount,
		PromptNone:  true,
	}, userID)
	require.NoError(t, err)
	assert.True(t, profileDecision.InteractionRequired)
	assert.Equal(t, "interaction_required", profileDecision.InteractionError)
	assert.NotEmpty(t, profileDecision.MissingFields)
	assert.Empty(t, profileDecision.ConsentURL)
	assert.Empty(t, profileDecision.ProfileCompletionURL)

	consentKeys, err := redis.Client.Keys(ctx, consentRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, consentKeys)
	completionKeys, err := redis.Client.Keys(ctx, completionRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, completionKeys)
}

func TestPromptConsentForcesConsentChallenge(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "prompt-consent-owner")
	userID := seedOpenPlatformUser(t, postgres, "prompt-consent-viewer")
	app := seedApprovedOpenPlatformAppWithName(t, ctx, repo, ownerID, "Prompt Consent Email", []string{ScopeEmailRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "test-seed",
		RequestID:   "seed-existing-consent",
	}, []string{ScopeEmailRead}))

	alreadyConsented, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	}, userID)
	require.NoError(t, err)
	assert.Empty(t, alreadyConsented.ConsentURL)
	assert.Empty(t, alreadyConsented.ProfileCompletionURL)
	assert.Equal(t, []string{ScopeEmailRead}, alreadyConsented.Scopes)

	forced, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:     app.ClientID,
		RedirectURI:  app.RedirectURIs[0],
		Scopes:       []string{"openid", "email"},
		Flow:         AuthorizeFlowAccount,
		ForceConsent: true,
	}, userID)
	require.NoError(t, err)
	require.NotEmpty(t, forced.ConsentURL)
	assert.Empty(t, forced.ProfileCompletionURL)
	assert.Equal(t, []string{ScopeEmailRead}, forced.Scopes)
	assert.Equal(t, []string{"openid", "email"}, forced.OAuthScopes)

	consentToken := queryValueFromURL(t, forced.ConsentURL, "token")
	page, err := service.GetConsentPage(ctx, consentToken, userID)
	require.NoError(t, err)
	require.Len(t, page.Scopes, 1)
	assert.Equal(t, ScopeEmailRead, page.Scopes[0].Scope)

	challenge, err := service.GrantConsent(ctx, consentToken, "forced-consent")
	require.NoError(t, err)
	assert.Equal(t, []string{ScopeEmailRead}, challenge.ConsentScopes)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.granted", 2)
}

func TestResourceScopesDoNotCreateUserConsent(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "resource-scope-owner")
	userID := seedOpenPlatformUser(t, postgres, "resource-scope-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeEmailRead,
		ScopeResourceRead,
		ScopeResourceWrite,
	})

	decision, err := service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", ScopeResourceRead},
		Flow:        AuthorizeFlowAccount,
	}, userID)
	require.NoError(t, err)
	assert.Empty(t, decision.ConsentURL)
	assert.Empty(t, decision.ProfileCompletionURL)
	assert.Equal(t, []string{ScopeResourceRead}, decision.Scopes)
	assert.Equal(t, []string{"openid", ScopeResourceRead}, decision.OAuthScopes)

	active, err := service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", ScopeResourceRead})
	require.NoError(t, err)
	assert.True(t, active)

	payload, err := service.UserInfoForIdentityToken(ctx, app.ClientID, userID, "stuhelper:resource-scope-viewer", []string{"openid", ScopeResourceRead})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"sub": "stuhelper:resource-scope-viewer"}, payload)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)

	decision, err = service.BeginAuthorization(ctx, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email", ScopeResourceRead, ScopeResourceWrite},
		Flow:        AuthorizeFlowAccount,
	}, userID)
	require.NoError(t, err)
	require.NotEmpty(t, decision.ConsentURL)
	assert.Equal(t, []string{ScopeEmailRead}, decision.Scopes)
	assert.Equal(t, []string{"openid", "email", ScopeResourceRead, ScopeResourceWrite}, decision.OAuthScopes)

	consentToken := queryValueFromURL(t, decision.ConsentURL, "token")
	page, err := service.GetConsentPage(ctx, consentToken, userID)
	require.NoError(t, err)
	require.Len(t, page.Scopes, 1)
	assert.Equal(t, ScopeEmailRead, page.Scopes[0].Scope)

	challenge, err := service.GrantConsent(ctx, consentToken, "resource-scope-consent")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{ScopeEmailRead, ScopeResourceRead, ScopeResourceWrite}, challenge.Scopes)
	assert.Equal(t, []string{"openid", "email", ScopeResourceRead, ScopeResourceWrite}, challenge.OAuthScopes)
	assert.Equal(t, []string{ScopeEmailRead}, challenge.ConsentScopes)

	consents, err = service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	require.Len(t, consents[0].Scopes, 1)
	assert.Equal(t, ScopeEmailRead, consents[0].Scopes[0].Scope)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.granted", 1)

	active, err = service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", "email", ScopeResourceWrite})
	require.NoError(t, err)
	assert.True(t, active)
}

func TestProfileCompletionPreservesOAuthScopesForConsentChallenge(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-scope-owner")
	userID := seedOpenPlatformUser(t, postgres, "completion-scope-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeEmailRead,
		ScopeResourceRead,
	})

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{
		ScopeEmailRead,
		ScopeResourceRead,
	}, AuthorizeRequest{
		ClientID:            app.ClientID,
		RedirectURI:         app.RedirectURIs[0],
		Scopes:              []string{"openid", "email", ScopeResourceRead},
		State:               "completion-oauth-scope-state",
		Flow:                AuthorizeFlowAccount,
		CodeChallenge:       "test-code-challenge",
		CodeChallengeMethod: "S256",
		Nonce:               "completion-oauth-scope-nonce",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "email", ScopeResourceRead}, completionChallenge.OAuthScopes)

	result, err := service.ContinueProfileCompletion(ctx, completionChallenge.Token, userID)
	require.NoError(t, err)
	require.NotEmpty(t, result.ConsentURL)

	consentToken := queryValueFromURL(t, result.ConsentURL, "token")
	consentChallenge, err := service.LoadConsentChallenge(ctx, consentToken)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{ScopeEmailRead, ScopeResourceRead}, consentChallenge.Scopes)
	assert.Equal(t, []string{"openid", "email", ScopeResourceRead}, consentChallenge.OAuthScopes)
	assert.Equal(t, []string{ScopeEmailRead}, consentChallenge.ConsentScopes)
	assert.Equal(t, "completion-oauth-scope-state", consentChallenge.State)
	assert.Equal(t, "completion-oauth-scope-nonce", consentChallenge.Nonce)
}

func TestProfileCompletionConsentChallengeCleanupSurvivesRequestCancellation(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redisFixture := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redisFixture.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-cancel-owner")
	userID := seedOpenPlatformUser(t, postgres, "completion-cancel-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		State:       "completion-cancel-state",
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	redisFixture.Client.AddHook(cancelAfterRedisSetHook{
		keyPrefix: consentRedisPrefix,
		cancel:    cancel,
	})

	result, err := service.ContinueProfileCompletion(requestCtx, completionChallenge.Token, userID)

	require.Error(t, err)
	require.Nil(t, result)
	consentKeys, err := redisFixture.Client.Keys(context.Background(), consentRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, consentKeys)
	_, err = service.LoadProfileCompletionChallenge(context.Background(), completionChallenge.Token)
	require.NoError(t, err)
}

func TestProfileCompletionContinueRejectsRedirectURIDrift(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-redirect-owner")
	adminID := seedOpenPlatformUser(t, postgres, "completion-redirect-admin")
	userID := seedOpenPlatformUser(t, postgres, "completion-redirect-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)

	approveOpenPlatformRedirectURIs(t, ctx, service, app.ID, ownerID, adminID, []string{
		"https://new-client.example.com/callback",
	}, "completion-redirect-drift")

	result, err := service.ContinueProfileCompletion(ctx, completionChallenge.Token, userID)
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)
	assert.Nil(t, result)

	_, err = service.LoadProfileCompletionChallenge(ctx, completionChallenge.Token)
	require.NoError(t, err)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
	keys, err := redis.Client.Keys(ctx, consentRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestProfileCompletionContinueRejectsInactiveApp(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-inactive-owner")
	adminID := seedOpenPlatformUser(t, postgres, "completion-inactive-admin")
	userID := seedOpenPlatformUser(t, postgres, "completion-inactive-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)
	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "integration test suspension",
		RequestID:   "completion-inactive-suspend",
	})
	require.NoError(t, err)

	result, err := service.ContinueProfileCompletion(ctx, completionChallenge.Token, userID)
	require.ErrorIs(t, err, ErrAppNotActive)
	assert.Nil(t, result)

	_, err = service.LoadProfileCompletionChallenge(ctx, completionChallenge.Token)
	require.NoError(t, err)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
}

func TestProfileCompletionContinueRejectsScopeApprovalDrift(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-scope-drift-owner")
	userID := seedOpenPlatformUser(t, postgres, "completion-scope-drift-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)
	_, err = postgres.DB.Exec(ctx, `
		DELETE FROM open_platform_approved_scopes
		WHERE app_id = $1 AND scope = $2
	`, app.ID, ScopeEmailRead)
	require.NoError(t, err)

	result, err := service.ContinueProfileCompletion(ctx, completionChallenge.Token, userID)
	require.ErrorIs(t, err, ErrScopeNotApproved)
	assert.Nil(t, result)

	_, err = service.LoadProfileCompletionChallenge(ctx, completionChallenge.Token)
	require.NoError(t, err)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
}

func TestProfileCompletionContinueKeepsChallengeWhenOIDCRedirectFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-oidc-owner")
	userID := seedOpenPlatformUser(t, postgres, "completion-oidc-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "test-seed",
		RequestID:   "completion-oidc-existing-consent",
	}, []string{ScopeEmailRead}))

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowCasdoor,
	})
	require.NoError(t, err)

	result, err := service.ContinueProfileCompletion(ctx, completionChallenge.Token, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC URL builder")
	assert.Nil(t, result)

	_, err = service.LoadProfileCompletionChallenge(ctx, completionChallenge.Token)
	require.NoError(t, err)
	keys, err := redis.Client.Keys(ctx, consentRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestProfileCompletionRedirectCleanupSurvivesRequestCancellation(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redisFixture := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	builder := &recordingOIDCAuthURLBuilder{url: "https://sso.example.com/login?state=completion-redirect-cancel"}
	service, err := NewService(
		repo,
		redisFixture.Client,
		WithConsentBaseURL("https://account.example.com"),
		WithOIDCAuthURLBuilder(builder),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "completion-redirect-cancel-owner")
	userID := seedOpenPlatformUser(t, postgres, "completion-redirect-cancel-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "test-seed",
		RequestID:   "completion-redirect-cancel-existing-consent",
	}, []string{ScopeEmailRead}))

	completionChallenge, err := service.BuildProfileCompletionChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		State:       "completion-redirect-cancel",
		Flow:        AuthorizeFlowCasdoor,
	})
	require.NoError(t, err)
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	redisFixture.Client.AddHook(cancelBeforeRedisDelHook{
		keyPrefix: completionRedisPrefix,
		cancel:    cancel,
	})

	result, err := service.ContinueProfileCompletion(requestCtx, completionChallenge.Token, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://sso.example.com/login?state=completion-redirect-cancel", result.RedirectURL)
	assert.Equal(t, app.ClientID, builder.clientID)
	assert.Equal(t, app.RedirectURIs[0], builder.redirectURI)
	assert.Equal(t, []string{"openid", "email"}, builder.scopes)
	assert.Equal(t, "completion-redirect-cancel", builder.state)
	_, err = service.LoadProfileCompletionChallenge(context.Background(), completionChallenge.Token)
	require.ErrorIs(t, err, ErrCompletionTokenInvalid)
}

func TestGrantConsentRejectsRedirectURIDriftBeforePersistingConsent(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "grant-redirect-owner")
	adminID := seedOpenPlatformUser(t, postgres, "grant-redirect-admin")
	userID := seedOpenPlatformUser(t, postgres, "grant-redirect-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	challenge, err := service.BuildConsentChallenge(ctx, app, userID, []string{ScopeEmailRead}, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: app.RedirectURIs[0],
		Scopes:      []string{"openid", "email"},
		Flow:        AuthorizeFlowAccount,
	})
	require.NoError(t, err)

	approveOpenPlatformRedirectURIs(t, ctx, service, app.ID, ownerID, adminID, []string{
		"https://new-client.example.com/callback",
	}, "grant-redirect-drift")

	granted, err := service.GrantConsent(ctx, challenge.Token, "grant-redirect-drift")
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)
	assert.Nil(t, granted)

	_, err = service.LoadConsentChallenge(ctx, challenge.Token)
	require.NoError(t, err)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
}

func TestIdentityAccessTokenActiveReflectsCurrentConsentAndAppStatus(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "identity-token-owner")
	adminID := seedOpenPlatformUser(t, postgres, "identity-token-admin")
	userID := seedOpenPlatformUser(t, postgres, "identity-token-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "identity-token-grant",
	}, []string{ScopeProfileBasicRead, ScopeEmailRead}))

	active, err := service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", "profile", "email"})
	require.NoError(t, err)
	assert.True(t, active)
	initialFingerprint, active, err := service.IdentityAuthorizationFingerprint(ctx, app.ClientID, userID, []string{"openid", "profile", "email"})
	require.NoError(t, err)
	assert.True(t, active)
	assert.NotEmpty(t, initialFingerprint)

	active, err = service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid"})
	require.NoError(t, err)
	assert.True(t, active)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.granted", 0)

	require.NoError(t, service.RevokeUserConsent(ctx, RevokeConsentInput{
		UserID:    userID,
		AppID:     app.ID,
		Scopes:    []string{ScopeEmailRead},
		RequestID: "identity-token-revoke-email",
	}))
	active, err = service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", "profile", "email"})
	require.NoError(t, err)
	assert.False(t, active)

	active, err = service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid", "profile"})
	require.NoError(t, err)
	assert.True(t, active)
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "identity-token-regrant-email",
	}, []string{ScopeEmailRead}))
	refreshedFingerprint, active, err := service.IdentityAuthorizationFingerprint(ctx, app.ClientID, userID, []string{"openid", "profile", "email"})
	require.NoError(t, err)
	assert.True(t, active)
	assert.NotEmpty(t, refreshedFingerprint)
	assert.NotEqual(t, initialFingerprint, refreshedFingerprint)

	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "incident response",
		RequestID:   "identity-token-suspend",
	})
	require.NoError(t, err)
	active, err = service.IdentityAccessTokenActive(ctx, app.ClientID, userID, []string{"openid"})
	require.NoError(t, err)
	assert.False(t, active)
}

func TestIdentityClientCredentialsTokenActiveReflectsCurrentAppScopeAndStatus(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "client-credentials-owner")
	adminID := seedOpenPlatformUser(t, postgres, "client-credentials-admin")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
	})

	active, err := service.IdentityClientCredentialsTokenActive(ctx, app.ClientID, []string{ScopeResourceRead})
	require.NoError(t, err)
	assert.True(t, active)

	active, err = service.IdentityClientCredentialsTokenActive(ctx, app.ClientID, []string{ScopeResourceWrite})
	require.NoError(t, err)
	assert.False(t, active)

	active, err = service.IdentityClientCredentialsTokenActive(ctx, app.ClientID, []string{ScopeOfflineAccess})
	require.NoError(t, err)
	assert.False(t, active)

	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "server-to-server access disabled",
		RequestID:   "client-credentials-suspend",
	})
	require.NoError(t, err)
	active, err = service.IdentityClientCredentialsTokenActive(ctx, app.ClientID, []string{ScopeResourceRead})
	require.NoError(t, err)
	assert.False(t, active)
}

func TestOpenPlatformAppSecretLifecycleAndStatusAudit(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "secret-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "secret-other")
	adminID := seedOpenPlatformUser(t, postgres, "secret-admin")
	userID := seedOpenPlatformUser(t, postgres, "secret-consent-user")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "secret-lifecycle-grant",
	}, []string{ScopeProfileBasicRead, ScopeEmailRead}))

	_, err = service.VerifyClientSecret(ctx, app.ClientID, "test-secret")
	require.NoError(t, err)

	_, err = service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: otherOwnerID,
		OwnerUserID: otherOwnerID,
		ActorType:   "developer",
		Reason:      "wrong owner should not rotate",
		RequestID:   "rotate-wrong-owner",
	})
	require.ErrorIs(t, err, ErrAppNotFound)

	rotated, err := service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: ownerID,
		OwnerUserID: ownerID,
		ActorType:   "developer",
		Reason:      "scheduled rotation",
		RequestID:   "rotate-owner",
	})
	require.NoError(t, err)
	require.NotEmpty(t, rotated.ClientSecret)
	assert.NotEqual(t, "test-secret", rotated.ClientSecret)
	assert.Equal(t, AppStatusApproved, rotated.App.Status)
	assertOpenPlatformAuditCount(t, postgres, app.ID, ownerID, "open_platform.app.secret_rotated", 1)

	_, err = service.VerifyClientSecret(ctx, app.ClientID, "test-secret")
	require.ErrorIs(t, err, ErrAppNotFound)
	_, err = service.VerifyClientSecret(ctx, app.ClientID, rotated.ClientSecret)
	require.NoError(t, err)

	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "incident response",
		RequestID:   "suspend-app",
	})
	require.NoError(t, err)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.suspended", 1)
	consents, err := service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	require.Len(t, consents[0].Scopes, 2)
	_, err = service.VerifyClientSecret(ctx, app.ClientID, rotated.ClientSecret)
	require.ErrorIs(t, err, ErrAppNotActive)

	resumed, err := service.ResumeApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "risk cleared",
		RequestID:   "resume-app",
	})
	require.NoError(t, err)
	assert.Equal(t, AppStatusApproved, resumed.App.Status)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.resumed", 1)
	consents, err = service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	require.Len(t, consents, 1)
	require.Len(t, consents[0].Scopes, 2)
	_, err = service.VerifyClientSecret(ctx, app.ClientID, rotated.ClientSecret)
	require.NoError(t, err)

	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "incident response follow-up",
		RequestID:   "suspend-app-again",
	})
	require.NoError(t, err)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.suspended", 2)
	_, err = service.VerifyClientSecret(ctx, app.ClientID, rotated.ClientSecret)
	require.ErrorIs(t, err, ErrAppNotActive)

	_, err = service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: ownerID,
		OwnerUserID: ownerID,
		ActorType:   "developer",
		Reason:      "owner cannot rotate suspended app",
		RequestID:   "rotate-owner-suspended",
	})
	require.ErrorIs(t, err, ErrAppNotActive)

	adminRotated, err := service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		ActorType:   "admin",
		Reason:      "rotate while suspended",
		RequestID:   "rotate-admin-suspended",
	})
	require.NoError(t, err)
	require.NotEmpty(t, adminRotated.ClientSecret)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.secret_rotated", 1)
	_, err = repo.VerifyClientSecret(ctx, app.ClientID, adminRotated.ClientSecret)
	require.NoError(t, err)

	_, err = service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "permanent shutdown",
		RequestID:   "revoke-app",
	})
	require.NoError(t, err)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.revoked", 1)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.consent.revoked", 2)
	consents, err = service.ListUserConsents(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, consents)
	revokedConsentEvents, err := service.ListUserConsentAuditEvents(ctx, ListUserConsentAuditEventsInput{
		UserID:    userID,
		AppID:     app.ID,
		EventType: "open_platform.consent.revoked",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, 2, revokedConsentEvents.Total)
	require.Len(t, revokedConsentEvents.List, 2)
	revokedScopes := make([]string, 0, len(revokedConsentEvents.List))
	for _, event := range revokedConsentEvents.List {
		require.NotNil(t, event.RequestID)
		assert.Equal(t, "revoke-app", *event.RequestID)
		assert.Equal(t, "admin", event.Details["actor"])
		assert.Equal(t, "permanent shutdown", event.Details["reason"])
		assert.Equal(t, "app_lifecycle", event.Details["source"])
		require.NotNil(t, event.Scope)
		revokedScopes = append(revokedScopes, *event.Scope)
	}
	assert.ElementsMatch(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, revokedScopes)
	_, err = service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		ActorType:   "admin",
		Reason:      "revoked app cannot rotate",
		RequestID:   "rotate-admin-revoked",
	})
	require.ErrorIs(t, err, ErrAppNotActive)

	_, err = service.ResumeApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "revoked app cannot resume",
		RequestID:   "resume-revoked",
	})
	require.ErrorIs(t, err, ErrInvalidAppStatus)

	_, err = service.SuspendApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      " ",
		RequestID:   "suspend-empty-reason",
	})
	require.ErrorIs(t, err, ErrLifecycleReasonRequired)
}

func TestRotateAppSecretSyncsCasdoorApplication(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "secret-casdoor-owner")
	adminID := seedOpenPlatformUser(t, postgres, "secret-casdoor-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Secret Casdoor App",
		Description:      "Secret rotation should update Casdoor.",
		HomepageURL:      "https://secret-casdoor.example.com",
		PrivacyPolicyURL: "https://secret-casdoor.example.com/privacy",
		RedirectURIs:     []string{"https://secret-casdoor.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-before-secret-sync",
	})
	require.NoError(t, err)

	rotated, err := service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       registered.App.ID,
		ActorUserID: ownerID,
		OwnerUserID: ownerID,
		ActorType:   "developer",
		Reason:      "sync Casdoor secret",
		RequestID:   "rotate-secret-sync-casdoor",
	})

	require.NoError(t, err)
	require.NotEqual(t, approved.ClientSecret, rotated.ClientSecret)
	require.Len(t, provisioner.ensuredSpecs, 2)
	rotatedSpec := provisioner.ensuredSpecs[1]
	assert.Equal(t, registered.App.CasdoorApplicationName, rotatedSpec.Name)
	assert.Equal(t, registered.App.ClientID, rotatedSpec.ClientID)
	assert.Equal(t, rotated.ClientSecret, rotatedSpec.ClientSecret)
	assert.Equal(t, registered.App.RedirectURIs, rotatedSpec.RedirectURIs)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.app.secret_rotated", 1)
}

func TestRotateAppSecretRollsBackCasdoorWhenLocalUpdateFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "secret-rollback-owner")
	adminID := seedOpenPlatformUser(t, postgres, "secret-rollback-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Secret Rollback App",
		Description:      "Secret rotation rollback should survive cancellation.",
		HomepageURL:      "https://secret-rollback.example.com",
		PrivacyPolicyURL: "https://secret-rollback.example.com/privacy",
		RedirectURIs:     []string{"https://secret-rollback.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-before-secret-rollback",
	})
	require.NoError(t, err)

	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	provisioner.onEnsure = cancel
	rotated, err := service.RotateAppSecret(requestCtx, RotateAppSecretInput{
		AppID:       registered.App.ID,
		ActorUserID: ownerID,
		OwnerUserID: ownerID,
		ActorType:   "developer",
		Reason:      "local update fails after Casdoor secret rotation",
		RequestID:   "rotate-secret-local-fail",
	})

	require.Nil(t, rotated)
	require.Error(t, err)
	require.Len(t, provisioner.ensuredSpecs, 3)
	assert.NotEqual(t, approved.ClientSecret, provisioner.ensuredSpecs[1].ClientSecret)
	assert.Equal(t, approved.ClientSecret, provisioner.ensuredSpecs[2].ClientSecret)
	assert.Equal(t, approved.ClientSecret, provisioner.existing.ClientSecret)
	_, err = service.VerifyClientSecret(context.Background(), registered.App.ClientID, approved.ClientSecret)
	require.NoError(t, err)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.app.secret_rotated", 0)
}

func TestRotateAppSecretDoesNotUpdateLocalWhenCasdoorUpdateFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "secret-casdoor-fail-owner")
	adminID := seedOpenPlatformUser(t, postgres, "secret-casdoor-fail-admin")
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Secret Casdoor Failure App",
		Description:      "Local secret should not change when Casdoor update fails.",
		HomepageURL:      "https://secret-casdoor-fail.example.com",
		PrivacyPolicyURL: "https://secret-casdoor-fail.example.com/privacy",
		RedirectURIs:     []string{"https://secret-casdoor-fail.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	approved, err := service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-before-secret-casdoor-fail",
	})
	require.NoError(t, err)

	provisioner.ensureErr = errors.New("casdoor update unavailable")
	rotated, err := service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       registered.App.ID,
		ActorUserID: ownerID,
		OwnerUserID: ownerID,
		ActorType:   "developer",
		Reason:      "Casdoor update fails",
		RequestID:   "rotate-secret-casdoor-fail",
	})

	require.Nil(t, rotated)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "casdoor update unavailable")
	require.Len(t, provisioner.ensuredSpecs, 1)
	_, err = service.VerifyClientSecret(ctx, registered.App.ClientID, approved.ClientSecret)
	require.NoError(t, err)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.app.secret_rotated", 0)
}

func TestOpenPlatformRepositoryAppLifecycleUpdatesRequireCurrentStatus(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)

	ownerID := seedOpenPlatformUser(t, postgres, "repo-lifecycle-owner")
	adminID := seedOpenPlatformUser(t, postgres, "repo-lifecycle-admin")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})

	revoked, err := repo.UpdateAppStatusWithAudit(
		ctx,
		app.ID,
		AppStatusRevoked,
		[]string{AppStatusApproved},
		adminID,
		"open_platform.app.revoked",
		"retire integration",
		"repo-lifecycle-revoke",
	)
	require.NoError(t, err)
	assert.Equal(t, AppStatusRevoked, revoked.Status)

	err = repo.MarkAppApproved(
		ctx,
		app.ID,
		hashClientSecret("should-not-restore"),
		adminID,
		"repo-lifecycle-approve-revoked",
	)
	require.ErrorIs(t, err, ErrInvalidAppStatus)
	assertOpenPlatformAppStatus(t, ctx, repo, app.ID, AppStatusRevoked)

	_, err = repo.UpdateAppStatusWithAudit(
		ctx,
		app.ID,
		AppStatusApproved,
		[]string{AppStatusSuspended},
		adminID,
		"open_platform.app.resumed",
		"cannot resume revoked",
		"repo-lifecycle-resume-revoked",
	)
	require.ErrorIs(t, err, ErrInvalidAppStatus)
	assertOpenPlatformAppStatus(t, ctx, repo, app.ID, AppStatusRevoked)
}

func TestOpenPlatformDeveloperWithdrawPendingApp(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "withdraw-app-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "withdraw-app-other")
	adminID := seedOpenPlatformUser(t, postgres, "withdraw-app-admin")

	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Withdrawn Pending App",
		Description:      "Owner can withdraw before admin approval.",
		HomepageURL:      "https://withdraw-app.example.com",
		PrivacyPolicyURL: "https://withdraw-app.example.com/privacy",
		RedirectURIs:     []string{"https://withdraw-app.example.com/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, AppStatusPending, registered.App.Status)

	_, err = service.WithdrawApp(ctx, AppWithdrawalInput{
		AppID:       registered.App.ID,
		OwnerUserID: otherOwnerID,
		Reason:      "wrong owner",
		RequestID:   "withdraw-app-wrong-owner",
	})
	require.ErrorIs(t, err, ErrAppNotFound)

	_, err = service.WithdrawApp(ctx, AppWithdrawalInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Reason:      " ",
		RequestID:   "withdraw-app-empty-reason",
	})
	require.ErrorIs(t, err, ErrLifecycleReasonRequired)

	withdrawn, err := service.WithdrawApp(ctx, AppWithdrawalInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Reason:      "submitted wrong redirect URI",
		RequestID:   "withdraw-app",
	})
	require.NoError(t, err)
	require.NotNil(t, withdrawn.App)
	assert.Equal(t, AppStatusRevoked, withdrawn.App.Status)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, ownerID, "open_platform.app.withdrawn", 1)

	ownerApps, err := service.ListApps(ctx, ListAppsInput{
		OwnerUserID: ownerID,
		Status:      "all",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, ownerApps.Total)
	require.Len(t, ownerApps.List[0].Scopes, 1)
	assert.Equal(t, ScopeStatusWithdrawn, ownerApps.List[0].Scopes[0].Status)

	require.ErrorIs(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		RequestID:      "approve-scope-withdrawn-app",
	}), ErrInvalidAppStatus)

	_, err = service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-withdrawn-app",
	})
	require.ErrorIs(t, err, ErrInvalidAppStatus)

	_, err = service.WithdrawApp(ctx, AppWithdrawalInput{
		AppID:       registered.App.ID,
		OwnerUserID: ownerID,
		Reason:      "cannot withdraw twice",
		RequestID:   "withdraw-app-again",
	})
	require.ErrorIs(t, err, ErrInvalidAppStatus)
}

func TestOpenPlatformAuditEventListFilters(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "audit-owner")
	userID := seedOpenPlatformUser(t, postgres, "audit-viewer")
	adminID := seedOpenPlatformUser(t, postgres, "audit-admin")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})

	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "audit-grant",
	}, []string{ScopeEmailRead, ScopeProfileBasicRead}))

	_, err = service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		ActorType:   "admin",
		Reason:      "routine rotation",
		RequestID:   "audit-rotate",
	})
	require.NoError(t, err)

	require.NoError(t, service.RevokeUserConsent(ctx, RevokeConsentInput{
		UserID:    userID,
		AppID:     app.ID,
		Scopes:    []string{ScopeEmailRead},
		RequestID: "audit-revoke",
	}))

	allEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 4, allEvents.Total)
	require.Len(t, allEvents.List, 4)
	assert.Equal(t, "open_platform.consent.revoked", allEvents.List[0].EventType)
	for index := 1; index < len(allEvents.List); index++ {
		assert.Greater(t, allEvents.List[index-1].ID, allEvents.List[index].ID)
	}

	appEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		AppID:    app.ID,
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, appEvents.Total)

	userEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		UserID:   userID,
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, userEvents.Total)

	grantEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		EventType: "open_platform.consent.granted",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, grantEvents.Total)

	profileEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		Scope:    ScopeProfileBasicRead,
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, profileEvents.Total)
	require.Len(t, profileEvents.List, 1)
	assert.Equal(t, ScopeProfileBasicRead, *profileEvents.List[0].Scope)

	rotationEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		UserID:    adminID,
		EventType: "open_platform.app.secret_rotated",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, rotationEvents.Total)
	require.Len(t, rotationEvents.List, 1)
	assert.Equal(t, "audit-rotate", *rotationEvents.List[0].RequestID)
	assert.Equal(t, "routine rotation", rotationEvents.List[0].Metadata["reason"])
	assert.Equal(t, "admin", rotationEvents.List[0].Metadata["actorType"])

	emptyEvents, err := service.ListAuditEvents(ctx, ListAuditEventsInput{
		UserID:   999_999,
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, emptyEvents.Total)
	assert.Empty(t, emptyEvents.List)

	_, err = service.ListAuditEvents(ctx, ListAuditEventsInput{AppID: -1})
	require.ErrorIs(t, err, ErrInvalidAuditFilter)
	_, err = service.ListAuditEvents(ctx, ListAuditEventsInput{Scope: "invalid.scope"})
	require.ErrorIs(t, err, ErrInvalidScope)
}

func TestDeveloperAppAuditEventsAreOwnerScopedAndSanitized(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithConsentBaseURL("https://account.example.com"))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "developer-audit-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "developer-audit-other-owner")
	userID := seedOpenPlatformUser(t, postgres, "developer-audit-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
	})
	otherApp := &App{
		CasdoorApplicationName: "casdoor-open-platform-other-audit-app",
		OwnerUserID:            otherOwnerID,
		ClientID:               "op_other_audit_client",
		ClientSecretHash:       hashClientSecret("test-secret"),
		DisplayName:            "Other Audit App",
		Description:            "Used to prove developer app audit isolation.",
		HomepageURL:            "https://other-client.example.com",
		PrivacyPolicyURL:       "https://other-client.example.com/privacy",
		RedirectURIs:           []string{"https://other-client.example.com/callback"},
		Status:                 AppStatusApproved,
	}
	require.NoError(t, repo.ImportApprovedApp(ctx, otherApp, []ScopeRequest{
		{
			Scope:          ScopeEmailRead,
			Reason:         "other app email",
			Status:         ScopeStatusApproved,
			ReviewerUserID: &otherOwnerID,
		},
	}, otherOwnerID))

	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "developer-audit-grant",
	}, []string{ScopeEmailRead}))
	_, err = service.RotateAppSecret(ctx, RotateAppSecretInput{
		AppID:       app.ID,
		ActorUserID: ownerID,
		ActorType:   "developer",
		OwnerUserID: ownerID,
		Reason:      "scheduled rotation",
		RequestID:   "developer-audit-rotate",
	})
	require.NoError(t, err)
	_, err = service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:       app.ID,
		OwnerUserID: ownerID,
		RedirectURIs: []string{
			"https://client.example.com/callback",
			"https://client.example.com/next-callback",
		},
		Reason:    "add new callback host",
		RequestID: "developer-audit-redirect",
	})
	require.NoError(t, err)
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    userID,
		EventType: "open_platform.disclosure.denied",
		RequestID: "developer-audit-denied",
		Metadata: map[string]any{
			"endpoint": "userinfo",
			"result":   "scope_denied",
			"scopes":   []string{ScopeEmailRead},
			"error":    "internal stack trace should not be exposed",
		},
	}))
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    ownerID,
		EventType: "open_platform.app.token_probe.runtime.failed",
		RequestID: "developer-audit-probe",
		Metadata: map[string]any{
			"result":          "failed",
			"probeType":       "runtime_code_flow",
			"probeMethod":     "authorization_code",
			"businessClaims":  []string{"phone"},
			"tokenClaims":     map[string][]string{"id_token": {"phone"}},
			"error":           "raw token probe error should not be exposed",
			"inspectedClaims": []string{"sub", "phone"},
		},
	}))
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    ownerID,
		EventType: "open_platform.app.approved_app_ensured",
		RequestID: "developer-audit-approved-ensured",
		Metadata: map[string]any{
			"clientID":    app.ClientID,
			"displayName": app.DisplayName,
		},
	}))
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     otherApp.ID,
		UserID:    otherOwnerID,
		EventType: "open_platform.app.secret_rotated",
		RequestID: "developer-audit-other-app",
		Metadata:  map[string]any{"reason": "other app event"},
	}))

	events, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       app.ID,
		PageSize:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, 6, events.Total)
	require.Len(t, events.List, 6)
	for _, event := range events.List {
		if event.RequestID != nil {
			assert.NotEqual(t, "developer-audit-other-app", *event.RequestID)
		}
	}

	denied := developerAppAuditEventByRequestID(t, events.List, "developer-audit-denied")
	assert.Equal(t, "open_platform.disclosure.denied", denied.EventType)
	assert.Equal(t, []string{ScopeEmailRead}, denied.Scopes)
	require.NotNil(t, denied.Endpoint)
	assert.Equal(t, "userinfo", *denied.Endpoint)
	require.NotNil(t, denied.Result)
	assert.Equal(t, "scope_denied", *denied.Result)
	assert.Equal(t, "userinfo", denied.Details["endpoint"])
	assert.Equal(t, "scope_denied", denied.Details["result"])
	assert.NotContains(t, denied.Details, "error")

	probe := developerAppAuditEventByRequestID(t, events.List, "developer-audit-probe")
	assert.Equal(t, "open_platform.app.token_probe.runtime.failed", probe.EventType)
	require.NotNil(t, probe.Result)
	assert.Equal(t, "failed", *probe.Result)
	assert.Contains(t, probe.Details, "businessClaims")
	assert.NotContains(t, probe.Details, "tokenClaims")
	assert.NotContains(t, probe.Details, "error")

	ensured, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       app.ID,
		EventType:   "open_platform.app.approved_app_ensured",
		PageSize:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, ensured.Total)
	require.Len(t, ensured.List, 1)
	require.NotNil(t, ensured.List[0].RequestID)
	assert.Equal(t, "developer-audit-approved-ensured", *ensured.List[0].RequestID)

	scoped, err := service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       app.ID,
		Scope:       ScopeEmailRead,
		PageSize:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, scoped.Total)

	_, err = service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: otherOwnerID,
		AppID:       app.ID,
	})
	require.ErrorIs(t, err, ErrAppNotFound)
	_, err = service.ListDeveloperAppAuditEvents(ctx, ListDeveloperAppAuditEventsInput{
		OwnerUserID: ownerID,
		AppID:       app.ID,
		EventType:   "open_platform.unlisted",
	})
	require.ErrorIs(t, err, ErrInvalidAuditFilter)
}

func TestOpenPlatformDisclosureAuditAndRateLimits(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(
		repo,
		redis.Client,
		WithConsentBaseURL("https://account.example.com"),
		WithDisclosureRateLimits(DisclosureRateLimitConfig{
			AppLimit:      1,
			AppUserLimit:  100,
			EndpointLimit: 100,
			ConsentLimit:  100,
			Window:        time.Minute,
		}),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "rl-owner")
	userID := seedOpenPlatformUser(t, postgres, "rl-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "rl-grant",
	}, []string{ScopeProfileBasicRead}))

	okBefore := promtestutil.ToFloat64(
		metrics.OpenPlatformDisclosureRequestsTotal.WithLabelValues(disclosureEndpointUserInfo, "ok"),
	)
	limitedBefore := promtestutil.ToFloat64(
		metrics.OpenPlatformDisclosureRequestsTotal.WithLabelValues(disclosureEndpointUserInfo, "rate_limited"),
	)

	payload, err := service.UserInfo(ctx, DisclosureRequest{
		ClientID:    app.ClientID,
		UserID:      userID,
		Scopes:      []string{ScopeProfileBasicRead},
		RedirectURI: app.RedirectURIs[0],
		RequestID:   "rl-ok",
	})
	require.NoError(t, err)
	assert.Equal(t, "rl-viewer", payload["username"])

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:    app.ClientID,
		UserID:      userID,
		Scopes:      []string{ScopeProfileBasicRead},
		RedirectURI: app.RedirectURIs[0],
		RequestID:   "rl-limited",
	})
	require.ErrorIs(t, err, ErrDisclosureRateLimited)

	assert.Equal(t, okBefore+1, promtestutil.ToFloat64(
		metrics.OpenPlatformDisclosureRequestsTotal.WithLabelValues(disclosureEndpointUserInfo, "ok"),
	))
	assert.Equal(t, limitedBefore+1, promtestutil.ToFloat64(
		metrics.OpenPlatformDisclosureRequestsTotal.WithLabelValues(disclosureEndpointUserInfo, "rate_limited"),
	))
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.granted", 1)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.denied", 1)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, userID, "open_platform.disclosure.denied", map[string]any{
		"rateLimitDimension": "app",
		"result":             "rate_limited",
	})
}

func TestOpenPlatformDisclosureReplayDetection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(
		repo,
		redis.Client,
		WithDisclosureRateLimits(DisclosureRateLimitConfig{
			AppLimit:      100,
			AppUserLimit:  100,
			EndpointLimit: 100,
			ConsentLimit:  100,
			ReplayLimit:   2,
		}),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "replay-owner")
	userID := seedOpenPlatformUser(t, postgres, "replay-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})
	beforeDetected := promtestutil.ToFloat64(
		metrics.OpenPlatformDisclosureReplayTotal.WithLabelValues(disclosureEndpointUserInfo, "detected"),
	)

	for range 2 {
		_, err = service.UserInfo(ctx, DisclosureRequest{
			ClientID:       app.ClientID,
			UserID:         userID,
			Scopes:         []string{ScopeEmailRead},
			RedirectURI:    app.RedirectURIs[0],
			ConsentBaseURL: "https://account.example.com",
			RequestID:      "replay-request",
		})
		var consentErr ConsentRequiredError
		require.ErrorAs(t, err, &consentErr)
	}

	assert.Equal(t, beforeDetected+1, promtestutil.ToFloat64(
		metrics.OpenPlatformDisclosureReplayTotal.WithLabelValues(disclosureEndpointUserInfo, "detected"),
	))
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.replay_detected", 1)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, userID, "open_platform.disclosure.replay_detected", map[string]any{
		"endpoint": disclosureEndpointUserInfo,
		"result":   "consent_required",
	})
}

func TestOpenPlatformDisclosureReportAggregatesAuditEvents(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "report-owner")
	userID := seedOpenPlatformUser(t, postgres, "report-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})

	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    userID,
		EventType: "open_platform.disclosure.granted",
		RequestID: "report-granted",
		Metadata: map[string]any{
			"endpoint": disclosureEndpointUserInfo,
			"result":   "ok",
			"scopes":   []string{ScopeProfileBasicRead},
		},
	}))
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    userID,
		EventType: "open_platform.disclosure.denied",
		RequestID: "report-denied",
		Metadata: map[string]any{
			"endpoint": disclosureEndpointUserInfo,
			"result":   "scope_denied",
			"scopes":   []string{ScopeProfileBasicRead},
		},
	}))
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    userID,
		EventType: "open_platform.disclosure.denied",
		RequestID: "report-limited",
		Metadata: map[string]any{
			"endpoint":           disclosureEndpointPhone,
			"result":             "rate_limited",
			"rateLimitDimension": "app_user",
			"scopes":             []string{ScopePhoneRead},
		},
	}))
	require.NoError(t, repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     app.ID,
		UserID:    userID,
		EventType: "open_platform.disclosure.replay_detected",
		RequestID: "report-replay",
		Metadata: map[string]any{
			"endpoint":      disclosureEndpointUserInfo,
			"result":        "consent_required",
			"count":         8,
			"signatureHash": "sig",
			"scopes":        []string{ScopeEmailRead},
		},
	}))

	report, err := service.DisclosureReport(ctx, DisclosureReportInput{WindowHours: 24})
	require.NoError(t, err)
	assert.Equal(t, DisclosureReportSummary{
		WindowHours:    24,
		Total:          4,
		Granted:        1,
		Denied:         2,
		RateLimited:    1,
		ReplayDetected: 1,
	}, report.Summary)
	require.NotEmpty(t, report.Endpoints)
	assert.Equal(t, "userinfo", report.Endpoints[0].Endpoint)
	require.Len(t, report.RateLimitDimensions, 1)
	assert.Equal(t, "app_user", report.RateLimitDimensions[0].Dimension)
	assert.Equal(t, 1, report.RateLimitDimensions[0].Total)
	require.Len(t, report.RecentReplayEvents, 1)
	assert.Equal(t, "consent_required", report.RecentReplayEvents[0].Result)
	assert.Equal(t, []string{ScopeEmailRead}, report.RecentReplayEvents[0].Scopes)
}

func TestOpenPlatformConsentChallengeRateLimit(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(
		repo,
		redis.Client,
		WithConsentBaseURL("https://account.example.com"),
		WithDisclosureRateLimits(DisclosureRateLimitConfig{
			AppLimit:      100,
			AppUserLimit:  100,
			EndpointLimit: 100,
			ConsentLimit:  1,
			Window:        time.Minute,
		}),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "consent-rl-owner")
	userID := seedOpenPlatformUser(t, postgres, "consent-rl-viewer")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeEmailRead})

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:       app.ClientID,
		UserID:         userID,
		Scopes:         []string{ScopeEmailRead},
		RedirectURI:    app.RedirectURIs[0],
		ConsentBaseURL: "https://account.example.com",
		RequestID:      "consent-required",
	})
	var consentErr ConsentRequiredError
	require.ErrorAs(t, err, &consentErr)

	_, err = service.UserInfo(ctx, DisclosureRequest{
		ClientID:       app.ClientID,
		UserID:         userID,
		Scopes:         []string{ScopeEmailRead},
		RedirectURI:    app.RedirectURIs[0],
		ConsentBaseURL: "https://account.example.com",
		RequestID:      "consent-limited",
	})
	require.ErrorIs(t, err, ErrDisclosureRateLimited)

	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.denied", 2)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, userID, "open_platform.disclosure.denied", map[string]any{
		"rateLimitDimension": "consent",
		"result":             "rate_limited",
	})
}

func TestOpenPlatformRedirectURIChangeReview(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "redirect-owner")
	otherOwnerID := seedOpenPlatformUser(t, postgres, "redirect-other")
	adminID := seedOpenPlatformUser(t, postgres, "redirect-admin")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeProfileBasicRead})

	_, err = service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        app.ID,
		OwnerUserID:  otherOwnerID,
		RedirectURIs: []string{"https://new-client.example.com/callback"},
		Reason:       "wrong owner should not request",
		RequestID:    "redirect-wrong-owner",
	})
	require.ErrorIs(t, err, ErrAppNotFound)

	_, err = service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        app.ID,
		OwnerUserID:  ownerID,
		RedirectURIs: []string{"https://new-client.example.com/callback"},
		Reason:       " ",
		RequestID:    "redirect-empty-reason",
	})
	require.ErrorIs(t, err, ErrRedirectURIReasonRequired)

	_, err = service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        app.ID,
		OwnerUserID:  ownerID,
		RedirectURIs: []string{"https://new-client.example.com/callback#fragment"},
		Reason:       "move callback",
		RequestID:    "redirect-invalid-uri",
	})
	require.ErrorIs(t, err, ErrRedirectURINotAllowed)

	requested, err := service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        app.ID,
		OwnerUserID:  ownerID,
		RedirectURIs: []string{"https://new-client.example.com/callback"},
		Reason:       "move callback host",
		RequestID:    "redirect-request",
	})
	require.NoError(t, err)
	assert.Equal(t, ScopeStatusPending, requested.Status)
	assert.Equal(t, []string{"https://new-client.example.com/callback"}, requested.RedirectURIs)
	assertOpenPlatformAuditCount(t, postgres, app.ID, ownerID, "open_platform.app.redirect_uris.requested", 1)

	_, err = service.WithdrawRedirectURIRequest(ctx, RedirectURIWithdrawalInput{
		AppID:                app.ID,
		RedirectURIRequestID: requested.ID,
		OwnerUserID:          otherOwnerID,
		Reason:               "wrong owner should not withdraw",
		RequestID:            "redirect-withdraw-wrong-owner",
	})
	require.ErrorIs(t, err, ErrAppNotFound)

	_, err = service.WithdrawRedirectURIRequest(ctx, RedirectURIWithdrawalInput{
		AppID:                app.ID,
		RedirectURIRequestID: requested.ID,
		OwnerUserID:          ownerID,
		Reason:               " ",
		RequestID:            "redirect-withdraw-empty-reason",
	})
	require.ErrorIs(t, err, ErrLifecycleReasonRequired)

	withdrawn, err := service.WithdrawRedirectURIRequest(ctx, RedirectURIWithdrawalInput{
		AppID:                app.ID,
		RedirectURIRequestID: requested.ID,
		OwnerUserID:          ownerID,
		Reason:               "domain is not ready",
		RequestID:            "redirect-withdraw",
	})
	require.NoError(t, err)
	assert.Equal(t, ScopeStatusWithdrawn, withdrawn.Status)
	require.NotNil(t, withdrawn.DecisionNote)
	assert.Equal(t, "domain is not ready", *withdrawn.DecisionNote)
	assertOpenPlatformAuditCount(t, postgres, app.ID, ownerID, "open_platform.app.redirect_uris.withdrawn", 1)

	_, err = service.ApproveRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                app.ID,
		RedirectURIRequestID: requested.ID,
		ReviewerUserID:       adminID,
		RequestID:            "redirect-approve-withdrawn",
	})
	require.ErrorIs(t, err, ErrRedirectURIRequestNotFound)

	requested, err = service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        app.ID,
		OwnerUserID:  ownerID,
		RedirectURIs: []string{"https://new-client.example.com/callback"},
		Reason:       "move callback host after domain is ready",
		RequestID:    "redirect-request-after-withdraw",
	})
	require.NoError(t, err)
	assert.Equal(t, ScopeStatusPending, requested.Status)
	assertOpenPlatformAuditCount(t, postgres, app.ID, ownerID, "open_platform.app.redirect_uris.requested", 2)

	ownerApps, err := service.ListApps(ctx, ListAppsInput{
		OwnerUserID: ownerID,
		Status:      "all",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, ownerApps.Total)
	require.Len(t, ownerApps.List[0].RedirectURIRequests, 2)
	assert.Equal(t, requested.ID, ownerApps.List[0].RedirectURIRequests[0].ID)
	assert.Equal(t, ScopeStatusWithdrawn, ownerApps.List[0].RedirectURIRequests[1].Status)

	reviewed, err := service.ApproveRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                app.ID,
		RedirectURIRequestID: requested.ID,
		ReviewerUserID:       adminID,
		DecisionNote:         "domain verified",
		RequestID:            "redirect-approve",
	})
	require.NoError(t, err)
	assert.Equal(t, ScopeStatusApproved, reviewed.Status)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.redirect_uris.approved", 1)

	updatedApp, err := repo.GetAppByID(ctx, app.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://new-client.example.com/callback"}, updatedApp.RedirectURIs)

	assert.False(t, RedirectURIAllowed(updatedApp, "https://client.example.com/callback"))
	assert.True(t, RedirectURIAllowed(updatedApp, "https://new-client.example.com/callback"))

	second, err := service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        app.ID,
		OwnerUserID:  ownerID,
		RedirectURIs: []string{"https://rejected.example.com/callback"},
		Reason:       "temporary host",
		RequestID:    "redirect-request-reject",
	})
	require.NoError(t, err)
	rejected, err := service.RejectRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                app.ID,
		RedirectURIRequestID: second.ID,
		ReviewerUserID:       adminID,
		DecisionNote:         "domain ownership not verified",
		RequestID:            "redirect-reject",
	})
	require.NoError(t, err)
	assert.Equal(t, ScopeStatusRejected, rejected.Status)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.redirect_uris.rejected", 1)

	updatedApp, err = repo.GetAppByID(ctx, app.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://new-client.example.com/callback"}, updatedApp.RedirectURIs)

	_, err = service.ApproveRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                app.ID,
		RedirectURIRequestID: second.ID,
		ReviewerUserID:       adminID,
		RequestID:            "redirect-approve-rejected",
	})
	require.ErrorIs(t, err, ErrRedirectURIRequestNotFound)
}

func TestApproveRedirectURIRequestSyncsCasdoorApplication(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "casdoor-sync")
	registered := fixture.registered
	newRedirectURIs := []string{"https://new-redirect-casdoor.example.com/callback"}
	requested, err := service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        registered.App.ID,
		OwnerUserID:  fixture.ownerID,
		RedirectURIs: newRedirectURIs,
		Reason:       "move callback host",
		RequestID:    "redirect-sync-request",
	})
	require.NoError(t, err)

	reviewed, err := service.ApproveRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                registered.App.ID,
		RedirectURIRequestID: requested.ID,
		ReviewerUserID:       fixture.adminID,
		DecisionNote:         "domain verified",
		RequestID:            "redirect-sync-approve",
	})

	require.NoError(t, err)
	assert.Equal(t, ScopeStatusApproved, reviewed.Status)
	require.Len(t, provisioner.ensuredSpecs, 2)
	redirectSpec := provisioner.ensuredSpecs[1]
	assert.Equal(t, registered.App.CasdoorApplicationName, redirectSpec.Name)
	assert.Equal(t, newRedirectURIs, redirectSpec.RedirectURIs)
	assert.NotEmpty(t, redirectSpec.ClientSecret)
	app, err := repo.GetAppByID(ctx, registered.App.ID)
	require.NoError(t, err)
	assert.Equal(t, newRedirectURIs, app.RedirectURIs)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.adminID, "open_platform.app.redirect_uris.approved", 1)
}

func TestApproveRedirectURIRequestDoesNotUpdateLocalWhenCasdoorSyncFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "casdoor-fail")
	registered := fixture.registered
	requested, err := service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        registered.App.ID,
		OwnerUserID:  fixture.ownerID,
		RedirectURIs: []string{"https://new-redirect-casdoor-fail.example.com/callback"},
		Reason:       "move callback host",
		RequestID:    "redirect-casdoor-fail-request",
	})
	require.NoError(t, err)

	provisioner.ensureErr = errors.New("casdoor redirect update unavailable")
	reviewed, err := service.ApproveRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                registered.App.ID,
		RedirectURIRequestID: requested.ID,
		ReviewerUserID:       fixture.adminID,
		DecisionNote:         "domain verified",
		RequestID:            "redirect-casdoor-fail-approve",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "casdoor redirect update unavailable")
	assert.Equal(t, RedirectURIRequest{}, reviewed)
	app, err := repo.GetAppByID(ctx, registered.App.ID)
	require.NoError(t, err)
	assert.Equal(t, fixture.initialRedirectURIs, app.RedirectURIs)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.adminID, "open_platform.app.redirect_uris.approved", 0)
}

func TestApproveRedirectURIRequestRollsBackCasdoorWhenLocalUpdateFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	provisioner := &fakeOpenPlatformAppProvisioner{}
	service, err := NewService(repo, redis.Client, WithAppProvisioner(provisioner))
	require.NoError(t, err)

	fixture := setupApprovedCasdoorOpenPlatformApp(t, ctx, postgres, service, "rollback")
	registered := fixture.registered
	newRedirectURIs := []string{"https://new-redirect-rollback.example.com/callback"}
	requested, err := service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        registered.App.ID,
		OwnerUserID:  fixture.ownerID,
		RedirectURIs: newRedirectURIs,
		Reason:       "move callback host",
		RequestID:    "redirect-rollback-request",
	})
	require.NoError(t, err)

	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	provisioner.onEnsure = cancel
	reviewed, err := service.ApproveRedirectURIRequest(requestCtx, RedirectURIReviewInput{
		AppID:                registered.App.ID,
		RedirectURIRequestID: requested.ID,
		ReviewerUserID:       fixture.adminID,
		DecisionNote:         "domain verified",
		RequestID:            "redirect-rollback-approve",
	})

	require.Error(t, err)
	assert.Equal(t, RedirectURIRequest{}, reviewed)
	require.Len(t, provisioner.ensuredSpecs, 3)
	assert.Equal(t, newRedirectURIs, provisioner.ensuredSpecs[1].RedirectURIs)
	assert.Equal(t, fixture.initialRedirectURIs, provisioner.ensuredSpecs[2].RedirectURIs)
	assert.Equal(t, fixture.initialRedirectURIs, provisioner.existing.RedirectURIs)
	app, err := repo.GetAppByID(context.Background(), registered.App.ID)
	require.NoError(t, err)
	assert.Equal(t, fixture.initialRedirectURIs, app.RedirectURIs)
	assertOpenPlatformAuditCount(t, postgres, registered.App.ID, fixture.adminID, "open_platform.app.redirect_uris.approved", 0)
}

type casdoorOpenPlatformAppFixture struct {
	registered          *RegisteredApp
	ownerID             int64
	adminID             int64
	initialRedirectURIs []string
}

func setupApprovedCasdoorOpenPlatformApp(
	t *testing.T,
	ctx context.Context,
	postgres *postgresfixture.Fixture,
	service *Service,
	suffix string,
) casdoorOpenPlatformAppFixture {
	t.Helper()
	ownerID := seedOpenPlatformUser(t, postgres, "redirect-"+suffix+"-owner")
	adminID := seedOpenPlatformUser(t, postgres, "redirect-"+suffix+"-admin")
	host := "redirect-" + suffix + ".example.com"
	registered, err := service.RegisterApp(ctx, RegisterAppInput{
		OwnerUserID:      ownerID,
		DisplayName:      "Redirect " + suffix + " App",
		Description:      "Used by redirect URI Casdoor synchronization tests.",
		HomepageURL:      "https://" + host,
		PrivacyPolicyURL: "https://" + host + "/privacy",
		RedirectURIs:     []string{"https://" + host + "/callback"},
		Scopes: []ScopeRequestInput{
			{Scope: ScopeProfileBasicRead, Reason: "show profile"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, service.ApproveScope(ctx, ApproveScopeInput{
		AppID:          registered.App.ID,
		Scope:          ScopeProfileBasicRead,
		ReviewerUserID: adminID,
		DecisionNote:   "profile is required",
	}))
	_, err = service.ApproveAppWithAudit(ctx, ApproveAppInput{
		AppID:          registered.App.ID,
		ReviewerUserID: adminID,
		RequestID:      "approve-before-redirect-" + suffix,
	})
	require.NoError(t, err)
	return casdoorOpenPlatformAppFixture{
		registered:          registered,
		ownerID:             ownerID,
		adminID:             adminID,
		initialRedirectURIs: append([]string(nil), registered.App.RedirectURIs...),
	}
}

func seedOpenPlatformUser(t *testing.T, fixture *postgresfixture.Fixture, suffix string) int64 {
	t.Helper()
	var id int64
	err := fixture.DB.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "open-platform-"+suffix, suffix, suffix+"@example.com").Scan(&id)
	require.NoError(t, err)
	return id
}

func queryValueFromURL(t *testing.T, rawURL string, key string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	value := parsed.Query().Get(key)
	require.NotEmpty(t, value)
	return value
}

type recordingOIDCAuthURLBuilder struct {
	url         string
	clientID    string
	redirectURI string
	scopes      []string
	state       string
}

func (b *recordingOIDCAuthURLBuilder) GetAuthURL(clientID string, redirectURI string, scopes []string, state string) string {
	b.clientID = clientID
	b.redirectURI = redirectURI
	b.scopes = append([]string(nil), scopes...)
	b.state = state
	return b.url
}

func assertOpenPlatformAppStatus(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	appID int64,
	expected string,
) {
	t.Helper()
	app, err := repo.GetAppByID(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, expected, app.Status)
}

func seedApprovedOpenPlatformApp(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	ownerID int64,
	scopes []string,
) *App {
	t.Helper()
	inputs := make([]ScopeRequestInput, 0, len(scopes))
	for _, scope := range scopes {
		inputs = append(inputs, ScopeRequestInput{
			Scope:  scope,
			Reason: "integration test",
		})
	}
	return seedApprovedOpenPlatformAppWithScopeInputs(t, ctx, repo, ownerID, inputs)
}

func seedApprovedOpenPlatformAppWithName(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	ownerID int64,
	displayName string,
	scopes []string,
) *App {
	t.Helper()
	inputs := make([]ScopeRequestInput, 0, len(scopes))
	for _, scope := range scopes {
		inputs = append(inputs, ScopeRequestInput{
			Scope:  scope,
			Reason: "integration test",
		})
	}
	suffix := strings.NewReplacer(" ", "-", "_", "-").Replace(strings.ToLower(displayName))
	return seedApprovedOpenPlatformAppWithAppFields(t, ctx, repo, ownerID, App{
		CasdoorApplicationName: "casdoor-open-platform-test-app-" + suffix,
		ClientID:               "op_test_client_" + suffix,
		DisplayName:            displayName,
		HomepageURL:            "https://" + suffix + ".example.com",
		PrivacyPolicyURL:       "https://" + suffix + ".example.com/privacy",
		RedirectURIs:           []string{"https://" + suffix + ".example.com/callback"},
	}, inputs)
}

func seedApprovedOpenPlatformAppWithScopeInputs(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	ownerID int64,
	scopes []ScopeRequestInput,
) *App {
	t.Helper()
	return seedApprovedOpenPlatformAppWithAppFields(t, ctx, repo, ownerID, App{
		CasdoorApplicationName: "casdoor-open-platform-test-app",
		ClientID:               "op_test_client",
		DisplayName:            "Open Platform Test App",
		HomepageURL:            "https://client.example.com",
		PrivacyPolicyURL:       "https://client.example.com/privacy",
		RedirectURIs:           []string{"https://client.example.com/callback"},
	}, scopes)
}

func seedApprovedOpenPlatformAppWithAppFields(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	ownerID int64,
	appFields App,
	scopes []ScopeRequestInput,
) *App {
	t.Helper()
	app := &App{
		CasdoorApplicationName: appFields.CasdoorApplicationName,
		OwnerUserID:            ownerID,
		ClientID:               appFields.ClientID,
		ClientSecretHash:       hashClientSecret("test-secret"),
		DisplayName:            appFields.DisplayName,
		Description:            "Used by open platform consent management tests.",
		HomepageURL:            appFields.HomepageURL,
		PrivacyPolicyURL:       appFields.PrivacyPolicyURL,
		RedirectURIs:           append([]string(nil), appFields.RedirectURIs...),
		Status:                 AppStatusApproved,
	}
	requests := make([]ScopeRequest, 0, len(scopes))
	for _, scope := range scopes {
		requests = append(requests, ScopeRequest{
			Scope:          scope.Scope,
			Reason:         scope.Reason,
			Status:         ScopeStatusApproved,
			ReviewerUserID: &ownerID,
		})
	}
	require.NoError(t, repo.ImportApprovedApp(ctx, app, requests, ownerID))
	return app
}

func approveOpenPlatformRedirectURIs(
	t *testing.T,
	ctx context.Context,
	service *Service,
	appID int64,
	ownerID int64,
	reviewerID int64,
	redirectURIs []string,
	requestID string,
) {
	t.Helper()
	requested, err := service.RequestRedirectURIChange(ctx, RedirectURIChangeInput{
		AppID:        appID,
		OwnerUserID:  ownerID,
		RedirectURIs: redirectURIs,
		Reason:       "integration test redirect update",
		RequestID:    requestID + "-request",
	})
	require.NoError(t, err)
	_, err = service.ApproveRedirectURIRequest(ctx, RedirectURIReviewInput{
		AppID:                appID,
		RedirectURIRequestID: requested.ID,
		ReviewerUserID:       reviewerID,
		DecisionNote:         "integration test redirect update",
		RequestID:            requestID + "-approve",
	})
	require.NoError(t, err)
}

func assertOpenPlatformAuditCount(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	appID int64,
	userID int64,
	eventType string,
	want int64,
) {
	t.Helper()
	var count int64
	query := `
		SELECT COUNT(*)
		FROM open_platform_audit_events
		WHERE user_id = $1
		  AND event_type = $2
	`
	args := []any{userID, eventType}
	if appID > 0 {
		query += ` AND app_id = $3`
		args = append(args, appID)
	} else {
		query += ` AND app_id IS NULL`
	}
	err := fixture.DB.QueryRow(context.Background(), query, args...).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, want, count)
}

func assertOpenPlatformAuditMetadata(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	appID int64,
	userID int64,
	eventType string,
	want map[string]any,
) {
	t.Helper()
	var raw []byte
	query := `
		SELECT metadata
		FROM open_platform_audit_events
		WHERE user_id = $1
		  AND event_type = $2
	`
	args := []any{userID, eventType}
	if appID > 0 {
		query += ` AND app_id = $3`
		args = append(args, appID)
	} else {
		query += ` AND app_id IS NULL`
	}
	query += `
		ORDER BY id DESC
		LIMIT 1
	`
	err := fixture.DB.QueryRow(context.Background(), query, args...).Scan(&raw)
	require.NoError(t, err)
	metadata := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &metadata))
	for key, value := range want {
		assert.Equal(t, value, metadata[key], key)
	}
}

type tokenProbeEvidenceForTest struct {
	Result          string
	InspectedClaims []string
	BusinessClaims  []string
	TokenClaims     map[string][]string
	Metadata        map[string]any
	Error           string
}

func latestOpenPlatformTokenProbeEvidence(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	appID int64,
) tokenProbeEvidenceForTest {
	t.Helper()
	var evidence tokenProbeEvidenceForTest
	var inspectedRaw []byte
	var businessRaw []byte
	var tokenRaw []byte
	var metadataRaw []byte
	err := fixture.DB.QueryRow(context.Background(), `
		SELECT result, inspected_claims, business_claims, token_claims, metadata, error
		FROM open_platform_token_probe_evidence
		WHERE app_id = $1
		ORDER BY id DESC
		LIMIT 1
	`, appID).Scan(
		&evidence.Result,
		&inspectedRaw,
		&businessRaw,
		&tokenRaw,
		&metadataRaw,
		&evidence.Error,
	)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(inspectedRaw, &evidence.InspectedClaims))
	require.NoError(t, json.Unmarshal(businessRaw, &evidence.BusinessClaims))
	require.NoError(t, json.Unmarshal(tokenRaw, &evidence.TokenClaims))
	require.NoError(t, json.Unmarshal(metadataRaw, &evidence.Metadata))
	return evidence
}

type fakeOpenPlatformAppProvisioner struct {
	existing     ProvisionedApplicationSpec
	ensured      *ProvisionedApplicationSpec
	ensuredSpecs []ProvisionedApplicationSpec
	deleted      []string
	err          error
	ensureErr    error
	deleteErr    error
	onEnsure     func()
}

func (f *fakeOpenPlatformAppProvisioner) GetApplication(_ context.Context, name string) (ProvisionedApplicationSpec, error) {
	if f.err != nil {
		return ProvisionedApplicationSpec{}, f.err
	}
	spec := f.existing
	if spec.Name == "" {
		spec.Name = name
	}
	return spec, nil
}

func (f *fakeOpenPlatformAppProvisioner) EnsureApplication(ctx context.Context, spec ProvisionedApplicationSpec) error {
	if f.ensureErr != nil {
		return f.ensureErr
	}
	if f.err != nil {
		return f.err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	specCopy := spec
	f.ensured = &specCopy
	f.ensuredSpecs = append(f.ensuredSpecs, specCopy)
	f.existing = specCopy
	if f.onEnsure != nil {
		f.onEnsure()
	}
	return nil
}

func (f *fakeOpenPlatformAppProvisioner) DeleteApplication(ctx context.Context, name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if f.err != nil {
		return f.err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, strings.TrimSpace(name))
	return nil
}

type cancelAfterRedisSetHook struct {
	keyPrefix string
	cancel    context.CancelFunc
}

func (h cancelAfterRedisSetHook) DialHook(next redisclient.DialHook) redisclient.DialHook {
	return next
}

func (h cancelAfterRedisSetHook) ProcessHook(next redisclient.ProcessHook) redisclient.ProcessHook {
	return func(ctx context.Context, cmd redisclient.Cmder) error {
		err := next(ctx, cmd)
		if err != nil || h.cancel == nil || strings.ToLower(cmd.Name()) != "set" {
			return err
		}
		args := cmd.Args()
		if len(args) < 2 {
			return err
		}
		key, ok := args[1].(string)
		if ok && strings.HasPrefix(key, h.keyPrefix) {
			h.cancel()
		}
		return err
	}
}

func (h cancelAfterRedisSetHook) ProcessPipelineHook(next redisclient.ProcessPipelineHook) redisclient.ProcessPipelineHook {
	return next
}

type cancelBeforeRedisDelHook struct {
	keyPrefix string
	cancel    context.CancelFunc
}

func (h cancelBeforeRedisDelHook) DialHook(next redisclient.DialHook) redisclient.DialHook {
	return next
}

func (h cancelBeforeRedisDelHook) ProcessHook(next redisclient.ProcessHook) redisclient.ProcessHook {
	return func(ctx context.Context, cmd redisclient.Cmder) error {
		if h.cancel != nil && strings.ToLower(cmd.Name()) == "del" {
			for _, arg := range cmd.Args()[1:] {
				key, ok := arg.(string)
				if ok && strings.HasPrefix(key, h.keyPrefix) {
					h.cancel()
					break
				}
			}
		}
		return next(ctx, cmd)
	}
}

func (h cancelBeforeRedisDelHook) ProcessPipelineHook(next redisclient.ProcessPipelineHook) redisclient.ProcessPipelineHook {
	return next
}

type fakeRuntimeTokenProber struct {
	result RuntimeTokenMinimizationProbeResult
	err    error
	spec   *ProvisionedApplicationSpec
}

func (f *fakeRuntimeTokenProber) ProbeTokenMinimization(
	_ context.Context,
	spec ProvisionedApplicationSpec,
) (RuntimeTokenMinimizationProbeResult, error) {
	specCopy := spec
	f.spec = &specCopy
	if f.err != nil {
		return f.result, f.err
	}
	return normalizeRuntimeTokenMinimizationProbeResult(f.result)
}
