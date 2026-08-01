package app

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

// authorizationIdentityAdapter is the only bridge from an authenticated
// provider subject to StuHelper's internal users.id for the authorization
// control plane. Business authorization code receives only the internal ID.
type authorizationIdentityAdapter struct {
	users                     *user.Repository
	authorization             *authorization.Service
	organizationAdminVerifier organizationAdminVerifier
}

type organizationAdminVerifier interface {
	ValidateOIDCSubject(ctx context.Context, subject string) (organizationAdmin bool, err error)
}

func newAuthorizationIdentityAdapter(
	users *user.Repository,
	authorizationService *authorization.Service,
	organizationAdminVerifier organizationAdminVerifier,
) authorizationIdentityAdapter {
	if users == nil {
		panic("newAuthorizationIdentityAdapter: user repository is required")
	}
	if authorizationService == nil {
		panic("newAuthorizationIdentityAdapter: authorization service is required")
	}
	return authorizationIdentityAdapter{
		users:                     users,
		authorization:             authorizationService,
		organizationAdminVerifier: organizationAdminVerifier,
	}
}

func (a authorizationIdentityAdapter) ResolveInternalUserID(
	ctx context.Context,
	providerSubject string,
) (int64, error) {
	userID, err := a.users.GetInternalUserID(ctx, providerSubject)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, authorization.ErrActorUserNotFound
	}
	return userID, err
}

func (a authorizationIdentityAdapter) ResolveAccessSnapshot(
	ctx context.Context,
	providerSubject string,
) ([]string, map[string][]string, error) {
	userID, err := a.ResolveInternalUserID(ctx, providerSubject)
	if err != nil {
		return nil, nil, err
	}
	snapshot, err := a.authorization.ResolveAccessSnapshotByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if !containsAuthorizationRole(snapshot.Roles, string(authorization.RoleSuperAdmin)) ||
		a.organizationAdminVerifier == nil {
		return snapshot.Roles, snapshot.RoleScopes, nil
	}
	organizationAdmin, err := a.organizationAdminVerifier.ValidateOIDCSubject(ctx, providerSubject)
	if err != nil {
		return nil, nil, err
	}
	if organizationAdmin {
		return snapshot.Roles, snapshot.RoleScopes, nil
	}
	if _, err := a.authorization.SyncCasdoorOrganizationAdmin(ctx, authorization.CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: false,
	}); err != nil {
		return nil, nil, err
	}
	snapshot, err = a.authorization.ResolveAccessSnapshotByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return snapshot.Roles, snapshot.RoleScopes, nil
}

func (a authorizationIdentityAdapter) SyncCasdoorOrganizationAdmin(
	ctx context.Context,
	providerSubject string,
	organizationAdmin bool,
) error {
	userID, err := a.ResolveInternalUserID(ctx, providerSubject)
	if err != nil {
		return err
	}
	_, err = a.authorization.SyncCasdoorOrganizationAdmin(ctx, authorization.CasdoorOrganizationAdminSyncInput{
		SubjectUserID:     userID,
		OrganizationAdmin: organizationAdmin,
	})
	return err
}

func containsAuthorizationRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}
