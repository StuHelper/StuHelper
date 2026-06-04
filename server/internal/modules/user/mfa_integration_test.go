package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestMFAEnrollmentRepositoryLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()
	userID := seedMFAUser(t, fixture)
	issuedAt := time.Now().Add(-time.Minute).UTC()

	err := repo.UpsertMFAEnrollment(ctx, MFAEnrollmentUpsert{
		UserID:                userID,
		Active:                true,
		Methods:               []string{" WebAuthn ", "sms", "totp", "totp"},
		RecoveryCodesIssuedAt: &issuedAt,
	})
	require.NoError(t, err)

	enrollment, err := repo.GetMFAEnrollment(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	assert.True(t, enrollment.Active)
	assert.Equal(t, []string{MFAMethodSMS, MFAMethodTOTP, MFAMethodWebAuthn}, enrollment.Methods)
	assert.NotNil(t, enrollment.LastEnrolledAt)
	assert.Nil(t, enrollment.LastDisabledAt)

	err = repo.UpsertMFAEnrollment(ctx, MFAEnrollmentUpsert{UserID: userID})
	require.NoError(t, err)

	enrollment, err = repo.GetMFAEnrollment(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	assert.False(t, enrollment.Active)
	assert.Empty(t, enrollment.Methods)
	assert.NotNil(t, enrollment.LastDisabledAt)
}

func TestMFAEnrollmentRepositoryRejectsInvalidState(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()
	userID := seedMFAUser(t, fixture)

	err := repo.UpsertMFAEnrollment(ctx, MFAEnrollmentUpsert{UserID: userID, Active: true})
	require.ErrorIs(t, err, ErrMFAEnrollmentMethodRequired)

	err = repo.UpsertMFAEnrollment(ctx, MFAEnrollmentUpsert{
		UserID:  userID,
		Active:  true,
		Methods: []string{"email"},
	})
	require.ErrorIs(t, err, ErrInvalidMFAMethod)
}

func seedMFAUser(t *testing.T, fixture *postgresfixture.Fixture) int64 {
	t.Helper()
	var userID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ('mfa-user', 'mfa-user', 'mfa-user@example.test')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)
	return userID
}
