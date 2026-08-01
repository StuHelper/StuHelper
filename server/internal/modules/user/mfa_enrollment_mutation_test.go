package user

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMFAResetSuperAdminAllowsSingleAdministratorSelfReset(t *testing.T) {
	repo := &mfaMutationFakeRepo{}
	manager := newTestMFARecoveryManager(t, repo)

	err := manager.ResetEnrollment(context.Background(), MFAEnrollmentAdminAction{
		ActorUserID:    2,
		TargetUserID:   2,
		TargetRoleKind: MFATargetRoleSuperAdmin,
	})

	require.NoError(t, err)
	require.Len(t, repo.stateChanges, 1)
	assert.Equal(t, int64(2), repo.deletedUserID)
	assert.True(t, repo.stateChanges[0].ResetRequired)
}

func TestMFADisableSuperAdminRejectsSelfDisable(t *testing.T) {
	repo := &mfaMutationFakeRepo{}
	manager := newTestMFARecoveryManager(t, repo)

	err := manager.DisableEnrollment(context.Background(), MFAEnrollmentAdminAction{
		ActorUserID:    2,
		TargetUserID:   2,
		TargetRoleKind: MFATargetRoleSuperAdmin,
	})

	require.ErrorIs(t, err, ErrMFASelfDisableForbidden)
	assert.Zero(t, repo.txCalls)
}

func TestMFAMutationRequiresExplicitTargetRoleKind(t *testing.T) {
	repo := &mfaMutationFakeRepo{}
	manager := newTestMFARecoveryManager(t, repo)

	err := manager.DisableEnrollment(context.Background(), MFAEnrollmentAdminAction{
		ActorUserID:  1,
		TargetUserID: 2,
	})

	require.ErrorIs(t, err, ErrMFATargetRoleInvalid)
	assert.Zero(t, repo.txCalls)
}

func TestMFAEnrollmentMutationAuditIncludesTargetRole(t *testing.T) {
	event := mfaEnrollmentMutationAuditEvent(mfaEnrollmentMutation{
		ActorUserID:    1,
		TargetUserID:   2,
		TargetRoleKind: MFATargetRoleSuperAdmin,
		AuditAction:    "reset",
	}, mfaEnrollmentAuditOutcome{Result: "success"})

	assert.Equal(t, string(MFATargetRoleSuperAdmin), event.Details["target_role_kind"])
}

type mfaMutationFakeRepo struct {
	mfaRecoveryFakeRepo
	txCalls       int
	stateChanges  []MFAEnrollmentStateChange
	deletedUserID int64
}

func (m *mfaMutationFakeRepo) WithTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	m.txCalls++
	return fn(ctx, nil)
}

func (m *mfaMutationFakeRepo) UpdateMFAEnrollmentStateTx(
	_ context.Context,
	_ pgx.Tx,
	params MFAEnrollmentStateChange,
) error {
	m.stateChanges = append(m.stateChanges, params)
	return nil
}

func (m *mfaMutationFakeRepo) DeleteMFARecoveryCodesTx(_ context.Context, _ pgx.Tx, userID int64) error {
	m.deletedUserID = userID
	return nil
}
