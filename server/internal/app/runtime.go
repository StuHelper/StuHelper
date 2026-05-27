package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	gozap "go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/observability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	redisclient "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// BuildInfo 描述当前二进制的构建元数据。
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
}

// Runtime 封装 stuhelper 进程的组合根和生命周期。
type Runtime struct {
	build BuildInfo

	cfg          *config.Config
	isProduction bool

	cleanups []func()
	bgWg     sync.WaitGroup
	bgCancel context.CancelFunc

	pgPool       *pgxpool.Pool
	redisClient  *redisclient.Client
	database     *db.DB
	fgaClient    *fga.Client
	tokenService *token.Service
	oidcClient   *oidc.Client
}

// Run 启动应用。
func Run(version, gitCommit, buildTime string) error {
	rt := &Runtime{
		build: BuildInfo{
			Version:   version,
			GitCommit: gitCommit,
			BuildTime: buildTime,
		},
	}
	return rt.run()
}

func (rt *Runtime) run() error {
	rt.loadDotEnv()

	if err := rt.initConfigAndLogger(); err != nil {
		return err
	}
	defer rt.closeLogger()

	if err := rt.initCoreServices(); err != nil {
		rt.runCleanups()
		return err
	}

	router, err := rt.buildRouter()
	if err != nil {
		rt.runCleanups()
		return err
	}

	return rt.serve(router)
}

func (rt *Runtime) loadDotEnv() {
	if config.IsProductionLikeEnv(os.Getenv("APP_ENV")) || os.Getenv("GIN_MODE") == "release" {
		return
	}
	if err := godotenv.Load("../.env"); err != nil {
		fmt.Fprintf(os.Stderr, "debug: .env not loaded: %v\n", err)
	}
}

func (rt *Runtime) initConfigAndLogger() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	rt.cfg = cfg
	rt.isProduction = config.IsProductionLikeEnv(cfg.App.Env)
	configureRBACAuthorizer(cfg.App.Env)

	logger.Init(logger.Config{
		Level:           cfg.Log.Level,
		Format:          cfg.Log.Format,
		Output:          cfg.Log.Output,
		SamplingEnabled: cfg.Log.SamplingEnabled,
		SamplingInitial: cfg.Log.SamplingInitial,
		SamplingAfter:   cfg.Log.SamplingAfter,
		ServiceName:     cfg.Log.ServiceName,
		Environment:     cfg.Log.Environment,
		ServiceVersion:  rt.build.Version,
	})
	return nil
}

func (rt *Runtime) closeLogger() {
	if err := logger.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "logger close error: %v\n", err)
	}
}

func (rt *Runtime) addCleanup(fn func()) {
	rt.cleanups = append(rt.cleanups, fn)
}

func (rt *Runtime) startBackgroundTask(ctx context.Context, name string, run func(context.Context)) {
	if run == nil {
		return
	}

	rt.bgWg.Add(1)
	go func() {
		defer rt.bgWg.Done()
		defer func() {
			if r := recover(); r != nil {
				fields := []gozap.Field{gozap.Any("panic", r)}
				if name != "" {
					fields = append(fields, gozap.String("task", name))
				}
				logger.L().Error("background task panicked", fields...)
			}
		}()

		if name != "" {
			logger.L().Info("background task started", gozap.String("task", name))
			defer logger.L().Info("background task stopped", gozap.String("task", name))
		}

		run(ctx)
	}()
}

func (rt *Runtime) runCleanups() {
	if rt.bgCancel != nil {
		rt.bgCancel()
	}
	rt.bgWg.Wait()
	for i := len(rt.cleanups) - 1; i >= 0; i-- {
		rt.cleanups[i]()
	}
}

func (rt *Runtime) initCoreServices() error {
	if err := crypto.InitHMACKey(rt.cfg.App.HMACSecret, rt.isProduction); err != nil {
		return fmt.Errorf("failed to initialize HMAC key: %w", err)
	}

	obsShutdown, err := observability.Setup(context.Background(), rt.cfg.Observability, rt.cfg.App.Env, observability.BuildInfo{
		Version:   rt.build.Version,
		GitCommit: rt.build.GitCommit,
		BuildTime: rt.build.BuildTime,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}
	rt.addCleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := obsShutdown(ctx); shutdownErr != nil {
			logger.L().Warn("observability shutdown error", gozap.Error(shutdownErr))
		}
	})

	redisClient, err := redisclient.NewClient(rt.cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	rt.redisClient = redisClient
	rt.addCleanup(func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.L().Warn("redis client close error", gozap.Error(closeErr))
		}
	})

	pgPool, err := db.NewPGPool(rt.cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	rt.pgPool = pgPool
	rt.database = db.NewDB(pgPool, time.Duration(rt.cfg.Database.QueryTimeout)*time.Second)
	rt.addCleanup(rt.database.Close)
	audit.ConfigureRepository(audit.NewRepository(rt.database))

	tokenService, err := token.NewService(token.ServiceConfig{
		RedisClient: redisClient.GetClient(),
		AccessTTL:   rt.cfg.Token.AccessTokenTTL,
		RefreshTTL:  rt.cfg.Token.RefreshTokenTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize token service: %w", err)
	}
	rt.tokenService = tokenService
	rt.addCleanup(tokenService.Close)

	oidcClient, err := oidc.NewClient(context.Background(), rt.cfg.Casdoor)
	if err != nil {
		return fmt.Errorf("failed to initialize OIDC client: %w", err)
	}
	rt.oidcClient = oidcClient

	return nil
}
