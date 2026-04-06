# 审计日志

> 状态：现行

## 三个组件

### 应用日志

Zap 结构化日志。

API：`logger.L()` / `logger.S()` / `logger.FromGin(c)`（带 request_id）

级别：`Info`（正常请求、审计事件）、`Warn`（可恢复问题、降级）、`Error`（5xx、panic 恢复）

输出：stdout 或文件（lumberjack 轮转，可选压缩）

脱敏：`MaskSensitiveData`（部分遮蔽）、`MaskIP`

### 请求日志

中间件 `RequestLogger`。

采集：path / method / query / status / latency / size / client_ip / user_agent（256 字符）/ user_id / request_id

敏感参数黑名单：`code` / `token` / `access_token` / `refresh_token` / `password` / `secret` / `state` → `[REDACTED]`

级别映射：500+ Error / 400+ Warn / 其他 Info

### 操作审计

表：`admin_operation_logs`

字段：admin_user_id / admin_username / action / resource_type / resource_id / old_value（JSONB）/ new_value（JSONB）/ ip_address / user_agent

查询：`GET /api/v1/course/review/admin/logs`

留存：90 天 + 每日清理。

## 审计事件

| 模块 | 事件 |
|------|------|
| 认证 | login / login_failed / logout / logout_all / token.refresh / token.revoked |
| 评课 | 发布/编辑/删除、投票、举报、回复、收藏 |
| 管理 | 隐藏/恢复/删除评课、编辑内容、处理举报 |

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `LOG_LEVEL` | 日志级别 | info |
| `LOG_FORMAT` | console / json | json |
| `LOG_OUTPUT` | stdout / stderr | stdout |
| `LOG_FILE_ENABLED` | 启用文件输出 | false |
| `LOG_FILE_PATH` | 文件路径 | — |
| `LOG_SAMPLING_ENABLED` | 启用采样 | false |

## 禁止记录

密码、access / refresh token、OAuth code、CSRF token、原始 secret、客户端堆栈追踪。

## 代码入口

| 组件 | 位置 |
|------|------|
| Logger | `server/internal/pkg/logger/` |
| 请求中间件 | `server/internal/pkg/middleware/logging.go` |
| 审计服务 | `server/internal/pkg/audit/` |
