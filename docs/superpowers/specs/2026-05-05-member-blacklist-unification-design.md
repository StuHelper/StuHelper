---
type: design
audience: backend-dev, koishi-dev, frontend-dev, product
status: approved-for-spec-review
authoritative-source: server/api/openapi.yaml and server/migrations/000001_initial_schema.up.sql after implementation
created: 2026-05-05
---

# 成员黑名单统一设计

## 背景

当前项目里存在多套名为黑名单的机制：

- admission 入群认证失败后写入后端 `group_admission_failures.blacklisted_at`。
- Koishi core 使用本地 `blacklist.json` 处理手动黑名单、`kick -b` 和控制台黑名单。
- Koishi 违规处理的 `kick_blacklist` 最终写入 Koishi 本地黑名单。
- auth/session 的 token blacklist 用于登录态吊销。

这些机制名称相同但权威来源、作用范围和解除入口不同。结果是管理员无法在一个地方看到“这个 QQ 为什么被拒绝入群”，也无法明确区分单群拉黑、全局拉黑、认证失败拉黑和人工拉黑。

本设计统一的是“成员黑名单”，即 QQ 用户或其他平台成员是否允许进入受 StuHelper 管理的群。auth/session token blacklist 属于登录态安全机制，不纳入本设计。

## 目标

- 建立一套后端权威的成员黑名单系统。
- 支持单群黑名单和全局黑名单。
- 所有 active 黑名单记录必须标注来源、原因、作用范围、创建入口和操作者。
- Koishi 命令、Koishi 控制台、admission 认证失败、moderation 违规处理都写入同一套后端 API。
- 入群申请判断由后端统一裁决，Koishi 不再把本地 JSON 当成业务真源。
- 后端 Admin 和 Koishi 控制台展示同一套黑名单数据。
- 因本仓库当前不要求保留历史迁移兼容，数据库实现直接更新初始化 schema；如需迁移旧运行态数据，提供显式一次性导入脚本，不作为隐式兼容路径。

## 非目标

- 不统一 auth/session token blacklist。
- 不把用户警告、禁言、退群冷却、审核工单等全部合并为黑名单；这些仍是独立事实，可在用户画像页聚合展示。
- 不让 Koishi 在后端不可用时伪造黑名单写入成功。
- 不引入事件溯源作为第一版必需架构；审计事件仍通过既有审计能力记录。
- 不保留 Koishi `blacklist.json` 作为长期写入路径。

## 统一边界

`server/` 是成员黑名单权威来源：

- 保存 active 和已解除黑名单记录。
- 决定某个成员能否进入某个群。
- 记录来源、原因、操作者、过期和解除信息。
- 暴露 bot、Admin 和 Koishi 控制台所需 API。

`bots/koishi/` 是执行端和入口端：

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

## 数据模型

新增后端权威表 `member_blacklist_entries`。

核心字段：

| 字段 | 说明 |
|---|---|
| `id` | 记录 ID |
| `platform` | 平台，例如 `qq` |
| `subject_type` | 成员类型，第一版使用 `qq_user` |
| `subject_id` | QQ 号或平台成员 ID |
| `scope_type` | `guild` 或 `global` |
| `guild_id` | `scope_type='guild'` 时必填；`global` 时为空 |
| `source` | 来源分类 |
| `reason_code` | 机器可读原因 |
| `reason_text` | 人类可读原因 |
| `created_by_type` | `system`、`admin_user`、`qq_operator`、`bot` |
| `created_by_id` | 操作者 ID；系统动作可为空或固定为 `system` |
| `created_from` | `admission_worker`、`qq_command`、`koishi_console`、`admin_console`、`moderation_review` |
| `expires_at` | 过期时间；为空表示永久 |
| `released_at` | 解除时间 |
| `released_by_type` | 解除操作者类型 |
| `released_by_id` | 解除操作者 ID |
| `release_reason` | 解除原因 |
| `metadata` | JSONB，保存 admission session、failure count、原始命令、审核单 ID 等上下文 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

active 唯一性：

- 全局 active 记录：同一 `platform + subject_type + subject_id + scope_type='global'` 只能有一条。
- 单群 active 记录：同一 `platform + subject_type + subject_id + scope_type='guild' + guild_id` 只能有一条。

`group_admission_failures` 继续保留为 admission 失败计数表，但不再作为黑名单权威表。达到阈值时，admission 服务创建 `member_blacklist_entries` 记录。

## 来源和原因

统一系统必须标注来源和原因，第一版固定以下枚举：

| `source` | `reason_code` | 默认范围 | 创建入口 | 说明 |
|---|---|---|---|---|
| `admission_failure` | `admission_timeout_limit` | `guild` | `admission_worker` | 连续多次进群认证超时 |
| `manual_admin` | `manual_blacklist` | 操作者选择 | `admin_console` 或 `koishi_console` | 管理员手动加入黑名单 |
| `kick_blacklist` | `manual_kick_blacklist` | 默认 `guild`，可显式 `global` | `qq_command` | 管理员踢出并拉黑 |
| `moderation_action` | `violation_review_blacklist` | 默认 `guild`，可显式 `global` | `moderation_review` | 违规审核动作要求踢出并拉黑 |
| `migration_legacy_koishi` | `legacy_koishi_blacklist` | `global`，除非旧记录可确定群 | 一次性导入脚本 | 从 Koishi `blacklist.json` 导入 |
| `migration_admission_failure` | `legacy_admission_blacklist` | `guild` | 一次性导入脚本 | 从既有 active admission blacklist 导入 |

展示文案由前端根据 `source`、`reason_code` 和 `reason_text` 组合。后端 API 必须返回原始枚举，不只返回 `blacklisted`。

## API 设计

### 准入判断

Bot 侧准入接口应返回明确决策：

```json
{
  "canJoin": false,
  "decision": "blocked",
  "matchedBlacklist": {
    "id": "blk_xxx",
    "platform": "qq",
    "subjectType": "qq_user",
    "subjectID": "10001",
    "scopeType": "guild",
    "guildID": "123456",
    "source": "admission_failure",
    "reasonCode": "admission_timeout_limit",
    "reasonText": "连续 3 次入群认证超时",
    "expiresAt": null
  }
}
```

查询参数必须包含当前群上下文：

```text
GET /api/v1/bot/member-blacklist/access?platform=qq&guildID=<guild>&subjectID=<qq>
```

旧 `GET /api/v1/bot/admission/qq-users/{qqID}/access` 不再作为业务入口；Koishi 调用点迁移到统一准入接口，后端发布前移除旧路由。

### 写入黑名单

创建接口：

```text
POST /api/v1/admin/member-blacklist
POST /api/v1/bot/member-blacklist
```

请求体必须包含：

- `platform`
- `subjectType`
- `subjectID`
- `scopeType`
- `guildID` when `scopeType='guild'`
- `source`
- `reasonCode`
- `reasonText`
- `expiresAt`
- `metadata`

后端根据认证上下文填充 `created_by_type`、`created_by_id` 和 `created_from`，不信任客户端伪造操作者。

### 解除黑名单

解除接口必须显式指定 scope：

```text
POST /api/v1/admin/member-blacklist/{id}/release
POST /api/v1/bot/member-blacklist/{id}/release
POST /api/v1/admin/member-blacklist/release-by-subject
```

按 subject 解除时必须提供：

- `platform`
- `subjectType`
- `subjectID`
- `scopeType`
- `guildID` when `scopeType='guild'`

不允许只传 QQ 号就隐式解除所有范围的黑名单。解除全部范围需要独立的显式操作和确认。

## admission 集成

admission 继续维护失败计数和 pending action：

1. 用户进群后产生 admission session。
2. session 超时后，后端根据 `group_admission_failures.failure_count + 1` 判断是否达到 `failed_join_limit`。
3. 未达到阈值时，下发 `kick` action。
4. 达到阈值时，下发 `blacklist` action。
5. Koishi 成功执行 `blacklist` action 后上报成功事件。
6. 后端在同一事务中更新 session、递增失败计数，并创建或复用 active 单群黑名单记录。

`admission_failure` 黑名单 metadata 至少包含：

- `admissionSessionID`
- `failureCount`
- `failedJoinLimit`
- `platform`
- `guildID`
- `botSelfID`

重复上报同一 session 的成功 kick 或 blacklist 事件不得重复增加失败计数或重复创建黑名单。

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

Koishi 对后端写入失败必须返回明确错误，不允许只完成 QQ 踢出却静默宣称已拉黑。

## 展示和审计

Admin 和 Koishi 控制台黑名单列表至少展示：

- QQ / subject ID
- scope：单群或全局
- 群号或群名
- 来源
- 原因
- 创建入口
- 创建人
- 创建时间
- 过期时间
- 解除状态

用户画像页可以聚合展示黑名单、警告、禁言、退群冷却、admission 失败次数和审核事件，但黑名单仍以 `member_blacklist_entries` 为权威。

所有创建和解除操作写审计事件。审计 payload 必须包含 source、reason_code、scope、subject、operator 和 metadata 摘要。

## 测试

后端测试覆盖：

- 单群黑名单只阻止指定群。
- 全局黑名单阻止所有群。
- 同时命中全局和单群时决策返回全局。
- active 唯一性防止重复创建。
- release-by-subject 必须指定 scope。
- admission 达到失败阈值后创建单群黑名单。
- 重复 bot success event 不重复增加失败次数。
- 过期黑名单不阻止入群。

Koishi 测试覆盖：

- 入群申请调用统一准入接口并按单群/全局决策拒绝。
- `config -b`、控制台新增、控制台解除均调用后端 API。
- `kick -b` 写入 `kick_blacklist` 来源。
- moderation `kick_blacklist` 写入 `moderation_action` 来源。
- 后端写入失败时命令返回失败，不产生假成功提示。

契约测试覆盖：

- OpenAPI schema 包含 source、reason、scope 和 matched blacklist。
- shared TS 类型与 OpenAPI 同步。
- Koishi shared client 请求路径和 body 与后端一致。

## 实施约束

- 改接口必须先改 OpenAPI，再生成 Go 和 TypeScript 类型。
- 数据库 schema 以 `server/migrations/000001_initial_schema.up.sql` 为最终初始化文件；不新增兼容迁移。
- SQL 只写在 repository 层。
- 后端 service 负责准入裁决和事务编排。
- Koishi 不再直接修改 `blacklist.json` 作为业务黑名单。
- 任何黑名单写入失败都必须显式暴露，不能静默降级。
