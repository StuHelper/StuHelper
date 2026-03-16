package rbac

import (
	"context"
	"errors"
	"fmt"
)

// ListRoles 获取角色列表
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreateRole 创建角色（名称唯一校验）
func (s *Service) CreateRole(ctx context.Context, name, displayName, description string) (*Role, error) {
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

// GetRolePermissionIDs 获取角色已分配权限 ID 列表
func (s *Service) GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	if _, err := s.repo.GetRoleByID(ctx, roleID); err != nil {
		return nil, err
	}
	return s.repo.GetRolePermissionIDs(ctx, roleID)
}

// SetRolePermissions 设置角色权限
func (s *Service) SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64, clearAll bool) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return ErrRoleIsSystem
	}

	uniquePermIDs := make([]int64, 0, len(permIDs))
	seen := make(map[int64]struct{}, len(permIDs))
	for _, permID := range permIDs {
		if permID <= 0 {
			return ErrPermissionSelectionInvalid
		}
		if _, ok := seen[permID]; ok {
			continue
		}
		seen[permID] = struct{}{}
		uniquePermIDs = append(uniquePermIDs, permID)
	}

	if len(uniquePermIDs) == 0 {
		if !clearAll {
			return ErrRolePermissionClearConfirmRequired
		}
		return s.repo.SetRolePermissions(ctx, roleID, nil)
	}

	perms, err := s.repo.GetPermissionsByIDs(ctx, uniquePermIDs)
	if err != nil {
		return fmt.Errorf("SetRolePermissions validate permissions: %w", err)
	}
	if len(perms) != len(uniquePermIDs) {
		return ErrPermissionSelectionInvalid
	}

	return s.repo.SetRolePermissions(ctx, roleID, uniquePermIDs)
}
