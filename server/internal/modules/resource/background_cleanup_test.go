package resource

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestProcessCleanupJobNormalizesPayloadBeforeDelete(t *testing.T) {
	store := &fakeObjectStore{}
	svc := &Service{storage: store}

	err := svc.processCleanupJob(context.Background(), cleanupJob{
		JobType: resourceCleanupJobType,
		Payload: []byte(`{"mountID":42,"objectKey":" resources/1/file.txt "}`),
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"resources/1/file.txt"}, store.deletedKeys)
}

func TestProcessCleanupJobRejectsInvalidPayloadBeforeDelete(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{name: "missing mount", payload: `{"objectKey":"resources/1/file.txt"}`},
		{name: "blank object key", payload: `{"mountID":42,"objectKey":" "}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeObjectStore{}
			svc := &Service{storage: store}

			err := svc.processCleanupJob(context.Background(), cleanupJob{
				JobType: resourceCleanupJobType,
				Payload: []byte(tc.payload),
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid resource cleanup payload")
			assert.Empty(t, store.deletedKeys)
		})
	}
}

func TestProcessCleanupBatchDeadLettersInvalidPayloadAfterMaxAttempts(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	repo := NewRepository(fixture.DB)
	store := &fakeObjectStore{}
	svc := &Service{repo: repo, storage: store}

	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO domain_event_outbox (
			stream, job_type, dedupe_key, payload, status, attempt_count, available_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4::jsonb, 'pending', $5, NOW(), NOW(), NOW())
	`, resourceCleanupOutboxStream, resourceCleanupJobType, "invalid-cleanup-payload", `{"mountID":0,"objectKey":" "}`, resourceCleanupMaxAttempts-1)
	require.NoError(t, err)

	err = svc.processCleanupBatch(ctx)

	require.NoError(t, err)
	assert.Empty(t, store.deletedKeys)

	var (
		status       string
		attemptCount int
		lastError    string
		availableAt  time.Time
	)
	err = fixture.Pool.QueryRow(ctx, `
		SELECT status, attempt_count, last_error, available_at
		FROM domain_event_outbox
		WHERE stream = $1 AND dedupe_key = $2
	`, resourceCleanupOutboxStream, "invalid-cleanup-payload").Scan(&status, &attemptCount, &lastError, &availableAt)
	require.NoError(t, err)
	assert.Equal(t, outbox.StatusDeadLetter, status)
	assert.Equal(t, resourceCleanupMaxAttempts, attemptCount)
	assert.Contains(t, lastError, "invalid resource cleanup payload")
	assert.False(t, availableAt.IsZero())
}
