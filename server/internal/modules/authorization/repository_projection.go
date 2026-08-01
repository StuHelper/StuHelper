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
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

func (r *Repository) ClaimProjectionJobs(
	ctx context.Context,
	limit int,
	staleAfter time.Duration,
) ([]outbox.Job, error) {
	return outbox.ClaimJobs(
		ctx,
		r.db,
		outbox.StreamIAMAuthorizationGrantProjection,
		limit,
		staleAfter,
	)
}

func (r *Repository) MarkProjectionJobDone(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
) error {
	return outbox.MarkJobDone(ctx, r.db, jobID, lockedAt)
}

func (r *Repository) MarkProjectionJobFailure(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
	terminal bool,
) error {
	var payload ProjectionPayload
	if terminal {
		var raw json.RawMessage
		err := r.db.QueryRow(db.WithTableHint(ctx, outbox.DomainEventOutboxTable), `
			SELECT payload
			FROM domain_event_outbox
			WHERE id = $1
			  AND stream = $2
		`, jobID, outbox.StreamIAMAuthorizationGrantProjection).Scan(&raw)
		if err != nil {
			return fmt.Errorf("load terminal authorization projection payload: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("decode terminal authorization projection payload: %w", err)
		}
	}

	if err := outbox.MarkJobFailure(
		ctx,
		r.db,
		jobID,
		lockedAt,
		nextAttemptAt,
		lastError,
		terminal,
	); err != nil {
		return err
	}
	if terminal {
		if err := r.MarkProjectionFailed(ctx, payload, lastError); err != nil {
			return fmt.Errorf("mark authorization projection failed: %w", err)
		}
	}
	return nil
}

func (r *Repository) MarkProjectionApplied(ctx context.Context, payload ProjectionPayload) (bool, error) {
	applied := false
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		before, err := getGrantForUpdateTx(ctx, tx, payload.GrantID)
		if errors.Is(err, ErrGrantNotFound) {
			return ErrProjectionStale
		}
		if err != nil {
			return err
		}
		if before.Revision != payload.Revision || before.DesiredState != payload.DesiredState {
			return nil
		}
		if before.ProjectionStatus == ProjectionApplied {
			applied = true
			return nil
		}

		after, err := scanGrant(tx.QueryRow(ctx, `
			UPDATE authorization_grants
			SET projection_status = 'applied',
			    activated_at = CASE
			        WHEN desired_state = 'granted' THEN COALESCE(activated_at, NOW())
			        ELSE activated_at
			    END,
			    revoked_at = CASE
			        WHEN desired_state = 'revoked' THEN COALESCE(revoked_at, NOW())
			        ELSE NULL
			    END,
			    projected_at = NOW(),
			    last_error = NULL,
			    updated_at = NOW()
			WHERE id = $1
			  AND revision = $2
			  AND desired_state = $3
			RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
		`, payload.GrantID, payload.Revision, payload.DesiredState))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("apply authorization projection: %w", err)
		}
		if err := insertProjectionAuditTx(ctx, tx, "applied", before, after, ""); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if errors.Is(err, ErrProjectionStale) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return applied, nil
}

func (r *Repository) MarkProjectionFailed(
	ctx context.Context,
	payload ProjectionPayload,
	lastError string,
) error {
	return r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		before, err := getGrantForUpdateTx(ctx, tx, payload.GrantID)
		if errors.Is(err, ErrGrantNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if before.Revision != payload.Revision ||
			before.DesiredState != payload.DesiredState ||
			before.ProjectionStatus == ProjectionApplied {
			return nil
		}
		after, err := scanGrant(tx.QueryRow(ctx, `
			UPDATE authorization_grants
			SET projection_status = 'failed',
			    last_error = LEFT($4, 1000),
			    updated_at = NOW()
			WHERE id = $1
			  AND revision = $2
			  AND desired_state = $3
			RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
		`, payload.GrantID, payload.Revision, payload.DesiredState, lastError))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("fail authorization projection: %w", err)
		}
		return insertProjectionAuditTx(ctx, tx, "failed", before, after, lastError)
	})
}

func insertProjectionAuditTx(
	ctx context.Context,
	tx pgx.Tx,
	outcome string,
	before Grant,
	after Grant,
	reason string,
) error {
	eventID, err := id.New()
	if err != nil {
		return err
	}
	result := "success"
	if outcome == "failed" {
		result = "failure"
	}
	event := audit.EventFromContext(ctx, audit.Event{
		Type:          audit.EventType("iam.authorization_grant.projection_" + outcome),
		Category:      "admin_operation",
		ActorType:     "system",
		UserID:        "authorization_projection_worker",
		ResourceType:  "authorization_grant",
		ResourceID:    strconv.FormatInt(after.ID, 10),
		ScopeSchoolID: nullableSchoolID(after.SchoolID),
		Action:        "projection_" + outcome,
		Result:        result,
		Reason:        reason,
		Before:        grantSnapshot(before),
		After:         grantSnapshot(after),
		Details: map[string]any{
			"target_user_id": after.SubjectUserID,
			"role":           after.Role,
			"revision":       after.Revision,
			"desired_state":  after.DesiredState,
			"section_id":     after.SectionID,
		},
	})
	beforeJSON, err := json.Marshal(event.Before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(event.After)
	if err != nil {
		return err
	}
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return err
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
			$10::jsonb, $11::jsonb, $12, NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''),
			NULL, NULL, $16::jsonb, $17
		)
	`,
		eventID, event.Category, event.Type, event.ActorType, event.UserID,
		event.Action, event.ResourceType, event.ResourceID, event.ScopeSchoolID,
		beforeJSON, afterJSON, event.Result, event.Reason, event.TraceID, event.RequestID,
		detailsJSON, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert authorization projection audit: %w", err)
	}
	return nil
}
