package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestProcessReportPreservesUserDeletedReview(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "举报状态机学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "举报状态机教师", departmentID)
	fixedUpdatedAt := time.Date(2000, time.January, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name         string
		action       string
		courseName   string
		reviewID     string
		ownerHash    string
		ownerSubject string
		reporterHash string
		reporterSub  string
	}{
		{
			name:         "hide closes report without resurrecting review",
			action:       "hide",
			courseName:   "举报隐藏状态机",
			reviewID:     "550e8400-e29b-41d4-a716-446655441801",
			ownerHash:    "u-report-state-owner-hide",
			ownerSubject: "ext-report-state-owner-hide",
			reporterHash: "u-report-state-reporter-hide",
			reporterSub:  "ext-report-state-reporter-hide",
		},
		{
			name:         "delete closes report without rewriting review",
			action:       "delete",
			courseName:   "举报删除状态机",
			reviewID:     "550e8400-e29b-41d4-a716-446655441802",
			ownerHash:    "u-report-state-owner-delete",
			ownerSubject: "ext-report-state-owner-delete",
			reporterHash: "u-report-state-reporter-delete",
			reporterSub:  "ext-report-state-reporter-delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			courseID := seedCourse(t, fixture, 4111010006, departmentID, tt.courseName)
			ownerID := seedUser(t, fixture, seedUserParams{
				CasdoorSubject: tt.ownerSubject,
				UserHash:       tt.ownerHash,
			})
			seedReviewWithRatings(
				t,
				fixture,
				tt.reviewID,
				courseID,
				teacherID,
				tt.ownerHash,
				4.5,
				StatusPublished,
				ReviewRatings{"teaching": 5},
				"用户随后删除的评课",
				"举报创建之后由作者主动删除的评课内容",
			)
			_, err := fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
			require.NoError(t, err)

			reporterID := seedUser(t, fixture, seedUserParams{
				CasdoorSubject: tt.reporterSub,
				UserHash:       tt.reporterHash,
			})
			reportID, err := svc.ReportReview(ctx, ReportReviewParams{
				ReviewID:               tt.reviewID,
				UserHash:               tt.reporterHash,
				ReporterInternalUserID: reporterID,
				Reason:                 reportReasonOther,
				Description:            "删除前创建的待处理举报",
			})
			require.NoError(t, err)

			require.NoError(t, svc.DeleteReview(ctx, DeleteReviewParams{
				ReviewID: tt.reviewID,
				UserHash: tt.ownerHash,
				Access:   fullReviewWriteAccess(ownerID),
			}))
			_, err = fixture.Pool.Exec(ctx, `UPDATE reviews SET updated_at = $2 WHERE id = $1`, tt.reviewID, fixedUpdatedAt)
			require.NoError(t, err)

			require.NoError(t, svc.ProcessReport(ctx, ProcessReportParams{
				ReportID:   reportID,
				Action:     tt.action,
				Note:       "评课已由作者删除，关闭举报",
				ResolvedBy: "admin-report-state",
			}))

			var (
				status      string
				updatedAt   time.Time
				reviewCount int
			)
			err = fixture.Pool.QueryRow(ctx, `
				SELECT r.status, r.updated_at, c.review_count
				FROM reviews r
				JOIN courses c ON c.id = r.course_id
				WHERE r.id = $1
			`, tt.reviewID).Scan(&status, &updatedAt, &reviewCount)
			require.NoError(t, err)
			assert.Equal(t, StatusDeleted, status)
			assert.True(t, updatedAt.Equal(fixedUpdatedAt), "report processing must not rewrite a user-deleted review")
			assert.Equal(t, 0, reviewCount)

			report, err := repo.GetReportByID(ctx, reportID)
			require.NoError(t, err)
			assert.Equal(t, ReportStatusResolved, report.Status)
			require.NotNil(t, report.ResolvedBy)
			assert.Equal(t, "admin-report-state", *report.ResolvedBy)
			require.NotNil(t, report.ResolutionNote)
			assert.Equal(t, "评课已由作者删除，关闭举报", *report.ResolutionNote)
			require.NotNil(t, report.ResolvedAt)

			_, err = svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{
				ReviewID: tt.reviewID,
				Action:   "restore",
				Reason:   "不应恢复用户已删除内容",
				AdminID:  "admin-report-state",
			})
			require.ErrorIs(t, err, ErrInvalidTransition)
		})
	}
}

func TestProcessReportUsesAdminReviewTransitionGuard(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 4111010006, "举报转换校验学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "举报转换校验教师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "举报转换校验课程")
	reviewID := "550e8400-e29b-41d4-a716-446655441803"
	seedReviewWithRatings(
		t,
		fixture,
		reviewID,
		courseID,
		teacherID,
		"u-report-transition-owner",
		4.5,
		StatusPublished,
		ReviewRatings{"teaching": 5},
		"待管理员隐藏的评课",
		"先创建举报再从另一管理员入口隐藏",
	)
	_, err := fixture.Pool.Exec(ctx, `UPDATE courses SET review_count = 1 WHERE id = $1`, courseID)
	require.NoError(t, err)

	reporterID := seedUser(t, fixture, seedUserParams{
		CasdoorSubject: "ext-report-transition-reporter",
		UserHash:       "u-report-transition-reporter",
	})
	reportID, err := svc.ReportReview(ctx, ReportReviewParams{
		ReviewID:               reviewID,
		UserHash:               "u-report-transition-reporter",
		ReporterInternalUserID: reporterID,
		Reason:                 reportReasonOther,
		Description:            "等待处理",
	})
	require.NoError(t, err)

	_, err = svc.AdminUpdateReview(ctx, AdminUpdateReviewParams{
		ReviewID: reviewID,
		Action:   "hide",
		Reason:   "由直接评课入口先行隐藏",
		AdminID:  "admin-transition",
	})
	require.NoError(t, err)

	err = svc.ProcessReport(ctx, ProcessReportParams{
		ReportID:   reportID,
		Action:     "hide",
		Note:       "不应重复转换",
		ResolvedBy: "admin-report",
	})
	require.ErrorIs(t, err, ErrInvalidTransition)

	var status string
	err = fixture.Pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, reviewID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, StatusHidden, status)

	report, err := repo.GetReportByID(ctx, reportID)
	require.NoError(t, err)
	assert.Equal(t, ReportStatusPending, report.Status)
	assert.Nil(t, report.ResolvedBy)
	assert.Nil(t, report.ResolvedAt)
}
