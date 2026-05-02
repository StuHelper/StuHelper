---
type: design
audience: backend-dev, ops
status: current
authoritative-source: server/internal/modules/auth/ + server/internal/pkg/outbox/ + server/internal/pkg/audit/
last-verified: 2026-05-02
---

# IAM 实施守卫

本文记录 IAM、认证、会话、outbox、审计 retention 和后台任务的硬性实现边界。它不是产品 spec，也不替代源码和测试；它只约束容易跨模块漂移的工程不变量。

## Token 与 Session 顺序

`login`、`refresh`、`logout`、`logout-all` 和任何 session rotation 流程必须先完成服务端权威状态写入，再向客户端发放 token、cookie 或成功响应。

- refresh 获取或签发新 token 后，必须先完成 session store 的 token family 更新；
- provider refresh token revoke / rotation 失败时，不得向客户端承诺成功；
- 本地 session 更新失败时，不得把新 access / refresh token 写入 `Set-Cookie` 或响应体；
- 失败注入测试必须覆盖 provider refresh 成功但本地 session rotate 失败的场景。

## 后台任务生命周期

业务模块不得用裸 `go fn()` 启动长期后台任务。长期任务必须通过 `Runtime.startBackgroundTask` 注册，以获得统一的 context cancellation、WaitGroup、启动/停止日志和 panic 记录。

- 模块级 `StartBackgroundJobs` 不允许 `nil` starter fallback；
- 缺少 starter 是启动期配置错误，应显式失败；
- 测试若需要同步执行任务，应传入测试 starter，而不是依赖生产 fallback；
- 单次 panic 可以被 runtime 记录，但不等同于任务可恢复，循环任务应在每次迭代边界处理 panic 并继续下一轮。

## Outbox Worker 语义

统一 outbox worker 必须按 per-job 隔离语义处理 batch。

- 单个 job 的 `process`、`markDone` 或 `markRetry` 失败不得阻止同 batch 后续 job；
- `process` 成功但 `markDone` 失败必须记录 job ID、job type、错误和指标，并在 batch 末返回聚合错误；
- outbox consumer 应尽量幂等，但 worker 不能把所有副作用严格幂等作为隐藏前提；
- `failed` 状态表示 retry-scheduled 或 terminal-delay，不等同于单独 DLQ 表。

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

## 边界标识规范化

用于限流、锁定、去重、缓存 key 或审计聚合的外部标识，必须在进入 key 之前 canonicalize。

- IP 地址使用 `net/netip.ParseAddr(...).String()`；
- 手机号和账号使用对应 normalize / hash helper；
- 学校、用户、资源 ID 使用内部 canonical ID，不使用展示名或 provider subject；
- 禁止只 `strings.TrimSpace` 后直接作为 Redis key、cache key 或 lock key 维度。

## Scope 术语

Casdoor JWT role claim 只承载扁平角色名。学校和资源范围来自 StuHelper DB / OpenFGA 投影，不从 Casdoor role 名或 token claim 中解析。

- 新代码使用 `school scope`、`section scope`、`ScopeSchoolIDs`、`ScopeSectionIDs`；
- `org` / `OrgScopedRoles` 属于历史 Zitadel 语义，只能作为迁移期内部兼容字段存在；
- 新业务授权必须通过 `CapabilityGrants`、DB 业务事实和 OpenFGA 资源关系表达；
- 禁止新增从 OIDC claim 解析 school/resource scope 的逻辑。
