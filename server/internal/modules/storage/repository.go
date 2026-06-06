package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

var ErrMountNotFound = errors.New("storage mount not found")

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("storage.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}

func (r *Repository) UpsertRuntimeDefaultMount(ctx context.Context, cfg config.ObjectStorageConfig) error {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	bucket := strings.TrimSpace(cfg.Bucket)
	if endpoint == "" {
		return nil
	}
	ctx = withDBTable(ctx, "storage_mounts")

	if err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var insertedBucket *string
		err := tx.QueryRow(ctx, `
			INSERT INTO storage_mounts (key, name, driver, bucket, base_path, credential_source, enabled)
			VALUES ($2, 'Default S3 Mount', 's3', NULLIF($1, ''), '', 'runtime_default_object_storage', TRUE)
			ON CONFLICT (key) DO NOTHING
			RETURNING bucket
		`, bucket, DefaultMountKey).Scan(&insertedBucket)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("insert runtime default mount: %w", err)
		}

		var existingBucket *string
		if err := tx.QueryRow(ctx, `
			SELECT bucket
			FROM storage_mounts
			WHERE key = $1
			FOR UPDATE
		`, DefaultMountKey).Scan(&existingBucket); err != nil {
			return fmt.Errorf("lock runtime default mount: %w", err)
		}

		if existingBucket == nil || *existingBucket == "" {
			_, err := tx.Exec(ctx, `
				UPDATE storage_mounts
				SET bucket = NULLIF($1, ''), updated_at = NOW()
				WHERE key = $2
			`, bucket, DefaultMountKey)
			if err != nil {
				return fmt.Errorf("backfill runtime default mount bucket: %w", err)
			}
			return nil
		}

		if *existingBucket != bucket {
			return defaultMountBucketDriftError(*existingBucket, bucket)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("upsert runtime default mount: %w", err)
	}
	return nil
}

func defaultMountBucketDriftError(existingBucket, configuredBucket string) error {
	return fmt.Errorf(
		"%w: existing default mount bucket %q differs from configured OBJECT_STORAGE_BUCKET %q; resolve default mount bucket drift with an explicit object migration or by creating a new storage mount instead of overriding it from startup configuration",
		ErrDefaultMountBucketDrift,
		existingBucket,
		configuredBucket,
	)
}

func (r *Repository) ListMounts(ctx context.Context) ([]Mount, error) {
	ctx = withDBTable(ctx, "storage_mounts")
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list storage mounts: %w", err)
	}
	return mounts, nil
}

func (r *Repository) CreateMount(ctx context.Context, req CreateMountRequest) (*Mount, error) {
	ctx = withDBTable(ctx, "storage_mounts")
	item, err := scanMount(r.db.QueryRow(ctx, `
		INSERT INTO storage_mounts (key, name, driver, bucket, base_path, credential_source, enabled)
		VALUES ($1, $2, $3, $4, $5, 'runtime_default_object_storage', $6)
		RETURNING id, key, name, driver, bucket, base_path, credential_source, enabled,
		          last_health_status, last_health_error, last_health_checked_at
	`, req.Key, req.Name, req.Driver, req.Bucket, req.BasePath, req.Enabled))
	if err != nil {
		if isStorageMountKeyUniqueViolation(err) {
			return nil, ErrMountAlreadyExists
		}
		return nil, fmt.Errorf("create storage mount: %w", err)
	}
	return &item, nil
}

func (r *Repository) GetMountByID(ctx context.Context, mountID int64) (*Mount, error) {
	ctx = withDBTable(ctx, "storage_mounts")
	item, err := scanMount(r.db.QueryRow(ctx, `
		SELECT id, key, name, driver, bucket, base_path, credential_source, enabled,
		       last_health_status, last_health_error, last_health_checked_at
		FROM storage_mounts
		WHERE id = $1
	`, mountID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get storage mount by id: %w", err)
	}
	return &item, nil
}

func (r *Repository) GetMountByKey(ctx context.Context, key string) (*Mount, error) {
	ctx = withDBTable(ctx, "storage_mounts")
	item, err := scanMount(r.db.QueryRow(ctx, `
		SELECT id, key, name, driver, bucket, base_path, credential_source, enabled,
		       last_health_status, last_health_error, last_health_checked_at
		FROM storage_mounts
		WHERE key = $1
	`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get storage mount by key: %w", err)
	}
	return &item, nil
}

func (r *Repository) UpdateMountHealth(ctx context.Context, mountID int64, status string, errorMessage *string) error {
	ctx = withDBTable(ctx, "storage_mounts")
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

type mountScanner interface {
	Scan(dest ...any) error
}

func scanMount(row mountScanner) (Mount, error) {
	var item Mount
	var checkedAt *time.Time
	if err := row.Scan(
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

func isStorageMountKeyUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "storage_mounts_key_key"
}
