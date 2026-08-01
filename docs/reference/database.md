---
type: reference
audience: backend-dev, ops
status: current
authoritative-source: server/migrations/
last-verified: 2026-08-01
---

# 数据库导航摘要

> 本文档仅做数据面与模块索引。表结构、索引、约束以 [`server/migrations/`](../../server/migrations/) 中按版本顺序应用后的完整 migration 集合为准；`000001_initial_schema.up.sql` 只是基线，不代表后续演进后的完整 schema。

## 数据面

| 存储 | 用途 | 权威来源 |
|------|------|----------|
| PostgreSQL | 业务数据 | `server/migrations/` 中的有序 migration 集合 |
| Redis | 会话、黑名单、限流、缓存、通知广播 | 代码使用处（无独立 schema） |
| 对象存储（MinIO/S3） | 证件照、资源文件 | 统一经 `storage` abstraction 访问 |
| Casdoor | 身份平面（账号、OIDC 会话、token、登录层 MFA） | Casdoor 管理端 |
| OpenFGA | 可从 PostgreSQL 授权账本/业务事实重建的资源关系投影 | [`docs/design/openfga-model.fga`](../design/openfga-model.fga) |

## 业务平面索引

仅列模块与权威规格跳转。具体表名、字段和约束全部去 migration 查。

| 业务模块 | 表前缀 | 业务规格 |
|----------|---------------|----------|
| 用户与学校 | `user_*`、`school_*`、`system_*` | [product-specs/user-system.md](../product-specs/user-system.md) |
| 课程与评课 | `course_*`、`teacher_*`、`review_*`、`rating_*` | [product-specs/course-review.md](../product-specs/course-review.md) |
| 教务展示 | `academic_*` | [product-specs/academics-data-integration.md](../product-specs/academics-data-integration.md) |
| 资源共享 | `resource_*` | [product-specs/resource-sharing.md](../product-specs/resource-sharing.md) |
| 通知 | `notification*` | [product-specs/notification.md](../product-specs/notification.md) |
| Admission 机器人动作 | `admission_bot_action_outbox` | [design/koishi-admission-verification.md](../design/koishi-admission-verification.md) |
| 开放平台 | `open_platform_*` | [design/open-platform-v1.md](../design/open-platform-v1.md) |
| 管理授权 | `authorization_grants`（scoped role 管理真源；Casdoor `super_admin` serving projection）、`authorization_authority_cutover`（一次性生产切换门禁） | [ADR-0008](../adr/0008-postgresql-authorization-control-plane.md) / [ADR-0009](../adr/0009-casdoor-organization-admin-super-admin-authority.md) |
| 审计与 outbox | `audit_events`、`domain_event_outbox` | [product-specs/audit-logging.md](../product-specs/audit-logging.md) |

## 设计约束（不改文档能看出来的除外）

这些约束写在这里是因为它们**跨多张表**，不是单个 migration 能完整表达：

- `users` 是 shadow user 表：业务外键锚点 + 最小用户画像缓存，身份真源是 Casdoor。
- `domain_event_outbox` 是统一 outbox：`stream + dedupe_key` 唯一键；`pending / processing / completed / failed / dead_letter` 状态机；后台 worker 按 stream 消费，主事务不直连外部系统。`revision` / `locked_revision` 是 supersession fence：处理中的同 key 新事件只提升 revision 并保留当前 lease，旧 worker 完成或失败时若发现 revision 已变化，就把最新修订重新排队，不能把新事件错误标成完成或 dead letter。进程取消时已 claim 但未处理的 job 以 detached finalize context 归还 pending，且不增加失败次数。单个 handler panic 在 per-job 边界转成带 stack 的普通失败，沿同一 attempt、指标和 dead-letter 路径处理；同批后续 job 继续，但这不把有外部副作用的消费提升为 exactly-once。
- `admission_bot_action_outbox.attempt_count` 同时作为下发世代号暴露为 `dispatchAttempt`；Koishi ACK 必须回传该值。服务端只有在 action 仍为 `dispatched` 且 attempt 相等时接受成功/失败回执；claim 路径的 preparation failure 与 stale 最终化也使用同一 fence，延迟到达的旧世代更新不能覆盖新一轮派发状态。
- Admission action claim 提交后只做每批两次 policy/failure 上下文查询。若批量查询失败或请求在 Service 返回前取消，动作尚未向 Bot 公开，服务端用独立 5 秒 context 一次性、批量归还仍属于本次 attempt 的 lease，并延后 30 秒再取；**补偿写成功时**不消耗业务重试预算，补偿写本身失败时原 claim 仍保持 `dispatched` 并保留此次 attempt，调用方会收到合并错误。单行动作缺少 policy 等确定性 preparation failure 则只让该行消费 attempt、按 backoff 重试并在第 5 次进入 `dead_letter`，不能阻塞健康行。这里的 attempt 回退只允许用于同步、未公开、不可重试的内部补偿：它会让下一次 claim 复用数值世代；若以后改为逐行流式发送、异步重试 cleanup 或多阶段处理，必须拆出单调 `dispatch_generation` / lease token，不能沿用此补偿。当前 Admission action 还没有显式 dead-letter replay API；terminal 行的运营恢复能力是未完成边界，不能把“poison 已隔离”解释为“poison 已可恢复”。
- `teacher_public_stats` 是异步投影。`reviews`、`teachers`、`departments` 的 statement-level trigger 在同一数据库事务中幂等 upsert `review_projection / teacher_public_stats_refresh / teacher_public_stats` job；单并发 worker 刷新物化视图后统一失效 Redis 缓存，周期性 enqueue 只承担漂移对账，不把同步刷新延迟带回 HTTP 请求。全量 `REFRESH MATERIALIZED VIEW CONCURRENTLY` 使用独立的 `REVIEW_TEACHER_STATS_REFRESH_TIMEOUT_SECONDS`（默认 60 秒、允许 5–90 秒），不受普通 `DB_QUERY_TIMEOUT` 的 5 秒默认值截断，但仍继承进程停机/调用方取消。上限必须低于共享 outbox 的 2 分钟 stale lease，避免旧刷新仍在执行时被另一 worker 重领；若 90 秒仍不足，应先测量并重新设计投影，不得继续放大 lease。延迟沿既有 `db_query_duration_seconds{operation="exec",table="mv_teacher_public_stats"}` 观测，重试/终止失败沿既有 outbox failure 指标和日志观测，不另建重复指标体系。
- 评课点赞与回复通知在业务写事务内写入 `review_notification` outbox。worker 可重试投递，`notifications.idempotency_key` 的 partial unique index 保证重复消费只产生一条持久通知；实时 SSE 只在首次插入时发送，不能把连接在线状态当作持久投递保证。
- `audit_events.category = 'admin_operation'` 收口所有管理员操作的审计留痕。
- `authorization_authority_cutover` 是 singleton 发布门禁，不是第二套授权账本。首次生产升级只有在
  经 Casdoor/OpenFGA 交叉验证的存量授权、对应 audit/outbox 与 marker 在同一事务提交后才从
  `pending` 变为 `completed`；production-like 应用启动必须检查该状态。重复发布只读取已完成
  digest，不得通过手工更新 marker 绕过冲突。
- `open_platform_user_consents` 是第三方 app + user + scope 授权事实；scope consent 不写入 OpenFGA。
- 能力由角色静态展开，**不落本地 RBAC 表**；资源级权限由 OpenFGA 承担。详见 [design/authorization-model.md](../design/authorization-model.md)。
- `pg_trgm` 已启用，`courses.name/code`、`teachers.name` 上建有 GIN trigram 索引。

## 查找指引

- 想看**完整 schema**：从 `000001` 开始按版本顺序阅读或应用 `server/migrations/*.up.sql`；不能只看 baseline。
- 想看**查询与事务**：`server/internal/modules/**/repository*.go`。
- 想看**本地 ERD**：`make db-erd`（如果启用了 schemaspy/atlas）。

## 规则

本文不再维护完整表清单。任何 schema 变更都通过新的递增 migration 体现；本文只在
**新增业务模块**或**出现跨表设计约束**时更新。
