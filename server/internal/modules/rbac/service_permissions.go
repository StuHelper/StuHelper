package rbac

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	appcapability "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// ListPermissions 获取权限列表
func (s *Service) ListPermissions(ctx context.Context, module string) ([]Permission, error) {
	return s.repo.ListPermissions(ctx, module)
}

// GetEffectivePermissions 获取用户最终生效权限
func (s *Service) GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error) {
	return s.repo.GetEffectivePermissions(ctx, userID)
}

// SetUserPermission 设置用户个人权限覆盖
func (s *Service) SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error {
	if _, err := s.repo.GetPermissionByID(ctx, permID); err != nil {
		return err
	}
	return s.repo.SetUserPermission(ctx, userID, permID, granted)
}

// GetUserCapabilities 返回用户在航小伴内的最终生效能力集合。
func (s *Service) GetUserCapabilities(ctx context.Context, externalID string) ([]string, error) {
	userID, err := s.repo.GetInternalUserID(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("GetUserCapabilities resolve user: %w", err)
	}

	effectivePerms, err := s.repo.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserCapabilities load permissions: %w", err)
	}

	names := make([]string, 0, len(effectivePerms))
	for _, perm := range effectivePerms {
		if !perm.Granted || perm.Name == "" {
			continue
		}
		names = append(names, perm.Name)
	}

	return appcapability.Normalize(names), nil
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
		return false, nil
	}

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

	if len(perm.ScopeRoles) > 0 {
		userRoleNames, err := s.repo.GetUserRoleNames(ctx, userID)
		if err != nil {
			logger.L().Warn("CheckPermissionScope: failed to get user roles",
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
