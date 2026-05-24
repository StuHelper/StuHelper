package openplatform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type appListFilter struct {
	OwnerUserID int64
	Status      string
	Limit       int
	Offset      int
}

func (r *Repository) ListApps(ctx context.Context, filter appListFilter) (AppListResult, error) {
	ctx = withDBTable(ctx, "open_platform_apps")
	whereSQL := "WHERE ($1::bigint = 0 OR owner_user_id = $1) AND ($2::text = '' OR status = $2)"

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM open_platform_apps
		`+whereSQL+`
	`, filter.OwnerUserID, filter.Status).Scan(&total); err != nil {
		return AppListResult{}, fmt.Errorf("ListApps count: %w", err)
	}
	if total == 0 {
		return AppListResult{List: []AppWithScopes{}, Total: 0}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, casdoor_application_name, owner_user_id, client_id,
		       client_secret_hash, display_name, description, homepage_url,
		       privacy_policy_url, redirect_uris, status, created_at, updated_at
		FROM open_platform_apps
		`+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, filter.OwnerUserID, filter.Status, filter.Limit, filter.Offset)
	if err != nil {
		return AppListResult{}, fmt.Errorf("ListApps query: %w", err)
	}
	defer rows.Close()

	list := make([]AppWithScopes, 0, filter.Limit)
	appIDs := make([]int64, 0, filter.Limit)
	appIndex := make(map[int64]int, filter.Limit)
	for rows.Next() {
		app, err := scanApp(rows)
		if err != nil {
			return AppListResult{}, fmt.Errorf("ListApps scan app: %w", err)
		}
		appIndex[app.ID] = len(list)
		appIDs = append(appIDs, app.ID)
		list = append(list, AppWithScopes{
			App:                 app,
			Scopes:              []ScopeRequest{},
			RedirectURIRequests: []RedirectURIRequest{},
		})
	}
	if err := rows.Err(); err != nil {
		return AppListResult{}, fmt.Errorf("ListApps rows: %w", err)
	}

	if err := r.loadScopeRequests(ctx, appIDs, list, appIndex); err != nil {
		return AppListResult{}, err
	}
	if err := r.loadRedirectURIRequests(ctx, appIDs, list, appIndex); err != nil {
		return AppListResult{}, err
	}
	return AppListResult{List: list, Total: total}, nil
}

func (r *Repository) loadScopeRequests(
	ctx context.Context,
	appIDs []int64,
	list []AppWithScopes,
	appIndex map[int64]int,
) error {
	if len(appIDs) == 0 {
		return nil
	}
	ctx = withDBTable(ctx, "open_platform_scope_requests")
	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, scope, reason, status, reviewer_user_id,
		       reviewed_at, decision_note, created_at, updated_at
		FROM open_platform_scope_requests
		WHERE app_id = ANY($1::bigint[])
		ORDER BY app_id ASC, created_at ASC, scope ASC
	`, appIDs)
	if err != nil {
		return fmt.Errorf("ListApps query scopes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		scope, err := scanScopeRequest(rows)
		if err != nil {
			return fmt.Errorf("ListApps scan scope: %w", err)
		}
		index, ok := appIndex[scope.AppID]
		if !ok {
			continue
		}
		list[index].Scopes = append(list[index].Scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ListApps scope rows: %w", err)
	}
	return nil
}

func (r *Repository) loadRedirectURIRequests(
	ctx context.Context,
	appIDs []int64,
	list []AppWithScopes,
	appIndex map[int64]int,
) error {
	if len(appIDs) == 0 {
		return nil
	}
	ctx = withDBTable(ctx, "open_platform_redirect_uri_requests")
	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, redirect_uris, reason, status, reviewer_user_id,
		       reviewed_at, decision_note, created_at, updated_at
		FROM open_platform_redirect_uri_requests
		WHERE app_id = ANY($1::bigint[])
		ORDER BY app_id ASC, created_at DESC, id DESC
	`, appIDs)
	if err != nil {
		return fmt.Errorf("ListApps query redirect URI requests: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		req, err := scanRedirectURIRequest(rows)
		if err != nil {
			return fmt.Errorf("ListApps scan redirect URI request: %w", err)
		}
		index, ok := appIndex[req.AppID]
		if !ok {
			continue
		}
		list[index].RedirectURIRequests = append(list[index].RedirectURIRequests, req)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ListApps redirect URI request rows: %w", err)
	}
	return nil
}

func scanScopeRequest(row pgx.Row) (ScopeRequest, error) {
	var req ScopeRequest
	if err := row.Scan(
		&req.ID,
		&req.AppID,
		&req.Scope,
		&req.Reason,
		&req.Status,
		&req.ReviewerUserID,
		&req.ReviewedAt,
		&req.DecisionNote,
		&req.CreatedAt,
		&req.UpdatedAt,
	); err != nil {
		return ScopeRequest{}, err
	}
	return req, nil
}

func scanRedirectURIRequest(row pgx.Row) (RedirectURIRequest, error) {
	var req RedirectURIRequest
	var redirects []byte
	if err := row.Scan(
		&req.ID,
		&req.AppID,
		&redirects,
		&req.Reason,
		&req.Status,
		&req.ReviewerUserID,
		&req.ReviewedAt,
		&req.DecisionNote,
		&req.CreatedAt,
		&req.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return RedirectURIRequest{}, ErrRedirectURIRequestNotFound
		}
		return RedirectURIRequest{}, err
	}
	if err := json.Unmarshal(redirects, &req.RedirectURIs); err != nil {
		return RedirectURIRequest{}, fmt.Errorf("unmarshal redirect URI request redirect_uris: %w", err)
	}
	return req, nil
}
