package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
)

var (
	ErrMFATargetRoleInvalid    = errors.New("mfa target role kind is invalid")
	ErrMFASelfDisableForbidden = errors.New("super admin cannot disable own mfa")
)

type MFATargetRoleKind string

const (
	MFATargetRoleStandard   MFATargetRoleKind = "standard"
	MFATargetRoleSuperAdmin MFATargetRoleKind = "super_admin"
)

type MFAEnrollmentAdminAction struct {
	ActorUserID    int64
	TargetUserID   int64
	TargetRoleKind MFATargetRoleKind
}

func (m *MFARecoveryManager) DisableEnrollment(ctx context.Context, params MFAEnrollmentAdminAction) error {
	mutation := mfaEnrollmentMutationFromAction(mfaEnrollmentMutationBuild{
		Params:      params,
		Action:      MFAEnrollmentChangeDisable,
		AuditAction: "disable",
	})
	if err := validateMFAEnrollmentAdminAction(params); err != nil {
		return m.rejectEnrollmentMutation(ctx, mutation, err)
	}
	if params.TargetRoleKind == MFATargetRoleSuperAdmin && params.ActorUserID == params.TargetUserID {
		return m.rejectEnrollmentMutation(ctx, mutation, ErrMFASelfDisableForbidden)
	}
	return m.mutateEnrollment(ctx, mutation)
}

func (m *MFARecoveryManager) ResetEnrollment(ctx context.Context, params MFAEnrollmentAdminAction) error {
	mutation := mfaEnrollmentMutationFromAction(mfaEnrollmentMutationBuild{
		Params:        params,
		Action:        MFAEnrollmentChangeReset,
		AuditAction:   "reset",
		ResetRequired: true,
	})
	if err := validateMFAEnrollmentAdminAction(params); err != nil {
		return m.rejectEnrollmentMutation(ctx, mutation, err)
	}
	return m.mutateEnrollment(ctx, mutation)
}

type mfaEnrollmentMutationBuild struct {
	Params        MFAEnrollmentAdminAction
	Action        MFAEnrollmentChangeAction
	AuditAction   string
	ResetRequired bool
}

func mfaEnrollmentMutationFromAction(input mfaEnrollmentMutationBuild) mfaEnrollmentMutation {
	params := input.Params
	return mfaEnrollmentMutation{
		ActorUserID:    params.ActorUserID,
		TargetUserID:   params.TargetUserID,
		TargetRoleKind: params.TargetRoleKind,
		Action:         input.Action,
		ResetRequired:  input.ResetRequired,
		AuditAction:    input.AuditAction,
	}
}

func validateMFAEnrollmentAdminAction(params MFAEnrollmentAdminAction) error {
	if params.ActorUserID <= 0 || params.TargetUserID <= 0 {
		return ErrMFAUserInvalid
	}
	if !validMFATargetRoleKind(params.TargetRoleKind) {
		return ErrMFATargetRoleInvalid
	}
	return nil
}

func validMFATargetRoleKind(kind MFATargetRoleKind) bool {
	return kind == MFATargetRoleStandard || kind == MFATargetRoleSuperAdmin
}

type mfaEnrollmentMutation struct {
	ActorUserID    int64
	TargetUserID   int64
	TargetRoleKind MFATargetRoleKind
	Action         MFAEnrollmentChangeAction
	ResetRequired  bool
	AuditAction    string
}

func (m *MFARecoveryManager) mutateEnrollment(ctx context.Context, params mfaEnrollmentMutation) error {
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

func (m *MFARecoveryManager) rejectEnrollmentMutation(
	ctx context.Context,
	params mfaEnrollmentMutation,
	err error,
) error {
	m.auditEnrollmentMutation(ctx, params, mfaEnrollmentAuditOutcome{Result: "failure", Reason: err.Error()})
	return err
}

func (m *MFARecoveryManager) auditEnrollmentMutation(
	ctx context.Context,
	params mfaEnrollmentMutation,
	outcome mfaEnrollmentAuditOutcome,
) {
	audit.LogContext(ctx, mfaEnrollmentMutationAuditEvent(params, outcome))
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
		Details:      mfaEnrollmentMutationDetails(params),
	}
}

func mfaEnrollmentMutationDetails(params mfaEnrollmentMutation) map[string]any {
	return map[string]any{"target_role_kind": string(params.TargetRoleKind)}
}
