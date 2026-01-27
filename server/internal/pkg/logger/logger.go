package logger

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

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
	}
}

// Init 初始化全局 Logger
func Init(cfg Config) error {
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
		fileWriter := &lumberjack.Logger{
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

	globalLogger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return nil
}

// L 返回全局 Logger
func L() *zap.Logger {
	if globalLogger == nil {
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// S 返回全局 SugaredLogger
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// Sync 刷新日志缓冲
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
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
	if strings.ToLower(format) == "console" {
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
