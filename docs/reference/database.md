---
type: reference
audience: backend-dev, ops
status: current
authoritative-source: server/migrations/
last-verified: 2026-07-29
---

# 数据库导航摘要

> 本文档仅做数据面与模块索引。表结构、索引、约束以 [`server/migrations/`](../../server/migrations/) 中按版本顺序应用后的完整 migration 集合为准；`000001_initial_schema.up.sql` 只是基线，不代表后续演进后的完整 schema。

## 数据面

| 存储 | 用途 | 权威来源 |
|------|------|----------|
| PostgreSQL | 业务数据 | `server/migrations/` 中的有序 migration 集合 |
| Redis | 会话、黑名单、限流、缓存、通知广播 | 代码使用处（无独立 schema） |
| 对象存储（MinIO/S3） | 证件照、资源文件 | 统一经 `storage` abstraction 访问 |
| Casdoor | 身份平面（账号、OIDC 会话、扁平角色目录） | Casdoor 管理端 |
| OpenFGA | 资源关系授权 | [`docs/design/openfga-model.fga`](../design/openfga-model.fga) |

## 业务平面索引

仅列模块与权威规格跳转。具体表名、字段和约束全部去 migration 查。

| 业务模块 | 表前缀 | 业务规格 |
|----------|---------------|----------|
| 用户与学校 | `user_*`、`school_*`、`system_*` | [product-specs/user-system.md](../product-specs/user-system.md) |
| 课程与评课 | `course_*`、`teacher_*`、`review_*`、`rating_*` | [product-specs/course-review.md](../product-specs/course-review.md) |
| 教务展示 | `academic_*` | [product-specs/academics-data-integration.md](../product-specs/academics-data-integration.md) |
| 资源共享 | `resource_*` | [product-specs/resource-sharing.md](../product-specs/resource-sharing.md) |
| 通知 | `notification*` | [product-specs/notification.md](../product-specs/notification.md) |
| 开放平台 | `open_platform_*` | [design/open-platform-v1.md](../design/open-platform-v1.md) |
| 审计与 outbox | `audit_events`、`domain_event_outbox` | [product-specs/audit-logging.md](../product-specs/audit-logging.md) |

## 设计约束（不改文档能看出来的除外）

这些约束写在这里是因为它们**跨多张表**，不是单个 migration 能完整表达：

- `users` 是 shadow user 表：业务外键锚点 + 最小用户画像缓存，身份真源是 Casdoor。
- `domain_event_outbox` 是统一 outbox：`stream + dedupe_key` 唯一键；`pending / processing / completed / failed` 状态机；后台 worker 按 stream 消费，主事务不直连外部系统。
- `audit_events.category = 'admin_operation'` 收口所有管理员操作的审计留痕。
- `open_platform_user_consents` 是第三方 app + user + scope 授权事实；scope consent 不写入 OpenFGA。
- 能力由角色静态展开，**不落本地 RBAC 表**；资源级权限由 OpenFGA 承担。详见 [design/authorization-model.md](../design/authorization-model.md)。
- `pg_trgm` 已启用，`courses.name/code`、`teachers.name` 上建有 GIN trigram 索引。

## 查找指引

- 想看**完整 schema**：从 `000001` 开始按版本顺序阅读或应用 `server/migrations/*.up.sql`；不能只看 baseline。
- 想看**查询与事务**：`server/internal/modules/**/repository*.go`。
- 想看**本地 ERD**：`make db-erd`（如果启用了 schemaspy/atlas）。

## 规则

本文不再维护完整表清单。任何 schema 变更直接体现在 baseline SQL；本文只在**新增业务模块**或**出现跨表设计约束**时更新。
