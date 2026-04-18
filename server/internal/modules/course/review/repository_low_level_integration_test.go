package review

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewRepository_LowLevelIntegrationPaths(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 10006, "电子信息学院")
	teacherID := seedTeacher(t, fixture, 10006, "李老师", departmentID)
	courseID := seedCourse(t, fixture, 10006, departmentID, "通信原理")
	reviewID := "550e8400-e29b-41d4-a716-446655440901"
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "u-repo-1", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "原始标题", "原始内容")

	var internalUserID int64
	err := fixture.Pool.QueryRow(ctx, `
		INSERT INTO users (external_id, username, email)
		VALUES ('ext-repo-1', 'repo-user', 'repo@example.com')
		RETURNING id
	`).Scan(&internalUserID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE users SET user_hash = $2 WHERE id = $1`, internalUserID, "u-repo-1")
	require.NoError(t, err)

	exists, err := repo.ReviewExists(ctx, reviewID)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ReviewExistsAny(ctx, reviewID)
	require.NoError(t, err)
	assert.True(t, exists)

	reviewed, err := repo.UserHasReviewedCourse(ctx, "u-repo-1", courseID)
	require.NoError(t, err)
	assert.True(t, reviewed)

	fetchedReview, err := repo.GetReviewByID(ctx, reviewID)
	require.NoError(t, err)
	require.NotNil(t, fetchedReview)
	assert.Equal(t, "u-repo-1", fetchedReview.UserHash)

	countAll, err := repo.CountAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, countAll)

	teacherName, err := repo.GetTeacherName(ctx, teacherID)
	require.NoError(t, err)
	assert.Equal(t, "李老师", teacherName)

	resolvedInternalID, err := repo.GetInternalUserIDByExternalID(ctx, "ext-repo-1")
	require.NoError(t, err)
	assert.Equal(t, internalUserID, resolvedInternalID)

	resolvedByHash, err := repo.GetUserIDByUserHash(ctx, "u-repo-1")
	require.NoError(t, err)
	assert.Equal(t, internalUserID, resolvedByHash)

	var createdReviewID = "550e8400-e29b-41d4-a716-446655440902"
	ratingsJSON, err := json.Marshal(ReviewRatings{"teaching": 4, "difficulty": 3})
	require.NoError(t, err)
	err = fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := repo.Create(ctx, tx, CreateParams{
			ID:        createdReviewID,
			CourseID:  courseID,
			TeacherID: &teacherID,
			TermID:    "2025-2",
			Title:     "新建评论",
			Content:   "新建内容",
			Grade:     "A",
			Ratings:   ratingsJSON,
			UserHash:  "u-repo-2",
			Status:    StatusHidden,
		}); err != nil {
			return err
		}

		created, err := repo.CreateVote(ctx, tx, reviewID, "u-voter-low-level", "like")
		if err != nil {
			return err
		}
		if !created {
			return assert.AnError
		}
		voteType, err := repo.GetVoteType(ctx, tx, reviewID, "u-voter-low-level")
		if err != nil {
			return err
		}
		if voteType != "like" {
			return assert.AnError
		}
		if err := repo.UpdateVoteType(ctx, tx, reviewID, "u-voter-low-level", "dislike"); err != nil {
			return err
		}
		if err := repo.IncrementLikeCount(ctx, tx, reviewID); err != nil {
			return err
		}
		if err := repo.IncrementDislikeCount(ctx, tx, reviewID); err != nil {
			return err
		}
		if err := repo.DecrementLikeCount(ctx, tx, reviewID); err != nil {
			return err
		}
		if err := repo.DecrementDislikeCount(ctx, tx, reviewID); err != nil {
			return err
		}
		if err := repo.DeleteVote(ctx, tx, reviewID, "u-voter-low-level"); err != nil {
			return err
		}
		replyID, _, err := repo.CreateReply(ctx, tx, CreateReplyParams{ReviewID: reviewID, UserHash: "u-replier-low-level", Content: "回复内容", Status: StatusPublished})
		if err != nil {
			return err
		}
		owner, reviewRef, status, err := repo.GetReplyOwnerAndReviewIDTx(ctx, tx, replyID)
		if err != nil {
			return err
		}
		if owner != "u-replier-low-level" || reviewRef != reviewID || status != StatusPublished {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, err)

	exists, err = repo.ReviewExistsAny(ctx, createdReviewID)
	require.NoError(t, err)
	assert.True(t, exists)

	reported, err := repo.ReportExists(ctx, reviewID, "u-reporter-low-level")
	require.NoError(t, err)
	assert.False(t, reported)

	require.NoError(t, repo.CreateReport(ctx, CreateReportParams{ReviewID: reviewID, ReporterHash: "u-reporter-low-level", Reason: "spam", Description: "可疑内容"}))
	reported, err = repo.ReportExists(ctx, reviewID, "u-reporter-low-level")
	require.NoError(t, err)
	assert.True(t, reported)

	err = fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := repo.ClearModerationTx(ctx, tx, createdReviewID); err != nil {
			return err
		}
		return repo.AdjustCourseCountsForBatchRestore(ctx, tx, []string{createdReviewID})
	})
	require.NoError(t, err)

	var restoredCourseCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT review_count FROM courses WHERE id = $1`, courseID).Scan(&restoredCourseCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, restoredCourseCount, 1)

	var replyID string
	err = fixture.Pool.QueryRow(ctx, `SELECT id FROM review_replies WHERE review_id = $1 LIMIT 1`, reviewID).Scan(&replyID)
	require.NoError(t, err)
	owner, err := repo.GetReplyOwner(ctx, replyID)
	require.NoError(t, err)
	assert.Equal(t, "u-replier-low-level", owner)
	combinedOwner, combinedReviewID, err := repo.GetReplyOwnerAndReviewID(ctx, replyID)
	require.NoError(t, err)
	assert.Equal(t, "u-replier-low-level", combinedOwner)
	assert.Equal(t, reviewID, combinedReviewID)
	parentReviewID, err := repo.GetReplyReviewID(ctx, replyID)
	require.NoError(t, err)
	assert.Equal(t, reviewID, parentReviewID)

	coreFields, err := repo.getReviewCoreFields(ctx, reviewID, false)
	require.NoError(t, err)
	assert.Equal(t, "u-repo-1", coreFields.userHash)
	assert.Equal(t, courseID, coreFields.courseID)
	require.NotNil(t, coreFields.teacherID)
	assert.Equal(t, teacherID, *coreFields.teacherID)
	assert.Equal(t, StatusPublished, coreFields.status)

	var likeCount, dislikeCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT like_count, dislike_count FROM reviews WHERE id = $1`, reviewID).Scan(&likeCount, &dislikeCount)
	require.NoError(t, err)
	assert.Equal(t, 0, likeCount)
	assert.Equal(t, 0, dislikeCount)
}
