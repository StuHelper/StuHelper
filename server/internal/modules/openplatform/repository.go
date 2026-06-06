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
	if database == nil {
		panic("openplatform.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
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
	ctx = withDBTable(ctx, "open_platform_apps")
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
	ctx = withDBTable(ctx, "open_platform_apps")
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

func (r *Repository) ApproveScope(
	ctx context.Context,
	appID int64,
	scope string,
	reviewerUserID int64,
	note string,
	requestID string,
) error {
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
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    reviewerUserID,
			EventType: "open_platform.scope.approved",
			Scope:     scope,
			RequestID: requestID,
			Metadata: map[string]any{
				"decisionNote": note,
				"status":       ScopeStatusApproved,
			},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (r *Repository) RejectScope(
	ctx context.Context,
	appID int64,
	scope string,
	reviewerUserID int64,
	note string,
	requestID string,
) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := updateScopeRequestDecision(ctx, tx.Exec, appID, scope, reviewerUserID, note, ScopeStatusRejected); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    reviewerUserID,
			EventType: "open_platform.scope.rejected",
			Scope:     scope,
			RequestID: requestID,
			Metadata: map[string]any{
				"decisionNote": note,
				"status":       ScopeStatusRejected,
			},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (r *Repository) UpsertScopeRequestsWithAudit(
	ctx context.Context,
	appID int64,
	requests []ScopeRequest,
	actorUserID int64,
	requestID string,
) (ScopeChangeResult, error) {
	created := make([]ScopeRequest, 0, len(requests))
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, req := range requests {
			saved, err := upsertScopeRequestForReview(ctx, tx, appID, req)
			if err != nil {
				return err
			}
			if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
				AppID:     appID,
				UserID:    actorUserID,
				EventType: "open_platform.scope.requested",
				Scope:     saved.Scope,
				RequestID: requestID,
				Metadata: map[string]any{
					"reason": saved.Reason,
					"status": saved.Status,
				},
			}); err != nil {
				return err
			}
			created = append(created, saved)
		}
		return nil
	})
	if err != nil {
		return ScopeChangeResult{}, err
	}
	return ScopeChangeResult{Scopes: created}, nil
}

func upsertScopeRequestForReview(
	ctx context.Context,
	tx pgx.Tx,
	appID int64,
	req ScopeRequest,
) (ScopeRequest, error) {
	saved, err := scanScopeRequest(tx.QueryRow(ctx, `
		INSERT INTO open_platform_scope_requests (
			app_id, scope, reason, status, reviewer_user_id, reviewed_at,
			decision_note, created_at, updated_at
		) VALUES ($1, $2, $3, 'pending', NULL, NULL, NULL, NOW(), NOW())
		ON CONFLICT (app_id, scope) DO NOTHING
		RETURNING id, app_id, scope, reason, status, reviewer_user_id,
		          reviewed_at, decision_note, created_at, updated_at
	`, appID, req.Scope, req.Reason))
	if err == nil {
		return saved, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ScopeRequest{}, fmt.Errorf("UpsertScopeRequestsWithAudit insert scope: %w", err)
	}

	existing, err := scanScopeRequest(tx.QueryRow(ctx, `
		SELECT id, app_id, scope, reason, status, reviewer_user_id,
		       reviewed_at, decision_note, created_at, updated_at
		FROM open_platform_scope_requests
		WHERE app_id = $1 AND scope = $2
		FOR UPDATE
	`, appID, req.Scope))
	if err != nil {
		return ScopeRequest{}, fmt.Errorf("UpsertScopeRequestsWithAudit lock existing scope: %w", err)
	}
	switch existing.Status {
	case ScopeStatusApproved:
		return ScopeRequest{}, ErrScopeAlreadyApproved
	case ScopeStatusPending:
		return ScopeRequest{}, ErrScopeAlreadyPending
	}

	saved, err = scanScopeRequest(tx.QueryRow(ctx, `
		UPDATE open_platform_scope_requests
		SET reason = $3,
		    status = 'pending',
		    reviewer_user_id = NULL,
		    reviewed_at = NULL,
		    decision_note = NULL,
		    updated_at = NOW()
		WHERE app_id = $1 AND scope = $2
		  AND status NOT IN ('approved', 'pending')
		RETURNING id, app_id, scope, reason, status, reviewer_user_id,
		          reviewed_at, decision_note, created_at, updated_at
	`, appID, req.Scope, req.Reason))
	if err != nil {
		return ScopeRequest{}, fmt.Errorf("UpsertScopeRequestsWithAudit reopen scope: %w", err)
	}
	return saved, nil
}

func (r *Repository) WithdrawScopeRequestWithAudit(
	ctx context.Context,
	appID int64,
	scope string,
	actorUserID int64,
	reason string,
	requestID string,
) (ScopeRequest, error) {
	var withdrawn ScopeRequest
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		updated, err := scanScopeRequest(tx.QueryRow(ctx, `
			UPDATE open_platform_scope_requests
			SET status = $3,
			    reviewer_user_id = NULL,
			    reviewed_at = NOW(),
			    decision_note = NULLIF($4, ''),
			    updated_at = NOW()
			WHERE app_id = $1 AND scope = $2
			  AND status = 'pending'
			RETURNING id, app_id, scope, reason, status, reviewer_user_id,
			          reviewed_at, decision_note, created_at, updated_at
		`, appID, scope, ScopeStatusWithdrawn, reason))
		if err != nil {
			if errors.Is(err, ErrAppNotFound) || errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidScope
			}
			return fmt.Errorf("WithdrawScopeRequestWithAudit update scope: %w", err)
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    actorUserID,
			EventType: "open_platform.scope.withdrawn",
			Scope:     scope,
			RequestID: requestID,
			Metadata: map[string]any{
				"reason": reason,
				"status": ScopeStatusWithdrawn,
			},
		}); err != nil {
			return err
		}
		withdrawn = updated
		return nil
	})
	if err != nil {
		return ScopeRequest{}, err
	}
	return withdrawn, nil
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
		  AND status = 'pending'
	`, appID, scope, status, reviewerUserID, note)
	if err != nil {
		return fmt.Errorf("update scope request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidScope
	}
	return nil
}

func (r *Repository) MarkAppApproved(ctx context.Context, appID int64, clientSecretHash string, reviewerUserID int64, requestID string) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
		UPDATE open_platform_apps
		SET status = $2,
		    client_secret_hash = $3,
		    updated_at = NOW()
		WHERE id = $1 AND status = $4
	`, appID, AppStatusApproved, clientSecretHash, AppStatusPending)
		if err != nil {
			return fmt.Errorf("MarkAppApproved: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrInvalidAppStatus
		}
		if reviewerUserID > 0 {
			if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
				AppID:     appID,
				UserID:    reviewerUserID,
				EventType: "open_platform.app.approved",
				RequestID: requestID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) RotateAppSecret(
	ctx context.Context,
	appID int64,
	clientSecretHash string,
	actorUserID int64,
	actorType string,
	reason string,
	requestID string,
) (*App, error) {
	var app *App
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		updated, err := scanApp(tx.QueryRow(ctx, `
			UPDATE open_platform_apps
			SET client_secret_hash = $2,
			    updated_at = NOW()
			WHERE id = $1
			RETURNING id, casdoor_application_name, owner_user_id, client_id,
			          client_secret_hash, display_name, description, homepage_url,
			          privacy_policy_url, redirect_uris, status, created_at, updated_at
		`, appID, clientSecretHash))
		if err != nil {
			return fmt.Errorf("RotateAppSecret update: %w", err)
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    actorUserID,
			EventType: "open_platform.app.secret_rotated",
			RequestID: requestID,
			Metadata: map[string]any{
				"actorType": actorType,
				"reason":    reason,
			},
		}); err != nil {
			return err
		}
		app = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *Repository) UpdateAppProfileWithAudit(
	ctx context.Context,
	app *App,
	actorUserID int64,
	reason string,
	requestID string,
) (*App, error) {
	var updated *App
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		current, err := scanApp(tx.QueryRow(ctx, `
			SELECT id, casdoor_application_name, owner_user_id, client_id,
			       client_secret_hash, display_name, description, homepage_url,
			       privacy_policy_url, redirect_uris, status, created_at, updated_at
			FROM open_platform_apps
			WHERE id = $1
			FOR UPDATE
		`, app.ID))
		if err != nil {
			return fmt.Errorf("UpdateAppProfile lock app: %w", err)
		}
		if current.Status == AppStatusRevoked {
			return ErrInvalidAppStatus
		}
		saved, err := scanApp(tx.QueryRow(ctx, `
			UPDATE open_platform_apps
			SET display_name = $2,
			    description = $3,
			    homepage_url = $4,
			    privacy_policy_url = $5,
			    updated_at = NOW()
			WHERE id = $1
			RETURNING id, casdoor_application_name, owner_user_id, client_id,
			          client_secret_hash, display_name, description, homepage_url,
			          privacy_policy_url, redirect_uris, status, created_at, updated_at
		`, app.ID, app.DisplayName, app.Description, app.HomepageURL, app.PrivacyPolicyURL))
		if err != nil {
			return fmt.Errorf("UpdateAppProfile update app: %w", err)
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     app.ID,
			UserID:    actorUserID,
			EventType: "open_platform.app.profile_updated",
			RequestID: requestID,
			Metadata: map[string]any{
				"changedFields":    appProfileChangedFields(current, saved),
				"displayName":      saved.DisplayName,
				"homepageURL":      saved.HomepageURL,
				"privacyPolicyURL": saved.PrivacyPolicyURL,
				"reason":           reason,
				"status":           saved.Status,
			},
		}); err != nil {
			return err
		}
		updated = saved
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func appProfileChangedFields(previous, current *App) []string {
	if previous == nil || current == nil {
		return []string{}
	}
	changed := []string{}
	if previous.DisplayName != current.DisplayName {
		changed = append(changed, "displayName")
	}
	if previous.Description != current.Description {
		changed = append(changed, "description")
	}
	if previous.HomepageURL != current.HomepageURL {
		changed = append(changed, "homepageURL")
	}
	if previous.PrivacyPolicyURL != current.PrivacyPolicyURL {
		changed = append(changed, "privacyPolicyURL")
	}
	return changed
}

func (r *Repository) UpsertRedirectURIRequestWithAudit(
	ctx context.Context,
	req RedirectURIRequest,
	actorUserID int64,
	requestID string,
) (RedirectURIRequest, error) {
	redirects, err := json.Marshal(req.RedirectURIs)
	if err != nil {
		return RedirectURIRequest{}, fmt.Errorf("UpsertRedirectURIRequest marshal redirects: %w", err)
	}
	var saved RedirectURIRequest
	err = r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		created, err := scanRedirectURIRequest(tx.QueryRow(ctx, `
			INSERT INTO open_platform_redirect_uri_requests (
				app_id, redirect_uris, reason, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (app_id) WHERE status = 'pending' DO UPDATE SET
				redirect_uris = EXCLUDED.redirect_uris,
				reason = EXCLUDED.reason,
				reviewer_user_id = NULL,
				reviewed_at = NULL,
				decision_note = NULL,
				updated_at = NOW()
			RETURNING id, app_id, redirect_uris, reason, status, reviewer_user_id,
			          reviewed_at, decision_note, created_at, updated_at
		`, req.AppID, redirects, req.Reason, ScopeStatusPending))
		if err != nil {
			return fmt.Errorf("UpsertRedirectURIRequest insert: %w", err)
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     req.AppID,
			UserID:    actorUserID,
			EventType: "open_platform.app.redirect_uris.requested",
			RequestID: requestID,
			Metadata: map[string]any{
				"reason":       req.Reason,
				"redirectURIs": req.RedirectURIs,
			},
		}); err != nil {
			return err
		}
		saved = created
		return nil
	})
	if err != nil {
		return RedirectURIRequest{}, err
	}
	return saved, nil
}

func (r *Repository) ReviewRedirectURIRequestWithAudit(
	ctx context.Context,
	appID int64,
	redirectURIRequestID int64,
	reviewerUserID int64,
	status string,
	decisionNote string,
	eventType string,
	requestID string,
) (RedirectURIRequest, error) {
	var reviewed RedirectURIRequest
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		updated, err := scanRedirectURIRequest(tx.QueryRow(ctx, `
			UPDATE open_platform_redirect_uri_requests
			SET status = $3,
			    reviewer_user_id = $4,
			    reviewed_at = NOW(),
			    decision_note = NULLIF($5, ''),
			    updated_at = NOW()
			WHERE app_id = $1
			  AND id = $2
			  AND status = 'pending'
			RETURNING id, app_id, redirect_uris, reason, status, reviewer_user_id,
			          reviewed_at, decision_note, created_at, updated_at
		`, appID, redirectURIRequestID, status, reviewerUserID, decisionNote))
		if err != nil {
			return fmt.Errorf("ReviewRedirectURIRequest update request: %w", err)
		}
		if status == ScopeStatusApproved {
			redirects, err := json.Marshal(updated.RedirectURIs)
			if err != nil {
				return fmt.Errorf("ReviewRedirectURIRequest marshal redirects: %w", err)
			}
			tag, err := tx.Exec(ctx, `
				UPDATE open_platform_apps
				SET redirect_uris = $2,
				    updated_at = NOW()
				WHERE id = $1
			`, appID, redirects)
			if err != nil {
				return fmt.Errorf("ReviewRedirectURIRequest update app: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return ErrAppNotFound
			}
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    reviewerUserID,
			EventType: eventType,
			RequestID: requestID,
			Metadata: map[string]any{
				"decisionNote": decisionNote,
				"redirectURIs": updated.RedirectURIs,
				"requestID":    redirectURIRequestID,
				"status":       status,
			},
		}); err != nil {
			return err
		}
		reviewed = updated
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrRedirectURIRequestNotFound) {
			return RedirectURIRequest{}, ErrRedirectURIRequestNotFound
		}
		return RedirectURIRequest{}, err
	}
	return reviewed, nil
}

func (r *Repository) WithdrawRedirectURIRequestWithAudit(
	ctx context.Context,
	appID int64,
	redirectURIRequestID int64,
	actorUserID int64,
	reason string,
	requestID string,
) (RedirectURIRequest, error) {
	var withdrawn RedirectURIRequest
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		updated, err := scanRedirectURIRequest(tx.QueryRow(ctx, `
			UPDATE open_platform_redirect_uri_requests
			SET status = $3,
			    reviewer_user_id = NULL,
			    reviewed_at = NOW(),
			    decision_note = NULLIF($4, ''),
			    updated_at = NOW()
			WHERE app_id = $1
			  AND id = $2
			  AND status = 'pending'
			RETURNING id, app_id, redirect_uris, reason, status, reviewer_user_id,
			          reviewed_at, decision_note, created_at, updated_at
		`, appID, redirectURIRequestID, ScopeStatusWithdrawn, reason))
		if err != nil {
			if errors.Is(err, ErrRedirectURIRequestNotFound) || errors.Is(err, pgx.ErrNoRows) {
				return ErrRedirectURIRequestNotFound
			}
			return fmt.Errorf("WithdrawRedirectURIRequestWithAudit update request: %w", err)
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    actorUserID,
			EventType: "open_platform.app.redirect_uris.withdrawn",
			RequestID: requestID,
			Metadata: map[string]any{
				"reason":       reason,
				"redirectURIs": updated.RedirectURIs,
				"requestID":    redirectURIRequestID,
				"status":       ScopeStatusWithdrawn,
			},
		}); err != nil {
			return err
		}
		withdrawn = updated
		return nil
	})
	if err != nil {
		return RedirectURIRequest{}, err
	}
	return withdrawn, nil
}

func (r *Repository) UpdateAppStatusWithAudit(
	ctx context.Context,
	appID int64,
	status string,
	allowedFrom []string,
	actorUserID int64,
	eventType string,
	reason string,
	requestID string,
) (*App, error) {
	var app *App
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		updated, err := scanApp(tx.QueryRow(ctx, `
			UPDATE open_platform_apps
			SET status = $2,
			    updated_at = NOW()
			WHERE id = $1
			  AND status = ANY($3::text[])
			RETURNING id, casdoor_application_name, owner_user_id, client_id,
			          client_secret_hash, display_name, description, homepage_url,
			          privacy_policy_url, redirect_uris, status, created_at, updated_at
		`, appID, status, allowedFrom))
		if err != nil {
			if errors.Is(err, ErrAppNotFound) {
				return ErrInvalidAppStatus
			}
			return fmt.Errorf("UpdateAppStatusWithAudit update: %w", err)
		}
		if status == AppStatusRevoked {
			if err := withdrawPendingChildRequestsForRevokedApp(ctx, tx.Exec, appID, reason); err != nil {
				return err
			}
			if err := revokeActiveConsentsForRevokedApp(
				ctx,
				tx.Exec,
				appID,
				actorUserID,
				lifecycleConsentRevocationActor(eventType),
				reason,
				requestID,
			); err != nil {
				return err
			}
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     appID,
			UserID:    actorUserID,
			EventType: eventType,
			RequestID: requestID,
			Metadata: map[string]any{
				"reason": reason,
				"status": status,
			},
		}); err != nil {
			return err
		}
		app = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return app, nil
}

func withdrawPendingChildRequestsForRevokedApp(
	ctx context.Context,
	exec execFn,
	appID int64,
	reason string,
) error {
	if _, err := exec(ctx, `
		UPDATE open_platform_scope_requests
		SET status = $2,
		    reviewer_user_id = NULL,
		    reviewed_at = NOW(),
		    decision_note = NULLIF($3, ''),
		    updated_at = NOW()
		WHERE app_id = $1
		  AND status = 'pending'
	`, appID, ScopeStatusWithdrawn, reason); err != nil {
		return fmt.Errorf("withdraw pending scope requests for revoked app: %w", err)
	}
	if _, err := exec(ctx, `
		UPDATE open_platform_redirect_uri_requests
		SET status = $2,
		    reviewer_user_id = NULL,
		    reviewed_at = NOW(),
		    decision_note = NULLIF($3, ''),
		    updated_at = NOW()
		WHERE app_id = $1
		  AND status = 'pending'
	`, appID, ScopeStatusWithdrawn, reason); err != nil {
		return fmt.Errorf("withdraw pending redirect URI requests for revoked app: %w", err)
	}
	return nil
}

func revokeActiveConsentsForRevokedApp(
	ctx context.Context,
	exec execFn,
	appID int64,
	actorUserID int64,
	actor string,
	reason string,
	requestID string,
) error {
	if _, err := exec(ctx, `
		WITH revoked AS (
			UPDATE open_platform_user_consents
			SET revoked_at = NOW()
			WHERE app_id = $1
			  AND revoked_at IS NULL
			RETURNING user_id, scope
		)
		INSERT INTO open_platform_audit_events (
			app_id, user_id, event_type, scope, request_id, metadata, created_at
		)
		SELECT
			$1,
			user_id,
			'open_platform.consent.revoked',
			scope,
			NULLIF($2, ''),
			jsonb_build_object(
				'actor', $3::text,
				'actorUserID', $4::bigint,
				'reason', $5::text,
				'source', 'app_lifecycle'
			),
			NOW()
		FROM revoked
	`, appID, requestID, actor, actorUserID, reason); err != nil {
		return fmt.Errorf("revoke active consents for revoked app: %w", err)
	}
	return nil
}

func lifecycleConsentRevocationActor(eventType string) string {
	if eventType == "open_platform.app.withdrawn" {
		return "developer"
	}
	return "admin"
}

func (r *Repository) ListApprovedScopes(ctx context.Context, appID int64) ([]string, error) {
	ctx = withDBTable(ctx, "open_platform_approved_scopes")
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

func (r *Repository) ListScopeReasons(ctx context.Context, appID int64, scopes []string) (map[string]string, error) {
	if len(scopes) == 0 {
		return map[string]string{}, nil
	}
	ctx = withDBTable(ctx, "open_platform_scope_requests")
	rows, err := r.db.Query(ctx, `
		SELECT scope, reason
		FROM open_platform_scope_requests
		WHERE app_id = $1
		  AND scope = ANY($2)
	`, appID, scopes)
	if err != nil {
		return nil, fmt.Errorf("ListScopeReasons: %w", err)
	}
	defer rows.Close()

	reasons := make(map[string]string, len(scopes))
	for rows.Next() {
		var scope string
		var reason string
		if err := rows.Scan(&scope, &reason); err != nil {
			return nil, fmt.Errorf("ListScopeReasons scan: %w", err)
		}
		reasons[scope] = reason
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListScopeReasons rows: %w", err)
	}
	return reasons, nil
}

func execRow(ctx context.Context, exec execFn, sql string, args ...any) error {
	_, err := exec(ctx, sql, args...)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
