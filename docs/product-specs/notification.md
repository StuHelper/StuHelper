# 通知中心

> 状态：现行，双轨逐步统一中

## 现状

### 轨道 A：前端主要链路

OpenAPI 和 `clients/shared` 使用的接口：

- `GET /api/v1/course/review/user/notifications`
- `GET /api/v1/course/review/user/notifications/unread-count`
- `PUT /api/v1/course/review/user/notifications/{notificationID}/read`
- `PUT /api/v1/course/review/user/notifications/read-all`

### 轨道 B：独立通知模块

后端已注册的运行时路由：

- `GET /api/v1/notifications`
- `GET /api/v1/notifications/unread-count`
- `PUT /api/v1/notifications/:id/read`
- `PUT /api/v1/notifications/read-all`
- `GET /api/v1/notifications/stream`（SSE）

目标：统一通知入口，`user_id` 归属键，Redis Pub/Sub 广播，SSE 推送。

## 数据模型

`notifications` 表核心字段：`user_id` / `type` / `title` / `body` / `source_module` / `source_id` / `source_url` / `is_read`

数据库层已迁移到独立通知模块结构。

## SSE

支持 `notification` 和 `unread_count` 事件，30 秒心跳。

流程：校验用户 → 建立连接 → 推送未读数 → 订阅 `notify:{userID}` → 收到消息后广播。

## 通知类型

`reply` / `review_hidden` / `review_restored` / `report_resolved` / `identity_approved` / `identity_rejected` / `student_approved` / `student_rejected`

## 统一方向

前端、OpenAPI、Shared API 统一到 `/api/v1/notifications/*`。

## 代码入口

| 组件 | 位置 |
|------|------|
| 独立通知模块 | `server/internal/modules/notification/` |
| 评课旧通知 | `server/internal/modules/course/review/review_notification.go` |
| Web 通知封装 | `clients/shared/src/api/notification.ts` |
