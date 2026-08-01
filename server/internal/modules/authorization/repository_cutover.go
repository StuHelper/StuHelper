package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
	"github.com/StuHelper/StuHelper/server/internal/pkg/id"
)

const (
	authorityCutoverTable  = "authorization_authority_cutover"
	authorityCutoverReason = "Imported from verified Casdoor and OpenFGA authority cutover snapshot"
)

var authorityCutoverAuditActor = grantAuditActor{
	actorType: "system",
	userID:    "authorization-authority-cutover",
}

func (r *Repository) ListAuthorityCutoverSchoolIDs(ctx context.Context) ([]int64, error) {
	ctx = db.WithTableHint(ctx, "schools")
	rows, err := r.db.Query(ctx, `SELECT id FROM schools ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list authorization cutover schools: %w", err)
	}
	defer rows.Close()

	schools := make([]int64, 0)
	for rows.Next() {
		var schoolID int64
		if err := rows.Scan(&schoolID); err != nil {
			return nil, fmt.Errorf("scan authorization cutover school: %w", err)
		}
		schools = append(schools, schoolID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list authorization cutover school rows: %w", err)
	}
	return schools, nil
}

func (r *Repository) AuthorityCutoverStatus(ctx context.Context) (AuthorityCutoverStatus, error) {
	ctx = db.WithTableHint(ctx, authorityCutoverTable)
	var (
		status       string
		sourceDigest *string
		count        int
		completedAt  *time.Time
	)
	err := r.db.QueryRow(ctx, `
		SELECT status, source_digest, imported_grant_count, completed_at
		FROM authorization_authority_cutover
		WHERE singleton_id = 1
	`,
	).Scan(&status, &sourceDigest, &count, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthorityCutoverStatus{}, ErrAuthorityCutoverIncomplete
	}
	if err != nil {
		return AuthorityCutoverStatus{}, fmt.Errorf("load authorization authority cutover: %w", err)
	}
	result := AuthorityCutoverStatus{
		Completed:          status == "completed",
		ImportedGrantCount: count,
		CompletedAt:        completedAt,
	}
	if sourceDigest != nil {
		result.SourceDigest = *sourceDigest
	}
	return result, nil
}

func (r *Repository) ApplyAuthorityCutover(
	ctx context.Context,
	input AuthorityCutoverInput,
) (AuthorityCutoverResult, error) {
	result := AuthorityCutoverResult{
		SourceDigest:       input.SourceDigest,
		ImportedGrantCount: len(input.Grants),
	}
	err := r.db.WithTx(db.WithTableHint(ctx, authorityCutoverTable), func(ctx context.Context, tx pgx.Tx) error {
		var (
			status       string
			sourceDigest *string
			imported     int
		)
		if err := tx.QueryRow(ctx, `
			SELECT status, source_digest, imported_grant_count
			FROM authorization_authority_cutover
			WHERE singleton_id = 1
			FOR UPDATE
		`).Scan(&status, &sourceDigest, &imported); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAuthorityCutoverIncomplete
			}
			return fmt.Errorf("lock authorization authority cutover: %w", err)
		}
		if status == "completed" {
			if sourceDigest != nil && *sourceDigest == input.SourceDigest && imported == len(input.Grants) {
				result.Changed = false
				return nil
			}
			return ErrAuthorityCutoverConflict
		}

		var existingGrantCount int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM authorization_grants`).Scan(&existingGrantCount); err != nil {
			return fmt.Errorf("count pre-cutover authorization grants: %w", err)
		}
		if existingGrantCount != 0 {
			return fmt.Errorf("%w: ledger contains %d grants before cutover", ErrAuthorityCutoverConflict, existingGrantCount)
		}

		for _, inputGrant := range input.Grants {
			grant, err := scanGrant(tx.QueryRow(ctx, `
				INSERT INTO authorization_grants (
					subject_user_id, role, source, school_id, section_id,
					desired_state, projection_status, revision, reason,
					created_by_user_id, updated_by_user_id,
					activated_at, projected_at, created_at, updated_at
				) VALUES (
					$1, $2, $3, $4, $5,
					'granted', 'applied', 1, $6,
					NULL, NULL,
					NOW(), NOW(), NOW(), NOW()
				)
				RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
			`,
				inputGrant.SubjectUserID,
				inputGrant.Role,
				inputGrant.Source,
				inputGrant.SchoolID,
				inputGrant.SectionID,
				authorityCutoverReason,
			))
			if err != nil {
				return fmt.Errorf("insert cutover authorization grant: %w", err)
			}
			if err := enqueueProjectionTx(ctx, tx, grant); err != nil {
				return err
			}
			if err := insertAuthorityCutoverGrantAuditTx(ctx, tx, grant); err != nil {
				return err
			}
		}

		commandTag, err := tx.Exec(ctx, `
			UPDATE authorization_authority_cutover
			SET status = 'completed',
			    source_digest = $1,
			    imported_grant_count = $2,
			    completed_at = NOW(),
			    updated_at = NOW()
			WHERE singleton_id = 1
			  AND status = 'pending'
		`, input.SourceDigest, len(input.Grants))
		if err != nil {
			return fmt.Errorf("complete authorization authority cutover: %w", err)
		}
		if commandTag.RowsAffected() != 1 {
			return ErrAuthorityCutoverConflict
		}
		result.Changed = true
		return nil
	})
	if err != nil {
		return AuthorityCutoverResult{}, err
	}
	return result, nil
}

func insertAuthorityCutoverGrantAuditTx(ctx context.Context, tx pgx.Tx, grant Grant) error {
	eventID, err := id.New()
	if err != nil {
		return err
	}
	event := audit.EventFromContext(ctx, audit.Event{
		Type:          audit.EventType("iam.authorization_grant.cutover_imported"),
		Category:      "admin_operation",
		ActorType:     authorityCutoverAuditActor.actorType,
		UserID:        authorityCutoverAuditActor.userID,
		ResourceType:  "authorization_grant",
		ResourceID:    strconv.FormatInt(grant.ID, 10),
		ScopeSchoolID: nullableSchoolID(grant.SchoolID),
		Action:        "cutover_import",
		Result:        "success",
		Reason:        authorityCutoverReason,
		Before:        nil,
		After:         grantSnapshot(grant),
		Details: map[string]any{
			"target_user_id": grant.SubjectUserID,
			"role":           grant.Role,
			"source":         grant.Source,
			"revision":       grant.Revision,
			"section_id":     grant.SectionID,
		},
	})
	afterJSON, err := json.Marshal(event.After)
	if err != nil {
		return fmt.Errorf("marshal cutover authorization audit after: %w", err)
	}
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("marshal cutover authorization audit details: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (
			id, category, event_type, actor_type, actor_user_id, actor_username,
			action, resource_type, resource_id, scope_school_id,
			before_data, after_data, result, reason, trace_id, request_id,
			ip_address, user_agent, details, created_at
		) VALUES (
			$1, $2, $3, $4, $5, '',
			$6, $7, $8, NULLIF($9, ''),
			NULL, $10::jsonb, $11, $12, NULLIF($13, ''), NULLIF($14, ''),
			NULL, NULL, $15::jsonb, $16
		)
	`,
		eventID, event.Category, event.Type, event.ActorType, event.UserID,
		event.Action, event.ResourceType, event.ResourceID, event.ScopeSchoolID,
		afterJSON, event.Result, event.Reason, event.TraceID, event.RequestID,
		detailsJSON, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert cutover authorization audit: %w", err)
	}
	return nil
}
