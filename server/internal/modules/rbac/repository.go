package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

// Repository RBAC 数据访问层
type Repository struct {
	db *db.DB
}

// 编译期接口合规检查
var _ Repo = (*Repository)(nil)

// NewRepository 创建 RBAC 数据访问层
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ---------------------------------------------------------------------------
// Role operations
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Permission operations
// ---------------------------------------------------------------------------

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

	perms := make([]Permission, 0, 32)
	for rows.Next() {
		var p Permission
		var scopeSchoolIDs, scopeRoles []byte
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Module, &p.Action, &p.DisplayName,
			&scopeSchoolIDs, &scopeRoles, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListPermissions scan: %w", err)
		}
		if scopeSchoolIDs != nil {
			_ = json.Unmarshal(scopeSchoolIDs, &p.ScopeSchoolIDs)
		}
		if scopeRoles != nil {
			_ = json.Unmarshal(scopeRoles, &p.ScopeRoles)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
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
		_ = json.Unmarshal(scopeSchoolIDs, &p.ScopeSchoolIDs)
	}
	if scopeRoles != nil {
		_ = json.Unmarshal(scopeRoles, &p.ScopeRoles)
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

	perms := make([]Permission, 0, len(ids))
	for rows.Next() {
		var p Permission
		var scopeSchoolIDs, scopeRoles []byte
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Module, &p.Action, &p.DisplayName,
			&scopeSchoolIDs, &scopeRoles, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("GetPermissionsByIDs scan: %w", err)
		}
		if scopeSchoolIDs != nil {
			_ = json.Unmarshal(scopeSchoolIDs, &p.ScopeSchoolIDs)
		}
		if scopeRoles != nil {
			_ = json.Unmarshal(scopeRoles, &p.ScopeRoles)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// ---------------------------------------------------------------------------
// Role-Permission mapping
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// User-Role mapping
// ---------------------------------------------------------------------------

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
			return fmt.Errorf("SetUserRoles delete: %w", err)
		}
		for _, rid := range roleIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
			`, userID, rid); err != nil {
				return fmt.Errorf("SetUserRoles insert: %w", err)
			}
		}
		return nil
	})
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

// ---------------------------------------------------------------------------
// User Group operations
// ---------------------------------------------------------------------------

// ListGroups 获取所有用户组列表（含成员计数）
func (r *Repository) ListGroups(ctx context.Context) ([]UserGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.name, g.display_name, g.description, g.created_by,
		       g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM user_group_members m WHERE m.group_id = g.id) AS member_count
		FROM user_groups g
		ORDER BY g.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListGroups: %w", err)
	}
	defer rows.Close()

	groups := make([]UserGroup, 0, 16)
	for rows.Next() {
		var g UserGroup
		if err := rows.Scan(
			&g.ID, &g.Name, &g.DisplayName, &g.Description, &g.CreatedBy,
			&g.CreatedAt, &g.UpdatedAt, &g.MemberCount,
		); err != nil {
			return nil, fmt.Errorf("ListGroups scan: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// GetGroupByID 根据 ID 获取用户组
func (r *Repository) GetGroupByID(ctx context.Context, id int64) (*UserGroup, error) {
	var g UserGroup
	err := r.db.QueryRow(ctx, `
		SELECT g.id, g.name, g.display_name, g.description, g.created_by,
		       g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM user_group_members m WHERE m.group_id = g.id) AS member_count
		FROM user_groups g
		WHERE g.id = $1
	`, id).Scan(
		&g.ID, &g.Name, &g.DisplayName, &g.Description, &g.CreatedBy,
		&g.CreatedAt, &g.UpdatedAt, &g.MemberCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("GetGroupByID: %w", err)
	}
	return &g, nil
}

// CreateGroup 创建用户组
func (r *Repository) CreateGroup(ctx context.Context, group *UserGroup) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO user_groups (name, display_name, description, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, group.Name, group.DisplayName, group.Description, group.CreatedBy).Scan(
		&group.ID, &group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateGroup: %w", err)
	}
	return nil
}

// UpdateGroup 更新用户组
func (r *Repository) UpdateGroup(ctx context.Context, group *UserGroup) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE user_groups SET display_name = $2, description = $3, updated_at = NOW()
		WHERE id = $1
	`, group.ID, group.DisplayName, group.Description)
	if err != nil {
		return fmt.Errorf("UpdateGroup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// DeleteGroup 删除用户组
func (r *Repository) DeleteGroup(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeleteGroup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// GetGroupMembers 获取用户组成员 ID 列表
func (r *Repository) GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id FROM user_group_members
		WHERE group_id = $1
		ORDER BY user_id ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("GetGroupMembers: %w", err)
	}
	defer rows.Close()

	userIDs := make([]int64, 0, 32)
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("GetGroupMembers scan: %w", err)
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, rows.Err()
}

// GetGroupMembersDetail 获取用户组成员详情（join users 表）
func (r *Repository) GetGroupMembersDetail(ctx context.Context, groupID int64) ([]GroupMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.username, u.email, u.avatar_url, m.created_at
		FROM user_group_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.group_id = $1
		ORDER BY m.created_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("GetGroupMembersDetail: %w", err)
	}
	defer rows.Close()

	members := make([]GroupMember, 0, 32)
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.Email, &m.AvatarURL, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("GetGroupMembersDetail scan: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// SetGroupMembers 设置用户组成员（事务内 DELETE + INSERT）
func (r *Repository) SetGroupMembers(ctx context.Context, groupID int64, userIDs []int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM user_group_members WHERE group_id = $1`, groupID); err != nil {
			return fmt.Errorf("SetGroupMembers delete: %w", err)
		}
		for _, uid := range userIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_group_members (group_id, user_id) VALUES ($1, $2)
			`, groupID, uid); err != nil {
				return fmt.Errorf("SetGroupMembers insert: %w", err)
			}
		}
		return nil
	})
}

// SetGroupPermissions 设置用户组权限（事务内 DELETE + INSERT）
func (r *Repository) SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM user_group_permissions WHERE group_id = $1`, groupID); err != nil {
			return fmt.Errorf("SetGroupPermissions delete: %w", err)
		}
		for _, pid := range permIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_group_permissions (group_id, permission_id) VALUES ($1, $2)
			`, groupID, pid); err != nil {
				return fmt.Errorf("SetGroupPermissions insert: %w", err)
			}
		}
		return nil
	})
}

// GetGroupPermissions 获取用户组拥有的权限列表
func (r *Repository) GetGroupPermissions(ctx context.Context, groupID int64) ([]Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.module, p.action, p.display_name, p.scope_school_ids, p.scope_roles, p.created_at
		FROM permissions p
		JOIN user_group_permissions gp ON gp.permission_id = p.id
		WHERE gp.group_id = $1
		ORDER BY p.module, p.action
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("GetGroupPermissions: %w", err)
	}
	defer rows.Close()
	return scanPermissions(rows)
}

// ---------------------------------------------------------------------------
// User permission overrides
// ---------------------------------------------------------------------------

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
		return fmt.Errorf("SetUserPermission: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Effective permissions
// ---------------------------------------------------------------------------

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
		-- 合并所有权限来源，个人覆盖优先级最高
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

// ---------------------------------------------------------------------------
// User lookup helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// scanPermissions 扫描权限行
func scanPermissions(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]Permission, error) {
	perms := make([]Permission, 0, 16)
	for rows.Next() {
		var p Permission
		var scopeSchoolIDs, scopeRoles []byte
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Module, &p.Action, &p.DisplayName,
			&scopeSchoolIDs, &scopeRoles, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		if scopeSchoolIDs != nil {
			_ = json.Unmarshal(scopeSchoolIDs, &p.ScopeSchoolIDs)
		}
		if scopeRoles != nil {
			_ = json.Unmarshal(scopeRoles, &p.ScopeRoles)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
