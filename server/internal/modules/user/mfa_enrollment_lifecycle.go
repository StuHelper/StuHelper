package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

type MFAEnrollmentComplete struct {
	UserID  int64
	Methods []string
}

func (m *MFARecoveryManager) CompleteEnrollment(
	ctx context.Context,
	params MFAEnrollmentComplete,
) (*MFARecoveryCodeBundle, error) {
	if params.UserID <= 0 {
		return nil, ErrMFARecoveryUserInvalid
	}
	methods, err := normalizeMFAMethods(params.Methods, true)
	if err != nil {
		return nil, err
	}
	codes, hashes, err := m.generateRecoveryCodeHashes(params.UserID)
	if err != nil {
		return nil, err
	}
	issuedAt := m.now().UTC()
	err = m.completeEnrollmentTx(ctx, mfaEnrollmentCompleteTx{
		UserID: params.UserID, Methods: methods, CodeHashes: hashes, IssuedAt: issuedAt,
	})
	if err != nil {
		return nil, err
	}
	audit.Log(audit.EventFromContext(ctx, mfaRecoveryAuditEvent(mfaRecoveryAuditInput{
		UserID: params.UserID,
		Action: "enroll",
		Result: "success",
	})))
	return &MFARecoveryCodeBundle{Codes: codes, IssuedAt: issuedAt}, nil
}

func (m *MFARecoveryManager) completeEnrollmentTx(
	ctx context.Context,
	params mfaEnrollmentCompleteTx,
) error {
	err := m.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := m.repo.UpsertMFAEnrollmentTx(ctx, tx, MFAEnrollmentUpsert{
			UserID: params.UserID, Active: true, Methods: params.Methods, RecoveryCodesIssuedAt: &params.IssuedAt,
		}); err != nil {
			return fmt.Errorf("complete mfa enrollment: %w", err)
		}
		return m.repo.ReplaceMFARecoveryCodesTx(ctx, tx, MFARecoveryCodeReplace{
			UserID: params.UserID, CodeHashes: params.CodeHashes, IssuedAt: params.IssuedAt,
		})
	})
	if err != nil {
		return fmt.Errorf("complete mfa enrollment transaction: %w", err)
	}
	return nil
}

type mfaEnrollmentCompleteTx struct {
	UserID     int64
	Methods    []string
	CodeHashes []string
	IssuedAt   time.Time
}
