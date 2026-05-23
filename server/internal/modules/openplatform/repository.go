package openplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type execFn func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) CreateApp(ctx context.Context, app *App, scopes []ScopeRequest) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := insertAppTx(ctx, tx, app); err != nil {
			return err
		}
		for i := range scopes {
			scopes[i].AppID = app.ID
			if err := insertScopeRequest(ctx, tx.Exec, &scopes[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) ImportApprovedApp(ctx context.Context, app *App, scopes []ScopeRequest, reviewerUserID int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := insertAppTx(ctx, tx, app); err != nil {
			if isUniqueViolation(err) {
				return ErrAppAlreadyExists
			}
			return err
		}
		for i := range scopes {
			scopes[i].AppID = app.ID
			if err := upsertApprovedScopeRequest(ctx, tx.Exec, &scopes[i]); err != nil {
				return err
			}
			if err := insertApprovedScope(ctx, tx.Exec, app.ID, scopes[i].Scope, reviewerUserID); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertAppTx(ctx context.Context, tx pgx.Tx, app *App) error {
	redirects, err := json.Marshal(app.RedirectURIs)
	if err != nil {
		return fmt.Errorf("CreateApp marshal redirects: %w", err)
	}
	return tx.QueryRow(ctx, `
		INSERT INTO open_platform_apps (
			casdoor_application_name, owner_user_id, client_id, client_secret_hash,
			display_name, description, homepage_url, privacy_policy_url,
			redirect_uris, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, app.CasdoorApplicationName, app.OwnerUserID, app.ClientID, app.ClientSecretHash,
		app.DisplayName, app.Description, app.HomepageURL, app.PrivacyPolicyURL,
		redirects, app.Status,
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)
}

func insertScopeRequest(ctx context.Context, exec execFn, req *ScopeRequest) error {
	return execRow(ctx, exec, `
		INSERT INTO open_platform_scope_requests (
			app_id, scope, reason, status, reviewer_user_id, reviewed_at,
			decision_note, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`, req.AppID, req.Scope, req.Reason, req.Status, req.ReviewerUserID,
		req.ReviewedAt, req.DecisionNote)
}

func upsertApprovedScopeRequest(ctx context.Context, exec execFn, req *ScopeRequest) error {
	return execRow(ctx, exec, `
		INSERT INTO open_platform_scope_requests (
			app_id, scope, reason, status, reviewer_user_id, reviewed_at,
			decision_note, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW(), NOW())
		ON CONFLICT (app_id, scope) DO UPDATE SET
			reason = EXCLUDED.reason,
			status = EXCLUDED.status,
			reviewer_user_id = EXCLUDED.reviewer_user_id,
			reviewed_at = NOW(),
			decision_note = EXCLUDED.decision_note,
			updated_at = NOW()
	`, req.AppID, req.Scope, req.Reason, ScopeStatusApproved, req.ReviewerUserID,
		req.DecisionNote)
}

func insertApprovedScope(ctx context.Context, exec execFn, appID int64, scope string, reviewerUserID int64) error {
	return execRow(ctx, exec, `
		INSERT INTO open_platform_approved_scopes (app_id, scope, approved_at, approved_by)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (app_id, scope) DO UPDATE SET
			approved_at = EXCLUDED.approved_at,
			approved_by = EXCLUDED.approved_by
	`, appID, scope, reviewerUserID)
}

func (r *Repository) GetAppByID(ctx context.Context, appID int64) (*App, error) {
	app, err := scanApp(r.db.QueryRow(ctx, `
		SELECT id, casdoor_application_name, owner_user_id, client_id,
		       client_secret_hash, display_name, description, homepage_url,
		       privacy_policy_url, redirect_uris, status, created_at, updated_at
		FROM open_platform_apps
		WHERE id = $1
	`, appID))
	if err != nil {
		return nil, fmt.Errorf("GetAppByID: %w", err)
	}
	return app, nil
}

func (r *Repository) GetAppByClientID(ctx context.Context, clientID string) (*App, error) {
	app, err := scanApp(r.db.QueryRow(ctx, `
		SELECT id, casdoor_application_name, owner_user_id, client_id,
		       client_secret_hash, display_name, description, homepage_url,
		       privacy_policy_url, redirect_uris, status, created_at, updated_at
		FROM open_platform_apps
		WHERE client_id = $1
	`, clientID))
	if err != nil {
		return nil, fmt.Errorf("GetAppByClientID: %w", err)
	}
	return app, nil
}

func (r *Repository) VerifyClientSecret(ctx context.Context, clientID, clientSecret string) (*App, error) {
	app, err := r.GetAppByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if app.ClientSecretHash == "" || hashClientSecret(clientSecret) != app.ClientSecretHash {
		return nil, ErrAppNotFound
	}
	return app, nil
}

func scanApp(row rowScanner) (*App, error) {
	var app App
	var redirects []byte
	err := row.Scan(&app.ID, &app.CasdoorApplicationName, &app.OwnerUserID,
		&app.ClientID, &app.ClientSecretHash, &app.DisplayName, &app.Description,
		&app.HomepageURL, &app.PrivacyPolicyURL, &redirects, &app.Status,
		&app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(redirects, &app.RedirectURIs); err != nil {
		return nil, fmt.Errorf("unmarshal redirect_uris: %w", err)
	}
	return &app, nil
}

func (r *Repository) ApproveScope(ctx context.Context, appID int64, scope string, reviewerUserID int64, note string) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := updateScopeRequestDecision(ctx, tx.Exec, appID, scope, reviewerUserID, note, ScopeStatusApproved); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO open_platform_approved_scopes (app_id, scope, approved_at, approved_by)
			VALUES ($1, $2, NOW(), $3)
			ON CONFLICT (app_id, scope) DO UPDATE SET
				approved_at = EXCLUDED.approved_at,
				approved_by = EXCLUDED.approved_by
		`, appID, scope, reviewerUserID)
		if err != nil {
			return fmt.Errorf("ApproveScope insert approved: %w", err)
		}
		return nil
	})
}

func updateScopeRequestDecision(
	ctx context.Context,
	exec execFn,
	appID int64,
	scope string,
	reviewerUserID int64,
	note string,
	status string,
) error {
	tag, err := exec(ctx, `
		UPDATE open_platform_scope_requests
		SET status = $3,
		    reviewer_user_id = $4,
		    reviewed_at = NOW(),
		    decision_note = NULLIF($5, ''),
		    updated_at = NOW()
		WHERE app_id = $1 AND scope = $2
	`, appID, scope, status, reviewerUserID, note)
	if err != nil {
		return fmt.Errorf("update scope request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidScope
	}
	return nil
}

func (r *Repository) SetAppStatus(ctx context.Context, appID int64, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE open_platform_apps
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, appID, status)
	if err != nil {
		return fmt.Errorf("SetAppStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAppNotFound
	}
	return nil
}

func (r *Repository) MarkAppApproved(ctx context.Context, appID int64, clientSecretHash string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE open_platform_apps
		SET status = $2,
		    client_secret_hash = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, appID, AppStatusApproved, clientSecretHash)
	if err != nil {
		return fmt.Errorf("MarkAppApproved: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAppNotFound
	}
	return nil
}

func (r *Repository) ListApprovedScopes(ctx context.Context, appID int64) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT scope
		FROM open_platform_approved_scopes
		WHERE app_id = $1
		ORDER BY scope ASC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("ListApprovedScopes: %w", err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, fmt.Errorf("ListApprovedScopes scan: %w", err)
		}
		scopes = append(scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListApprovedScopes rows: %w", err)
	}
	return scopes, nil
}

func execRow(ctx context.Context, exec execFn, sql string, args ...any) error {
	_, err := exec(ctx, sql, args...)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
