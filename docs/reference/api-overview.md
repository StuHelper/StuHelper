---
type: reference
audience: backend-dev, frontend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-08-07
---

# API 导航摘要

> 本文档仅做模块分组索引。完整路径、参数、响应 schema 以 [`server/api/openapi.yaml`](../../server/api/openapi.yaml) 为准。改接口永远先改 OpenAPI，再 `make generate`。

## 基础路径

`/api/v1`

## 模块分组

| 模块 | 前缀 | 权威规格 |
|------|------|----------|
| 健康检查 | `/health/*` | `server/api/openapi.yaml` |
| 认证与会话 | `/api/v1/auth/*` | [design/auth-and-session.md](../design/auth-and-session.md) |
| 课程实体 | `/api/v1/course/*`（不含 `review`） | [product-specs/course-review.md](../product-specs/course-review.md) |
| 评课（公开） | `/api/v1/course/review/*`（无需认证） | [product-specs/course-review.md](../product-specs/course-review.md) |
| 评课（认证） | `/api/v1/course/review/*`（需认证） | [product-specs/course-review.md](../product-specs/course-review.md) |
| 用户中心 | `/api/v1/course/review/user/*` | [product-specs/notification.md](../product-specs/notification.md) |
| 评课后台 | `/api/v1/course/review/admin/*` | [design/authorization-model.md](../design/authorization-model.md) |
| 用户系统 | `/api/v1/user/*` | [product-specs/user-system.md](../product-specs/user-system.md) |
| 用户系统后台 | `/api/v1/admin/*` | [product-specs/user-system.md](../product-specs/user-system.md) |
| 学生认证 | `/api/v1/student-verification/*` | [product-specs/student-verification-and-group-admission.md](../product-specs/student-verification-and-group-admission.md) |
| 账号手机号 | `/api/v1/account/phone/*` | [product-specs/user-system.md](../product-specs/user-system.md) |
| 学生认证 Webhook | `/api/v1/webhooks/student-verification/*` | [product-specs/student-verification-and-group-admission.md](../product-specs/student-verification-and-group-admission.md) |
| 学生资格内部接口 | `/api/v1/internal/student-eligibility/*` | [product-specs/student-verification-and-group-admission.md](../product-specs/student-verification-and-group-admission.md) |
| 手机号门禁内部接口 | `/api/v1/internal/phone-gates/*` | [product-specs/student-verification-and-group-admission.md](../product-specs/student-verification-and-group-admission.md) |
| 入群与新生认证 | `/api/v1/admission/*` | [design/koishi-admission-verification.md](../design/koishi-admission-verification.md) |
| 机器人内部接口 | `/api/v1/bot/*`（`serviceTokenAuth`） | [product-specs/user-system.md](../product-specs/user-system.md) / [guides/koishi-development.md](../guides/koishi-development.md) |
| 教务展示 | `/api/v1/academics/*` | [product-specs/academics-data-integration.md](../product-specs/academics-data-integration.md) |
| 资源共享 | `/api/v1/resources/*` | [product-specs/resource-sharing.md](../product-specs/resource-sharing.md) |
| 通知 | `/api/v1/course/review/user/notifications/*`（含 SSE 子路径） | [product-specs/notification.md](../product-specs/notification.md) |
| 开放平台 | `/api/v1/open-platform/*` | [design/open-platform-v1.md](../design/open-platform-v1.md) |
| 开放平台后台 | `/api/v1/admin/open-platform/*` | [design/open-platform-v1.md](../design/open-platform-v1.md) / [design/authorization-model.md](../design/authorization-model.md) |
| 指标采集 | `/api/v1/metrics/*` | [guides/observability.md](../guides/observability.md) |

## 查找指引

- 想看**全量路由和字段**：打开 `server/api/openapi.yaml` 或本地启动后的 `/docs/` Swagger UI。
- 想看**业务规则**：去上表对应的 `product-specs/` 文档。
- 想看**认证/授权/会话机制**：去上表对应的 `design/` 文档。
- 想看**Go 生成类型**：`server/internal/api/gen/`（禁止手改）。
- 想看**TS 生成类型**：`clients/shared/src/types/api.gen.ts`（禁止手改）。

## 规则

本文件严格只做导航摘要，不复制 OpenAPI 中的路径、参数和字段定义。任何新增接口的权威入口永远是 `openapi.yaml`。本文仅在新增**模块一级**时更新。
