package openplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type adminUserConsentListFilter struct {
	AppID  int64
	UserID int64
	Limit  int
	Offset int
}

func (r *Repository) HasActiveConsents(ctx context.Context, appID, userID int64, scopes []string) (bool, error) {
	if len(scopes) == 0 {
		return false, ErrInvalidScope
	}
	ctx = withDBTable(ctx, "open_platform_user_consents")
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT scope)
		FROM open_platform_user_consents
		WHERE app_id = $1
		  AND user_id = $2
		  AND scope = ANY($3)
		  AND revoked_at IS NULL
	`, appID, userID, scopes).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("HasActiveConsents: %w", err)
	}
	return count == len(scopes), nil
}

func (r *Repository) ActiveConsentFingerprint(ctx context.Context, appID, userID int64, scopes []string) (string, bool, error) {
	if len(scopes) == 0 {
		return "", false, ErrInvalidScope
	}
	ctx = withDBTable(ctx, "open_platform_user_consents")
	rows, err := r.db.Query(ctx, `
		SELECT scope, granted_at
		FROM open_platform_user_consents
		WHERE app_id = $1
		  AND user_id = $2
		  AND scope = ANY($3)
		  AND revoked_at IS NULL
		ORDER BY scope ASC
	`, appID, userID, scopes)
	if err != nil {
		return "", false, fmt.Errorf("ActiveConsentFingerprint: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]struct{}, len(scopes))
	var builder strings.Builder
	for rows.Next() {
		var scope string
		var grantedAt time.Time
		if err := rows.Scan(&scope, &grantedAt); err != nil {
			return "", false, fmt.Errorf("ActiveConsentFingerprint scan: %w", err)
		}
		seen[scope] = struct{}{}
		builder.WriteString(scope)
		builder.WriteByte(0)
		builder.WriteString(grantedAt.UTC().Format(time.RFC3339Nano))
		builder.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		return "", false, fmt.Errorf("ActiveConsentFingerprint rows: %w", err)
	}
	if len(seen) != len(scopes) {
		return "", false, nil
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:]), true, nil
}

func (r *Repository) GrantConsents(ctx context.Context, consent Consent, scopes []string) error {
	if len(scopes) == 0 {
		return ErrInvalidScope
	}
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, scope := range scopes {
			if _, err := tx.Exec(ctx, `
				INSERT INTO open_platform_user_consents (
					app_id, user_id, scope, granted_at, revoked_at, grant_source,
					request_id
				) VALUES ($1, $2, $3, NOW(), NULL, $4, $5)
				ON CONFLICT (app_id, user_id, scope) DO UPDATE SET
					granted_at = NOW(),
					revoked_at = NULL,
					grant_source = EXCLUDED.grant_source,
					request_id = EXCLUDED.request_id
			`, consent.AppID, consent.UserID, scope,
				nonBlankOrDefault(consent.GrantSource, "web"), consent.RequestID,
			); err != nil {
				return fmt.Errorf("GrantConsents scope %s: %w", scope, err)
			}
			if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
				AppID:     consent.AppID,
				UserID:    consent.UserID,
				EventType: "open_platform.consent.granted",
				Scope:     scope,
				RequestID: consent.RequestID,
				Metadata: map[string]any{
					"grantSource": nonBlankOrDefault(consent.GrantSource, "web"),
				},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) RevokeAppConsents(ctx context.Context, appID, userID int64, scopes []string, requestID string) error {
	return r.RevokeAppConsentsWithAuditMetadata(ctx, appID, userID, scopes, requestID, map[string]any{
		"actor": "user",
	})
}

func (r *Repository) RevokeAppConsentsWithAuditMetadata(
	ctx context.Context,
	appID, userID int64,
	scopes []string,
	requestID string,
	metadata map[string]any,
) error {
	if len(scopes) == 0 {
		return r.revokeAllAppConsents(ctx, appID, userID, requestID, metadata)
	}
	return r.revokeMatchingAppConsents(ctx, appID, userID, scopes, requestID, metadata)
}

func (r *Repository) revokeAllAppConsents(ctx context.Context, appID, userID int64, requestID string, metadata map[string]any) error {
	return r.revokeMatchingAppConsents(ctx, appID, userID, nil, requestID, metadata)
}

func (r *Repository) revokeMatchingAppConsents(
	ctx context.Context,
	appID, userID int64,
	scopes []string,
	requestID string,
	metadata map[string]any,
) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		revokedScopes, err := revokeConsentRows(ctx, tx, appID, userID, scopes)
		if err != nil {
			return err
		}
		if len(revokedScopes) == 0 {
			return ErrConsentRequired
		}
		for _, scope := range revokedScopes {
			if err := insertAuditEvent(ctx, tx.Exec, auditEvent{
				AppID:     appID,
				UserID:    userID,
				EventType: "open_platform.consent.revoked",
				Scope:     scope,
				RequestID: requestID,
				Metadata:  cloneAuditMetadata(metadata),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func revokeConsentRows(ctx context.Context, tx pgx.Tx, appID, userID int64, scopes []string) ([]string, error) {
	sql := `
		UPDATE open_platform_user_consents
		SET revoked_at = NOW()
		WHERE app_id = $1
		  AND user_id = $2
		  AND revoked_at IS NULL
	`
	args := []any{appID, userID}
	if len(scopes) > 0 {
		sql += ` AND scope = ANY($3)`
		args = append(args, scopes)
	}
	sql += ` RETURNING scope`

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("RevokeAppConsents: %w", err)
	}
	defer rows.Close()

	revokedScopes := make([]string, 0, len(scopes))
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, fmt.Errorf("RevokeAppConsents scan: %w", err)
		}
		revokedScopes = append(revokedScopes, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("RevokeAppConsents rows: %w", err)
	}
	return revokedScopes, nil
}

func cloneAuditMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func (r *Repository) ListUserConsents(ctx context.Context, userID int64) ([]UserAuthorizedApp, error) {
	ctx = withDBTable(ctx, "open_platform_user_consents")
	rows, err := r.db.Query(ctx, `
		SELECT a.id,
		       a.casdoor_application_name,
		       a.owner_user_id,
		       a.client_id,
		       a.client_secret_hash,
		       a.display_name,
		       a.description,
		       a.homepage_url,
		       a.privacy_policy_url,
		       a.redirect_uris,
		       a.status,
		       a.created_at,
		       a.updated_at,
		       c.scope,
		       c.granted_at,
		       usage.last_used_at,
		       c.grant_source,
		       c.request_id,
		       COALESCE(sr.reason, '') AS reason
		FROM open_platform_user_consents c
		JOIN open_platform_apps a ON a.id = c.app_id
		LEFT JOIN open_platform_scope_requests sr
		  ON sr.app_id = c.app_id
		 AND sr.scope = c.scope
		LEFT JOIN LATERAL (
			SELECT e.created_at AS last_used_at
			FROM open_platform_audit_events e
			WHERE e.app_id = c.app_id
			  AND e.user_id = c.user_id
			  AND e.event_type = 'open_platform.disclosure.granted'
			  AND (e.metadata -> 'scopes') ? c.scope
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT 1
		) usage ON true
		WHERE c.user_id = $1
		  AND c.revoked_at IS NULL
		ORDER BY lower(a.display_name) ASC, a.id ASC, c.scope ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("ListUserConsents: %w", err)
	}
	defer rows.Close()

	byApp := map[int64]*UserAuthorizedApp{}
	order := make([]int64, 0)
	for rows.Next() {
		var app App
		var redirects []byte
		var scope UserConsentScope
		if err := rows.Scan(&app.ID, &app.CasdoorApplicationName, &app.OwnerUserID,
			&app.ClientID, &app.ClientSecretHash, &app.DisplayName, &app.Description,
			&app.HomepageURL, &app.PrivacyPolicyURL, &redirects, &app.Status,
			&app.CreatedAt, &app.UpdatedAt, &scope.Scope, &scope.GrantedAt, &scope.LastUsedAt,
			&scope.GrantSource, &scope.RequestID, &scope.Reason); err != nil {
			return nil, fmt.Errorf("ListUserConsents scan: %w", err)
		}
		item, ok := byApp[app.ID]
		if !ok {
			if err := json.Unmarshal(redirects, &app.RedirectURIs); err != nil {
				return nil, fmt.Errorf("ListUserConsents unmarshal redirect_uris: %w", err)
			}
			appCopy := app
			item = &UserAuthorizedApp{App: &appCopy}
			byApp[app.ID] = item
			order = append(order, app.ID)
		}
		item.Scopes = append(item.Scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListUserConsents rows: %w", err)
	}

	result := make([]UserAuthorizedApp, 0, len(order))
	for _, appID := range order {
		result = append(result, *byApp[appID])
	}
	return result, nil
}

func (r *Repository) ListAdminUserConsents(ctx context.Context, filter adminUserConsentListFilter) (AdminUserConsentListResult, error) {
	ctx = withDBTable(ctx, "open_platform_user_consents")
	whereSQL := `
		WHERE ($1::bigint = 0 OR c.app_id = $1)
		  AND ($2::bigint = 0 OR c.user_id = $2)
		  AND c.revoked_at IS NULL
	`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT c.user_id, c.app_id
			FROM open_platform_user_consents c
			`+whereSQL+`
			GROUP BY c.user_id, c.app_id
		) grouped
	`, filter.AppID, filter.UserID).Scan(&total); err != nil {
		return AdminUserConsentListResult{}, fmt.Errorf("ListAdminUserConsents count: %w", err)
	}
	if total == 0 {
		return AdminUserConsentListResult{List: []AdminUserAuthorizedApp{}, Total: 0}, nil
	}

	rows, err := r.db.Query(ctx, `
		WITH selected AS (
			SELECT c.user_id, c.app_id
			FROM open_platform_user_consents c
			JOIN open_platform_apps a ON a.id = c.app_id
			`+whereSQL+`
			GROUP BY c.user_id, c.app_id, lower(a.display_name), a.id
			ORDER BY c.user_id ASC, lower(a.display_name) ASC, a.id ASC
			LIMIT $3 OFFSET $4
		)
		SELECT selected.user_id,
		       a.id,
		       a.casdoor_application_name,
		       a.owner_user_id,
		       a.client_id,
		       a.client_secret_hash,
		       a.display_name,
		       a.description,
		       a.homepage_url,
		       a.privacy_policy_url,
		       a.redirect_uris,
		       a.status,
		       a.created_at,
		       a.updated_at,
		       c.scope,
		       c.granted_at,
		       usage.last_used_at,
		       c.grant_source,
		       c.request_id,
		       COALESCE(sr.reason, '') AS reason
		FROM selected
		JOIN open_platform_user_consents c
		  ON c.user_id = selected.user_id
		 AND c.app_id = selected.app_id
		 AND c.revoked_at IS NULL
		JOIN open_platform_apps a ON a.id = c.app_id
		LEFT JOIN open_platform_scope_requests sr
		  ON sr.app_id = c.app_id
		 AND sr.scope = c.scope
		LEFT JOIN LATERAL (
			SELECT e.created_at AS last_used_at
			FROM open_platform_audit_events e
			WHERE e.app_id = c.app_id
			  AND e.user_id = c.user_id
			  AND e.event_type = 'open_platform.disclosure.granted'
			  AND (e.metadata -> 'scopes') ? c.scope
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT 1
		) usage ON true
		ORDER BY selected.user_id ASC, lower(a.display_name) ASC, a.id ASC, c.scope ASC
	`, filter.AppID, filter.UserID, filter.Limit, filter.Offset)
	if err != nil {
		return AdminUserConsentListResult{}, fmt.Errorf("ListAdminUserConsents: %w", err)
	}
	defer rows.Close()

	byEntry := map[string]*AdminUserAuthorizedApp{}
	order := make([]string, 0)
	for rows.Next() {
		var app App
		var redirects []byte
		var scope UserConsentScope
		var userID int64
		if err := rows.Scan(&userID, &app.ID, &app.CasdoorApplicationName, &app.OwnerUserID,
			&app.ClientID, &app.ClientSecretHash, &app.DisplayName, &app.Description,
			&app.HomepageURL, &app.PrivacyPolicyURL, &redirects, &app.Status,
			&app.CreatedAt, &app.UpdatedAt, &scope.Scope, &scope.GrantedAt, &scope.LastUsedAt,
			&scope.GrantSource, &scope.RequestID, &scope.Reason); err != nil {
			return AdminUserConsentListResult{}, fmt.Errorf("ListAdminUserConsents scan: %w", err)
		}
		key := fmt.Sprintf("%d:%d", userID, app.ID)
		item, ok := byEntry[key]
		if !ok {
			if err := json.Unmarshal(redirects, &app.RedirectURIs); err != nil {
				return AdminUserConsentListResult{}, fmt.Errorf("ListAdminUserConsents unmarshal redirect_uris: %w", err)
			}
			appCopy := app
			item = &AdminUserAuthorizedApp{UserID: userID, App: &appCopy}
			byEntry[key] = item
			order = append(order, key)
		}
		item.Scopes = append(item.Scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return AdminUserConsentListResult{}, fmt.Errorf("ListAdminUserConsents rows: %w", err)
	}

	result := make([]AdminUserAuthorizedApp, 0, len(order))
	for _, key := range order {
		result = append(result, *byEntry[key])
	}
	return AdminUserConsentListResult{List: result, Total: total}, nil
}

func nonBlankOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

type auditEvent struct {
	AppID     int64
	UserID    int64
	EventType string
	Scope     string
	RequestID string
	Metadata  map[string]any
}

func (r *Repository) RecordAuditEvent(ctx context.Context, event auditEvent) error {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	return insertAuditEvent(ctx, r.db.Exec, event)
}

func insertAuditEvent(ctx context.Context, exec execFn, event auditEvent) error {
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("open platform audit metadata: %w", err)
	}
	return execRow(ctx, exec, `
		INSERT INTO open_platform_audit_events (
			app_id, user_id, event_type, scope, request_id, metadata, created_at
		) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6::jsonb, NOW())
	`, nullableAuditID(event.AppID), nullableAuditID(event.UserID), event.EventType, event.Scope,
		strings.TrimSpace(event.RequestID), string(rawMetadata))
}

func nullableAuditID(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
