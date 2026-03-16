package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListRoles 获取所有角色列表
func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, display_name, description, is_system, created_at, updated_at
		FROM roles
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListRoles: %w", err)
	}
	defer rows.Close()

	roles := make([]Role, 0, 16)
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListRoles scan: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// GetRoleByID 根据 ID 获取角色
func (r *Repository) GetRoleByID(ctx context.Context, id int64) (*Role, error) {
	var role Role
	err := r.db.QueryRow(ctx, `
		SELECT id, name, display_name, description, is_system, created_at, updated_at
		FROM roles WHERE id = $1
	`, id).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("GetRoleByID: %w", err)
	}
	return &role, nil
}

// GetRoleByName 根据名称获取角色
func (r *Repository) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := r.db.QueryRow(ctx, `
		SELECT id, name, display_name, description, is_system, created_at, updated_at
		FROM roles WHERE name = $1
	`, name).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("GetRoleByName: %w", err)
	}
	return &role, nil
}

// CreateRole 创建角色
func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO roles (name, display_name, description, is_system, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, role.Name, role.DisplayName, role.Description, role.IsSystem).Scan(
		&role.ID, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateRole: %w", err)
	}
	return nil
}

// UpdateRole 更新角色
func (r *Repository) UpdateRole(ctx context.Context, role *Role) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE roles SET display_name = $2, description = $3, updated_at = NOW()
		WHERE id = $1
	`, role.ID, role.DisplayName, role.Description)
	if err != nil {
		return fmt.Errorf("UpdateRole: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// DeleteRole 删除角色
func (r *Repository) DeleteRole(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeleteRole: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// SetRolePermissions 设置角色的权限列表（事务内 DELETE + INSERT）
func (r *Repository) SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
			return fmt.Errorf("SetRolePermissions delete: %w", err)
		}
		for _, pid := range permIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
			`, roleID, pid); err != nil {
				return fmt.Errorf("SetRolePermissions insert: %w", err)
			}
		}
		return nil
	})
}

// GetRolePermissionIDs 获取角色拥有的权限 ID 列表
func (r *Repository) GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT permission_id
		FROM role_permissions
		WHERE role_id = $1
		ORDER BY permission_id ASC
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("GetRolePermissionIDs: %w", err)
	}
	defer rows.Close()

	permIDs := make([]int64, 0, 16)
	for rows.Next() {
		var permID int64
		if err := rows.Scan(&permID); err != nil {
			return nil, fmt.Errorf("GetRolePermissionIDs scan: %w", err)
		}
		permIDs = append(permIDs, permID)
	}
	return permIDs, rows.Err()
}
