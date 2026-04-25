package academics

import "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}
