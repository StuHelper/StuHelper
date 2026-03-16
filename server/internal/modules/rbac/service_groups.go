package rbac

import (
	"context"
	"fmt"
)

// ListGroups 获取用户组列表
func (s *Service) ListGroups(ctx context.Context) ([]UserGroup, error) {
	return s.repo.ListGroups(ctx)
}

// CreateGroup 创建用户组
func (s *Service) CreateGroup(ctx context.Context, name, displayName, desc string, createdBy int64) (*UserGroup, error) {
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
