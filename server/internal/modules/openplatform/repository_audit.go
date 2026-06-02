package openplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type auditEventListFilter struct {
	AppID     int64
	UserID    int64
	EventType string
	Scope     string
	Limit     int
	Offset    int
}

type userConsentAuditEventListFilter struct {
	UserID    int64
	AppID     int64
	EventType string
	Scope     string
	Limit     int
	Offset    int
}

type developerAppAuditEventListFilter struct {
	AppID     int64
	EventType string
	Scope     string
	Limit     int
	Offset    int
}

var userConsentAuditEventTypes = []string{
	"open_platform.consent.granted",
	"open_platform.consent.denied",
	"open_platform.consent.revoked",
	"open_platform.disclosure.granted",
	"open_platform.disclosure.denied",
	"open_platform.disclosure.replay_detected",
}

var developerAppAuditEventTypes = []string{
	"open_platform.app.approved",
	"open_platform.app.approved_app_ensured",
	"open_platform.app.profile_updated",
	"open_platform.app.revoked",
	"open_platform.app.resumed",
	"open_platform.app.secret_rotated",
	"open_platform.app.suspended",
	"open_platform.app.redirect_uris.approved",
	"open_platform.app.redirect_uris.rejected",
	"open_platform.app.redirect_uris.requested",
	"open_platform.app.token_probe.failed",
	"open_platform.app.token_probe.passed",
	"open_platform.app.token_probe.runtime.failed",
	"open_platform.app.token_probe.runtime.passed",
	"open_platform.consent.denied",
	"open_platform.consent.granted",
	"open_platform.consent.revoked",
	"open_platform.disclosure.denied",
	"open_platform.disclosure.granted",
	"open_platform.disclosure.replay_detected",
	"open_platform.resource_access.checked",
	"open_platform.resource_access.granted",
	"open_platform.resource_access.revoked",
	"open_platform.app.withdrawn",
	"open_platform.scope.approved",
	"open_platform.scope.rejected",
	"open_platform.scope.requested",
	"open_platform.scope.withdrawn",
	"open_platform.app.redirect_uris.withdrawn",
}

func (r *Repository) ListAuditEvents(ctx context.Context, filter auditEventListFilter) (AuditEventListResult, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	whereSQL := `
		WHERE ($1::bigint = 0 OR app_id = $1)
		  AND ($2::bigint = 0 OR user_id = $2)
		  AND ($3::text = '' OR event_type = $3)
		  AND ($4::text = '' OR scope = $4)
	`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM open_platform_audit_events
		`+whereSQL+`
	`, filter.AppID, filter.UserID, filter.EventType, filter.Scope).Scan(&total); err != nil {
		return AuditEventListResult{}, fmt.Errorf("ListAuditEvents count: %w", err)
	}
	if total == 0 {
		return AuditEventListResult{List: []AuditEvent{}, Total: 0}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, user_id, event_type, scope, request_id, metadata, created_at
		FROM open_platform_audit_events
		`+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $5 OFFSET $6
	`, filter.AppID, filter.UserID, filter.EventType, filter.Scope, filter.Limit, filter.Offset)
	if err != nil {
		return AuditEventListResult{}, fmt.Errorf("ListAuditEvents query: %w", err)
	}
	defer rows.Close()

	list := make([]AuditEvent, 0, filter.Limit)
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return AuditEventListResult{}, fmt.Errorf("ListAuditEvents scan: %w", err)
		}
		list = append(list, event)
	}
	if err := rows.Err(); err != nil {
		return AuditEventListResult{}, fmt.Errorf("ListAuditEvents rows: %w", err)
	}
	return AuditEventListResult{List: list, Total: total}, nil
}

func (r *Repository) ListUserConsentAuditEvents(
	ctx context.Context,
	filter userConsentAuditEventListFilter,
) (UserConsentAuditEventListResult, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	whereSQL := `
		WHERE e.user_id = $1
		  AND ($2::bigint = 0 OR e.app_id = $2)
		  AND ($3::text = '' OR e.event_type = $3)
		  AND (
		    $4::text = ''
		    OR e.scope = $4
		    OR COALESCE(e.metadata->'scopes', '[]'::jsonb) ? $4
		  )
		  AND e.event_type = ANY($5::text[])
	`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM open_platform_audit_events e
		`+whereSQL+`
	`, filter.UserID, filter.AppID, filter.EventType, filter.Scope, userConsentAuditEventTypes).Scan(&total); err != nil {
		return UserConsentAuditEventListResult{}, fmt.Errorf("ListUserConsentAuditEvents count: %w", err)
	}
	if total == 0 {
		return UserConsentAuditEventListResult{List: []UserConsentAuditEvent{}, Total: 0}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			e.id, e.app_id, a.display_name, a.client_id, e.event_type,
			e.scope, e.request_id, e.metadata, e.created_at
		FROM open_platform_audit_events e
		LEFT JOIN open_platform_apps a ON a.id = e.app_id
		`+whereSQL+`
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $6 OFFSET $7
	`, filter.UserID, filter.AppID, filter.EventType, filter.Scope,
		userConsentAuditEventTypes, filter.Limit, filter.Offset,
	)
	if err != nil {
		return UserConsentAuditEventListResult{}, fmt.Errorf("ListUserConsentAuditEvents query: %w", err)
	}
	defer rows.Close()

	list := make([]UserConsentAuditEvent, 0, filter.Limit)
	for rows.Next() {
		event, err := scanUserConsentAuditEvent(rows)
		if err != nil {
			return UserConsentAuditEventListResult{}, fmt.Errorf("ListUserConsentAuditEvents scan: %w", err)
		}
		list = append(list, event)
	}
	if err := rows.Err(); err != nil {
		return UserConsentAuditEventListResult{}, fmt.Errorf("ListUserConsentAuditEvents rows: %w", err)
	}
	return UserConsentAuditEventListResult{List: list, Total: total}, nil
}

func (r *Repository) ListDeveloperAppAuditEvents(
	ctx context.Context,
	filter developerAppAuditEventListFilter,
) (DeveloperAppAuditEventListResult, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	whereSQL := `
		WHERE e.app_id = $1
		  AND ($2::text = '' OR e.event_type = $2)
		  AND (
		    $3::text = ''
		    OR e.scope = $3
		    OR COALESCE(e.metadata->'scopes', '[]'::jsonb) ? $3
		  )
		  AND e.event_type = ANY($4::text[])
	`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM open_platform_audit_events e
		`+whereSQL+`
	`, filter.AppID, filter.EventType, filter.Scope, developerAppAuditEventTypes).Scan(&total); err != nil {
		return DeveloperAppAuditEventListResult{}, fmt.Errorf("ListDeveloperAppAuditEvents count: %w", err)
	}
	if total == 0 {
		return DeveloperAppAuditEventListResult{List: []DeveloperAppAuditEvent{}, Total: 0}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.event_type, e.scope, e.request_id, e.metadata, e.created_at
		FROM open_platform_audit_events e
		`+whereSQL+`
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $5 OFFSET $6
	`, filter.AppID, filter.EventType, filter.Scope, developerAppAuditEventTypes,
		filter.Limit, filter.Offset,
	)
	if err != nil {
		return DeveloperAppAuditEventListResult{}, fmt.Errorf("ListDeveloperAppAuditEvents query: %w", err)
	}
	defer rows.Close()

	list := make([]DeveloperAppAuditEvent, 0, filter.Limit)
	for rows.Next() {
		event, err := scanDeveloperAppAuditEvent(rows)
		if err != nil {
			return DeveloperAppAuditEventListResult{}, fmt.Errorf("ListDeveloperAppAuditEvents scan: %w", err)
		}
		list = append(list, event)
	}
	if err := rows.Err(); err != nil {
		return DeveloperAppAuditEventListResult{}, fmt.Errorf("ListDeveloperAppAuditEvents rows: %w", err)
	}
	return DeveloperAppAuditEventListResult{List: list, Total: total}, nil
}

func (r *Repository) DisclosureReport(ctx context.Context, windowHours int) (DisclosureReport, error) {
	report := DisclosureReport{}
	summary, err := r.disclosureReportSummary(ctx, windowHours)
	if err != nil {
		return DisclosureReport{}, err
	}
	report.Summary = summary
	endpoints, err := r.disclosureReportEndpoints(ctx, windowHours)
	if err != nil {
		return DisclosureReport{}, err
	}
	report.Endpoints = endpoints
	reasons, err := r.disclosureReportDenialReasons(ctx, windowHours)
	if err != nil {
		return DisclosureReport{}, err
	}
	report.DenialReasons = reasons
	dimensions, err := r.disclosureReportRateLimitDimensions(ctx, windowHours)
	if err != nil {
		return DisclosureReport{}, err
	}
	report.RateLimitDimensions = dimensions
	replayEvents, err := r.disclosureReportReplayEvents(ctx, windowHours)
	if err != nil {
		return DisclosureReport{}, err
	}
	report.RecentReplayEvents = replayEvents
	return report, nil
}

func (r *Repository) disclosureReportSummary(ctx context.Context, windowHours int) (DisclosureReportSummary, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	var summary DisclosureReportSummary
	err := r.db.QueryRow(ctx, `
		SELECT
			$1::int AS window_hours,
			COUNT(*)::int AS total,
			COUNT(*) FILTER (WHERE event_type = 'open_platform.disclosure.granted')::int AS granted,
			COUNT(*) FILTER (WHERE event_type = 'open_platform.disclosure.denied')::int AS denied,
			COUNT(*) FILTER (
				WHERE event_type = 'open_platform.disclosure.denied'
				  AND metadata->>'result' = 'rate_limited'
			)::int AS rate_limited,
			COUNT(*) FILTER (WHERE event_type = 'open_platform.disclosure.replay_detected')::int AS replay_detected
		FROM open_platform_audit_events
		WHERE event_type IN (
			'open_platform.disclosure.granted',
			'open_platform.disclosure.denied',
			'open_platform.disclosure.replay_detected'
		)
		  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
	`, windowHours).Scan(
		&summary.WindowHours,
		&summary.Total,
		&summary.Granted,
		&summary.Denied,
		&summary.RateLimited,
		&summary.ReplayDetected,
	)
	if err != nil {
		return DisclosureReportSummary{}, fmt.Errorf("DisclosureReport summary: %w", err)
	}
	return summary, nil
}

func (r *Repository) disclosureReportEndpoints(ctx context.Context, windowHours int) ([]DisclosureEndpointStats, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	rows, err := r.db.Query(ctx, `
		SELECT
			COALESCE(NULLIF(metadata->>'endpoint', ''), 'userinfo') AS endpoint,
			COUNT(*)::int AS total,
			COUNT(*) FILTER (WHERE event_type = 'open_platform.disclosure.granted')::int AS granted,
			COUNT(*) FILTER (WHERE event_type = 'open_platform.disclosure.denied')::int AS denied,
			COUNT(*) FILTER (
				WHERE event_type = 'open_platform.disclosure.denied'
				  AND metadata->>'result' = 'rate_limited'
			)::int AS rate_limited,
			COUNT(*) FILTER (WHERE event_type = 'open_platform.disclosure.replay_detected')::int AS replay_detected
		FROM open_platform_audit_events
		WHERE event_type IN (
			'open_platform.disclosure.granted',
			'open_platform.disclosure.denied',
			'open_platform.disclosure.replay_detected'
		)
		  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
		GROUP BY endpoint
		ORDER BY total DESC, endpoint ASC
	`, windowHours)
	if err != nil {
		return nil, fmt.Errorf("DisclosureReport endpoints: %w", err)
	}
	defer rows.Close()

	result := []DisclosureEndpointStats{}
	for rows.Next() {
		var item DisclosureEndpointStats
		if err := rows.Scan(
			&item.Endpoint,
			&item.Total,
			&item.Granted,
			&item.Denied,
			&item.RateLimited,
			&item.ReplayDetected,
		); err != nil {
			return nil, fmt.Errorf("DisclosureReport endpoints scan: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("DisclosureReport endpoints rows: %w", err)
	}
	return result, nil
}

func (r *Repository) disclosureReportDenialReasons(ctx context.Context, windowHours int) ([]DisclosureReasonStats, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(NULLIF(metadata->>'result', ''), 'unknown') AS reason,
		       COUNT(*)::int AS total
		FROM open_platform_audit_events
		WHERE event_type = 'open_platform.disclosure.denied'
		  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
		GROUP BY reason
		ORDER BY total DESC, reason ASC
		LIMIT 20
	`, windowHours)
	if err != nil {
		return nil, fmt.Errorf("DisclosureReport denial reasons: %w", err)
	}
	defer rows.Close()

	result := []DisclosureReasonStats{}
	for rows.Next() {
		var item DisclosureReasonStats
		if err := rows.Scan(&item.Reason, &item.Total); err != nil {
			return nil, fmt.Errorf("DisclosureReport denial reasons scan: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("DisclosureReport denial reasons rows: %w", err)
	}
	return result, nil
}

func (r *Repository) disclosureReportRateLimitDimensions(ctx context.Context, windowHours int) ([]DisclosureRateLimitStats, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(NULLIF(metadata->>'rateLimitDimension', ''), 'unknown') AS dimension,
		       COUNT(*)::int AS total
		FROM open_platform_audit_events
		WHERE event_type = 'open_platform.disclosure.denied'
		  AND metadata->>'result' = 'rate_limited'
		  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
		GROUP BY dimension
		ORDER BY total DESC, dimension ASC
	`, windowHours)
	if err != nil {
		return nil, fmt.Errorf("DisclosureReport rate-limit dimensions: %w", err)
	}
	defer rows.Close()

	result := []DisclosureRateLimitStats{}
	for rows.Next() {
		var item DisclosureRateLimitStats
		if err := rows.Scan(&item.Dimension, &item.Total); err != nil {
			return nil, fmt.Errorf("DisclosureReport rate-limit dimensions scan: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("DisclosureReport rate-limit dimensions rows: %w", err)
	}
	return result, nil
}

func (r *Repository) disclosureReportReplayEvents(ctx context.Context, windowHours int) ([]DisclosureReplayEvent, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, user_id, request_id, metadata, created_at
		FROM open_platform_audit_events
		WHERE event_type = 'open_platform.disclosure.replay_detected'
		  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
		ORDER BY created_at DESC, id DESC
		LIMIT 20
	`, windowHours)
	if err != nil {
		return nil, fmt.Errorf("DisclosureReport replay events: %w", err)
	}
	defer rows.Close()

	result := []DisclosureReplayEvent{}
	for rows.Next() {
		var event DisclosureReplayEvent
		var metadata []byte
		if err := rows.Scan(
			&event.ID,
			&event.AppID,
			&event.UserID,
			&event.RequestID,
			&metadata,
			&event.DetectedAt,
		); err != nil {
			return nil, fmt.Errorf("DisclosureReport replay events scan: %w", err)
		}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
				return nil, fmt.Errorf("DisclosureReport replay events metadata: %w", err)
			}
		}
		if event.Metadata == nil {
			event.Metadata = map[string]any{}
		}
		event.Endpoint = metadataString(event.Metadata, "endpoint", disclosureEndpointUserInfo)
		event.Result = metadataString(event.Metadata, "result", "unknown")
		event.Count = metadataInt(event.Metadata, "count")
		event.Scopes = metadataStringSlice(event.Metadata, "scopes")
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("DisclosureReport replay events rows: %w", err)
	}
	return result, nil
}

func scanAuditEvent(row rowScanner) (AuditEvent, error) {
	var event AuditEvent
	var metadata []byte
	if err := row.Scan(
		&event.ID,
		&event.AppID,
		&event.UserID,
		&event.EventType,
		&event.Scope,
		&event.RequestID,
		&metadata,
		&event.CreatedAt,
	); err != nil {
		return AuditEvent{}, err
	}
	if len(metadata) == 0 {
		event.Metadata = map[string]any{}
		return event, nil
	}
	if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
		return AuditEvent{}, fmt.Errorf("unmarshal audit metadata: %w", err)
	}
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	return event, nil
}

func scanUserConsentAuditEvent(row rowScanner) (UserConsentAuditEvent, error) {
	var event UserConsentAuditEvent
	var metadata []byte
	if err := row.Scan(
		&event.ID,
		&event.AppID,
		&event.AppDisplayName,
		&event.ClientID,
		&event.EventType,
		&event.Scope,
		&event.RequestID,
		&metadata,
		&event.CreatedAt,
	); err != nil {
		return UserConsentAuditEvent{}, err
	}
	rawDetails := map[string]any{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &rawDetails); err != nil {
			return UserConsentAuditEvent{}, fmt.Errorf("unmarshal user consent audit metadata: %w", err)
		}
	}
	event.Scopes = userConsentAuditScopes(event.Scope, rawDetails)
	event.Endpoint = stringMetadataPointer(rawDetails, "endpoint")
	event.Result = stringMetadataPointer(rawDetails, "result")
	event.Details = sanitizeUserConsentAuditDetails(rawDetails, event.Scopes)
	return event, nil
}

func scanDeveloperAppAuditEvent(row rowScanner) (DeveloperAppAuditEvent, error) {
	var event DeveloperAppAuditEvent
	var metadata []byte
	if err := row.Scan(
		&event.ID,
		&event.EventType,
		&event.Scope,
		&event.RequestID,
		&metadata,
		&event.CreatedAt,
	); err != nil {
		return DeveloperAppAuditEvent{}, err
	}
	rawDetails := map[string]any{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &rawDetails); err != nil {
			return DeveloperAppAuditEvent{}, fmt.Errorf("unmarshal developer app audit metadata: %w", err)
		}
	}
	event.Scopes = auditScopes(event.Scope, rawDetails)
	event.Endpoint = stringMetadataPointer(rawDetails, "endpoint")
	event.Result = firstStringMetadataPointer(rawDetails, "result", "reason")
	event.Details = sanitizeDeveloperAppAuditDetails(rawDetails, event.Scopes)
	return event, nil
}

func userConsentAuditScopes(scope *string, metadata map[string]any) []string {
	return auditScopes(scope, metadata)
}

func auditScopes(scope *string, metadata map[string]any) []string {
	seen := map[string]struct{}{}
	scopes := make([]string, 0)
	add := func(value string) {
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		scopes = append(scopes, value)
	}
	if scope != nil {
		add(*scope)
	}
	switch values := metadata["scopes"].(type) {
	case []any:
		for _, value := range values {
			if scopeValue, ok := value.(string); ok {
				add(scopeValue)
			}
		}
	case []string:
		for _, scopeValue := range values {
			add(scopeValue)
		}
	}
	return scopes
}

func stringMetadataPointer(metadata map[string]any, key string) *string {
	value, ok := metadata[key].(string)
	if !ok || value == "" {
		return nil
	}
	return &value
}

func firstStringMetadataPointer(metadata map[string]any, keys ...string) *string {
	for _, key := range keys {
		if value := stringMetadataPointer(metadata, key); value != nil {
			return value
		}
	}
	return nil
}

func sanitizeUserConsentAuditDetails(metadata map[string]any, scopes []string) map[string]any {
	details := map[string]any{}
	for _, key := range []string{
		"grantSource",
		"actor",
		"endpoint",
		"reason",
		"result",
		"source",
		"rateLimitDimension",
	} {
		if value, ok := metadata[key]; ok {
			details[key] = value
		}
	}
	if len(scopes) > 0 {
		details["scopes"] = append([]string(nil), scopes...)
	}
	return details
}

func sanitizeDeveloperAppAuditDetails(metadata map[string]any, scopes []string) map[string]any {
	details := map[string]any{}
	for _, key := range []string{
		"action",
		"actions",
		"actor",
		"actorType",
		"allowed",
		"businessClaims",
		"casdoorApplicationName",
		"changedFields",
		"clientID",
		"decisionNote",
		"displayName",
		"endpoint",
		"grantSource",
		"homepageURL",
		"inspectedClaims",
		"probeMethod",
		"probeType",
		"privacyPolicyURL",
		"rateLimitDimension",
		"reason",
		"redirectURI",
		"redirectURIs",
		"relation",
		"resourceID",
		"resourceType",
		"result",
		"source",
		"status",
	} {
		if value, ok := metadata[key]; ok {
			details[key] = value
		}
	}
	if len(scopes) > 0 {
		details["scopes"] = append([]string(nil), scopes...)
	}
	return details
}

func metadataString(metadata map[string]any, key string, fallback string) string {
	if value, ok := metadata[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

func metadataInt(metadata map[string]any, key string) int {
	switch value := metadata[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case json.Number:
		n, err := value.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	case string:
		n, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func metadataStringSlice(metadata map[string]any, key string) []string {
	raw, ok := metadata[key]
	if !ok {
		return []string{}
	}
	switch value := raw.(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return []string{}
	}
}
