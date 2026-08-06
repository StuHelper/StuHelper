package campusconnector

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

const (
	requestReplayPrefix    = "campus_connector:request_nonce:v1:"
	operationRatePrefix    = "campus_connector:operation_rate:v1:"
	manualRosterSyncTTL    = 24 * time.Hour
	manualRosterClaimLease = 3 * time.Minute
	manualRosterMaxClaims  = 5
)

type SchoolAccountInput struct {
	SchoolID       int64
	OperationKey   string
	AdapterID      string
	AdapterVersion string
	StudentID      string
	Password       []byte
	ApplicationID  *string
}

type SchoolAccountResult struct {
	AccountSubject string
	StudentID      string
}

type ManualRosterSyncInput struct {
	SchoolCode  string
	ActorUserID int64
	Reason      string
}

type SnapshotImportRequest struct {
	Manifest  connectorprotocol.SnapshotManifest
	Payload   connectorprotocol.RosterSnapshotPayload
	NodeID    string
	Signature []byte
}

type SnapshotImporter interface {
	ImportCampusConnectorSnapshot(ctx context.Context, request SnapshotImportRequest) (snapshotID string, err error)
}

type Config struct {
	ProtocolVersion         string
	RequestSignatureMaxSkew time.Duration
	RequestReplayTTL        time.Duration
	SnapshotKeyID           string
	SnapshotPrivateKey      []byte
	MaxSnapshotPlaintext    int
	MaxInteractivePassword  int
}

type Service struct {
	repo        *Repository
	broker      *Broker
	redis       *redis.Client
	importer    SnapshotImporter
	config      Config
	now         func() time.Time
	semaphoreMu sync.Mutex
	semaphores  map[string]chan struct{}
}

func NewService(repo *Repository, broker *Broker, redisClient *redis.Client, importer SnapshotImporter, cfg Config) (*Service, error) {
	if repo == nil || broker == nil || redisClient == nil {
		return nil, errors.New("campus connector service dependencies are required")
	}
	if cfg.ProtocolVersion == "" {
		cfg.ProtocolVersion = connectorprotocol.ProtocolVersion
	}
	if cfg.ProtocolVersion != connectorprotocol.ProtocolVersion {
		return nil, errors.New("unsupported campus connector protocol version")
	}
	if cfg.RequestSignatureMaxSkew <= 0 {
		cfg.RequestSignatureMaxSkew = 2 * time.Minute
	}
	if cfg.RequestReplayTTL < cfg.RequestSignatureMaxSkew*2 {
		cfg.RequestReplayTTL = cfg.RequestSignatureMaxSkew * 3
	}
	if cfg.MaxSnapshotPlaintext <= 0 {
		cfg.MaxSnapshotPlaintext = 128 << 20
	}
	if cfg.MaxInteractivePassword <= 0 || cfg.MaxInteractivePassword > 4096 {
		cfg.MaxInteractivePassword = 1024
	}
	cfg.SnapshotPrivateKey = append([]byte(nil), cfg.SnapshotPrivateKey...)
	return &Service{
		repo: repo, broker: broker, redis: redisClient, importer: importer,
		config: cfg, now: time.Now, semaphores: make(map[string]chan struct{}),
	}, nil
}

func (s *Service) AuthenticateSchoolAccount(ctx context.Context, input SchoolAccountInput) (*SchoolAccountResult, error) {
	if input.SchoolID <= 0 || input.OperationKey == "" || input.AdapterID == "" ||
		input.AdapterVersion == "" || input.StudentID == "" || len(input.Password) == 0 ||
		len(input.Password) > s.config.MaxInteractivePassword {
		return nil, ErrRejected
	}
	now := s.now().UTC()
	operation, err := s.repo.ResolveSchoolAccountOperation(
		ctx, input.SchoolID, input.OperationKey, input.AdapterID, input.AdapterVersion, now,
	)
	if err != nil {
		return nil, ErrUnavailable
	}
	if err := s.consumeOperationRate(ctx, operation); err != nil {
		return nil, ErrUnavailable
	}
	release, err := s.acquireOperation(ctx, operation)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer release()
	timeout := time.Duration(operation.TimeoutMilliseconds) * time.Millisecond
	if timeout <= 0 || timeout > 2*time.Minute {
		timeout = 8 * time.Second
	}
	deadline := now.Add(timeout)
	requestID := uuid.NewString()
	request := InteractiveRequest{
		ID: requestID, NodeID: operation.NodeID, SchoolID: operation.SchoolID,
		SchoolCode:   operation.SchoolCode,
		OperationKey: operation.OperationKey, AdapterID: operation.AdapterID,
		AdapterVersion: operation.AdapterVersion, StudentID: input.StudentID,
		Password: input.Password, DeadlineAt: deadline, ApplicationID: input.ApplicationID,
	}
	if err := s.repo.CreateInteractiveRequest(ctx, request, now); err != nil {
		return nil, ErrUnavailable
	}
	result, brokerErr := s.broker.Submit(ctx, request)
	finishedAt := s.now().UTC()
	if brokerErr != nil {
		if completeErr := s.repo.CompleteRequest(
			context.WithoutCancel(ctx), requestID, "connector_timeout", false, finishedAt,
		); completeErr != nil {
			return nil, errors.Join(ErrUnavailable, fmt.Errorf("record connector timeout: %w", completeErr))
		}
		return nil, ErrUnavailable
	}
	succeeded := result.ResultCode == ResultSuccess
	if err := s.repo.CompleteRequest(context.WithoutCancel(ctx), requestID, result.ResultCode, succeeded, finishedAt); err != nil {
		return nil, ErrUnavailable
	}
	switch result.ResultCode {
	case ResultSuccess:
		if strings.TrimSpace(result.AccountSubject) == "" || strings.TrimSpace(result.StudentID) == "" {
			return nil, ErrInvalidResult
		}
		return &SchoolAccountResult{
			AccountSubject: strings.TrimSpace(result.AccountSubject),
			StudentID:      strings.TrimSpace(result.StudentID),
		}, nil
	case ResultRejected:
		return nil, ErrRejected
	case ResultAccountLocked:
		return nil, ErrAccountLocked
	case ResultNotStudent:
		return nil, ErrNotStudent
	default:
		return nil, ErrUnavailable
	}
}

func (s *Service) RequestManualRosterSync(
	ctx context.Context,
	input ManualRosterSyncInput,
) (*ManualRosterSyncRequest, error) {
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.Reason = strings.TrimSpace(input.Reason)
	if !registrySchoolCodePattern.MatchString(input.SchoolCode) || input.ActorUserID <= 0 ||
		utf8.RuneCountInString(input.Reason) < 4 || utf8.RuneCountInString(input.Reason) > 500 {
		return nil, ErrRejected
	}
	now := s.now().UTC()
	operation, err := s.repo.ResolveRosterOperationForSchool(ctx, input.SchoolCode, now)
	if err != nil {
		return nil, ErrUnavailable
	}
	request := ManualRosterSyncRequest{
		ID: uuid.NewString(), NodeID: operation.NodeID, SchoolID: operation.SchoolID,
		SchoolCode: operation.SchoolCode, OperationKey: operation.OperationKey,
		AdapterID: operation.AdapterID, AdapterVersion: operation.AdapterVersion,
		Status: "pending", ActorUserID: input.ActorUserID, Reason: input.Reason,
		DeadlineAt: now.Add(manualRosterSyncTTL), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateManualRosterSyncRequest(ctx, request); err != nil {
		return nil, err
	}
	return &request, nil
}

func (s *Service) ListManualRosterSyncRequests(
	ctx context.Context,
	schoolCode string,
	limit int,
) ([]ManualRosterSyncRequest, error) {
	schoolCode = strings.TrimSpace(schoolCode)
	if !registrySchoolCodePattern.MatchString(schoolCode) || limit < 1 || limit > 50 {
		return nil, ErrRejected
	}
	return s.repo.ListManualRosterSyncRequests(ctx, schoolCode, limit)
}

func (s *Service) ClaimManualRosterSync(
	ctx context.Context,
	nodeID string,
) (*connectorprotocol.RosterSyncCommand, error) {
	request, err := s.repo.ClaimManualRosterSyncRequest(
		ctx, strings.TrimSpace(nodeID), s.now().UTC(),
		manualRosterClaimLease, manualRosterMaxClaims,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &connectorprotocol.RosterSyncCommand{
		RequestID: request.ID, SchoolCode: request.SchoolCode,
		OperationKey: request.OperationKey, AdapterID: request.AdapterID,
		AdapterVersion: request.AdapterVersion, DeadlineAt: request.DeadlineAt,
	}, nil
}

func (s *Service) CompleteManualRosterSync(
	ctx context.Context,
	nodeID string,
	result connectorprotocol.RosterSyncResult,
) error {
	if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(result.RequestID) == "" ||
		!validRosterFailureCode(result.ResultCode) {
		return ErrInvalidResult
	}
	return s.repo.CompleteManualRosterSyncRequest(
		ctx, strings.TrimSpace(nodeID), strings.TrimSpace(result.RequestID),
		result.ResultCode, s.now().UTC(),
	)
}

func (s *Service) VerifyGatewayRequest(ctx context.Context, req *http.Request, body []byte, allowRegistered bool) (*NodeIdentity, error) {
	if req == nil || req.TLS == nil || len(req.TLS.PeerCertificates) < 1 || len(req.TLS.VerifiedChains) < 1 {
		return nil, ErrAuthentication
	}
	nodeID := strings.TrimSpace(req.Header.Get(connectorprotocol.HeaderNodeID))
	if nodeID == "" || req.Header.Get(connectorprotocol.HeaderProtocolVersion) != s.config.ProtocolVersion {
		return nil, ErrAuthentication
	}
	node, err := s.repo.GetNodeIdentity(ctx, nodeID)
	if err != nil || node.RevokedAt != nil || node.Status == "revoked" || !node.CertificateNotAfter.After(s.now()) {
		return nil, ErrAuthentication
	}
	if node.Status == "registered" && !allowRegistered {
		return nil, ErrAuthentication
	}
	if node.Status != "registered" && node.Status != "active" && node.Status != "degraded" {
		return nil, ErrAuthentication
	}
	if node.ProtocolVersion != s.config.ProtocolVersion ||
		req.Header.Get(connectorprotocol.HeaderSigningKeyID) != node.SigningKeyID {
		return nil, ErrAuthentication
	}
	if !certificateMatchesNode(req.TLS.PeerCertificates[0], *node, s.now()) {
		return nil, ErrAuthentication
	}
	timestamp, err := strconv.ParseInt(req.Header.Get(connectorprotocol.HeaderTimestamp), 10, 64)
	if err != nil {
		return nil, ErrAuthentication
	}
	requestTime := time.Unix(timestamp, 0)
	if absoluteDuration(s.now().Sub(requestTime)) > s.config.RequestSignatureMaxSkew {
		return nil, ErrAuthentication
	}
	nonce := strings.TrimSpace(req.Header.Get(connectorprotocol.HeaderNonce))
	decodedNonce, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil || len(decodedNonce) < 16 || len(decodedNonce) > 64 {
		return nil, ErrAuthentication
	}
	publicKey := ed25519.PublicKey(node.SigningPublicKey)
	if !connectorprotocol.VerifyRequestSignature(req, publicKey, body) {
		return nil, ErrAuthentication
	}
	replayKey := requestReplayPrefix + node.ID + ":" + nonce
	accepted, err := s.redis.SetNX(ctx, replayKey, "1", s.config.RequestReplayTTL).Result()
	if err != nil {
		return nil, ErrAuthentication
	}
	if !accepted {
		return nil, ErrReplay
	}
	return node, nil
}

func (s *Service) RecordHeartbeat(ctx context.Context, node NodeIdentity, heartbeat connectorprotocol.Heartbeat) error {
	if strings.TrimSpace(heartbeat.SoftwareVersion) == "" || len(heartbeat.SoftwareVersion) > 128 ||
		heartbeat.ProtocolVersion != s.config.ProtocolVersion || len(heartbeat.Operations) > 256 {
		return ErrInvalidResult
	}
	seen := make(map[string]struct{}, len(heartbeat.Operations))
	for _, operation := range heartbeat.Operations {
		if operation.SchoolCode == "" || operation.OperationKey == "" || operation.HealthCode == "" {
			return ErrInvalidResult
		}
		operationIdentity := operation.SchoolCode + "\x00" + operation.OperationKey
		if _, duplicate := seen[operationIdentity]; duplicate {
			return ErrInvalidResult
		}
		seen[operationIdentity] = struct{}{}
	}
	return s.repo.RecordHeartbeat(ctx, node, heartbeat.SoftwareVersion,
		heartbeat.ProtocolVersion, heartbeat.Operations, s.now().UTC())
}

func (s *Service) ImportSnapshot(
	ctx context.Context,
	node NodeIdentity,
	envelope connectorprotocol.EncryptedSnapshot,
) (string, error) {
	manifest := envelope.Manifest
	if s.importer == nil || len(s.config.SnapshotPrivateKey) != 32 ||
		manifest.SchemaVersion != 1 || manifest.RequestID == "" || manifest.NodeID != node.ID ||
		manifest.SigningKeyID != node.SigningKeyID || manifest.EncryptionKeyID != s.config.SnapshotKeyID ||
		manifest.SourceStartedAt.IsZero() || manifest.SourceCutoffAt.IsZero() ||
		manifest.SourceCutoffAt.Before(manifest.SourceStartedAt) || manifest.RowCount < 0 {
		return "", ErrSnapshotRejected
	}
	operation, err := s.repo.GetRosterOperation(
		ctx, node.ID, manifest.SchoolCode, manifest.OperationKey,
		manifest.AdapterID, manifest.AdapterVersion,
	)
	if err != nil || operation.NodeProtocolVersion != s.config.ProtocolVersion {
		return "", ErrSnapshotRejected
	}
	now := s.now().UTC()
	deadline := now.Add(time.Duration(operation.TimeoutMilliseconds) * time.Millisecond)
	if !deadline.After(now) {
		deadline = now.Add(2 * time.Minute)
	}
	if err := s.repo.EnsureSnapshotRequest(
		ctx, manifest.RequestID, node.ID, operation.SchoolID,
		operation.OperationKey, deadline, now,
	); err != nil {
		return "", ErrSnapshotRejected
	}
	plaintext, err := connectorprotocol.DecryptSnapshot(
		envelope, s.config.SnapshotPrivateKey, ed25519.PublicKey(node.SigningPublicKey),
		s.config.MaxSnapshotPlaintext,
	)
	if err != nil {
		return "", s.rejectSnapshotRequest(ctx, manifest.RequestID, snapshotFailureCode(err))
	}
	defer wipe(plaintext)
	var payload connectorprotocol.RosterSnapshotPayload
	decoder := json.NewDecoder(bytes.NewReader(plaintext))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || !errors.Is(decoder.Decode(&struct{}{}), io.EOF) ||
		int64(len(payload.Records)) != manifest.RowCount {
		wipeRosterPayload(&payload)
		return "", s.rejectSnapshotRequest(ctx, manifest.RequestID, "snapshot_payload_invalid")
	}
	defer wipeRosterPayload(&payload)
	signature, err := base64.RawStdEncoding.DecodeString(envelope.Signature)
	if err != nil {
		return "", s.rejectSnapshotRequest(ctx, manifest.RequestID, "snapshot_signature_invalid")
	}
	snapshotID, err := s.importer.ImportCampusConnectorSnapshot(ctx, SnapshotImportRequest{
		Manifest: manifest, Payload: payload, NodeID: node.ID, Signature: signature,
	})
	if err != nil || snapshotID == "" {
		return "", s.rejectSnapshotRequest(ctx, manifest.RequestID, "snapshot_import_failed")
	}
	if err := s.repo.AttachSnapshotResult(
		context.WithoutCancel(ctx), manifest.RequestID, uuid.NewString(), snapshotID,
		manifest.SchemaVersion, manifest.PlaintextSHA256, manifest.SigningKeyID, s.now().UTC(),
	); err != nil {
		return "", ErrSnapshotRejected
	}
	return snapshotID, nil
}

func (s *Service) rejectSnapshotRequest(ctx context.Context, requestID string, resultCode string) error {
	if err := s.repo.CompleteRequest(
		context.WithoutCancel(ctx), requestID, resultCode, false, s.now().UTC(),
	); err != nil {
		return errors.Join(ErrSnapshotRejected, fmt.Errorf("record rejected connector snapshot: %w", err))
	}
	return ErrSnapshotRejected
}

func (s *Service) consumeOperationRate(ctx context.Context, operation *SchoolOperation) error {
	window := s.now().UTC().Format("200601021504")
	key := operationRatePrefix + operation.NodeID + ":" + operation.OperationKey + ":" + window
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if count == 1 {
		if err := s.redis.Expire(ctx, key, 2*time.Minute).Err(); err != nil {
			return err
		}
	}
	if count > int64(operation.RateLimitPerMinute) {
		return ErrUnavailable
	}
	return nil
}

func (s *Service) acquireOperation(ctx context.Context, operation *SchoolOperation) (func(), error) {
	limit := operation.MaxConcurrency
	if operation.NodeMaxConcurrency < limit {
		limit = operation.NodeMaxConcurrency
	}
	if limit <= 0 {
		return nil, ErrUnavailable
	}
	key := operation.NodeID + "\x00" + operation.OperationKey
	s.semaphoreMu.Lock()
	semaphore := s.semaphores[key]
	if semaphore == nil {
		semaphore = make(chan struct{}, limit)
		s.semaphores[key] = semaphore
	}
	s.semaphoreMu.Unlock()
	select {
	case semaphore <- struct{}{}:
		return func() { <-semaphore }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func certificateMatchesNode(cert *x509.Certificate, node NodeIdentity, now time.Time) bool {
	if cert == nil || now.Before(cert.NotBefore) || !now.Before(cert.NotAfter) {
		return false
	}
	digest := sha256.Sum256(cert.Raw)
	expected, err := hex.DecodeString(node.CertificateFingerprint)
	if err != nil || len(expected) != len(digest) {
		return false
	}
	return subtle.ConstantTimeCompare(digest[:], expected) == 1
}

func absoluteDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func snapshotFailureCode(err error) string {
	switch {
	case errors.Is(err, connectorprotocol.ErrSnapshotSignature):
		return "snapshot_signature_invalid"
	case errors.Is(err, connectorprotocol.ErrSnapshotDecryption):
		return "snapshot_decryption_failed"
	default:
		return "snapshot_envelope_invalid"
	}
}

func wipeRosterPayload(payload *connectorprotocol.RosterSnapshotPayload) {
	if payload == nil {
		return
	}
	for index := range payload.Records {
		record := &payload.Records[index]
		record.StudentID = ""
		record.Name = ""
		record.DocumentType = ""
		record.DocumentNumber = ""
		record.Phone = ""
		record.StudentStatus = ""
		record.OnCampusStatus = ""
		record.RegistrationStatus = ""
		record.EducationLevel = ""
		record.StudentCategory = ""
		record.EligibilityCode = ""
	}
	payload.Records = nil
}

func (s *Service) Broker() *Broker { return s.broker }

func (s *Service) SetSnapshotImporter(importer SnapshotImporter) {
	s.importer = importer
}

func (s *Service) MaxInteractivePasswordBytes() int { return s.config.MaxInteractivePassword }

func (s *Service) Close() {
	s.broker.Close()
	wipe(s.config.SnapshotPrivateKey)
	s.config.SnapshotPrivateKey = nil
}

func normalizeConnectorResult(result connectorprotocol.InteractiveResult) (InteractiveResult, error) {
	if result.RequestID == "" {
		return InteractiveResult{}, ErrInvalidResult
	}
	switch result.ResultCode {
	case ResultSuccess, ResultRejected, ResultAccountLocked, ResultNotStudent,
		ResultUnavailable, ResultTLSFailure, ResultSchemaUnknown, ResultCancelled:
	default:
		return InteractiveResult{}, ErrInvalidResult
	}
	if result.ResultCode == ResultSuccess &&
		(strings.TrimSpace(result.AccountSubject) == "" || strings.TrimSpace(result.StudentID) == "") {
		return InteractiveResult{}, ErrInvalidResult
	}
	return InteractiveResult{
		ResultCode: result.ResultCode, AccountSubject: strings.TrimSpace(result.AccountSubject),
		StudentID: strings.TrimSpace(result.StudentID),
	}, nil
}

func validRosterFailureCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "secret_unavailable", "tls_failure", "schema_unknown",
		"upstream_unavailable", "snapshot_encryption_failed", "cancelled":
		return true
	default:
		return false
	}
}

func requestDebugReference(requestID string) string {
	digest := sha256.Sum256([]byte("campus-connector-debug-reference:v1\x00" + requestID))
	return fmt.Sprintf("cc-%s", hex.EncodeToString(digest[:6]))
}
