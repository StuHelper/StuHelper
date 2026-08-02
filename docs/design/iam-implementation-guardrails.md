---
type: design
audience: backend-dev, ops
status: current
authoritative-source: server/internal/modules/auth/ + server/internal/modules/authorization/ + server/internal/modules/user/ + server/internal/platform/authorization/ + server/internal/pkg/fga/ + server/internal/pkg/outbox/ + server/internal/pkg/audit/ + server/migrations/000024_authorization_authority_cutover.*.sql + infra/ops/authorization-ledger-cutover.sh
last-verified: 2026-08-02
---

# IAM 实施守卫

本文记录 IAM、认证、会话、outbox、审计 retention 和后台任务的硬性实现边界。它不是产品 spec，也不替代源码和测试；它只约束容易跨模块漂移的工程不变量。

## Token 与 Session 顺序

`login`、`refresh`、`logout`、`logout-all` 和任何 session rotation 流程必须先完成服务端权威状态写入，再向客户端发放 token、cookie 或成功响应。

- refresh 获取或签发新 token 后，必须先完成 session store 的 token family 更新；
- OIDC refresh 在调用 provider token endpoint 前，必须先从受校验的本地 session 取出绑定的
  provider subject，并完成 Casdoor 服务端 user lookup。lookup 依赖不可用时不得消费/轮换
  provider refresh token，也不得清除仍有效的本地 session cookie；
- provider 返回的新 ID token subject 必须与 session 绑定 subject 完全一致。subject 改变时清除
  客户端会话并拒绝，不能把 refresh 变成换号登录；
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

## 授权权威、账本与 OpenFGA 投影

授权权威按角色拆分，禁止把两个来源混成同一个泛化的 role claim 模型：

- `super_admin` 的唯一管理权威是 Casdoor **目标 StuHelper organization 的用户对象**：
  `Owner == CASDOOR_ORGANIZATION`、`IsAdmin == true`，且用户未被 forbidden/deleted；
- PostgreSQL `authorization_grants` 保存上述事实的持久、可审计 serving projection，
  `source=casdoor_org_admin`；它不是一个可由 StuHelper 管理 API 独立修改的第二权威；
- `school_admin` 与 `section_*` 的唯一管理真源仍是 PostgreSQL 授权账本，
  `source=manual`；
- Casdoor JWT `roles` claim、普通 Casdoor role membership、`/userinfo` 或 introspection 中的
  role 列表仍不得创建、续期、撤销或恢复任何 StuHelper 授权。

首次把旧生产系统切到 PostgreSQL 授权账本时，必须使用一次性、可重试、失败关闭的切换门禁：

- migration 创建 durable singleton marker，初始为 `pending`；production / prod-parity 应用在
  marker 完成前拒绝启动授权路由和 worker；
- `infra/ops/authorization-ledger-cutover.sh` 必须在 migration、Casdoor bootstrap 和 OpenFGA
  model/bootstrap 之后、应用启动之前运行；它先创建受控快照，再在单个 DB 事务中写 grant、
  audit、outbox 和 completed marker；
- `super_admin` 只从目标 organization 的当前有效 `IsAdmin` 导入；旧 scoped operator 只有同时
  存在目标 organization 的遗留 Casdoor role membership 与对应 OpenFGA **direct tuple** 时才
  导入。这里读取 role membership 只是一次性迁移证据交集，不把 Casdoor role 恢复为运行时
  权威；
- OpenFGA direct tuple 读取必须使用 higher-consistency 并遍历全部 continuation token。未知
  subject、无法与 Casdoor 身份对应的 tuple、嵌套 role/group/domain、既有非空账本或来源冲突
  都必须中止切换，不得自动猜测、扩大权限或直接改 completed marker；
- fresh installation 可以用空 grant 集合完成 marker；已完成 marker 的重复执行只返回原 digest
  与数量，不重读旧权威、不重复写入；发布回滚保留已经导入的 grant、audit 与 marker。

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
- StuHelper grant API 必须拒绝手工创建或撤销 `super_admin`；管理员应在 Casdoor 的目标
  organization 中修改 `IsAdmin`，不使用 Casdoor role membership 替代；
- Web OIDC callback、native callback 与 refresh 必须通过服务端 Casdoor user lookup 同步当前
  `IsAdmin` 状态，并在同一 DB 事务写 grant/revoke、system audit 与 outbox；同步失败不得发放
  新 session/token；
- 若当前 `IsAdmin` 与 DB desired state 已一致，但 provider-managed grant 的
  `projection_status=failed`，同一同步路径必须通过既有 reconcile 逻辑增加 revision、清除终止
  错误、重排 outbox 并写 system audit；不得把“provider 状态没变”误判为无需恢复；
- DB 快照中已有 `super_admin` 的受保护请求必须实时复核 Casdoor 用户状态。Casdoor 不可用时
  fail-closed；检测到降权时先提交 DB revoke 围栏并重载快照，再由 outbox 异步删除 tuple；
- Koishi、service credential、后台任务等非 HTTP 可信入口，只要根据内部 user ID 做管理员
  capability 判定，也必须通过同一个 identity adapter 做上述实时复核；不得直接读取
  Authorization Service 的陈旧 DB snapshot 绕过降权门禁；
- Casdoor user lookup 的依赖故障必须返回 503，不得伪装成主体无效的 401；refresh 在该故障下
  不得清除现有客户端 session cookie。跨 organization、主体不存在或其他身份校验失败仍按
  不可信身份拒绝，不能借 503 分类放宽校验；
- Casdoor 晋升对既有 StuHelper session 不承诺瞬时生效：用户需重新登录或 refresh；新 grant
  只有在 OpenFGA verified projection 后才进入 access snapshot；
- Casdoor 降权可以撤销最后一个 `super_admin`。系统不要求第二个管理员，也不以“防锁死”为由
  覆盖 owner 在身份平面的明确决定；恢复路径是在 Casdoor 中重新设置组织管理员后登录/refresh。

## 特权 MFA 管理

管理端按风险分层执行认证强度策略：只读 dashboard 可以免除 5 分钟 freshness，但不能免除
MFA 本身。生产与 prod-parity 的 `/course/review/admin/stats` 必须同时满足活动 enrollment 和
当前认证会话携带的有效 MFA proof；proof 可以早于 5 分钟，但不得为空或来自未来。其余受保护
管理路由继续要求活动 enrollment 与最近 5 分钟的 step-up proof。路由注册必须显式区分
dashboard 与 privileged middleware，不能依赖 Gin group 注册顺序形成隐式例外。

`super_admin` 的 MFA reset 是单人授权操作，不是双人审批流程。实现中不得重新引入第二名
`super_admin`、reviewer user ID、reviewer role 或“至少两个管理员”作为 reset 前置条件。

- reset 操作者可以与目标用户相同，但请求仍必须通过对应 capability、最近 step-up MFA、目标角色
  校验与完整安全审计；不得因为取消 reviewer 就绕过这些现有门禁；
- `super_admin` 主动 disable 自己 MFA 与 reset 是不同风险动作：自行 disable 的禁令继续保留；
- reset / disable 的成功、拒绝与失败都必须记录 actor、target、目标角色、action、结果和原因，不记录
  factor secret、recovery code 或 token；
- 项目允许只有一个或暂时没有 `super_admin`，MFA 逻辑不得以本地管理员数量改变上述语义。

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

Casdoor JWT role claim 不解析、不参与 StuHelper 授权。`super_admin` 只来自目标 organization
用户对象的 `IsAdmin`，并投影到 PostgreSQL；其余角色和学校/资源范围来自 PostgreSQL 授权
账本。OpenFGA 只承载可重建的运行时关系投影。

- 新代码使用 `school scope`、`section scope`、`ScopeSchoolIDs`、`ScopeSectionIDs` 和
  `ScopedRoleGrants`；
- `org` / `OrgScopedRoles` 属于历史 Zitadel 语义，运行时接口已经移除；OIDC 负向测试可以
  保留旧 claim 样例，用于证明 provider scope 不会进入授权上下文；
- 新业务授权必须通过 `CapabilityGrants`、DB 业务事实和 OpenFGA 资源关系表达；
- 禁止新增从 OIDC claim 解析 role、school 或 resource scope 的逻辑；Casdoor `IsAdmin` 必须
  通过受限的服务端 user lookup 读取，并校验 organization owner 与禁用/删除状态；
- `/auth/me`、后台导航和 API middleware 必须使用同一份 DB-derived access snapshot。
