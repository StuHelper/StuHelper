package campusconnector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository { return &Repository{db: database} }

type NodeIdentity struct {
	ID                       string
	Status                   string
	ProtocolVersion          string
	CertificateFingerprint   string
	SigningKeyID             string
	SigningPublicKey         []byte
	MaxConcurrency           int
	HeartbeatIntervalSeconds int
	CertificateNotAfter      time.Time
	RevokedAt                *time.Time
}

type SchoolOperation struct {
	NodeID                  string
	SchoolID                int64
	SchoolCode              string
	OperationKey            string
	OperationType           string
	AdapterID               string
	AdapterVersion          string
	UpstreamProtocol        string
	TargetHost              string
	TargetPort              int
	TargetTLSServerName     *string
	TimeoutMilliseconds     int
	MaxConcurrency          int
	RateLimitPerMinute      int
	NodeMaxConcurrency      int
	NodeProtocolVersion     string
	NodeCertificateNotAfter time.Time
}

type ManualRosterSyncRequest struct {
	ID                  string
	NodeID              string
	SchoolID            int64
	SchoolCode          string
	OperationKey        string
	AdapterID           string
	AdapterVersion      string
	Status              string
	ResultCode          *string
	ActorUserID         int64
	Reason              string
	DeadlineAt          time.Time
	ClaimedAt           *time.Time
	ClaimAttempts       int
	CompletedAt         *time.Time
	LatencyMilliseconds *int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (r *Repository) GetNodeIdentity(ctx context.Context, nodeID string) (*NodeIdentity, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_nodes")
	var node NodeIdentity
	err := r.db.QueryRow(ctx, `
		SELECT id, status, protocol_version, certificate_fingerprint,
		       signing_key_id, signing_public_key, max_concurrency,
		       heartbeat_interval_seconds, certificate_not_after, revoked_at
		FROM campus_connector_nodes
		WHERE id = $1
	`, nodeID).Scan(
		&node.ID, &node.Status, &node.ProtocolVersion, &node.CertificateFingerprint,
		&node.SigningKeyID, &node.SigningPublicKey, &node.MaxConcurrency,
		&node.HeartbeatIntervalSeconds, &node.CertificateNotAfter, &node.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *Repository) ResolveSchoolAccountOperation(
	ctx context.Context,
	schoolID int64,
	operationKey string,
	adapterID string,
	adapterVersion string,
	now time.Time,
) (*SchoolOperation, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_school_operations")
	var operation SchoolOperation
	err := r.db.QueryRow(ctx, `
		SELECT operation.node_id, operation.school_id, school.code,
		       operation.operation_key, operation.operation_type,
		       operation.adapter_id, operation.adapter_version,
		       operation.upstream_protocol, operation.target_host,
		       operation.target_port, operation.target_tls_server_name,
		       operation.timeout_milliseconds, operation.max_concurrency,
		       operation.rate_limit_per_minute, node.max_concurrency,
		       node.protocol_version, node.certificate_not_after
		FROM campus_connector_school_operations operation
		JOIN campus_connector_nodes node ON node.id = operation.node_id
		JOIN schools school ON school.id = operation.school_id
		WHERE operation.school_id = $1
		  AND operation.operation_key = $2
		  AND operation.operation_type = 'school_account_authenticate'
		  AND operation.adapter_id = $3
		  AND operation.adapter_version = $4
		  AND operation.enabled
		  AND operation.validation_status = 'valid'
		  AND operation.health_status IN ('healthy', 'degraded')
		  AND node.status IN ('active', 'degraded')
		  AND node.revoked_at IS NULL
		  AND node.certificate_not_after > $5
		  AND node.last_heartbeat_at > $5 - make_interval(secs => node.heartbeat_interval_seconds * 3)
		ORDER BY operation.health_status = 'healthy' DESC,
		         node.last_heartbeat_at DESC, operation.node_id
		LIMIT 1
	`, schoolID, operationKey, adapterID, adapterVersion, now).Scan(
		&operation.NodeID, &operation.SchoolID, &operation.SchoolCode,
		&operation.OperationKey, &operation.OperationType, &operation.AdapterID,
		&operation.AdapterVersion, &operation.UpstreamProtocol, &operation.TargetHost,
		&operation.TargetPort, &operation.TargetTLSServerName, &operation.TimeoutMilliseconds,
		&operation.MaxConcurrency, &operation.RateLimitPerMinute,
		&operation.NodeMaxConcurrency, &operation.NodeProtocolVersion,
		&operation.NodeCertificateNotAfter,
	)
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

func (r *Repository) ResolveRosterOperationForSchool(
	ctx context.Context,
	schoolCode string,
	now time.Time,
) (*SchoolOperation, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_school_operations")
	var operation SchoolOperation
	err := r.db.QueryRow(ctx, `
		SELECT operation.node_id, operation.school_id, school.code,
		       operation.operation_key, operation.operation_type,
		       operation.adapter_id, operation.adapter_version,
		       operation.upstream_protocol, operation.target_host,
		       operation.target_port, operation.target_tls_server_name,
		       operation.timeout_milliseconds, operation.max_concurrency,
		       operation.rate_limit_per_minute, node.max_concurrency,
		       node.protocol_version, node.certificate_not_after
		FROM campus_connector_school_operations operation
		JOIN campus_connector_nodes node ON node.id = operation.node_id
		JOIN schools school ON school.id = operation.school_id
		WHERE school.code = $1
		  AND operation.operation_type = 'roster_snapshot_upload'
		  AND operation.enabled
		  AND operation.validation_status = 'valid'
		  AND operation.health_status IN ('healthy', 'degraded')
		  AND node.status IN ('active', 'degraded')
		  AND node.revoked_at IS NULL
		  AND node.certificate_not_after > $2
		  AND node.last_heartbeat_at > $2 - make_interval(secs => node.heartbeat_interval_seconds * 3)
		ORDER BY operation.health_status = 'healthy' DESC,
		         node.last_heartbeat_at DESC, operation.node_id, operation.operation_key
		LIMIT 1
	`, schoolCode, now).Scan(
		&operation.NodeID, &operation.SchoolID, &operation.SchoolCode,
		&operation.OperationKey, &operation.OperationType, &operation.AdapterID,
		&operation.AdapterVersion, &operation.UpstreamProtocol, &operation.TargetHost,
		&operation.TargetPort, &operation.TargetTLSServerName, &operation.TimeoutMilliseconds,
		&operation.MaxConcurrency, &operation.RateLimitPerMinute,
		&operation.NodeMaxConcurrency, &operation.NodeProtocolVersion,
		&operation.NodeCertificateNotAfter,
	)
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

func (r *Repository) ListNodeOperations(ctx context.Context, nodeID string) ([]SchoolOperation, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_school_operations")
	rows, err := r.db.Query(ctx, `
		SELECT operation.node_id, operation.school_id, school.code,
		       operation.operation_key, operation.operation_type,
		       operation.adapter_id, operation.adapter_version,
		       operation.upstream_protocol, operation.target_host,
		       operation.target_port, operation.target_tls_server_name,
		       operation.timeout_milliseconds, operation.max_concurrency,
		       operation.rate_limit_per_minute, node.max_concurrency,
		       node.protocol_version, node.certificate_not_after
		FROM campus_connector_school_operations operation
		JOIN campus_connector_nodes node ON node.id = operation.node_id
		JOIN schools school ON school.id = operation.school_id
		WHERE operation.node_id = $1
		ORDER BY operation.school_id, operation.operation_key
	`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	operations := make([]SchoolOperation, 0)
	for rows.Next() {
		var operation SchoolOperation
		if err := rows.Scan(
			&operation.NodeID, &operation.SchoolID, &operation.SchoolCode,
			&operation.OperationKey, &operation.OperationType, &operation.AdapterID,
			&operation.AdapterVersion, &operation.UpstreamProtocol, &operation.TargetHost,
			&operation.TargetPort, &operation.TargetTLSServerName, &operation.TimeoutMilliseconds,
			&operation.MaxConcurrency, &operation.RateLimitPerMinute,
			&operation.NodeMaxConcurrency, &operation.NodeProtocolVersion,
			&operation.NodeCertificateNotAfter,
		); err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, rows.Err()
}

func (r *Repository) RecordHeartbeat(
	ctx context.Context,
	node NodeIdentity,
	softwareVersion string,
	protocolVersion string,
	reports []campusconnectorprotocol.OperationHealth,
	now time.Time,
) error {
	operations, err := r.ListNodeOperations(ctx, node.ID)
	if err != nil {
		return err
	}
	reported := make(map[string]campusconnectorprotocol.OperationHealth, len(reports))
	for _, report := range reports {
		reported[report.SchoolCode+"\x00"+report.OperationKey] = report
	}
	overallStatus := "active"
	overallCode := "ok"
	if protocolVersion != node.ProtocolVersion || protocolVersion != campusconnectorprotocol.ProtocolVersion {
		overallStatus = "degraded"
		overallCode = "protocol_version_mismatch"
	}
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, operation := range operations {
			report, ok := reported[operation.SchoolCode+"\x00"+operation.OperationKey]
			tlsName := ""
			if operation.TargetTLSServerName != nil {
				tlsName = *operation.TargetTLSServerName
			}
			expectedFingerprint := campusconnectorprotocol.OperationTargetFingerprint(
				operation.UpstreamProtocol, operation.TargetHost, operation.TargetPort, tlsName,
			)
			matches := ok && report.SchoolCode == operation.SchoolCode && report.OperationType == operation.OperationType &&
				report.AdapterID == operation.AdapterID && report.AdapterVersion == operation.AdapterVersion &&
				report.UpstreamProtocol == operation.UpstreamProtocol &&
				strings.EqualFold(report.TargetFingerprint, expectedFingerprint)
			health := "unavailable"
			code := "operation_not_reported"
			if ok && !matches {
				code = "operation_configuration_mismatch"
			}
			if matches {
				health, code = normalizeReportedHealth(report.HealthCode)
			}
			if health != "healthy" && overallStatus == "active" {
				overallStatus = "degraded"
				overallCode = code
			}
			if _, err := tx.Exec(ctx, `
				UPDATE campus_connector_school_operations
				SET health_status = $4, health_code = $5, health_checked_at = $6,
				    updated_at = $6
				WHERE node_id = $1 AND school_id = $2 AND operation_key = $3
			`, node.ID, operation.SchoolID, operation.OperationKey, health, code, now); err != nil {
				return err
			}
		}
		if node.Status == "registered" && overallStatus == "degraded" {
			overallStatus = "registered"
		}
		command, err := tx.Exec(ctx, `
			UPDATE campus_connector_nodes
			SET status = $2, software_version = $3, last_heartbeat_at = $4,
			    last_health_code = $5, updated_at = $4, revision = revision + 1
			WHERE id = $1 AND revoked_at IS NULL AND status <> 'revoked'
		`, node.ID, overallStatus, softwareVersion, now, overallCode)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrAuthentication
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO campus_connector_node_events (
				node_id, event_type, event_code, revision, occurred_at, created_at
			)
			SELECT id, 'health_changed', $2, revision, $3, $3
			FROM campus_connector_nodes WHERE id = $1
		`, node.ID, overallCode, now)
		return err
	})
}

func (r *Repository) CreateInteractiveRequest(ctx context.Context, request InteractiveRequest, now time.Time) error {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	return r.createRequest(ctx, request.ID, request.NodeID, request.SchoolID, request.OperationKey,
		"interactive_school_account", request.ApplicationID, nil, request.DeadlineAt, now)
}

func (r *Repository) CreateSnapshotRequest(
	ctx context.Context,
	requestID, nodeID string,
	schoolID int64,
	operationKey string,
	deadlineAt, now time.Time,
) error {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	return r.createRequest(ctx, requestID, nodeID, schoolID, operationKey,
		"roster_snapshot_push", nil, nil, deadlineAt, now)
}

func (r *Repository) CreateManualRosterSyncRequest(
	ctx context.Context,
	request ManualRosterSyncRequest,
) error {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	referenceHash := sha256.Sum256([]byte("campus-connector-request:v1\x00" + request.ID))
	_, err := r.db.Exec(ctx, `
		INSERT INTO campus_connector_requests (
			id, request_reference_hash, node_id, school_id, operation_key,
			request_kind, status, actor_user_id, request_reason,
			deadline_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			'roster_snapshot_manual', 'pending', $6, $7,
			$8, $9, $9
		)
	`, request.ID, hex.EncodeToString(referenceHash[:]), request.NodeID,
		request.SchoolID, request.OperationKey, request.ActorUserID,
		request.Reason, request.DeadlineAt, request.CreatedAt)
	if isConstraintViolation(err, "campus_connector_requests_manual_inflight_uidx") {
		return ErrRequestInFlight
	}
	return err
}

func (r *Repository) ListManualRosterSyncRequests(
	ctx context.Context,
	schoolCode string,
	limit int,
) ([]ManualRosterSyncRequest, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	rows, err := r.db.Query(ctx, `
		SELECT request.id, request.node_id, request.school_id, school.code,
		       request.operation_key, operation.adapter_id, operation.adapter_version,
		       request.status, request.result_code, request.actor_user_id,
		       request.request_reason, request.deadline_at, request.claimed_at,
		       request.claim_attempts, request.completed_at,
		       request.latency_milliseconds, request.created_at, request.updated_at
		FROM campus_connector_requests request
		JOIN schools school ON school.id = request.school_id
		JOIN campus_connector_school_operations operation
		  ON operation.node_id = request.node_id
		 AND operation.school_id = request.school_id
		 AND operation.operation_key = request.operation_key
		WHERE request.request_kind = 'roster_snapshot_manual'
		  AND school.code = $1
		ORDER BY request.created_at DESC, request.id DESC
		LIMIT $2
	`, schoolCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]ManualRosterSyncRequest, 0)
	for rows.Next() {
		var request ManualRosterSyncRequest
		if err := rows.Scan(
			&request.ID, &request.NodeID, &request.SchoolID, &request.SchoolCode,
			&request.OperationKey, &request.AdapterID, &request.AdapterVersion,
			&request.Status, &request.ResultCode, &request.ActorUserID,
			&request.Reason, &request.DeadlineAt, &request.ClaimedAt,
			&request.ClaimAttempts, &request.CompletedAt,
			&request.LatencyMilliseconds, &request.CreatedAt, &request.UpdatedAt,
		); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (r *Repository) ClaimManualRosterSyncRequest(
	ctx context.Context,
	nodeID string,
	now time.Time,
	lease time.Duration,
	maxAttempts int,
) (*ManualRosterSyncRequest, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	leaseCutoff := now.Add(-lease)
	var claimed ManualRosterSyncRequest
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE campus_connector_requests
			SET status = 'timed_out', result_code = 'delivery_deadline_exceeded',
			    completed_at = $2,
			    latency_milliseconds = GREATEST(0, (EXTRACT(EPOCH FROM ($2 - created_at)) * 1000)::integer),
			    updated_at = $2
			WHERE node_id = $1
			  AND request_kind = 'roster_snapshot_manual'
			  AND status IN ('pending', 'started')
			  AND deadline_at <= $2
		`, nodeID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE campus_connector_requests
			SET status = 'failed', result_code = 'delivery_attempts_exhausted',
			    completed_at = $3,
			    latency_milliseconds = GREATEST(0, (EXTRACT(EPOCH FROM ($3 - created_at)) * 1000)::integer),
			    updated_at = $3
			WHERE node_id = $1
			  AND request_kind = 'roster_snapshot_manual'
			  AND status = 'started'
			  AND updated_at <= $2
			  AND claim_attempts >= $4
		`, nodeID, leaseCutoff, now, maxAttempts); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			SELECT request.id, request.node_id, request.school_id, school.code,
			       request.operation_key, operation.adapter_id, operation.adapter_version,
			       request.status, request.result_code, request.actor_user_id,
			       request.request_reason, request.deadline_at, request.claimed_at,
			       request.claim_attempts, request.completed_at,
			       request.latency_milliseconds, request.created_at, request.updated_at
			FROM campus_connector_requests request
			JOIN schools school ON school.id = request.school_id
			JOIN campus_connector_school_operations operation
			  ON operation.node_id = request.node_id
			 AND operation.school_id = request.school_id
			 AND operation.operation_key = request.operation_key
			WHERE request.node_id = $1
			  AND request.request_kind = 'roster_snapshot_manual'
			  AND request.deadline_at > $2
			  AND request.claim_attempts < $4
			  AND (
			      request.status = 'pending'
			      OR (request.status = 'started' AND request.updated_at <= $3)
			  )
			ORDER BY request.created_at, request.id
			FOR UPDATE OF request SKIP LOCKED
			LIMIT 1
		`, nodeID, now, leaseCutoff, maxAttempts)
		if err := row.Scan(
			&claimed.ID, &claimed.NodeID, &claimed.SchoolID, &claimed.SchoolCode,
			&claimed.OperationKey, &claimed.AdapterID, &claimed.AdapterVersion,
			&claimed.Status, &claimed.ResultCode, &claimed.ActorUserID,
			&claimed.Reason, &claimed.DeadlineAt, &claimed.ClaimedAt,
			&claimed.ClaimAttempts, &claimed.CompletedAt,
			&claimed.LatencyMilliseconds, &claimed.CreatedAt, &claimed.UpdatedAt,
		); err != nil {
			return err
		}
		claimed.Status = "started"
		claimed.ClaimAttempts++
		claimed.ClaimedAt = &now
		claimed.UpdatedAt = now
		command, err := tx.Exec(ctx, `
			UPDATE campus_connector_requests
			SET status = 'started', claimed_at = $2,
			    claim_attempts = claim_attempts + 1, updated_at = $2
			WHERE id = $1 AND request_kind = 'roster_snapshot_manual'
		`, claimed.ID, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrRequestNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (r *Repository) CompleteManualRosterSyncRequest(
	ctx context.Context,
	nodeID, requestID, resultCode string,
	now time.Time,
) error {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	command, err := r.db.Exec(ctx, `
		UPDATE campus_connector_requests
		SET status = 'failed', result_code = $3, completed_at = $4,
		    latency_milliseconds = GREATEST(0, (EXTRACT(EPOCH FROM ($4 - created_at)) * 1000)::integer),
		    updated_at = $4
		WHERE id = $2 AND node_id = $1
		  AND request_kind = 'roster_snapshot_manual'
		  AND status = 'started'
	`, nodeID, requestID, resultCode, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrRequestNotFound
	}
	return nil
}

func (r *Repository) EnsureSnapshotRequest(
	ctx context.Context,
	requestID, nodeID string,
	schoolID int64,
	operationKey string,
	deadlineAt, now time.Time,
) error {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	var existingNodeID, kind, status, existingOperationKey string
	var existingSchoolID int64
	var existingDeadline time.Time
	err := r.db.QueryRow(ctx, `
		SELECT node_id, school_id, operation_key, request_kind, status, deadline_at
		FROM campus_connector_requests
		WHERE id = $1
	`, requestID).Scan(
		&existingNodeID, &existingSchoolID, &existingOperationKey,
		&kind, &status, &existingDeadline,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.CreateSnapshotRequest(
			ctx, requestID, nodeID, schoolID, operationKey, deadlineAt, now,
		)
	}
	if err != nil {
		return err
	}
	if kind != "roster_snapshot_manual" || status != "started" ||
		existingNodeID != nodeID || existingSchoolID != schoolID ||
		existingOperationKey != operationKey || !existingDeadline.After(now) {
		return ErrRequestNotFound
	}
	return nil
}

func (r *Repository) createRequest(
	ctx context.Context,
	requestID, nodeID string,
	schoolID int64,
	operationKey, kind string,
	applicationID, rosterSnapshotID *string,
	deadlineAt, now time.Time,
) error {
	referenceHash := sha256.Sum256([]byte("campus-connector-request:v1\x00" + requestID))
	_, err := r.db.Exec(ctx, `
		INSERT INTO campus_connector_requests (
			id, request_reference_hash, node_id, school_id, operation_key,
			request_kind, status, application_id, roster_snapshot_id,
			deadline_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'started', $7, $8, $9, $10, $10)
	`, requestID, hex.EncodeToString(referenceHash[:]), nodeID, schoolID, operationKey,
		kind, applicationID, rosterSnapshotID, deadlineAt, now)
	return err
}

func (r *Repository) CompleteRequest(ctx context.Context, requestID, resultCode string, succeeded bool, now time.Time) error {
	ctx = db.WithTableHint(ctx, "campus_connector_requests")
	status := "failed"
	if succeeded {
		status = "succeeded"
	}
	command, err := r.db.Exec(ctx, `
		UPDATE campus_connector_requests
		SET status = $2, result_code = $3, completed_at = $4,
		    latency_milliseconds = GREATEST(0, (EXTRACT(EPOCH FROM ($4 - created_at)) * 1000)::integer),
		    updated_at = $4
		WHERE id = $1 AND status = 'started'
	`, requestID, status, resultCode, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrRequestNotFound
	}
	return nil
}

func (r *Repository) AttachSnapshotResult(
	ctx context.Context,
	requestID, uploadID, snapshotID string,
	manifestSchemaVersion int,
	manifestChecksum, signingKeyID string,
	now time.Time,
) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `
			UPDATE campus_connector_requests
			SET roster_snapshot_id = $2, status = 'succeeded', result_code = 'snapshot_imported',
			    completed_at = $3,
			    latency_milliseconds = GREATEST(0, (EXTRACT(EPOCH FROM ($3 - created_at)) * 1000)::integer),
			    updated_at = $3
			WHERE id = $1 AND status = 'started'
		`, requestID, snapshotID, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrRequestNotFound
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO campus_connector_snapshot_uploads (
				id, request_id, snapshot_id, manifest_schema_version,
				manifest_checksum, signature_key_id, signature_verified,
				status, received_at, verified_at, imported_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, true, 'imported', $7, $7, $7, $7, $7)
		`, uploadID, requestID, snapshotID, manifestSchemaVersion,
			manifestChecksum, signingKeyID, now)
		return err
	})
}

func (r *Repository) GetRosterOperation(
	ctx context.Context,
	nodeID, schoolCode, operationKey, adapterID, adapterVersion string,
) (*SchoolOperation, error) {
	ctx = db.WithTableHint(ctx, "campus_connector_school_operations")
	var operation SchoolOperation
	err := r.db.QueryRow(ctx, `
		SELECT operation.node_id, operation.school_id, school.code,
		       operation.operation_key, operation.operation_type,
		       operation.adapter_id, operation.adapter_version,
		       operation.upstream_protocol, operation.target_host,
		       operation.target_port, operation.target_tls_server_name,
		       operation.timeout_milliseconds, operation.max_concurrency,
		       operation.rate_limit_per_minute, node.max_concurrency,
		       node.protocol_version, node.certificate_not_after
		FROM campus_connector_school_operations operation
		JOIN campus_connector_nodes node ON node.id = operation.node_id
		JOIN schools school ON school.id = operation.school_id
		WHERE operation.node_id = $1 AND school.code = $2 AND operation.operation_key = $3
		  AND operation.operation_type = 'roster_snapshot_upload'
		  AND operation.adapter_id = $4 AND operation.adapter_version = $5
		  AND operation.enabled AND operation.validation_status = 'valid'
		  AND node.status IN ('active', 'degraded') AND node.revoked_at IS NULL
	`, nodeID, schoolCode, operationKey, adapterID, adapterVersion).Scan(
		&operation.NodeID, &operation.SchoolID, &operation.SchoolCode,
		&operation.OperationKey, &operation.OperationType, &operation.AdapterID,
		&operation.AdapterVersion, &operation.UpstreamProtocol, &operation.TargetHost,
		&operation.TargetPort, &operation.TargetTLSServerName, &operation.TimeoutMilliseconds,
		&operation.MaxConcurrency, &operation.RateLimitPerMinute,
		&operation.NodeMaxConcurrency, &operation.NodeProtocolVersion,
		&operation.NodeCertificateNotAfter,
	)
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

func normalizeReportedHealth(code string) (status, normalizedCode string) {
	switch strings.TrimSpace(code) {
	case "ok":
		return "healthy", "ok"
	case "upstream_slow", "rate_limited":
		return "degraded", strings.TrimSpace(code)
	case "tls_failure", "secret_unavailable", "schema_unknown", "upstream_unavailable", "circuit_open", "snapshot_encryption_failed":
		return "unavailable", strings.TrimSpace(code)
	default:
		return "unavailable", "unknown_health_code"
	}
}

func isConstraintViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func (o SchoolOperation) TargetFingerprint() string {
	tlsName := ""
	if o.TargetTLSServerName != nil {
		tlsName = *o.TargetTLSServerName
	}
	return campusconnectorprotocol.OperationTargetFingerprint(
		o.UpstreamProtocol, o.TargetHost, o.TargetPort, tlsName,
	)
}

func (o SchoolOperation) ValidateExact(report campusconnectorprotocol.OperationHealth) error {
	if report.SchoolCode != o.SchoolCode || report.OperationKey != o.OperationKey || report.OperationType != o.OperationType ||
		report.AdapterID != o.AdapterID || report.AdapterVersion != o.AdapterVersion ||
		report.UpstreamProtocol != o.UpstreamProtocol ||
		!strings.EqualFold(report.TargetFingerprint, o.TargetFingerprint()) {
		return fmt.Errorf("operation configuration does not match approved registry")
	}
	return nil
}
