package rbac

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeServiceRepo struct {
	onGetRoleByID          func(ctx context.Context, id int64) (*Role, error)
	onGetRolePermissionIDs func(ctx context.Context, roleID int64) ([]int64, error)
	onGetPermissionsByIDs  func(ctx context.Context, ids []int64) ([]Permission, error)
	onSetRolePermissions   func(ctx context.Context, roleID int64, permIDs []int64) error
	onUpdateRole           func(ctx context.Context, role *Role) error
	onGetGroupByID         func(ctx context.Context, id int64) (*UserGroup, error)
	onUpdateGroup          func(ctx context.Context, group *UserGroup) error
}

func (f *fakeServiceRepo) ListRoles(context.Context) ([]Role, error) { return nil, nil }
func (f *fakeServiceRepo) GetRoleByName(context.Context, string) (*Role, error) {
	return nil, ErrRoleNotFound
}
func (f *fakeServiceRepo) CreateRole(context.Context, *Role) error { return nil }
func (f *fakeServiceRepo) GetRoleByID(ctx context.Context, id int64) (*Role, error) {
	if f.onGetRoleByID != nil {
		return f.onGetRoleByID(ctx, id)
	}
	return nil, ErrRoleNotFound
}
func (f *fakeServiceRepo) UpdateRole(ctx context.Context, role *Role) error {
	if f.onUpdateRole != nil {
		return f.onUpdateRole(ctx, role)
	}
	return nil
}
func (f *fakeServiceRepo) DeleteRole(context.Context, int64) error { return nil }
func (f *fakeServiceRepo) GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	if f.onGetRolePermissionIDs != nil {
		return f.onGetRolePermissionIDs(ctx, roleID)
	}
	return nil, nil
}
func (f *fakeServiceRepo) SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64) error {
	if f.onSetRolePermissions != nil {
		return f.onSetRolePermissions(ctx, roleID, permIDs)
	}
	return nil
}
func (f *fakeServiceRepo) ListPermissions(context.Context, string) ([]Permission, error) {
	return nil, nil
}
func (f *fakeServiceRepo) GetPermissionByID(context.Context, int64) (*Permission, error) {
	return nil, ErrPermNotFound
}
func (f *fakeServiceRepo) GetPermissionsByIDs(ctx context.Context, ids []int64) ([]Permission, error) {
	if f.onGetPermissionsByIDs != nil {
		return f.onGetPermissionsByIDs(ctx, ids)
	}
	return nil, nil
}
func (f *fakeServiceRepo) GetUserRoles(context.Context, int64) ([]Role, error) { return nil, nil }
func (f *fakeServiceRepo) SetUserRoles(context.Context, int64, []int64) error  { return nil }
func (f *fakeServiceRepo) GetEffectivePermissions(context.Context, int64) ([]EffectivePermission, error) {
	return nil, nil
}
func (f *fakeServiceRepo) SetUserPermission(context.Context, int64, int64, bool) error { return nil }
func (f *fakeServiceRepo) GetUserRoleNames(context.Context, int64) ([]string, error)   { return nil, nil }
func (f *fakeServiceRepo) ListGroups(context.Context) ([]UserGroup, error)             { return nil, nil }
func (f *fakeServiceRepo) GetGroupByID(ctx context.Context, id int64) (*UserGroup, error) {
	if f.onGetGroupByID != nil {
		return f.onGetGroupByID(ctx, id)
	}
	return nil, ErrGroupNotFound
}
func (f *fakeServiceRepo) CreateGroup(context.Context, *UserGroup) error { return nil }
func (f *fakeServiceRepo) UpdateGroup(ctx context.Context, group *UserGroup) error {
	if f.onUpdateGroup != nil {
		return f.onUpdateGroup(ctx, group)
	}
	return nil
}
func (f *fakeServiceRepo) DeleteGroup(context.Context, int64) error { return nil }
func (f *fakeServiceRepo) GetGroupMembersDetail(context.Context, int64) ([]GroupMember, error) {
	return nil, nil
}
func (f *fakeServiceRepo) SetGroupMembers(context.Context, int64, []int64) error     { return nil }
func (f *fakeServiceRepo) SetGroupPermissions(context.Context, int64, []int64) error { return nil }
func (f *fakeServiceRepo) GetInternalUserID(context.Context, string) (int64, error)  { return 0, nil }

func TestUpdateRole_MergesPartialFields(t *testing.T) {
	oldDesc := "old description"
	existing := &Role{
		ID:          7,
		Name:        "moderator",
		DisplayName: "Moderator",
		Description: &oldDesc,
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	var saved *Role
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			assert.Equal(t, int64(7), id)
			roleCopy := *existing
			return &roleCopy, nil
		},
		onUpdateRole: func(_ context.Context, role *Role) error {
			roleCopy := *role
			saved = &roleCopy
			return nil
		},
	})

	newDesc := "new description"
	role, err := svc.UpdateRole(context.Background(), 7, UpdateRoleInput{
		Description: &newDesc,
	})

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Moderator", saved.DisplayName)
	require.NotNil(t, saved.Description)
	assert.Equal(t, "new description", *saved.Description)
	assert.Equal(t, saved.DisplayName, role.DisplayName)
}

func TestUpdateRole_EmptyInputIsNoOp(t *testing.T) {
	existing := &Role{
		ID:          8,
		Name:        "reviewer",
		DisplayName: "Reviewer",
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	updateCalled := false
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			assert.Equal(t, int64(8), id)
			roleCopy := *existing
			return &roleCopy, nil
		},
		onUpdateRole: func(_ context.Context, role *Role) error {
			updateCalled = true
			return nil
		},
	})

	role, err := svc.UpdateRole(context.Background(), 8, UpdateRoleInput{})

	require.NoError(t, err)
	assert.False(t, updateCalled)
	assert.Equal(t, "Reviewer", role.DisplayName)
}

func TestGetRolePermissionIDs_ChecksRoleExistenceFirst(t *testing.T) {
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			return &Role{ID: id, Name: "editor", DisplayName: "Editor"}, nil
		},
		onGetRolePermissionIDs: func(_ context.Context, roleID int64) ([]int64, error) {
			assert.Equal(t, int64(7), roleID)
			return []int64{1, 3, 5}, nil
		},
	})

	permIDs, err := svc.GetRolePermissionIDs(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, []int64{1, 3, 5}, permIDs)
}

func TestSetRolePermissions_EmptySelectionRequiresClearAllConfirmation(t *testing.T) {
	called := false
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			return &Role{ID: id, Name: "editor", DisplayName: "Editor"}, nil
		},
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64) error {
			called = true
			return nil
		},
	})

	err := svc.SetRolePermissions(context.Background(), 7, nil, false)

	require.ErrorIs(t, err, ErrRolePermissionClearConfirmRequired)
	assert.False(t, called)
}

func TestSetRolePermissions_ClearAllAllowsEmptySelection(t *testing.T) {
	var (
		capturedRoleID  int64
		capturedPermIDs []int64
	)
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			return &Role{ID: id, Name: "editor", DisplayName: "Editor"}, nil
		},
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64) error {
			capturedRoleID = roleID
			capturedPermIDs = append([]int64(nil), permIDs...)
			return nil
		},
	})

	err := svc.SetRolePermissions(context.Background(), 7, []int64{}, true)

	require.NoError(t, err)
	assert.Equal(t, int64(7), capturedRoleID)
	assert.Empty(t, capturedPermIDs)
}

func TestSetRolePermissions_RejectsUnknownPermissionIDs(t *testing.T) {
	called := false
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			return &Role{ID: id, Name: "editor", DisplayName: "Editor"}, nil
		},
		onGetPermissionsByIDs: func(_ context.Context, ids []int64) ([]Permission, error) {
			assert.Equal(t, []int64{1, 2}, ids)
			return []Permission{
				{ID: 1, Name: "perm.one", Module: "rbac", Action: "read", DisplayName: "One"},
			}, nil
		},
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64) error {
			called = true
			return nil
		},
	})

	err := svc.SetRolePermissions(context.Background(), 7, []int64{1, 2}, false)

	require.ErrorIs(t, err, ErrPermissionSelectionInvalid)
	assert.False(t, called)
}

func TestSetRolePermissions_RejectsSystemRole(t *testing.T) {
	called := false
	svc := NewService(&fakeServiceRepo{
		onGetRoleByID: func(_ context.Context, id int64) (*Role, error) {
			return &Role{ID: id, Name: "admin", DisplayName: "Admin", IsSystem: true}, nil
		},
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64) error {
			called = true
			return nil
		},
	})

	err := svc.SetRolePermissions(context.Background(), 1, []int64{1, 2}, false)

	require.ErrorIs(t, err, ErrRoleIsSystem)
	assert.False(t, called)
}

func TestUpdateGroup_MergesPartialFields(t *testing.T) {
	oldDesc := "old description"
	existing := &UserGroup{
		ID:          21,
		Name:        "reviewers",
		DisplayName: "Reviewers",
		Description: &oldDesc,
		MemberCount: 3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	var saved *UserGroup
	svc := NewService(&fakeServiceRepo{
		onGetGroupByID: func(_ context.Context, id int64) (*UserGroup, error) {
			assert.Equal(t, int64(21), id)
			groupCopy := *existing
			return &groupCopy, nil
		},
		onUpdateGroup: func(_ context.Context, group *UserGroup) error {
			groupCopy := *group
			saved = &groupCopy
			return nil
		},
	})

	newName := "Core Reviewers"
	group, err := svc.UpdateGroup(context.Background(), 21, UpdateGroupInput{
		DisplayName: &newName,
	})

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Core Reviewers", saved.DisplayName)
	require.NotNil(t, saved.Description)
	assert.Equal(t, "old description", *saved.Description)
	assert.Equal(t, saved.DisplayName, group.DisplayName)
}

func TestUpdateGroup_EmptyInputIsNoOp(t *testing.T) {
	existing := &UserGroup{
		ID:          22,
		Name:        "ops",
		DisplayName: "Ops",
		MemberCount: 2,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	updateCalled := false
	svc := NewService(&fakeServiceRepo{
		onGetGroupByID: func(_ context.Context, id int64) (*UserGroup, error) {
			assert.Equal(t, int64(22), id)
			groupCopy := *existing
			return &groupCopy, nil
		},
		onUpdateGroup: func(_ context.Context, group *UserGroup) error {
			updateCalled = true
			return nil
		},
	})

	group, err := svc.UpdateGroup(context.Background(), 22, UpdateGroupInput{})

	require.NoError(t, err)
	assert.False(t, updateCalled)
	assert.Equal(t, "Ops", group.DisplayName)
}
