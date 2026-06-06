package review

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestReviewService_AdminAndExportIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	h := &Handler{service: svc}
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "自动化学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "赵老师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "控制理论")
	otherCourseID := seedCourse(t, fixture, 4111010006, departmentID, "信号系统")

	publishedID := "550e8400-e29b-41d4-a716-446655440101"
	hiddenID := "550e8400-e29b-41d4-a716-446655440102"
	pendingFlaggedID := "550e8400-e29b-41d4-a716-446655440103"
	warnFlaggedID := "550e8400-e29b-41d4-a716-446655440104"
	batchDeleteID := "550e8400-e29b-41d4-a716-446655440105"
	batchHideID := "550e8400-e29b-41d4-a716-446655440106"
	pendingRestoreID := "550e8400-e29b-41d4-a716-446655440107"
	hiddenDeleteID := "550e8400-e29b-41d4-a716-446655440108"
	batchPendingRestoreID := "550e8400-e29b-41d4-a716-446655440109"

	seedReviewWithRatings(t, fixture, publishedID, courseID, teacherID, "u-admin-1", 4.6, StatusPublished, ReviewRatings{"teaching": 5, "difficulty": 4}, "已发布评论", "内容一")
	seedReviewWithRatings(t, fixture, hiddenID, courseID, teacherID, "u-admin-2", 3.9, StatusHidden, ReviewRatings{"teaching": 4, "difficulty": 4}, "隐藏评论", "内容二")
	seedReviewWithRatings(t, fixture, pendingFlaggedID, courseID, teacherID, "u-admin-3", 4.2, StatusPendingReview, ReviewRatings{"teaching": 4, "difficulty": 5}, "待复核评论", "内容三")
	seedReviewWithRatings(t, fixture, warnFlaggedID, otherCourseID, teacherID, "u-admin-4", 3.7, StatusPublished, ReviewRatings{"teaching": 3, "difficulty": 4}, "警告评论", "内容四")
	seedReviewWithRatings(t, fixture, batchDeleteID, otherCourseID, teacherID, "u-admin-5", 4.0, StatusPublished, ReviewRatings{"teaching": 4, "difficulty": 4}, "批量删除评论", "内容五")
	seedReviewWithRatings(t, fixture, batchHideID, courseID, teacherID, "u-admin-6", 4.1, StatusPublished, ReviewRatings{"teaching": 4}, "批量屏蔽评论", "内容六")
	seedReviewWithRatings(t, fixture, pendingRestoreID, courseID, teacherID, "u-admin-7", 4.3, StatusPendingReview, ReviewRatings{"teaching": 4}, "待恢复评论", "内容七")
	seedReviewWithRatings(t, fixture, hiddenDeleteID, courseID, teacherID, "u-admin-8", 3.6, StatusHidden, ReviewRatings{"teaching": 3}, "隐藏待删除", "内容八")
	seedReviewWithRatings(t, fixture, batchPendingRestoreID, otherCourseID, teacherID, "u-admin-9", 4.0, StatusPendingReview, ReviewRatings{"teaching": 4}, "批量恢复待审", "内容九")

	contentFlagReview := ContentFlagReview
	contentFlagWarn := ContentFlagWarn
	_, err := fixture.Pool.Exec(ctx, `UPDATE reviews SET content_flag = $2 WHERE id = $1`, pendingFlaggedID, contentFlagReview)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE reviews SET content_flag = $2 WHERE id = $1`, pendingRestoreID, contentFlagReview)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE reviews SET content_flag = $2 WHERE id = $1`, batchPendingRestoreID, contentFlagReview)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE reviews SET content_flag = $2 WHERE id = $1`, warnFlaggedID, contentFlagWarn)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 3 WHERE id = $1`, courseID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 2 WHERE id = $1`, otherCourseID)
	require.NoError(t, err)

	reporterID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "ext-reporter-admin", UserHash: "u-reporter-admin"})
	reportID, err := svc.ReportReview(ctx, ReportReviewParams{ReviewID: publishedID, UserHash: "u-reporter-admin", ReporterInternalUserID: reporterID, Reason: "spam", Description: "待处理"})
	require.NoError(t, err)
	require.NotEmpty(t, reportID)

	stats, err := svc.GetAdminStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 9, stats.TotalReviews)
	assert.Equal(t, 4, stats.PublishedReviews)
	assert.Equal(t, 2, stats.HiddenReviews)
	assert.Equal(t, 1, stats.PendingReports)
	assert.Equal(t, 1, stats.TotalReports)

	allReviews, err := svc.ListAllReviews(ctx, ListAllReviewsParams{Status: StatusAll, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 9, allReviews.Total)
	assert.Len(t, allReviews.List, 9)

	publishedReviews, err := svc.ListAllReviews(ctx, ListAllReviewsParams{Status: StatusPublished, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 4, publishedReviews.Total)

	trend, err := svc.GetRatingTrend(ctx, courseID)
	require.NoError(t, err)
	require.Len(t, trend, 1)
	assert.Equal(t, "2025-2", trend[0].TermID)
	assert.Equal(t, 2, trend[0].Count)

	hotCourses, err := svc.GetHotCourses(ctx, "all", 10)
	require.NoError(t, err)
	require.NotEmpty(t, hotCourses)
	assert.Equal(t, courseID, hotCourses[0].CourseID)

	flagged, totalFlagged, err := svc.ListFlaggedReviews(ctx, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 4, totalFlagged)
	require.Len(t, flagged, 4)
	assert.Nil(t, flagged[0].ContentFlagClearedBy)
	assert.Empty(t, flagged[0].UserHash)

	require.NoError(t, svc.ClearContentFlag(ctx, pendingFlaggedID, "admin-clear-1"))
	var clearedStatus string
	var clearedFlag *string
	err = fixture.Pool.QueryRow(ctx, `SELECT status, content_flag FROM reviews WHERE id = $1`, pendingFlaggedID).Scan(&clearedStatus, &clearedFlag)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, clearedStatus)
	require.NotNil(t, clearedFlag)
	assert.Equal(t, "cleared", *clearedFlag)

	updateResult, err := svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{ReviewID: publishedID, Action: "hide", Reason: "spam", AdminID: "admin-1"})
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, updateResult.OldStatus)
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, publishedID).Scan(&clearedStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusHidden, clearedStatus)

	restoreResult, err := svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{ReviewID: pendingRestoreID, Action: "restore", Reason: "审核通过", AdminID: "admin-restore"})
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, restoreResult.OldStatus)
	var restoredStatus string
	var restoredFlag *string
	err = fixture.Pool.QueryRow(ctx, `SELECT status, content_flag FROM reviews WHERE id = $1`, pendingRestoreID).Scan(&restoredStatus, &restoredFlag)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, restoredStatus)
	require.NotNil(t, restoredFlag)
	assert.Equal(t, "cleared", *restoredFlag)

	restoreHiddenResult, err := svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{ReviewID: hiddenID, Action: "restore", Reason: "恢复展示", AdminID: "admin-restore-hidden"})
	require.NoError(t, err)
	assert.Equal(t, StatusHidden, restoreHiddenResult.OldStatus)
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, hiddenID).Scan(&restoredStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusPublished, restoredStatus)

	deleteHiddenResult, err := svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{ReviewID: hiddenDeleteID, Action: "delete", Reason: "隐藏删除", AdminID: "admin-delete"})
	require.NoError(t, err)
	assert.Equal(t, StatusHidden, deleteHiddenResult.OldStatus)
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, hiddenDeleteID).Scan(&restoredStatus)
	require.NoError(t, err)
	assert.Equal(t, StatusDeleted, restoredStatus)

	require.NoError(t, svc.AdminEditReview(ctx, AdminEditReviewParams{ReviewID: hiddenID, Title: "管理员改标题", Content: "管理员改内容", Reason: "修正描述", AdminID: "admin-2"}))
	var title, content string
	err = fixture.Pool.QueryRow(ctx, `SELECT title, content FROM reviews WHERE id = $1`, hiddenID).Scan(&title, &content)
	require.NoError(t, err)
	assert.Equal(t, "管理员改标题", title)
	assert.Equal(t, "管理员改内容", content)

	batchHide, err := svc.BatchUpdateReviews(ctx, BatchUpdateReviewsParams{
		IDs:     []string{batchHideID},
		Action:  "hide",
		AdminID: "admin-batch-hide",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, batchHide.Affected)

	var moderatedBy *string
	err = fixture.Pool.QueryRow(ctx, `SELECT moderated_by FROM reviews WHERE id = $1`, batchHideID).Scan(&moderatedBy)
	require.NoError(t, err)
	require.NotNil(t, moderatedBy)
	assert.Equal(t, "admin-batch-hide", *moderatedBy)

	batchResult, err := svc.BatchUpdateReviews(ctx, BatchUpdateReviewsParams{
		IDs:     []string{warnFlaggedID, batchDeleteID},
		Action:  "delete",
		AdminID: "admin-batch-delete",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, batchResult.Affected)

	restoredReviewID := "550e8400-e29b-41d4-a716-446655440110"
	seedReviewWithRatings(t, fixture, restoredReviewID, otherCourseID, teacherID, "u-admin-6", 4.1, StatusHidden, ReviewRatings{"teaching": 4}, "待恢复评论", "内容六")
	_, err = fixture.Pool.Exec(ctx, `
		UPDATE reviews
		SET moderation_reason = 'spam', moderated_by = 'admin-hide', moderated_at = NOW()
		WHERE id = $1
	`, restoredReviewID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 0 WHERE id = $1`, otherCourseID)
	require.NoError(t, err)
	batchRestore, err := svc.BatchUpdateReviewsWithAudit(ctx, BatchUpdateReviewsParams{IDs: []string{restoredReviewID}, Action: "restore"}, "admin-batch", "root")
	require.NoError(t, err)
	assert.EqualValues(t, 1, batchRestore.Affected)

	err = fixture.Pool.QueryRow(ctx, `SELECT moderated_by FROM reviews WHERE id = $1`, restoredReviewID).Scan(&moderatedBy)
	require.NoError(t, err)
	assert.Nil(t, moderatedBy)

	batchPendingRestore, err := svc.BatchUpdateReviews(ctx, BatchUpdateReviewsParams{
		IDs:     []string{batchPendingRestoreID},
		Action:  "restore",
		AdminID: "admin-batch-restore",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, batchPendingRestore.Affected)

	var clearedBy *string
	err = fixture.Pool.QueryRow(ctx, `SELECT content_flag_cleared_by FROM reviews WHERE id = $1`, batchPendingRestoreID).Scan(&clearedBy)
	require.NoError(t, err)
	require.NotNil(t, clearedBy)
	assert.Equal(t, "admin-batch-restore", *clearedBy)

	_, err = svc.BatchUpdateReviews(ctx, BatchUpdateReviewsParams{IDs: []string{hiddenDeleteID}, Action: "restore", AdminID: "admin-batch"})
	require.ErrorIs(t, err, ErrInvalidTransition)

	var deletedCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE id = ANY($1) AND status = 'deleted'`, []string{warnFlaggedID, batchDeleteID}).Scan(&deletedCount)
	require.NoError(t, err)
	assert.Equal(t, 2, deletedCount)

	require.NoError(t, svc.LogOperation(ctx, LogOperationParams{
		AdminUserID:   "admin-log-1",
		AdminUsername: "root",
		Action:        "hide",
		ResourceType:  "review",
		ResourceID:    publishedID,
		OldValue:      map[string]string{"status": "published"},
		NewValue:      map[string]string{"status": "hidden"},
		IPAddress:     "127.0.0.1",
		UserAgent:     "go-test",
	}))

	logs, err := svc.GetOperationLogs(ctx, GetOperationLogsParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, logs.Total, 1)
	require.NotEmpty(t, logs.List)
	assert.Equal(t, "hide", logs.List[0].Action)

	var oldLogID string
	err = fixture.Pool.QueryRow(ctx, `SELECT id FROM audit_events WHERE category = 'admin_operation' ORDER BY created_at ASC LIMIT 1`).Scan(&oldLogID)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(ctx, `UPDATE audit_events SET created_at = NOW() - INTERVAL '120 days' WHERE id = $1`, oldLogID)
	require.NoError(t, err)
	deletedLogs, err := svc.CleanupOldOperationLogs(ctx, 90)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deletedLogs, int64(1))

	exported := make([]string, 0, 8)
	require.NoError(t, svc.StreamExportReviews(ctx, StatusAll, func(r Review) error {
		exported = append(exported, r.ID)
		return nil
	}))
	assert.Contains(t, exported, publishedID)
	assert.Contains(t, exported, hiddenID)

	adminTeachers, totalTeachers, err := svc.ListAdminTeachers(ctx, "赵", 0, 20, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totalTeachers, 1)
	require.NotEmpty(t, adminTeachers)
	assert.Equal(t, teacherID, adminTeachers[0].ID)

	_, err = svc.CreateTeacher(ctx, "无部门老师", nil)
	require.ErrorIs(t, err, ErrTeacherDepartmentRequired)

	missingDepartmentID := departmentID + 999999
	_, err = svc.CreateTeacher(ctx, "缺失院系老师", &missingDepartmentID)
	require.ErrorIs(t, err, ErrTeacherDepartmentNotFound)

	_, err = svc.CreateTeacher(ctx, strings.Repeat("师", maxAdminTeacherNameRunes+1), &departmentID)
	require.ErrorIs(t, err, ErrTeacherNameInvalid)

	require.ErrorIs(t, svc.DeleteTeacher(ctx, teacherID), ErrTeacherHasReviews)

	createdTeacher, err := svc.CreateTeacher(ctx, "新老师", &departmentID)
	require.NoError(t, err)
	require.NotNil(t, createdTeacher)
	assert.Equal(t, "新老师", createdTeacher.Name)

	require.NoError(t, svc.UpdateTeacher(ctx, createdTeacher.ID, "更新老师", &departmentID))
	var updatedTeacherName string
	err = fixture.Pool.QueryRow(ctx, `SELECT name FROM teachers WHERE id = $1`, createdTeacher.ID).Scan(&updatedTeacherName)
	require.NoError(t, err)
	assert.Equal(t, "更新老师", updatedTeacherName)

	require.NoError(t, svc.DeleteTeacher(ctx, createdTeacher.ID))
	var remainingTeacherCount int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM teachers WHERE id = $1`, createdTeacher.ID).Scan(&remainingTeacherCount)
	require.NoError(t, err)
	assert.Equal(t, 0, remainingTeacherCount)

	// handler 层覆盖：operation logs / export / content flag list
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	h.GetOperationLogs(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total")

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/export?format=json&status=all", nil)
	h.ExportReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), publishedID)
	assert.Contains(t, w.Body.String(), "# EXPORT_COMPLETE")

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/export?format=csv&status=all", nil)
	h.ExportReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "课程名称")
	assert.Contains(t, w.Body.String(), "# EXPORT_COMPLETE")

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/content-flags", nil)
	h.ListFlaggedReviews(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), warnFlaggedID)
}
