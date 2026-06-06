package resource

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

var ErrResourceNotFound = errors.New("resource not found")
var ErrResourceForbidden = errors.New("resource forbidden")

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("resource.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}

func (r *Repository) CreateResource(ctx context.Context, ownerUserID string, req CreateRequest, mountID int64, stored *StoredObject) (*Item, error) {
	var resourceID int64
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO resource_items (owner_user_id, title, description, category, visibility)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, ownerUserID, req.Title, req.Description, req.Category, req.Visibility).Scan(&resourceID); err != nil {
			return fmt.Errorf("insert resource item: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO resource_versions (resource_id, version_no, mount_id, object_key, filename, content_type, size_bytes)
			VALUES ($1, 1, $2, $3, $4, $5, $6)
		`, resourceID, mountID, stored.ObjectKey, req.Filename, stored.ContentType, stored.SizeBytes); err != nil {
			return fmt.Errorf("insert resource version: %w", err)
		}
		return replaceTagsAndBindingsTx(ctx, tx, resourceID, req.Tags, req.Bindings)
	})
	if err != nil {
		return nil, err
	}
	return r.GetResourceByID(ctx, resourceID)
}

func (r *Repository) ListResources(ctx context.Context, filters ListFilters) ([]Item, int, error) {
	ctx = withDBTable(ctx, "resource_items")
	offset := httputil.SafeOffset(filters.Page, filters.PageSize)
	queryPattern := "%" + httputil.EscapeLikePattern(filters.Query) + "%"
	rows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.owner_user_id, ri.title, ri.description, ri.category, ri.visibility, ri.created_at, ri.updated_at,
		       rv.id, rv.version_no, rv.mount_id, rv.object_key, rv.filename, rv.content_type, rv.size_bytes, rv.created_at,
		       COUNT(*) OVER()
		FROM resource_items ri
		JOIN LATERAL (
			SELECT id, version_no, mount_id, object_key, filename, content_type, size_bytes, created_at
			FROM resource_versions
			WHERE resource_id = ri.id
			ORDER BY version_no DESC
			LIMIT 1
		) rv ON TRUE
		WHERE ri.visibility = 'public'
		  AND ($1 = '%%' OR ri.title ILIKE $1 ESCAPE '\' OR COALESCE(ri.description, '') ILIKE $1 ESCAPE '\')
		  AND ($2 = '' OR EXISTS (
			SELECT 1 FROM resource_tags rt WHERE rt.resource_id = ri.id AND rt.tag = $2
		  ))
		  AND (($3 = '' AND $4 = '') OR EXISTS (
			SELECT 1
			FROM resource_bindings rb
			WHERE rb.resource_id = ri.id
			  AND ($3 = '' OR rb.binding_type = $3)
			  AND ($4 = '' OR rb.binding_value = $4)
		  ))
		ORDER BY ri.created_at DESC, ri.id DESC
		LIMIT $5 OFFSET $6
	`, queryPattern, filters.Tag, filters.BindingType, filters.BindingValue, filters.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()
	return r.scanItems(ctx, rows)
}

func (r *Repository) GetResourceByID(ctx context.Context, resourceID int64) (*Item, error) {
	ctx = withDBTable(ctx, "resource_items")
	rows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.owner_user_id, ri.title, ri.description, ri.category, ri.visibility, ri.created_at, ri.updated_at,
		       rv.id, rv.version_no, rv.mount_id, rv.object_key, rv.filename, rv.content_type, rv.size_bytes, rv.created_at,
		       COUNT(*) OVER()
		FROM resource_items ri
		JOIN LATERAL (
			SELECT id, version_no, mount_id, object_key, filename, content_type, size_bytes, created_at
			FROM resource_versions
			WHERE resource_id = ri.id
			ORDER BY version_no DESC
			LIMIT 1
		) rv ON TRUE
		WHERE ri.id = $1
	`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("get resource by id: %w", err)
	}
	defer rows.Close()
	items, _, err := r.scanItems(ctx, rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrResourceNotFound
	}
	return &items[0], nil
}

func (r *Repository) UpdateResource(ctx context.Context, resourceID int64, ownerUserID string, req UpdateRequest) (*Item, error) {
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE resource_items
			SET title = $3, description = $4, category = $5, visibility = $6, updated_at = NOW()
			WHERE id = $1 AND owner_user_id = $2
		`, resourceID, ownerUserID, req.Title, req.Description, req.Category, req.Visibility)
		if err != nil {
			return fmt.Errorf("update resource item: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrResourceForbidden
		}
		return replaceTagsAndBindingsTx(ctx, tx, resourceID, req.Tags, req.Bindings)
	})
	if err != nil {
		return nil, err
	}
	return r.GetResourceByID(ctx, resourceID)
}

func (r *Repository) scanItems(ctx context.Context, rows pgx.Rows) ([]Item, int, error) {
	items := make([]Item, 0)
	total := 0
	for rows.Next() {
		item, err := scanItem(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		item.Tags, err = r.loadTags(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		item.Bindings, err = r.loadBindings(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func scanItem(rows pgx.Rows, total *int) (Item, error) {
	var item Item
	var createdAt time.Time
	var updatedAt time.Time
	var versionCreatedAt time.Time
	if err := rows.Scan(
		&item.ID, &item.OwnerUserID, &item.Title, &item.Description, &item.Category, &item.Visibility,
		&createdAt, &updatedAt, &item.LatestVersion.ID, &item.LatestVersion.VersionNo, &item.LatestVersion.MountID,
		&item.LatestVersion.ObjectKey, &item.LatestVersion.Filename, &item.LatestVersion.ContentType,
		&item.LatestVersion.SizeBytes, &versionCreatedAt, total,
	); err != nil {
		return Item{}, fmt.Errorf("scan resource item: %w", err)
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	item.LatestVersion.CreatedAt = versionCreatedAt.UTC().Format(time.RFC3339)
	return item, nil
}

func (r *Repository) loadTags(ctx context.Context, resourceID int64) ([]string, error) {
	ctx = withDBTable(ctx, "resource_tags")
	rows, err := r.db.Query(ctx, `SELECT tag FROM resource_tags WHERE resource_id = $1 ORDER BY tag ASC`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("load resource tags: %w", err)
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan resource tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *Repository) loadBindings(ctx context.Context, resourceID int64) ([]Binding, error) {
	ctx = withDBTable(ctx, "resource_bindings")
	rows, err := r.db.Query(ctx, `
		SELECT binding_type, binding_value
		FROM resource_bindings
		WHERE resource_id = $1
		ORDER BY binding_type ASC, binding_value ASC
	`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("load resource bindings: %w", err)
	}
	defer rows.Close()
	var bindings []Binding
	for rows.Next() {
		var item Binding
		if err := rows.Scan(&item.Type, &item.Value); err != nil {
			return nil, fmt.Errorf("scan resource binding: %w", err)
		}
		bindings = append(bindings, item)
	}
	return bindings, rows.Err()
}

func replaceTagsAndBindingsTx(ctx context.Context, tx pgx.Tx, resourceID int64, tags []string, bindings []Binding) error {
	if _, err := tx.Exec(ctx, `DELETE FROM resource_tags WHERE resource_id = $1`, resourceID); err != nil {
		return fmt.Errorf("clear resource tags: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM resource_bindings WHERE resource_id = $1`, resourceID); err != nil {
		return fmt.Errorf("clear resource bindings: %w", err)
	}
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO resource_tags (resource_id, tag) VALUES ($1, $2)`, resourceID, tag); err != nil {
			return fmt.Errorf("insert resource tag: %w", err)
		}
	}
	for _, binding := range bindings {
		if binding.Type == "" || binding.Value == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO resource_bindings (resource_id, binding_type, binding_value)
			VALUES ($1, $2, $3)
		`, resourceID, binding.Type, binding.Value); err != nil {
			return fmt.Errorf("insert resource binding: %w", err)
		}
	}
	return nil
}
