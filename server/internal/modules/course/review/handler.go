package review

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
)

// Handler 评课社区处理器
type Handler struct {
	db            *db.DB
	cache         *cache.Helper
	service       *Service
	ssoClient     *sso.Client
	postLimiter   *middleware.RedisRateLimiter // 发布评论限流
	voteLimiter   *middleware.RedisRateLimiter // 投票限流
	reportLimiter *middleware.RedisRateLimiter // 举报限流
	replyLimiter  *middleware.RedisRateLimiter // 回复限流
	writeLimiter  *middleware.RedisRateLimiter // 更新/删除限流
}

// NewHandler 创建处理器
func NewHandler(database *db.DB, rdb *redis.Client, ssoClient *sso.Client, rlCfg config.ReviewRateLimitConfig) *Handler {
	repo := NewRepository(database)
	svc := NewService(database, repo)
	return &Handler{
		db:            database,
		cache:         cache.NewHelper(rdb),
		service:       svc,
		ssoClient:     ssoClient,
		postLimiter:   middleware.NewRedisRateLimiter(rdb, rlCfg.PostLimit, time.Minute),
		voteLimiter:   middleware.NewRedisRateLimiter(rdb, rlCfg.VoteLimit, time.Minute),
		reportLimiter: middleware.NewRedisRateLimiter(rdb, rlCfg.ReportLimit, time.Minute),
		replyLimiter:  middleware.NewRedisRateLimiter(rdb, rlCfg.ReplyLimit, time.Minute),
		writeLimiter:  middleware.NewRedisRateLimiter(rdb, rlCfg.WriteLimit, time.Minute),
	}
}

// RegisterRoutes 注册评课社区路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// 评分维度配置
	r.GET("/rating-dimensions", h.GetRatingDimensions)

	// 课程评分统计
	r.GET("/courses/:id/rating-stats", h.GetCourseRatingStats)
	r.GET("/courses/:id/rating-trend", h.GetRatingTrend)

	// 课程教师列表
	r.GET("/courses/:id/teachers", h.GetCourseTeachers)

	// 测评
	r.GET("/courses/:id/reviews", h.GetCourseReviews)
	r.GET("/reviews/latest", h.GetLatestReviews)
	r.GET("/reviews/batch", h.GetBatchCourseReviews)
	r.POST("/reviews", authMiddleware, middleware.EndpointRateLimitMiddleware(h.postLimiter, "post-review"), h.PostReview)
	r.PUT("/reviews/:id", authMiddleware, middleware.EndpointRateLimitMiddleware(h.writeLimiter, "update-review"), h.UpdateReview)
	r.DELETE("/reviews/:id", authMiddleware, middleware.EndpointRateLimitMiddleware(h.writeLimiter, "delete-review"), h.DeleteReview)
	r.POST("/reviews/:id/vote", authMiddleware, middleware.EndpointRateLimitMiddleware(h.voteLimiter, "vote"), h.VoteReview)
	r.POST("/reviews/:id/report", authMiddleware, middleware.EndpointRateLimitMiddleware(h.reportLimiter, "report"), h.ReportReview)

	// 回复
	r.GET("/reviews/:id/replies", h.GetReplies)
	r.POST("/reviews/:id/replies", authMiddleware, middleware.EndpointRateLimitMiddleware(h.replyLimiter, "reply"), h.CreateReply)
	r.DELETE("/replies/:id", authMiddleware, middleware.EndpointRateLimitMiddleware(h.writeLimiter, "delete-reply"), h.DeleteReply)

	// 评课统计
	r.GET("/stats", h.GetStats)
	r.GET("/rankings/hot", h.GetHotCourses)
	r.GET("/teachers/:id/stats", h.GetTeacherRatingStats)

	// 内容检查（需要认证，防止敏感词列表被探测）
	r.POST("/content/check", authMiddleware, h.CheckContent)

	// 用户中心（需要认证）
	user := r.Group("/user")
	user.Use(authMiddleware)
	{
		user.GET("/reviews", h.GetUserReviews)
		user.GET("/votes", h.GetUserVotes)
		user.GET("/favorites", h.GetUserFavorites)
	}

	// 通知（需要认证）
	r.GET("/notifications", authMiddleware, h.GetNotifications)
	r.GET("/notifications/unread-count", authMiddleware, h.GetUnreadCount)
	r.PUT("/notifications/:id/read", authMiddleware, h.MarkNotificationRead)
	r.PUT("/notifications/read-all", authMiddleware, h.MarkAllNotificationsRead)

	// 草稿（需要认证）
	r.POST("/drafts", authMiddleware, h.SaveDraft)
	r.GET("/drafts/:courseID", authMiddleware, h.GetDraft)
	r.DELETE("/drafts/:courseID", authMiddleware, h.DeleteDraft)

	// 课程收藏（需要认证）
	r.POST("/courses/:id/favorite", authMiddleware, h.AddFavorite)
	r.DELETE("/courses/:id/favorite", authMiddleware, h.RemoveFavorite)

	// 管理员路由组
	admin := r.Group("/admin")
	admin.Use(authMiddleware, middleware.RequireAdmin(h.ssoClient))
	{
		admin.GET("/reports", h.ListReports)
		admin.PUT("/reports/:id", h.ProcessReport)
		admin.GET("/reviews", h.ListAllReviews)
		admin.PUT("/reviews/:id", h.AdminUpdateReview)
		admin.POST("/reviews/batch", h.BatchUpdateReviews)
		admin.GET("/stats", h.GetAdminStats)
		admin.GET("/logs", h.GetOperationLogs)
		admin.GET("/export", h.ExportReviews)
	}
}

// CleanupOldLogs 清理过期操作日志（保留 90 天）
// 供上层 course.Handler 后台定时任务调用
func (h *Handler) CleanupOldLogs(ctx context.Context) (int64, error) {
	const retentionDays = 90
	return h.service.CleanupOldOperationLogs(ctx, retentionDays)
}

// invalidateReviewCaches 失效评论相关缓存
// courseID > 0 时精确失效该课程的缓存（review:course:{courseID}），否则失效全局 review:course 前缀
// extraKeys 为额外需要失效的缓存前缀
// M-48: 缓存失效失败时记录详细日志（包含具体 key），但不阻塞业务流程
func (h *Handler) invalidateReviewCaches(c *gin.Context, courseID int64, extraKeys ...string) {
	ctx := c.Request.Context()
	l := logger.FromGin(c)

	// 失效课程缓存：按课程粒度或全局
	if courseID > 0 {
		key := "review:course:" + strconv.FormatInt(courseID, 10)
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			l.Warn("failed to invalidate course cache",
				zap.String("cache_key", key),
				zap.Int64("course_id", courseID),
				zap.Error(err))
		}
	} else {
		// 无法确定课程 ID 时，使用 SCAN 前缀匹配失效所有课程缓存
		if err := h.cache.Invalidate(ctx, "review:course:"); err != nil {
			l.Warn("failed to invalidate all course caches",
				zap.String("cache_prefix", "review:course:"),
				zap.Error(err))
		}
	}

	// 始终失效 latest 缓存（跨课程聚合）
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		l.Warn("failed to invalidate cache",
			zap.String("cache_key", "review:latest"),
			zap.Error(err))
	}

	// 失效额外指定的缓存前缀
	for _, key := range extraKeys {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			l.Warn("failed to invalidate cache",
				zap.String("cache_key", key),
				zap.Error(err))
		}
	}
}
