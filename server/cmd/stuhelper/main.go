package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化 Redis 客户端
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// 初始化 Token 服务
	tokenService := token.NewService(
		redisClient.GetClient(),
		cfg.Token.AccessTokenTTL,
		cfg.Token.RefreshTokenTTL,
	)

	// 根据环境设置 Gin 模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由
	r := gin.Default()

	// 配置 CORS（验证通配符安全性）
	corsOrigins := cfg.App.CORSOrigins
	for _, origin := range corsOrigins {
		if strings.TrimSpace(origin) == "*" {
			log.Fatal("CORS configuration error: wildcard '*' is not allowed when AllowCredentials is true")
		}
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 注册 API 路由
	api := r.Group("/api")
	{
		// 注册认证模块路由
		authHandler := auth.NewHandler(cfg, tokenService)
		authHandler.RegisterRoutes(api)
	}

	// 启动服务器（支持优雅关闭）
	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Printf("Server starting on :%s", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 优雅关闭，等待最多 5 秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
