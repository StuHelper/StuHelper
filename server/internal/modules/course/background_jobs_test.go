package course

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type noopCourseReviewSender struct{}

func (noopCourseReviewSender) Send(context.Context, notification.SendParams) error        { return nil }
func (noopCourseReviewSender) SendBatch(context.Context, []notification.SendParams) error { return nil }

type noopCourseReviewFGA struct{}

func (noopCourseReviewFGA) Check(context.Context, string, string, string) (bool, error) {
	return true, nil
}
func (noopCourseReviewFGA) WriteReviewRelations(context.Context, string, string, string) error {
	return nil
}
func (noopCourseReviewFGA) WriteReportRelations(context.Context, string, string) error {
	return nil
}

type noopCourseReviewAccessReader struct{}

func (noopCourseReviewAccessReader) ListReviewAccessSchoolConfigs(context.Context) ([]reviewaccess.SchoolConfig, error) {
	return []reviewaccess.SchoolConfig{}, nil
}
func (noopCourseReviewAccessReader) ListReviewAccessSystemConfigs(context.Context) ([]reviewaccess.SystemConfig, error) {
	return []reviewaccess.SystemConfig{}, nil
}
func (noopCourseReviewAccessReader) GetReviewAccessSubject(context.Context, string) (*reviewaccess.Subject, error) {
	return nil, nil
}

func newCourseModuleHandler(t *testing.T) *Handler {
	t.Helper()
	fixture := postgresfixture.Start(t)
	courseRepo := NewRepository(fixture.DB)
	courseSvc := NewService(courseRepo, zap.NewNop())
	reviewRepo := review.NewRepository(fixture.DB)
	reviewSvc := review.NewService(fixture.DB, reviewRepo, noopCourseReviewSender{}, noopCourseReviewFGA{}, noopCourseReviewAccessReader{})

	redisFixture := redisfixture.Start(t)

	reviewHandler := review.NewHandler(review.HandlerConfig{
		CacheHelper: cache.NewHelper(redisFixture.Client),
		Service:     reviewSvc,
		Redis:       redisFixture.Client,
		RateLimit:   config.ReviewRateLimitConfig{},
		Authorizer:  noopCourseReviewFGA{},
		InternalUserIDResolver: func(context.Context, string) (int64, error) {
			return 42, nil
		},
	})
	return NewHandler(cache.NewHelper(redisFixture.Client), courseSvc, reviewHandler)
}

func TestCourseBackgroundJobsAndConstructor(t *testing.T) {
	h := newCourseModuleHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NotPanics(t, func() { h.runLogCleanup(ctx) })
	require.NotPanics(t, func() { h.runTeacherPublicStatsRefresh(ctx) })
	require.NotPanics(t, func() {
		h.StartBackgroundJobs(ctx, func(_ string, run func(context.Context)) {
			go run(ctx)
		})
	})

	cancel()
	time.Sleep(20 * time.Millisecond)
}
