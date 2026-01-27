# 审计日志设计

## 审计事件类型

| 事件类型                | 说明     | 保留期 |
| ----------------------- | -------- | ------ |
| `user.login`            | 用户登录 | 90 天  |
| `user.logout`           | 用户登出 | 90 天  |
| `user.password_change`  | 密码修改 | 永久   |
| `admin.config_change`   | 配置变更 | 永久   |
| `data.export`           | 数据导出 | 永久   |
| `security.failed_login` | 登录失败 | 180 天 |

## 审计日志格式

```json
{
	"timestamp": "2024-01-15T10:30:45.123Z",
	"level": "info",
	"message": "audit event",
	"event_type": "user.login",
	"event_result": "success",
	"user_id": "user_abc123",
	"client_ip": "192.168.1.100",
	"user_agent": "Mozilla/5.0...",
	"module": "auth",
	"action": "login",
	"details": {
		"method": "oauth2",
		"provider": "casdoor"
	}
}
```

## 存储与完整性

**建议存储方式**：

- 数据库表（便于检索与审计报表）
- 只读对象存储（用于归档与合规保留）

**完整性建议**：

- 重要审计事件可追加签名字段（例如 `event_hash`）
- 采用不可变存储或 WORM 策略

## 保留与清理策略

- 安全相关事件（失败登录、权限变更）保留 ≥ 180 天
- 业务敏感事件（导出、配置变更）保留 ≥ 365 天
- 超期按合规要求安全清理或归档

## 审计日志服务

```go
// internal/pkg/audit/audit.go
package audit

import (
    "context"
    "go.uber.org/zap"
    "your-project/internal/pkg/logger"
)

type EventType string

const (
    EventUserLogin    EventType = "user.login"
    EventUserLogout   EventType = "user.logout"
    EventDataExport   EventType = "data.export"
    EventConfigChange EventType = "admin.config_change"
)

type AuditEvent struct {
    EventType   EventType
    UserID      string
    ClientIP    string
    UserAgent   string
    Module      string
    Action      string
    Result      string
    Details     map[string]interface{}
}

func Log(ctx context.Context, event AuditEvent) {
    log := logger.FromContext(ctx)

    fields := []zap.Field{
        zap.String("event_type", string(event.EventType)),
        zap.String("event_result", event.Result),
        zap.String("user_id", event.UserID),
        zap.String("client_ip", event.ClientIP),
        zap.String("module", event.Module),
        zap.String("action", event.Action),
    }

    if event.Details != nil {
        fields = append(fields, zap.Any("details", event.Details))
    }

    log.Info("audit event", fields...)
}
```
