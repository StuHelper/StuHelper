package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewService_IntegrationErrorBranches(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "软件学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "李老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "编译原理")
	reviewID := "550e8400-e29b-41d4-a716-446655440201"
	hiddenReviewID := "550e8400-e29b-41d4-a716-446655440202"
	deletedReviewID := "550e8400-e29b-41d4-a716-446655440203"
	missingReviewID := "550e8400-e29b-41d4-a716-446655440299"

	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "u-review-owner", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "主评课", "主评课内容")
	seedReviewWithRatings(t, fixture, hiddenReviewID, courseID, teacherID, "u-hidden-owner", 4.0, StatusHidden, ReviewRatings{"teaching": 4}, "隐藏评课", "隐藏评课内容")
	seedReviewWithRatings(t, fixture, deletedReviewID, courseID, teacherID, "u-deleted-owner", 3.0, StatusDeleted, ReviewRatings{"teaching": 3}, "已删评课", "已删评课内容")

	err := svc.AddFavorite(ctx, AddFavoriteParams{UserHash: "u-fav", CourseID: 999999})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.SaveDraft(ctx, SaveDraftParams{
		UserHash: "u-draft",
		CourseID: &courseID,
		Title:    "草稿",
		Content:  `<script>alert(1)</script>`,
		Ratings:  ReviewRatings{"teaching": 5},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, err = svc.CreateReply(ctx, CreateReplyParams{
		ReviewID: missingReviewID,
		UserHash: "u-reply",
		Content:  "回复内容",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReviewNotFound)

	_, err = svc.CreateReply(ctx, CreateReplyParams{
		ReviewID: reviewID,
		UserHash: "u-reply",
		Content:  `<script>alert(1)</script>`,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	createdReply, err := svc.CreateReply(ctx, CreateReplyParams{
		ReviewID: reviewID,
		UserHash: "u-reply-owner",
		Content:  "这是一个正常回复",
	})
	require.NoError(t, err)

	err = svc.DeleteReply(ctx, DeleteReplyParams{
		ReplyID:  createdReply.Reply.ID,
		UserHash: "u-other-user",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotReplyOwner)

	err = svc.DeleteReply(ctx, DeleteReplyParams{
		ReplyID:  "550e8400-e29b-41d4-a716-446655440298",
		UserHash: "u-other-user",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReplyNotFound)

	reporterID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-report", UserHash: "u-report"})
	firstReportID, err := svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               reviewID,
		UserHash:               "u-report",
		ReporterInternalUserID: reporterID,
		Reason:                 "spam",
		Description:            "需要处理",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, firstReportID)

	_, err = svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               reviewID,
		UserHash:               "u-report",
		ReporterInternalUserID: reporterID,
		Reason:                 "spam",
		Description:            "重复举报",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAlreadyReported)

	err = svc.ProcessReport(ctx, ProcessReportParams{
		ReportID:   "550e8400-e29b-41d4-a716-446655440297",
		Action:     "reject",
		Note:       "missing",
		ResolvedBy: "admin",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReportNotFound)

	_, err = svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{
		ReviewID: hiddenReviewID,
		Action:   "hide",
		Reason:   "重复隐藏",
		AdminID:  "admin",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)

	err = svc.AdminEditReview(ctx, AdminEditReviewParams{
		ReviewID: deletedReviewID,
		Title:    "不应修改",
		Content:  "已删除评论不应再被编辑",
		Reason:   "状态错误",
		AdminID:  "admin",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)
	var deletedTitle string
	err = fixture.Pool.QueryRow(ctx, `SELECT title FROM reviews WHERE id = $1`, deletedReviewID).Scan(&deletedTitle)
	require.NoError(t, err)
	assert.Equal(t, "已删评课", deletedTitle)

	err = svc.ClearContentFlag(ctx, missingReviewID, "admin")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReviewNotFound)
}
