package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestTeacherPublicStatsProjectionTriggerCoalescesAndFencesRefreshes(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "投影学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "投影老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "投影课程")
	reviewID := "550e8400-e29b-41d4-a716-446655440991"
	seedReviewWithRatings(
		t,
		fixture,
		reviewID,
		courseID,
		teacherID,
		"projection-user",
		4.0,
		StatusPublished,
		ReviewRatings{"teaching": 4},
		"投影测试",
		"投影测试内容",
	)

	first, err := repo.ClaimTeacherPublicStatsRefreshJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, first, 1, "all source writes must coalesce into one projection job")

	_, err = fixture.Pool.Exec(
		ctx,
		`UPDATE reviews SET avg_rating = 5, updated_at = NOW() WHERE id = $1`,
		reviewID,
	)
	require.NoError(t, err)

	concurrent, err := repo.ClaimTeacherPublicStatsRefreshJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, concurrent, "a newer revision must preserve the active projection lease")

	require.NoError(
		t,
		repo.MarkTeacherPublicStatsRefreshJobDone(ctx, first[0].ID, first[0].LockedAt),
	)

	latest, err := repo.ClaimTeacherPublicStatsRefreshJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, latest, 1, "a write during refresh must requeue the superseding revision")
	assert.Equal(t, first[0].ID, latest[0].ID)
	require.NoError(
		t,
		repo.MarkTeacherPublicStatsRefreshJobDone(ctx, latest[0].ID, latest[0].LockedAt),
	)

	remaining, err := repo.ClaimTeacherPublicStatsRefreshJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestTeacherPublicStatsProjectionSingletonWaitIsObservableAndCoalesced(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "锁等待学院")
	firstTeacherID := seedTeacher(t, fixture, 4111010006, "锁等待教师一", departmentID)
	secondTeacherID := seedTeacher(t, fixture, 4111010006, "锁等待教师二", departmentID)

	blocker, err := fixture.Pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocker.Rollback(ctx) })
	_, err = blocker.Exec(ctx, `UPDATE teachers SET name = name WHERE id = $1`, firstTeacherID)
	require.NoError(t, err)

	const waiterApplicationName = "teacher-stats-lock-wait-test"
	waiterStarted := make(chan struct{})
	waiterDone := make(chan error, 1)
	go func() {
		waiter, beginErr := fixture.Pool.Begin(ctx)
		if beginErr != nil {
			waiterDone <- beginErr
			return
		}
		defer func() { _ = waiter.Rollback(ctx) }()
		if _, setErr := waiter.Exec(ctx, `SELECT set_config('application_name', $1, true)`, waiterApplicationName); setErr != nil {
			waiterDone <- setErr
			return
		}
		close(waiterStarted)
		if _, updateErr := waiter.Exec(ctx, `UPDATE teachers SET name = name WHERE id = $1`, secondTeacherID); updateErr != nil {
			waiterDone <- updateErr
			return
		}
		waiterDone <- waiter.Commit(ctx)
	}()
	<-waiterStarted

	require.Eventually(t, func() bool {
		var waitEventType string
		err := fixture.Pool.QueryRow(ctx, `
			SELECT COALESCE(wait_event_type, '')
			FROM pg_stat_activity
			WHERE application_name = $1
		`, waiterApplicationName).Scan(&waitEventType)
		return err == nil && waitEventType == "Lock"
	}, 3*time.Second, 25*time.Millisecond, "the singleton dedupe row wait must remain visible in pg_stat_activity")

	require.NoError(t, blocker.Commit(ctx))
	require.NoError(t, <-waiterDone)

	var jobCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM domain_event_outbox
		WHERE stream = 'review_projection'
		  AND dedupe_key = 'teacher_public_stats'
	`).Scan(&jobCount))
	assert.Equal(t, 1, jobCount, "concurrent source writes must still coalesce into one durable refresh job")
}
