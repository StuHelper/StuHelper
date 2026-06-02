package review

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestRefreshReviewTargetsTx_DedupesAndSkipsInvalidIDs(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "生物学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "邓老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "分子生物学")
	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655441401", courseID, teacherID, "u-refresh-stats", 4.6, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "统计评论", "统计评论内容")

	err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return svc.refreshReviewTargetsTx(ctx, tx, []int64{0, courseID, courseID}, []int64{0, teacherID, teacherID})
	})
	require.NoError(t, err)

	var courseStatsCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM course_rating_stats WHERE course_id = $1`, courseID).Scan(&courseStatsCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, courseStatsCount, 1)

	var teacherStatsCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM teacher_rating_stats WHERE teacher_id = $1`, teacherID).Scan(&teacherStatsCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, teacherStatsCount, 1)
}
