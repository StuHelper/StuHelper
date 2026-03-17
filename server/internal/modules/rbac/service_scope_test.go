package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scopeAwareRepo struct {
	*fakeServiceRepo

	permission       *Permission
	permissionErr    error
	userRoleNames    []string
	userRoleNamesErr error
}

func (r *scopeAwareRepo) GetPermissionByID(context.Context, int64) (*Permission, error) {
	if r.permissionErr != nil {
		return nil, r.permissionErr
	}
	return r.permission, nil
}

func (r *scopeAwareRepo) GetUserRoleNames(context.Context, int64) ([]string, error) {
	if r.userRoleNamesErr != nil {
		return nil, r.userRoleNamesErr
	}
	return r.userRoleNames, nil
}

func TestCheckPermissionScope_DeniesWhenScopeSchoolConfiguredButSchoolContextMissing(t *testing.T) {
	repo := &scopeAwareRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		permission: &Permission{
			ID:             1,
			Name:           "admin.users.read",
			ScopeSchoolIDs: []string{"10006"},
		},
	}
	svc := NewService(repo)

	allowed, err := svc.CheckPermissionScope(context.Background(), EffectivePermission{
		PermissionID: 1,
		Name:         "admin.users.read",
		Granted:      true,
	}, 42, nil)

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckPermissionScope_DeniesWhenScopeSchoolConfiguredButSchoolContextEmpty(t *testing.T) {
	repo := &scopeAwareRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		permission: &Permission{
			ID:             1,
			Name:           "admin.users.read",
			ScopeSchoolIDs: []string{"10006"},
		},
	}
	svc := NewService(repo)

	empty := "   "
	allowed, err := svc.CheckPermissionScope(context.Background(), EffectivePermission{
		PermissionID: 1,
		Name:         "admin.users.read",
		Granted:      true,
	}, 42, &empty)

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckPermissionScope_AllowsWhenScopeSchoolMatches(t *testing.T) {
	repo := &scopeAwareRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		permission: &Permission{
			ID:             1,
			Name:           "admin.users.read",
			ScopeSchoolIDs: []string{"10006"},
		},
	}
	svc := NewService(repo)

	schoolID := "10006"
	allowed, err := svc.CheckPermissionScope(context.Background(), EffectivePermission{
		PermissionID: 1,
		Name:         "admin.users.read",
		Granted:      true,
	}, 42, &schoolID)

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestCheckPermissionScope_DeniesWhenScopeSchoolNotMatch(t *testing.T) {
	repo := &scopeAwareRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		permission: &Permission{
			ID:             1,
			Name:           "admin.users.read",
			ScopeSchoolIDs: []string{"10006"},
		},
	}
	svc := NewService(repo)

	schoolID := "20001"
	allowed, err := svc.CheckPermissionScope(context.Background(), EffectivePermission{
		PermissionID: 1,
		Name:         "admin.users.read",
		Granted:      true,
	}, 42, &schoolID)

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckPermissionScope_ReturnsErrorWhenPermissionLookupFails(t *testing.T) {
	repo := &scopeAwareRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		permissionErr:   errors.New("db unavailable"),
	}
	svc := NewService(repo)

	allowed, err := svc.CheckPermissionScope(context.Background(), EffectivePermission{
		PermissionID: 1,
		Name:         "admin.users.read",
		Granted:      true,
	}, 42, nil)

	assert.False(t, allowed)
	require.Error(t, err)
	assert.ErrorContains(t, err, "get permission detail")
}

func TestCheckPermissionScope_ReturnsErrorWhenUserRoleLookupFails(t *testing.T) {
	repo := &scopeAwareRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		permission: &Permission{
			ID:         1,
			Name:       "admin.users.read",
			ScopeRoles: []string{"admin"},
		},
		userRoleNamesErr: errors.New("db unavailable"),
	}
	svc := NewService(repo)

	allowed, err := svc.CheckPermissionScope(context.Background(), EffectivePermission{
		PermissionID: 1,
		Name:         "admin.users.read",
		Granted:      true,
	}, 42, nil)

	assert.False(t, allowed)
	require.Error(t, err)
	assert.ErrorContains(t, err, "get user roles")
}
