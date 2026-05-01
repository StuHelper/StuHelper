package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

type MFAEnrollmentAdminAction struct {
	ActorUserID  int64
	TargetUserID int64
}

func (m *MFARecoveryManager) DisableEnrollment(ctx context.Context, params MFAEnrollmentAdminAction) error {
	return m.mutateEnrollment(ctx, mfaEnrollmentMutation{
		ActorUserID:  params.ActorUserID,
		TargetUserID: params.TargetUserID,
		Action:       MFAEnrollmentChangeDisable,
		AuditAction:  "disable",
	})
}

func (m *MFARecoveryManager) ResetEnrollment(ctx context.Context, params MFAEnrollmentAdminAction) error {
	return m.mutateEnrollment(ctx, mfaEnrollmentMutation{
		ActorUserID:   params.ActorUserID,
		TargetUserID:  params.TargetUserID,
		Action:        MFAEnrollmentChangeReset,
		ResetRequired: true,
		AuditAction:   "reset",
	})
}

type mfaEnrollmentMutation struct {
	ActorUserID   int64
	TargetUserID  int64
	Action        MFAEnrollmentChangeAction
	ResetRequired bool
	AuditAction   string
}

func (m *MFARecoveryManager) mutateEnrollment(ctx context.Context, params mfaEnrollmentMutation) error {
	if params.ActorUserID <= 0 || params.TargetUserID <= 0 {
		return ErrMFAUserInvalid
	}
	changedAt := m.now().UTC()
	err := m.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := m.repo.UpdateMFAEnrollmentStateTx(ctx, tx, MFAEnrollmentStateChange{
			UserID: params.TargetUserID, ChangedAt: changedAt, Action: params.Action,
			ResetRequired: params.ResetRequired,
		}); err != nil {
			return err
		}
		return m.repo.DeleteMFARecoveryCodesTx(ctx, tx, params.TargetUserID)
	})
	if err != nil {
		m.auditEnrollmentMutation(ctx, params, mfaEnrollmentAuditOutcome{Result: "failure", Reason: err.Error()})
		return fmt.Errorf("mfa enrollment %s: %w", params.AuditAction, err)
	}
	m.auditEnrollmentMutation(ctx, params, mfaEnrollmentAuditOutcome{Result: "success"})
	return nil
}

func (m *MFARecoveryManager) auditEnrollmentMutation(
	ctx context.Context,
	params mfaEnrollmentMutation,
	outcome mfaEnrollmentAuditOutcome,
) {
	audit.Log(audit.EventFromContext(ctx, mfaEnrollmentMutationAuditEvent(params, outcome)))
}

type mfaEnrollmentAuditOutcome struct {
	Result string
	Reason string
}

func mfaEnrollmentMutationAuditEvent(params mfaEnrollmentMutation, outcome mfaEnrollmentAuditOutcome) audit.Event {
	return audit.Event{
		Type:         audit.EventType("iam.mfa." + params.AuditAction),
		Category:     "audit",
		ActorType:    "admin",
		UserID:       fmt.Sprintf("%d", params.ActorUserID),
		ResourceType: "iam.mfa",
		ResourceID:   fmt.Sprintf("user:%d", params.TargetUserID),
		Action:       params.AuditAction,
		Result:       outcome.Result,
		Reason:       outcome.Reason,
	}
}
