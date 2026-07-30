---
type: design
audience: backend-dev, ops
status: current
authoritative-source: server/internal/modules/auth/ + server/internal/modules/user/repository_auth_sync.go + server/internal/pkg/fga/ + server/internal/pkg/outbox/ + server/internal/pkg/audit/
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

## 平台角色 OpenFGA 投影

Casdoor 的扁平 `super_admin` role 投影到 `ecosystem:stuhelper#super_admin` 时，必须区分角色
内容与角色来源。只有刚签发、完成验签、且明确包含结构合法 `roles` claim 的 ID token 才能
作为增删平台级 tuple 的权威输入。

- Web 登录、原生登录和 provider refresh 返回的新 ID token 可以触发 reconcile；
- `/auth/me`、旧 access token、`/userinfo` 或 introspection 结果只能参与当前请求授权和
  shadow profile 更新，不得重新授予或撤销平台级 tuple；
- claim 缺失、`null`、结构畸形或解析失败时必须跳过 tuple mutation，不能把空角色切片误当成
  撤权信号；
- 撤权前必须按完整 `user + relation + object` 精确读取 direct tuple，不能用可能包含 computed
  userset 的 `Check` 代替；安全撤权读取使用 higher consistency；
- 删除必须显式使用 OpenFGA `on_missing=ignore` 保持并发与重试幂等，OpenFGA 读取或写入失败时
  认证同步 fail-closed；实际撤权记录 `iam.role.revoke` 审计事件；
- 本规则只适用于平台级扁平角色；school/section scope 继续以 StuHelper DB / OpenFGA 业务投影
  为权威，不得从 Casdoor role 名推导。

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

Casdoor JWT role claim 只承载扁平角色名。学校和资源范围来自 StuHelper DB / OpenFGA 投影，不从 Casdoor role 名或 token claim 中解析。

- 新代码使用 `school scope`、`section scope`、`ScopeSchoolIDs`、`ScopeSectionIDs`；
- `org` / `OrgScopedRoles` 属于历史 Zitadel 语义，只能作为迁移期内部兼容字段存在；
- 新业务授权必须通过 `CapabilityGrants`、DB 业务事实和 OpenFGA 资源关系表达；
- 禁止新增从 OIDC claim 解析 school/resource scope 的逻辑。
