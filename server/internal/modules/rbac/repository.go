package rbac

import (
	"encoding/json"
	"fmt"

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
			if err := json.Unmarshal(scopeSchoolIDs, &p.ScopeSchoolIDs); err != nil {
				return nil, fmt.Errorf("scanPermissions: corrupt scope_school_ids for permission %d: %w", p.ID, err)
			}
		}
		if scopeRoles != nil {
			if err := json.Unmarshal(scopeRoles, &p.ScopeRoles); err != nil {
				return nil, fmt.Errorf("scanPermissions: corrupt scope_roles for permission %d: %w", p.ID, err)
			}
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
