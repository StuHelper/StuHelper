package authorization

import (
	"context"
	"fmt"
	"sync"
	"testing"

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

	beforeProjection, err := repo.ResolveAccessSnapshot(ctx, "grant-target-subject")
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

	afterProjection, err := repo.ResolveAccessSnapshot(ctx, "grant-target-subject")
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
	duringRevocation, err := repo.ResolveAccessSnapshot(ctx, "grant-target-subject")
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

func TestAuthorizationReconcileRequeuesCurrentDesiredStateFailClosed(t *testing.T) {
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

	duringReconcile, err := repo.ResolveAccessSnapshot(ctx, "reconcile-target-subject")
	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, duringReconcile.Roles)

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

func TestAuthorizationPreventsRevokingLastAppliedSuperAdmin(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	projector := newFakeProjectionClient()
	service := NewService(repo, WithProjectionClient(projector))

	actorID := seedAuthorizationUser(t, postgres, "super-actor")
	secondID := seedAuthorizationUser(t, postgres, "super-second")

	first, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: actorID,
		Role:          RoleSuperAdmin,
		Reason:        "bootstrap primary",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	require.NoError(t, service.ProcessProjectionBatch(ctx))

	_, err = service.RevokeGrant(ctx, RevokeGrantInput{
		GrantID:     first.Grant.ID,
		Reason:      "must retain one administrator",
		ActorUserID: actorID,
	})
	require.ErrorIs(t, err, ErrLastSuperAdmin)

	second, err := service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: secondID,
		Role:          RoleSuperAdmin,
		Reason:        "establish redundant administrator",
		ActorUserID:   actorID,
	})
	require.NoError(t, err)
	require.NoError(t, service.ProcessProjectionBatch(ctx))
	assert.NotEqual(t, first.Grant.ID, second.Grant.ID)

	_, err = service.RevokeGrant(ctx, RevokeGrantInput{
		GrantID:     first.Grant.ID,
		Reason:      "rotate primary administrator",
		ActorUserID: actorID,
	})
	require.NoError(t, err)
}

func TestAuthorizationGrantTransactionRollsBackWhenAuditWriteFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service := NewService(repo)

	actorID := seedAuthorizationUser(t, postgres, "atomic-actor")
	targetID := seedAuthorizationUser(t, postgres, "atomic-target")
	_, err := postgres.Pool.Exec(ctx, `DROP TABLE audit_events`)
	require.NoError(t, err)

	_, err = service.CreateGrant(ctx, CreateGrantInput{
		SubjectUserID: targetID,
		Role:          RoleSuperAdmin,
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
	`, name+"-subject", name, name+"@example.com", fmt.Sprintf("%064d", len(name))).Scan(&userID)
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
