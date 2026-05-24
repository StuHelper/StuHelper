package openplatform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type EnsureApprovedAppOptions struct {
	ReviewerUserID     int64
	RequestID          string
	AllowRevokedRepair bool
	AuditEventType     string
}

func (r *Repository) EnsureApprovedApp(
	ctx context.Context,
	app *App,
	scopes []ScopeRequest,
	opts EnsureApprovedAppOptions,
) (*App, error) {
	if app == nil {
		return nil, ErrAppNotFound
	}
	returning := (*App)(nil)
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := ensureUserExistsTx(ctx, tx, app.OwnerUserID, "owner"); err != nil {
			return err
		}
		if opts.ReviewerUserID > 0 {
			if err := ensureUserExistsTx(ctx, tx, opts.ReviewerUserID, "reviewer"); err != nil {
				return err
			}
		}
		existing, err := scanApp(tx.QueryRow(ctx, `
			SELECT id, casdoor_application_name, owner_user_id, client_id,
			       client_secret_hash, display_name, description, homepage_url,
			       privacy_policy_url, redirect_uris, status, created_at, updated_at
			FROM open_platform_apps
			WHERE client_id = $1
			FOR UPDATE
		`, app.ClientID))
		if err != nil && err != ErrAppNotFound {
			return fmt.Errorf("EnsureApprovedApp lock existing app: %w", err)
		}
		if existing != nil && existing.Status == AppStatusRevoked && !opts.AllowRevokedRepair {
			return ErrInvalidAppStatus
		}

		if existing == nil {
			if err := insertAppTx(ctx, tx, app); err != nil {
				return fmt.Errorf("EnsureApprovedApp insert app: %w", err)
			}
			returning = app
		} else {
			updated, err := updateApprovedAppTx(ctx, tx, existing.ID, app)
			if err != nil {
				return err
			}
			returning = updated
		}

		for i := range scopes {
			scopes[i].AppID = returning.ID
			scopes[i].Status = ScopeStatusApproved
			scopes[i].ReviewerUserID = &opts.ReviewerUserID
			if err := upsertApprovedScopeRequest(ctx, tx.Exec, &scopes[i]); err != nil {
				return err
			}
			if err := insertApprovedScope(ctx, tx.Exec, returning.ID, scopes[i].Scope, opts.ReviewerUserID); err != nil {
				return err
			}
		}
		eventType := opts.AuditEventType
		if eventType == "" {
			eventType = "open_platform.app.approved_app_ensured"
		}
		if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
			AppID:     returning.ID,
			UserID:    opts.ReviewerUserID,
			EventType: eventType,
			RequestID: opts.RequestID,
			Metadata: map[string]any{
				"clientID":    returning.ClientID,
				"displayName": returning.DisplayName,
				"scopes":      scopeRequestNames(scopes),
			},
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return returning, nil
}

func ensureUserExistsTx(ctx context.Context, tx pgx.Tx, userID int64, label string) error {
	if userID <= 0 {
		return fmt.Errorf("open platform %s user id is required", label)
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return fmt.Errorf("EnsureApprovedApp validate %s user: %w", label, err)
	}
	if !exists {
		return fmt.Errorf("open platform %s user id %d does not exist", label, userID)
	}
	return nil
}

func updateApprovedAppTx(ctx context.Context, tx pgx.Tx, appID int64, app *App) (*App, error) {
	redirects, err := json.Marshal(app.RedirectURIs)
	if err != nil {
		return nil, fmt.Errorf("EnsureApprovedApp marshal redirects: %w", err)
	}
	updated, err := scanApp(tx.QueryRow(ctx, `
		UPDATE open_platform_apps
		SET casdoor_application_name = $2,
		    owner_user_id = $3,
		    client_secret_hash = $4,
		    display_name = $5,
		    description = $6,
		    homepage_url = $7,
		    privacy_policy_url = $8,
		    redirect_uris = $9,
		    status = $10,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, casdoor_application_name, owner_user_id, client_id,
		          client_secret_hash, display_name, description, homepage_url,
		          privacy_policy_url, redirect_uris, status, created_at, updated_at
	`, appID, app.CasdoorApplicationName, app.OwnerUserID, app.ClientSecretHash,
		app.DisplayName, app.Description, app.HomepageURL, app.PrivacyPolicyURL,
		redirects, AppStatusApproved))
	if err != nil {
		return nil, fmt.Errorf("EnsureApprovedApp update app: %w", err)
	}
	return updated, nil
}

func scopeRequestNames(scopes []ScopeRequest) []string {
	names := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		names = append(names, scope.Scope)
	}
	return names
}
