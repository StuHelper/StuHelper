package campusconnector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

const (
	heartbeatPath      = "/v1/heartbeat"
	pollPath           = "/v1/poll"
	resultPrefix       = "/v1/results/"
	rosterResultPrefix = "/v1/roster-results/"
	snapshotPath       = "/v1/snapshots"
)

type GatewayConfig struct {
	PollWait     time.Duration
	MaxJSONBytes int64
}

type Gateway struct {
	service *Service
	config  GatewayConfig
	handler http.Handler
}

func NewGateway(service *Service, cfg GatewayConfig) (*Gateway, error) {
	if service == nil {
		return nil, errors.New("campus connector gateway service is required")
	}
	if cfg.PollWait <= 0 || cfg.PollWait > 60*time.Second {
		cfg.PollWait = 25 * time.Second
	}
	if cfg.MaxJSONBytes <= 0 {
		cfg.MaxJSONBytes = 192 << 20
	}
	gateway := &Gateway{service: service, config: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc(heartbeatPath, gateway.handleHeartbeat)
	mux.HandleFunc(pollPath, gateway.handlePoll)
	mux.HandleFunc(resultPrefix, gateway.handleResult)
	mux.HandleFunc(rosterResultPrefix, gateway.handleRosterResult)
	mux.HandleFunc(snapshotPath, gateway.handleSnapshot)
	gateway.handler = securityHeaders(mux)
	return gateway, nil
}

func (g *Gateway) Handler() http.Handler { return g.handler }

func (g *Gateway) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayError(w, http.StatusMethodNotAllowed)
		return
	}
	body, err := readBoundedBody(w, r, 128<<10)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	defer wipe(body)
	node, err := g.service.VerifyGatewayRequest(r.Context(), r, body, true)
	if err != nil {
		writeGatewayError(w, http.StatusUnauthorized)
		return
	}
	var heartbeat connectorprotocol.Heartbeat
	if err := decodeStrictJSON(body, &heartbeat); err != nil {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	if err := g.service.RecordHeartbeat(r.Context(), *node, heartbeat); err != nil {
		writeGatewayError(w, http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *Gateway) handlePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.ContentLength > 0 {
		writeGatewayError(w, http.StatusMethodNotAllowed)
		return
	}
	node, err := g.service.VerifyGatewayRequest(r.Context(), r, nil, false)
	if err != nil {
		writeGatewayError(w, http.StatusUnauthorized)
		return
	}
	rosterCommand, err := g.service.ClaimManualRosterSync(r.Context(), node.ID)
	if err != nil {
		writeGatewayError(w, http.StatusServiceUnavailable)
		return
	}
	if rosterCommand != nil {
		payload, encodeErr := json.Marshal(rosterCommand)
		if encodeErr != nil {
			writeGatewayError(w, http.StatusInternalServerError)
			return
		}
		payload = append(payload, '\n')
		w.Header().Set("Content-Type", connectorprotocol.RosterSyncContentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		// A failed write leaves the durable command claimed only until its short
		// lease expires, after which another poll can safely claim it again.
		if _, writeErr := w.Write(payload); writeErr != nil {
			return
		}
		return
	}
	pollCtx, cancel := context.WithTimeout(r.Context(), g.config.PollWait)
	defer cancel()
	delivery, err := g.service.Broker().Claim(pollCtx, node.ID)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		writeGatewayError(w, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", connectorprotocol.InteractiveContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if err := connectorprotocol.WriteInteractiveDelivery(w, delivery.Metadata, delivery.Password); err != nil {
		g.service.Broker().FailDelivery(delivery.Metadata.RequestID)
	}
}

func (g *Gateway) handleRosterResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayError(w, http.StatusMethodNotAllowed)
		return
	}
	requestID := strings.TrimPrefix(r.URL.Path, rosterResultPrefix)
	if requestID == "" || strings.Contains(requestID, "/") {
		writeGatewayError(w, http.StatusNotFound)
		return
	}
	body, err := readBoundedBody(w, r, 16<<10)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	defer wipe(body)
	node, err := g.service.VerifyGatewayRequest(r.Context(), r, body, false)
	if err != nil {
		writeGatewayError(w, http.StatusUnauthorized)
		return
	}
	var result connectorprotocol.RosterSyncResult
	if err := decodeStrictJSON(body, &result); err != nil || result.RequestID != requestID {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	if err := g.service.CompleteManualRosterSync(r.Context(), node.ID, result); err != nil {
		if errors.Is(err, ErrRequestNotFound) {
			writeGatewayError(w, http.StatusGone)
			return
		}
		writeGatewayError(w, http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *Gateway) handleResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayError(w, http.StatusMethodNotAllowed)
		return
	}
	requestID := strings.TrimPrefix(r.URL.Path, resultPrefix)
	if requestID == "" || strings.Contains(requestID, "/") {
		writeGatewayError(w, http.StatusNotFound)
		return
	}
	body, err := readBoundedBody(w, r, 32<<10)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	defer wipe(body)
	node, err := g.service.VerifyGatewayRequest(r.Context(), r, body, false)
	if err != nil {
		writeGatewayError(w, http.StatusUnauthorized)
		return
	}
	var wireResult connectorprotocol.InteractiveResult
	if err := decodeStrictJSON(body, &wireResult); err != nil || wireResult.RequestID != requestID {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	result, err := normalizeConnectorResult(wireResult)
	if err != nil {
		writeGatewayError(w, http.StatusUnprocessableEntity)
		return
	}
	if err := g.service.Broker().Complete(node.ID, requestID, result); err != nil {
		writeGatewayError(w, http.StatusGone)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *Gateway) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayError(w, http.StatusMethodNotAllowed)
		return
	}
	body, err := readBoundedBody(w, r, g.config.MaxJSONBytes)
	if err != nil {
		writeGatewayError(w, http.StatusRequestEntityTooLarge)
		return
	}
	defer wipe(body)
	node, err := g.service.VerifyGatewayRequest(r.Context(), r, body, false)
	if err != nil {
		writeGatewayError(w, http.StatusUnauthorized)
		return
	}
	var envelope connectorprotocol.EncryptedSnapshot
	if err := decodeStrictJSON(body, &envelope); err != nil {
		writeGatewayError(w, http.StatusBadRequest)
		return
	}
	snapshotID, err := g.service.ImportSnapshot(r.Context(), *node, envelope)
	if err != nil {
		writeGatewayError(w, http.StatusUnprocessableEntity)
		return
	}
	writeGatewayJSON(w, http.StatusCreated, map[string]string{
		"snapshotID": snapshotID,
		"reference":  requestDebugReference(envelope.Manifest.RequestID),
	})
}

func readBoundedBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	reader := http.MaxBytesReader(w, r.Body, maxBytes)
	body, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		wipe(body)
		return nil, err
	}
	return body, nil
}

func decodeStrictJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

func writeGatewayError(w http.ResponseWriter, status int) {
	writeGatewayJSON(w, status, map[string]string{"error": "connector_request_rejected"})
}

func writeGatewayJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
