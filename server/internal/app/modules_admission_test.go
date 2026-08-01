package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorizationmodule "github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
)

type fixedAdmissionAccessSnapshotResolver struct {
	snapshot authorizationmodule.AccessSnapshot
	err      error
	userID   int64
}

func (r *fixedAdmissionAccessSnapshotResolver) ResolveAccessSnapshotByUserID(
	_ context.Context,
	userID int64,
) (authorizationmodule.AccessSnapshot, error) {
	r.userID = userID
	return r.snapshot, r.err
}

func TestAdmissionAuthorizationGatewayUsesRevalidatedAccessSnapshot(t *testing.T) {
	schoolID := int64(4111010006)
	resolver := &fixedAdmissionAccessSnapshotResolver{
		snapshot: authorizationmodule.AccessSnapshot{
			InternalUserID: 42,
			Roles:          []string{string(authorizationmodule.RoleSuperAdmin)},
		},
	}
	gateway := admissionAuthorizationGateway{authorization: resolver}

	allowed, err := gateway.UserHasCapabilityInSchool(
		context.Background(),
		42,
		capability.AdmissionFreshmanReview,
		schoolID,
	)

	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(42), resolver.userID)
}

func TestAdmissionAuthorizationGatewayFailsClosedOnRevalidationError(t *testing.T) {
	expectedErr := errors.New("Casdoor unavailable")
	gateway := admissionAuthorizationGateway{
		authorization: &fixedAdmissionAccessSnapshotResolver{err: expectedErr},
	}

	allowed, err := gateway.UserHasCapabilityInSchool(
		context.Background(),
		42,
		capability.AdmissionFreshmanReview,
		4111010006,
	)

	require.ErrorIs(t, err, expectedErr)
	assert.False(t, allowed)
}
