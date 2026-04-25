package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

var ErrMountNotFound = errors.New("storage mount not found")

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) UpsertRuntimeDefaultMount(ctx context.Context, cfg config.ObjectStorageConfig) error {
	if cfg.Endpoint == "" && cfg.Bucket == "" {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO storage_mounts (key, name, driver, bucket, base_path, credential_source, enabled)
		VALUES ($2, 'Default S3 Mount', 's3', $1, '', 'runtime_default_object_storage', TRUE)
		ON CONFLICT (key)
		DO UPDATE SET bucket = COALESCE(NULLIF(EXCLUDED.bucket, ''), storage_mounts.bucket),
		              updated_at = NOW()
	`, cfg.Bucket, DefaultMountKey)
	if err != nil {
		return fmt.Errorf("upsert runtime default mount: %w", err)
	}
	return nil
}

func (r *Repository) ListMounts(ctx context.Context) ([]Mount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, driver, bucket, base_path, credential_source, enabled,
		       last_health_status, last_health_error, last_health_checked_at
		FROM storage_mounts
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list storage mounts: %w", err)
	}
	defer rows.Close()
	var mounts []Mount
	for rows.Next() {
		item, err := scanMount(rows)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, item)
	}
	return mounts, rows.Err()
}

func (r *Repository) CreateMount(ctx context.Context, req CreateMountRequest) (*Mount, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO storage_mounts (key, name, driver, bucket, base_path, credential_source, enabled)
		VALUES ($1, $2, $3, $4, $5, 'runtime_default_object_storage', $6)
		RETURNING id, key, name, driver, bucket, base_path, credential_source, enabled,
		          last_health_status, last_health_error, last_health_checked_at
	`, req.Key, req.Name, req.Driver, req.Bucket, req.BasePath, req.Enabled)
	if err != nil {
		return nil, fmt.Errorf("create storage mount: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("create storage mount: %w", pgx.ErrNoRows)
	}
	item, err := scanMount(rows)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetMountByID(ctx context.Context, mountID int64) (*Mount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, driver, bucket, base_path, credential_source, enabled,
		       last_health_status, last_health_error, last_health_checked_at
		FROM storage_mounts
		WHERE id = $1
	`, mountID)
	if err != nil {
		return nil, fmt.Errorf("get storage mount by id: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrMountNotFound
	}
	item, err := scanMount(rows)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetMountByKey(ctx context.Context, key string) (*Mount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, driver, bucket, base_path, credential_source, enabled,
		       last_health_status, last_health_error, last_health_checked_at
		FROM storage_mounts
		WHERE key = $1
	`, key)
	if err != nil {
		return nil, fmt.Errorf("get storage mount by key: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrMountNotFound
	}
	item, err := scanMount(rows)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpdateMountHealth(ctx context.Context, mountID int64, status string, errorMessage *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE storage_mounts
		SET last_health_status = $2, last_health_error = $3, last_health_checked_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, mountID, status, errorMessage)
	if err != nil {
		return fmt.Errorf("update storage mount health: %w", err)
	}
	return nil
}

func scanMount(rows pgx.Rows) (Mount, error) {
	var item Mount
	var checkedAt *time.Time
	if err := rows.Scan(
		&item.ID, &item.Key, &item.Name, &item.Driver, &item.Bucket, &item.BasePath,
		&item.CredentialSource, &item.Enabled, &item.LastHealthStatus, &item.LastHealthError, &checkedAt,
	); err != nil {
		return Mount{}, fmt.Errorf("scan storage mount: %w", err)
	}
	item.LastHealthCheckedAt = formatTimePtr(checkedAt)
	return item, nil
}

func formatTimePtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
