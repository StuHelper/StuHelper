package review

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

type noopReviewFGAWriter struct{}

func (noopReviewFGAWriter) WriteReviewRelations(context.Context, string, string, string) error {
	return nil
}
func (noopReviewFGAWriter) WriteReportRelations(context.Context, string, string) error {
	return nil
}

type noopNotificationSender struct{}

func (noopNotificationSender) SendReviewNotification(context.Context, ReviewNotification) error {
	return nil
}

func TestReviewService_IntegrationReadAndWritePaths(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "计算机学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "王老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "数据库系统")
	otherCourseID := seedCourse(t, fixture, 4111010006, departmentID, "分布式系统")

	seedReviewWithRatings(t, fixture, "review-read-1", courseID, teacherID, "u-read-1", 4.5, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "数据库真不错", "内容一")
	seedReviewWithRatings(t, fixture, "review-read-2", otherCourseID, teacherID, "u-read-2", 4.0, StatusPublished, ReviewRatings{"teaching": 4, "difficulty": 4}, "分布式很赞", "内容二")

	courseReviews, err := svc.GetCourseReviews(ctx, GetCourseReviewsParams{CourseID: courseID, Page: 1, PageSize: 10, Sort: SortTime})
	require.NoError(t, err)
	require.Len(t, courseReviews.List, 1)
	assert.Equal(t, "review-read-1", courseReviews.List[0].ID)

	batched, err := svc.GetBatchCourseReviews(ctx, GetBatchCourseReviewsParams{CourseIDs: []int64{courseID, otherCourseID}, PageSize: 5, Sort: SortTime})
	require.NoError(t, err)
	assert.Len(t, batched.Reviews[courseID], 1)
	assert.Len(t, batched.Reviews[otherCourseID], 1)

	exists, err := svc.CheckCourseExists(ctx, courseID)
	require.NoError(t, err)
	assert.True(t, exists)

	latest, err := svc.GetLatestReviews(ctx, GetLatestReviewsParams{Page: 1, PageSize: 10, Sort: SortTime})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, latest.Total, 2)

	searched, err := svc.SearchReviews(ctx, SearchReviewsParams{Query: "数据库", DepartmentID: departmentID, TeacherName: "王老师", TermID: "2025-2", Page: 1, PageSize: 10, Sort: SortTime})
	require.NoError(t, err)
	require.Len(t, searched.List, 1)
	assert.Equal(t, "review-read-1", searched.List[0].ID)

	stats, err := svc.GetStats(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.CourseCount, 2)
	assert.GreaterOrEqual(t, stats.ReviewCount, 2)

	dimensions, err := svc.GetRatingDimensions(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, dimensions)

	dimensionNames, err := svc.GetDimensionNames(ctx)
	require.NoError(t, err)
	assert.Contains(t, dimensionNames, "teaching")

	ratingStats, err := svc.GetCourseRatingStats(ctx, courseID)
	require.NoError(t, err)
	assert.NotEmpty(t, ratingStats)

	teachers, err := svc.GetCourseTeachers(ctx, courseID)
	require.NoError(t, err)
	require.Len(t, teachers, 1)
	assert.Equal(t, teacherID, teachers[0].TeacherID)

	reviewByID, err := svc.GetReviewByID(ctx, "review-read-1")
	require.NoError(t, err)
	assert.Equal(t, courseID, reviewByID.CourseID)

	word, err := svc.CreateSensitiveWord(ctx, "敏感词条", "custom", ContentFlagWarn)
	require.NoError(t, err)
	checked, err := svc.CheckContent(ctx, "这是一段敏感词条内容")
	require.NoError(t, err)
	assert.Equal(t, ContentFlagWarn, checked.Level)

	listedWords, totalWords, err := svc.ListSensitiveWords(ctx, "custom", ContentFlagWarn, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totalWords, 1)
	assert.NotEmpty(t, listedWords)

	inactive := false
	require.NoError(t, svc.UpdateSensitiveWord(ctx, word.ID, nil, nil, nil, &inactive))
	require.NoError(t, svc.DeleteSensitiveWord(ctx, word.ID))

	require.NoError(t, svc.RefreshTeacherPublicStats(ctx))
	teacherList, totalTeachers, err := svc.ListTeachers(ctx, "王", &departmentID, "reviews", 1, 10)
	require.NoError(t, err)
	require.Len(t, teacherList, 1)
	assert.Equal(t, 1, totalTeachers)

	hotTeachers, err := svc.ListHotTeachers(ctx, 5)
	require.NoError(t, err)
	require.NotEmpty(t, hotTeachers)

	teacherStats, err := svc.GetTeacherRatingStats(ctx, teacherID)
	require.NoError(t, err)
	assert.Equal(t, teacherID, teacherStats.TeacherID)
	assert.GreaterOrEqual(t, teacherStats.ReviewCount, 2)
	assert.NotEmpty(t, teacherStats.Courses)

	postUserID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-u-post-1", UserHash: "u-post-1"})
	postAccess := fullReviewWriteAccess(postUserID)
	posted, err := svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "新评课",
		Content:              "内容三用于满足评课正文长度",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5, "difficulty": 4},
		UserHash:             "u-post-1",
		AuthorInternalUserID: postUserID,
		Access:               postAccess,
	})
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, posted.Review.Status)

	_, err = svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "重复评课",
		Content:              "重复内容用于满足评课正文长度",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5},
		UserHash:             "u-post-1",
		AuthorInternalUserID: postUserID,
		Access:               postAccess,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAlreadyReviewed)

	require.NoError(t, svc.UpdateReview(ctx, UpdateReviewParams{
		ReviewID: posted.Review.ID,
		UserHash: "u-post-1",
		Access:   postAccess,
		Title:    strPtr("更新后标题"),
		Content:  strPtr("更新后内容用于满足正文长度"),
		Grade:    strPtr("A+"),
		Ratings:  ratingsPtr(ReviewRatings{"teaching": 4, "difficulty": 5}),
	}))

	voteCourseID, err := svc.VoteReview(ctx, VoteReviewParams{ReviewID: posted.Review.ID, UserHash: "u-voter-1", VoteType: "like"})
	require.NoError(t, err)
	assert.Equal(t, courseID, voteCourseID)

	reporterUserID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-u-reporter-1", UserHash: "u-reporter-1"})
	reportID, err := svc.ReportReview(ctx, ReportReviewParams{ReviewID: posted.Review.ID, UserHash: "u-reporter-1", ReporterInternalUserID: reporterUserID, Reason: "spam", Description: "需要处理"})
	require.NoError(t, err)
	assert.NotEmpty(t, reportID)

	reports, err := svc.ListReports(ctx, ListReportsParams{Status: ReportStatusPending, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, reports.List, 1)
	assert.Equal(t, reportID, reports.List[0].ID)

	require.NoError(t, svc.ProcessReport(ctx, ProcessReportParams{ReportID: reportID, Action: "reject", Note: "误报", ResolvedBy: "admin-1"}))
	report, err := repo.GetReportByID(ctx, reportID)
	require.NoError(t, err)
	assert.Equal(t, ReportStatusRejected, report.Status)
	err = svc.ProcessReport(ctx, ProcessReportParams{ReportID: reportID, Action: "hide", Note: "重复处理", ResolvedBy: "admin-2"})
	require.ErrorIs(t, err, ErrInvalidTransition)
	var postedStatus string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, posted.Review.ID).Scan(&postedStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, postedStatus)
	report, err = repo.GetReportByID(ctx, reportID)
	require.NoError(t, err)
	assert.Equal(t, ReportStatusRejected, report.Status)

	hideReviewID := "review-report-hide-1"
	deleteReviewID := "review-report-delete-1"
	seedReviewWithRatings(t, fixture, hideReviewID, courseID, teacherID, "u-report-hide", 4.2, StatusPublished, ReviewRatings{"teaching": 4}, "待隐藏", "待处理内容")
	seedReviewWithRatings(t, fixture, deleteReviewID, courseID, teacherID, "u-report-delete", 4.1, StatusPublished, ReviewRatings{"teaching": 4}, "待删除", "待删除内容")
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 3 WHERE id = $1`, courseID)
	require.NoError(t, err)

	hideReporterID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-u-reporter-hide", UserHash: "u-reporter-hide"})
	hideReportID, err := svc.ReportReview(ctx, ReportReviewParams{ReviewID: hideReviewID, UserHash: "u-reporter-hide", ReporterInternalUserID: hideReporterID, Reason: reportReasonInappropriate, Description: "需要隐藏"})
	require.NoError(t, err)
	require.NoError(t, svc.ProcessReport(ctx, ProcessReportParams{ReportID: hideReportID, Action: "hide", Note: "隐藏处理", ResolvedBy: "admin-2"}))

	var hiddenStatus string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, hideReviewID).Scan(&hiddenStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusHidden, hiddenStatus)
	hideReport, err := repo.GetReportByID(ctx, hideReportID)
	require.NoError(t, err)
	assert.Equal(t, ReportStatusResolved, hideReport.Status)

	deleteReporterID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-u-reporter-delete", UserHash: "u-reporter-delete"})
	deleteReportID, err := svc.ReportReview(ctx, ReportReviewParams{ReviewID: deleteReviewID, UserHash: "u-reporter-delete", ReporterInternalUserID: deleteReporterID, Reason: reportReasonOther, Description: "需要删除"})
	require.NoError(t, err)
	require.NoError(t, svc.ProcessReport(ctx, ProcessReportParams{ReportID: deleteReportID, Action: "delete", Note: "删除处理", ResolvedBy: "admin-3"}))

	var deletedReportStatus string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, deleteReviewID).Scan(&deletedReportStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusDeleted, deletedReportStatus)
	deleteReport, err := repo.GetReportByID(ctx, deleteReportID)
	require.NoError(t, err)
	assert.Equal(t, ReportStatusResolved, deleteReport.Status)

	require.NoError(t, svc.DeleteReview(ctx, DeleteReviewParams{ReviewID: posted.Review.ID, UserHash: "u-post-1", Access: postAccess}))
	var deletedStatus string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, posted.Review.ID).Scan(&deletedStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusDeleted, deletedStatus)
}

func seedReviewWithRatings(t *testing.T, fixture *postgresfixture.Fixture, reviewID string, courseID, teacherID int64, userHash string, avgRating float64, status string, ratings ReviewRatings, title, content string) {
	t.Helper()
	ratingsJSON, err := json.Marshal(ratings)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		INSERT INTO reviews (
			id, course_id, school_id, teacher_id, term_id, user_hash, title, content, grade, ratings, avg_rating, status
		) VALUES (
			$1, $2, (SELECT school_id FROM courses WHERE id = $2), $3, '2025-2', $4, $5, $6, 'A', $7::jsonb, $8, $9
		)
	`, reviewID, courseID, teacherID, userHash, title, content, string(ratingsJSON), avgRating, status)
	require.NoError(t, err)
}

type seedUserParams struct {
	CasdoorSubject string
	UserHash       string
}

func seedUser(t *testing.T, fixture *postgresfixture.Fixture, params seedUserParams) int64 {
	t.Helper()
	var userID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email, user_hash)
		VALUES ($1, $1, $2, NULLIF($3, ''))
		ON CONFLICT (casdoor_subject) DO UPDATE SET
			user_hash = COALESCE(EXCLUDED.user_hash, users.user_hash)
		RETURNING id
	`, params.CasdoorSubject, params.CasdoorSubject+"@example.test", params.UserHash).Scan(&userID)
	require.NoError(t, err)
	return userID
}

func TestReviewService_IntegrationInteractionPaths(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "软件学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "陈老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "软件工程")
	seedReviewWithRatings(t, fixture, "review-owner-1", courseID, teacherID, "u-owner-1", 4.8, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "很棒", "值得推荐")

	// 收藏链路
	require.NoError(t, svc.AddFavorite(ctx, AddFavoriteParams{UserHash: "u-fav-1", CourseID: courseID}))
	favorited, err := svc.GetFavoriteStatus(ctx, "u-fav-1", courseID)
	require.NoError(t, err)
	assert.True(t, favorited)

	favorites, err := svc.GetUserFavorites(ctx, GetUserFavoritesParams{UserHash: "u-fav-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, favorites.List, 1)
	assert.Equal(t, courseID, favorites.List[0].ID)
	assert.Equal(t, 1, favorites.Total)

	require.NoError(t, svc.RemoveFavorite(ctx, "u-fav-1", courseID))
	favorited, err = svc.GetFavoriteStatus(ctx, "u-fav-1", courseID)
	require.NoError(t, err)
	assert.False(t, favorited)

	// 草稿链路
	draft, err := svc.SaveDraft(ctx, SaveDraftParams{
		UserHash:  "u-draft-1",
		CourseID:  &courseID,
		TeacherID: &teacherID,
		TermID:    "2025-2",
		Title:     "草稿标题",
		Content:   "草稿内容",
		Grade:     "A-",
		Ratings:   ReviewRatings{"teaching": 4, "difficulty": 3},
	})
	require.NoError(t, err)
	require.NotNil(t, draft.CourseID)
	assert.Equal(t, courseID, *draft.CourseID)
	assert.Equal(t, "草稿标题", draft.Title)

	loadedDraft, err := svc.GetDraft(ctx, "u-draft-1")
	require.NoError(t, err)
	assert.Equal(t, draft.ID, loadedDraft.ID)
	assert.Equal(t, "草稿内容", loadedDraft.Content)
	assert.Equal(t, 4, loadedDraft.Ratings["teaching"])

	otherCourseID := seedCourse(t, fixture, 4111010006, departmentID, "编译原理")
	overwrittenDraft, err := svc.SaveDraft(ctx, SaveDraftParams{
		UserHash: "u-draft-1",
		CourseID: &otherCourseID,
		Title:    "覆盖后的草稿",
		Content:  "覆盖后的内容",
		Ratings:  ReviewRatings{"teaching": 5},
	})
	require.NoError(t, err)
	assert.Equal(t, draft.ID, overwrittenDraft.ID)
	require.NotNil(t, overwrittenDraft.CourseID)
	assert.Equal(t, otherCourseID, *overwrittenDraft.CourseID)
	assert.Equal(t, "覆盖后的草稿", overwrittenDraft.Title)

	courseFreeDraft, err := svc.SaveDraft(ctx, SaveDraftParams{
		UserHash: "u-draft-free",
		Title:    "还没选课程",
		Content:  "先写内容再搜索课程",
	})
	require.NoError(t, err)
	assert.Nil(t, courseFreeDraft.CourseID)
	assert.Empty(t, courseFreeDraft.TermID)
	assert.Equal(t, "还没选课程", courseFreeDraft.Title)

	require.NoError(t, svc.DeleteDraft(ctx, "u-draft-1"))
	_, err = svc.GetDraft(ctx, "u-draft-1")
	require.ErrorIs(t, err, ErrDraftNotFound)

	// 用户评论 / 投票链路
	userReviews, err := svc.GetUserReviews(ctx, GetUserReviewsParams{UserHash: "u-owner-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, userReviews.List, 1)
	assert.Equal(t, "review-owner-1", userReviews.List[0].ID)

	voteCourseID, err := svc.VoteReview(ctx, VoteReviewParams{ReviewID: "review-owner-1", UserHash: "u-voter-2", VoteType: "like"})
	require.NoError(t, err)
	assert.Equal(t, courseID, voteCourseID)

	userVotes, err := svc.GetUserVotes(ctx, GetUserVotesParams{UserHash: "u-voter-2", VoteType: "like", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, userVotes.List, 1)
	assert.Equal(t, "review-owner-1", userVotes.List[0].ID)

	// 回复链路
	replyResult, err := svc.CreateReply(ctx, CreateReplyParams{ReviewID: "review-owner-1", UserHash: "u-replier-1", Content: "收到，感谢分享"})
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, replyResult.Reply.Status)

	replies, err := svc.GetReplies(ctx, GetRepliesParams{ReviewID: "review-owner-1", UserHash: "u-replier-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, replies.List, 1)
	assert.True(t, replies.List[0].IsOwner)
	assert.Empty(t, replies.List[0].UserHash)
	assert.Equal(t, 1, replies.Total)

	require.NoError(t, svc.DeleteReply(ctx, DeleteReplyParams{ReplyID: replyResult.Reply.ID, UserHash: "u-replier-1"}))
	replies, err = svc.GetReplies(ctx, GetRepliesParams{ReviewID: "review-owner-1", UserHash: "u-replier-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Empty(t, replies.List)
	assert.Equal(t, 0, replies.Total)
}

func TestAdminUpdateReplyPublishesPendingReplyAndRestoresCount(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	svc.filter = seededFilter([]SensitiveWord{{Word: "reviewword", Level: ContentFlagReview}})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "回复审核学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "回复审核老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "回复审核课程")
	seedReviewWithRatings(t, fixture, "review-reply-admin-1", courseID, teacherID, "u-owner-reply-admin", 4.8, StatusPublished, ReviewRatings{"teaching": 5}, "回复审核", "原评课")

	replyResult, err := svc.CreateReply(ctx, CreateReplyParams{ReviewID: "review-reply-admin-1", UserHash: "u-replier-admin", Content: "reviewword 需要审核"})
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, replyResult.Reply.Status)

	var replyCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT reply_count FROM reviews WHERE id = $1`, "review-reply-admin-1").Scan(&replyCount)
	require.NoError(t, err)
	assert.Equal(t, 0, replyCount)

	result, err := svc.AdminUpdateReply(ctx, AdminUpdateReplyParams{ReplyID: replyResult.Reply.ID, Action: "restore", AdminID: "admin-reply-1"})
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, result.OldStatus)

	var status string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM review_replies WHERE id = $1`, replyResult.Reply.ID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, status)
	err = fixture.Pool.QueryRow(ctx, `SELECT reply_count FROM reviews WHERE id = $1`, "review-reply-admin-1").Scan(&replyCount)
	require.NoError(t, err)
	assert.Equal(t, 1, replyCount)
}

func TestCreateReplyRejectsParentFromDifferentReview(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "回复父级学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "回复父级老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "回复父级课程")
	seedReviewWithRatings(t, fixture, "review-parent-a", courseID, teacherID, "u-parent-a", 4.8, StatusPublished, ReviewRatings{"teaching": 5}, "父级 A", "原评课 A")
	seedReviewWithRatings(t, fixture, "review-parent-b", courseID, teacherID, "u-parent-b", 4.7, StatusPublished, ReviewRatings{"teaching": 5}, "父级 B", "原评课 B")

	parent, err := svc.CreateReply(ctx, CreateReplyParams{ReviewID: "review-parent-a", UserHash: "u-parent-replier", Content: "父回复"})
	require.NoError(t, err)
	parentID := parent.Reply.ID

	_, err = svc.CreateReply(ctx, CreateReplyParams{ReviewID: "review-parent-b", ParentID: &parentID, UserHash: "u-child-replier", Content: "跨评课子回复"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReplyNotFound)
}

func TestPostReviewRejectsTeacherFromDifferentSchool(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	courseDepartmentID := seedDepartment(t, fixture, 4111010006, "跨校课程学院")
	seedSchool(t, fixture, 4111010007, "4111010007", "跨校教师学校")
	teacherDepartmentID := seedDepartment(t, fixture, 4111010007, "跨校教师学院")
	courseID := seedCourse(t, fixture, 4111010006, courseDepartmentID, "跨校校验课程")
	teacherID := seedTeacher(t, fixture, 4111010007, "跨校教师", teacherDepartmentID)

	crossSchoolAuthorID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-cross-school-teacher", UserHash: "u-cross-school-teacher"})
	_, err := svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "跨校教师应被拒绝",
		Content:              "这条评课不能把其它学校教师挂到本课程上",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5},
		UserHash:             "u-cross-school-teacher",
		AuthorInternalUserID: crossSchoolAuthorID,
		Access:               fullReviewWriteAccess(crossSchoolAuthorID),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTeacherNotFound)
}

func TestPostReviewAndReportMaterializeSchoolID(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	schoolID := int64(4111010006)
	departmentID := seedDepartment(t, fixture, schoolID, "物化学院")
	teacherID := seedTeacher(t, fixture, schoolID, "物化老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "物化课程")
	authorID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-materialized-review", UserHash: "u-materialized-review"})
	authorAccess := fullReviewWriteAccess(authorID)

	created, err := svc.PostReview(ctx, PostReviewParams{
		CourseID:             courseID,
		TeacherID:            &teacherID,
		TermID:               "2025-2",
		Title:                "学校物化",
		Content:              "评课和举报都应该物化学校 ID",
		Grade:                "A",
		Ratings:              ReviewRatings{"teaching": 5},
		UserHash:             "u-materialized-review",
		AuthorInternalUserID: authorID,
		Access:               authorAccess,
	})
	require.NoError(t, err)

	var reviewSchoolID int64
	err = fixture.Pool.QueryRow(ctx, `SELECT school_id FROM reviews WHERE id = $1`, created.Review.ID).Scan(&reviewSchoolID)
	require.NoError(t, err)
	assert.Equal(t, schoolID, reviewSchoolID)
	resolvedReviewSchoolID, err := svc.GetReviewSchoolID(ctx, created.Review.ID)
	require.NoError(t, err)
	assert.Equal(t, schoolID, resolvedReviewSchoolID)

	reporterID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-materialized-report", UserHash: "u-materialized-report"})
	reportID, err := svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               created.Review.ID,
		UserHash:               "u-materialized-report",
		ReporterInternalUserID: reporterID,
		Reason:                 "spam",
		Description:            "需要处理",
	})
	require.NoError(t, err)

	var reportSchoolID int64
	err = fixture.Pool.QueryRow(ctx, `SELECT school_id FROM review_reports WHERE id = $1`, reportID).Scan(&reportSchoolID)
	require.NoError(t, err)
	assert.Equal(t, schoolID, reportSchoolID)
	resolvedReportSchoolID, err := svc.GetReportSchoolID(ctx, reportID)
	require.NoError(t, err)
	assert.Equal(t, schoolID, resolvedReportSchoolID)
}

func TestSaveDraftRejectsTeacherFromDifferentSchool(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	courseDepartmentID := seedDepartment(t, fixture, 4111010006, "草稿课程学院")
	seedSchool(t, fixture, 4111010008, "4111010008", "草稿跨校教师学校")
	teacherDepartmentID := seedDepartment(t, fixture, 4111010008, "草稿跨校教师学院")
	courseID := seedCourse(t, fixture, 4111010006, courseDepartmentID, "草稿跨校校验课程")
	teacherID := seedTeacher(t, fixture, 4111010008, "草稿跨校教师", teacherDepartmentID)

	_, err := svc.SaveDraft(ctx, SaveDraftParams{
		UserHash:  "u-draft-cross-school-teacher",
		CourseID:  &courseID,
		TeacherID: &teacherID,
		TermID:    "2025-2",
		Title:     "跨校教师草稿",
		Content:   "草稿不能保存其它学校教师",
		Grade:     "A",
		Ratings:   ReviewRatings{"teaching": 5},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTeacherNotFound)
}

func seedSchool(t *testing.T, fixture *postgresfixture.Fixture, schoolID int64, code string, name string) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO schools (id, code, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, schoolID, code, name)
	require.NoError(t, err)
}
