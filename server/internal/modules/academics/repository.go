package academics

import (
	"context"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("academics.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}
