---
type: design
audience: backend-dev, ops
status: current
authoritative-source: server/internal/modules/auth/ + server/internal/modules/authorization/ + server/internal/platform/authorization/ + server/internal/pkg/fga/ + server/internal/pkg/outbox/ + server/internal/pkg/audit/
last-verified: 2026-07-31
---

# IAM 实施守卫

本文记录 IAM、认证、会话、outbox、审计 retention 和后台任务的硬性实现边界。它不是产品 spec，也不替代源码和测试；它只约束容易跨模块漂移的工程不变量。

## Token 与 Session 顺序

`login`、`refresh`、`logout`、`logout-all` 和任何 session rotation 流程必须先完成服务端权威状态写入，再向客户端发放 token、cookie 或成功响应。

- refresh 获取或签发新 token 后，必须先完成 session store 的 token family 更新；
- provider refresh token revoke / rotation 失败时，不得向客户端承诺成功；
- 本地 session 更新失败时，不得把新 access / refresh token 写入 `Set-Cookie` 或响应体；
- 失败注入测试必须覆盖 provider refresh 成功但本地 session rotate 失败的场景。

## OIDC JWKS 缓存

每个 provider verifier 必须复用一个进程生命周期的 `go-oidc` `RemoteKeySet`，不得用固定
时间 TTL 周期性销毁并重建它。

- 已缓存且签名有效的已知 `kid` 在 Casdoor 短暂不可用时继续本地验签；token 的 `iss`、
  `aud`、`exp`、session hash 和 blacklist 等其他门禁仍照常执行；
- 未知 `kid` 必须触发 `RemoteKeySet` 的一次回源；回源失败映射为 provider unavailable，
  当前请求 fail-closed，不能用任意已知 key 或陈旧 claim 猜测通过；
- 未知 `kid` 回源失败不能清空已有 key cache，恢复后仍可重新拉取；
- Casdoor 轮换签名 key 必须至少覆盖现存 token 的最大有效期，同时发布新旧 key。若需要紧急
  移除已泄漏的已知 key，先在 provider 撤下该 key、撤销受影响 session，再滚动重启 API
  verifier；不要靠任意分钟数的应用侧 TTL 猜测撤权窗口。

## 授权账本与 OpenFGA 投影

PostgreSQL `authorization_grants` 是 `super_admin`、`school_admin` 与 `section_*` 人员授权的
唯一管理真源。Casdoor `roles` claim、Casdoor role membership、`/userinfo`、introspection
和旧 access/ID token 均不得创建、续期、撤销或恢复 StuHelper 授权。

- grant/revoke 必须在同一 DB 事务写 desired state、`audit_events` 与
  `domain_event_outbox`；任一步失败整体回滚；
- 授予先写 `desired=granted, projection=pending, activated_at=NULL`，只有 OpenFGA 精确
  tuple 写入并验证后才能设置首次 `activated_at`、标记 `projection=applied` 并进入授权快照；
- 撤销先写 `desired=revoked`；授权快照和 Authorization Service 必须立即拒绝，随后才异步
  删除 OpenFGA tuple。不得等 token 到期、缓存 TTL 或 tuple 删除成功后才撤权；
- `projection_status` 只表示投影健康；已激活 grant 做 reconcile 时必须保留 `activated_at`，
  使 DB 管理面保持可恢复。新建/恢复 grant 仍必须在首次 verified projection 前 fail-closed；
- 每次 mutation 单调增加 grant revision。worker 只允许完成与 payload revision 相同的期望
  状态；并发 grant/revoke 和 outbox supersession 必须由 revision fencing 收敛；
- OpenFGA 删除使用完整 `user + relation + object`、higher-consistency 读取/验证和
  `on_missing=ignore`，保持重试与并发幂等；
- dead-letter 只能通过受审计的 replay/reconcile 恢复；reconciliation 从 DB desired state
  重建 OpenFGA，不能反向把未知 tuple 导入 DB；
- 每日 drift reconciliation 只精确检查 DB 已管理的 direct tuple；只重排 failed、超时
  pending 或实际不一致的 grant。超过修复阈值必须告警并停止自动修复；未知 tuple 不自动
  导入或删除。全量 rebuild 必须走受 `iam:grants:manage` 与 step-up MFA 保护的受审计 API；
- mutation 只接受 ADR-0008 固定的 role/scope 组合，不提供任意 tuple 写入 API；
- 最后一名 active `super_admin` 不得撤销，判断和 mutation 必须处于同一加锁事务。
- 生产首次 bootstrap 至少要求两名已存在于 `users` 的目标，并在一个 DB 事务中以 system
  actor 写全部 grant、审计和 outbox；任何一项失败全部回滚。账本已有 desired
  `super_admin` 时必须整体跳过，日常部署不得自动复活已撤销主体。

## School / Section Scope 完整性

DB 授权快照查询失败时必须 fail-closed 并返回依赖不可用；不得把网络、超时或服务端错误
降级成“没有 scope”。资源级 OpenFGA 查询失败时同样 fail-closed。

单个 `section` grant 的 object ID 无法按受支持的 review-moderation codec 解析时，应把该
grant 视为无效权限并忽略，不能让它授予 capability，也不能让同一用户的其他合法 scope
全部 503。每个被忽略的 grant 必须：

- 增加无 label 的 `iam_invalid_role_scope_total`，避免把外部 ID 引入指标基数；
- 记录包含内部 FGA user、固定 role 和无效 section ID 的 warning；
- 触发 `StuHelperInvalidOpenFGARoleScope` 告警，由值班人员定位并清理陈旧 tuple。

普通读路径不得自动删除 tuple。授权投影 worker 和 reconciliation 可以依据 DB desired state
精确删除受支持的管理员 tuple；任何不在 DB 账本且无法归属的 tuple 只告警，不在请求热路径
猜测或清理。

## 后台任务生命周期

业务模块不得用裸 `go fn()` 启动长期后台任务。长期任务必须通过 `Runtime.startBackgroundTask` 注册，以获得统一的 context cancellation、WaitGroup、启动/停止日志和 panic 记录。

- 模块级 `StartBackgroundJobs` 不允许 `nil` starter fallback；
- 缺少 starter 是启动期配置错误，应显式失败；
- 测试若需要同步执行任务，应传入测试 starter，而不是依赖生产 fallback；
- 单次 panic 可以被 runtime 记录，但不等同于任务可恢复，循环任务应在每次迭代边界处理 panic 并继续下一轮。

## Outbox Worker 语义

统一 outbox worker 必须按 per-job 隔离语义处理 batch。

- 单个 job 的 `process`、`markDone` 或 `markRetry` 失败不得阻止同 batch 后续 job；
- `process` panic 必须在 per-job 边界转成带 stack 的普通失败，复用相同的 retry、
  `dead_letter` 和 `outbox_job_failures_total` 路径；不得让 panic 越过整个 polling loop，
  也不得用无界 root supervisor 反复重启 poison job；
- `process` 成功但 `markDone` 失败必须记录 job ID、job type、错误和指标，并在 batch 末返回聚合错误；
- outbox consumer 应尽量幂等，但 worker 不能把所有副作用严格幂等作为隐藏前提；
- `failed` 状态只表示 retry-scheduled，不得用远未来 `available_at` 表达终止失败；
- 达到 `MaxAttempts` 后必须写入 `dead_letter`，并通过显式 replay API 重放。

## Retention Cleanup

线上可能无限增长的表不得使用单条无界大删除执行 retention cleanup。

- 默认使用 chunked delete，或用分区表按时间裁剪；
- chunk size 必须是命名常量或配置项；
- cleanup 每次循环应受 context cancellation 控制；
- SQL 必须参数化；如果需要动态条件，只能来自代码内白名单常量。

示例形状：

```sql
DELETE FROM audit_events
WHERE id IN (
  SELECT id
  FROM audit_events
  WHERE category = $1
    AND created_at < NOW() - make_interval(days => $2)
  ORDER BY created_at ASC
  LIMIT $3
)
```

## 审计写入上下文

请求链路内的审计事件必须使用 `audit.LogContext(ctx, event)`、`audit.LogFromGin(c, event)`、`audit.LogSuccessContext(...)` 或 `audit.LogFailureContext(...)`，不得先用 `audit.EventFromContext` 手动补字段后再调用无上下文 `audit.Log`。

- `audit.LogContext` 会从 context 补齐 `request_id` / `trace_id`；
- 持久化使用 `context.WithoutCancel(ctx)`，保留 trace baggage 和 request-scoped values，同时避免客户端断开导致安全审计丢失；
- 后台启动、bootstrap 等没有请求上下文的调用可以继续使用 `audit.Log`，但有业务 `ctx` 时必须优先传入。

## 敏感日志门禁

手机号、账号、OTP 相关链路不得把原始手机号写入日志字段。Go 代码中 phone/mobile 类 zap 字段必须先经过 `maskPhone`、`phoneutil.Mask`、`logger.MaskPhone` 或 `logger.MaskSensitiveData`。

- CI 通过 `scripts/check-semgrep-custom-rules.sh` 运行 `tools/semgrep/stuhelper-security.yml`；
- 新增手机号日志字段时，必须同步通过 custom Semgrep fixture 和 `server/internal` 扫描；
- 确需记录关联维度时，优先记录 hash、内部用户 ID 或已脱敏展示值。

## 边界标识规范化

用于限流、锁定、去重、缓存 key 或审计聚合的外部标识，必须在进入 key 之前 canonicalize。

- IP 地址使用 `net/netip.ParseAddr(...).String()`；
- 手机号和账号使用对应 normalize / hash helper；
- 学校、用户、资源 ID 使用内部 canonical ID，不使用展示名或 provider subject；
- 禁止只 `strings.TrimSpace` 后直接作为 Redis key、cache key 或 lock key 维度。

## Scope 术语

Casdoor JWT role claim 不解析、不参与 StuHelper 授权。角色和学校/资源范围来自 PostgreSQL
授权账本，OpenFGA 只承载可重建的运行时关系投影。

- 新代码使用 `school scope`、`section scope`、`ScopeSchoolIDs`、`ScopeSectionIDs` 和
  `ScopedRoleGrants`；
- `org` / `OrgScopedRoles` 属于历史 Zitadel 语义，运行时接口已经移除；OIDC 负向测试可以
  保留旧 claim 样例，用于证明 provider scope 不会进入授权上下文；
- 新业务授权必须通过 `CapabilityGrants`、DB 业务事实和 OpenFGA 资源关系表达；
- 禁止新增从 OIDC claim 解析 role、school 或 resource scope 的逻辑；
- `/auth/me`、后台导航和 API middleware 必须使用同一份 DB-derived access snapshot。
