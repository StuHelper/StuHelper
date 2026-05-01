package serviceaccount

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestVerifierBootstrapAndVerifyLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	verifier := newTestVerifier(t, fixture)
	ctx := context.Background()

	result, err := verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:     KoishiRuntimeCredentialName,
		RawToken: "koishi-token",
		Audience: []string{AudienceBotAPI},
		Scopes:   KoishiRuntimeScopes(),
	})
	require.NoError(t, err)
	assert.Equal(t, BootstrapCreated, result.Status)

	err = verifier.Verify(ctx, "koishi-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.NoError(t, err)
	assertCredentialLastUsed(t, fixture, result.ID)

	err = verifier.Verify(ctx, "koishi-token", "/api/v1/admin/system-configs", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialForbidden)

	err = verifier.Verify(ctx, "koishi-token", "/api/v1/bot/qq-binding/consume", "admin.config.write")
	require.ErrorIs(t, err, ErrCredentialForbidden)

	err = verifier.Verify(ctx, "wrong-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialInvalid)

	require.NoError(t, verifier.Revoke(ctx, KoishiRuntimeCredentialName))
	err = verifier.Verify(ctx, "koishi-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialInvalid)
}

func TestVerifierVerifyPersistsCallAuditEvents(t *testing.T) {
	fixture := postgresfixture.Start(t)
	verifier := newTestVerifier(t, fixture)
	ctx := context.Background()

	_, err := verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:     KoishiRuntimeCredentialName,
		RawToken: "koishi-token",
		Audience: []string{AudienceBotAPI},
		Scopes:   KoishiRuntimeScopes(),
	})
	require.NoError(t, err)

	audit.ConfigureRepository(audit.NewRepository(fixture.DB))
	defer audit.ConfigureRepository(nil)

	err = verifier.Verify(ctx, "koishi-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.NoError(t, err)
	err = verifier.Verify(ctx, "koishi-token", "/api/v1/admin/system-configs", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialForbidden)
	err = verifier.Verify(ctx, "wrong-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialInvalid)

	events := loadServiceAccountCallEvents(t, fixture)
	require.Len(t, events, 3)
	assert.Equal(t, "success", events[0].Result)
	assert.Equal(t, KoishiRuntimeCredentialName, events[0].ResourceID)
	assert.Contains(t, events[0].Details, ScopeBotQQBindingConsume)
	assert.NotContains(t, events[0].Details, "koishi-token")

	assert.Equal(t, "failure", events[1].Result)
	assert.Contains(t, events[1].Details, "/api/v1/admin/system-configs")

	assert.Equal(t, "failure", events[2].Result)
	assert.Equal(t, "unknown", events[2].ResourceID)
	assert.Contains(t, events[2].Details, "token_hash_prefix")
	assert.NotContains(t, events[2].Details, "wrong-token")
}

func TestVerifierBootstrapRotationReactivatesRevokedCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	verifier := newTestVerifier(t, fixture)
	ctx := context.Background()

	_, err := verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:     KoishiRuntimeCredentialName,
		RawToken: "old-token",
		Audience: []string{AudienceBotAPI},
		Scopes:   KoishiRuntimeScopes(),
	})
	require.NoError(t, err)
	require.NoError(t, verifier.Revoke(ctx, KoishiRuntimeCredentialName))

	sameResult, err := verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:     KoishiRuntimeCredentialName,
		RawToken: "old-token",
		Audience: []string{AudienceBotAPI},
		Scopes:   KoishiRuntimeScopes(),
	})
	require.NoError(t, err)
	assert.Equal(t, BootstrapUnchanged, sameResult.Status)
	err = verifier.Verify(ctx, "old-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialInvalid)

	rotatedResult, err := verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:     KoishiRuntimeCredentialName,
		RawToken: "new-token",
		Audience: []string{AudienceBotAPI},
		Scopes:   KoishiRuntimeScopes(),
	})
	require.NoError(t, err)
	assert.Equal(t, BootstrapRotated, rotatedResult.Status)
	require.NoError(t, verifier.Verify(ctx, "new-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume))
}

func TestVerifierRejectsExpiredAndIncompleteCredentials(t *testing.T) {
	fixture := postgresfixture.Start(t)
	verifier := newTestVerifier(t, fixture)
	ctx := context.Background()
	expiredAt := time.Now().Add(-time.Minute)

	_, err := verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:      "expired-koishi",
		RawToken:  "expired-token",
		Audience:  []string{AudienceBotAPI},
		Scopes:    KoishiRuntimeScopes(),
		ExpiresAt: &expiredAt,
	})
	require.NoError(t, err)

	err = verifier.Verify(ctx, "expired-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)
	require.ErrorIs(t, err, ErrCredentialInvalid)

	_, err = verifier.EnsureBootstrapCredential(ctx, BootstrapCredential{
		Name:     "missing-scopes",
		RawToken: "token",
		Audience: []string{AudienceBotAPI},
	})
	require.True(t, errors.Is(err, ErrCredentialNotConfigured))
}

func newTestVerifier(t *testing.T, fixture *postgresfixture.Fixture) *Verifier {
	t.Helper()
	verifier, err := NewVerifier(fixture.DB, []byte("test-service-account-hmac-key-32!"))
	require.NoError(t, err)
	return verifier
}

func assertCredentialLastUsed(t *testing.T, fixture *postgresfixture.Fixture, id int64) {
	t.Helper()
	var lastUsedAt *time.Time
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT last_used_at
		FROM bot_service_credentials
		WHERE id = $1
	`, id).Scan(&lastUsedAt)
	require.NoError(t, err)
	require.NotNil(t, lastUsedAt)
}

type callAuditEventRow struct {
	Result     string
	ResourceID string
	Details    string
}

func loadServiceAccountCallEvents(t *testing.T, fixture *postgresfixture.Fixture) []callAuditEventRow {
	t.Helper()
	rows, err := fixture.Pool.Query(context.Background(), `
		SELECT result, resource_id, details::text
		FROM audit_events
		WHERE event_type = 'iam.service_account.call'
		ORDER BY created_at ASC, id ASC
	`)
	require.NoError(t, err)
	defer rows.Close()

	var events []callAuditEventRow
	for rows.Next() {
		var event callAuditEventRow
		require.NoError(t, rows.Scan(&event.Result, &event.ResourceID, &event.Details))
		events = append(events, event)
	}
	require.NoError(t, rows.Err())
	return events
}
