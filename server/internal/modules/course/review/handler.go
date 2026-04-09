package review

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// Handler 评课社区处理器
type Handler struct {
	db             *db.DB
	cache          *cache.Helper
	service        *Service
	userRepo       *user.Repository
	fga            *fga.Client
	postLimiter    *middleware.RedisRateLimiter
	voteLimiter    *middleware.RedisRateLimiter
	reportLimiter  *middleware.RedisRateLimiter
	replyLimiter   *middleware.RedisRateLimiter
	writeLimiter   *middleware.RedisRateLimiter
	accessPolicySF singleflight.Group
}

// NewHandler 创建处理器
func NewHandler(database *db.DB, cacheHelper *cache.Helper, rdb *redis.Client, rlCfg config.ReviewRateLimitConfig, fgaClient *fga.Client, notifSender notification.Sender) *Handler {
	repo := NewRepository(database)
	svc := NewService(database, repo, notifSender)
	return &Handler{
		db:            database,
		cache:         cacheHelper,
		service:       svc,
		userRepo:      user.NewRepository(database),
		fga:           fgaClient,
		postLimiter:   middleware.NewRedisRateLimiter(rdb, rlCfg.PostLimit, time.Minute),
		voteLimiter:   middleware.NewRedisRateLimiter(rdb, rlCfg.VoteLimit, time.Minute),
		reportLimiter: middleware.NewRedisRateLimiter(rdb, rlCfg.ReportLimit, time.Minute),
		replyLimiter:  middleware.NewRedisRateLimiter(rdb, rlCfg.ReplyLimit, time.Minute),
		writeLimiter:  middleware.NewRedisRateLimiter(rdb, rlCfg.WriteLimit, time.Minute),
	}
}

// RegisterRoutes 注册评课社区路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware, optionalAuthMiddleware gin.HandlerFunc) {
	// 评分维度配置
	r.GET("/rating-dimensions", h.GetRatingDimensions)

	// 课程评分统计
	r.GET("/courses/:courseID/rating-stats", h.GetCourseRatingStats)
	r.GET("/courses/:courseID/rating-trend", h.GetRatingTrend)

	// 课程教师列表
	r.GET("/courses/:courseID/teachers", h.GetCourseTeachers)

	// 测评（使用 optionalAuth：已登录用户看完整内容，未登录用户看脱敏内容）
	r.GET("/courses/:courseID/reviews", optionalAuthMiddleware, h.GetCourseReviews)
	r.GET("/reviews/latest", optionalAuthMiddleware, h.GetLatestReviews)
	r.GET("/reviews/search", optionalAuthMiddleware, h.SearchReviews)
	r.GET("/reviews/batch", optionalAuthMiddleware, h.GetBatchCourseReviews)
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
		userGroup.GET("/notifications", h.GetNotifications)
		userGroup.GET("/notifications/unread-count", h.GetUnreadCount)
		userGroup.PUT("/notifications/:notificationID/read", h.MarkNotificationRead)
		userGroup.PUT("/notifications/read-all", h.MarkAllNotificationsRead)
	}

	// 草稿（需要认证）
	r.POST("/drafts", authMiddleware, h.SaveDraft)
	r.GET("/drafts/:courseID", authMiddleware, h.GetDraft)
	r.DELETE("/drafts/:courseID", authMiddleware, h.DeleteDraft)

	// 课程收藏（需要认证）
	r.GET("/courses/:courseID/favorites", authMiddleware, h.GetFavoriteStatus)
	r.POST("/courses/:courseID/favorites", authMiddleware, h.AddFavorite)
	r.DELETE("/courses/:courseID/favorites", authMiddleware, h.RemoveFavorite)

	// 管理员路由组
	admin := r.Group("/admin")
	admin.Use(authMiddleware, rbac.RequireAnyCapability(capability.AdminEntryCapabilities...))
	{
		admin.GET("/reports", rbac.RequireCapability(capability.AdminReportsManage), h.ListReports)
		admin.PUT("/reports/:reportID", rbac.RequireCapability(capability.AdminReportsManage), h.ProcessReport)
		admin.GET("/reviews", rbac.RequireCapability(capability.AdminReviewsManage), h.ListAllReviews)
		admin.PUT("/reviews/:reviewID", rbac.RequireCapability(capability.AdminReviewsManage), h.AdminUpdateReview)
		admin.POST("/reviews/:reviewID/edit", rbac.RequireCapability(capability.AdminReviewsManage), h.AdminEditReviewContent)
		admin.POST("/reviews/batch", rbac.RequireCapability(capability.AdminReviewsManage), h.BatchUpdateReviews)
		admin.GET("/stats", rbac.RequireCapability(capability.AdminDashboardView), h.GetAdminStats)
		admin.GET("/logs", rbac.RequireCapability(capability.AdminLogsView), h.GetOperationLogs)
		admin.GET("/export", rbac.RequireCapability(capability.AdminReviewsManage), h.ExportReviews)

		admin.GET("/teachers", rbac.RequireCapability(capability.AdminTeachersManage), h.ListAdminTeachers)
		admin.POST("/teachers", rbac.RequireCapability(capability.AdminTeachersManage), h.CreateTeacher)
		admin.PUT("/teachers/:teacherID", rbac.RequireCapability(capability.AdminTeachersManage), h.UpdateTeacher)
		admin.DELETE("/teachers/:teacherID", rbac.RequireCapability(capability.AdminTeachersManage), h.DeleteTeacher)

		admin.GET("/sensitive-words", rbac.RequireCapability(capability.AdminSensitiveWordsManage), h.ListSensitiveWords)
		admin.POST("/sensitive-words", rbac.RequireCapability(capability.AdminSensitiveWordsManage), h.CreateSensitiveWord)
		admin.PUT("/sensitive-words/:sensitiveWordID", rbac.RequireCapability(capability.AdminSensitiveWordsManage), h.UpdateSensitiveWord)
		admin.DELETE("/sensitive-words/:sensitiveWordID", rbac.RequireCapability(capability.AdminSensitiveWordsManage), h.DeleteSensitiveWord)

		admin.GET("/content-flags", rbac.RequireCapability(capability.AdminReviewsManage), h.ListFlaggedReviews)
		admin.PUT("/content-flags/:reviewID/clear", rbac.RequireCapability(capability.AdminReviewsManage), h.ClearContentFlag)
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
	if err := h.service.LogOperation(c.Request.Context(), LogOperationParams{
		AdminUserID:   middleware.GetUserID(c),
		AdminUsername: middleware.GetUsername(c),
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		OldValue:      oldValue,
		NewValue:      newValue,
		IPAddress:     c.ClientIP(),
		UserAgent:     httputil.TruncateUserAgent(c.GetHeader("User-Agent")),
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
			metrics.CacheInvalidationFailuresTotal.WithLabelValues("review:course").Inc()
			l.Warn("failed to invalidate course cache",
				zap.String("cache_key", key),
				zap.Int64("course_id", courseID),
				zap.Error(err))
		}
	} else {
		// 无法确定课程 ID 时，失效全局课程缓存版本号（避免 SCAN）
		if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
			metrics.CacheInvalidationFailuresTotal.WithLabelValues("review:course").Inc()
			l.Warn("failed to invalidate global course cache version",
				zap.String("cache_key", "review:course"),
				zap.Error(err))
		}
	}

	// 始终失效 latest 缓存（跨课程聚合）
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		metrics.CacheInvalidationFailuresTotal.WithLabelValues("review:latest").Inc()
		l.Warn("failed to invalidate cache",
			zap.String("cache_key", "review:latest"),
			zap.Error(err))
	}

	// 失效额外指定的缓存前缀
	for _, key := range extraKeys {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			metrics.CacheInvalidationFailuresTotal.WithLabelValues(key).Inc()
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
			metrics.CacheInvalidationFailuresTotal.WithLabelValues(key).Inc()
			l.Warn("failed to invalidate cache",
				zap.String("cache_key", key),
				zap.Error(err))
		}
	}
}

func (h *Handler) invalidateReviewAggregateCaches(c *gin.Context) {
	h.invalidateCachePrefixes(c,
		"review:stats",
		"review:rating_stats",
		"review:rating_trend",
		"review:hot",
		"review:course_teachers",
		"review:teacher_stats",
		"review:admin:stats",
	)
}
