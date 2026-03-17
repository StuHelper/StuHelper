# 日志与审计模块

日志系统由三部分组成：应用日志（Zap 结构化日志）、请求日志（中间件）和审计日志（认证和业务事件）。

## 代码路径

| 位置 | 职责 |
| --- | --- |
| `server/internal/pkg/logger` | Zap 全局 Logger、上下文字段传播、敏感值脱敏 |
| `server/internal/pkg/middleware/logging.go` | 请求日志中间件（RequestLogger）、请求 ID 注入（RequestIDMiddleware）、panic 恢复（Recovery） |
| `server/internal/pkg/audit` | 认证和业务事件审计日志 |
| `server/internal/modules/course/review/*log*` | 评课管理操作日志的写入、查询和清理 |

## 应用日志

基于 Zap 的结构化日志，支持 JSON 和 Console 两种输出格式。

- `logger.L()` 返回全局 Logger，`logger.S()` 返回 SugaredLogger
- `logger.FromGin(c)` 从 Gin context 获取带 `request_id` 的 Logger
- `logger.GinContext(c, l)` 将 Logger 注入 Gin context
- 敏感值脱敏：`MaskSensitiveData`（用户名部分遮盖）、`MaskIP`（IP 地址部分遮盖）
- 支持文件输出（lumberjack 轮转：按大小、备份数、保留天数、压缩）
- 支持采样配置（高吞吐场景下限制日志量）

## 请求日志

`RequestLogger` 中间件记录每个 HTTP 请求：

- 请求路径、方法、Query（敏感参数脱敏）
- 响应状态码、耗时、响应大小
- 客户端 IP、User-Agent（截断到 256 字符）
- 用户 ID（认证后可用）
- 请求 ID（`X-Request-ID` 回传或自动生成 UUID）

日志级别根据状态码自动选择：500+ 为 Error，400+ 为 Warn，其余为 Info。

敏感 Query 参数黑名单：`code`、`token`、`access_token`、`refresh_token`、`password`、`secret`、`state` 等，值替换为 `[REDACTED]`。

## 审计日志

`server/internal/pkg/audit` 通过结构化日志记录认证和关键业务事件。

认证事件：

| 事件 | 说明 |
| --- | --- |
| `user.login` | 登录成功 |
| `user.login_failed` | 登录失败 |
| `user.logout` | 单设备登出 |
| `user.logout_all` | 全设备登出 |
| `token.refresh` | 令牌刷新 |
| `token.revoked` | 令牌被撤销 |

业务事件：

| 事件 | 说明 |
| --- | --- |
| `user.review_post` / `edit` / `delete` | 用户评课操作 |
| `user.vote` / `report` / `reply` / `favorite` | 用户互动操作 |
| `admin.review_hide` / `delete` | 管理员审核操作 |
| `admin.report_resolve` / `config_change` / `user_ban` | 管理员管理操作 |
| `system.cron_*` / `cache_refresh` / `stats_update` | 系统操作 |

审计日志包含 `user_id`、`username`（脱敏）、`ip`（脱敏）、`user_agent`、`request_id`、`resource`、`action`、`result` 等字段。

## 操作日志查询 API

| 端点 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/course/review/admin/logs` | GET | 查询评课管理操作日志 |

操作日志存储在 `admin_operation_logs` 表中，包含 `admin_username`、`admin_user_id`、`action`、`resource_type`、`resource_id`、`old_value`（JSONB）、`new_value`（JSONB）、`ip_address`、`user_agent`。

## 环境变量配置

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `LOG_LEVEL` | 最低日志级别（`debug`、`info`、`warn`、`error`） | `info` |
| `LOG_FORMAT` | 输出格式（`console`、`json`） | `json` |
| `LOG_OUTPUT` | 输出目标（`stdout`、`stderr`） | `stdout` |
| `APP_ENV` | 应用环境（`development`、`production`） | `development` |
| `LOG_FILE_ENABLED` | 是否启用文件输出 | `false` |
| `LOG_FILE_PATH` | 日志文件路径 | `logs/app.log` |
| `LOG_FILE_MAX_SIZE` | 单文件最大大小（MB） | `100` |
| `LOG_FILE_MAX_BACKUPS` | 最大备份数 | `3` |
| `LOG_FILE_MAX_AGE` | 最大保留天数 | `7` |
| `LOG_FILE_COMPRESS` | 是否压缩旧文件 | `true` |
| `LOG_SAMPLING_ENABLED` | 是否启用采样 | `false` |

## 存储说明

| 日志类型 | 存储位置 | 说明 |
| --- | --- | --- |
| 应用日志 | stdout/文件 | 由容器运行时或 lumberjack 管理 |
| 请求日志 | stdout/文件 | 同应用日志 |
| 审计日志 | stdout/文件 | 通过 Zap 结构化日志输出 |
| 操作审计 | `admin_operation_logs` 表 | 可通过管理 API 查询 |
