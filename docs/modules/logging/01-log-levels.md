# 日志级别规范

## 级别定义

| 级别      | 用途                           | 示例                                       |
| --------- | ------------------------------ | ------------------------------------------ |
| **DEBUG** | 开发调试信息，生产环境关闭     | 变量值、SQL 语句、缓存命中                 |
| **INFO**  | 正常业务流程记录               | 用户登录、请求完成、定时任务执行           |
| **WARN**  | 潜在问题，不影响主流程         | 重试成功、降级处理、配置缺失使用默认值     |
| **ERROR** | 错误，需要关注但服务可继续     | API 调用失败、数据库查询错误               |
| **FATAL** | 致命错误，服务无法继续         | 数据库连接失败、配置文件缺失               |

## 使用指南

### DEBUG - 仅开发环境

```go
logger.Debug("cache lookup",
    zap.String("key", cacheKey),
    zap.Bool("hit", hit),
)
```

**适用场景**：变量值调试、SQL 语句输出、缓存命中/未命中、函数入参/出参

### INFO - 正常业务事件

```go
logger.Info("user logged in",
    zap.String("user_id", userID),
    zap.String("method", "oauth2"),
)
```

**适用场景**：用户登录/登出、请求处理完成、定时任务执行、资源创建/更新/删除

### WARN - 可恢复的问题

```go
logger.Warn("retry succeeded",
    zap.Int("attempt", 3),
    zap.Duration("total_time", elapsed),
)
```

**适用场景**：重试后成功、降级处理、配置缺失使用默认值、慢请求

### ERROR - 需要关注的错误

```go
logger.Error("failed to fetch user profile",
    zap.Error(err),
    zap.String("user_id", userID),
)
```

**适用场景**：API 调用失败、数据库查询错误、外部服务不可用、业务逻辑异常

### FATAL - 致命错误

```go
logger.Fatal("database connection failed", zap.Error(err))
```

**适用场景**：数据库连接失败、配置文件缺失、必要服务不可用

> **注意**：`Fatal` 会调用 `os.Exit(1)`，仅在服务无法继续运行时使用。

## 环境配置建议

| 环境 | 推荐级别 | 说明                       |
| ---- | -------- | -------------------------- |
| 开发 | DEBUG    | 输出所有日志，便于调试     |
| 测试 | DEBUG    | 输出所有日志，便于问题排查 |
| 生产 | INFO     | 仅输出业务日志，减少日志量 |
