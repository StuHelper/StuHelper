package user

import (
	"context"
	"fmt"
	"strconv"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/usersync"
)

const superAdminRoleName = "super_admin"

type roleFGAClient interface {
	WriteMissingTuples(ctx context.Context, desired []fga.Tuple) error
}

// AuthSyncInput 认证同步输入，别名到 usersync.Input。
type AuthSyncInput = usersync.Input

// UserSyncRepository 用户同步持久化层，负责认证流程中的用户创建 / 查找 / 回填。
// 业务归属为 user domain，auth domain 通过窄接口依赖。
type UserSyncRepository struct {
	db      *db.DB
	hmacKey []byte
	roleFGA roleFGAClient
}

// NewUserSyncRepository 创建用户同步仓储
func NewUserSyncRepository(database *db.DB, hmacKey []byte) *UserSyncRepository {
	return &UserSyncRepository{
		db:      database,
		hmacKey: hmacKey,
	}
}

func (r *UserSyncRepository) WithRoleFGAClient(client roleFGAClient) *UserSyncRepository {
	r.roleFGA = client
	return r
}

// UpsertUser 同步 OIDC / SSO 登录用户到本地 shadow user 表。
func (r *UserSyncRepository) UpsertUser(ctx context.Context, input AuthSyncInput) error {
	ctx = withDBTable(ctx, "users")
	if input.CasdoorSubject == "" || input.Username == "" {
		return fmt.Errorf("UpsertUser: casdoorSubject and username are required")
	}

	userHash, err := crypto.HMACHashWithKey(input.CasdoorSubject, r.hmacKey)
	if err != nil {
		return fmt.Errorf("UpsertUser: compute user_hash: %w", err)
	}

	var userID int64
	err = r.db.QueryRow(ctx, `
		INSERT INTO users (casdoor_subject, username, email, avatar_url, user_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (casdoor_subject) DO UPDATE SET
			username = EXCLUDED.username,
			email = COALESCE(EXCLUDED.email, users.email),
			avatar_url = COALESCE(EXCLUDED.avatar_url, users.avatar_url),
			user_hash = COALESCE(EXCLUDED.user_hash, users.user_hash),
			updated_at = NOW()
		RETURNING id
	`, input.CasdoorSubject, input.Username, emptyToNil(input.Email), input.AvatarURL, userHash).Scan(&userID)
	if err != nil {
		return fmt.Errorf("UpsertUser: %w", err)
	}
	return r.syncGlobalRoleRelations(ctx, userID, input.Roles)
}

func (r *UserSyncRepository) syncGlobalRoleRelations(ctx context.Context, userID int64, roles []string) error {
	if r.roleFGA == nil || !hasSyncRole(roles, superAdminRoleName) {
		return nil
	}
	if userID <= 0 {
		return fmt.Errorf("sync global role relations: userID is required")
	}
	err := r.roleFGA.WriteMissingTuples(ctx, []fga.Tuple{{
		User:     "user:" + strconv.FormatInt(userID, 10),
		Relation: superAdminRoleName,
		Object:   "ecosystem:stuhelper",
	}})
	if err != nil {
		return fmt.Errorf("sync global role relations: %w", err)
	}
	return nil
}

func hasSyncRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

// ExistsByCasdoorSubject 检查 casdoor_subject 对应的用户是否仍然存在。
func (r *UserSyncRepository) ExistsByCasdoorSubject(ctx context.Context, casdoorSubject string) (bool, error) {
	ctx = withDBTable(ctx, "users")
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE casdoor_subject = $1)`, casdoorSubject).Scan(&exists); err != nil {
		return false, fmt.Errorf("ExistsByCasdoorSubject: %w", err)
	}
	return exists, nil
}

// CountMissingUserHashes 返回 user_hash 为空的用户数量（用于启动时检查）。
func (r *UserSyncRepository) CountMissingUserHashes(ctx context.Context) (int64, error) {
	ctx = withDBTable(ctx, "users")
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE user_hash IS NULL`).Scan(&count); err != nil {
		return 0, fmt.Errorf("CountMissingUserHashes: %w", err)
	}
	return count, nil
}

// BackfillUserHashes 回填所有 user_hash 为空的用户。
func (r *UserSyncRepository) BackfillUserHashes(ctx context.Context) (int64, error) {
	ctx = withDBTable(ctx, "users")
	const batchSize = 500
	var totalCount int64

	for {
		rows, err := r.db.Query(ctx, `SELECT id, casdoor_subject FROM users WHERE user_hash IS NULL LIMIT $1`, batchSize)
		if err != nil {
			return totalCount, fmt.Errorf("BackfillUserHashes query: %w", err)
		}

		var batchCount int64
		for rows.Next() {
			var id int64
			var casdoorSubject string
			if err := rows.Scan(&id, &casdoorSubject); err != nil {
				rows.Close()
				return totalCount, fmt.Errorf("BackfillUserHashes scan: %w", err)
			}
			hash, err := crypto.HMACHashWithKey(casdoorSubject, r.hmacKey)
			if err != nil {
				rows.Close()
				return totalCount, fmt.Errorf("BackfillUserHashes hash user %d: %w", id, err)
			}
			if _, err := r.db.Exec(ctx, `UPDATE users SET user_hash = $1 WHERE id = $2 AND user_hash IS NULL`, hash, id); err != nil {
				rows.Close()
				return totalCount, fmt.Errorf("BackfillUserHashes update user %d: %w", id, err)
			}
			batchCount++
			totalCount++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return totalCount, err
		}

		if batchCount < batchSize {
			break
		}
	}

	return totalCount, nil
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
