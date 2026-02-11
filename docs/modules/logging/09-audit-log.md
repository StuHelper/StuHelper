# 审计日志设计

## 审计事件类型

### 认证相关

| 事件类型                | 说明     | 保留期 |
| ----------------------- | -------- | ------ |
| `user.login`            | 用户登录 | 90 天  |
| `user.login_failed`     | 登录失败 | 180 天 |
| `user.logout`           | 用户登出 | 90 天  |
| `user.logout_all`       | 全设备登出 | 90 天  |
| `token.refresh`         | Token 刷新 | 30 天  |
| `token.revoked`         | Token 撤销 | 90 天  |

### 用户操作

| 事件类型                | 说明     | 保留期 |
| ----------------------- | -------- | ------ |
| `user.review_post`      | 发布评论 | 180 天 |
| `user.review_edit`      | 编辑评论 | 180 天 |
| `user.review_delete`    | 删除评论 | 180 天 |
| `user.vote`             | 投票     | 90 天  |
| `user.report`           | 举报     | 180 天 |
| `user.reply`            | 回复     | 90 天  |
| `user.favorite`         | 收藏     | 90 天  |

### 管理员操作

| 事件类型                | 说明     | 保留期 |
| ----------------------- | -------- | ------ |
| `admin.review_hide`     | 隐藏评论 | 永久   |
| `admin.review_delete`   | 删除评论 | 永久   |
| `admin.report_resolve`  | 处理举报 | 永久   |
| `admin.config_change`   | 配置变更 | 永久   |
| `admin.user_ban`        | 封禁用户 | 永久   |
| `admin.batch_operation` | 批量操作 | 永久   |

### 数据操作

| 事件类型                | 说明     | 保留期 |
| ----------------------- | -------- | ------ |
| `data.access`           | 数据访问 | 30 天  |
| `data.create`           | 数据创建 | 180 天 |
| `data.update`           | 数据更新 | 180 天 |
| `data.delete`           | 数据删除 | 永久   |
| `data.export`           | 数据导出 | 永久   |

### 系统操作

| 事件类型                | 说明       | 保留期 |
| ----------------------- | ---------- | ------ |
| `system.cron_start`     | 定时任务开始 | 30 天  |
| `system.cron_complete`  | 定时任务完成 | 30 天  |
| `system.cache_refresh`  | 缓存刷新   | 30 天  |
| `system.stats_update`   | 统计更新   | 30 天  |
| `system.error`          | 系统错误   | 180 天 |

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
