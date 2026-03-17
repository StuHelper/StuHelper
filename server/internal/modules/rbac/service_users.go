package rbac

import "context"

// GetUserRoles 获取用户角色
func (s *Service) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	if validator, ok := s.repo.(userExistenceRepo); ok {
		exists, err := validator.UserExists(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrUserNotFound
		}
	}

	return s.repo.GetUserRoles(ctx, userID)
}

// SetUserRoles 设置用户角色
func (s *Service) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	normalizedRoleIDs, ok := uniquePositiveIDs(roleIDs)
	if !ok {
		return ErrRoleSelectionInvalid
	}

	if validator, ok := s.repo.(userExistenceRepo); ok {
		exists, err := validator.UserExists(ctx, userID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrUserNotFound
		}
	}

	if len(normalizedRoleIDs) > 0 {
		if validator, ok := s.repo.(roleCountRepo); ok {
			count, err := validator.CountRolesByIDs(ctx, normalizedRoleIDs)
			if err != nil {
				return err
			}
			if count != len(normalizedRoleIDs) {
				return ErrRoleSelectionInvalid
			}
		}
	}

	return s.repo.SetUserRoles(ctx, userID, normalizedRoleIDs)
}

// GetInternalUserID 根据 Casdoor external_id 获取内部 user.id
func (s *Service) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	return s.repo.GetInternalUserID(ctx, externalID)
}
