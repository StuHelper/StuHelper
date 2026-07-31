---
type: adr
audience: maintainers, backend-dev, ops
status: current
authoritative-source: server/migrations/ + server/internal/modules/authorization/ + server/internal/platform/authorization/
last-verified: 2026-07-31
---

# ADR-0008: PostgreSQL 授权控制面与 OpenFGA 运行时判定面

**Date**: 2026-07-31
**Status**: accepted
**Deciders**: 项目 owner

## Context

StuHelper 已使用 Casdoor OIDC 认证用户、Capability 表达功能入口，并使用 OpenFGA 判断资源关系。
此前管理员角色来自 Casdoor JWT 的扁平 `roles` claim，school / section 范围再从 OpenFGA
反查。这个模型能在缺少 scope 时安全拒绝，但没有受支持、可审计、可撤销、可重建的授权
生命周期，并形成了 Casdoor role、OpenFGA tuple 与业务数据库三方协同的问题。

本决策只选择长期逻辑架构；不以重构成本、实施时间或短期兼容性为决策依据。

## Decision

采用 StuHelper Authorization Control Plane：

1. PostgreSQL 的授权授予账本是管理员角色和 scope 的唯一管理真源（desired state）。
2. OpenFGA 是运行时关系判定面和可重建 serving projection，不是人员授权的管理真源。
3. Casdoor 只负责认证、会话、token 签发和登录层 MFA。Casdoor JWT 中的业务 role claim
   不参与任何 StuHelper 授权决策，Casdoor 也不再维护 `super_admin`、`school_admin`、
   `section_*`、`verified_student` 等 StuHelper 业务角色目录或 membership。
4. StuHelper Authorization Service 是业务模块唯一 PDP 入口。业务 handler 不解析 provider
   role，不直接构造 OpenFGA client；Capability、DB 事实、撤权栅栏与 OpenFGA 关系由该服务
   组合并统一 fail-closed。
5. 后台入口和 `/auth/me` 中的角色、Capability、scope 均从 DB 授权快照及 DB 业务事实派生，
   不从 IdP claim 派生。

## 授予与撤销状态机

授权变更必须在 PostgreSQL 事务中同时写入：

- `authorization_grants` 期望状态与单调递增 revision；
- `audit_events` 中的不可变安全审计；
- `domain_event_outbox` 中的 OpenFGA 精确 tuple 投影任务。

授予顺序：

```text
DB desired=granted, projection=pending + audit + outbox
  -> OpenFGA write / higher-consistency verify
  -> DB projection=applied
  -> 授权快照开始包含该 grant
```

撤销顺序：

```text
DB desired=revoked, projection=pending
  -> 撤权栅栏立即从授权快照排除该 grant
  -> OpenFGA delete with on_missing=ignore / verify absent
  -> DB projection=applied, revoked_at set
```

因此：

- 授予不会在 OpenFGA 投影成功前提前生效；
- 撤销在 DB 事务提交后立即拒绝，即使 OpenFGA 暂时不可用或仍有陈旧 tuple；
- outbox 重试、重复投递、并发 grant/revoke 由 grant revision 和 outbox revision fencing
  保证只有最新 desired state 可以完成；
- dead-letter 必须可显式重放，并由 reconciliation 从 DB 重建 OpenFGA。

## 固定授权类型

管理 API 只接受代码内固定的 role / scope 组合：

| Role | Scope type | OpenFGA direct tuple |
|------|------------|----------------------|
| `super_admin` | `ecosystem:stuhelper` | `user:<users.id>#super_admin@ecosystem:stuhelper` |
| `school_admin` | `school:<school_id>` | `user:<users.id>#admin@school:<school_id>` |
| `section_admin` | `section:<section_id>` | `user:<users.id>#section_admin@section:<section_id>` |
| `section_moderator` | `section:<section_id>` | `user:<users.id>#section_moderator@section:<section_id>` |
| `section_reviewer` | `section:<section_id>` | `user:<users.id>#section_reviewer@section:<section_id>` |

不提供任意 user / relation / object tuple 写 API。OpenFGA user 必须使用内部 `users.id`。
首版 section scope 只接受既有 review-moderation section codec。

## 管理面安全

- grant/list/revoke 需要全局 `iam:grants:manage` Capability；
- mutation 需要现有管理员 MFA enrollment 与 5 分钟 step-up proof；
- 原因必填，审计记录 actor、target、role、scope、revision、before/after 与 outcome；
- 不能撤销最后一个已生效且 desired=granted 的 `super_admin`；
- 重复 grant/revoke 是幂等成功，不产生扩大权限的旁路；
- DB、审计或 outbox 任一步失败，事务整体回滚；OpenFGA 故障不返回“授权已生效”。

## 一致性与缓存

授权快照可以在进程内或 Redis 做短 TTL 缓存，但缓存不是事实源。任何 grant/revoke 提交都必须
按 subject + revision 主动失效；撤权检查不得只依赖 TTL。资源 mutation 的 OpenFGA allow
还必须受 DB desired-state 撤权栅栏约束。

## 迁移与回滚

切换前必须先把现有直接 OpenFGA 管理员 tuple 导入 DB 账本并验证投影一致，至少保留两名
可用 `super_admin`。切换后：

- provider `roles` 仅可暂时作为遥测字段，不能参与 allow/deny；
- Casdoor 中遗留业务 role/membership 先冻结写入，再清理；
- 旧 Casdoor role-sync credential、worker、配置和 bootstrap catalog 在代码切换完成后移除。

若发布回滚，只回滚应用读取路径；DB 授权账本与审计不得删除或反向覆盖。回滚窗口内旧版本
若仍信任 Casdoor role，会违反本 ADR，因此生产切换必须使用兼容发布序列或维护窗口，不能把
重新启用 Casdoor role authority 当作正常回滚方案。

## Consequences

### Positive

- 授权所有权、审计、恢复和灾备都以 PostgreSQL 为中心，OpenFGA 可从 DB 重建。
- IdP compromise 或陈旧 role claim 不能直接产生 StuHelper 管理权限。
- 授予和撤销有明确的部分失败语义，撤权时效不依赖 token 到期或投影重试完成。
- 后台入口、Capability 与资源授权来自同一授权快照，不再形成 claim/tuple 双重权威。

### Negative

- 授权 mutation 需要事务、outbox worker、reconciliation、迁移工具和更严格的运维顺序。
- OpenFGA serving projection 与 DB desired state 之间存在可观测、可修复的最终一致窗口。
- 授权快照读取增加一次 DB/缓存依赖；依赖失败时受保护请求必须 fail-closed。

## Rejected Alternatives

- **OpenFGA tuple 作为唯一管理真源**：缺少与业务事务一致的审批、审计和灾后重建中心。
- **Casdoor role 或 metadata 承载 scope**：把业务授权事实放回 IdP，并重新引入陈旧 token
  与 membership 的双重权威。
- **受保护的运维清单作为长期真源**：适合 bootstrap，不适合多操作员、在线撤权和审计查询。
- **DB 与 Casdoor 双写业务角色**：Casdoor projection 对运行时决策没有必要，只增加故障域。
