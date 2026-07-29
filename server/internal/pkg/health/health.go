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

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

// defaultCheckTimeout 健康检查默认超时时间
const defaultCheckTimeout = 3 * time.Second

// BuildInfo 构建信息
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
}

// Handler 健康检查处理器
type Handler struct {
	pgPool       *pgxpool.Pool
	redisClient  *redis.Client
	buildInfo    BuildInfo
	startTime    time.Time
	isProduction bool
	checkTimeout time.Duration
}

// NewHandler 创建健康检查处理器
// checkTimeout 为健康检查探测超时，传 0 使用默认值（3 秒）
func NewHandler(pgPool *pgxpool.Pool, redisClient *redis.Client, buildInfo BuildInfo, isProduction bool, checkTimeout time.Duration) *Handler {
	if checkTimeout <= 0 {
		checkTimeout = defaultCheckTimeout
	}
	return &Handler{
		pgPool:       pgPool,
		redisClient:  redisClient,
		buildInfo:    buildInfo,
		startTime:    time.Now(),
		isProduction: isProduction,
		checkTimeout: checkTimeout,
	}
}

// RegisterRoutes 注册健康检查路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	health := r.Group("/health")
	{
		health.GET("/live", h.Liveness)
		health.GET("/ready", h.Readiness)
	}
}

// Liveness 存活探针 - 检查应用是否运行
// Kubernetes 使用此端点判断是否需要重启容器
func (h *Handler) Liveness(c *gin.Context) {
	response.Success(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// maxHealthyGoroutines goroutine 数量阈值，超过则标记为 degraded
const maxHealthyGoroutines = 10000

// Readiness 就绪探针 - 检查应用是否可以接收流量
// Kubernetes 使用此端点判断是否将流量路由到此实例
func (h *Handler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.checkTimeout)
	defer cancel()

	checks := make(map[string]CheckResult)
	var mu sync.Mutex

	// 使用子 context，确保探针返回后后台检查协程能及时退出
	checkCtx, checkCancel := context.WithCancel(ctx)
	defer checkCancel()

	var wg sync.WaitGroup
	// 并行检查各依赖
	wg.Add(2)
	go func() {
		defer wg.Done()
		// 捕获探针协程 panic，避免健康检查整体失效
		defer func() {
			if r := recover(); r != nil {
				logger.L().Error("health check: postgres goroutine panicked", zap.Any("panic", r))
				mu.Lock()
				checks["postgres"] = CheckResult{Status: "unhealthy", Error: "internal panic"}
				mu.Unlock()
			}
		}()
		result := h.checkPostgres(checkCtx)
		mu.Lock()
		checks["postgres"] = result
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		// 捕获探针协程 panic，避免健康检查整体失效
		defer func() {
			if r := recover(); r != nil {
				logger.L().Error("health check: redis goroutine panicked", zap.Any("panic", r))
				mu.Lock()
				checks["redis"] = CheckResult{Status: "unhealthy", Error: "internal panic"}
				mu.Unlock()
			}
		}()
		result := h.checkRedis(checkCtx)
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

	// goroutine 数量超过阈值时标记为 degraded
	if numGoroutines := runtime.NumGoroutine(); numGoroutines > maxHealthyGoroutines {
		logger.L().Warn("health check: goroutine count exceeds threshold",
			zap.Int("goroutines", numGoroutines),
			zap.Int("threshold", maxHealthyGoroutines),
		)
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	// 就绪检查失败时返回 503，需要保留 HTTP 状态码语义
	if httpStatus != http.StatusOK {
		response.Error(c, httpStatus, errs.ErrServiceUnavailable, status)
		return
	}

	// 生产环境返回精简信息，避免泄露内部细节
	if h.isProduction {
		response.Success(c, gin.H{
			"status":    status,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// 开发环境返回详细信息便于调试
	response.Success(c, gin.H{
		"status":    status,
		"checks":    checks,
		"info":      h.getSystemInfo(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// CheckResult 单项检查结果
type CheckResult struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
	Details any    `json:"details,omitempty"`
}

func (h *Handler) checkPostgres(ctx context.Context) CheckResult {
	if h.pgPool == nil {
		return CheckResult{
			Status: "unhealthy",
			Error:  "not configured",
		}
	}

	start := time.Now()
	err := h.pgPool.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		logger.L().Warn("health check: postgres ping failed", zap.Error(err))
		return CheckResult{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   "unavailable",
		}
	}

	// 获取连接池状态
	stats := h.pgPool.Stat()
	return CheckResult{
		Status:  "healthy",
		Latency: latency.String(),
		Details: map[string]any{
			"total_conns":    stats.TotalConns(),
			"idle_conns":     stats.IdleConns(),
			"acquired_conns": stats.AcquiredConns(),
		},
	}
}

func (h *Handler) checkRedis(ctx context.Context) CheckResult {
	if h.redisClient == nil {
		return CheckResult{
			Status: "unhealthy",
			Error:  "not configured",
		}
	}

	start := time.Now()
	pong, err := h.redisClient.Ping(ctx).Result()
	latency := time.Since(start)

	if err != nil {
		logger.L().Warn("health check: redis ping failed", zap.Error(err))
		return CheckResult{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   "unavailable",
		}
	}

	// 验证 PING 响应确实是 "PONG"
	if pong != "PONG" {
		logger.L().Warn("health check: redis ping returned unexpected response", zap.String("response", pong))
		return CheckResult{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   "unexpected ping response",
		}
	}

	// 获取连接池状态
	stats := h.redisClient.PoolStats()
	return CheckResult{
		Status:  "healthy",
		Latency: latency.String(),
		Details: map[string]any{
			"hits":        stats.Hits,
			"misses":      stats.Misses,
			"timeouts":    stats.Timeouts,
			"total_conns": stats.TotalConns,
			"idle_conns":  stats.IdleConns,
		},
	}
}

func (h *Handler) getSystemInfo() map[string]any {
	return map[string]any{
		"version":    h.buildInfo.Version,
		"git_commit": h.buildInfo.GitCommit,
		"build_time": h.buildInfo.BuildTime,
		"uptime":     time.Since(h.startTime).String(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
	}
}
