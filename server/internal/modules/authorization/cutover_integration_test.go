package authorization

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type fakeAuthorityCutoverTupleReader struct {
	tuples map[string][]fga.Tuple
	err    error
	reads  int
}

func (f *fakeAuthorityCutoverTupleReader) ReadTuples(
	_ context.Context,
	object,
	relation string,
) ([]fga.Tuple, error) {
	f.reads++
	if f.err != nil {
		return nil, f.err
	}
	return append([]fga.Tuple(nil), f.tuples[object+"#"+relation]...), nil
}

func TestAuthorityCutoverImportsVerifiedIntersectionAndSealsMarker(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	projection := newFakeProjectionClient()
	service := NewService(NewRepository(postgres.DB), WithProjectionClient(projection))

	require.ErrorIs(t, service.RequireAuthorityCutoverComplete(ctx), ErrAuthorityCutoverIncomplete)

	superAdminID := seedAuthorizationUser(t, postgres, "cutover-super-admin")
	scopedAdminID := seedAuthorizationUser(t, postgres, "cutover-scoped-admin")
	inactiveID := seedAuthorizationUser(t, postgres, "cutover-inactive-admin")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010024)
	sectionID := fga.ReviewModerationSectionID(fmt.Sprintf("%d", schoolID))

	reader := &fakeAuthorityCutoverTupleReader{tuples: map[string][]fga.Tuple{
		fmt.Sprintf("school:%d#admin", schoolID): {
			{User: fmt.Sprintf("user:%d", scopedAdminID), Relation: "admin", Object: fmt.Sprintf("school:%d", schoolID)},
			{User: fmt.Sprintf("user:%d", inactiveID), Relation: "admin", Object: fmt.Sprintf("school:%d", schoolID)},
		},
		"section:" + sectionID + "#section_moderator": {
			{User: fmt.Sprintf("user:%d", scopedAdminID), Relation: "section_moderator", Object: "section:" + sectionID},
		},
	}}
	snapshot := LegacyAuthoritySnapshot{
		Organization: "stuhelper",
		Users: []LegacyAuthorityIdentity{
			{
				ID:                "cutover-super-admin-subject",
				Owner:             "stuhelper",
				Name:              "cutover-super-admin",
				OrganizationAdmin: true,
			},
			{
				ID:    "cutover-scoped-admin-subject",
				Owner: "stuhelper",
				Name:  "cutover-scoped-admin",
			},
			{
				ID:                 "cutover-inactive-admin-subject",
				Owner:              "stuhelper",
				Name:               "cutover-inactive-admin",
				ForbiddenOrDeleted: true,
			},
		},
		RoleMembers: map[Role][]string{
			RoleSchoolAdmin:      {"stuhelper/cutover-scoped-admin", "stuhelper/cutover-inactive-admin"},
			RoleSectionModerator: {"cutover-scoped-admin-subject"},
		},
	}

	users := []AuthorityCutoverUser{
		{InternalUserID: superAdminID, ProviderSubject: "cutover-super-admin-subject"},
		{InternalUserID: scopedAdminID, ProviderSubject: "cutover-scoped-admin-subject"},
		{InternalUserID: inactiveID, ProviderSubject: "cutover-inactive-admin-subject"},
	}
	result, err := service.ImportLegacyAuthority(ctx, users, snapshot, reader)
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.Len(t, result.SourceDigest, 64)
	assert.Equal(t, 3, result.ImportedGrantCount)
	assert.Equal(t, 1, result.SkippedTupleCount)
	require.NoError(t, service.RequireAuthorityCutoverComplete(ctx))

	assert.True(t, projection.has(fga.Tuple{
		User: "user:" + fmt.Sprint(superAdminID), Relation: "super_admin", Object: "ecosystem:stuhelper",
	}))
	assert.True(t, projection.has(fga.Tuple{
		User: "user:" + fmt.Sprint(scopedAdminID), Relation: "admin", Object: fmt.Sprintf("school:%d", schoolID),
	}))
	assert.True(t, projection.has(fga.Tuple{
		User: "user:" + fmt.Sprint(scopedAdminID), Relation: "section_moderator", Object: "section:" + sectionID,
	}))

	var (
		grantCount  int
		outboxCount int
		auditCount  int
	)
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM authorization_grants
		WHERE desired_state = 'granted'
		  AND projection_status = 'applied'
	`).Scan(&grantCount))
	assert.Equal(t, 3, grantCount)
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM domain_event_outbox
		WHERE stream = $1
	`, outbox.StreamIAMAuthorizationGrantProjection).Scan(&outboxCount))
	assert.Equal(t, 3, outboxCount)
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM audit_events
		WHERE event_type = 'iam.authorization_grant.cutover_imported'
	`).Scan(&auditCount))
	assert.Equal(t, 3, auditCount)

	readsBeforeRetry := reader.reads
	retry, err := service.ImportLegacyAuthority(ctx, nil, LegacyAuthoritySnapshot{}, nil)
	require.NoError(t, err)
	assert.False(t, retry.Changed)
	assert.Equal(t, result.SourceDigest, retry.SourceDigest)
	assert.Equal(t, result.ImportedGrantCount, retry.ImportedGrantCount)
	assert.Equal(t, readsBeforeRetry, reader.reads)
}

func TestAuthorityCutoverRefusesIndirectOpenFGASubject(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(
		NewRepository(postgres.DB),
		WithProjectionClient(newFakeProjectionClient()),
	)
	userID := seedAuthorizationUser(t, postgres, "cutover-indirect-subject-user")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010026)
	reader := &fakeAuthorityCutoverTupleReader{tuples: map[string][]fga.Tuple{
		fmt.Sprintf("school:%d#admin", schoolID): {
			{
				User:     "group:legacy-operators#member",
				Relation: "admin",
				Object:   fmt.Sprintf("school:%d", schoolID),
			},
		},
	}}
	snapshot := LegacyAuthoritySnapshot{
		Organization: "stuhelper",
		Users: []LegacyAuthorityIdentity{
			{ID: "cutover-indirect-subject", Owner: "stuhelper", Name: "cutover-indirect-subject-user"},
		},
		RoleMembers: map[Role][]string{RoleSchoolAdmin: {"cutover-indirect-subject"}},
	}

	_, err := service.ImportLegacyAuthority(ctx, []AuthorityCutoverUser{
		{InternalUserID: userID, ProviderSubject: "cutover-indirect-subject"},
	}, snapshot, reader)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAuthorityCutoverConflict)
	require.ErrorIs(t, service.RequireAuthorityCutoverComplete(ctx), ErrAuthorityCutoverIncomplete)
}

func TestAuthorityCutoverRefusesOpenFGATupleWithoutCasdoorIdentity(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(
		NewRepository(postgres.DB),
		WithProjectionClient(newFakeProjectionClient()),
	)
	userID := seedAuthorizationUser(t, postgres, "cutover-missing-provider-user")
	schoolID := seedAuthorizationSchool(t, postgres, 4111010025)
	reader := &fakeAuthorityCutoverTupleReader{tuples: map[string][]fga.Tuple{
		fmt.Sprintf("school:%d#admin", schoolID): {
			{User: fmt.Sprintf("user:%d", userID), Relation: "admin", Object: fmt.Sprintf("school:%d", schoolID)},
		},
	}}
	snapshot := LegacyAuthoritySnapshot{
		Organization: "stuhelper",
		Users: []LegacyAuthorityIdentity{
			{ID: "different-subject", Owner: "stuhelper", Name: "different-user"},
		},
		RoleMembers: map[Role][]string{RoleSchoolAdmin: {"different-subject"}},
	}

	_, err := service.ImportLegacyAuthority(ctx, []AuthorityCutoverUser{
		{InternalUserID: userID, ProviderSubject: "cutover-missing-provider-user-subject"},
	}, snapshot, reader)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAuthorityCutoverConflict))
	require.ErrorIs(t, service.RequireAuthorityCutoverComplete(ctx), ErrAuthorityCutoverIncomplete)
}

func TestAuthorityCutoverCanSealAnEmptyFreshInstallation(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	service := NewService(
		NewRepository(postgres.DB),
		WithProjectionClient(newFakeProjectionClient()),
	)
	reader := &fakeAuthorityCutoverTupleReader{tuples: map[string][]fga.Tuple{}}

	result, err := service.ImportLegacyAuthority(ctx, nil, LegacyAuthoritySnapshot{
		Organization: "stuhelper",
		RoleMembers:  map[Role][]string{},
	}, reader)

	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.Zero(t, result.ImportedGrantCount)
	assert.Len(t, result.SourceDigest, 64)
	require.NoError(t, service.RequireAuthorityCutoverComplete(ctx))
}
