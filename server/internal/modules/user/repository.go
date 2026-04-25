package user

import (
	"context"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

// Repository 用户数据访问层
type Repository struct {
	db      *db.DB
	hmacKey []byte
}

// 编译期接口合规检查
var _ Repo = (*Repository)(nil)

// NewRepository 创建数据访问层
func NewRepository(database *db.DB, hmacKey []byte) *Repository {
	return &Repository{db: database, hmacKey: hmacKey}
}

func (r *Repository) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return r.db.WithTx(ctx, fn)
}
