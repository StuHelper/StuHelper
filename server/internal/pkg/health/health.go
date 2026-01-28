package health

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// 健康检查超时时间
const checkTimeout = 2 * time.Second

// BuildInfo 构建信息
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
}

// Handler 健康检查处理器
type Handler struct {
	pgPool      *pgxpool.Pool
	redisClient *redis.Client
	buildInfo   BuildInfo
	startTime   time.Time
}

// NewHandler 创建健康检查处理器
func NewHandler(pgPool *pgxpool.Pool, redisClient *redis.Client, buildInfo BuildInfo) *Handler {
	return &Handler{
		pgPool:      pgPool,
		redisClient: redisClient,
		buildInfo:   buildInfo,
		startTime:   time.Now(),
	}
}

// RegisterRoutes 注册健康检查路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	health := r.Group("/health")
	{
		health.GET("/live", h.Liveness)
		health.GET("/ready", h.Readiness)
	}
	// 保留旧的 /health 端点以兼容
	r.GET("/health", h.Readiness)
}

// Liveness 存活探针 - 检查应用是否运行
// Kubernetes 使用此端点判断是否需要重启容器
func (h *Handler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Readiness 就绪探针 - 检查应用是否可以接收流量
// Kubernetes 使用此端点判断是否将流量路由到此实例
func (h *Handler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	checks := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并行检查各依赖
	wg.Add(2)
	go func() {
		defer wg.Done()
		result := h.checkPostgres(ctx)
		mu.Lock()
		checks["postgres"] = result
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		result := h.checkRedis(ctx)
		mu.Lock()
		checks["redis"] = result
		mu.Unlock()
	}()
	wg.Wait()

	// 判断整体状态
	status := "ok"
	httpStatus := http.StatusOK
	for _, check := range checks {
		if check.Status != "healthy" {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(httpStatus, gin.H{
		"status":    status,
		"checks":    checks,
		"info":      h.getSystemInfo(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// CheckResult 单项检查结果
type CheckResult struct {
	Status   string `json:"status"`
	Latency  string `json:"latency,omitempty"`
	Error    string `json:"error,omitempty"`
	Details  any    `json:"details,omitempty"`
}

func (h *Handler) checkPostgres(ctx context.Context) CheckResult {
	start := time.Now()
	err := h.pgPool.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	// 获取连接池状态
	stats := h.pgPool.Stat()
	return CheckResult{
		Status:  "healthy",
		Latency: latency.String(),
		Details: map[string]any{
			"total_conns":   stats.TotalConns(),
			"idle_conns":    stats.IdleConns(),
			"acquired_conns": stats.AcquiredConns(),
		},
	}
}

func (h *Handler) checkRedis(ctx context.Context) CheckResult {
	start := time.Now()
	err := h.redisClient.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	// 获取连接池状态
	stats := h.redisClient.PoolStats()
	return CheckResult{
		Status:  "healthy",
		Latency: latency.String(),
		Details: map[string]any{
			"hits":       stats.Hits,
			"misses":     stats.Misses,
			"timeouts":   stats.Timeouts,
			"total_conns": stats.TotalConns,
			"idle_conns":  stats.IdleConns,
		},
	}
}

func (h *Handler) getSystemInfo() map[string]any {
	return map[string]any{
		"version":     h.buildInfo.Version,
		"git_commit":  h.buildInfo.GitCommit,
		"build_time":  h.buildInfo.BuildTime,
		"uptime":      time.Since(h.startTime).String(),
		"go_version":  runtime.Version(),
		"goroutines":  runtime.NumGoroutine(),
	}
}
