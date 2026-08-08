package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/StuHelper/StuHelper/server/internal/modules/campusconnector"
	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

type Runner struct {
	cfg    Config
	client *Client
	logger *slog.Logger
	ldap   map[string]*LDAPAdapter
	states map[string]*operationState
}

type operationState struct {
	mu               sync.Mutex
	consecutiveFails int
	openUntil        time.Time
	healthCode       string
	running          bool
}

func NewRunner(cfg Config, client *Client, logger *slog.Logger) (*Runner, error) {
	if client == nil {
		return nil, errors.New("campus connector client is required")
	}
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	runner := &Runner{
		cfg: cfg, client: client, logger: logger,
		ldap:   make(map[string]*LDAPAdapter),
		states: make(map[string]*operationState),
	}
	for _, operation := range cfg.Operations {
		runner.states[operation.Key] = &operationState{healthCode: "upstream_unavailable"}
		if operation.Type == "school_account_authenticate" {
			adapter, err := NewLDAPAdapter(operation)
			if err != nil {
				return nil, fmt.Errorf("initialize LDAP operation %q: %w", operation.Key, err)
			}
			runner.ldap[operation.Key] = adapter
		}
	}
	return runner, nil
}

func (r *Runner) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return r.runHeartbeat(groupCtx) })
	for worker := 0; worker < r.cfg.PollWorkers; worker++ {
		group.Go(func() error { return r.runPollWorker(groupCtx) })
	}
	for _, operation := range r.cfg.Operations {
		if operation.Type != "roster_snapshot_upload" {
			continue
		}
		operation := operation
		group.Go(func() error { return r.runRosterSchedule(groupCtx, operation) })
	}
	return group.Wait()
}

func (r *Runner) runHeartbeat(ctx context.Context) error {
	ticker := time.NewTicker(r.cfg.HeartbeatInterval())
	defer ticker.Stop()
	for {
		r.sendHeartbeat(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runner) sendHeartbeat(ctx context.Context) {
	reports := make([]connectorprotocol.OperationHealth, 0, len(r.cfg.Operations))
	for _, operation := range r.cfg.Operations {
		health := r.operationHealth(ctx, operation)
		reports = append(reports, connectorprotocol.OperationHealth{
			OperationKey: operation.Key, SchoolCode: operation.SchoolCode, OperationType: operation.Type,
			AdapterID: operation.AdapterID, AdapterVersion: operation.AdapterVersion,
			UpstreamProtocol: operation.UpstreamProtocol,
			TargetFingerprint: connectorprotocol.OperationTargetFingerprint(
				operation.UpstreamProtocol, operation.TargetHost,
				operation.TargetPort, operation.TLSServerName,
			),
			HealthCode: health,
		})
	}
	heartbeatCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := r.client.PostHeartbeat(heartbeatCtx, connectorprotocol.Heartbeat{
		SoftwareVersion: r.cfg.SoftwareVersion,
		ProtocolVersion: r.cfg.ProtocolVersion,
		Operations:      reports,
	}); err != nil && ctx.Err() == nil {
		r.logger.Warn("campus connector heartbeat failed", "error_class", "central_unavailable")
	}
}

func (r *Runner) operationHealth(ctx context.Context, operation OperationConfig) string {
	state := r.states[operation.Key]
	if state.isOpen(time.Now()) {
		return "circuit_open"
	}
	if operation.Type == "school_account_authenticate" {
		healthCtx, cancel := context.WithTimeout(ctx, time.Duration(operation.TimeoutMilliseconds)*time.Millisecond)
		code := r.ldap[operation.Key].Health(healthCtx)
		cancel()
		if code == "ok" {
			state.success()
		} else {
			state.failure(code)
		}
		return code
	}
	if operation.OracleRoster == nil || !secretFileAvailable(operation.OracleRoster.UsernameFile) ||
		!secretFileAvailable(operation.OracleRoster.PasswordFile) {
		state.failure("secret_unavailable")
		return "secret_unavailable"
	}
	if operation.UpstreamProtocol == "oracle_tls" {
		if _, err := os.Stat(operation.OracleRoster.CAFile); err != nil {
			state.failure("tls_failure")
			return "tls_failure"
		}
	}
	return state.currentHealth("ok")
}

func (r *Runner) runPollWorker(ctx context.Context) error {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return nil
		}
		delivery, err := r.client.Poll(ctx)
		if err != nil {
			if !waitContext(ctx, backoff) {
				return nil
			}
			if backoff < 15*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		if delivery == nil {
			continue
		}
		if delivery.RosterSync != nil {
			r.handleRosterSync(ctx, *delivery.RosterSync)
			continue
		}
		if delivery.Interactive != nil {
			r.handleInteractive(ctx, *delivery.Interactive, delivery.Password)
			continue
		}
		wipe(delivery.Password)
	}
}

func (r *Runner) handleRosterSync(
	ctx context.Context,
	command connectorprotocol.RosterSyncCommand,
) {
	operation, ok := r.cfg.Operation(command.OperationKey)
	resultCode := "schema_unknown"
	if ok && operation.Type == "roster_snapshot_upload" && operation.OracleRoster != nil &&
		operation.SchoolCode == command.SchoolCode &&
		operation.AdapterID == command.AdapterID && operation.AdapterVersion == command.AdapterVersion &&
		command.DeadlineAt.After(time.Now()) {
		requestCtx, cancel := context.WithDeadline(ctx, command.DeadlineAt)
		resultCode = r.syncRoster(requestCtx, operation, command.RequestID)
		cancel()
	} else if command.DeadlineAt.Before(time.Now()) {
		resultCode = "cancelled"
	}
	if resultCode == campusconnector.ResultSuccess || resultCode == "operation_busy" ||
		resultCode == "central_unavailable" {
		return
	}
	resultCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := r.client.PostRosterResult(resultCtx, connectorprotocol.RosterSyncResult{
		RequestID: command.RequestID, ResultCode: resultCode,
	})
	cancel()
	if err != nil && ctx.Err() == nil {
		r.logger.Warn("campus connector roster result delivery failed",
			"request_reference", safeRequestReference(command.RequestID),
			"error_class", "central_unavailable",
		)
	}
}

func (r *Runner) handleInteractive(
	ctx context.Context,
	metadata connectorprotocol.InteractiveMetadata,
	password []byte,
) {
	defer wipe(password)
	result := connectorprotocol.InteractiveResult{
		RequestID:  metadata.RequestID,
		ResultCode: campusconnector.ResultSchemaUnknown,
	}
	operation, ok := r.cfg.Operation(metadata.OperationKey)
	adapter := r.ldap[metadata.OperationKey]
	if ok && adapter != nil && operation.Type == "school_account_authenticate" &&
		operation.SchoolCode == metadata.SchoolCode &&
		operation.AdapterID == metadata.AdapterID && operation.AdapterVersion == metadata.AdapterVersion &&
		metadata.DeadlineAt.After(time.Now()) && !r.states[operation.Key].isOpen(time.Now()) {
		requestCtx, cancel := context.WithDeadline(ctx, metadata.DeadlineAt)
		result = adapter.Authenticate(requestCtx, metadata, password)
		cancel()
		if connectorDependencyFailure(result.ResultCode) {
			r.states[operation.Key].failure(result.ResultCode)
		} else if result.ResultCode == campusconnector.ResultSuccess {
			r.states[operation.Key].success()
		}
	} else if ok && r.states[operation.Key].isOpen(time.Now()) {
		result.ResultCode = campusconnector.ResultUnavailable
	}
	resultCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := r.client.PostResult(resultCtx, result)
	cancel()
	result.AccountSubject = ""
	result.StudentID = ""
	if err != nil && ctx.Err() == nil {
		r.logger.Warn("campus connector result delivery failed",
			"request_reference", safeRequestReference(metadata.RequestID),
			"error_class", "central_unavailable",
		)
	}
}

func (r *Runner) runRosterSchedule(ctx context.Context, operation OperationConfig) error {
	interval := time.Duration(operation.OracleRoster.FullSnapshotIntervalMinutes) * time.Minute
	maximumRetryDelay := min(interval, 6*time.Hour)
	retryDelay := min(15*time.Minute, maximumRetryDelay)
	initial := time.NewTimer(2 * time.Second)
	defer initial.Stop()
	select {
	case <-ctx.Done():
		return nil
	case <-initial.C:
	}
	for {
		resultCode := r.syncRoster(ctx, operation, "")
		delay := interval
		if resultCode != campusconnector.ResultSuccess {
			delay = retryDelay
			retryDelay = min(retryDelay*2, maximumRetryDelay)
		} else {
			retryDelay = 15 * time.Minute
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (r *Runner) syncRoster(ctx context.Context, operation OperationConfig, requestID string) string {
	state := r.states[operation.Key]
	if !state.begin(time.Now()) {
		return "operation_busy"
	}
	defer state.end()
	cfg := operation.OracleRoster
	snapshotConfig, err := BuildOracleRosterSnapshotConfig(operation)
	if err != nil {
		state.failure("secret_unavailable")
		return "secret_unavailable"
	}
	timeout := snapshotConfig.QueryTimeout
	syncCtx, cancel := context.WithTimeout(ctx, timeout)
	snapshot, err := externaldata.ReadOracleFullRosterSnapshot(syncCtx, snapshotConfig)
	cancel()
	if err != nil {
		resultCode := classifyRosterFailure(err)
		state.failure(resultCode)
		r.logger.Warn("campus connector roster snapshot failed",
			"operation", operation.Key, "error_class", resultCode)
		return resultCode
	}
	if !snapshot.Quality.LeastPrivilegeVerified {
		r.logger.Warn("campus connector Oracle account exceeds preferred least-privilege grants",
			"operation", operation.Key, "risk", "existing_account_privileges")
	}
	defer clearConnectorRecords(snapshot.Records)
	payload, err := json.Marshal(connectorprotocol.RosterSnapshotPayload{Records: snapshot.Records})
	if err != nil {
		state.failure("schema_unknown")
		return "schema_unknown"
	}
	defer wipe(payload)
	digest := sha256.Sum256(payload)
	sourceVersion := "full-" + snapshot.SourceCutoffAt.Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(digest[:8])
	if requestID == "" {
		requestID = uuid.NewString()
	}
	manifest := connectorprotocol.SnapshotManifest{
		SchemaVersion: 1, RequestID: requestID, NodeID: r.cfg.NodeID,
		SchoolCode: operation.SchoolCode, OperationKey: operation.Key,
		AdapterID: operation.AdapterID, AdapterVersion: operation.AdapterVersion,
		MappingVersion: cfg.MappingVersion, SourceVersion: sourceVersion,
		SourceStartedAt: snapshot.SourceStartedAt, SourceCutoffAt: snapshot.SourceCutoffAt,
		RowCount:        int64(len(snapshot.Records)),
		EncryptionKeyID: r.cfg.SnapshotEncryptionKeyID,
		SigningKeyID:    r.cfg.SigningKeyID,
		QualitySummary: &connectorprotocol.SnapshotQualitySummary{
			RowsRead: snapshot.Quality.RowsRead, RecordsEmitted: snapshot.Quality.RecordsEmitted,
			MissingDocumentNumber: snapshot.Quality.MissingDocumentNumber,
			InvalidDocumentNumber: snapshot.Quality.InvalidDocumentNumber,
			MissingPhone:          snapshot.Quality.MissingPhone, InvalidPhone: snapshot.Quality.InvalidPhone,
			MissingEnrollmentYear: snapshot.Quality.MissingEnrollmentYear,
			InvalidEnrollmentYear: snapshot.Quality.InvalidEnrollmentYear,
		},
	}
	envelope, err := r.client.EncryptSnapshot(manifest, payload)
	if err != nil {
		state.failure("snapshot_encryption_failed")
		return "snapshot_encryption_failed"
	}
	uploadCtx, uploadCancel := context.WithTimeout(ctx, timeout)
	err = r.client.UploadSnapshot(uploadCtx, *envelope)
	uploadCancel()
	envelope.Ciphertext = ""
	envelope.Signature = ""
	envelope.EphemeralPublicKey = ""
	envelope.Nonce = ""
	if err != nil {
		r.logger.Warn("campus connector snapshot upload failed",
			"operation", operation.Key, "error_class", "central_unavailable")
		return "central_unavailable"
	}
	state.success()
	r.logger.Info("campus connector roster snapshot uploaded",
		"operation", operation.Key, "row_count", len(snapshot.Records))
	return campusconnector.ResultSuccess
}

func (state *operationState) isOpen(now time.Time) bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.openUntil.After(now)
}

func (state *operationState) begin(now time.Time) bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.running || state.openUntil.After(now) {
		return false
	}
	state.running = true
	return true
}

func (state *operationState) end() {
	state.mu.Lock()
	state.running = false
	state.mu.Unlock()
}

func (state *operationState) success() {
	state.mu.Lock()
	state.consecutiveFails = 0
	state.openUntil = time.Time{}
	state.healthCode = "ok"
	state.mu.Unlock()
}

func (state *operationState) failure(code string) {
	state.mu.Lock()
	state.consecutiveFails++
	state.healthCode = code
	if state.consecutiveFails >= 5 {
		state.openUntil = time.Now().Add(30 * time.Second)
	}
	state.mu.Unlock()
}

func (state *operationState) currentHealth(fallback string) string {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.healthCode == "" || state.healthCode == "upstream_unavailable" {
		return fallback
	}
	return state.healthCode
}

func connectorDependencyFailure(code string) bool {
	return code == campusconnector.ResultUnavailable || code == campusconnector.ResultTLSFailure ||
		code == campusconnector.ResultSchemaUnknown || code == campusconnector.ResultCancelled
}

func classifyRosterFailure(err error) string {
	value := strings.ToLower(err.Error())
	if strings.Contains(value, "certificate") || strings.Contains(value, "tls") {
		return "tls_failure"
	}
	if strings.Contains(value, "identity") || strings.Contains(value, "column") ||
		strings.Contains(value, "row") || strings.Contains(value, "invalid") {
		return "schema_unknown"
	}
	return "upstream_unavailable"
}

func clearConnectorRecords(records []connectorprotocol.RosterRecord) {
	for index := range records {
		records[index].StudentID = ""
		records[index].Name = ""
		records[index].DocumentType = ""
		records[index].DocumentNumber = ""
		records[index].Phone = ""
		records[index].StudentStatus = ""
		records[index].OnCampusStatus = ""
		records[index].RegistrationStatus = ""
		records[index].EducationLevel = ""
		records[index].StudentCategory = ""
		records[index].EligibilityCode = ""
	}
}

func safeRequestReference(requestID string) string {
	digest := sha256.Sum256([]byte("campus-connector-node-request-reference:v1\x00" + requestID))
	return "cc-" + hex.EncodeToString(digest[:6])
}

func waitContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
