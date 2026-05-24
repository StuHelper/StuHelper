package academics

import (
	"context"

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
