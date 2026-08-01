package authorization

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestAuthorizationGrantLifecycleUsesDBFenceAndOpenFGAProjection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "grant-actor")
	targetID := seedAuthorizationUser(t, postgres, "grant-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010006)

	created, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSchoolAdmin,
		SchoolID:      &schoolID,
		Reason:        "manage school reviews",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	assert.True(t, created.Changed)
	assert.Equal(t, DesiredGranted, created.Grant.DesiredState)
	assert.Equal(t, ProjectionPending, created.Grant.ProjectionStatus)

	beforeProjection, err := repo.ResolveAccessSnapshotByUserID(ctx, targetID)
	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, beforeProjection.Roles)
	assert.Nil(t, beforeProjection.RoleScopes)
	assertAuthorizationOutboxStatus(t, postgres, created.Grant.ID, outbox.StatusPending)
	assertAuthorizationAuditCount(t, postgres, created.Grant.ID, "grant", 1)

	require.NoError(t, service.ProcessProjectionBatch(ctx))
	applied, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, ProjectionApplied, applied.ProjectionStatus)
	require.NotNil(t, applied.ActivatedAt)
	assertAuthorizationOutboxStatus(t, postgres, created.Grant.ID, outbox.StatusCompleted)
	assertAuthorizationAuditCount(t, postgres, created.Grant.ID, "projection_applied", 1)

	afterProjection, err := repo.ResolveAccessSnapshotByUserID(ctx, targetID)
	require.NoError(t, err)
	assert.Contains(t, afterProjection.Roles, string(RoleSchoolAdmin))
	assert.Equal(t, []string{"4111010006"}, afterProjection.RoleScopes[string(RoleSchoolAdmin)])
	assert.True(t, projector.has(fga.Tuple{
		User:     fmt.Sprintf("user:%d", targetID),
		Relation: "admin",
		Object:   "school:4111010006",
	}))

	duplicate, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSchoolAdmin,
		SchoolID:      &schoolID,
		Reason:        "idempotent retry",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	assert.False(t, duplicate.Changed)
	assert.Equal(t, applied.Revision, duplicate.Grant.Revision)

	revoked, err := service.RevokeGrant(ctx, RevokeGrantInput{
		GrantID:     created.Grant.ID,
		Reason:      "operator rotation",
		ActorUserID: actorID,
	})
	require.NoError(t, err)
	assert.True(t, revoked.Changed)
	assert.Equal(t, DesiredRevoked, revoked.Grant.DesiredState)
	assert.Equal(t, ProjectionPending, revoked.Grant.ProjectionStatus)

	// The DB desired-state fence removes access before OpenFGA deletion succeeds.
	duringRevocation, err := repo.ResolveAccessSnapshotByUserID(ctx, targetID)
	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, duringRevocation.Roles)
	assert.True(t, projector.has(fga.Tuple{
		User:     fmt.Sprintf("user:%d", targetID),
		Relation: "admin",
		Object:   "school:4111010006",
	}))

	require.NoError(t, service.ProcessProjectionBatch(ctx))
	finalGrant, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, DesiredRevoked, finalGrant.DesiredState)
	assert.Equal(t, ProjectionApplied, finalGrant.ProjectionStatus)
	require.NotNil(t, finalGrant.RevokedAt)
	assert.False(t, projector.has(fga.Tuple{
		User:     fmt.Sprintf("user:%d", targetID),
		Relation: "admin",
		Object:   "school:4111010006",
	}))

	duplicateRevoke, err := service.RevokeGrant(ctx, RevokeGrantInput{
		GrantID:     created.Grant.ID,
		Reason:      "idempotent retry",
		ActorUserID: actorID,
	})
	require.NoError(t, err)
	assert.False(t, duplicateRevoke.Changed)
	assert.Equal(t, finalGrant.Revision, duplicateRevoke.Grant.Revision)
}

func TestAuthorizationConcurrentCreateIsIdempotentWhileProjectionIsPending(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(NewRepository(postgres.DB))

	actorID := seedAuthorizationUser(t, postgres, "concurrent-grant-actor")
	targetID := seedAuthorizationUser(t, postgres, "concurrent-grant-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010010)

	start := make(chan struct{})
	results := make(chan MutationResult, 2)
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			result, err := service.CreateGrant(ctx, CreateGrantInput{
				SubjectUserID: targetID,
				Role:          RoleSchoolAdmin,
				SchoolID:      &schoolID,
				Reason:        "concurrent idempotent request",
				ActorUserID:   actorID,
			})
			results <- result
			errors <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errors)

	changed := 0
	var grantID int64
	for err := range errors {
		require.NoError(t, err)
	}
	for result := range results {
		if result.Changed {
			changed++
		}
		grantID = result.Grant.ID
		assert.Equal(t, int64(1), result.Grant.Revision)
		assert.Equal(t, ProjectionPending, result.Grant.ProjectionStatus)
	}
	assert.Equal(t, 1, changed)

	var count int
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM authorization_grants
		WHERE subject_user_id = $1
		  AND role = 'school_admin'
	`, targetID).Scan(&count))
	assert.Equal(t, 1, count)
	assertAuthorizationAuditCount(t, postgres, grantID, "grant", 1)
	assertAuthorizationOutboxStatus(t, postgres, grantID, outbox.StatusPending)
}

func TestAuthorizationSyncCasdoorOrganizationAdminIsSystemAttributed(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(NewRepository(postgres.DB))

	userID := seedAuthorizationUser(t, postgres, "casdoor-organization-admin")
	result, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: true,
	})
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.Equal(t, RoleSuperAdmin, result.Grant.Role)
	assert.Equal(t, GrantSourceCasdoorOrganizationAdmin, result.Grant.Source)
	assert.Equal(t, DesiredGranted, result.Grant.DesiredState)
	assert.Equal(t, ProjectionPending, result.Grant.ProjectionStatus)
	assert.Nil(t, result.Grant.CreatedByUserID)
	assert.Nil(t, result.Grant.UpdatedByUserID)
	assertAuthorizationOutboxStatus(t, postgres, result.Grant.ID, outbox.StatusPending)

	var actorType, actorUserID string
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT actor_type, actor_user_id
		FROM audit_events
		WHERE resource_type = 'authorization_grant'
		  AND resource_id = $1
		  AND event_type = 'iam.authorization_grant.provider_grant_requested'
	`, fmt.Sprintf("%d", result.Grant.ID)).Scan(&actorType, &actorUserID))
	assert.Equal(t, "system", actorType)
	assert.Equal(t, "casdoor-org-admin-sync", actorUserID)

	unchanged, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: true,
	})
	require.NoError(t, err)
	assert.False(t, unchanged.Changed)
	assert.Equal(t, result.Grant.ID, unchanged.Grant.ID)
}

func TestAuthorizationSyncCasdoorOrganizationAdminRequeuesFailedProjection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(NewRepository(postgres.DB))

	userID := seedAuthorizationUser(t, postgres, "casdoor-organization-admin-retry")
	granted, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: true,
	})
	require.NoError(t, err)

	markAuthorizationProjectionFailed(t, postgres, granted.Grant.ID)
	retriedGrant, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: true,
	})
	require.NoError(t, err)
	assert.True(t, retriedGrant.Changed)
	assert.Equal(t, granted.Grant.Revision+1, retriedGrant.Grant.Revision)
	assert.Equal(t, DesiredGranted, retriedGrant.Grant.DesiredState)
	assert.Equal(t, ProjectionPending, retriedGrant.Grant.ProjectionStatus)
	assertAuthorizationOutboxRetryReset(t, postgres, granted.Grant.ID)
	assertAuthorizationAuditCount(t, postgres, granted.Grant.ID, "provider_reconcile", 1)

	revoked, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: false,
	})
	require.NoError(t, err)
	require.True(t, revoked.Changed)
	markAuthorizationProjectionFailed(t, postgres, granted.Grant.ID)

	retriedRevoke, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: false,
	})
	require.NoError(t, err)
	assert.True(t, retriedRevoke.Changed)
	assert.Equal(t, revoked.Grant.Revision+1, retriedRevoke.Grant.Revision)
	assert.Equal(t, DesiredRevoked, retriedRevoke.Grant.DesiredState)
	assert.Equal(t, ProjectionPending, retriedRevoke.Grant.ProjectionStatus)
	assertAuthorizationOutboxRetryReset(t, postgres, granted.Grant.ID)
	assertAuthorizationAuditCount(t, postgres, granted.Grant.ID, "provider_reconcile", 2)
}

func TestAuthorizationGrantRevisionSupersedesPendingProjection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "revision-actor")
	targetID := seedAuthorizationUser(t, postgres, "revision-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010007)

	created, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSectionModerator,
		SchoolID:      &schoolID,
		SectionID:     authorizationSectionID(schoolID),
		Reason:        "temporary moderation",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	revoked, err := service.RevokeGrant(ctx, RevokeGrantInput{
		GrantID:     created.Grant.ID,
		Reason:      "cancel before activation",
		ActorUserID: actorID,
	})
	require.NoError(t, err)
	assert.Greater(t, revoked.Grant.Revision, created.Grant.Revision)

	require.NoError(t, service.ProcessProjectionBatch(ctx))
	finalGrant, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, DesiredRevoked, finalGrant.DesiredState)
	assert.Equal(t, ProjectionApplied, finalGrant.ProjectionStatus)
	assert.Empty(t, projector.snapshot())
}

func TestAuthorizationReconcilePreservesActiveGrantWhileRepairingProjection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "reconcile-actor")
	targetID := seedAuthorizationUser(t, postgres, "reconcile-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010008)

	created, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSchoolAdmin,
		SchoolID:      &schoolID,
		Reason:        "school administration",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	require.NoError(t, service.ProcessProjectionBatch(ctx))

	reconciled, err := service.ReconcileGrant(ctx, ReconcileGrantInput{
		GrantID:     created.Grant.ID,
		Reason:      "repair serving projection",
		ActorUserID: actorID,
	})
	require.NoError(t, err)
	assert.True(t, reconciled.Changed)
	assert.Equal(t, DesiredGranted, reconciled.Grant.DesiredState)
	assert.Equal(t, ProjectionPending, reconciled.Grant.ProjectionStatus)
	assert.Greater(t, reconciled.Grant.Revision, created.Grant.Revision)
	assertAuthorizationOutboxStatus(t, postgres, created.Grant.ID, outbox.StatusPending)
	assertAuthorizationAuditCount(t, postgres, created.Grant.ID, "reconcile", 1)

	duringReconcile, err := repo.ResolveAccessSnapshotByUserID(ctx, targetID)
	require.NoError(t, err)
	assert.Contains(t, duringReconcile.Roles, string(RoleSchoolAdmin))
	assert.Equal(t, []string{"4111010008"}, duringReconcile.RoleScopes[string(RoleSchoolAdmin)])

	require.NoError(t, service.ProcessProjectionBatch(ctx))
	applied, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, ProjectionApplied, applied.ProjectionStatus)

	projection := ProjectionApplied
	list, err := service.ListGrants(ctx, ListGrantsFilter{Projection: &projection})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, created.Grant.ID, list.Items[0].ID)
	assert.NotEmpty(t, list.Items[0].SubjectUsername)
}

func TestAuthorizationReconcileAllRebuildsOpenFGAFromLedgerWithoutLockout(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "rebuild-actor")
	targetID := seedAuthorizationUser(t, postgres, "rebuild-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010009)
	created, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSchoolAdmin,
		SchoolID:      &schoolID,
		Reason:        "school administration",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	require.NoError(t, service.ProcessProjectionBatch(ctx))
	projector.clear()

	result, err := service.ReconcileAll(ctx, ReconcileAllInput{
		Reason:      "disaster recovery rebuild",
		ActorUserID: actorID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Queued)

	pending, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, ProjectionPending, pending.ProjectionStatus)
	require.NotNil(t, pending.ActivatedAt)
	snapshot, err := repo.ResolveAccessSnapshotByUserID(ctx, targetID)
	require.NoError(t, err)
	assert.Contains(t, snapshot.Roles, string(RoleSchoolAdmin))

	require.NoError(t, service.ProcessProjectionBatch(ctx))
	assert.True(t, projector.has(fga.Tuple{
		User:     fmt.Sprintf("user:%d", targetID),
		Relation: "admin",
		Object:   "school:4111010009",
	}))
	applied, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, ProjectionApplied, applied.ProjectionStatus)
	assertAuthorizationAuditCount(t, postgres, created.Grant.ID, "reconcile", 1)
}

func TestAuthorizationScheduledReconciliationRepairsOnlyDriftedProjection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "scheduled-reconcile-actor")
	targetID := seedAuthorizationUser(t, postgres, "scheduled-reconcile-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010011)
	created, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSchoolAdmin,
		SchoolID:      &schoolID,
		Reason:        "school administration",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	require.NoError(t, service.ProcessProjectionBatch(ctx))

	queued, err := service.ReconcileProjectionDrift(ctx, 100, time.Now())
	require.NoError(t, err)
	assert.Zero(t, queued)

	projector.clear()
	queued, err = service.ReconcileProjectionDrift(ctx, 100, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, queued)

	pending, err := repo.GetGrant(ctx, created.Grant.ID)
	require.NoError(t, err)
	assert.Equal(t, ProjectionPending, pending.ProjectionStatus)
	require.NotNil(t, pending.ActivatedAt)
	assert.Greater(t, pending.Revision, created.Grant.Revision)
	snapshot, err := repo.ResolveAccessSnapshotByUserID(ctx, targetID)
	require.NoError(t, err)
	assert.Contains(t, snapshot.Roles, string(RoleSchoolAdmin))

	var actorType, actorUserID string
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT actor_type, actor_user_id
		FROM audit_events
		WHERE resource_type = 'authorization_grant'
		  AND resource_id = $1
		  AND event_type = 'iam.authorization_grant.reconcile_requested'
		ORDER BY created_at DESC
		LIMIT 1
	`, fmt.Sprintf("%d", created.Grant.ID)).Scan(&actorType, &actorUserID))
	assert.Equal(t, "system", actorType)
	assert.Equal(t, "authorization-reconciliation", actorUserID)

	require.NoError(t, service.ProcessProjectionBatch(ctx))
	assert.True(t, projector.has(fga.Tuple{
		User:     fmt.Sprintf("user:%d", targetID),
		Relation: "admin",
		Object:   "school:4111010011",
	}))
}

func TestAuthorizationScheduledReconciliationStopsAboveDriftThreshold(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "drift-threshold-actor")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010012)
	grantIDs := make([]int64, 0, 2)
	for index := range 2 {
		targetID := seedAuthorizationUser(
			t,
			postgres,
			fmt.Sprintf("drift-threshold-target-%d", index),
		)
		created, err := service.CreateGrant(ctx, CreateGrantInput{
			SubjectUserID: targetID,
			Role:          RoleSchoolAdmin,
			SchoolID:      &schoolID,
			Reason:        "school administration",
			ActorUserID:   actorID,
		})
		require.NoError(t, err)
		grantIDs = append(grantIDs, created.Grant.ID)
	}
	require.NoError(t, service.ProcessProjectionBatch(ctx))
	projector.clear()

	queued, err := service.ReconcileProjectionDrift(ctx, 1, time.Now())
	require.ErrorIs(t, err, ErrReconciliationLimit)
	assert.Zero(t, queued)
	for _, grantID := range grantIDs {
		grant, getErr := repo.GetGrant(ctx, grantID)
		require.NoError(t, getErr)
		assert.Equal(t, ProjectionApplied, grant.ProjectionStatus)
		assert.Equal(t, int64(1), grant.Revision)
	}
}

func TestAuthorizationSuperAdminIsProviderManagedAndSingleAdminCanBeDemoted(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "super-actor")

	first, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: actorID,
		Role:          RoleSuperAdmin,
		Reason:        "manual platform administrator",
		ActorUserID:   actorID,
	})
	require.ErrorIs(t, err, ErrProviderManagedRole)
	assert.Zero(t, first.Grant.ID)

	first, err = service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     actorID,
		OrganizationAdmin: true,
	})
	require.NoError(t, err)
	require.NoError(t, service.ProcessProjectionBatch(ctx))

	_, err = service.RevokeGrant(ctx, RevokeGrantInput{
		GrantID:     first.Grant.ID,
		Reason:      "manual revoke is forbidden",
		ActorUserID: actorID,
	})
	require.ErrorIs(t, err, ErrProviderManagedRole)

	revoked, err := service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     actorID,
		OrganizationAdmin: false,
	})
	require.NoError(t, err)
	assert.True(t, revoked.Changed)
	assert.Equal(t, DesiredRevoked, revoked.Grant.DesiredState)
	snapshot, err := service.ResolveAccessSnapshotByUserID(ctx, actorID)
	require.NoError(t, err)
	assert.NotContains(t, snapshot.Roles, string(RoleSuperAdmin))
}

func TestAuthorizationGrantSourceConstraintRejectsMixedAuthorities(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	userID := seedAuthorizationUser(t, postgres, "authority-source-constraint")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010888)

	_, err := postgres.Pool.Exec(ctx, `
		INSERT INTO authorization_grants (subject_user_id, role, reason)
		VALUES ($1, 'super_admin', 'manual source must be rejected')
	`, userID)
	require.Error(t, err)

	_, err = postgres.Pool.Exec(ctx, `
		INSERT INTO authorization_grants (
			subject_user_id, role, source, school_id, reason
		) VALUES ($1, 'school_admin', 'casdoor_org_admin', $2, 'provider source must be rejected')
	`, userID, schoolID)
	require.Error(t, err)

	var count int
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM authorization_grants
		WHERE subject_user_id = $1
	`, userID).Scan(&count))
	assert.Zero(t, count)
}

func TestAuthorizationGrantTransactionRollsBackWhenAuditWriteFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service := NewService(repo)

	actorID := seedAuthorizationUser(t, postgres, "atomic-actor")
	targetID := seedAuthorizationUser(t, postgres, "atomic-target")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010999)
	_, err := postgres.Pool.Exec(ctx, `DROP TABLE audit_events`)
	require.NoError(t, err)

	_, err = service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSchoolAdmin,
		SchoolID:      &schoolID,
		Reason:        "must roll back",
		ActorUserID:   actorID,
	})
	require.Error(t, err)

	var count int
	require.NoError(t, postgres.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM authorization_grants`).Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM domain_event_outbox
		WHERE stream = $1
	`, outbox.StreamIAMAuthorizationGrantProjection).Scan(&count))
	assert.Zero(t, count)
}

func TestAuthorizationCasdoorAdminSyncRollsBackWhenAuditWriteFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(NewRepository(postgres.DB))

	userID := seedAuthorizationUser(t, postgres, "atomic-casdoor-admin")
	_, err := postgres.Pool.Exec(ctx, `DROP TABLE audit_events`)
	require.NoError(t, err)

	_, err = service.SyncCasdoorOrganizationAdmin(ctx, CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: true,
	})
	require.Error(t, err)

	var count int
	require.NoError(t, postgres.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM authorization_grants`).Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM domain_event_outbox
		WHERE stream = $1
	`, outbox.StreamIAMAuthorizationGrantProjection).Scan(&count))
	assert.Zero(t, count)
}

type fakeProjectionClient struct {
	mu     sync.Mutex
	tuples map[fga.Tuple]struct{}
	err    error
}

func newFakeProjectionClient() *fakeProjectionClient {
	return &fakeProjectionClient{tuples: make(map[fga.Tuple]struct{})}
}

func (f *fakeProjectionClient) WriteMissingTuples(_ context.Context, desired []fga.Tuple) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	for _, tuple := range desired {
		f.tuples[tuple] = struct{}{}
	}
	return nil
}

func (f *fakeProjectionClient) TupleExists(_ context.Context, tuple fga.Tuple) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return false, f.err
	}
	_, exists := f.tuples[tuple]
	return exists, nil
}

func (f *fakeProjectionClient) DeleteTuplesIgnoringMissing(_ context.Context, tuples []fga.Tuple) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	for _, tuple := range tuples {
		delete(f.tuples, tuple)
	}
	return nil
}

func (f *fakeProjectionClient) has(tuple fga.Tuple) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, exists := f.tuples[tuple]
	return exists
}

func (f *fakeProjectionClient) snapshot() []fga.Tuple {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]fga.Tuple, 0, len(f.tuples))
	for tuple := range f.tuples {
		result = append(result, tuple)
	}
	return result
}

func (f *fakeProjectionClient) clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tuples = make(map[fga.Tuple]struct{})
}

func seedAuthorizationUser(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	name string,
) int64 {
	t.Helper()
	var userID int64
	err := postgres.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email, user_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, name+"-subject", name, name+"@example.com", fmt.Sprintf("%x", sha256.Sum256([]byte(name)))).Scan(&userID)
	require.NoError(t, err)
	return userID
}

func seedAuthorizationSchool(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	schoolID int64,
) int64 {
	t.Helper()
	_, err := postgres.Pool.Exec(context.Background(), `
		INSERT INTO schools (id, code, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name
	`, schoolID, fmt.Sprintf("%010d", schoolID%10000000000), fmt.Sprintf("School %d", schoolID))
	require.NoError(t, err)
	return schoolID
}

func authorizationSectionID(schoolID int64) *string {
	value := fga.ReviewModerationSectionID(fmt.Sprintf("%d", schoolID))
	return &value
}

func assertAuthorizationOutboxStatus(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	grantID int64,
	want string,
) {
	t.Helper()
	var status string
	err := postgres.Pool.QueryRow(context.Background(), `
		SELECT status
		FROM domain_event_outbox
		WHERE stream = $1 AND dedupe_key = $2
	`, outbox.StreamIAMAuthorizationGrantProjection, projectionDedupeKey(grantID)).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, want, status)
}

func markAuthorizationProjectionFailed(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	grantID int64,
) {
	t.Helper()
	grantTag, err := postgres.Pool.Exec(context.Background(), `
		UPDATE authorization_grants
		SET projection_status = 'failed',
		    last_error = 'injected terminal projection failure',
		    updated_at = NOW()
		WHERE id = $1
	`, grantID)
	require.NoError(t, err)
	require.Equal(t, int64(1), grantTag.RowsAffected())

	outboxTag, err := postgres.Pool.Exec(context.Background(), `
		UPDATE domain_event_outbox
		SET status = 'dead_letter',
		    attempt_count = 5,
		    locked_at = NULL,
		    locked_revision = NULL,
		    last_error = 'injected terminal projection failure',
		    updated_at = NOW()
		WHERE stream = $1 AND dedupe_key = $2
	`, outbox.StreamIAMAuthorizationGrantProjection, projectionDedupeKey(grantID))
	require.NoError(t, err)
	require.Equal(t, int64(1), outboxTag.RowsAffected())
}

func assertAuthorizationOutboxRetryReset(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	grantID int64,
) {
	t.Helper()
	var status string
	var attempts int
	var lastError *string
	err := postgres.Pool.QueryRow(context.Background(), `
		SELECT status, attempt_count, last_error
		FROM domain_event_outbox
		WHERE stream = $1 AND dedupe_key = $2
	`, outbox.StreamIAMAuthorizationGrantProjection, projectionDedupeKey(grantID)).Scan(
		&status,
		&attempts,
		&lastError,
	)
	require.NoError(t, err)
	assert.Equal(t, outbox.StatusPending, status)
	assert.Zero(t, attempts)
	assert.Nil(t, lastError)
}

func assertAuthorizationAuditCount(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	grantID int64,
	action string,
	want int,
) {
	t.Helper()
	var count int
	err := postgres.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM audit_events
		WHERE resource_type = 'authorization_grant'
		  AND resource_id = $1
		  AND action = $2
	`, fmt.Sprintf("%d", grantID), action).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, want, count)
}
