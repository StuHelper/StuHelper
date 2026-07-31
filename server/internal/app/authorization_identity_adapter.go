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
	users         *user.Repository
	authorization *authorization.Service
}

func newAuthorizationIdentityAdapter(
	users *user.Repository,
	authorizationService *authorization.Service,
) authorizationIdentityAdapter {
	if users == nil {
		panic("newAuthorizationIdentityAdapter: user repository is required")
	}
	if authorizationService == nil {
		panic("newAuthorizationIdentityAdapter: authorization service is required")
	}
	return authorizationIdentityAdapter{
		users:         users,
		authorization: authorizationService,
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
	return snapshot.Roles, snapshot.RoleScopes, nil
}
