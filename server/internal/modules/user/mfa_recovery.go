package user

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

const (
	mfaRecoveryCodeCount       = 10
	mfaRecoveryCodeRandomBytes = 20
	mfaRecoveryCodeGroupSize   = 4
	mfaRecoveryCodeHashPrefix  = "mfa_recovery"
)

var (
	ErrMFARecoveryCodeInvalid = errors.New("mfa recovery code is invalid")
)

type MFARecoveryRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
	UpsertMFAEnrollmentTx(ctx context.Context, tx pgx.Tx, params MFAEnrollmentUpsert) error
	UpdateMFAEnrollmentStateTx(ctx context.Context, tx pgx.Tx, params MFAEnrollmentStateChange) error
	ReplaceMFARecoveryCodesTx(ctx context.Context, tx pgx.Tx, params MFARecoveryCodeReplace) error
	ConsumeMFARecoveryCodeTx(ctx context.Context, tx pgx.Tx, params MFARecoveryCodeConsume) (bool, error)
	DeleteMFARecoveryCodesTx(ctx context.Context, tx pgx.Tx, userID int64) error
}

type MFARecoveryCodeReplace struct {
	UserID     int64
	CodeHashes []string
	IssuedAt   time.Time
}

type MFARecoveryCodeConsume struct {
	UserID   int64
	CodeHash string
	UsedAt   time.Time
}

type MFARecoveryCodeBundle struct {
	Codes    []string
	IssuedAt time.Time
}

type MFARecoveryManager struct {
	repo    MFARecoveryRepository
	hmacKey []byte
	now     func() time.Time
}

func NewMFARecoveryManager(repo MFARecoveryRepository, hmacKey []byte) (*MFARecoveryManager, error) {
	if repo == nil {
		return nil, errors.New("mfa recovery manager: repo is required")
	}
	if len(hmacKey) == 0 {
		return nil, errors.New("mfa recovery manager: hmac key is required")
	}
	return &MFARecoveryManager{repo: repo, hmacKey: append([]byte(nil), hmacKey...), now: time.Now}, nil
}

func (m *MFARecoveryManager) IssueRecoveryCodes(ctx context.Context, userID int64) (*MFARecoveryCodeBundle, error) {
	if userID <= 0 {
		return nil, ErrMFARecoveryUserInvalid
	}
	codes, hashes, err := m.generateRecoveryCodeHashes(userID)
	if err != nil {
		return nil, err
	}
	issuedAt := m.now().UTC()
	err = m.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return m.repo.ReplaceMFARecoveryCodesTx(ctx, tx, MFARecoveryCodeReplace{
			UserID:     userID,
			CodeHashes: hashes,
			IssuedAt:   issuedAt,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("issue mfa recovery codes: %w", err)
	}
	audit.LogContext(ctx, mfaRecoveryAuditEvent(mfaRecoveryAuditInput{
		UserID: userID,
		Action: "recovery_codes_issue",
		Result: "success",
	}))
	return &MFARecoveryCodeBundle{Codes: codes, IssuedAt: issuedAt}, nil
}

func (m *MFARecoveryManager) ConsumeRecoveryCode(ctx context.Context, userID int64, code string) error {
	if userID <= 0 {
		return ErrMFARecoveryUserInvalid
	}
	hash, err := m.hashRecoveryCode(userID, code)
	if err != nil {
		m.auditRecoveryUse(ctx, mfaRecoveryAuditFailure(userID))
		return err
	}
	usedAt := m.now().UTC()
	consumed, err := m.consumeRecoveryCodeHash(ctx, MFARecoveryCodeConsume{
		UserID: userID, CodeHash: hash, UsedAt: usedAt,
	})
	if err != nil {
		return err
	}
	if !consumed {
		m.auditRecoveryUse(ctx, mfaRecoveryAuditFailure(userID))
		return ErrMFARecoveryCodeInvalid
	}
	m.auditRecoveryUse(ctx, mfaRecoveryAuditInput{
		UserID: userID,
		Action: "recovery_code_use",
		Result: "success",
	})
	return nil
}

func (m *MFARecoveryManager) consumeRecoveryCodeHash(ctx context.Context, params MFARecoveryCodeConsume) (bool, error) {
	var consumed bool
	err := m.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		consumed, err = m.repo.ConsumeMFARecoveryCodeTx(ctx, tx, params)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("consume mfa recovery code: %w", err)
	}
	return consumed, nil
}

func (m *MFARecoveryManager) generateRecoveryCodeHashes(userID int64) ([]string, []string, error) {
	codes := make([]string, 0, mfaRecoveryCodeCount)
	hashes := make([]string, 0, mfaRecoveryCodeCount)
	seen := make(map[string]struct{}, mfaRecoveryCodeCount)
	for len(codes) < mfaRecoveryCodeCount {
		code, err := generateMFARecoveryCode()
		if err != nil {
			return nil, nil, fmt.Errorf("generate mfa recovery code: %w", err)
		}
		hash, err := m.hashRecoveryCode(userID, code)
		if err != nil {
			return nil, nil, err
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		codes = append(codes, code)
		hashes = append(hashes, hash)
	}
	return codes, hashes, nil
}

func (m *MFARecoveryManager) hashRecoveryCode(userID int64, code string) (string, error) {
	normalized := normalizeMFARecoveryCode(code)
	if normalized == "" {
		return "", ErrMFARecoveryCodeInvalid
	}
	payload := fmt.Sprintf("%s:%d:%s", mfaRecoveryCodeHashPrefix, userID, normalized)
	hash, err := crypto.HMACHashWithKey(payload, m.hmacKey)
	if err != nil {
		return "", fmt.Errorf("hash mfa recovery code: %w", err)
	}
	return hash, nil
}

func (m *MFARecoveryManager) auditRecoveryUse(ctx context.Context, input mfaRecoveryAuditInput) {
	audit.LogContext(ctx, mfaRecoveryAuditEvent(input))
}

type mfaRecoveryAuditInput struct {
	UserID int64
	Action string
	Result string
	Reason string
}

func mfaRecoveryAuditFailure(userID int64) mfaRecoveryAuditInput {
	return mfaRecoveryAuditInput{
		UserID: userID,
		Action: "recovery_code_use",
		Result: "failure",
		Reason: "invalid recovery code",
	}
}

func mfaRecoveryAuditEvent(input mfaRecoveryAuditInput) audit.Event {
	return audit.Event{
		Type:         audit.EventType("iam.mfa." + input.Action),
		Category:     "audit",
		ActorType:    "user",
		UserID:       fmt.Sprintf("%d", input.UserID),
		ResourceType: mfaAuditResourceType(input.Action),
		ResourceID:   fmt.Sprintf("user:%d", input.UserID),
		Action:       input.Action,
		Result:       input.Result,
		Reason:       input.Reason,
	}
}

func mfaAuditResourceType(action string) string {
	if action == "recovery_code_use" || action == "recovery_codes_issue" {
		return "iam.mfa.recovery_code"
	}
	return "iam.mfa"
}

func generateMFARecoveryCode() (string, error) {
	buf := make([]byte, mfaRecoveryCodeRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return groupMFARecoveryCode(encoded), nil
}

func groupMFARecoveryCode(code string) string {
	var builder strings.Builder
	for i, r := range code {
		if i > 0 && i%mfaRecoveryCodeGroupSize == 0 {
			builder.WriteByte('-')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func normalizeMFARecoveryCode(code string) string {
	var builder strings.Builder
	for _, r := range code {
		if r == '-' || unicode.IsSpace(r) {
			continue
		}
		builder.WriteRune(unicode.ToUpper(r))
	}
	return builder.String()
}
