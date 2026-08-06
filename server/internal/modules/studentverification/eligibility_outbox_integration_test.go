package studentverification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type recordingEligibilityConsumer struct {
	calls    int
	userID   int64
	schoolID int64
	revision int64
	err      error
}

func (c *recordingEligibilityConsumer) ReevaluateStudentEligibility(
	_ context.Context,
	userID int64,
	schoolID int64,
	minimumRevision int64,
) error {
	c.calls++
	c.userID = userID
	c.schoolID = schoolID
	c.revision = minimumRevision
	return c.err
}

func TestEligibilityOutboxPublishesWithOwnerFence(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "eligibility-outbox")
	repository := NewRepository(fixture.DB)
	require.NoError(t, repository.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repository.BumpEligibilityRevisionTx(ctx, tx, userID, testSchoolID, "test_event", now)
	}))

	consumer := &recordingEligibilityConsumer{}
	service, err := NewService(
		repository,
		[]byte("student-verification-outbox-test-key"),
		WithClock(func() time.Time { return now }),
	)
	require.NoError(t, err)
	service.SetEligibilityEventConsumer(consumer)

	service.processEligibilityOutboxBatch(ctx, "11111111-1111-4111-8111-111111111111")
	assert.Equal(t, 1, consumer.calls)
	assert.Equal(t, userID, consumer.userID)
	assert.Equal(t, testSchoolID, consumer.schoolID)
	assert.Equal(t, int64(1), consumer.revision)

	var status string
	var owner *string
	var publishedAt *time.Time
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT status, claim_owner, published_at
		FROM student_verification_event_outbox
		WHERE user_id = $1 AND school_id = $2
	`, userID, testSchoolID).Scan(&status, &owner, &publishedAt))
	assert.Equal(t, "published", status)
	assert.Nil(t, owner)
	assert.NotNil(t, publishedAt)

	service.processEligibilityOutboxBatch(ctx, "22222222-2222-4222-8222-222222222222")
	assert.Equal(t, 1, consumer.calls, "published events must not be delivered again")
}

func TestEligibilityOutboxRetriesWithoutLosingEvent(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := seedVerificationUser(t, fixture, "eligibility-outbox-retry")
	repository := NewRepository(fixture.DB)
	require.NoError(t, repository.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repository.BumpEligibilityRevisionTx(ctx, tx, userID, testSchoolID, "test_retry", now)
	}))

	consumer := &recordingEligibilityConsumer{err: errors.New("temporary consumer failure")}
	service, err := NewService(
		repository,
		[]byte("student-verification-outbox-retry-key"),
		WithClock(func() time.Time { return now }),
	)
	require.NoError(t, err)
	service.SetEligibilityEventConsumer(consumer)
	service.processEligibilityOutboxBatch(ctx, "33333333-3333-4333-8333-333333333333")

	var status string
	var attempts int
	var owner *string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT status, attempts, claim_owner
		FROM student_verification_event_outbox
		WHERE user_id = $1 AND school_id = $2
	`, userID, testSchoolID).Scan(&status, &attempts, &owner))
	assert.Equal(t, "pending", status)
	assert.Equal(t, 1, attempts)
	assert.Nil(t, owner)
}
