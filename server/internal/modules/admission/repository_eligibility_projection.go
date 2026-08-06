package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ApplyStudentEligibilityDecisionTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	schoolID int64,
	eligible bool,
	credentialClass string,
	revision int64,
	now time.Time,
) ([]AdmissionSession, error) {
	if err := r.supersedeReleaseActionsForEligibilityTx(
		ctx,
		tx,
		userID,
		schoolID,
		revision,
		eligible,
		credentialClass,
		now,
	); err != nil {
		return nil, err
	}
	_, err := tx.Exec(ctx, `
			UPDATE group_admission_sessions AS session
			SET status = CASE
			        WHEN session.requirements_status = $5 THEN $5
			        ELSE $4
			    END,
			    verified_at = NULL,
			    eligibility_revision = $3,
			    eligibility_evaluated_at = $6,
			    next_reminder_at = $6,
			    last_bot_error = NULL,
			    updated_at = $6
			FROM group_admission_policies AS policy
			WHERE session.platform = policy.platform
			  AND session.guild_id = policy.guild_id
			  AND session.user_id = $1
			  AND policy.school_id = $2
			  AND session.status IN ($4, $5, $7, $8)
			  AND session.cancelled_at IS NULL
			  AND (session.eligibility_revision IS NULL OR session.eligibility_revision <= $3)
			  AND NOT (
			      $9
			      AND (
			          $10 = 'formal_student'
			          OR ($10 = 'temporary_freshman' AND policy.allow_temporary_freshman)
			      )
			  )
		`, userID, schoolID, revision, StatusLinked, StatusMaterialSubmitted, now, StatusEligible, StatusVerified,
		eligible, credentialClass)
	if err != nil {
		return nil, fmt.Errorf("apply ineligible admission decision: %w", err)
	}

	rows, err := tx.Query(ctx, `
		UPDATE group_admission_sessions AS session
		SET requirements_status = CASE
		        WHEN session.status IN ($4, $5) THEN session.status
		        ELSE COALESCE(session.requirements_status, $4)
		    END,
		    status = $6,
		    verified_at = COALESCE(session.verified_at, $7),
		    eligibility_revision = $3,
		    eligibility_evaluated_at = $7,
		    next_reminder_at = NULL,
		    last_bot_error = NULL,
		    updated_at = $7
		FROM group_admission_policies AS policy
		WHERE session.platform = policy.platform
		  AND session.guild_id = policy.guild_id
		  AND session.user_id = $1
		  AND policy.school_id = $2
		  AND $9
		  AND (
		      $10 = 'formal_student'
		      OR ($10 = 'temporary_freshman' AND policy.allow_temporary_freshman)
		  )
		  AND session.status IN ($4, $5, $8, $6)
		  AND session.cancelled_at IS NULL
		  AND (session.eligibility_revision IS NULL OR session.eligibility_revision <= $3)
		RETURNING `+admissionSessionScanColumnsWithAlias("session")+`
	`, userID, schoolID, revision, StatusLinked, StatusMaterialSubmitted, StatusVerified, now, StatusEligible,
		eligible, credentialClass)
	if err != nil {
		return nil, fmt.Errorf("apply eligible admission decision: %w", err)
	}
	defer rows.Close()
	sessions, err := scanAdmissionSessions(rows)
	if err != nil {
		return nil, fmt.Errorf("scan eligible admission sessions: %w", err)
	}
	return sessions, nil
}

func (r *Repository) supersedeReleaseActionsForEligibilityTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	schoolID int64,
	revision int64,
	eligible bool,
	credentialClass string,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE admission_bot_action_outbox AS action
		SET status = 'stale',
		    last_error = NULL,
		    updated_at = $6
		FROM group_admission_sessions AS session
		JOIN group_admission_policies AS policy
		  ON policy.platform = session.platform
		 AND policy.guild_id = session.guild_id
		WHERE action.session_id = session.id
		  AND action.action = 'release'
		  AND action.status IN ('pending', 'failed', 'dispatched', 'dead_letter')
		  AND session.user_id = $1
		  AND policy.school_id = $2
		  AND session.cancelled_at IS NULL
		  AND (
		      NOT (
		          $4
		          AND (
		              $5 = 'formal_student'
		              OR ($5 = 'temporary_freshman' AND policy.allow_temporary_freshman)
		          )
		      )
		      OR action.eligibility_revision IS DISTINCT FROM $3
		  )
	`, userID, schoolID, revision, eligible, credentialClass, now)
	if err != nil {
		return fmt.Errorf("supersede admission release actions: %w", err)
	}
	return nil
}
