package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

const (
	defaultAuditCategory   = "audit"
	adminOperationCategory = "admin_operation"
)

type AdminOperationRecord struct {
	ID            string
	ActorUserID   string
	ActorUsername string
	Action        string
	ResourceType  string
	ResourceID    string
	BeforeData    json.RawMessage
	AfterData     json.RawMessage
	IPAddress     string
	UserAgent     string
	CreatedAt     time.Time
}

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("audit.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func (r *Repository) WriteEvent(ctx context.Context, event Event) error {
	normalized, beforeData, afterData, details, err := normalizePersistedEvent(event)
	if err != nil {
		return err
	}
	newID, err := id.New()
	if err != nil {
		return fmt.Errorf("generate audit event id: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_events (
			id, category, event_type, actor_type, actor_user_id, actor_username,
			action, resource_type, resource_id, scope_school_id,
			before_data, after_data, result, reason, trace_id, request_id,
			ip_address, user_agent, details, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20
		)
	`,
		newID, normalized.Category, normalized.Type, normalized.ActorType, normalized.UserID, normalized.Username,
		normalized.Action, normalized.ResourceType, normalized.ResourceID, normalized.ScopeSchoolID,
		beforeData, afterData, normalized.Result, normalized.Reason, normalized.TraceID, normalized.RequestID,
		normalized.IP, normalized.UserAgent, details, normalized.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func (r *Repository) ListAdminOperations(ctx context.Context, limit, offset int) ([]AdminOperationRecord, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, actor_user_id, actor_username, action, resource_type, resource_id,
		       before_data, after_data, ip_address, user_agent, created_at,
		       COUNT(*) OVER() AS total
		FROM audit_events
		WHERE category = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, adminOperationCategory, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin operations: %w", err)
	}
	defer rows.Close()

	items := make([]AdminOperationRecord, 0, limit)
	total := 0
	for rows.Next() {
		var item AdminOperationRecord
		if err := rows.Scan(
			&item.ID, &item.ActorUserID, &item.ActorUsername, &item.Action, &item.ResourceType, &item.ResourceID,
			&item.BeforeData, &item.AfterData, &item.IPAddress, &item.UserAgent, &item.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan admin operation: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) CleanupAdminOperations(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = defaultAdminOperationRetentionDays
	}
	result, err := r.db.Exec(ctx, `
		DELETE FROM audit_events
		WHERE category = $1
		  AND event_type NOT LIKE 'iam.%'
		  AND created_at < NOW() - make_interval(days => $2)
	`, adminOperationCategory, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("cleanup admin operations: %w", err)
	}
	return result.RowsAffected(), nil
}

var (
	globalRepoMu sync.RWMutex
	globalRepo   *Repository
)

func ConfigureRepository(repo *Repository) {
	globalRepoMu.Lock()
	defer globalRepoMu.Unlock()
	globalRepo = repo
}

func configuredRepository() *Repository {
	globalRepoMu.RLock()
	defer globalRepoMu.RUnlock()
	return globalRepo
}

func normalizePersistedEvent(event Event) (Event, []byte, []byte, []byte, error) {
	event = normalizeEvent(event)

	beforeData, err := marshalOptional(event.Before)
	if err != nil {
		return Event{}, nil, nil, nil, fmt.Errorf("marshal audit before data: %w", err)
	}
	afterData, err := marshalOptional(event.After)
	if err != nil {
		return Event{}, nil, nil, nil, fmt.Errorf("marshal audit after data: %w", err)
	}
	details, err := marshalDetails(event.Details)
	if err != nil {
		return Event{}, nil, nil, nil, fmt.Errorf("marshal audit details: %w", err)
	}
	return event, beforeData, afterData, details, nil
}

func normalizeEvent(event Event) Event {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	event.Category = strings.TrimSpace(event.Category)
	if event.Category == "" {
		event.Category = defaultAuditCategory
	}
	event.ActorType = strings.TrimSpace(event.ActorType)
	if event.ActorType == "" {
		if event.UserID == "" {
			event.ActorType = "system"
		} else {
			event.ActorType = "user"
		}
	}
	event.ResourceType = strings.TrimSpace(event.ResourceType)
	if event.ResourceType == "" {
		event.ResourceType = strings.TrimSpace(event.Resource)
	}
	event.ResourceID = strings.TrimSpace(event.ResourceID)
	event.Action = strings.TrimSpace(event.Action)
	if event.Action == "" {
		event.Action = "unknown"
	}
	event.Result = strings.TrimSpace(event.Result)
	if event.Result == "" {
		event.Result = "success"
	}
	return event
}

func marshalOptional(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func marshalDetails(details map[string]any) ([]byte, error) {
	if len(details) == 0 {
		return []byte(`{}`), nil
	}
	return json.Marshal(details)
}
