# 应用级通知中心

> **本文档是设计目标，当前尚未实现。** 现有通知接口暂时挂在评课模块路由下（`/api/v1/course/review/user/notifications/*`），只支持拉取模式。本文档描述的是未来要落地的应用级实时通知能力。

## 设计目标

| 目标     | 说明                                                       |
| -------- | ---------------------------------------------------------- |
| 应用级   | 通知中心不属于任何业务模块，所有模块共享同一套通知基础设施 |
| 实时推送 | 新通知通过 SSE 实时推送到前端，用户无需刷新页面            |
| 可扩展   | 新模块接入只需发送通知事件，不需要改通知中心代码           |
| 可靠     | 推送失败不丢通知，用户上线后能拉到离线期间的未读通知       |

## 技术选型

采用 **SSE（Server-Sent Events）** 作为实时推送通道。

| 对比项   | SSE                             | WebSocket                         | 轮询             |
| -------- | ------------------------------- | --------------------------------- | ---------------- |
| 通信方向 | 服务端 → 客户端（单向）         | 双向                              | 客户端 → 服务端  |
| 协议     | HTTP                            | 独立协议（ws://）                 | HTTP             |
| 认证     | Cookie 自动携带                 | 需要在握手或首条消息中处理        | Cookie 自动携带  |
| 断线重连 | 浏览器原生支持（`EventSource`） | 需要手动实现                      | 不适用           |
| 反向代理 | Nginx/Caddy 原生支持            | 需要额外配置 Upgrade              | 原生支持         |
| HTTP/2   | 多路复用，不额外占连接          | 独立 TCP 连接                     | 每次请求一个连接 |
| 复杂度   | 低（Gin 原生支持流式响应）      | 中（需要 gorilla/websocket 等库） | 最低             |

选择 SSE 的理由：通知是纯服务端推送场景，不需要客户端向服务端发消息；SSE 基于 HTTP 协议，认证、CORS、反向代理都走现有机制，不引入新的基础设施依赖。

## 整体架构

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  评课模块    │────>│              │     │                 │
│  回复/审核   │     │  通知总线     │────>│  SSE 连接管理    │───> 浏览器 A
├─────────────┤     │ (Redis Pub/  │     │  (per-user      │───> 浏览器 B
│  用户模块    │────>│   Sub)       │     │   goroutine)    │
│  认证结果    │     │              │     │                 │
├─────────────┤     └──────────────┘     └─────────────────┘
│  未来模块    │────>        |                     |
│  ...        │             v                     v
└─────────────┘     ┌──────────────┐     ┌─────────────────┐
                    │  通知存储     │     │  REST API       │
                    │ (notifications│<────│  拉取/已读/设置  │
                    │   表)        │     │                 │
                    └──────────────┘     └─────────────────┘
```

### 数据流

1. 业务模块产生事件（如评论被回复、实名认证通过）
2. 业务模块调用通知服务的 `Send` 方法，写入 `notifications` 表
3. 写入成功后，通过 Redis `PUBLISH` 发布事件到用户频道
4. SSE 连接管理器订阅了该用户频道，收到事件后推送到浏览器
5. 用户离线期间的通知留在数据库，上线后通过 REST API 拉取

## 接收者标识策略

通知中心统一使用内部 `user_id` 作为接收者标识。

`user_hash` 只适合匿名展示，不适合做通知中心的主键。它来自 HMAC 哈希，不能从一条匿名内容直接反查出接收者，也不适合做 SSE 频道、通知归属和跨模块关联。

对评课社区这类匿名业务，目标方案是在业务表里补充仅后端可见的 `owner_user_id` 或 `author_user_id` 字段。前端和公开 API 仍然只暴露匿名 `userHash`，不会因为引入内部用户 ID 破坏匿名性。

## SSE 连接管理

### 端点

```
GET /api/v1/notifications/stream
```

需要认证（HttpOnly Cookie）。响应头：

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no       # 禁用 Nginx 缓冲
```

### 事件类型

| 事件           | 数据           | 触发时机                           |
| -------------- | -------------- | ---------------------------------- |
| `notification` | 完整通知 JSON  | 收到新通知                         |
| `unread_count` | `{"count": N}` | 未读数变化（新通知到达或标记已读） |
| `:` (注释)     | 空             | 每 30 秒心跳保活，防止代理超时断连 |

### 事件格式

```
event: notification
data: {"id":"abc","type":"reply","title":"你的评论收到了新回复","body":"张三回复了你的评论","sourceModule":"course_review","sourceID":"review-123","createdAt":"2026-03-17T12:00:00Z"}

event: unread_count
data: {"count":5}

: heartbeat
```

### 连接生命周期

```
客户端                              服务端
  |                                   |
  |-- GET /notifications/stream ----->|
  |                                   |-- 验证认证 Cookie
  |                                   |-- 获取 userID
  |                                   |-- 创建 SSE 响应流
  |                                   |-- 注册到连接管理器（userID → chan）
  |                                   |-- 订阅 Redis channel: notify:{userID}
  |                                   |
  |<-- event: unread_count -----------|  (连接建立时推送当前未读数)
  |                                   |
  |<-- : heartbeat -------------------|  (每 30 秒)
  |                                   |
  |   ... (业务模块产生通知) ...       |
  |                                   |
  |<-- event: notification -----------|  (实时推送)
  |<-- event: unread_count -----------|  (未读数 +1)
  |                                   |
  |-- (连接断开 / 页面关闭) --------->|
  |                                   |-- 从连接管理器注销
  |                                   |-- 取消 Redis 订阅
  |                                   |
  |-- (浏览器自动重连 EventSource) -->|
  |                                   |-- 重复上述流程
```

### 多标签页 / 多设备

同一用户可能同时打开多个浏览器标签页或多台设备。连接管理器为每个用户维护一个连接列表（`map[userID][]chan`），新通知广播到该用户的所有活跃连接。

## 通知总线

使用 Redis Pub/Sub 作为进程间通知通道。

### 频道命名

```
notify:{userID}
```

每个用户一个频道。SSE 连接建立时订阅，断开时取消。

### 发布消息格式

```json
{
  "type": "new_notification",
  "notificationID": "abc-123",
  "notification": { ... }
}
```

### 为什么用 Redis Pub/Sub

- 项目已经依赖 Redis（限流、缓存、黑名单），不引入新依赖
- Pub/Sub 是 fire-and-forget 语义，用户不在线时消息丢弃，但通知已持久化到数据库，不会丢失
- 多实例部署时，Redis Pub/Sub 天然跨进程广播

### 备选方案（后续评估）

- **PostgreSQL LISTEN/NOTIFY**：不引入 Redis 依赖，但连接池管理更复杂
- **Redis Streams**：支持消费确认和持久化，适合更高可靠性要求的场景

## 通知数据模型

### notifications 表（目标结构）

| 字段            | 类型        | 说明                                     |
| --------------- | ----------- | ---------------------------------------- |
| `id`            | UUID        | 主键                                     |
| `user_id`       | INT8        | 接收者内部用户 ID，通知中心唯一归属键    |
| `type`          | VARCHAR     | 通知类型（见下表）                       |
| `title`         | VARCHAR     | 通知标题                                 |
| `body`          | TEXT        | 通知正文（可为空）                       |
| `source_module` | VARCHAR     | 来源模块标识                             |
| `source_id`     | VARCHAR     | 来源资源 ID（如 review ID、identity ID） |
| `source_url`    | VARCHAR     | 点击跳转路径（前端路由）                 |
| `is_read`       | BOOLEAN     | 是否已读                                 |
| `created_at`    | TIMESTAMPTZ | 创建时间                                 |

### 通知类型

| type                | 来源模块      | 触发场景         |
| ------------------- | ------------- | ---------------- |
| `reply`             | course_review | 评论收到新回复   |
| `review_hidden`     | course_review | 评论被管理员隐藏 |
| `review_restored`   | course_review | 评论被管理员恢复 |
| `report_resolved`   | course_review | 用户的举报已处理 |
| `identity_approved` | user          | 实名认证通过     |
| `identity_rejected` | user          | 实名认证被拒绝   |
| `student_approved`  | user          | 学生认证通过     |
| `student_rejected`  | user          | 学生认证被拒绝   |

后续模块接入时扩展此枚举即可。

## REST API

通知中心迁移到独立路由后的端点设计：

| 端点                                 | 方法 | 说明                                                     |
| ------------------------------------ | ---- | -------------------------------------------------------- |
| `/api/v1/notifications/stream`       | GET  | SSE 实时推送连接                                         |
| `/api/v1/notifications`              | GET  | 分页拉取通知列表（支持 `?type=reply&isRead=false` 过滤） |
| `/api/v1/notifications/unread-count` | GET  | 未读通知计数                                             |
| `/api/v1/notifications/:id/read`     | PUT  | 标记单条已读                                             |
| `/api/v1/notifications/read-all`     | PUT  | 全部标记已读                                             |
| `/api/v1/notifications/settings`     | GET  | 获取通知偏好设置（后续）                                 |
| `/api/v1/notifications/settings`     | PUT  | 更新通知偏好设置（后续）                                 |

### 迁移路径

1. 在匿名业务表中补充仅后端可见的 `owner_user_id` / `author_user_id` 字段，后续通知一律按 `user_id` 发送
2. 新建 `server/internal/modules/notification/` 模块
3. 将 `notifications` 表迁移到新结构，并在兼容期内保留旧字段用于回填和过渡
4. 更新 OpenAPI、后端生成代码和前端类型，统一新通知模型
5. 将现有 `course/review/review_notification.go` 和 `repository_notification.go` 中的通知查询逻辑迁移到新模块
6. 评课模块的通知路由（`/course/review/user/notifications/*`）标记为 deprecated，保留一段时间后移除
7. 前端切换到新路由并接入 SSE

## 业务模块接入接口

业务模块通过依赖注入获得 `NotificationSender` 接口：

```go
// NotificationSender 通知发送接口，业务模块依赖此接口发送通知
type NotificationSender interface {
    // Send 创建通知并推送到在线用户
    // 写入数据库 + 发布到 Redis Pub/Sub
    Send(ctx context.Context, params SendParams) error

    // SendBatch 批量发送（如管理员批量操作后通知多个用户）
    SendBatch(ctx context.Context, params []SendParams) error
}

type SendParams struct {
    UserID       int64   // 接收者内部用户 ID
    Type         string  // 通知类型
    Title        string  // 标题
    Body         string  // 正文（可选）
    SourceModule string  // 来源模块标识
    SourceID     string  // 来源资源 ID
    SourceURL    string  // 前端跳转路径
}
```

评课模块使用示例：

```go
// 回复创建后发送通知
func (s *Service) CreateReply(ctx context.Context, params CreateReplyParams) (*Reply, error) {
    // ... 创建回复逻辑 ...

    // 通知评论作者
    _ = s.notifier.Send(ctx, notification.SendParams{
        UserID:       review.OwnerUserID,
        Type:         "reply",
        Title:        "你的评论收到了新回复",
        SourceModule: "course_review",
        SourceID:     params.ReviewID,
        SourceURL:    fmt.Sprintf("/course/%d/reviews/%s", review.CourseID, params.ReviewID),
    })

    return reply, nil
}
```

## 前端接入

### EventSource 连接

```typescript
const source = new EventSource("/api/v1/notifications/stream", {
	withCredentials: true, // 携带 HttpOnly Cookie
});

source.addEventListener("notification", (event) => {
	const notification = JSON.parse(event.data);
	// 显示桌面通知或 Toast
	showToast(notification.title);
	// 更新通知列表
	notificationStore.prepend(notification);
});

source.addEventListener("unread_count", (event) => {
	const { count } = JSON.parse(event.data);
	// 更新导航栏未读角标
	notificationStore.setUnreadCount(count);
});

source.onerror = () => {
	// EventSource 会自动重连，无需手动处理
	// 可以在此更新 UI 状态为"重连中"
};
```

### 页面可见性优化

当用户切换到后台标签页时，可以关闭 SSE 连接节省资源，切回前台时重新连接并拉取离线期间的通知：

```typescript
document.addEventListener("visibilitychange", () => {
	if (document.hidden) {
		source.close();
	} else {
		reconnect();
		fetchUnreadNotifications(); // 拉取离线期间的通知
	}
});
```

## 部署注意事项

### Nginx 配置

SSE 需要禁用缓冲和超时调整：

```nginx
location /api/v1/notifications/stream {
    proxy_pass http://backend;
    proxy_set_header Connection '';
    proxy_http_version 1.1;
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 3600s;  # 1 小时，配合心跳保活
}
```

### 连接数控制

每个 SSE 连接占用一个服务端 goroutine。需要：

- 设置单用户最大连接数（建议 5，防止标签页泄漏）
- 设置全局最大连接数（根据服务器资源）
- 心跳超时后主动关闭无响应连接

## 实现优先级

| 阶段 | 内容                          | 依赖 |
| ---- | ----------------------------- | ---- |
| P0   | 通知模块独立、REST API 迁移   | 无   |
| P1   | SSE 推送 + Redis Pub/Sub 总线 | P0   |
| P2   | 前端 EventSource 接入 + 角标  | P1   |
| P3   | 通知偏好设置（按类型开关）    | P0   |
| P4   | 页面可见性优化、连接数控制    | P2   |

## 相关文档

- [课程评论社区](../course/README.md) — 当前通知接口的临时宿主
- [互动系统](../course/02-interaction.md) — 回复与通知接入计划
- [用户系统](../user-system/README.md) — 认证结果触发通知的场景
