---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-04-19
---

# 通知中心

> 状态：现行，通知实现已统一收口到 `notification` 模块，对外沿用评课用户命名空间

## 现状

OpenAPI 和 `clients/shared` 使用的接口：

- `GET /api/v1/course/review/user/notifications`
- `GET /api/v1/course/review/user/notifications/stream`
- `GET /api/v1/course/review/user/notifications/unread-count`
- `PUT /api/v1/course/review/user/notifications/{notificationID}/read`
- `PUT /api/v1/course/review/user/notifications/read-all`

统一入口：`/api/v1/course/review/user/notifications/*`。该路径前缀是稳定的对外命名空间，不代表代码归属仍在 `review` 模块。

运行时实现统一由 `notification` 模块负责：通知写入、列表查询、已读更新、未读数统计、Redis Pub/Sub 广播和 SSE 推送都在同一模块内闭环。

## 数据模型

`notifications` 表核心字段：`user_id` / `type` / `title` / `body` / `payload`（JSONB） / `source_module` / `source_id` / `source_url` / `source_course_id`（int64） / `is_read`

数据库层已迁移到独立通知模块结构。

## SSE

支持 `notification` 和 `unread_count` 事件，30 秒心跳。

流程：校验用户 → 建立连接 → 推送未读数 → 订阅 `notify:{userID}` → 收到消息后广播。

## 通知类型

`reply` / `like` / `review_hidden` / `review_restored` / `report_resolved` / `identity_approved` / `identity_rejected` / `student_approved` / `student_rejected` / `system`

## 代码入口

| 组件 | 位置 |
|------|------|
| 通知模块 | `server/internal/modules/notification/` |
| 评课域通知发送方 | `server/internal/modules/course/review/service_review_write.go` / `service_interaction.go` |
| Shared 通知封装 | `clients/shared/src/api/notification.ts` |
