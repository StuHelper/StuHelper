package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListPermissions 获取权限列表，可按模块过滤
func (r *Repository) ListPermissions(ctx context.Context, module string) ([]Permission, error) {
	var query string
	var args []any

	if module != "" {
		query = `
			SELECT id, name, module, action, display_name, scope_school_ids, scope_roles, created_at
			FROM permissions
			WHERE module = $1
			ORDER BY module, action
		`
		args = []any{module}
	} else {
		query = `
			SELECT id, name, module, action, display_name, scope_school_ids, scope_roles, created_at
			FROM permissions
			ORDER BY module, action
		`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListPermissions: %w", err)
	}
	defer rows.Close()

	return scanPermissions(rows)
}

// GetPermissionByID 根据 ID 获取权限
func (r *Repository) GetPermissionByID(ctx context.Context, id int64) (*Permission, error) {
	var p Permission
	var scopeSchoolIDs, scopeRoles []byte
	err := r.db.QueryRow(ctx, `
		SELECT id, name, module, action, display_name, scope_school_ids, scope_roles, created_at
		FROM permissions WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Module, &p.Action, &p.DisplayName,
		&scopeSchoolIDs, &scopeRoles, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPermNotFound
		}
		return nil, fmt.Errorf("GetPermissionByID: %w", err)
	}
	if scopeSchoolIDs != nil {
		if err := json.Unmarshal(scopeSchoolIDs, &p.ScopeSchoolIDs); err != nil {
			return nil, fmt.Errorf("GetPermissionByID: corrupt scope_school_ids for permission %d: %w", p.ID, err)
		}
	}
	if scopeRoles != nil {
		if err := json.Unmarshal(scopeRoles, &p.ScopeRoles); err != nil {
			return nil, fmt.Errorf("GetPermissionByID: corrupt scope_roles for permission %d: %w", p.ID, err)
		}
	}
	return &p, nil
}

// GetPermissionsByIDs 批量获取权限
func (r *Repository) GetPermissionsByIDs(ctx context.Context, ids []int64) ([]Permission, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, name, module, action, display_name, scope_school_ids, scope_roles, created_at
		FROM permissions WHERE id = ANY($1)
		ORDER BY module, action
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("GetPermissionsByIDs: %w", err)
	}
	defer rows.Close()

	return scanPermissions(rows)
}
