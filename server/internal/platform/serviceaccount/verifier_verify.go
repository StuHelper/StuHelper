package serviceaccount

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	serviceAccountCallAction  = "call"
	auditResultSuccess        = "success"
	auditResultFailure        = "failure"
	invalidCredentialReason   = "invalid credential"                    // #nosec G101 -- audit reason, not a secret.
	forbiddenCredentialReason = "credential audience or scope denied"   // #nosec G101 -- audit reason, not a secret.
	revokedCredentialReason   = "credential revoked"                    // #nosec G101 -- audit reason, not a secret.
	expiredCredentialReason   = "credential expired"                    // #nosec G101 -- audit reason, not a secret.
	storeUnavailableReason    = "credential store unavailable"          // #nosec G101 -- audit reason, not a secret.
	usageTrackingReason       = "credential usage tracking unavailable" // #nosec G101 -- audit reason, not a secret.
)

type verifyInput struct {
	Audience        string
	Scope           string
	TokenHashPrefix string
}

type serviceAccountCallAudit struct {
	Input      verifyInput
	Credential *credentialRecord
	Result     string
	Reason     string
}

func (v *Verifier) Verify(ctx context.Context, rawToken, audience, scope string) error {
	tokenHash, err := v.hashToken(strings.TrimSpace(rawToken))
	if err != nil {
		return err
	}

	input := verifyInput{
		Audience:        strings.TrimSpace(audience),
		Scope:           strings.TrimSpace(scope),
		TokenHashPrefix: hashPrefix(tokenHash),
	}
	record, err := v.store.LoadCredentialByTokenHash(ctx, tokenHash)
	if err != nil {
		return v.handleCredentialLoadError(ctx, input, err)
	}
	if err := v.validateCredential(ctx, input, record); err != nil {
		return err
	}
	if err := v.store.TouchLastUsed(ctx, record.ID); err != nil {
		logger.L().Warn(
			"touch service account credential usage failed",
			zap.Error(err),
			zap.Int64("credential_id", record.ID),
			zap.String("credential_name", record.Name),
			zap.String("audience", input.Audience),
			zap.String("scope", input.Scope),
		)
		logServiceAccountCall(ctx, input, record, auditResultSuccess, usageTrackingReason)
		return nil
	}

	logServiceAccountCall(ctx, input, record, auditResultSuccess, "")
	return nil
}

func (v *Verifier) handleCredentialLoadError(ctx context.Context, input verifyInput, err error) error {
	if errors.Is(err, errCredentialRecordNotFound) {
		logServiceAccountCall(ctx, input, nil, auditResultFailure, invalidCredentialReason)
		return ErrCredentialInvalid
	}
	logServiceAccountCall(ctx, input, nil, auditResultFailure, storeUnavailableReason)
	return fmt.Errorf("%w: %v", ErrCredentialStoreUnavailable, err)
}

func (v *Verifier) validateCredential(ctx context.Context, input verifyInput, record *credentialRecord) error {
	switch {
	case record.RevokedAt.Valid:
		logServiceAccountCall(ctx, input, record, auditResultFailure, revokedCredentialReason)
		return ErrCredentialInvalid
	case isExpired(record.ExpiresAt):
		logServiceAccountCall(ctx, input, record, auditResultFailure, expiredCredentialReason)
		return ErrCredentialInvalid
	case !credentialAllows(record, input):
		logServiceAccountCall(ctx, input, record, auditResultFailure, forbiddenCredentialReason)
		return ErrCredentialForbidden
	default:
		return nil
	}
}

func credentialAllows(record *credentialRecord, input verifyInput) bool {
	return audienceAllowed(record.Audiences, input.Audience) && slices.Contains(record.Scopes, input.Scope)
}

func logServiceAccountCall(ctx context.Context, input verifyInput, record *credentialRecord, result, reason string) {
	audit.LogContext(ctx, serviceAccountCallAuditEvent(serviceAccountCallAudit{
		Input:      input,
		Credential: record,
		Result:     result,
		Reason:     reason,
	}))
}

func serviceAccountCallAuditEvent(entry serviceAccountCallAudit) audit.Event {
	details := serviceAccountCallDetails(entry)
	return audit.Event{
		Type:         audit.EventType("iam.service_account.call"),
		Category:     "admin_operation",
		ActorType:    "service_account",
		ResourceType: "iam.service_account",
		ResourceID:   credentialResourceID(entry.Credential),
		Action:       serviceAccountCallAction,
		Result:       entry.Result,
		Reason:       entry.Reason,
		Details:      details,
	}
}

func serviceAccountCallDetails(entry serviceAccountCallAudit) map[string]any {
	details := map[string]any{
		"audience": entry.Input.Audience,
		"scope":    entry.Input.Scope,
	}
	if entry.Credential != nil {
		details["credential_id"] = entry.Credential.ID
		details["name"] = entry.Credential.Name
		return details
	}
	details["token_hash_prefix"] = entry.Input.TokenHashPrefix
	return details
}

func credentialResourceID(record *credentialRecord) string {
	if record == nil {
		return "unknown"
	}
	return record.Name
}

func hashPrefix(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func isExpired(expiresAt sql.NullTime) bool {
	return expiresAt.Valid && !expiresAt.Time.After(time.Now())
}
