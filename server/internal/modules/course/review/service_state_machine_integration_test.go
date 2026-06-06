package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

type recordingNotificationSender struct {
	ch chan ReviewNotification
}

func (s *recordingNotificationSender) SendReviewNotification(_ context.Context, params ReviewNotification) error {
	if s.ch != nil {
		s.ch <- params
	}
	return nil
}

func waitNotification(t *testing.T, ch <-chan ReviewNotification, wantType string) ReviewNotification {
	t.Helper()
	select {
	case got := <-ch:
		require.Equal(t, wantType, got.Type)
		return got
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s notification", wantType)
		return ReviewNotification{}
	}
}

func TestReviewService_StateTransitionsAndNotifications(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	sender := &recordingNotificationSender{ch: make(chan ReviewNotification, 8)}
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	svc.filter = seededFilter([]SensitiveWord{
		{Word: "reviewword", Level: ContentFlagReview},
		{Word: "warnword", Level: ContentFlagWarn},
	})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "数学学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "周老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "高等数学")
	reviewID := "550e8400-e29b-41d4-a716-446655440951"
	warnFlaggedID := "550e8400-e29b-41d4-a716-446655440952"
	hiddenReviewID := "550e8400-e29b-41d4-a716-446655440953"

	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "u-owner-state", 4.7, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "原始标题", "原始内容")
	seedReviewWithRatings(t, fixture, warnFlaggedID, courseID, teacherID, "u-owner-flagged", 4.1, StatusPublished, ReviewRatings{"teaching": 4}, "待清除标记", "待清除标记内容")
	seedReviewWithRatings(t, fixture, hiddenReviewID, courseID, teacherID, "u-owner-hidden", 3.8, StatusHidden, ReviewRatings{"teaching": 4}, "隐藏评论", "隐藏内容")

	var ownerUserID int64
	err := fixture.Pool.QueryRow(ctx, `
		INSERT INTO users (casdoor_subject, username, email, user_hash)
		VALUES ('ext-owner-state', 'owner-state', 'owner-state@example.com', 'u-owner-state')
		RETURNING id
	`).Scan(&ownerUserID)
	require.NoError(t, err)
	ownerAccess := fullReviewWriteAccess(ownerUserID)
	_, err = fixture.Pool.Exec(ctx, `
		INSERT INTO users (casdoor_subject, username, email, user_hash)
		VALUES ('ext-owner-flagged', 'owner-flagged', 'owner-flagged@example.com', 'u-owner-flagged')
	`)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE reviews SET content_flag = $2 WHERE id = $1`, warnFlaggedID, ContentFlagWarn)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
	require.NoError(t, err)

	// Vote 状态机：新增 like -> 取消 -> dislike -> 切换 like -> 切换 dislike
	gotCourseID, err := svc.VoteReview(ctx, VoteReviewParams{ReviewID: reviewID, UserHash: "u-voter-state", VoteType: "like"})
	require.NoError(t, err)
	assert.Equal(t, courseID, gotCourseID)
	var likeCount, dislikeCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT like_count, dislike_count FROM reviews WHERE id = $1`, reviewID).Scan(&likeCount, &dislikeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, likeCount)
	assert.Equal(t, 0, dislikeCount)

	_, err = svc.VoteReview(ctx, VoteReviewParams{ReviewID: reviewID, UserHash: "u-voter-state", VoteType: "like"})
	require.NoError(t, err)
	err = fixture.Pool.QueryRow(ctx, `SELECT like_count, dislike_count FROM reviews WHERE id = $1`, reviewID).Scan(&likeCount, &dislikeCount)
	require.NoError(t, err)
	assert.Equal(t, 0, likeCount)
	assert.Equal(t, 0, dislikeCount)

	_, err = svc.VoteReview(ctx, VoteReviewParams{ReviewID: reviewID, UserHash: "u-voter-state", VoteType: "dislike"})
	require.NoError(t, err)
	_, err = svc.VoteReview(ctx, VoteReviewParams{ReviewID: reviewID, UserHash: "u-voter-state", VoteType: "like"})
	require.NoError(t, err)
	err = fixture.Pool.QueryRow(ctx, `SELECT like_count, dislike_count FROM reviews WHERE id = $1`, reviewID).Scan(&likeCount, &dislikeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, likeCount)
	assert.Equal(t, 0, dislikeCount)

	_, err = svc.VoteReview(ctx, VoteReviewParams{ReviewID: reviewID, UserHash: "u-voter-state", VoteType: "dislike"})
	require.NoError(t, err)
	err = fixture.Pool.QueryRow(ctx, `SELECT like_count, dislike_count FROM reviews WHERE id = $1`, reviewID).Scan(&likeCount, &dislikeCount)
	require.NoError(t, err)
	assert.Equal(t, 0, likeCount)
	assert.Equal(t, 1, dislikeCount)

	// Reply 链路。
	replyResult, err := svc.CreateReply(ctx, CreateReplyParams{
		ReviewID: reviewID,
		UserHash: "u-replier-state",
		Content:  "这是一条回复",
	})
	require.NoError(t, err)

	_, err = svc.CreateReply(ctx, CreateReplyParams{
		ReviewID: reviewID,
		UserHash: "u-owner-state",
		Content:  "作者自己的回复",
	})
	require.NoError(t, err)

	// 直接覆盖通知 helper 分支：他人操作触发，自操作不触发。
	helperSvc := NewService(fixture.DB, repo, sender, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	helperSvc.filter = svc.filter
	helperSvc.sendVoteNotification(ctx, reviewID, "u-voter-state")
	likeNotif := waitNotification(t, sender.ch, "like")
	assert.Equal(t, ownerUserID, likeNotif.UserID)

	helperSvc.sendReplyNotification(ctx, reviewID, "u-replier-state")
	replyNotif := waitNotification(t, sender.ch, "reply")
	assert.Equal(t, ownerUserID, replyNotif.UserID)

drainLoop:
	for {
		select {
		case <-sender.ch:
		default:
			break drainLoop
		}
	}

	helperSvc.sendVoteNotification(ctx, reviewID, "u-owner-state")
	helperSvc.sendReplyNotification(ctx, reviewID, "u-owner-state")
	select {
	case unexpected := <-sender.ch:
		t.Fatalf("unexpected self-notification: %+v", unexpected)
	case <-time.After(150 * time.Millisecond):
	}

	// UpdateReview: published -> pending_review -> published
	err = svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: reviewID,
		UserHash: "u-owner-state",
		Access:   ownerAccess,
		Title:    strPtr("需要复核"),
		Content:  strPtr("reviewword content"),
		Grade:    strPtr("A"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 4, "difficulty": 5}),
	})
	require.NoError(t, err)

	var status string
	var contentFlag *string
	var reviewCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT status, content_flag FROM reviews WHERE id = $1`, reviewID).Scan(&status, &contentFlag)
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, status)
	require.NotNil(t, contentFlag)
	assert.Equal(t, ContentFlagReview, *contentFlag)
	err = fixture.Pool.QueryRow(ctx, `SELECT review_count FROM courses WHERE id = $1`, courseID).Scan(&reviewCount)
	require.NoError(t, err)
	assert.Equal(t, 0, reviewCount)

	err = svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: reviewID,
		UserHash: "u-owner-state",
		Access:   ownerAccess,
		Title:    strPtr("恢复发布"),
		Content:  strPtr("正常内容用于恢复公开状态"),
		Grade:    strPtr("A+"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 5, "difficulty": 4}),
	})
	require.NoError(t, err)
	err = fixture.Pool.QueryRow(ctx, `SELECT status, content_flag FROM reviews WHERE id = $1`, reviewID).Scan(&status, &contentFlag)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, status)
	assert.Nil(t, contentFlag)
	err = fixture.Pool.QueryRow(ctx, `SELECT review_count FROM courses WHERE id = $1`, courseID).Scan(&reviewCount)
	require.NoError(t, err)
	assert.Equal(t, 1, reviewCount)

	// 隐藏评论删除不再重复扣减课程计数。
	require.NoError(t, svc.DeleteReview(ctx, DeleteReviewParams{ReviewID: hiddenReviewID, UserHash: "u-owner-hidden", Access: ownerAccess}))
	err = fixture.Pool.QueryRow(ctx, `SELECT review_count FROM courses WHERE id = $1`, courseID).Scan(&reviewCount)
	require.NoError(t, err)
	assert.Equal(t, 1, reviewCount)

	// ClearContentFlag: warn 标记仅清除标记，不改变已发布状态与计数。
	require.NoError(t, svc.ClearContentFlag(ctx, warnFlaggedID, "admin-state-1"))
	err = fixture.Pool.QueryRow(ctx, `SELECT status, content_flag FROM reviews WHERE id = $1`, warnFlaggedID).Scan(&status, &contentFlag)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, status)
	require.NotNil(t, contentFlag)
	assert.Equal(t, "cleared", *contentFlag)
	err = fixture.Pool.QueryRow(ctx, `SELECT review_count FROM courses WHERE id = $1`, courseID).Scan(&reviewCount)
	require.NoError(t, err)
	assert.Equal(t, 1, reviewCount)

	// ListReports 对未知状态回退到 StatusAll。
	reporterID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-report-state", UserHash: "u-report-state"})
	_, err = svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               reviewID,
		UserHash:               "u-report-state",
		ReporterInternalUserID: reporterID,
		Reason:                 "spam",
		Description:            "需要处理",
	})
	require.NoError(t, err)
	reports, err := svc.ListReports(ctx, ListReportsParams{Status: "unknown-status", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.NotEmpty(t, reports.List)
	assert.GreaterOrEqual(t, reports.Total, 1)

	_ = replyResult
}
