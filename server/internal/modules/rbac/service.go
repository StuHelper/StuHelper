package rbac

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// 业务错误定义
var (
	ErrRoleNotFound   = errors.New("role not found")
	ErrRoleNameTaken  = errors.New("role name already taken")
	ErrRoleIsSystem   = errors.New("system role cannot be modified")
	ErrGroupNotFound  = errors.New("group not found")
	ErrGroupNameTaken = errors.New("group name already taken")
	ErrPermNotFound   = errors.New("permission not found")
)

// Repo defines the repository methods required by the RBAC service.
type Repo interface {
	ListRoles(ctx context.Context) ([]Role, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	CreateRole(ctx context.Context, role *Role) error
	GetRoleByID(ctx context.Context, id int64) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id int64) error
	SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64) error

	ListPermissions(ctx context.Context, module string) ([]Permission, error)
	GetPermissionByID(ctx context.Context, id int64) (*Permission, error)

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
// Role CRUD
// ---------------------------------------------------------------------------

// ListRoles 获取角色列表
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreateRole 创建角色（名称唯一校验）
func (s *Service) CreateRole(ctx context.Context, name, displayName, description string) (*Role, error) {
	// 检查名称是否已存在
	existing, err := s.repo.GetRoleByName(ctx, name)
	if err != nil && !errors.Is(err, ErrRoleNotFound) {
		return nil, fmt.Errorf("CreateRole check: %w", err)
	}
	if existing != nil {
		return nil, ErrRoleNameTaken
	}

	role := &Role{
		Name:        name,
		DisplayName: displayName,
		Description: strPtr(description),
		IsSystem:    false,
	}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole 更新角色（系统角色不可修改）
// 使用 merge-update 语义：仅覆盖请求中显式提供的字段。
func (s *Service) UpdateRole(ctx context.Context, id int64, input UpdateRoleInput) (*Role, error) {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role.IsSystem {
		return nil, ErrRoleIsSystem
	}

	changed := false
	if input.DisplayName != nil {
		role.DisplayName = *input.DisplayName
		changed = true
	}
	if input.Description != nil {
		role.Description = strPtr(*input.Description)
		changed = true
	}
	if !changed {
		return role, nil
	}

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// DeleteRole 删除角色（系统角色不可删除）
func (s *Service) DeleteRole(ctx context.Context, id int64) error {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return ErrRoleIsSystem
	}
	return s.repo.DeleteRole(ctx, id)
}

// SetRolePermissions 设置角色权限
func (s *Service) SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64) error {
	// 校验角色存在
	if _, err := s.repo.GetRoleByID(ctx, roleID); err != nil {
		return err
	}
	return s.repo.SetRolePermissions(ctx, roleID, permIDs)
}

// ---------------------------------------------------------------------------
// Permission listing
// ---------------------------------------------------------------------------

// ListPermissions 获取权限列表
func (s *Service) ListPermissions(ctx context.Context, module string) ([]Permission, error) {
	return s.repo.ListPermissions(ctx, module)
}

// ---------------------------------------------------------------------------
// User role management
// ---------------------------------------------------------------------------

// GetUserRoles 获取用户角色
func (s *Service) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	return s.repo.GetUserRoles(ctx, userID)
}

// SetUserRoles 设置用户角色
func (s *Service) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	return s.repo.SetUserRoles(ctx, userID, roleIDs)
}

// ---------------------------------------------------------------------------
// Group CRUD
// ---------------------------------------------------------------------------

// ListGroups 获取用户组列表
func (s *Service) ListGroups(ctx context.Context) ([]UserGroup, error) {
	return s.repo.ListGroups(ctx)
}

// CreateGroup 创建用户组
func (s *Service) CreateGroup(ctx context.Context, name, displayName, desc string, createdBy int64) (*UserGroup, error) {
	// 检查名称唯一（复用 GetGroupByID 不行，需要按名称查）
	groups, err := s.repo.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("CreateGroup list: %w", err)
	}
	for _, g := range groups {
		if g.Name == name {
			return nil, ErrGroupNameTaken
		}
	}

	group := &UserGroup{
		Name:        name,
		DisplayName: displayName,
		Description: strPtr(desc),
		CreatedBy:   &createdBy,
	}
	if err := s.repo.CreateGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateGroup 更新用户组
// 使用 merge-update 语义：仅覆盖请求中显式提供的字段。
func (s *Service) UpdateGroup(ctx context.Context, id int64, input UpdateGroupInput) (*UserGroup, error) {
	group, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return nil, err
	}

	changed := false
	if input.DisplayName != nil {
		group.DisplayName = *input.DisplayName
		changed = true
	}
	if input.Description != nil {
		group.Description = strPtr(*input.Description)
		changed = true
	}
	if !changed {
		return group, nil
	}

	if err := s.repo.UpdateGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

// DeleteGroup 删除用户组
func (s *Service) DeleteGroup(ctx context.Context, id int64) error {
	if _, err := s.repo.GetGroupByID(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteGroup(ctx, id)
}

// GetGroupMembers 获取用户组成员
func (s *Service) GetGroupMembers(ctx context.Context, groupID int64) ([]GroupMember, error) {
	return s.repo.GetGroupMembersDetail(ctx, groupID)
}

// SetGroupMembers 设置用户组成员
func (s *Service) SetGroupMembers(ctx context.Context, groupID int64, userIDs []int64) error {
	if _, err := s.repo.GetGroupByID(ctx, groupID); err != nil {
		return err
	}
	return s.repo.SetGroupMembers(ctx, groupID, userIDs)
}

// SetGroupPermissions 设置用户组权限
func (s *Service) SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error {
	if _, err := s.repo.GetGroupByID(ctx, groupID); err != nil {
		return err
	}
	return s.repo.SetGroupPermissions(ctx, groupID, permIDs)
}

// ---------------------------------------------------------------------------
// User permissions
// ---------------------------------------------------------------------------

// GetEffectivePermissions 获取用户最终生效权限
func (s *Service) GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error) {
	return s.repo.GetEffectivePermissions(ctx, userID)
}

// SetUserPermission 设置用户个人权限覆盖
func (s *Service) SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error {
	// 校验权限存在
	if _, err := s.repo.GetPermissionByID(ctx, permID); err != nil {
		return err
	}
	return s.repo.SetUserPermission(ctx, userID, permID, granted)
}

// CheckPermission 核心授权检查
// 检查用户是否拥有指定权限名，并验证 scope 限制（scope_school_ids, scope_roles）
func (s *Service) CheckPermission(ctx context.Context, userID int64, permName string, schoolID *string) (bool, error) {
	// 获取用户最终生效权限列表
	effectivePerms, err := s.repo.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("CheckPermission get effective: %w", err)
	}

	// 查找目标权限
	var found *EffectivePermission
	for i := range effectivePerms {
		if effectivePerms[i].Name == permName {
			found = &effectivePerms[i]
			break
		}
	}
	if found == nil {
		return false, nil
	}
	if !found.Granted {
		return false, nil
	}

	// 获取完整权限记录以检查 scope 限制
	perm, err := s.repo.GetPermissionByID(ctx, found.PermissionID)
	if err != nil {
		logger.L().Warn("CheckPermission: failed to get permission detail",
			zap.Int64("permission_id", found.PermissionID),
			zap.Error(err),
		)
		// 权限记录异常，fail-closed
		return false, nil
	}

	// 检查 school scope 限制
	if len(perm.ScopeSchoolIDs) > 0 && schoolID != nil {
		schoolAllowed := false
		for _, sid := range perm.ScopeSchoolIDs {
			if sid == *schoolID {
				schoolAllowed = true
				break
			}
		}
		if !schoolAllowed {
			return false, nil
		}
	}

	// 检查 role scope 限制
	if len(perm.ScopeRoles) > 0 {
		userRoleNames, err := s.repo.GetUserRoleNames(ctx, userID)
		if err != nil {
			logger.L().Warn("CheckPermission: failed to get user roles",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			return false, nil
		}
		roleAllowed := false
		for _, requiredRole := range perm.ScopeRoles {
			for _, userRole := range userRoleNames {
				if userRole == requiredRole {
					roleAllowed = true
					break
				}
			}
			if roleAllowed {
				break
			}
		}
		if !roleAllowed {
			return false, nil
		}
	}

	return true, nil
}

// GetInternalUserID 根据 Casdoor external_id 获取内部 user.id
func (s *Service) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	return s.repo.GetInternalUserID(ctx, externalID)
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
