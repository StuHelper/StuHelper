package serviceaccount

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

const wildcardSuffix = "*"

type BootstrapStatus string

const (
	BootstrapUnchanged BootstrapStatus = "unchanged"
	BootstrapCreated   BootstrapStatus = "created"
	BootstrapRotated   BootstrapStatus = "rotated"
)

type BootstrapCredential struct {
	Name      string
	RawToken  string
	Audience  []string
	Scopes    []string
	ExpiresAt *time.Time
}

type BootstrapResult struct {
	ID     int64
	Name   string
	Status BootstrapStatus
}

type credentialStore interface {
	EnsureBootstrapCredential(ctx context.Context, credential BootstrapCredential, tokenHash string) (BootstrapResult, error)
	RevokeCredential(ctx context.Context, name string) (int64, error)
	LoadCredentialByTokenHash(ctx context.Context, tokenHash string) (*credentialRecord, error)
	TouchLastUsed(ctx context.Context, id int64) error
}

type Verifier struct {
	store   credentialStore
	hmacKey []byte
}

func NewVerifier(database *db.DB, hmacKey []byte) (*Verifier, error) {
	if database == nil {
		return nil, fmt.Errorf("service account verifier: database is required")
	}
	return newVerifier(NewRepository(database), hmacKey)
}

func newVerifier(store credentialStore, hmacKey []byte) (*Verifier, error) {
	if store == nil {
		return nil, fmt.Errorf("service account verifier: credential store is required")
	}
	if len(hmacKey) == 0 {
		return nil, fmt.Errorf("service account verifier: HMAC key is required")
	}
	keyCopy := append([]byte(nil), hmacKey...)
	return &Verifier{store: store, hmacKey: keyCopy}, nil
}

func (v *Verifier) EnsureBootstrapCredential(ctx context.Context, input BootstrapCredential) (BootstrapResult, error) {
	normalized, tokenHash, err := v.normalizeBootstrap(input)
	if err != nil {
		return BootstrapResult{}, err
	}

	result, err := v.store.EnsureBootstrapCredential(ctx, normalized, tokenHash)
	if err != nil {
		return BootstrapResult{}, err
	}
	logBootstrapCredential(ctx, result)
	return result, nil
}

func (v *Verifier) Revoke(ctx context.Context, name string) error {
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		return ErrCredentialNotConfigured
	}
	id, err := v.store.RevokeCredential(ctx, normalizedName)
	if errors.Is(err, errCredentialRecordNotFound) {
		return ErrCredentialInvalid
	}
	if err != nil {
		return err
	}
	audit.LogContext(ctx, serviceAccountCredentialAuditEvent(normalizedName, id, "revoked"))
	return nil
}

func (v *Verifier) normalizeBootstrap(input BootstrapCredential) (BootstrapCredential, string, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.RawToken = strings.TrimSpace(input.RawToken)
	input.Audience = compactNonEmpty(input.Audience)
	input.Scopes = compactNonEmpty(input.Scopes)
	if input.Name == "" || input.RawToken == "" || len(input.Audience) == 0 || len(input.Scopes) == 0 {
		return BootstrapCredential{}, "", ErrCredentialNotConfigured
	}
	tokenHash, err := v.hashToken(input.RawToken)
	if err != nil {
		return BootstrapCredential{}, "", err
	}
	return input, tokenHash, nil
}

func (v *Verifier) hashToken(rawToken string) (string, error) {
	if rawToken == "" {
		return "", ErrCredentialInvalid
	}
	tokenHash, err := crypto.HMACHashWithKey(rawToken, v.hmacKey)
	if err != nil {
		return "", fmt.Errorf("hash service account token: %w", err)
	}
	return tokenHash, nil
}

func logBootstrapCredential(ctx context.Context, result BootstrapResult) {
	if result.Status == BootstrapUnchanged {
		return
	}
	audit.LogContext(ctx, serviceAccountCredentialAuditEvent(result.Name, result.ID, string(result.Status)))
}

func serviceAccountCredentialAuditEvent(name string, id int64, action string) audit.Event {
	return audit.Event{
		Type:         audit.EventType("iam.service_account." + action),
		Category:     "admin_operation",
		ActorType:    "system",
		ResourceType: "iam.service_account",
		ResourceID:   name,
		Action:       action,
		Result:       "success",
		Details: map[string]any{
			"credential_id": id,
			"name":          name,
		},
	}
}

func compactNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && !slices.Contains(result, trimmed) {
			result = append(result, trimmed)
		}
	}
	return result
}

func audienceAllowed(audiences []string, required string) bool {
	required = strings.TrimSpace(required)
	for _, audience := range audiences {
		audience = strings.TrimSpace(audience)
		if audience == required {
			return true
		}
		if strings.HasSuffix(audience, wildcardSuffix) && strings.HasPrefix(required, strings.TrimSuffix(audience, wildcardSuffix)) {
			return true
		}
	}
	return false
}
