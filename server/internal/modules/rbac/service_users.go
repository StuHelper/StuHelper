package rbac

import "context"

// GetUserRoles 获取用户角色
func (s *Service) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	return s.repo.GetUserRoles(ctx, userID)
}

// SetUserRoles 设置用户角色
func (s *Service) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	return s.repo.SetUserRoles(ctx, userID, roleIDs)
}

// GetInternalUserID 根据 Casdoor external_id 获取内部 user.id
func (s *Service) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	return s.repo.GetInternalUserID(ctx, externalID)
}
