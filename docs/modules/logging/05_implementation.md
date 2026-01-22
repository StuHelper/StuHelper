# Logger 核心实现

## 初始化代码

```go
// internal/pkg/logger/logger.go
package logger

import (
    "os"
    "time"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"
)

var (
    globalLogger *zap.Logger
    sugar        *zap.SugaredLogger
)

type Config struct {
    Level           string
    Format          string
    Output          string
    SamplingEnabled bool
    SamplingInitial int
    SamplingAfter   int
    FileConfig      *FileConfig
}

type FileConfig struct {
    Path       string
    MaxSize    int
    MaxBackups int
    MaxAge     int
    Compress   bool
}
```

## Init 函数

```go
func Init(cfg *Config) error {
    level, err := zapcore.ParseLevel(cfg.Level)
    if err != nil {
        level = zapcore.InfoLevel
    }

    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "timestamp",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        MessageKey:     "message",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.MillisDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    var encoder zapcore.Encoder
    if cfg.Format == "console" {
        encoder = zapcore.NewConsoleEncoder(encoderConfig)
    } else {
        encoder = zapcore.NewJSONEncoder(encoderConfig)
    }

    var cores []zapcore.Core

    if cfg.Output == "stdout" || cfg.Output == "both" {
        stdoutCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
        cores = append(cores, stdoutCore)
    }

    if cfg.FileConfig != nil && cfg.Output != "stdout" {
        fileWriter := &lumberjack.Logger{
            Filename:   cfg.FileConfig.Path,
            MaxSize:    cfg.FileConfig.MaxSize,
            MaxBackups: cfg.FileConfig.MaxBackups,
            MaxAge:     cfg.FileConfig.MaxAge,
            Compress:   cfg.FileConfig.Compress,
        }
        fileCore := zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level)
        cores = append(cores, fileCore)
    }

    core := zapcore.NewTee(cores...)

    if cfg.SamplingEnabled {
        core = zapcore.NewSamplerWithOptions(core, time.Second, cfg.SamplingInitial, cfg.SamplingAfter)
    }

    globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
    sugar = globalLogger.Sugar()

    return nil
}
```

## 访问函数

```go
func L() *zap.Logger {
    if globalLogger == nil {
        globalLogger, _ = zap.NewProduction()
    }
    return globalLogger
}

func S() *zap.SugaredLogger {
    if sugar == nil {
        sugar = L().Sugar()
    }
    return sugar
}

func Sync() error {
    if globalLogger != nil {
        return globalLogger.Sync()
    }
    return nil
}
```
