package rbac

import (
	"context"
	"errors"
)

// 业务错误定义
var (
	ErrRoleNotFound                       = errors.New("role not found")
	ErrRoleNameTaken                      = errors.New("role name already taken")
	ErrRoleIsSystem                       = errors.New("system role cannot be modified")
	ErrGroupNotFound                      = errors.New("group not found")
	ErrGroupNameTaken                     = errors.New("group name already taken")
	ErrPermNotFound                       = errors.New("permission not found")
	ErrPermissionSelectionInvalid         = errors.New("one or more selected permissions are invalid")
	ErrRolePermissionClearConfirmRequired = errors.New("clearing all role permissions requires explicit confirmation")
)

// Repo defines the repository methods required by the RBAC service.
type Repo interface {
	ListRoles(ctx context.Context) ([]Role, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	CreateRole(ctx context.Context, role *Role) error
	GetRoleByID(ctx context.Context, id int64) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id int64) error
	GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error)
	SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64) error

	ListPermissions(ctx context.Context, module string) ([]Permission, error)
	GetPermissionByID(ctx context.Context, id int64) (*Permission, error)
	GetPermissionsByIDs(ctx context.Context, ids []int64) ([]Permission, error)

	GetUserRoles(ctx context.Context, userID int64) ([]Role, error)
	SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error
	GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error)
	SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error
	GetUserRoleNames(ctx context.Context, userID int64) ([]string, error)

	ListGroups(ctx context.Context) ([]UserGroup, error)
	GetGroupByID(ctx context.Context, id int64) (*UserGroup, error)
	CreateGroup(ctx context.Context, group *UserGroup) error
	UpdateGroup(ctx context.Context, group *UserGroup) error
	DeleteGroup(ctx context.Context, id int64) error
	GetGroupMembersDetail(ctx context.Context, groupID int64) ([]GroupMember, error)
	SetGroupMembers(ctx context.Context, groupID int64, userIDs []int64) error
	SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error

	GetInternalUserID(ctx context.Context, externalID string) (int64, error)
}

type UpdateRoleInput struct {
	DisplayName *string
	Description *string
}

type UpdateGroupInput struct {
	DisplayName *string
	Description *string
}

// Service RBAC 服务层
type Service struct {
	repo Repo
}

// NewService 创建 RBAC 服务
func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// strPtr 返回字符串指针，空字符串返回 nil
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
