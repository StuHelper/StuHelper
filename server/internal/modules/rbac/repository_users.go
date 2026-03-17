package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// GetUserRoles 获取用户拥有的角色列表
func (r *Repository) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ro.id, ro.name, ro.display_name, ro.description, ro.is_system, ro.created_at, ro.updated_at
		FROM roles ro
		JOIN user_roles ur ON ur.role_id = ro.id
		WHERE ur.user_id = $1
		ORDER BY ro.id ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserRoles: %w", err)
	}
	defer rows.Close()

	roles := make([]Role, 0, 4)
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("GetUserRoles scan: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// SetUserRoles 设置用户的角色列表（事务内 DELETE + INSERT）
func (r *Repository) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
			return mapSetUserRolesWriteError(err)
		}
		for _, rid := range roleIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
			`, userID, rid); err != nil {
				return mapSetUserRolesWriteError(err)
			}
		}
		return nil
	})
}

// UserExists 检查用户是否存在
func (r *Repository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("UserExists: %w", err)
	}
	return exists, nil
}

// CountRolesByIDs 返回 roleIDs 中存在的角色数量（去重后）
func (r *Repository) CountRolesByIDs(ctx context.Context, roleIDs []int64) (int, error) {
	if len(roleIDs) == 0 {
		return 0, nil
	}

	var count int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::INT
		FROM (
			SELECT DISTINCT id
			FROM roles
			WHERE id = ANY($1)
		) AS matched_roles
	`, roleIDs).Scan(&count); err != nil {
		return 0, fmt.Errorf("CountRolesByIDs: %w", err)
	}
	return count, nil
}

// HasRole 检查用户是否拥有指定角色
func (r *Repository) HasRole(ctx context.Context, userID int64, roleName string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur
			JOIN roles ro ON ro.id = ur.role_id
			WHERE ur.user_id = $1 AND ro.name = $2
		)
	`, userID, roleName).Scan(&exists)
	return exists, err
}

// GetUserPermissionOverrides 获取用户的个人权限覆盖列表
func (r *Repository) GetUserPermissionOverrides(ctx context.Context, userID int64) ([]UserPermissionOverride, error) {
	rows, err := r.db.Query(ctx, `
		SELECT up.user_id, up.permission_id, p.name, up.granted
		FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id
		WHERE up.user_id = $1
		ORDER BY p.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserPermissionOverrides: %w", err)
	}
	defer rows.Close()

	overrides := make([]UserPermissionOverride, 0, 8)
	for rows.Next() {
		var o UserPermissionOverride
		if err := rows.Scan(&o.UserID, &o.PermissionID, &o.PermissionName, &o.Granted); err != nil {
			return nil, fmt.Errorf("GetUserPermissionOverrides scan: %w", err)
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

// SetUserPermission UPSERT 用户个人权限覆盖
func (r *Repository) SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_permissions (user_id, permission_id, granted)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, permission_id) DO UPDATE SET granted = EXCLUDED.granted
	`, userID, permID, granted)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			switch pgErr.ConstraintName {
			case "user_permissions_user_id_fkey":
				return ErrUserNotFound
			case "user_permissions_permission_id_fkey":
				return ErrPermNotFound
			}
		}
		return fmt.Errorf("SetUserPermission: %w", err)
	}
	return nil
}

// GetEffectivePermissions 获取用户最终生效权限
// UNION query: 角色权限 + 用户组权限 + 个人覆盖，个人覆盖优先
func (r *Repository) GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error) {
	rows, err := r.db.Query(ctx, `
		WITH role_perms AS (
			SELECT p.id AS permission_id, p.name, p.module, p.action,
			       'role' AS source, TRUE AS granted
			FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN user_roles ur ON ur.role_id = rp.role_id
			WHERE ur.user_id = $1
		),
		group_perms AS (
			SELECT p.id AS permission_id, p.name, p.module, p.action,
			       'group' AS source, TRUE AS granted
			FROM permissions p
			JOIN user_group_permissions gp ON gp.permission_id = p.id
			JOIN user_group_members gm ON gm.group_id = gp.group_id
			WHERE gm.user_id = $1
		),
		overrides AS (
			SELECT p.id AS permission_id, p.name, p.module, p.action,
			       'override' AS source, up.granted
			FROM permissions p
			JOIN user_permissions up ON up.permission_id = p.id
			WHERE up.user_id = $1
		),
		combined AS (
			SELECT permission_id, name, module, action, source, granted,
			       ROW_NUMBER() OVER (
			           PARTITION BY permission_id
			           ORDER BY CASE source
			               WHEN 'override' THEN 1
			               WHEN 'group' THEN 2
			               WHEN 'role' THEN 3
			           END
			       ) AS rn
			FROM (
				SELECT * FROM role_perms
				UNION ALL
				SELECT * FROM group_perms
				UNION ALL
				SELECT * FROM overrides
			) all_perms
		)
		SELECT permission_id, name, module, action, source, granted
		FROM combined
		WHERE rn = 1
		ORDER BY module, action
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("GetEffectivePermissions: %w", err)
	}
	defer rows.Close()

	perms := make([]EffectivePermission, 0, 32)
	for rows.Next() {
		var ep EffectivePermission
		if err := rows.Scan(
			&ep.PermissionID, &ep.Name, &ep.Module, &ep.Action,
			&ep.Source, &ep.Granted,
		); err != nil {
			return nil, fmt.Errorf("GetEffectivePermissions scan: %w", err)
		}
		perms = append(perms, ep)
	}
	return perms, rows.Err()
}

// GetInternalUserID 根据 Casdoor external_id 获取内部 user.id
func (r *Repository) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE external_id = $1`, externalID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("user not found for external_id %s", externalID)
		}
		return 0, fmt.Errorf("GetInternalUserID: %w", err)
	}
	return id, nil
}

func mapSetUserRolesWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			switch pgErr.ConstraintName {
			case "user_roles_user_id_fkey":
				return ErrUserNotFound
			case "user_roles_role_id_fkey":
				return ErrRoleSelectionInvalid
			}
			return ErrRoleSelectionInvalid
		case "23505":
			return ErrRoleSelectionInvalid
		}
	}
	return fmt.Errorf("SetUserRoles: %w", err)
}

// GetUserRoleNames 获取用户拥有的角色名列表（用于 scope 检查）
func (r *Repository) GetUserRoleNames(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ro.name FROM roles ro
		JOIN user_roles ur ON ur.role_id = ro.id
		WHERE ur.user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserRoleNames: %w", err)
	}
	defer rows.Close()

	names := make([]string, 0, 4)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("GetUserRoleNames scan: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
