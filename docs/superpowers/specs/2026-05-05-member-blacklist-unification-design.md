---
type: design
audience: backend-dev, koishi-dev, frontend-dev, product
status: approved-for-spec-review
authoritative-source: server/api/openapi.yaml and server/migrations/000001_initial_schema.up.sql after implementation
created: 2026-05-05
---

# 成员黑名单统一设计

## 背景

当前项目存在多套成员黑名单真源：

| 来源 | 当前存储 | 当前用途 |
|---|---|---|
| admission 认证失败 | `group_admission_failures.blacklisted_at` | 多次进群未认证后阻止后续入群 |
| Koishi 手动黑名单 | `blacklist.json` | `config -b`、控制台添加、入群/好友申请拒绝 |
| Koishi `kick -b` | `blacklist.json` | 踢出并拉黑 |
| Koishi `kick_blacklist` | `blacklist.json` | 违规审核动作拉黑 |

这些机制名称相同但权威来源、作用范围和解除入口不同。管理员无法在一个地方回答“这个 QQ 为什么被拒绝入群”。本设计统一的是“成员黑名单”，即 QQ 用户或其他平台成员是否允许进入受 StuHelper 管理的群。auth/session token blacklist 属于登录态吊销，不纳入本设计。

## 目标

- 后端 PostgreSQL 成为成员黑名单唯一权威来源。
- 支持单群黑名单和全局黑名单。
- 所有 active 黑名单记录必须标注来源、原因、作用范围、创建入口和操作者。
- Koishi 命令、Koishi 控制台、admission 认证失败、moderation 违规处理都写入同一套后端 API。
- 入群申请判断由后端统一裁决，Koishi 不再把本地 JSON 当成业务真源。
- 后端 Admin 和 Koishi 控制台展示同一套黑名单数据。
- 当前仓库不保留旧运行态兼容；开发期 schema 直接重写，旧运行态可丢弃。若上线后再做此变更，另补显式导入脚本。

## 非目标

- 不统一 auth/session token blacklist。
- 不把警告、禁言、退群冷却、审核工单合并为黑名单。
- 不引入 `school` scope；学校维度黑名单是 v2。
- 不让 Koishi 在后端不可用时伪造黑名单写入成功。
- 不引入事件溯源作为第一版必需架构。
- 不保留 Koishi `blacklist.json` 作为长期写入路径。

## 统一边界

`server/` 保存和裁决：

- active、已过期和已解除黑名单记录。
- 某个成员能否进入某个群。
- 来源、原因、操作者、过期和解除信息。
- bot、Admin 和 Koishi 控制台所需 API。

`bots/koishi/` 执行和转发：

- 入群申请时调用后端准入接口。
- 命令或控制台新增黑名单时调用后端写入接口。
- `kick -b` 和 moderation `kick_blacklist` 先执行 QQ 踢出，再写入后端黑名单；写入失败必须向操作者暴露。
- admission `blacklist` action 仍调用 `kickGuildMember(guildID, qqID, true)`，用于 QQ 平台侧拒绝再次加入。

## 作用范围

黑名单支持两种 scope：

- `guild`: 单群黑名单。只阻止成员进入指定 `guild_id`。
- `global`: 全局黑名单。阻止成员进入所有受 StuHelper 管理的群。

入群准入判断顺序：

1. 查询 active 全局黑名单。
2. 查询 active 当前群黑名单。
3. 任一命中则阻止入群或禁止 auto-approve。
4. 如果同时命中全局和单群，API 决策优先返回全局记录，详情接口可展示全部命中记录。

同一 subject 可以同时存在 global 和 guild active 记录。Admin 列表显示为两条记录。解除同一 subject 全部 scope 需要前端按 scope 分别调用 release，不提供 release-all 单接口，避免误操作。

## 数据模型

新增后端权威表 `member_blacklist_entries`。

| 字段 | 说明 |
|---|---|
| `id` | 记录 ID |
| `platform` | 平台，例如 `qq` |
| `subject_type` | 成员类型，第一版使用 `qq_user` |
| `subject_id` | QQ 号或平台成员 ID |
| `scope_type` | `guild` 或 `global` |
| `guild_id` | `scope_type='guild'` 时必填；`global` 时为空 |
| `source` | 来源分类 |
| `reason_code` | 机器可读创建原因 |
| `reason_text` | 人类可读创建原因 |
| `created_by_type` | `system`、`admin_user`、`qq_operator`、`service_account` |
| `created_by_id` | 操作者 ID；系统动作固定为 `system` |
| `created_from` | `admission_worker`、`qq_command`、`koishi_console`、`admin_console`、`moderation_review`、`migration_script` |
| `expires_at` | 过期时间；为空表示永久 |
| `released_at` | 解除时间 |
| `released_by_type` | 解除操作者类型 |
| `released_by_id` | 解除操作者 ID |
| `release_reason_code` | 机器可读解除原因 |
| `release_reason` | 人类可读解除原因 |
| `metadata` | JSONB，上下文数据 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

active 定义：

```sql
released_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
```

DB 约束和查询实现：

- DB CHECK 约束：`scope_type='guild'` 时 `guild_id` 必填；`scope_type='global'` 时 `guild_id IS NULL`；`platform`、`subject_id`、`source`、`reason_code` 非空。
- 准入查询是纯读路径，只按 active 条件过滤，不写表。
- 写路径在创建前于同一事务释放同 subject + scope 已过期未释放记录，`release_reason_code='policy_expired_auto'`。
- 后台 worker 定期 sweep 到期记录，执行 `policy_expired_auto`、重置 admission 失败计数和写审计。
- active 唯一性用两个 partial unique index 实现：global 记录按 `(platform, subject_type, subject_id) WHERE released_at IS NULL AND scope_type='global'`，guild 记录按 `(platform, subject_type, subject_id, guild_id) WHERE released_at IS NULL AND scope_type='guild'`。
- 过期但未 sweep 的记录会挡住 insert，所以写路径必须先 release expired row。
- 准入查询主路径使用 `(platform, subject_type, subject_id, scope_type, guild_id) WHERE released_at IS NULL` 索引，并在查询条件中排除 `expires_at <= NOW()`。

`group_admission_failures` 继续作为 admission 失败计数表，但不再作为黑名单权威表。达到阈值时，admission 服务创建 `member_blacklist_entries` 记录。

## 来源、原因和 metadata

第一版固定这些创建枚举：

| `source` | `reason_code` | 默认范围 | 创建入口 | 必填 metadata |
|---|---|---|---|---|
| `admission_failure` | `admission_timeout_limit` | `guild` | `admission_worker` | `admissionSessionID`, `failureCount`, `failedJoinLimit`, `platform`, `guildID`, `botSelfID` |
| `manual_admin` | `manual_blacklist` | 操作者选择 | `admin_console`、`koishi_console` 或 `qq_command` | `operatorInput`, `scopeSelectionContext` |
| `kick_blacklist` | `manual_kick_blacklist` | 默认 `guild`，可显式 `global` | `qq_command` | `rawCommand`, `targetGuildID`, `operatorQQID` |
| `moderation_action` | `violation_review_blacklist` | 默认 `guild`，可显式 `global` | `moderation_review` | `reviewID`, `workItemID`, `targetGuildID` |
| `migration_legacy_koishi` | `legacy_koishi_blacklist` | `global`，除非旧记录可确定群 | 一次性导入脚本 | `legacyFile`, `legacyUserID` |
| `migration_admission_failure` | `legacy_admission_blacklist` | `guild` | 一次性导入脚本 | `legacyFailureCount`, `legacyGuildID` |

`scopeSelectionContext` 记录操作者如何选择 scope，例如 `current_guild_command`、`admin_console_form`、`koishi_console_form` 或 `explicit_global_flag`。
当 `manual_admin` 由 QQ 命令创建时，metadata 还必须包含 `operatorQQID`，服务端用它填充 `created_by_id`；Koishi Console 创建时操作者上下文保存在 `consoleAuthID`，但后端审计 actor 仍是 Koishi runtime service account。

解除原因枚举：

| `release_reason_code` | 说明 |
|---|---|
| `manual_pardon` | 管理员手动宽恕并解除 |
| `release_only` | 管理员仅解除黑名单，不表达宽恕语义 |
| `policy_expired_auto` | 到期自动解除 |
| `admission_appeal_passed` | 申诉通过 |
| `migration_inverse` | 数据整理或导入回滚 |

系统自动解除使用 `released_by_type='system'` 和 `released_by_id='system'`。

展示文案由前端根据枚举和文本组合。后端 API 必须返回原始枚举，不只返回 `blacklisted`。

## API 设计

### 准入判断

Bot 侧准入接口：

```text
GET /api/v1/bot/member-blacklist/access?platform=qq&subjectType=qq_user&guildID=<guild>&subjectID=<qq>
```

blocked 响应必须包含 `canJoin=false`、`decision='blocked'` 和 `matchedBlacklist`；`matchedBlacklist` 至少包含 id、platform、subjectType、subjectID、scopeType、source、reasonCode、reasonText、expiresAt，guild scope 还必须包含 guildID。

列表接口 `GET /api/v1/admin/member-blacklist` 和 `GET /api/v1/bot/member-blacklist` 必须分页，并支持 subject、scope、source、guild、createdByID 和 active/released/expired 状态过滤；默认只返回 active 记录。

### 写入黑名单

```text
POST /api/v1/admin/member-blacklist
POST /api/v1/bot/member-blacklist
```

请求体必须包含 `platform`、`subjectType`、`subjectID`、`scopeType`、`source`、`reasonCode`、`reasonText`、`expiresAt` 和 `metadata`。`scopeType='guild'` 时必须包含 `guildID`。`expiresAt` 为空表示永久；非空时必须晚于服务端当前时间。后端根据认证上下文填充 `created_by_type` 和 `created_by_id`，不信任客户端伪造操作者。Bot API 的 `manual_admin` 来源必须在顶层传入结构化 `createdFrom`（`qq_command` 或 `koishi_console`），服务端按入口和 source 校验；`metadata.createdFrom` 一律不参与审计字段判定。

服务端必须按调用入口校验 `source`，违反矩阵返回 400：

| `source` | admin API | bot API | 内部 service |
|---|---|---|---|
| `admission_failure` | 禁止 | 禁止 | 允许 |
| `manual_admin` | 允许 | 仅 `koishi_console` | 不需要 |
| `kick_blacklist` | 禁止 | 允许 | 不需要 |
| `moderation_action` | 禁止 | 允许 | 不需要 |
| `migration_*` | 禁止 | 禁止 | 仅脚本 |

Admin API 必须要求成员黑名单读/写 capability；bot API 必须要求独立 service-account scope，并把 QQ 命令或 Koishi 控制台操作者写入 metadata。

### 解除黑名单

```text
POST /api/v1/admin/member-blacklist/{id}/release
POST /api/v1/bot/member-blacklist/{id}/release
POST /api/v1/admin/member-blacklist/release-by-subject
POST /api/v1/bot/member-blacklist/release-by-subject
```

UI 场景必须优先使用 `/{id}/release`。`release-by-subject` 用于 QQ 命令等只知道 QQ 号的场景。按 subject 解除时必须提供 `platform`、`subjectType`、`subjectID`、`scopeType`，单群解除还必须提供 `guildID`。不允许只传 QQ 号就隐式解除所有范围。

待移除旧路由：

- `GET /api/v1/bot/admission/qq-users/{qqID}/access`
- `POST /api/v1/bot/admission/blacklist/{qqID}/release`
- `POST /api/v1/admin/admission/blacklist/{qqID}/release`

## admission 集成

admission 继续维护失败计数和 pending action：

1. session 超时后，后端根据 `group_admission_failures.failure_count + 1` 判断是否达到 `failed_join_limit`。
2. 未达到阈值时下发 `kick` action。
3. 达到阈值时下发 `blacklist` action。
4. Koishi 成功执行 `blacklist` action 后上报成功事件。
5. 后端在同一事务中更新 session、递增失败计数，并创建或复用 active 单群黑名单记录。

重复上报同一 session 的成功 kick 或 blacklist 事件不得重复增加失败次数或重复创建黑名单。

admission 失败计数与 release 联动：

- 手动解除 `source='admission_failure'` 的黑名单时，默认使用 `manual_pardon`，并将同一 `platform + guild_id + qq_id` 的 `failure_count` 重置为 0。
- 如管理员选择 `release_only`，只解除黑名单，保留失败计数；下一次 admission 失败可能立即再次拉黑。
- `policy_expired_auto` 自动解除 admission_failure 黑名单时，重置对应失败计数为 0。

## Koishi 集成

Koishi core 调整：

- `event-handlers.ts` 的入群申请黑名单判断改为调用后端统一准入接口。
- `blacklist.json` 不再作为写入真源。
- 控制台黑名单页面改为调用后端 list/create/release API。
- `config -b` 命令改为后端写入和解除。
- `kick -b` 执行 QQ 踢出后调用后端创建 `source='kick_blacklist'` 的黑名单记录。
- moderation `kick_blacklist` 调用同一创建接口，使用 `source='moderation_action'`。

命令默认 scope：

- 在群内执行且未指定 scope：默认单群。
- 在私聊或控制台执行：必须显式选择单群或全局。
- 全局拉黑必须显式使用 `--global` 或 UI 中的全局选项。

准入接口读取失败策略：

- Koishi 调用后端准入接口使用短超时，建议 800ms。
- 超时或后端不可用时，不伪造“已拒绝”，也不伪造“已通过”。
- 该次入群申请不在 request 阶段做黑名单决策，继续走 admission session 创建、禁言和认证流程。
- 后续 admission worker 再按后端权威黑名单和认证状态执行最终裁决。

Koishi 对黑名单写入失败必须返回明确错误，不允许只完成 QQ 踢出却静默宣称已拉黑。

## 展示和审计

Admin 和 Koishi 控制台黑名单列表至少展示 QQ、scope、群号或群名、来源、原因、创建入口、创建人、创建时间、过期时间和解除状态。用户画像页可以聚合展示黑名单、警告、禁言、退群冷却、admission 失败次数和审核事件，但黑名单仍以 `member_blacklist_entries` 为权威。

审计事件类型：

- `member_blacklist.created`
- `member_blacklist.released`

审计 payload 必须包含 source、reason_code、scope、subject、operator、release_reason_code 和 metadata 摘要。

## 测试

后端测试覆盖：

- 单群黑名单只阻止指定群。
- 全局黑名单阻止所有群。
- 同时命中全局和单群时决策返回全局。
- active 唯一性防止重复创建。
- global 唯一索引不得因 `guild_id IS NULL` 允许重复 active 记录。
- 准入查询不写表。
- 过期记录自动释放后可再次创建。
- 列表接口分页和过滤有效。
- 调用入口与 `source` 矩阵不匹配时返回 400。
- release-by-subject 必须指定 scope。
- `/{id}/release` 与 release-by-subject 均记录 release_reason_code。
- admission 达到失败阈值后创建单群黑名单。
- admission manual_pardon 和 policy_expired_auto 重置失败计数。
- admission release_only 保留失败计数。
- 重复 bot success event 不重复增加失败次数。
- 过期黑名单不阻止入群。

Koishi 测试覆盖：

- 入群申请调用统一准入接口并按单群/全局决策拒绝。
- 准入接口超时时不拒绝、不批准，继续 admission 流程。
- `config -b`、控制台新增、控制台解除均调用后端 API。
- `kick -b` 写入 `kick_blacklist` 来源。
- moderation `kick_blacklist` 写入 `moderation_action` 来源。
- 后端写入失败时命令返回失败，不产生假成功提示。

契约测试覆盖：

- OpenAPI schema 包含 source、reason、release reason、scope 和 matched blacklist。
- shared TS 类型与 OpenAPI 同步。
- Koishi shared client 请求路径和 body 与后端一致。

## 实施约束

- 改接口必须先改 OpenAPI，再生成 Go 和 TypeScript 类型。
- 数据库 schema 以 `server/migrations/000001_initial_schema.up.sql` 为最终初始化文件；不新增兼容迁移。
- SQL 只写在 repository 层。
- 后端 service 负责准入裁决、过期释放和事务编排。
- Koishi 不再直接修改 `blacklist.json` 作为业务黑名单。
- 任何黑名单写入失败都必须显式暴露，不能静默降级。
