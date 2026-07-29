package review

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

type StepUpVerifier func(*gin.Context) bool

const (
	teacherPublicStatsRefreshTimeout      = 10 * time.Second
	teacherPublicCacheInvalidationTimeout = 2 * time.Second
	teacherPublicListCacheKey             = "review:teachers"
	teacherPublicHotCacheKey              = "review:hot_teachers"
	courseRatingStatsCacheKey             = "review:rating_stats:v2"
)

type AdminAuthorizers struct {
	Entry                gin.HandlerFunc
	DashboardView        gin.HandlerFunc
	LogsView             gin.HandlerFunc
	ReviewsManage        gin.HandlerFunc
	TeachersManage       gin.HandlerFunc
	SensitiveWordsManage gin.HandlerFunc
	StepUpVerified       StepUpVerifier
}

// Handler 评课社区处理器
type Handler struct {
	cache                  *cache.Helper
	courseCache            *cache.Helper
	service                *Service
	fga                    AuthorizationProvider
	internalUserIDResolver middleware.InternalUserIDResolver
	adminAuthorizers       AdminAuthorizers
	postLimiter            *middleware.RedisRateLimiter
	voteLimiter            *middleware.RedisRateLimiter
	reportLimiter          *middleware.RedisRateLimiter
	replyLimiter           *middleware.RedisRateLimiter
	writeLimiter           *middleware.RedisRateLimiter
	searchAnonLimiter      *middleware.RedisRateLimiter
	searchUserLimiter      *middleware.RedisRateLimiter
	batchAnonLimiter       *middleware.RedisRateLimiter
	batchUserLimiter       *middleware.RedisRateLimiter
}

// StartBackgroundJobs 启动评课后台任务（如 FGA 同步队列）。
func (h *Handler) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	h.service.StartBackgroundJobs(ctx, start)
}

// RefreshTeacherPublicStats 刷新公开教师统计物化视图，并失效依赖该视图的公开教师缓存。
func (h *Handler) RefreshTeacherPublicStats(ctx context.Context) error {
	if err := h.service.RefreshTeacherPublicStats(ctx); err != nil {
		return err
	}
	h.invalidateTeacherPublicCachesDetached(ctx, logger.FromContext(ctx))
	return nil
}

// RegisterRoutes 注册评课社区路由
func (h *Handler) RegisterRoutes(
	r *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
	optionalAuthMiddleware gin.HandlerFunc,
	adminMiddlewares ...gin.HandlerFunc,
) {
	// 评分维度配置
	r.GET("/rating-dimensions", h.GetRatingDimensions)

	// 课程评分统计
	r.GET("/courses/:courseID/rating-stats", h.GetCourseRatingStats)
	r.GET("/courses/:courseID/rating-trend", h.GetRatingTrend)

	// 课程教师列表
	r.GET("/courses/:courseID/teachers", h.GetCourseTeachers)

	// 测评（使用 optionalAuth：已登录用户看完整内容，未登录用户看脱敏内容）
	r.GET("/courses/:courseID/reviews", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourseReviews)
	r.GET("/reviews/latest", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetLatestReviews)
	r.GET("/reviews/search",
		optionalAuthMiddleware,
		middleware.RequireHealthyOptionalAuth(),
		middleware.ProgressiveEndpointRateLimitMiddleware(h.searchAnonLimiter, h.searchUserLimiter, "review-search"),
		h.SearchReviews,
	)
	r.GET("/reviews/batch",
		optionalAuthMiddleware,
		middleware.RequireHealthyOptionalAuth(),
		middleware.ProgressiveEndpointRateLimitMiddleware(h.batchAnonLimiter, h.batchUserLimiter, "review-batch"),
		h.GetBatchCourseReviews,
	)
	r.POST("/reviews", authMiddleware, middleware.EndpointRateLimitMiddleware(h.postLimiter, "post-review"), h.PostReview)
	r.PUT("/reviews/:reviewID", authMiddleware, middleware.EndpointRateLimitMiddleware(h.writeLimiter, "update-review"), h.UpdateReview)
	r.DELETE("/reviews/:reviewID", authMiddleware, middleware.EndpointRateLimitMiddleware(h.writeLimiter, "delete-review"), h.DeleteReview)
	r.POST("/reviews/:reviewID/votes", authMiddleware, middleware.EndpointRateLimitMiddleware(h.voteLimiter, "vote"), h.VoteReview)
	r.POST("/reviews/:reviewID/reports", authMiddleware, middleware.EndpointRateLimitMiddleware(h.reportLimiter, "report"), h.ReportReview)

	// 回复
	r.GET("/reviews/:reviewID/replies", h.GetReplies)
	r.POST("/reviews/:reviewID/replies", authMiddleware, middleware.EndpointRateLimitMiddleware(h.replyLimiter, "reply"), h.CreateReply)
	r.DELETE("/replies/:replyID", authMiddleware, middleware.EndpointRateLimitMiddleware(h.writeLimiter, "delete-reply"), h.DeleteReply)

	// 评课统计
	r.GET("/stats", h.GetStats)
	r.GET("/rankings/hot", h.GetHotCourses)

	// 教师公开列表（必须在 /teachers/:teacherID 之前注册，避免路由冲突）
	r.GET("/teachers", h.ListTeachers)
	r.GET("/teachers/hot", h.ListHotTeachers)

	r.GET("/teachers/:teacherID/stats", h.GetTeacherRatingStats)

	// 内容检查（需要认证，防止敏感词列表被探测）
	r.POST("/content/check", authMiddleware, h.CheckContent)

	// 用户中心（需要认证）
	userGroup := r.Group("/user")
	userGroup.Use(authMiddleware)
	{
		userGroup.GET("/reviews", h.GetUserReviews)
		userGroup.GET("/votes", h.GetUserVotes)
		userGroup.GET("/favorites", h.GetUserFavorites)
	}

	// 草稿（需要认证）
	r.POST("/drafts", authMiddleware, h.SaveDraft)
	r.GET("/drafts", authMiddleware, h.GetDraft)
	r.DELETE("/drafts", authMiddleware, h.DeleteDraft)

	// 课程收藏（需要认证）
	r.GET("/courses/:courseID/favorites", authMiddleware, h.GetFavoriteStatus)
	r.POST("/courses/:courseID/favorites", authMiddleware, h.AddFavorite)
	r.DELETE("/courses/:courseID/favorites", authMiddleware, h.RemoveFavorite)

	// 管理员路由组
	admin := r.Group("/admin")
	admin.GET(
		"/stats",
		httputil.RouteHandlers(
			h.GetAdminStats,
			authMiddleware,
			h.adminAuthorizers.Entry,
			h.adminAuthorizers.DashboardView,
		)...,
	)

	adminRouteMiddlewares := append([]gin.HandlerFunc{authMiddleware}, adminMiddlewares...)
	adminRouteMiddlewares = httputil.AppendRouteMiddlewares(adminRouteMiddlewares, h.adminAuthorizers.Entry)
	admin.Use(adminRouteMiddlewares...)
	{
		admin.GET("/reports", requireModerationRole(), h.ListReports)
		admin.PUT("/reports/:reportID", requireModerationRole(), h.ProcessReport)
		admin.GET("/reviews", requireModerationRole(), h.ListAllReviews)
		admin.PUT("/reviews/:reviewID", requireModerationRole(), h.AdminUpdateReview)
		admin.POST("/reviews/:reviewID/edit", requireSchoolAdminRole(), h.AdminEditReviewContent)
		admin.PATCH("/reviews/batch", requireModerationRole(), h.BatchUpdateReviews)
		admin.GET("/logs", httputil.RouteHandlers(h.GetOperationLogs, h.adminAuthorizers.LogsView)...)
		admin.GET("/export", httputil.RouteHandlers(h.ExportReviews, h.adminAuthorizers.ReviewsManage)...)

		admin.GET("/teachers", httputil.RouteHandlers(h.ListAdminTeachers, h.adminAuthorizers.TeachersManage)...)
		admin.POST("/teachers", httputil.RouteHandlers(h.CreateTeacher, h.adminAuthorizers.TeachersManage)...)
		admin.PUT("/teachers/:teacherID", httputil.RouteHandlers(h.UpdateTeacher, h.adminAuthorizers.TeachersManage)...)
		admin.DELETE("/teachers/:teacherID", httputil.RouteHandlers(h.DeleteTeacher, h.adminAuthorizers.TeachersManage)...)

		admin.GET("/sensitive-words", httputil.RouteHandlers(h.ListSensitiveWords, h.adminAuthorizers.SensitiveWordsManage)...)
		admin.POST("/sensitive-words", httputil.RouteHandlers(h.CreateSensitiveWord, h.adminAuthorizers.SensitiveWordsManage)...)
		admin.PUT("/sensitive-words/:sensitiveWordID", httputil.RouteHandlers(h.UpdateSensitiveWord, h.adminAuthorizers.SensitiveWordsManage)...)
		admin.DELETE("/sensitive-words/:sensitiveWordID", httputil.RouteHandlers(h.DeleteSensitiveWord, h.adminAuthorizers.SensitiveWordsManage)...)

		admin.GET("/content-flags", requireModerationRole(), h.ListFlaggedReviews)
		admin.PUT("/content-flags/:reviewID/clear", requireModerationRole(), h.ClearContentFlag)
	}
}

// CleanupOldLogs 清理过期操作日志（保留 90 天）
// 供上层 course.Handler 后台定时任务调用
func (h *Handler) CleanupOldLogs(ctx context.Context) (int64, error) {
	const retentionDays = 90
	return h.service.CleanupOldOperationLogs(ctx, retentionDays)
}

// logAdminOp 记录管理员操作日志的辅助函数，提取 gin context 中的公共字段
func (h *Handler) logAdminOp(c *gin.Context, action, resourceType, resourceID string, oldValue, newValue any) {
	event := audit.EventFromGin(c, audit.Event{})
	if err := h.service.LogOperation(c.Request.Context(), LogOperationParams{
		AdminUserID:   event.UserID,
		AdminUsername: event.Username,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		OldValue:      oldValue,
		NewValue:      newValue,
		RequestID:     event.RequestID,
		TraceID:       event.TraceID,
		IPAddress:     event.IP,
		UserAgent:     event.UserAgent,
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}
}

// invalidateReviewCaches 失效评论相关缓存
// courseID > 0 时精确失效该课程的缓存（review:course:{courseID}），否则失效全局 review:course 前缀
// extraKeys 为额外需要失效的缓存前缀
// 缓存失效失败时记录详细日志 + prometheus 计数器，但不阻塞业务流程
func (h *Handler) invalidateReviewCaches(c *gin.Context, courseID int64, extraKeys ...string) {
	ctx := c.Request.Context()
	l := logger.FromGin(c)

	// 失效课程缓存：按课程粒度或全局
	if courseID > 0 {
		key := "review:course:" + strconv.FormatInt(courseID, 10)
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			metrics.ObserveCacheInvalidationFailure(cache.NamespaceReview)
			l.Warn("failed to invalidate course cache",
				zap.String("cache_key", key),
				zap.Int64("course_id", courseID),
				zap.Error(err))
		}
	} else {
		// 无法确定课程 ID 时，失效全局课程缓存版本号（避免 SCAN）
		if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
			metrics.ObserveCacheInvalidationFailure(cache.NamespaceReview)
			l.Warn("failed to invalidate global course cache version",
				zap.String("cache_key", "review:course"),
				zap.Error(err))
		}
	}

	// 始终失效 latest 缓存（跨课程聚合）
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		metrics.ObserveCacheInvalidationFailure(cache.NamespaceReview)
		l.Warn("failed to invalidate cache",
			zap.String("cache_key", "review:latest"),
			zap.Error(err))
	}

	// 失效额外指定的缓存前缀
	for _, key := range extraKeys {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			metrics.ObserveCacheInvalidationFailure(cache.NamespaceReview)
			l.Warn("failed to invalidate cache",
				zap.String("cache_key", key),
				zap.Error(err))
		}
	}
}

func (h *Handler) invalidateCachePrefixes(c *gin.Context, keys ...string) {
	ctx := c.Request.Context()
	l := logger.FromGin(c)
	for _, key := range keys {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			metrics.ObserveCacheInvalidationFailure(cache.NamespaceReview)
			l.Warn("failed to invalidate cache",
				zap.String("cache_key", key),
				zap.Error(err))
		}
	}
}

func (h *Handler) invalidateReviewAggregateCaches(c *gin.Context) {
	h.invalidateCachePrefixes(c,
		"review:stats",
		courseRatingStatsCacheKey,
		"review:rating_trend",
		"review:hot",
		"review:course_teachers",
		"review:teacher_stats",
		"review:admin:stats",
	)
}

func (h *Handler) refreshTeacherPublicStatsAndInvalidateCaches(c *gin.Context) {
	ctx, cancel := detachedRefreshContext(c.Request.Context(), teacherPublicStatsRefreshTimeout)
	defer cancel()

	l := logger.FromGin(c)
	if err := h.RefreshTeacherPublicStats(ctx); err != nil {
		l.Warn("failed to refresh teacher public stats", zap.Error(err))
	}
}

func (h *Handler) invalidateTeacherPublicCaches(ctx context.Context, l *zap.Logger) {
	for _, key := range []string{teacherPublicListCacheKey, teacherPublicHotCacheKey} {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			metrics.ObserveCacheInvalidationFailure(cache.NamespaceReview)
			l.Warn("failed to invalidate teacher public cache",
				zap.String("cache_key", key),
				zap.Error(err))
		}
	}
}

func (h *Handler) invalidateTeacherPublicCachesDetached(parent context.Context, l *zap.Logger) {
	ctx, cancel := detachedRefreshContext(parent, teacherPublicCacheInvalidationTimeout)
	defer cancel()
	h.invalidateTeacherPublicCaches(ctx, l)
}

func (h *Handler) invalidateCourseReviewCountCaches(c *gin.Context) {
	ctx := c.Request.Context()
	l := logger.FromGin(c)
	courseCache := h.courseCache
	if courseCache == nil {
		courseCache = h.cache
	}
	for _, key := range []string{"course:courses", "course:course"} {
		if err := courseCache.InvalidateByVersion(ctx, key); err != nil {
			metrics.ObserveCacheInvalidationFailure(cache.NamespaceCourse)
			l.Warn("failed to invalidate course review count cache",
				zap.String("cache_key", key),
				zap.Error(err))
		}
	}
}
