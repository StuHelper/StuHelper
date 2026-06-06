package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewService_PostReviewPendingAndWriteEarlyReturns(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	svc.filter = seededFilter([]SensitiveWord{{Word: "reviewword", Level: ContentFlagReview}})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "化学学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "黄老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "有机化学")

	pendingAuthorID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-u-pending-post", UserHash: "u-pending-post"})
	pendingAccess := fullReviewWriteAccess(pendingAuthorID)
	posted, err := svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "待复核评论",
		Content:              "reviewword content that should enter moderation",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5, "difficulty": 4},
		UserHash:             "u-pending-post",
		AuthorInternalUserID: pendingAuthorID,
		Access:               pendingAccess,
	})
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, posted.Review.Status)
	require.NotNil(t, posted.Review.ContentFlag)
	assert.Equal(t, ContentFlagReview, *posted.Review.ContentFlag)

	var reviewCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT review_count FROM courses WHERE id = $1`, courseID).Scan(&reviewCount)
	require.NoError(t, err)
	assert.Equal(t, 0, reviewCount)

	var outboxCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM domain_event_outbox WHERE stream = $1 AND dedupe_key = $2`, outbox.StreamIAMOpenFGATupleSync, reviewRelationsSyncKey(posted.Review.ID)).Scan(&outboxCount)
	require.NoError(t, err)
	assert.Equal(t, 1, outboxCount)

	_, err = svc.PostReview(ctx, PostReviewParams{
		CourseID:             999999,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "不存在课程",
		Content:              "正常内容用于不存在课程分支",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5},
		UserHash:             "u-missing-course",
		AuthorInternalUserID: pendingAuthorID,
		Access:               pendingAccess,
	})
	require.ErrorIs(t, err, ErrCourseNotFound)

	missingTeacherID := teacherID + 99999
	_, err = svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &missingTeacherID,
		TermID:               "2025-2",
		Title:                "不存在教师",
		Content:              "正常内容用于不存在教师分支",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5},
		UserHash:             "u-missing-teacher",
		AuthorInternalUserID: pendingAuthorID,
		Access:               pendingAccess,
	})
	require.ErrorIs(t, err, ErrTeacherNotFound)

	publishedID := "550e8400-e29b-41d4-a716-446655441201"
	hiddenID := "550e8400-e29b-41d4-a716-446655441202"
	deletedID := "550e8400-e29b-41d4-a716-446655441203"
	seedReviewWithRatings(t, fixture, publishedID, courseID, teacherID, "u-owner-write", 4.4, StatusPublished, ReviewRatings{"teaching": 5}, "已发布", "内容一")
	seedReviewWithRatings(t, fixture, hiddenID, courseID, teacherID, "u-hidden-write", 4.0, StatusHidden, ReviewRatings{"teaching": 4}, "隐藏评论", "内容二")
	seedReviewWithRatings(t, fixture, deletedID, courseID, teacherID, "u-deleted-write", 3.8, StatusDeleted, ReviewRatings{"teaching": 3}, "已删评论", "内容三")

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: "550e8400-e29b-41d4-a716-446655449997",
		UserHash: "u-owner-write",
		Access:   pendingAccess,
		Title:    strPtr("缺失评论"),
		Content:  strPtr("缺失评论内容"),
		Grade:    strPtr("A"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 4}),
	})
	require.ErrorIs(t, err, ErrReviewNotFound)

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: hiddenID,
		UserHash: "u-hidden-write",
		Access:   pendingAccess,
		Title:    strPtr("更新隐藏评论"),
		Content:  strPtr("更新后的隐藏评论内容"),
		Grade:    strPtr("A"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 4}),
	})
	require.ErrorIs(t, err, ErrReviewNotFound)

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: publishedID,
		UserHash: "u-other-write",
		Access:   pendingAccess,
		Title:    strPtr("越权更新"),
		Content:  strPtr("越权更新内容"),
		Grade:    strPtr("A"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 4}),
	})
	require.ErrorIs(t, err, ErrNotReviewOwner)

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: publishedID,
		UserHash: "u-owner-write",
		Access:   pendingAccess,
		Title:    strPtr("非法成绩"),
		Content:  strPtr("非法成绩内容足够长用于通过其他校验"),
		Grade:    strPtr("S"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 4}),
	})
	require.ErrorIs(t, err, ErrInvalidGrade)

	_, err = svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "重复评课",
		Content:              "正常内容用于重复评课分支",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5},
		UserHash:             "u-owner-write",
		AuthorInternalUserID: pendingAuthorID,
		Access:               pendingAccess,
	})
	require.ErrorIs(t, err, ErrAlreadyReviewed)

	_, err = svc.VoteReview(ctx, VoteReviewParams{ReviewID: "550e8400-e29b-41d4-a716-446655449996", UserHash: "u-vote-write", VoteType: "like"})
	require.ErrorIs(t, err, ErrReviewNotFound)

	err = svc.DeleteReview(ctx, DeleteReviewParams{ReviewID: publishedID, UserHash: "u-other-write", Access: pendingAccess})
	require.ErrorIs(t, err, ErrNotReviewOwner)

	err = svc.DeleteReview(ctx, DeleteReviewParams{ReviewID: deletedID, UserHash: "u-deleted-write", Access: pendingAccess})
	require.ErrorIs(t, err, ErrReviewNotFound)
}
