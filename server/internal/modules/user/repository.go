package user

import "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"

// Repository 用户数据访问层
type Repository struct {
	db *db.DB
}

// 编译期接口合规检查
var _ Repo = (*Repository)(nil)

// NewRepository 创建数据访问层
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}
