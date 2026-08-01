package review

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type recordingReviewFGAWriter struct {
	reviewIDs []string
	reportIDs []string
	err       error
}

func (r *recordingReviewFGAWriter) Check(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r *recordingReviewFGAWriter) WriteReviewRelations(_ context.Context, reviewID, _ string, _ string) error {
	r.reviewIDs = append(r.reviewIDs, reviewID)
	return r.err
}
func (r *recordingReviewFGAWriter) WriteReportRelations(_ context.Context, reportID, _ string) error {
	r.reportIDs = append(r.reportIDs, reportID)
	return r.err
}

type noopReviewSender2 struct{}

func (noopReviewSender2) SendReviewNotification(context.Context, ReviewNotification) error {
	return nil
}

func TestReviewFilterRefreshAndBackgroundJobs(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	schoolID := int64(4111010006)
	writer := &recordingReviewFGAWriter{}
	svc := NewService(fixture.DB, repo, noopReviewSender2{}, writer, fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: schoolID}},
		subject: &reviewaccess.Subject{InternalUserID: 42, SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	})
	ctx := context.Background()

	word, err := svc.CreateSensitiveWord(ctx, "危险词", "custom", ContentFlagWarn)
	require.NoError(t, err)
	require.NoError(t, svc.filter.Refresh(ctx))
	result, err := svc.CheckContent(ctx, "这里包含危险词")
	require.NoError(t, err)
	assert.Equal(t, ContentFlagWarn, result.Level)
	require.NoError(t, svc.DeleteSensitiveWord(ctx, word.ID))

	departmentID := seedDepartment(t, fixture, schoolID, "法学院")
	teacherID := seedTeacher(t, fixture, schoolID, "背景老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "法理学")
	seedReviewWithRatings(t, fixture, "550e8400-e29b-41d4-a716-446655440881", courseID, teacherID, "u-bg-1", 4.0, StatusPublished, ReviewRatings{"teaching": 4}, "背景任务", "背景内容")

	redisFixture := redisfixture.Start(t)
	h := NewHandler(HandlerConfig{
		CacheHelper:            cache.NewHelper(redisFixture.Client),
		Service:                svc,
		Redis:                  redisFixture.Client,
		RateLimit:              config.ReviewRateLimitConfig{},
		Authorizer:             writer,
		InternalUserIDResolver: staticInternalUserID(42),
	})

	teachersVersion, hotTeachersVersion := teacherPublicCacheVersions(ctx, h)
	assert.NoError(t, h.RefreshTeacherPublicStats(ctx))
	assertTeacherPublicCachesBumped(t, ctx, h, teachersVersion, hotTeachersVersion)
	_, err = h.CleanupOldLogs(ctx)
	assert.NoError(t, err)

	jobCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	launches := make(chan string, 8)
	start := func(name string, run func(context.Context)) {
		launches <- name
		go run(jobCtx)
	}
	h.StartBackgroundJobs(jobCtx, start)
	svc.StartBackgroundJobs(jobCtx, start)
	requireReviewBackgroundLaunches(t, launches,
		"review fga sync worker",
		"review fga sync reconciliation",
		"review notification worker",
		"review teacher public stats projection",
		"review fga sync worker",
		"review fga sync reconciliation",
		"review notification worker",
	)
}

func TestReviewFGASyncProcessBatchAndHelpers(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	writer := &recordingReviewFGAWriter{}
	svc := NewService(fixture.DB, repo, noopReviewSender2{}, writer, fakeAccessReader{})
	ctx := context.Background()

	err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReviewRelations, reviewRelationsSyncKey("review-sync-1"), []byte(`{"reviewID":"review-sync-1","authorUserID":"user-1","schoolID":4111010006}`)); err != nil {
			return err
		}
		return repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReportRelations, reportRelationsSyncKey("report-sync-1"), []byte(`{"reportID":"report-sync-1","schoolID":4111010006}`))
	})
	require.NoError(t, err)

	require.NoError(t, svc.processFGASyncBatch(ctx))
	assert.Contains(t, writer.reviewIDs, "review-sync-1")
	assert.Contains(t, writer.reportIDs, "report-sync-1")
	assert.Contains(t, truncateFGASyncError(fmt.Errorf("%s", strings.Repeat("x", 300))), "xxx")
}

func TestStartBackgroundJobsRequiresStarter(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopReviewSender2{}, &recordingReviewFGAWriter{}, fakeAccessReader{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Panics(t, func() {
		svc.StartBackgroundJobs(ctx, nil)
	})
}

func requireReviewBackgroundLaunches(t *testing.T, launches <-chan string, want ...string) {
	t.Helper()
	for _, expected := range want {
		select {
		case name := <-launches:
			assert.Equal(t, expected, name)
		case <-time.After(time.Second):
			t.Fatalf("expected background job %q to launch", expected)
		}
	}
}

func TestReviewServiceStartsDurableNotificationWorker(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopReviewSender2{}, &recordingReviewFGAWriter{}, fakeAccessReader{})

	launches := make(chan string, 3)
	start := func(name string, _ func(context.Context)) {
		launches <- name
	}
	svc.StartBackgroundJobs(context.Background(), start)

	requireReviewBackgroundLaunches(
		t,
		launches,
		"review fga sync worker",
		"review fga sync reconciliation",
		"review notification worker",
	)
}
