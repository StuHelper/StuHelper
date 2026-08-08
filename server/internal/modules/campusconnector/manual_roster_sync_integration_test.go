package campusconnector

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func TestManualRosterSyncIsPersistentDeduplicatedAndCompletesWithStableFailure(t *testing.T) {
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const (
		schoolCode = "4111010006"
		nodeID     = "00000000-0000-4000-8000-000000000036"
		operation  = "buaa.roster.snapshot"
	)
	actorUserID := seedManualRosterSyncFixture(t, postgres, now, schoolCode, nodeID, operation)

	service, err := NewService(
		NewRepository(postgres.DB),
		NewBroker(4),
		redis.Client,
		nil,
		Config{SnapshotPrivateKey: bytes.Repeat([]byte{1}, 32), SnapshotKeyID: "snapshot-key-1"},
	)
	require.NoError(t, err)
	t.Cleanup(service.Close)
	service.now = func() time.Time { return now }

	request, err := service.RequestManualRosterSync(ctx, ManualRosterSyncInput{
		SchoolCode: schoolCode, ActorUserID: actorUserID,
		Reason: "operator requested a current full roster",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", request.Status)
	assert.Equal(t, now.Add(24*time.Hour), request.DeadlineAt)

	_, err = service.RequestManualRosterSync(ctx, ManualRosterSyncInput{
		SchoolCode: schoolCode, ActorUserID: actorUserID,
		Reason: "duplicate request must not fork the same operation",
	})
	require.ErrorIs(t, err, ErrRequestInFlight)

	command, err := service.ClaimManualRosterSync(ctx, nodeID)
	require.NoError(t, err)
	require.NotNil(t, command)
	assert.Equal(t, request.ID, command.RequestID)
	assert.Equal(t, operation, command.OperationKey)

	requests, err := service.ListManualRosterSyncRequests(ctx, schoolCode, 20)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, "started", requests[0].Status)
	assert.Equal(t, 1, requests[0].ClaimAttempts)
	assert.NotNil(t, requests[0].ClaimedAt)

	require.NoError(t, service.CompleteManualRosterSync(ctx, nodeID, connectorprotocol.RosterSyncResult{
		RequestID: request.ID, ResultCode: "secret_unavailable",
	}))
	requests, err = service.ListManualRosterSyncRequests(ctx, schoolCode, 20)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, "failed", requests[0].Status)
	require.NotNil(t, requests[0].ResultCode)
	assert.Equal(t, "secret_unavailable", *requests[0].ResultCode)
	assert.NotNil(t, requests[0].CompletedAt)

	_, err = service.RequestManualRosterSync(ctx, ManualRosterSyncInput{
		SchoolCode: schoolCode, ActorUserID: actorUserID,
		Reason: "a completed request no longer blocks a new full sync",
	})
	require.NoError(t, err)
}

func seedManualRosterSyncFixture(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	now time.Time,
	schoolCode string,
	nodeID string,
	operationKey string,
) int64 {
	t.Helper()
	ctx := context.Background()
	var schoolID int64
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT id FROM schools WHERE code = $1
	`, schoolCode).Scan(&schoolID))
	var actorUserID int64
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		INSERT INTO users (casdoor_subject, username, email)
		VALUES ('manual-roster-actor', 'manual_roster_actor', 'manual-roster@example.test')
		RETURNING id
	`).Scan(&actorUserID))
	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO campus_connector_nodes (
			id, display_name, status, protocol_version, software_version,
			certificate_fingerprint, signing_key_id, signing_public_key,
			max_concurrency, heartbeat_interval_seconds, last_heartbeat_at,
			last_health_code, certificate_not_after, revision, created_at, updated_at
		) VALUES (
			$1, 'test campus node', 'active', '1', 'test',
			repeat('a', 64), 'signing-key-1', $2,
			4, 30, $3, 'ok', $4, 1, $3, $3
		)
	`, nodeID, bytes.Repeat([]byte{2}, 32), now, now.Add(24*time.Hour))
	require.NoError(t, err)
	tlsName := "oracle.internal.example"
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO campus_connector_school_operations (
			node_id, school_id, operation_key, operation_type,
			adapter_id, adapter_version, upstream_protocol,
			target_host, target_port, target_tls_server_name,
			allowlisted_attributes, timeout_milliseconds, max_concurrency,
			rate_limit_per_minute, enabled, validation_status,
			health_status, health_code, health_checked_at,
			config_revision, validated_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'roster_snapshot_upload',
			'buaa_oracle_roster', '1', 'oracle_tls',
			'oracle.internal.example', 2484, $4,
			ARRAY['XH', 'XM', 'SFZJH', 'RXNJ'], 120000, 1,
			4, true, 'valid', 'healthy', 'ok', $5,
			1, $5, $5, $5
		)
	`, nodeID, schoolID, operationKey, tlsName, now)
	require.NoError(t, err)
	return actorUserID
}
