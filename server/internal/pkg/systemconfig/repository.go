package systemconfig

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		return nil
	}
	return &Repository{db: database}
}

func (r *Repository) GetEmailDeliveryPolicyValue(ctx context.Context) (string, bool, error) {
	if r == nil || r.db == nil {
		return "", false, nil
	}
	ctx = db.WithTableHint(ctx, "system_configs")
	var value string
	err := r.db.QueryRow(ctx, `
		SELECT value
		FROM system_configs
		WHERE key = $1
	`, EmailDeliveryPolicyKey).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("GetEmailDeliveryPolicyValue: %w", err)
	}
	return value, true, nil
}
