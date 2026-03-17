package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
			return mapSetGroupMembersWriteError(err)
		}
		for _, uid := range userIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_group_members (group_id, user_id) VALUES ($1, $2)
			`, groupID, uid); err != nil {
				return mapSetGroupMembersWriteError(err)
			}
		}
		return nil
	})
}

// SetGroupPermissions 设置用户组权限（事务内 DELETE + INSERT）
func (r *Repository) SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM user_group_permissions WHERE group_id = $1`, groupID); err != nil {
			return mapSetGroupPermissionsWriteError(err)
		}
		for _, pid := range permIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_group_permissions (group_id, permission_id) VALUES ($1, $2)
			`, groupID, pid); err != nil {
				return mapSetGroupPermissionsWriteError(err)
			}
		}
		return nil
	})
}

// CountUsersByIDs 返回 userIDs 中存在的用户数量（去重后）
func (r *Repository) CountUsersByIDs(ctx context.Context, userIDs []int64) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	var count int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::INT
		FROM (
			SELECT DISTINCT id
			FROM users
			WHERE id = ANY($1)
		) AS matched_users
	`, userIDs).Scan(&count); err != nil {
		return 0, fmt.Errorf("CountUsersByIDs: %w", err)
	}
	return count, nil
}

// CountPermissionsByIDs 返回 permIDs 中存在的权限数量（去重后）
func (r *Repository) CountPermissionsByIDs(ctx context.Context, permIDs []int64) (int, error) {
	if len(permIDs) == 0 {
		return 0, nil
	}

	var count int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::INT
		FROM (
			SELECT DISTINCT id
			FROM permissions
			WHERE id = ANY($1)
		) AS matched_permissions
	`, permIDs).Scan(&count); err != nil {
		return 0, fmt.Errorf("CountPermissionsByIDs: %w", err)
	}
	return count, nil
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

func mapSetGroupMembersWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			switch pgErr.ConstraintName {
			case "user_group_members_group_id_fkey":
				return ErrGroupNotFound
			case "user_group_members_user_id_fkey":
				return ErrUserSelectionInvalid
			}
			return ErrUserSelectionInvalid
		case "23505":
			return ErrUserSelectionInvalid
		}
	}
	return fmt.Errorf("SetGroupMembers: %w", err)
}

func mapSetGroupPermissionsWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			switch pgErr.ConstraintName {
			case "user_group_permissions_group_id_fkey":
				return ErrGroupNotFound
			case "user_group_permissions_permission_id_fkey":
				return ErrPermissionSelectionInvalid
			}
			return ErrPermissionSelectionInvalid
		case "23505":
			return ErrPermissionSelectionInvalid
		}
	}
	return fmt.Errorf("SetGroupPermissions: %w", err)
}
