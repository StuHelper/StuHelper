package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestMFARecoveryCodesIssueAndConsumeOnce(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-mfa-recovery-hmac-material-32!"))
	manager := newTestMFARecoveryManager(t, repo)
	userID := seedMFAUser(t, fixture)

	bundle, err := manager.IssueRecoveryCodes(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, bundle.Codes, mfaRecoveryCodeCount)
	assertMFARecoveryCodesUnique(t, bundle.Codes)
	checks := mfaRecoveryDBChecks{fixture: fixture, userID: userID}
	checks.assertStoredCodes(t, bundle.Codes)
	checks.assertIssuedAt(t, bundle.IssuedAt)

	require.NoError(t, manager.ConsumeRecoveryCode(context.Background(), userID, bundle.Codes[0]))
	err = manager.ConsumeRecoveryCode(context.Background(), userID, bundle.Codes[0])
	require.ErrorIs(t, err, ErrMFARecoveryCodeInvalid)
	checks.assertUsedCount(t, 1)
}

func TestMFARecoveryCodesPersistAuditWithoutPlaintextCode(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-mfa-recovery-hmac-material-32!"))
	manager := newTestMFARecoveryManager(t, repo)
	userID := seedMFAUser(t, fixture)

	audit.ConfigureRepository(audit.NewRepository(fixture.DB))
	defer audit.ConfigureRepository(nil)

	bundle, err := manager.IssueRecoveryCodes(context.Background(), userID)
	require.NoError(t, err)
	require.NoError(t, manager.ConsumeRecoveryCode(context.Background(), userID, bundle.Codes[0]))
	err = manager.ConsumeRecoveryCode(context.Background(), userID, "invalid-code")
	require.ErrorIs(t, err, ErrMFARecoveryCodeInvalid)

	events := loadMFARecoveryAuditEvents(t, fixture)
	require.Len(t, events, 3)
	assert.Equal(t, "iam.mfa.recovery_codes_issue", events[0].EventType)
	assert.Equal(t, "success", events[0].Result)
	assert.Equal(t, "iam.mfa.recovery_code_use", events[1].EventType)
	assert.Equal(t, "success", events[1].Result)
	assert.Equal(t, "iam.mfa.recovery_code_use", events[2].EventType)
	assert.Equal(t, "failure", events[2].Result)
	for _, event := range events {
		assert.NotContains(t, event.Details, bundle.Codes[0])
	}
}

func newTestMFARecoveryManager(t *testing.T, repo MFARecoveryRepository) *MFARecoveryManager {
	t.Helper()
	manager, err := NewMFARecoveryManager(repo, []byte("test-mfa-recovery-hmac-material-32!"))
	require.NoError(t, err)
	manager.now = func() time.Time { return time.Date(2026, 5, 2, 3, 4, 5, 0, time.UTC) }
	return manager
}

func assertMFARecoveryCodesUnique(t *testing.T, codes []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		require.NotContains(t, seen, code)
		seen[code] = struct{}{}
	}
}

type mfaRecoveryDBChecks struct {
	fixture *postgresfixture.Fixture
	userID  int64
}

func (c mfaRecoveryDBChecks) assertStoredCodes(t *testing.T, codes []string) {
	t.Helper()
	var storedCount int64
	var rawMatches int64
	err := c.fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE code_hash = ANY($2))
		FROM user_mfa_recovery_codes
		WHERE user_id = $1
	`, c.userID, codes).Scan(&storedCount, &rawMatches)
	require.NoError(t, err)
	assert.Equal(t, int64(len(codes)), storedCount)
	assert.Zero(t, rawMatches)
}

func (c mfaRecoveryDBChecks) assertIssuedAt(t *testing.T, issuedAt time.Time) {
	t.Helper()
	var got time.Time
	err := c.fixture.Pool.QueryRow(context.Background(), `
		SELECT recovery_codes_issued_at
		FROM user_mfa_enrollment
		WHERE user_id = $1
	`, c.userID).Scan(&got)
	require.NoError(t, err)
	assert.True(t, got.Equal(issuedAt))
}

func (c mfaRecoveryDBChecks) assertUsedCount(t *testing.T, expected int) {
	t.Helper()
	var usedCount int64
	err := c.fixture.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM user_mfa_recovery_codes
		WHERE user_id = $1 AND used_at IS NOT NULL
	`, c.userID).Scan(&usedCount)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), usedCount)
}

type mfaRecoveryAuditRow struct {
	EventType string
	Result    string
	Details   string
}

func loadMFARecoveryAuditEvents(t *testing.T, fixture *postgresfixture.Fixture) []mfaRecoveryAuditRow {
	t.Helper()
	rows, err := fixture.Pool.Query(context.Background(), `
		SELECT event_type, result, COALESCE(details::text, '')
		FROM audit_events
		WHERE event_type LIKE 'iam.mfa.recovery%'
		ORDER BY created_at ASC, id ASC
	`)
	require.NoError(t, err)
	defer rows.Close()

	var events []mfaRecoveryAuditRow
	for rows.Next() {
		var event mfaRecoveryAuditRow
		require.NoError(t, rows.Scan(&event.EventType, &event.Result, &event.Details))
		events = append(events, event)
	}
	require.NoError(t, rows.Err())
	return events
}
