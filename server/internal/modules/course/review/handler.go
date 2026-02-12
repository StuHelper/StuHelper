package review

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
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
func NewHandler(database *db.DB, rdb *redis.Client, ssoClient *sso.Client) *Handler {
	repo := NewRepository(database)
	svc := NewService(database, repo)
	return &Handler{
		db:            database,
		cache:         cache.NewHelper(rdb),
		service:       svc,
		ssoClient:     ssoClient,
		postLimiter:   middleware.NewRedisRateLimiter(rdb, 5, time.Minute),   // 每分钟最多发布5条评论
		voteLimiter:   middleware.NewRedisRateLimiter(rdb, 30, time.Minute),  // 每分钟最多投票30次
		reportLimiter: middleware.NewRedisRateLimiter(rdb, 10, time.Minute),  // 每分钟最多举报10次
		replyLimiter:  middleware.NewRedisRateLimiter(rdb, 10, time.Minute),  // 每分钟最多回复10次
		writeLimiter:  middleware.NewRedisRateLimiter(rdb, 10, time.Minute),  // 每分钟最多更新/删除10次
	}
}

// RegisterRoutes 注册评课社区路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// 评分维度配置
	r.GET("/rating-dimensions", h.GetRatingDimensions)

	// 课程评分统计
	r.GET("/courses/:id/rating-stats", h.GetCourseRatingStats)
	r.GET("/courses/:id/rating-trend", h.GetRatingTrend)

	// 测评
	r.GET("/courses/:id/reviews", h.GetCourseReviews)
	r.GET("/reviews/latest", h.GetLatestReviews)
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
	r.GET("/drafts/:courseId", authMiddleware, h.GetDraft)
	r.DELETE("/drafts/:courseId", authMiddleware, h.DeleteDraft)

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

// invalidateReviewCaches 失效评论相关缓存，keys 为额外需要失效的缓存前缀
func (h *Handler) invalidateReviewCaches(c *gin.Context, keys ...string) {
	ctx := c.Request.Context()
	l := logger.FromGin(c)
	// 始终失效 course 和 latest 缓存
	for _, key := range []string{"review:course", "review:latest"} {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			l.Warn("failed to invalidate cache", zap.String("key", key), zap.Error(err))
		}
	}
	// 失效额外指定的缓存前缀
	for _, key := range keys {
		if err := h.cache.InvalidateByVersion(ctx, key); err != nil {
			l.Warn("failed to invalidate cache", zap.String("key", key), zap.Error(err))
		}
	}
}
