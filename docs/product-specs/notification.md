# 通知中心

> 状态：现行，已统一到评课用户命名空间

## 现状

OpenAPI 和 `clients/shared` 使用的接口：

- `GET /api/v1/course/review/user/notifications`
- `GET /api/v1/course/review/user/notifications/stream`
- `GET /api/v1/course/review/user/notifications/unread-count`
- `PUT /api/v1/course/review/user/notifications/{notificationID}/read`
- `PUT /api/v1/course/review/user/notifications/read-all`

运行时 SSE 由独立通知模块提供，但仍挂在同一路径前缀下：

- `GET /api/v1/course/review/user/notifications/stream`

统一入口：`/api/v1/course/review/user/notifications/*`，`user_id` 归属键，Redis Pub/Sub 广播，SSE 推送。

## 数据模型

`notifications` 表核心字段：`user_id` / `type` / `title` / `body` / `source_module` / `source_id` / `source_url` / `is_read`

数据库层已迁移到独立通知模块结构。

## SSE

支持 `notification` 和 `unread_count` 事件，30 秒心跳。

流程：校验用户 → 建立连接 → 推送未读数 → 订阅 `notify:{userID}` → 收到消息后广播。

## 通知类型

`reply` / `review_hidden` / `review_restored` / `report_resolved` / `identity_approved` / `identity_rejected` / `student_approved` / `student_rejected`

## 代码入口

| 组件 | 位置 |
|------|------|
| 独立通知模块 | `server/internal/modules/notification/` |
| 评课旧通知 | `server/internal/modules/course/review/review_notification.go` |
| Web 通知封装 | `clients/shared/src/api/notification.ts` |
