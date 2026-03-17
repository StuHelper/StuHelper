package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type writeValidationServiceRepo struct {
	*fakeServiceRepo

	onUserExists        func(ctx context.Context, userID int64) (bool, error)
	onCountRolesByIDs   func(ctx context.Context, roleIDs []int64) (int, error)
	onCountUsersByIDs   func(ctx context.Context, userIDs []int64) (int, error)
	onCountPermsByIDs   func(ctx context.Context, permIDs []int64) (int, error)
	onSetUserRoles      func(ctx context.Context, userID int64, roleIDs []int64) error
	onSetGroupMembers   func(ctx context.Context, groupID int64, userIDs []int64) error
	onSetGroupPerms     func(ctx context.Context, groupID int64, permIDs []int64) error
	onSetUserPermission func(ctx context.Context, userID int64, permID int64, granted bool) error
	onGetPermissionByID func(ctx context.Context, id int64) (*Permission, error)
}

func (r *writeValidationServiceRepo) UserExists(ctx context.Context, userID int64) (bool, error) {
	if r.onUserExists != nil {
		return r.onUserExists(ctx, userID)
	}
	return true, nil
}

func (r *writeValidationServiceRepo) CountRolesByIDs(ctx context.Context, roleIDs []int64) (int, error) {
	if r.onCountRolesByIDs != nil {
		return r.onCountRolesByIDs(ctx, roleIDs)
	}
	return len(roleIDs), nil
}

func (r *writeValidationServiceRepo) CountUsersByIDs(ctx context.Context, userIDs []int64) (int, error) {
	if r.onCountUsersByIDs != nil {
		return r.onCountUsersByIDs(ctx, userIDs)
	}
	return len(userIDs), nil
}

func (r *writeValidationServiceRepo) CountPermissionsByIDs(ctx context.Context, permIDs []int64) (int, error) {
	if r.onCountPermsByIDs != nil {
		return r.onCountPermsByIDs(ctx, permIDs)
	}
	return len(permIDs), nil
}

func (r *writeValidationServiceRepo) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	if r.onSetUserRoles != nil {
		return r.onSetUserRoles(ctx, userID, roleIDs)
	}
	return nil
}

func (r *writeValidationServiceRepo) SetGroupMembers(ctx context.Context, groupID int64, userIDs []int64) error {
	if r.onSetGroupMembers != nil {
		return r.onSetGroupMembers(ctx, groupID, userIDs)
	}
	return nil
}

func (r *writeValidationServiceRepo) SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error {
	if r.onSetGroupPerms != nil {
		return r.onSetGroupPerms(ctx, groupID, permIDs)
	}
	return nil
}

func (r *writeValidationServiceRepo) SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error {
	if r.onSetUserPermission != nil {
		return r.onSetUserPermission(ctx, userID, permID, granted)
	}
	return nil
}

func (r *writeValidationServiceRepo) GetPermissionByID(ctx context.Context, id int64) (*Permission, error) {
	if r.onGetPermissionByID != nil {
		return r.onGetPermissionByID(ctx, id)
	}
	return &Permission{ID: id, Name: "perm.test"}, nil
}

func TestSetUserRoles_ReturnsUserNotFoundWhenTargetUserMissing(t *testing.T) {
	repo := &writeValidationServiceRepo{
		fakeServiceRepo: &fakeServiceRepo{},
		onUserExists: func(_ context.Context, userID int64) (bool, error) {
			assert.Equal(t, int64(42), userID)
			return false, nil
		},
	}
	svc := NewService(repo)

	err := svc.SetUserRoles(context.Background(), 42, []int64{1})
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestSetUserRoles_ReturnsRoleSelectionInvalidForNonPositiveIDs(t *testing.T) {
	repo := &writeValidationServiceRepo{fakeServiceRepo: &fakeServiceRepo{}}
	svc := NewService(repo)

	err := svc.SetUserRoles(context.Background(), 42, []int64{1, 0})
	require.ErrorIs(t, err, ErrRoleSelectionInvalid)
}

func TestSetGroupMembers_ReturnsUserSelectionInvalidWhenUserListIncomplete(t *testing.T) {
	repo := &writeValidationServiceRepo{
		fakeServiceRepo: &fakeServiceRepo{
			onGetGroupByID: func(_ context.Context, id int64) (*UserGroup, error) {
				return &UserGroup{ID: id, Name: "reviewers"}, nil
			},
		},
		onCountUsersByIDs: func(_ context.Context, userIDs []int64) (int, error) {
			assert.Equal(t, []int64{10, 11}, userIDs)
			return 1, nil
		},
	}
	svc := NewService(repo)

	err := svc.SetGroupMembers(context.Background(), 3, []int64{10, 11})
	require.ErrorIs(t, err, ErrUserSelectionInvalid)
}

func TestSetGroupPermissions_NormalizesDuplicateIDsBeforeWrite(t *testing.T) {
	var captured []int64
	repo := &writeValidationServiceRepo{
		fakeServiceRepo: &fakeServiceRepo{
			onGetGroupByID: func(_ context.Context, id int64) (*UserGroup, error) {
				return &UserGroup{ID: id, Name: "reviewers"}, nil
			},
		},
		onCountPermsByIDs: func(_ context.Context, permIDs []int64) (int, error) {
			assert.Equal(t, []int64{5, 7}, permIDs)
			return 2, nil
		},
		onSetGroupPerms: func(_ context.Context, groupID int64, permIDs []int64) error {
			assert.Equal(t, int64(3), groupID)
			captured = append([]int64(nil), permIDs...)
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.SetGroupPermissions(context.Background(), 3, []int64{5, 7, 5})
	require.NoError(t, err)
	assert.Equal(t, []int64{5, 7}, captured)
}
