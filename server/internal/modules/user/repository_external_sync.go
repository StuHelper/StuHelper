package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

var externalSyncOutboxStreams = []string{
	outbox.StreamIAMCasdoorRoleSync,
	outbox.StreamIAMOpenFGATupleSync,
	externalSyncStreamAdmissionVerificationProjection,
}

const externalSyncStreamAdmissionVerificationProjection = "admission_verification_projection"

var externalSyncJobTypesByStream = map[string][]string{
	outbox.StreamIAMCasdoorRoleSync: {
		externalSyncJobTypeVerifiedStudentRole,
		externalSyncJobTypeFreshmanProvisionalRole,
	},
	outbox.StreamIAMOpenFGATupleSync: {
		externalSyncJobTypeUserProfileProjection,
	},
	externalSyncStreamAdmissionVerificationProjection: {
		externalSyncJobTypeAdmissionVerification,
	},
}

func (r *Repository) UpsertExternalSyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	stream, err := externalSyncStreamForJobType(jobType)
	if err != nil {
		return err
	}
	if err := outbox.UpsertJobTx(ctx, tx, stream, jobType, dedupeKey, payload); err != nil {
		return fmt.Errorf("UpsertExternalSyncJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimExternalSyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error) {
	if limit <= 0 {
		return nil, nil
	}
	state := externalSyncClaimState{
		jobs:       make([]outbox.Job, 0, limit),
		limit:      limit,
		staleAfter: staleAfter,
	}
	if err := r.claimExternalSyncFairShare(ctx, &state); err != nil {
		return nil, err
	}
	if err := r.claimExternalSyncRemaining(ctx, &state); err != nil {
		return nil, err
	}
	return mapExternalSyncJobs(state.jobs), nil
}

type externalSyncClaimState struct {
	jobs       []outbox.Job
	limit      int
	staleAfter time.Duration
}

type externalSyncStreamClaim struct {
	stream string
	limit  int
}

func (state *externalSyncClaimState) remainingLimit() int {
	return state.limit - len(state.jobs)
}

func (r *Repository) claimExternalSyncFairShare(ctx context.Context, state *externalSyncClaimState) error {
	for index, stream := range externalSyncOutboxStreams {
		streamLimit := fairExternalSyncStreamLimit(state.remainingLimit(), len(externalSyncOutboxStreams)-index)
		claim := externalSyncStreamClaim{stream: stream, limit: streamLimit}
		if err := r.claimExternalSyncStream(ctx, state, claim); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) claimExternalSyncRemaining(ctx context.Context, state *externalSyncClaimState) error {
	for _, stream := range externalSyncOutboxStreams {
		remainingLimit := state.remainingLimit()
		if remainingLimit == 0 {
			return nil
		}
		claim := externalSyncStreamClaim{stream: stream, limit: remainingLimit}
		if err := r.claimExternalSyncStream(ctx, state, claim); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) claimExternalSyncStream(ctx context.Context, state *externalSyncClaimState, claim externalSyncStreamClaim) error {
	jobTypes := externalSyncJobTypesByStream[claim.stream]
	claimed, err := outbox.ClaimJobsByTypes(ctx, r.db, claim.stream, jobTypes, claim.limit, state.staleAfter)
	if err != nil {
		return fmt.Errorf("ClaimExternalSyncJobs: %w", err)
	}
	state.jobs = append(state.jobs, claimed...)
	return nil
}

func fairExternalSyncStreamLimit(remainingLimit, remainingStreams int) int {
	if remainingLimit <= 0 || remainingStreams <= 0 {
		return 0
	}
	return (remainingLimit + remainingStreams - 1) / remainingStreams
}

func (r *Repository) MarkExternalSyncJobDone(ctx context.Context, jobID int64) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID); err != nil {
		return fmt.Errorf("MarkExternalSyncJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkExternalSyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	if err := outbox.MarkJobRetry(ctx, r.db, jobID, nextAttemptAt, lastError); err != nil {
		return fmt.Errorf("MarkExternalSyncJobRetry: %w", err)
	}
	return nil
}

func (r *Repository) MarkExternalSyncJobFailure(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string, terminal bool) error {
	if err := outbox.MarkJobFailure(ctx, r.db, jobID, nextAttemptAt, lastError, terminal); err != nil {
		return fmt.Errorf("MarkExternalSyncJobFailure: %w", err)
	}
	return nil
}

func (r *Repository) ListStudentRoleProjectionStates(ctx context.Context, limit int) ([]StudentRoleProjectionState, error) {
	if limit <= 0 {
		return nil, nil
	}
	ctx = withDBTable(ctx, "user_profiles")
	rows, err := r.db.Query(ctx, `
		SELECT user_id,
		       verification_status = 'verified' AS approved
		FROM user_profiles
		ORDER BY user_id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("ListStudentRoleProjectionStates: %w", err)
	}
	defer rows.Close()

	states := make([]StudentRoleProjectionState, 0, limit)
	for rows.Next() {
		var state StudentRoleProjectionState
		if err := rows.Scan(&state.UserID, &state.Approved); err != nil {
			return nil, fmt.Errorf("ListStudentRoleProjectionStates scan: %w", err)
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListStudentRoleProjectionStates rows: %w", err)
	}
	return states, nil
}

func mapExternalSyncJobs(jobs []outbox.Job) []ExternalSyncJob {
	items := make([]ExternalSyncJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, ExternalSyncJob{
			ID:           job.ID,
			JobType:      job.JobType,
			Payload:      append([]byte(nil), job.Payload...),
			AttemptCount: job.AttemptCount,
		})
	}
	return items
}

func externalSyncStreamForJobType(jobType string) (string, error) {
	switch jobType {
	case externalSyncJobTypeVerifiedStudentRole, externalSyncJobTypeFreshmanProvisionalRole:
		return outbox.StreamIAMCasdoorRoleSync, nil
	case externalSyncJobTypeUserProfileProjection:
		return outbox.StreamIAMOpenFGATupleSync, nil
	case externalSyncJobTypeAdmissionVerification:
		return externalSyncStreamAdmissionVerificationProjection, nil
	default:
		return "", fmt.Errorf("unsupported external sync job type: %s", jobType)
	}
}
