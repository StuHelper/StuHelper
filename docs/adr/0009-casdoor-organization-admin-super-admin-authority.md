---
type: adr
audience: maintainers, backend-dev, ops
status: current
authoritative-source: server/migrations/000023_casdoor_org_admin_authority.*.sql + server/migrations/000024_authorization_authority_cutover.*.sql + server/internal/modules/authorization/ + server/internal/platform/casdoor/
last-verified: 2026-08-01
---

# ADR-0009: Casdoor 组织管理员作为 StuHelper `super_admin` 权威

**Date**: 2026-08-01
**Status**: accepted
**Deciders**: 项目 owner

## Context

ADR-0008 原先把所有管理员角色都放在 PostgreSQL 管理面，并增加“两名初始
`super_admin`”“不得撤销最后一名管理员”“`super_admin` MFA reset 必须由另一名
`super_admin` 复核”等约束。这些规则适合职责分离要求很高的组织，但不是 StuHelper owner
希望采用的治理模型：本项目允许只有一个组织管理员，并希望在 Casdoor 的 StuHelper
organization 中完成最高管理员身份的唯一管理。

普通 OIDC `roles` claim 仍不适合作为授权权威：它可能陈旧、缺少组织边界，也不能承载
StuHelper 的 school/section scope。因此本决策只为 Casdoor 用户对象上有明确语义和组织归属的
`IsAdmin` 建立一个窄例外，不恢复旧的 claim-driven RBAC。

## Decision

1. `CASDOOR_ORGANIZATION`（生产为 `stuhelper`）中，满足以下全部条件的用户自动拥有
   StuHelper `super_admin`：
   - Casdoor user `Owner == CASDOOR_ORGANIZATION`；
   - `IsAdmin == true`；
   - `IsForbidden == false`；
   - `IsDeleted == false`。
2. 该状态必须由 StuHelper 服务端使用最小权限 Casdoor user lookup credential 读取，不能从
   浏览器、OIDC `roles` claim、客户端请求参数或未校验的 `/userinfo` role 列表取得。
3. PostgreSQL `authorization_grants` 保存 `source=casdoor_org_admin` 的持久 serving projection，
   用于审计、撤权围栏、Capability 展开与 OpenFGA 重建；Casdoor `IsAdmin` 是管理权威，DB
   projection 不是可被手工 grant/revoke 的第二权威。
4. `school_admin` 与 `section_*` 继续以 PostgreSQL `source=manual` grant 为唯一管理真源。
   Casdoor 普通 role catalog、role membership 与 JWT `roles` claim 继续完全不参与授权。
5. Web OIDC callback、native callback 与 refresh 在发放新 session/token 前同步组织管理员状态。
   grant/revoke、system audit 与 outbox 必须在同一个 PostgreSQL 事务内提交；任一步失败，登录或
   refresh fail-closed。若 provider 状态未改变，但对应 grant/revoke 投影已进入终止失败，下一次
   登录或 refresh 必须复用受审计的 reconcile 状态机，增加 revision 并重新排入 outbox；不能让
   唯一组织管理员只能等待每日 reconciliation 或依赖另一个 StuHelper 管理员手工恢复。
6. 当 DB 快照当前含 `super_admin` 时，受保护请求实时复核 Casdoor 用户状态：
   - Casdoor 不可用或返回不可信结果时 fail-closed；
   - 若用户已降权/禁用/删除，先提交 DB revoke 围栏并重载 access snapshot，再异步删除
     OpenFGA tuple；不能等待 token 到期或 tuple 删除完成。
7. Casdoor 晋升通过下一次登录或 refresh 触发。新 grant 在 OpenFGA 首次写入并
   higher-consistency 验证前保持 pending，不能提前授予后台权限。
8. StuHelper 管理 API 禁止手工创建或撤销 `super_admin`；管理端只显示其来源并引导操作者去
   Casdoor 修改 organization administrator。
9. 项目不要求两个 `super_admin`，也不保留“最后一名管理员不可撤销”规则。Casdoor owner 可以
   让系统暂时没有 `super_admin`；恢复方式是在 Casdoor 中重新设置组织管理员并登录/refresh。
10. `super_admin` MFA reset 不要求另一名 `super_admin` 复核。一次授权充分的操作者即可执行，
    但现有身份确认、目标角色检查、step-up/能力门禁和不可变审计必须保留。为避免把 reset 与
    主动绕过本人 MFA 混为一谈，`super_admin` 自行 disable 自己 MFA 的禁令保持不变。

## Security boundaries

- 这是对 Casdoor **organization administrator flag** 的信任，不是对任意 Casdoor role 的信任。
- 查询结果必须同时校验 organization owner、forbidden/deleted 状态；跨 organization 的
  `IsAdmin` 不能获得 StuHelper 权限。
- 只有 DB 快照中的候选 `super_admin` 需要在受保护请求热路径做实时 lookup；普通用户和 scoped
  admin 不因此依赖 Casdoor 管理 API。
- Casdoor 管理平面或只读 lookup credential 被攻破将影响 StuHelper 最高权限，这是 owner 明确
  接受的集中治理风险；应通过 Casdoor MFA、最小权限 credential、审计和告警降低风险。
- Casdoor 短暂不可用会让候选 `super_admin` 的受保护请求失败。这是为了保证降权时效而接受的
  fail-closed 可用性取舍。

## Migration

迁移 `000023_casdoor_org_admin_authority` 为 grant 增加 `source` 并用数据库约束禁止
`super_admin/manual` 或 scoped role/provider source 的非法组合。迁移
`000024_authorization_authority_cutover` 增加一次性 pending/completed marker；发布脚本在应用
启动前读取当前 Casdoor organization `IsAdmin`，并把验证后的 `super_admin` 与 scoped legacy
交集写入账本、审计和 outbox。未完成 marker 时 production-like 应用拒绝启动，不能依赖用户
“下一次请求”再被动纠正首次切换造成的空窗。

旧的 `authorization-bootstrap` binary、运维脚本、`STUHELPER_INITIAL_SUPER_ADMINS` 配置和
双管理员合同测试被删除。生产初始化改为：设置 Casdoor organization administrator → 正常登录
或 refresh → 等待 authorization projection `applied` → 验证 Admin 与 step-up 链路。

## Consequences

### Positive

- 单个最高管理员在一个明确控制面中管理，避免 Casdoor 与 StuHelper 两处人工授权漂移。
- 不再强迫小型项目维护两个独立管理员账号，也不再为 MFA reset 引入超出 owner 需求的双人流程。
- 仍保留 DB 撤权围栏、事务审计、outbox 和 OpenFGA 可重建能力。
- 普通 Casdoor roles 和 scoped authorization 仍保持清晰隔离。

### Negative

- Casdoor 组织管理员配置成为最高权限控制面的关键依赖和高价值目标。
- 候选 `super_admin` 请求多一次 Casdoor lookup；Casdoor 故障会使该类请求 fail-closed。
- Casdoor 晋升不会让已有 session 在无 refresh 的情况下即时获得权限。
- 允许只有一个或零个 `super_admin` 意味着 owner 必须自行承担单账号恢复与可用性风险。

## Supersedes

本 ADR 修订 ADR-0008 中以下内容：

- `super_admin` 由 PostgreSQL 人工 grant 独立管理；
- 至少两个初始 `super_admin` 的 bootstrap；
- 最后一名 `super_admin` 不可撤销；
- `super_admin` MFA reset 必须由另一名 `super_admin` 二次确认。

ADR-0008 对 scoped roles、Capability、DB 事务审计/outbox、撤权围栏、OpenFGA projection 与
reconciliation 的决定继续有效。
