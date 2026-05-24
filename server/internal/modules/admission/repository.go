package admission

import (
	"context"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}

func (r *Repository) WithTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	return r.db.WithTx(ctx, fn)
}
