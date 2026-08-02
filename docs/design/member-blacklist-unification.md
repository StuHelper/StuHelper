---
type: design
audience: backend-dev, frontend-dev, product, maintainers
status: current
authoritative-source: server/api/openapi.yaml + server/migrations/ + server/internal/modules/admission/ + bots/koishi/plugins/stuhelper-core/
last-verified: 2026-08-02
---

# 成员黑名单架构

本文说明当前成员黑名单的权威边界和跨服务行为。API 字段以
`server/api/openapi.yaml` 为准，数据库结构以 `server/migrations/` 为准，运行时行为以源码和
测试为准。

## 权威边界

PostgreSQL `member_blacklist_entries` 是 QQ 用户等平台成员能否进入受管群的唯一业务真源。
登录态 token blacklist 属于认证会话撤销，不在本模型内。

- `server/internal/modules/admission/` 负责准入裁决、创建、解除、到期处理和审计；
- StuHelper Admin 通过 `/api/v1/admin/member-blacklist*` 管理记录；
- Koishi 通过 `/api/v1/bot/member-blacklist*` 查询和写入，不保存第二份业务真源；
- QQ 平台侧的踢出/拒绝再次加入是执行结果，不替代 PostgreSQL 记录；
- `group_admission_failures` 只保存 admission 失败计数，不再表示黑名单状态。

系统不再使用 Koishi `blacklist.json` 作为成员黑名单写入路径。`config -b`、`kick -b`、
moderation `kick_blacklist`、Koishi Console 和 Admin Console 最终都调用后端权威 API。

## 数据与作用范围

每条记录包含平台、subject、作用范围、来源、原因、操作者、创建入口、到期和解除信息。

支持两种范围：

- `global`：阻止 subject 进入所有受管群，`guild_id` 必须为空；
- `guild`：只阻止 subject 进入指定群，`guild_id` 必须存在。

active 的运行时定义为：

```sql
released_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
```

同一 subject 可以同时存在 global 和 guild active 记录。准入时先检查 global，再检查目标群；
两者同时命中时，决策优先返回 global 记录。数据库用独立 partial unique index 防止同一 scope
出现重复 active 行，并用访问索引支持热路径查询。

到期但尚未被 worker 标记 released 的行不参与准入。创建同 scope 新记录前，服务在同一事务中
先把已到期旧行标记为 `policy_expired_auto`，避免 partial unique index 阻塞新记录。

## 来源与审计主体

受支持的创建来源固定为：

| `source` | 典型入口 | 默认范围 |
|---|---|---|
| `admission_failure` | admission worker 达到失败阈值 | guild |
| `manual_admin` | Admin / Koishi Console | 操作者显式选择 |
| `kick_blacklist` | QQ `kick -b` 命令 | guild，除非显式 global |
| `moderation_action` | moderation 复核动作 | guild，除非显式 global |
| `migration_legacy_koishi` | 受控历史导入 | 按导入证据 |
| `migration_admission_failure` | 受控历史导入 | guild |

后端根据已验证的调用入口填充 `created_by_type`、`created_by_id` 和 `created_from`，不信任请求
伪造操作者。Admin API、bot API 和内部 service 各自只能使用允许的 source 组合；违反矩阵时返回
400。Bot 入口的 QQ 操作者或 Console 上下文只作为受控 metadata 和审计证据，不改变 service
credential 的认证主体。

创建和解除分别写 `member_blacklist.created` 与 `member_blacklist.released` 审计事件。审计包含
source、reason code、scope、subject、operator、release reason 和受限 metadata 摘要，不记录
凭据。

## API 与授权

Admin API：

- `GET /api/v1/admin/member-blacklist`
- `POST /api/v1/admin/member-blacklist`
- `POST /api/v1/admin/member-blacklist/{id}/release`
- `POST /api/v1/admin/member-blacklist/release-by-subject`

Bot API：

- `GET /api/v1/bot/member-blacklist/access`
- `GET /api/v1/bot/member-blacklist`
- `POST /api/v1/bot/member-blacklist`
- `POST /api/v1/bot/member-blacklist/{id}/release`
- `POST /api/v1/bot/member-blacklist/release-by-subject`

Admin 路由使用成员黑名单 capability；bot 路由使用独立 service credential scope。列表接口分页
并支持 subject、scope、source、guild 和状态过滤。UI 已知记录 ID 时使用 `/{id}/release`；
`release-by-subject` 只用于命令等没有记录 ID 的场景，而且必须显式给出 scope，不能只凭 QQ 号
解除所有范围。

## Admission 联动

Admission session 超时后，服务在同一事务中推进 session、失败计数和权威黑名单：

1. 未达到策略阈值时产生 `kick` action；
2. 达到阈值时产生 `blacklist` action；
3. Koishi 执行平台动作并回写结果；
4. 服务按 session/attempt 幂等更新，不重复增加失败次数或创建记录。

解除 `source=admission_failure` 的记录时：

- `manual_pardon` 与 `policy_expired_auto` 会重置同一群的 admission 失败计数；
- `release_only` 只解除记录，保留计数；下一次失败仍可能重新触发黑名单；
- 到期 worker 使用有界批次处理过期行，并通过统一后台任务生命周期响应 shutdown。

## Koishi 故障语义

Koishi 在入群申请阶段通过后端准入 API 判断 global/guild 黑名单。短暂超时或后端不可用时，
Koishi 既不伪造“允许”，也不伪造“拒绝”；该请求继续进入 admission session、禁言和认证流程，
由后续权威状态决定最终动作。

所有写入遵循“平台动作成功不等于后端黑名单成功”：

- `kick -b` 或 moderation 踢出后，后端写入失败必须向操作者报告部分失败；
- Console 和 `config -b` 的创建/解除失败不能显示成功；
- 私聊或 Console 没有当前群上下文时必须显式选择 guild 或 global；
- global 是高影响范围，必须由显式参数或 UI 选择触发。

## 展示与验证

Admin 和 Koishi Console 都从后端读取，并展示 subject、scope、群、来源、原因、创建入口、
操作者、创建/到期时间和解除状态。展示层可以聚合 admission 失败、警告和 moderation 记录，但
不能反向推导或覆盖黑名单权威状态。

回归测试至少保护：

- global/guild 决策优先级和 active 唯一性；
- 过期行不阻止准入，并能安全重建同 scope 记录；
- source/入口矩阵和 scope 完整性；
- 创建、解除、到期、manual pardon 的事务与审计；
- admission 回写幂等和失败计数联动；
- Koishi 准入依赖故障的保守流程；
- `config -b`、`kick -b`、moderation 和两套 Console 都走后端；
- 后端写入失败不会产生假成功提示；
- OpenAPI、Go/TypeScript 生成物和调用路径保持同步。

生产验收仍需使用真实 QQ 群和受控测试账号确认平台踢出、拒绝再次加入、后端记录与审计事件
四者一致；本地数据库和 mock bot 测试不能替代该证据。
