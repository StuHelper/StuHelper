package rbac

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	appcapability "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// ListPermissions 获取权限列表
func (s *Service) ListPermissions(ctx context.Context, module string) ([]Permission, error) {
	return s.repo.ListPermissions(ctx, module)
}

// GetEffectivePermissions 获取用户最终生效权限
func (s *Service) GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error) {
	if validator, ok := s.repo.(userExistenceRepo); ok {
		exists, err := validator.UserExists(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrUserNotFound
		}
	}

	return s.repo.GetEffectivePermissions(ctx, userID)
}

// SetUserPermission 设置用户个人权限覆盖
func (s *Service) SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error {
	if _, err := s.repo.GetPermissionByID(ctx, permID); err != nil {
		return err
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

	return s.repo.SetUserPermission(ctx, userID, permID, granted)
}

// GetUserCapabilities 返回用户在航小伴内的最终生效能力集合。
func (s *Service) GetUserCapabilities(ctx context.Context, externalID string) ([]string, error) {
	snapshot, err := s.GetUserCapabilitySnapshot(ctx, externalID)
	if err != nil {
		return nil, err
	}
	return snapshot.Capabilities, nil
}

func (s *Service) GetUserCapabilitySnapshot(ctx context.Context, externalID string) (appcapability.UserAccessSnapshot, error) {
	userID, err := s.repo.GetInternalUserID(ctx, externalID)
	if err != nil {
		return appcapability.UserAccessSnapshot{}, fmt.Errorf("GetUserCapabilitySnapshot resolve user: %w", err)
	}

	effectivePerms, err := s.repo.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return appcapability.UserAccessSnapshot{}, fmt.Errorf("GetUserCapabilitySnapshot load permissions: %w", err)
	}

	permissionIDs := make([]int64, 0, len(effectivePerms))
	for _, perm := range effectivePerms {
		if !perm.Granted || perm.Name == "" {
			continue
		}
		permissionIDs = append(permissionIDs, perm.PermissionID)
	}
	if len(permissionIDs) == 0 {
		return appcapability.UserAccessSnapshot{
			Capabilities:       []string{},
			GlobalCapabilities: []string{},
			CapabilityGrants:   []appcapability.Grant{},
		}, nil
	}

	permissions, err := s.repo.GetPermissionsByIDs(ctx, permissionIDs)
	if err != nil {
		return appcapability.UserAccessSnapshot{}, fmt.Errorf("GetUserCapabilitySnapshot load permission detail: %w", err)
	}

	permissionsByID := make(map[int64]Permission, len(permissions))
	for _, permission := range permissions {
		permissionsByID[permission.ID] = permission
	}

	grants := make([]appcapability.Grant, 0, len(effectivePerms))
	for _, effectivePerm := range effectivePerms {
		if !effectivePerm.Granted || effectivePerm.Name == "" {
			continue
		}

		permission, ok := permissionsByID[effectivePerm.PermissionID]
		if !ok {
			return appcapability.UserAccessSnapshot{}, fmt.Errorf("GetUserCapabilitySnapshot missing permission detail for %d", effectivePerm.PermissionID)
		}

		grants = append(grants, appcapability.Grant{
			Name:           effectivePerm.Name,
			ScopeSchoolIDs: append([]string(nil), permission.ScopeSchoolIDs...),
			ScopeRoles:     append([]string(nil), permission.ScopeRoles...),
		})
	}

	return appcapability.BuildUserAccessSnapshot(grants), nil
}

// CheckPermission 核心授权检查
// 检查用户是否拥有指定权限名，并验证 scope 限制（scope_school_ids, scope_roles）。
// 注意：此方法每次调用都会从数据库加载 effective permissions。如果调用方已经
// 持有缓存的 effective permissions（例如中间件链），应改用
// CheckPermissionScope 以避免重复查询。
func (s *Service) CheckPermission(ctx context.Context, userID int64, permName string, schoolID *string) (bool, error) {
	effectivePerms, err := s.repo.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("CheckPermission get effective: %w", err)
	}

	var found *EffectivePermission
	for i := range effectivePerms {
		if effectivePerms[i].Name == permName {
			found = &effectivePerms[i]
			break
		}
	}
	if found == nil || !found.Granted {
		return false, nil
	}

	return s.CheckPermissionScope(ctx, *found, userID, schoolID)
}

// CheckPermissionScope 验证已匹配的 effective permission 的 scope 约束
// （scope_school_ids、scope_roles）。当调用方已持有缓存的 effective
// permissions 并完成了名称匹配时，使用此方法可避免重新加载 effective
// permissions。
func (s *Service) CheckPermissionScope(ctx context.Context, ep EffectivePermission, userID int64, schoolID *string) (bool, error) {
	if !ep.Granted {
		return false, nil
	}

	perm, err := s.repo.GetPermissionByID(ctx, ep.PermissionID)
	if err != nil {
		logger.L().Warn("CheckPermissionScope: failed to get permission detail",
			zap.Int64("permission_id", ep.PermissionID),
			zap.Error(err),
		)
		return false, fmt.Errorf("CheckPermissionScope get permission detail: %w", err)
	}

	if !isSchoolScopeAllowed(perm.ScopeSchoolIDs, schoolID) {
		return false, nil
	}

	if len(perm.ScopeRoles) > 0 {
		userRoleNames, err := s.repo.GetUserRoleNames(ctx, userID)
		if err != nil {
			logger.L().Warn("CheckPermissionScope: failed to get user roles",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			return false, fmt.Errorf("CheckPermissionScope get user roles: %w", err)
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

func isSchoolScopeAllowed(scopeSchoolIDs []string, schoolID *string) bool {
	if len(scopeSchoolIDs) == 0 {
		return true
	}
	if schoolID == nil {
		return false
	}
	requestSchoolID := strings.TrimSpace(*schoolID)
	if requestSchoolID == "" {
		return false
	}
	for _, allowedSchoolID := range scopeSchoolIDs {
		if requestSchoolID == allowedSchoolID {
			return true
		}
	}
	return false
}
