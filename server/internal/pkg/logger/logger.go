package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger atomic.Pointer[zap.Logger]
	fallbackOnce sync.Once
	fileWriter   *lumberjack.Logger // 保留引用以便关闭
)

// Config 日志配置
type Config struct {
	Level           string
	Format          string // json, console
	Output          string // stdout, stderr, file
	SamplingEnabled bool
	SamplingInitial int
	SamplingAfter   int
	FileEnabled     bool
	FilePath        string
	FileMaxSize     int // MB
	FileMaxBackups  int
	FileMaxAge      int // days
	FileCompress    bool
	ServiceName     string
	Environment     string
	ServiceVersion  string
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Level:           "info",
		Format:          "json",
		Output:          "stdout",
		SamplingEnabled: false,
		SamplingInitial: 100,
		SamplingAfter:   100,
		FileEnabled:     false,
		FilePath:        "logs/app.log",
		FileMaxSize:     100,
		FileMaxBackups:  3,
		FileMaxAge:      7,
		FileCompress:    true,
		ServiceName:     "stuhelper-backend",
		Environment:     "development",
		ServiceVersion:  "dev",
	}
}

// Init 初始化全局 Logger
func Init(cfg Config) {
	level := parseLevel(cfg.Level)
	encoder := buildEncoder(cfg.Format)

	var cores []zapcore.Core

	// 控制台输出
	consoleCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(getOutput(cfg.Output)),
		level,
	)
	cores = append(cores, consoleCore)

	// 文件输出
	if cfg.FileEnabled {
		fileWriter = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.FileMaxSize,
			MaxBackups: cfg.FileMaxBackups,
			MaxAge:     cfg.FileMaxAge,
			Compress:   cfg.FileCompress,
		}
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(buildEncoderConfig()),
			zapcore.AddSync(fileWriter),
			level,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)

	// 采样配置
	if cfg.SamplingEnabled {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			cfg.SamplingInitial,
			cfg.SamplingAfter,
		)
	}

	// CallerSkip(0)：L() 直接返回 logger 给业务代码使用（logger.L().Info(...)），
	// 无包级 wrapper 函数，不需要跳过额外调用层
	l := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if cfg.ServiceName != "" || cfg.Environment != "" || cfg.ServiceVersion != "" {
		fields := make([]zap.Field, 0, 3)
		if cfg.ServiceName != "" {
			fields = append(fields, zap.String("service.name", cfg.ServiceName))
		}
		if cfg.Environment != "" {
			fields = append(fields, zap.String("deployment.environment", cfg.Environment))
		}
		if cfg.ServiceVersion != "" {
			fields = append(fields, zap.String("service.version", cfg.ServiceVersion))
		}
		l = l.With(fields...)
	}

	// 替换前先 Sync 旧 logger，避免丢失缓冲日志
	if old := globalLogger.Swap(l); old != nil {
		_ = old.Sync() //nolint:errcheck // best-effort flush
	}
}

// L 返回全局 Logger
func L() *zap.Logger {
	if l := globalLogger.Load(); l != nil {
		return l
	}
	fallbackOnce.Do(func() {
		l, err := zap.NewProduction()
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to create fallback logger: %v\n", err)
			l = zap.NewNop()
		}
		globalLogger.Store(l)
	})
	// fallbackOnce.Do 保证 Store 已完成，防御性兜底避免返回 nil
	if l := globalLogger.Load(); l != nil {
		return l
	}
	return zap.NewNop()
}

// S 返回全局 SugaredLogger
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// Sync 刷新日志缓冲
func Sync() error {
	if l := globalLogger.Load(); l != nil {
		return l.Sync()
	}
	return nil
}

// Close 刷新日志缓冲并关闭文件写入器
func Close() error {
	err := Sync()
	// Sync 失败时重试一次，尽量保证缓冲数据落盘
	if err != nil {
		if retryErr := Sync(); retryErr == nil {
			err = nil
		}
		// 重试仍失败则继续关闭 fileWriter，避免资源泄漏；
		// Sync 错误已记录在 err 中，调用方可据此判断是否有数据丢失风险
	}
	if fileWriter != nil {
		if closeErr := fileWriter.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		fileWriter = nil
	}
	return err
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func buildEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func buildEncoder(format string) zapcore.Encoder {
	cfg := buildEncoderConfig()
	if strings.EqualFold(format, "console") {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

func getOutput(output string) *os.File {
	switch strings.ToLower(output) {
	case "stderr":
		return os.Stderr
	default:
		return os.Stdout
	}
}
