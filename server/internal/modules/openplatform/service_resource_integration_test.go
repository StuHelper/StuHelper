package openplatform

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestResourceAccessGrantCheckListAndRevoke(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "resource-admin")
	ownerID := seedOpenPlatformUser(t, postgres, "resource-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
		ScopeResourceWrite,
	})

	granted, err := service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "42",
		Actions:        []string{ResourceAccessActionRead, ResourceAccessActionWrite, ResourceAccessActionRead},
		Reason:         "integration-test grant",
		RequestID:      "grant-resource-access",
	})
	require.NoError(t, err)
	require.Len(t, granted.Grants, 2)
	assert.Equal(t, []fga.Tuple{
		{User: openPlatformAppFGAUser(app.ID), Relation: ResourceRelationReadByApp, Object: "resource_item:42"},
		{User: openPlatformAppFGAUser(app.ID), Relation: ResourceRelationWriteByApp, Object: "resource_item:42"},
	}, resourceFGA.sortedTuples())
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.resource_access.granted", 1)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, adminID, "open_platform.resource_access.granted", map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "42",
		"reason":       "integration-test grant",
	})

	listed, err := service.ListResourceGrants(ctx, ResourceGrantListInput{
		AppID:        app.ID,
		ResourceType: ResourceTypeResourceItem,
	})
	require.NoError(t, err)
	require.Len(t, listed.Grants, 2)
	assert.Equal(t, ResourceAccessActionRead, listed.Grants[0].Action)
	assert.Equal(t, ResourceAccessActionWrite, listed.Grants[1].Action)

	decision, err := service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientID:     app.ClientID,
		ClientSecret: "test-secret",
		ResourceType: ResourceTypeResourceItem,
		ResourceID:   "42",
		Action:       ResourceAccessActionRead,
		RequestID:    "check-resource-read",
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
	assert.Equal(t, ResourceRelationReadByApp, decision.Relation)
	assertOpenPlatformAuditNullUserCount(t, postgres, app.ID, "open_platform.resource_access.checked", 1)

	revoked, err := service.RevokeResourceAccess(ctx, ResourceGrantRevokeInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "42",
		Actions:        []string{ResourceAccessActionRead},
		Reason:         "integration-test revoke",
		RequestID:      "revoke-resource-access",
	})
	require.NoError(t, err)
	require.Len(t, revoked.Grants, 1)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.resource_access.revoked", 1)

	decision, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientID:     app.ClientID,
		ClientSecret: "test-secret",
		ResourceType: ResourceTypeResourceItem,
		ResourceID:   "42",
		Action:       ResourceAccessActionRead,
		RequestID:    "check-resource-read-revoked",
	})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, "fga_denied", decision.Reason)

	decision, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientID:     app.ClientID,
		ClientSecret: "test-secret",
		ResourceType: ResourceTypeResourceItem,
		ResourceID:   "42",
		Action:       ResourceAccessActionWrite,
		RequestID:    "check-resource-write-still-granted",
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestResourceAccessGrantAndRevokeRequireReasons(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "resource-reason-admin")
	ownerID := seedOpenPlatformUser(t, postgres, "resource-reason-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
		ScopeResourceWrite,
	})

	_, err = service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "reason-42",
		Actions:        []string{ResourceAccessActionRead},
		Reason:         " ",
		RequestID:      "grant-resource-access-empty-reason",
	})
	require.ErrorIs(t, err, ErrResourceAccessReasonRequired)

	_, err = service.RevokeResourceAccess(ctx, ResourceGrantRevokeInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "reason-42",
		Actions:        []string{ResourceAccessActionRead},
		Reason:         "\t",
		RequestID:      "revoke-resource-access-empty-reason",
	})
	require.ErrorIs(t, err, ErrResourceAccessReasonRequired)
}

func TestResourceAccessCheckAcceptsClientCredentialsAccessTokenContext(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "resource-token-admin")
	ownerID := seedOpenPlatformUser(t, postgres, "resource-token-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
		ScopeResourceWrite,
	})
	_, err = service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "token-42",
		Actions:        []string{ResourceAccessActionRead, ResourceAccessActionWrite},
		Reason:         "token access",
		RequestID:      "grant-token-resource-access",
	})
	require.NoError(t, err)

	decision, err := service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		AccessTokenClientID: app.ClientID,
		AccessTokenScopes:   []string{ScopeResourceRead},
		ResourceType:        ResourceTypeResourceItem,
		ResourceID:          "token-42",
		Action:              ResourceAccessActionRead,
		RequestID:           "check-resource-token-read",
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)

	decision, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		AccessTokenClientID: app.ClientID,
		AccessTokenScopes:   []string{ScopeResourceRead},
		ResourceType:        ResourceTypeResourceItem,
		ResourceID:          "token-42",
		Action:              ResourceAccessActionWrite,
		RequestID:           "check-resource-token-write-missing-scope",
	})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, "token_scope_missing", decision.Reason)

	_, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientID:            app.ClientID,
		AccessTokenClientID: app.ClientID,
		AccessTokenScopes:   []string{ScopeResourceRead},
		ResourceType:        ResourceTypeResourceItem,
		ResourceID:          "token-42",
		Action:              ResourceAccessActionRead,
	})
	require.ErrorIs(t, err, ErrInvalidResourceAccess)

	_, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientSecret:        "body-secret-must-not-mix-with-bearer",
		AccessTokenClientID: app.ClientID,
		AccessTokenScopes:   []string{ScopeResourceRead},
		ResourceType:        ResourceTypeResourceItem,
		ResourceID:          "token-42",
		Action:              ResourceAccessActionRead,
	})
	require.ErrorIs(t, err, ErrInvalidResourceAccess)

	_, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ResourceType: ResourceTypeResourceItem,
		ResourceID:   "token-42",
		Action:       ResourceAccessActionRead,
	})
	require.ErrorIs(t, err, ErrInvalidResourceAccess)
}

func TestResourceAccessCheckDeniesWhenScopeNotApproved(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "resource-scope-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopeResourceRead})
	require.NoError(t, resourceFGA.WriteMissingTuples(ctx, []fga.Tuple{{
		User:     openPlatformAppFGAUser(app.ID),
		Relation: ResourceRelationWriteByApp,
		Object:   resourceFGAObject(ResourceTypeResourceItem, "77"),
	}}))

	decision, err := service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientID:     app.ClientID,
		ClientSecret: "test-secret",
		ResourceType: ResourceTypeResourceItem,
		ResourceID:   "77",
		Action:       ResourceAccessActionWrite,
		RequestID:    "check-resource-write-no-scope",
	})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, "scope_not_approved", decision.Reason)
	assertOpenPlatformAuditNullUserCount(t, postgres, app.ID, "open_platform.resource_access.checked", 1)
}

func TestResourceAccessUserProfileSupportsReadOnly(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "resource-profile-admin")
	ownerID := seedOpenPlatformUser(t, postgres, "resource-profile-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
		ScopeResourceWrite,
	})

	_, err = service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeUserProfile,
		ResourceID:     "1001",
		Actions:        []string{ResourceAccessActionWrite},
		Reason:         "user profile write should be rejected",
	})
	require.ErrorIs(t, err, ErrInvalidResourceAccess)

	granted, err := service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeUserProfile,
		ResourceID:     "1001",
		Actions:        []string{ResourceAccessActionRead},
		Reason:         "user profile read approved",
	})
	require.NoError(t, err)
	require.Len(t, granted.Grants, 1)
	assert.Equal(t, ResourceRelationReadByApp, granted.Grants[0].Relation)

	require.NoError(t, resourceFGA.WriteMissingTuples(ctx, []fga.Tuple{{
		User:     openPlatformAppFGAUser(app.ID),
		Relation: ResourceRelationWriteByApp,
		Object:   resourceFGAObject(ResourceTypeUserProfile, "1001"),
	}}))
	listed, err := service.ListResourceGrants(ctx, ResourceGrantListInput{
		AppID:        app.ID,
		ResourceType: ResourceTypeUserProfile,
	})
	require.NoError(t, err)
	require.Len(t, listed.Grants, 1)
	assert.Equal(t, ResourceAccessActionRead, listed.Grants[0].Action)

	_, err = service.CheckResourceAccess(ctx, ResourceAccessCheckInput{
		ClientID:     app.ClientID,
		ClientSecret: "test-secret",
		ResourceType: ResourceTypeUserProfile,
		ResourceID:   "1001",
		Action:       ResourceAccessActionWrite,
	})
	require.ErrorIs(t, err, ErrInvalidResourceAccess)
}

func TestRevokeAppDeletesResourceAccessTuples(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "resource-cleanup-admin")
	ownerID := seedOpenPlatformUser(t, postgres, "resource-cleanup-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
		ScopeResourceWrite,
	})
	otherAppID := app.ID + 999
	require.NoError(t, resourceFGA.WriteMissingTuples(ctx, []fga.Tuple{
		{
			User:     openPlatformAppFGAUser(app.ID),
			Relation: ResourceRelationReadByApp,
			Object:   resourceFGAObject(ResourceTypeResourceItem, "42"),
		},
		{
			User:     openPlatformAppFGAUser(app.ID),
			Relation: ResourceRelationWriteByApp,
			Object:   resourceFGAObject(ResourceTypeResourceItem, "42"),
		},
		{
			User:     openPlatformAppFGAUser(app.ID),
			Relation: ResourceRelationReadByApp,
			Object:   resourceFGAObject(ResourceTypeUserProfile, "1001"),
		},
		{
			User:     openPlatformAppFGAUser(otherAppID),
			Relation: ResourceRelationReadByApp,
			Object:   resourceFGAObject(ResourceTypeResourceItem, "keep"),
		},
	}))

	revoked, err := service.RevokeApp(ctx, AppLifecycleActionInput{
		AppID:       app.ID,
		ActorUserID: adminID,
		Reason:      "cleanup resource grants when revoking app",
		RequestID:   "revoke-resource-cleanup",
	})
	require.NoError(t, err)
	assert.Equal(t, AppStatusRevoked, revoked.App.Status)
	assert.Equal(t, []fga.Tuple{{
		User:     openPlatformAppFGAUser(otherAppID),
		Relation: ResourceRelationReadByApp,
		Object:   resourceFGAObject(ResourceTypeResourceItem, "keep"),
	}}, resourceFGA.sortedTuples())

	_, err = service.VerifyClientSecret(ctx, app.ClientID, "test-secret")
	require.ErrorIs(t, err, ErrAppNotActive)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.app.revoked", 1)
	assertOpenPlatformAuditCount(t, postgres, app.ID, adminID, "open_platform.resource_access.revoked", 2)
	assertOpenPlatformAuditMetadata(t, postgres, app.ID, adminID, "open_platform.resource_access.revoked", map[string]any{
		"reason": "cleanup resource grants when revoking app",
		"source": "app_lifecycle",
	})

	resourceItemGrants, err := service.ListResourceGrants(ctx, ResourceGrantListInput{
		AppID:        app.ID,
		ResourceType: ResourceTypeResourceItem,
	})
	require.NoError(t, err)
	assert.Empty(t, resourceItemGrants.Grants)
	userProfileGrants, err := service.ListResourceGrants(ctx, ResourceGrantListInput{
		AppID:        app.ID,
		ResourceType: ResourceTypeUserProfile,
	})
	require.NoError(t, err)
	assert.Empty(t, userProfileGrants.Grants)
}

func TestResourceAccessFailsClosedWithoutOpenFGA(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client)
	require.NoError(t, err)

	_, err = service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          1,
		ReviewerUserID: 1,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "42",
		Actions:        []string{ResourceAccessActionRead},
	})
	require.ErrorIs(t, err, ErrResourceAccessUnavailable)
}

func assertOpenPlatformAuditNullUserCount(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	appID int64,
	eventType string,
	want int64,
) {
	t.Helper()
	var count int64
	err := fixture.DB.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM open_platform_audit_events
		WHERE app_id = $1
		  AND user_id IS NULL
		  AND event_type = $2
	`, appID, eventType).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, want, count)
}

type fakeResourceFGA struct {
	tuples map[fga.Tuple]struct{}
}

func newFakeResourceFGA() *fakeResourceFGA {
	return &fakeResourceFGA{tuples: map[fga.Tuple]struct{}{}}
}

func (f *fakeResourceFGA) Check(_ context.Context, user, relation, object string) (bool, error) {
	_, ok := f.tuples[fga.Tuple{User: user, Relation: relation, Object: object}]
	return ok, nil
}

func (f *fakeResourceFGA) WriteMissingTuples(_ context.Context, desired []fga.Tuple) error {
	for _, tuple := range desired {
		f.tuples[tuple] = struct{}{}
	}
	return nil
}

func (f *fakeResourceFGA) DeleteTuples(_ context.Context, tuples []fga.Tuple) error {
	for _, tuple := range tuples {
		delete(f.tuples, tuple)
	}
	return nil
}

func (f *fakeResourceFGA) ListObjects(_ context.Context, user, relation, objectType string) ([]string, error) {
	objects := make([]string, 0)
	prefix := objectType + ":"
	for tuple := range f.tuples {
		if tuple.User == user && tuple.Relation == relation && strings.HasPrefix(tuple.Object, prefix) {
			objects = append(objects, tuple.Object)
		}
	}
	sort.Strings(objects)
	return objects, nil
}

func (f *fakeResourceFGA) sortedTuples() []fga.Tuple {
	tuples := make([]fga.Tuple, 0, len(f.tuples))
	for tuple := range f.tuples {
		tuples = append(tuples, tuple)
	}
	sort.Slice(tuples, func(i, j int) bool {
		if tuples[i].Object != tuples[j].Object {
			return tuples[i].Object < tuples[j].Object
		}
		if tuples[i].Relation != tuples[j].Relation {
			return tuples[i].Relation < tuples[j].Relation
		}
		return tuples[i].User < tuples[j].User
	})
	return tuples
}
