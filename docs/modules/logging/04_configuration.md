# 日志配置

## 环境变量

```bash
# 日志级别: debug, info, warn, error
LOG_LEVEL=info

# 输出格式: json（生产）, console（开发）
LOG_FORMAT=json

# 输出目标: stdout, file, both
LOG_OUTPUT=stdout

# 采样配置（防止日志风暴）
LOG_SAMPLING_ENABLED=true
LOG_SAMPLING_INITIAL=100
LOG_SAMPLING_THEREAFTER=10

# 文件日志配置
LOG_FILE_ENABLED=false
LOG_FILE_PATH=./logs/app.log
LOG_FILE_MAX_SIZE=100
LOG_FILE_MAX_BACKUPS=10
LOG_FILE_MAX_AGE=30
LOG_FILE_COMPRESS=true

# 审计日志保留期（天）
AUDIT_LOG_RETENTION_DAYS=180

# 是否开启审计日志
AUDIT_LOG_ENABLED=true
```

## 配置结构体

```go
type LogConfig struct {
    Level           string `env:"LOG_LEVEL" default:"info"`
    Format          string `env:"LOG_FORMAT" default:"json"`
    Output          string `env:"LOG_OUTPUT" default:"stdout"`
    SamplingEnabled bool   `env:"LOG_SAMPLING_ENABLED" default:"true"`
    SamplingInitial int    `env:"LOG_SAMPLING_INITIAL" default:"100"`
    SamplingAfter   int    `env:"LOG_SAMPLING_THEREAFTER" default:"10"`

    FileEnabled    bool   `env:"LOG_FILE_ENABLED" default:"false"`
    FilePath       string `env:"LOG_FILE_PATH" default:"./logs/app.log"`
    FileMaxSize    int    `env:"LOG_FILE_MAX_SIZE" default:"100"`
    FileMaxBackups int    `env:"LOG_FILE_MAX_BACKUPS" default:"10"`
    FileMaxAge     int    `env:"LOG_FILE_MAX_AGE" default:"30"`
    FileCompress   bool   `env:"LOG_FILE_COMPRESS" default:"true"`
}
```

## 环境配置建议

| 环境 | LOG_LEVEL | LOG_FORMAT | LOG_OUTPUT | 采样 |
| ---- | --------- | ---------- | ---------- | ---- |
| 开发 | debug     | console    | stdout     | 关闭 |
| 测试 | debug     | json       | both       | 关闭 |
| 生产 | info      | json       | both       | 开启 |

## 采样策略建议

| 场景        | 建议策略                       |
| ----------- | ------------------------------ |
| 高 QPS 业务 | 启用采样，保留 ERROR/WARN 全量 |
| 敏感接口    | 禁止采样（如登录、权限变更）   |
| 定时任务    | INFO 采样，ERROR 全量          |

## 留存与合规建议

- 访问日志保留 7–30 天（按成本与合规要求调整）。
- 审计日志保留 90–365 天（按合规要求调整）。
- 涉及个人数据的字段默认脱敏或哈希化。
