# StuHelper Koishi 服务端插件深度分析报告

分析范围：`bots/koishi` 下 `plugins/stuhelper-core/src`、`plugins/stuhelper-group-guard/src`、`plugins/stuhelper-binding/src`、`plugins/stuhelper-admin/src`、`packages/shared/src`、`plugins/test-utils`（含其直接依赖 `packages/moderation-core` 的架构性引用）。非测试源码合计约 26,000 行，测试约 22,000 行。

---

## 1. 功能清单

### 1.1 stuhelper-core（群管中心 / WebUI 宿主）

入口 `plugins/stuhelper-core/src/index.ts:32-38`，按 5 步装配：

| 能力 | 实现 | 说明 |
|---|---|---|
| 核心服务 | `core/services/stuhelper-group-center.service.ts:47`（注册为 `ctx.stuhelperGroupCenter`，`inject: ['database']`） | 聚合 DataManager（JSON 文件存储）、SettingsManager、CacheService、AuthService、模块注册表 |
| Console 前端入口 | `setup/register-console-entry.ts`（`consoleCtx.console.addEntry(...)`） | 挂载 WebUI |
| Console WebSocket API | `setup/register-console-api.ts:27-38` → `core/api/index.ts:30-70` | 注册约 **75 个** authority=4 的 console 事件（`stuhelperGroupCenter/*`），覆盖 config/auth/warns/subscriptions/stats/logs/settings/各 runtime-settings/keyword-rules/cache/chat/blacklist |
| 页面域 API | `core/api/page-api.ts:23-46` | dashboard / identity / review / config-governance / entity-profile 五个页面数据端点，带 guild 范围裁剪（`page-scope.ts`） |
| 工作项动作 API | `core/api/review-actions.ts:19-42`（`action/review`、`action/work-item`）、`core/api/governance-actions.ts:55-73`（保存命令策略 / guard 模板，zod 校验） | 复核执行/驳回、admission approve/deny/defer、report dismiss/escalate/create-review（`review-action-handler.ts`） |
| 聊天面板 | `core/api/chat-api.ts:22-42`（guild-members/guild-info/user-info/send/recall/image-fetch）+ `core/api/chat-message-broadcast.ts:106-114`（监听 `message`/`send` 事件向 console 客户端实时推送） | 推送时按角色→guild 范围过滤（`chat-delivery.ts`） |
| 后台任务 | `review-claim-recovery.ts:64-100` | 每 60s 恢复卡死的 approved 复核（5 分钟 stale 阈值，置 `stuck_manual` 并写 critical 事件） |
| 旧运行时模块 | `setup/register-runtime-modules.ts:8-10` | **已停用，仅打日志**；`core/modules/` 22 个模块（banme/dice/antirecall/AI 等）不再被实例化 |
| 安全门 | `console-auth.ts:6-16` | 启动时校验 `STUHELPER_CONSOLE_ADMIN_PASSWORD` 非空且非占位符 |

数据模型：自有 JSON 文件存储 12 个（`core/data/data.service.ts:29-42`：warns/groupConfig/mutes/banmeRecords/lockedNames/antiRepeat/subscriptions/recallRecords/commandLogs/leaveRecords/authRoles/authUsers）+ `cache.json` + `settings.json`；同时通过 console API 写共享数据库表（见 §2.4）。

### 1.2 stuhelper-group-guard（入群认证 + 消息治理执行端）

入口 `plugins/stuhelper-group-guard/src/index.ts:55-191`。

| 能力 | 实现 |
|---|---|
| 事件监听 | `events.ts:21-27` `guild-member-added`（post-join 禁言+认证提醒）、`guild-member-request`（join-request 审批策略）；`events.ts:31-49` `message`/`message-deleted`（moderation 开关受 WebUI 控制） |
| 入群处理策略 | `post-join-guard-strategy.ts:75-139`（建后端 admission session→禁言→发认证链接→记事件，后端不可用时本地兜底禁言 `failClosedBackendUnavailableJoin:309-340`）；`join-request-review-strategy.ts:28-67`（请求后端决策 approve/reject 并回执） |
| 定时任务 | `events.ts:52-61` 兜底扫描（默认 300s，扫 backendSyncPending 成员 + claim 队列动作 + 转发新生材料）；`index.ts:166-168` 每 max(60, scanInterval)s 从后端同步 admission policy targets 到本地 GuardPolicyStore（`guard-policy-bootstrap.ts:38-66`） |
| SSE 推送 | `admission-action-stream.ts:33-137` 每个 QQ bot 一条 `/admission/actions/stream` 长连接，收到 remind/release/kick/blacklist 动作即执行，断线按 `reconnectDelaySeconds` 重连 |
| 动作执行 | `admission-actions.ts:31-54`（remind/release/kick/blacklist），边界校验 `admission-action-boundary.ts:28-39`（platform/botSelfID/guildID/本地 policy 四重校验） |
| 黑名单拒入 | `member-blacklist-rejection.ts:27-58`（命中 `admission.member_blacklisted` 即踢出并记事件） |
| 新生材料转发 | `freshman-forward.ts:25-52`（拉 pending-forward → 发图+摘要到管理群 → 回执 forwarded） |
| 公开命令 | `commands.ts:35-93`：`举报`（AI 审核 `report-service.ts:60-78`）、`骰子`、`抽禁言`（保底机制） |
| 管理员命令 | `admission-admin-commands.ts:71-200`：`查询入群认证`、`重发认证链接`、`重新生成认证链接`、`跳过入群认证`、`清空入群未认证次数`、`解除入群拉黑` |
| Console API | `admission-console-api.ts:40-42,86-104`：`stuhelperGroupGuard/page/admission-runtime`（运行态页面数据）、`action/admission-member`（6 种成员操作）、`action/save-admission-runtime-settings` |
| 并发控制 | `admission-subject-coordinator.ts`（按 subject 串行化 + 取消标记）、`admission-reminder-deduper.ts`（30s 提醒去重） |

### 1.3 stuhelper-binding（QQ 绑定）

单文件 `plugins/stuhelper-binding/src/index.ts:31-65`：一个 `ctx.middleware`，私聊中解析绑定命令（命令字由 WebUI runtime settings 控制），调 `POST /api/v1/bot/qq-binding/consume`，按 HTTP 状态码映射 9 种文案（`index.ts:102-122`）。

### 1.4 stuhelper-admin（管理员命令）

入口 `plugins/stuhelper-admin/src/index.ts:32-65`。

- 群审命令（`commands.ts:34-42`）：`群审状态`、`群审警告`、`群审复核`、`群审禁言`（批量）、`群审踢人申请`、`群审拉黑申请`（后两者写 moderation 复核队列）。
- 新生审核命令（`admission-review-commands.ts:30-41`）：`新生审核查看/通过/驳回`、`新生黑名单解除`，全部代理到 Go 后端 freshman API，403 错误码细分回复（`admission-review-commands.ts:225-243`）。
- 权限：Koishi authority + moderation-core 命令策略双重校验（`command-access.ts:15-42`），开关由 WebUI runtime settings 控制。

### 1.5 packages/shared（跨插件契约层）

- **PlatformClient**（`platform/index.ts:92-136`）：对 Go 后端全部 22 个 bot API 的类型化封装（含 SSE 解析 `platform/index.ts:330-382`）、`PlatformAPIError`、`${{ env.X }}` 占位符解析（`platform/index.ts:428-445`）。
- **数据模型注册 + Store**：`guard/member-model.ts`（守护成员表）、`guard/policy.ts`（模板/绑定表）、`guard/runtime-settings.ts`、`guard/ai-settings.ts`、`guard/behavior-settings.ts`、`guard/message-settings.ts`、`admin/runtime-settings.ts`、`binding/runtime-settings.ts`、`guard/member-work-item-store.ts`、`guard/member-admin-store.ts`。
- **文案系统**：`message-template.ts`（约 250 条默认中文模板 + `renderMessageTemplate`）、`command-description-sync.ts`（把 runtime 文案同步到 i18n 命令描述）。
- **配置 Schema**：`config/index.ts`（四个插件的 koishi Schema）。

### 1.6 plugins/test-utils

`runtime.ts:8-29`：统一管理测试中 koishi fork 的注册/释放，规避 plugin-mock 在 `root.stop()` 时的清理缺陷。

---

## 2. 架构图

### 2.1 插件/包依赖

```
koishi runtime (koishi.yml group:stuhelper, 4 插件均启用)
│
├─ stuhelper-core ──────┬──> @stuhelper/koishi-shared
│   (WebUI 宿主)         └──> @stuhelper/koishi-moderation-core
├─ stuhelper-group-guard┬──> shared
│   (执行端)             └──> moderation-core
├─ stuhelper-admin ─────┬──> shared
│                       └──> moderation-core
├─ stuhelper-binding ─────> shared
│
├─ @stuhelper/koishi-moderation-core ──> shared (store.ts:5 引 GUARD_MEMBER_TABLE)
└─ @stuhelper/koishi-shared ──> koishi（无其他依赖）
```

无循环依赖（shared 是底座，moderation-core 居中，插件在顶层）。插件之间**没有直接代码依赖**，但通过两条隐式通道耦合：共享数据库表（§2.4）和 Go 后端。

### 2.2 与 Go 后端（server/）的 HTTP 对接点

唯一出口是 `packages/shared/src/platform/index.ts`（路径常量 35-47 行）+ `platform/freshman-client.ts:7-8`。鉴权：`Authorization: Bearer <serviceToken>`（`platform/index.ts:469-473`），默认 8s 超时（`platform/index.ts:50,475-480`）。

| 端点 | 方法 | 调用方 |
|---|---|---|
| `/health/live` | GET | （定义于 `platform/index.ts:140-146`，当前无活动调用方） |
| `/api/v1/bot/qq-binding/consume` | POST | binding `index.ts:47-50` |
| `/api/v1/bot/qq-users/{qq}/verification` | GET | core 身份页 `core/api/identity-profile-lookup.ts`（经 `page-api-runtime.ts:55-58`） |
| `/api/v1/bot/admission/sessions` | POST | group-guard 入群建会话 `post-join-guard-strategy.ts:258-280`、兜底同步 `member-guard.ts:199-238` |
| `/api/v1/bot/admission/sessions/member`（+`/resend`、`/regenerate`、`/skip`） | GET/POST | admission 管理命令 `admission-admin-commands.ts` 与 console `admission-console-api.ts` |
| `/api/v1/bot/admission/failures/reset` | POST | 同上 |
| `/api/v1/bot/admission/join-requests/decision`、`/events` | POST | join-request 策略 `join-request-review-strategy.ts:31-37`、`member-guard.ts:347-358` |
| `/api/v1/bot/admission/policies/targets` | GET | group-guard 定时同步 `index.ts:175-182` |
| `/api/v1/bot/admission/sessions/pending`、`/actions/claim` | GET/POST | 兜底扫描 `member-guard.ts:170-186` |
| `/api/v1/bot/admission/actions/stream`（SSE） | GET | `admission-action-stream.ts:103-115` |
| `/api/v1/bot/admission/sessions/{id}/events`、`/actions/{id}/events` | POST | 动作回执 `member-guard.ts:339-345` |
| `/api/v1/bot/member-blacklist`（+`/access`、`/{id}/release`、`/release-by-subject`） | GET/POST | core 黑名单 console API（`member-blacklist-console-api.ts`）、entity 页 `page-api-runtime.ts:214-230`、group-guard/admin 解黑命令 |
| `/api/v1/bot/admission/freshman/applications/pending-forward`、`/{id}/forwarded`、`/{id}/view`、`/{id}/review` | GET/POST | group-guard 转发 `member-guard.ts:401-417`、admin 新生命令 `admission-review-commands.ts` |

### 2.3 Console 集成方式

- core 通过 `console.addEntry` 挂前端，所有数据走 console WebSocket 事件（非 HTTP），统一 authority=4（`core/api/authority-listener.ts:4`）。
- 范围控制：`console-guild-scope.ts:25-69` 把 console 账号 → koishi `binding` 表 → core 自有 JSON 角色（`auth_roles.json`）→ guildIds 集合；页面/聊天/动作 API 全部经此裁剪。
- group-guard 自行注册 3 个 `stuhelperGroupGuard/*` 事件（`admission-console-api.ts:40-42`），与 core 的 entry 共用一个前端。

### 2.4 数据库表（koishi database / SQLite）

| 所有者 | 表 |
|---|---|
| shared(guard) | `stuhelper_guard_member`、`stuhelper_guard_template`、`stuhelper_guard_group_binding`、`stuhelper_admission_runtime_settings`、`stuhelper_group_guard_ai_settings`、`stuhelper_group_guard_behavior_settings`、`stuhelper_group_guard_message_settings` |
| shared(admin/binding) | `stuhelper_admin_runtime_settings`、`stuhelper_binding_runtime_settings` |
| moderation-core | `stuhelper_moderation_event/review/message_ledger/warning/keyword_rule/member_role/command_policy/fun_profile/report`（`packages/moderation-core/src/constants.ts:1-9`） |

**关键流向**：core 的 console API 写这些 settings 表（`core/api/index.ts:62-66`），binding/admin/group-guard 在运行时读 → WebUI 是全部插件的控制面，数据库表是事实上的跨插件 RPC 总线。

---

## 3. 耦合与边界问题

### 3.1 大体量死代码：core 的旧运行时模块（最严重）

- `setup/register-runtime-modules.ts:8-10` 是 no-op（"旧群管运行时模块已停用"）；`runtime/registry.ts:52` 的 `getRuntimeModules` **只被测试引用**（`runtime/registry.test.ts:6`、`runtime-contract.test.ts:180`）。
- 但 `core/modules/` 仍保留 **11,166 行源码 + 3,380 行测试**（22 个模块：banme/dice/antirecall/AI/keyword/welcome/orderManage 等），且 `core/index.ts:15` 仍 `export * from './modules'`。
- 活代码对死目录有 4 处真实依赖：`core/api/logs-api.ts:6-7`、`core/api/log-module-lookup.ts:2-3`、`core/api/stats-api.ts:7`（仅取 `CommandLogRecord` 类型与 redaction/normalize 工具）、`stuhelper-group-center.service.ts:17`（`log-redaction`）。这把整个死目录"钉"在了构建图里。
- 连锁死代码：`StuhelperGroupCenterService.registerModule/initModules/getAllModules`（service.ts:102-140）运行时模块表恒为空，`page-api-runtime.ts` 的 `loadModuleStates` 永远返回 `[]`；`DataManager` 的 mutes/banmeRecords/lockedNames/antiRepeat/leaveRecords 等存储（`data.service.ts:99-174`）只有死模块写入。

### 3.2 admission 命令与 console API 之间约 200 行复制粘贴

`admission-admin-commands.ts` 与 `admission-console-api.ts` 完整复制了以下函数（逐对行号）：

| 函数 | console 版 | 命令版 |
|---|---|---|
| `formatAdmissionSessionSummary` | admission-console-api.ts:595-613 | admission-admin-commands.ts:409-427 |
| `statusLabel` | 725-745 | 668-688 |
| `nextAdmissionStep` | 699-723 | 642-666 |
| `studentVerificationLabel` | 685-697 | 628-640 |
| `describeDeadline` | 656-672 | 599-615 |
| `isQQLinked`/`hasLinkedUser` | 674-683 | 617-626 |
| `reminderDeadline` | 584-593 | 588-597 |
| `compactRenderedMessage` | 747-753 | 690-696 |
| `skipAdmissionSessionOrUseCancelled` | 415-435 | 203-219 |
| `isAdmissionInvalidStateError` | 648-654 | 405-407 |

两份实现已经开始漂移（console 版 `formatAdmissionConsoleActionError:619-646` 额外做了双重错误形状检查），这正是复制粘贴腐化的早期信号。

### 3.3 console 范围解析逻辑双份实现

`console-guild-scope.ts:25-69`（resolveConsoleGuildScope）与 `chat-delivery.ts:80-122`（buildClientScope）是同一段"console 账号 → binding → roleIds → guildIds"逻辑的两份拷贝，包括相同的"找不到角色就 throw"和"空角色集 = 全局范围"语义。改一处漏一处的风险高。

### 3.4 core 成为所有插件的设置面板（职责泄漏，方向性问题）

`core/api/index.ts:62-66` 注册了 admin/binding/group-guard 三个插件的 runtime-settings API。core 必须理解其他每个插件的设置语义；其他插件不装时这些 API 仍然存在并可写表。这是有意的"WebUI 控制面"设计，但边界没有显式化——core 对 group-guard 的耦合还体现在 `page-api-runtime.ts:96-99,150-156` 直接读 `GUARD_MEMBER_TABLE` 与 `GuardPolicyStore`，绕过了 group-guard 插件本身。

### 3.5 moderation-core 反向引用 guard 表

`packages/moderation-core/src/store.ts:5,356-371`：`getOverview` 直接查 `GUARD_MEMBER_TABLE`。moderation（消息治理）层向 admission（入群守护）层伸手，分层被打穿了一格。

### 3.6 God file（>500 行非测试源码）

| 文件 | 行数 |
|---|---|
| `plugins/stuhelper-group-guard/src/admission-console-api.ts` | 764 |
| `plugins/stuhelper-group-guard/src/admission-admin-commands.ts` | 696 |
| `plugins/stuhelper-group-guard/src/member-guard.ts` | 552 |

（测试侧超 500 行的有 `member-guard.test.ts` 1478、`chat-scope.test.ts` 1156、`commands.test.ts` 932、`review-actions.test.ts` 861、`admission-console-api.test.ts` 648、`packages/shared/src/platform/index.test.ts` 522；`packages/shared/src/platform/index.ts` 495 行逼近阈值。）前两个 god file 的体量主要来自 §3.2 的重复，去重后可同时解决。

### 3.7 双轨制数据/权限体系

- **数据**：core 同时维护 JSON 文件存储（warns/groupConfig 等，`data.service.ts`）和数据库表；entity/config 页面把两边缝在一起（`page-api-runtime.ts:160-167` 读 `service.data.groupConfig`、`:196` 读 `service.data.warns`，同函数其余字段全部来自数据库/后端）。
- **权限**：三套并行——koishi authority、core 自有 JSON 角色系统（`auth.service.ts`，用于 console 范围）、moderation-core 的 member_role + command_policy（用于聊天命令）。`commands.ts:174-200`（group-guard）与 `command-access.ts:15-42`（admin）又各自实现了一遍"取 policy + roles + authority → canExecuteCommand"的组合逻辑。

---

## 4. 代码质量问题

整体水准较高（全仓库源码无 `any`、无 `console.log`、仅 1 处 `as unknown as`），以下是具体扣分项：

### 4.1 用运行时 duck-typing 削弱已有的静态类型

- `member-guard.ts:451-456`：把 `guardStore` 强转成"可能有 `findByAdmissionSessionID`"再 `typeof === 'function'` 检查——但 `GuardMemberStore` 本来就有这个方法（`packages/shared/src/guard/member-store.ts`）。同样模式见 `member-guard.ts:522-530`（`getActiveByID`）、`post-join-guard-strategy.ts:172-176`（`getAdmissionSessionByMember`）、`post-join-guard-strategy.ts:244-252`。这些分支只为容忍测试里的不完整 fake 而存在，把编译期保证降级成了运行期猜测，并产生了"方法不存在则默认放行（return true）"这类隐藏语义。
- `admission-console-api.ts:638-645,652-654`：`instanceof PlatformAPIError` 失败后再按 `error.name === 'PlatformAPIError'` 二次判断——这是对"同一类可能被打包成两个实例"的补丁，应在构建/解析层解决而不是在每个调用点写两遍。
- `chat-message-broadcast.ts:158-162`：`session.bot || ctx.bots.find(...)` 加 `typeof bot?.getMessage !== 'function'`，同类模式重复出现在 209、236、292 行。

### 4.2 吞错

- `cache.service.ts:68-69, 114-115, 162-163`：三处空 `catch (e)`（"继续尝试下一个 bot"），连 debug 日志都没有，bot 全部失败时无法定位原因。
- `log-module-lookup.ts:36-42`：`safeGetLogModule` 空 catch 返回 undefined。
- `json.store.ts:150-152 + 167-173`：`markDirty` 用 `setTimeout(() => this.flush())` 延迟写盘，而 `flush()` 在写失败时 **throw**——异常发生在裸 timer 回调里，等于未捕获异常（Node 默认会击穿进程）。

### 4.3 可变共享状态

- `json.store.ts:78-80`：`getAll()` 直接返回内部 `this.data` 引用；`cache.service.ts` 的惯用法是 `const data = this.store.getAll(); data.guilds[id] = info; this.store.setAll(data)`——原地修改共享对象后再"set 回去"，任何持有旧引用的读者都会看到中途状态。与全局不可变性原则直接冲突。
- `chat-message-broadcast.ts:283-289` 的 `enrichAtElements` 做对了（拷贝后改），但 295-303 行 `fillAtElementName` 仍直接 `element.attrs.name = ...` 改入参。

### 4.4 魔法数字 / 重复常量

- console 权限阈值 `4` 在 **6 个文件**各自定义：`page-api.ts:21`、`authority-listener.ts:4`、`chat-delivery.ts:6`、`console-guild-scope.ts:5`、`review-actions.ts:34,42`（裸字面量）、`governance-actions.ts:67,73`（裸字面量）、`admission-console-api.ts:43`。
- 30 秒去重窗口三处独立定义：`member-guard.ts:60`、`admission-admin-commands.ts:37`、`admission-reminder-deduper.ts`（构造默认 30_000）。
- `data.service.ts:204`：`date.setHours(date.getHours() + 8)` 手写 UTC+8 时区换算后再走 `toISOString`，产出的是"假 ISO"时间戳；而 `stuhelper-group-center.utils.ts` 已有 `formatShanghaiTimestamp`，两套时间格式化并存。
- `admission-action-stream.ts:108`：SSE 订阅 `limit: 50` 裸字面量。

### 4.5 安全相关弱点

- **fail-open 的 console 范围**：`console-guild-scope.ts:44-46,60-66`——authority≥4 且**没有任何角色分配**的账号直接获得 `kind:'all'` 全局范围（chat-delivery.ts:96-99,113-116 同样语义）。当前依赖"只有管理员能到 authority 4"这一前提，一旦 auth 配置失误即全量泄露所有群数据。
- **AI 审核结果零校验**：`report-service.ts:283-307` 对任意可配置 endpoint 的响应仅做 `payload.severity || 'none'`，随后 `applyAIResult`（report-service.ts:186-204）依据该值**自动禁言**（medium）或进入踢人/拉黑队列（high）。响应畸形会静默降级为 none；endpoint 被劫持则可批量制造处罚。没有 zod 校验（governance-actions.ts 证明仓库已有 zod 依赖与使用惯例）。
- `chat-image-fetch.ts:45-63`：`createChatImageAccessRegistry` 的 `entries` Map 只增不减——既是内存泄漏（§5.6），也意味着图片访问授权永不过期。

### 4.6 测试覆盖盲区

测试基建很好（node:test + sqlite + plugin-mock 集成测试），但以下核心逻辑无直接测试：

- **`admission-admin-commands.ts`（696 行）**：grep 全仓库，仅被 `index.ts` 引用，没有任何 `.test.ts` 导入它。6 个管理员命令的去重、跳过/重生成的本地记录同步、错误映射全部裸奔。对比之下功能等价的 console 版有 648 行测试。
- **`settings.manager.ts`（262 行）**：含文件监视、防抖重载、diff 持久化逻辑，零测试。
- **`cache.service.ts`（293 行）**：仅有 16 行 contract test（`cache.service.contract.test.ts`）。
- **`auth.service.ts`（295 行）**：仅周边的 `auth-permissions`/`auth-guild-admin` 有测试，角色 CRUD/内置角色初始化无测试。
- **`chat-api.ts`（259 行）**：send/recall/guild-members 处理器本体无测试（范围裁剪部分由 `chat-scope.test.ts` 覆盖）。
- `member-guard-effects.ts`、`join-request-review-strategy.ts`、`member-blacklist-rejection.ts`：仅经由 1478 行的 `member-guard.test.ts` 间接覆盖，定位失败成本高。

---

## 5. 性能与效率问题

### 5.1 每条群消息触发 5+ 次数据库往返（最高优先级）

moderation 开启后，单条消息的路径是：

1. `events.ts:31-39`：`runtimeSettings.isModerationEnabled()` → 全量 settings 行查询（`runtime-settings.ts` 的 `getSettings` 每次都打 DB，无缓存）；
2. `message-guard.ts:63-75`：`saveMessage` 先查后写（`moderation-core/store.ts:65-81`，2 次操作）；
3. `message-guard.ts:124`：`listKeywordRules` → **关键词规则全表扫描后 JS 过滤**（store.ts:153-156）；
4. `message-guard.ts:232`：`listRecentMessages` → **该群消息全表拉取 + 内存排序后 slice**（store.ts:83-86），表越大越慢；
5. `getMessages()`/`getModerationSettings()` 各再打一次 DB（message-guard.ts:284-292）。

所有 settings store（runtime/behavior/message/ai，`packages/shared/src/guard/*.ts` 与 admin/binding 同构）都是"每次调用一次 DB 读"，没有任何 TTL/失效缓存。

### 5.2 message ledger 无限增长

`stuhelper_moderation_message_ledger` 没有任何修剪/TTL 机制（moderation-core 全文检索确认仅有 `markMessageDeleted`），叠加 §5.1-4 的全量扫描，性能随运行时间单调劣化。`stuhelper_moderation_event` 同理（`listRecentEvents` store.ts:60-63 也是全表扫描 + 内存排序，被 dashboard/review 页和 `getOverview` 反复调用）。

### 5.3 聊天广播的逐消息远程调用

`chat-message-broadcast.ts` 对**每条**消息：guild 名称/头像缺失时调 `bot.getGuild`（194-218 行，无缓存，仅 bot 登录信息有缓存 79 行）；每个 `at` 元素调 `bot.getGuildMember`（291-303 行，无缓存）；每个 console 客户端做一次 `database.get('binding')`（chat-delivery.ts:88，缓存仅在单条消息内有效）。core 自己的 `CacheService` 明明缓存了 guild/user/member 信息却未在此复用。

### 5.4 跨 HTTP 边界的 N+1

- **身份页**：每个成员一次 `GET /qq-users/{id}/verification`（`page-api-runtime.ts:55-58` + `identity-profile-lookup.ts`）。虽有 60s TTL + 并发 8 的本地缓解，根因是后端缺批量端点。
- **entity 页**：每次请求把后端**整个活动黑名单**分页拉完（`page-api-runtime.ts:214-230` `loadActiveMemberBlacklists` 无限循环翻页），无缓存。
- **兜底扫描**：每个 backendSyncPending 记录一次 `POST /admission/sessions`（member-guard.ts:188-197），每轮每条。

### 5.5 page-api-runtime 绕过索引查询

`page-api-runtime.ts:235-241`：`listActiveGuardMembers` 先 `database.get(GUARD_MEMBER_TABLE, {})` 拉全表再 JS 过滤 released/kicked，而 `GuardMemberStore.listActive`（member-store.ts）已有条件下推的版本——同仓库两种写法，慢的那种被 dashboard 和 review 两个页面用。

### 5.6 内存增长点

- `chat-image-fetch.ts:45-63`：图片访问注册表永不清理（每张聊天图片一条，直到重启）。
- `logs-api.ts` 每次搜索把全部命令日志载入内存过滤分页（`log-module-lookup.ts:23-30` 读整个 JSON store 并 reverse）。

### 5.7 推送/轮询设计（总体合理，两处缺口）

SSE 主通道 + 可开关兜底轮询（300s）+ claim 防重，设计是对的。缺口：

- **SSE 无心跳超时**：`platform/index.ts:330-361` 的读循环没有 inactivity timer，服务端半开连接（TCP 假活）时 `reader.read()` 永久挂起，`onError` 不触发，重连逻辑（admission-action-stream.ts:121-136）失效，且日志静默。
- **新生转发队列单点阻塞**：`member-guard.ts:410-416` 顺序 for 循环中任何一个 `forwardFreshmanMaterial` 抛错都会中断整轮（错误冒泡到 events.ts:58 的 catch），其后的待转发项全部延后；坏项每轮都失败 → 永久阻塞队列。

---

## 6. 改进机会（按优先级）

1. **删除 core 旧运行时模块**（高，~14.5k 行）：先把 `log-redaction.ts`、`command-log-records.ts`、`log.module` 的 `CommandLogRecord` 类型迁出 `core/modules/`（活代码仅依赖这 3 件，见 §3.1），再整体删除 `core/modules/`、`runtime/`、`core/index.ts:15` 的 re-export，连带清理 `StuhelperGroupCenterService` 的模块注册表（service.ts:102-140）与 `DataManager` 中仅死模块使用的存储。理由：项目处于"重写优于打补丁"阶段（项目方针），这是最大的认知负担来源。

2. **提取 admission 展示/错误层**（高）：新建 `plugins/stuhelper-group-guard/src/admission-summary.ts`（或并入 `admission-format.ts`），收编 §3.2 列出的 10 组重复函数，`admission-console-api.ts` 与 `admission-admin-commands.ts` 双双降到 500 行以内。

3. **给 settings store 加缓存**（高）：在 `packages/shared/src/guard/runtime-settings.ts`（及同构的 behavior/message/ai/admin/binding store）内加"读缓存 + save 时失效"（单行表，缓存极简单），一次性消除 §5.1 中每消息/每命令的重复 DB 读。WebUI 保存路径都经过同一 store 实例，失效是可靠的。

4. **moderation-core 查询下推 + 修剪任务**（高）：`store.ts:60-63/83-86/153-156` 改用 driver 的 sort/limit/条件查询；为 message_ledger 和 event 表加保留期清理（可挂在 group-guard 现有的 `ctx.setInterval` 体系）。理由：当前实现随数据量线性劣化，且在每消息热路径上。

5. **AI 审核响应 zod 校验**（高，安全）：`report-service.ts:283-307` 用 zod 严格校验 `{severity, summary}`，未知 severity 一律按失败处理（走已有的 `aiStatus: 'failed'` 分支），不要默认 `'none'` 静默放行。仓库已有 zod 惯例（governance-actions.ts:38-53）。

6. **补 `admission-admin-commands.ts` 测试**（高）：参照 `admission-console-api.test.ts` 的形态补齐 6 个命令的测试，特别是 skip/regenerate 的取消协调与去重回滚分支（admission-admin-commands.ts:124-165 的 try/catch/forget 逻辑）。

7. **SSE 心跳与失活检测**（中）：`platform/index.ts:330-361` 加 inactivity timeout（配合后端定期发 `:ping` 注释行），超时主动 abort 触发现有重连。理由：半开连接是该架构唯一的静默失效模式。

8. **统一 console 范围解析**（中）：让 `chat-delivery.ts` 复用 `console-guild-scope.ts` 的 `resolveConsoleGuildScope`（两者 deps 形状已一致），删除 80-122 行的拷贝；同时把 authority=4 常量收敛到 shared 一处。

9. **fail-closed 范围选项**（中，安全）：`console-guild-scope.ts:44-46`"无角色 = 全局"改为显式配置（例如仅 `authority >= 5` 或白名单账号默认全局），其余无角色账号拒绝访问。

10. **新生转发逐项容错**（中）：`member-guard.ts:410-416` 改为 per-item try/catch + 记录失败原因（可写 moderation event），避免单个坏项阻塞队列；同时考虑给后端回执失败状态。

11. **删除 duck-typing 防御分支**（中）：§4.1 列出的 4 处强转检查改为让测试 fake 实现完整接口（或抽出窄接口类型由 store 实现），恢复编译期约束；顺带消除"方法缺失默认放行"的隐藏语义。

12. **后端补批量校验端点**（中，需 server/ 配合）：`POST /api/v1/bot/qq-users/verification:batch`，身份页一次请求拿全量，删掉 `identity-profile-lookup.ts` 的并发池与 LRU（该文件 161 行可缩到 ~30 行）。

13. **小项清理**（低）：`json.store.ts:150-152` 的 timer flush 包 try/catch；`data.service.ts:204` 时区 hack 换用 `formatShanghaiTimestamp`；`chat-image-fetch.ts` 注册表加 TTL/LRU；`moderation-core/store.ts:332` `listOpenReports` 补 status 过滤或更名；group-guard `index.ts:162-173` 合并两个 `ctx.on('ready')`；`cache.service.ts` 三处空 catch 至少补 debug 日志。
