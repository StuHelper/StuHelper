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
