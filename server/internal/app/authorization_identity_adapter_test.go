package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorizationmodule "github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type fixedOrganizationAdminVerifier struct {
	organizationAdmin bool
	err               error
	calls             int
}

func (v *fixedOrganizationAdminVerifier) ValidateOIDCSubject(
	_ context.Context,
	_ string,
) (bool, error) {
	v.calls++
	return v.organizationAdmin, v.err
}

func TestAuthorizationIdentityAdapterRevokesDemotedCasdoorAdminBeforeReturningSnapshot(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	userID := seedAuthorizationIdentityUser(t, postgres, "casdoor-admin-subject")
	authorizationService := authorizationmodule.NewService(
		authorizationmodule.NewRepository(postgres.DB),
	)
	_, err := authorizationService.SyncCasdoorOrganizationAdmin(
		ctx,
		authorizationmodule.CasdoorOrganizationAdminSyncInput{
			SubjectUserID:     userID,
			OrganizationAdmin: true,
		},
	)
	require.NoError(t, err)
	_, err = postgres.Pool.Exec(ctx, `
		UPDATE authorization_grants
		SET projection_status = 'applied', activated_at = NOW(), projected_at = NOW()
		WHERE subject_user_id = $1 AND role = 'super_admin'
	`, userID)
	require.NoError(t, err)

	verifier := &fixedOrganizationAdminVerifier{organizationAdmin: false}
	adapter := newAuthorizationIdentityAdapter(
		user.NewRepository(postgres.DB, []byte("0123456789abcdef0123456789abcdef")),
		authorizationService,
		verifier,
	)

	roles, _, err := adapter.ResolveAccessSnapshot(ctx, "casdoor-admin-subject")

	require.NoError(t, err)
	assert.NotContains(t, roles, string(authorizationmodule.RoleSuperAdmin))
	assert.Equal(t, 1, verifier.calls)
	var desiredState string
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT desired_state
		FROM authorization_grants
		WHERE subject_user_id = $1 AND role = 'super_admin'
	`, userID).Scan(&desiredState))
	assert.Equal(t, string(authorizationmodule.DesiredRevoked), desiredState)
}

func TestAuthorizationIdentityAdapterFailsClosedWhenCasdoorAdminVerificationFails(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	userID := seedAuthorizationIdentityUser(t, postgres, "casdoor-admin-unavailable")
	authorizationService := authorizationmodule.NewService(
		authorizationmodule.NewRepository(postgres.DB),
	)
	_, err := authorizationService.SyncCasdoorOrganizationAdmin(
		ctx,
		authorizationmodule.CasdoorOrganizationAdminSyncInput{
			SubjectUserID:     userID,
			OrganizationAdmin: true,
		},
	)
	require.NoError(t, err)
	_, err = postgres.Pool.Exec(ctx, `
		UPDATE authorization_grants
		SET projection_status = 'applied', activated_at = NOW(), projected_at = NOW()
		WHERE subject_user_id = $1 AND role = 'super_admin'
	`, userID)
	require.NoError(t, err)

	expectedErr := errors.New("Casdoor unavailable")
	adapter := newAuthorizationIdentityAdapter(
		user.NewRepository(postgres.DB, []byte("0123456789abcdef0123456789abcdef")),
		authorizationService,
		&fixedOrganizationAdminVerifier{err: expectedErr},
	)

	roles, scopes, err := adapter.ResolveAccessSnapshot(ctx, "casdoor-admin-unavailable")

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, roles)
	assert.Nil(t, scopes)
}

func seedAuthorizationIdentityUser(
	t *testing.T,
	postgres *postgresfixture.Fixture,
	subject string,
) int64 {
	t.Helper()
	var userID int64
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(subject)))
	err := postgres.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email, user_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, subject, subject, subject+"@example.com", hash).Scan(&userID)
	require.NoError(t, err)
	return userID
}
