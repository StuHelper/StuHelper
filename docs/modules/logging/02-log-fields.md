# 日志字段规范

## 标准字段定义

所有日志必须包含以下基础字段：

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "request completed",
  "caller": "handler.go:42",
  "request_id": "req_abc123",
  "service": "stuhelper"
}
```

## 字段分类

### 请求相关字段

| 字段名       | 类型   | 必填 | 说明             |
| ------------ | ------ | ---- | ---------------- |
| `request_id` | string | 是   | 请求唯一标识     |
| `trace_id`   | string | 否   | 分布式追踪 ID    |
| `method`     | string | 是   | HTTP 方法        |
| `path`       | string | 是   | 请求路径         |
| `status_code`| int    | 是   | 响应状态码       |
| `latency_ms` | int64  | 是   | 响应时间（毫秒） |
| `client_ip`  | string | 是   | 客户端 IP        |

### 用户相关字段

| 字段名       | 类型   | 必填 | 说明               |
| ------------ | ------ | ---- | ------------------ |
| `user_id`    | string | 否   | 用户 ID（已认证时）|
| `username`   | string | 否   | 用户名（脱敏）     |
| `session_id` | string | 否   | 会话 ID            |

### 错误相关字段

| 字段名        | 类型   | 必填 | 说明                             |
| ------------- | ------ | ---- | -------------------------------- |
| `error`       | string | 否   | 错误信息                         |
| `error_code`  | string | 否   | 业务错误码                       |
| `stack_trace` | string | 否   | 堆栈信息（ERROR 级别自动添加）   |

### 业务相关字段

| 字段名        | 类型   | 必填 | 说明                           |
| ------------- | ------ | ---- | ------------------------------ |
| `module`      | string | 否   | 业务模块（auth/course/...）    |
| `action`      | string | 否   | 操作类型（create/update/delete）|
| `resource`    | string | 否   | 资源类型（user/review/...）    |
| `resource_id` | string | 否   | 资源 ID                        |

## 日志输出示例

### 请求日志

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "request completed",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "POST",
  "path": "/api/v1/reviews",
  "status": 201,
  "latency_ms": 45,
  "client_ip": "192.168.1.100",
  "user_id": "user_abc123"
}
```

### 错误日志

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "error",
  "message": "failed to create review",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "error": "duplicate entry",
  "error_code": "REVIEW_DUPLICATE",
  "module": "course",
  "action": "create"
}
```
