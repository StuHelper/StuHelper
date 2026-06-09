# StuHelper Koishi Workspace

该目录是 StuHelper 的 QQ 机器人工作区，基于 Koishi 官方 boilerplate 初始化，并已接入 StuHelper 平台后端的 QQ 绑定、学生入群认证与群管联动能力。

## 边界

- `server/` 仍然是 StuHelper 的业务权威系统，负责 QQ 绑定关系、绑定码、学生认证状态、新生材料审核和 admission policy。
- `bots/koishi/` 负责 QQ 机器人运行时、群管执行逻辑与管理员命令；入群认证状态和动作以服务端下发为准。
- NapCat 作为外部部署的 OneBot 适配层，不在本目录内实现。

## 当前包与入口

`koishi.yml` 当前显式加载 `stuhelper-core`、`stuhelper-binding`、`stuhelper-group-guard` 与 `stuhelper-admin`，保留完整群管中心页面与功能入口。

- `packages/shared`：共享配置、日志、平台客户端与基础类型。
- `packages/moderation-core`：群管领域模型、SQLite 表、规则引擎与动作服务。
- `plugins/stuhelper-core`：当前入口插件，承载完整群管中心页面、控制台 API 与 WebSocket 交互；WebUI 包含“入群认证”页面，用于查看 group-guard 实际运行态、目标群策略、受限成员队列和学生认证联动状态。
- `plugins/stuhelper-binding`：处理私聊绑定命令，消费平台绑定码并建立 QQ 绑定；命令字和绑定流程提示从 WebUI runtime settings 读取。
- `plugins/stuhelper-group-guard`：处理入群 admission session 创建、禁言、认证链接提醒、后端 pending action 执行、材料转发、关键词命中、撤回留痕、举报流和娱乐命令。
- `plugins/stuhelper-admin`：提供文本管理员命令，用于查看待认证成员、查询警告、查看复核队列、批量禁言、提交踢人/拉黑复核申请，以及 QQ 管理群新生材料审核。

Koishi 群管中心 WebUI 只由 `koishi-plugin-stuhelper-core` 注册到 Koishi Console；`stuhelper-group-guard` 不提供单独前端入口，但会注册 `stuhelperGroupGuard/page/admission-runtime` Console API 供 core WebUI 消费。历史上讨论过的 `stuhelper-console` / `stuhelper-platform` 已按 ADR-0006 从运行路径移除，所以“注册 WebUI”对应的是 `stuhelper-core` 的 Console 入口，而不是 admission 插件本身。

`stuhelper-core` 现在只提供 WebUI、Console API、后台恢复任务和共享服务，不再暴露旧群管运行时模块开关；旧 `report`、`sub`、`config`、`ai` 等命令面由拆分后的插件和 WebUI runtime settings 接管。其原生插件配置只保留 `platform.baseUrl` 和 `platform.serviceToken`，用于访问 StuHelper 后端。

## 本地命令

```bash
cd bots/koishi
corepack yarn install
corepack yarn dev      # 源码态调试，适合本地开发
corepack yarn start    # 产物态启动，先构建再启动 Koishi
corepack yarn build    # 生成各本地包的 lib/index.js 与类型声明
corepack yarn test     # packages/plugins 单元测试 + 真实启动烟雾验证
corepack yarn workspaces list
```

启动前要求：

- `STUHELPER_CONSOLE_ADMIN_PASSWORD` 必须为非空值；`koishi.yml` 会把它作为 Koishi Console 管理员密码。
- `STUHELPER_PLATFORM_BASE_URL` 指向 StuHelper 后端地址；`STUHELPER_PLATFORM_SERVICE_TOKEN` 是 Koishi 调用后端机器人接口时发送的 Bearer token，应与后端 `BOT_SERVICE_TOKEN` 保持一致。
- `STUHELPER_PLATFORM_SERVICE_TOKEN` 对应的 Koishi runtime service account 必须至少具备 `bot.qq_binding.consume`、`bot.qq_verification.read`、`bot.admission.session`、`bot.admission.event`、`bot.admission.review`、`bot.admission.forward` scopes。
- `STUHELPER_GROUP_CENTER_DATA_DIR` 可选覆盖群管中心 JSON 数据目录；留空时使用 Koishi baseDir 下的 `data/stuhelperGroupCenter`。UI smoke 会自动指向临时目录，避免污染本地开发数据。
- `STUHELPER_FRESHMAN_MATERIAL_HOSTS` 是新生材料图片 URL 允许转发的 HTTPS host 白名单；当前 MVP 生产默认不启用材料原图转发扫描。
- 本地可直接 `export STUHELPER_CONSOLE_ADMIN_PASSWORD=dev-console-admin-password`，或把同名变量写入仓库根目录 `.env` / 生产环境变量文件。

## Admission 策略边界

新生入群认证目标群、准入与会话策略由后端 admission policy 决定，并由 `stuhelper-group-guard` 同步为 Koishi 本地 guard policy 执行态缓存。Koishi WebUI 的“同步绑定”只读展示该缓存；目标认证群的增删、启停和入群处理策略请在 StuHelper Admin 的入群认证策略页面修改。`koishi.yml` 不再保留本地 `guard` 业务字段，也不再提供静态目标群兜底。后端负责 `auto_approve_verified_join`、`auto_approve_unverified_join`、初始禁言时长、link/submission/manual-review 等待时间、提醒间隔、失败次数拉黑、黑名单期限、新生通道关闭时间、原始材料转发开关和 `management_guild_ids`。

Koishi 在 admission 流程中只做执行器：入群后创建后端 session，发送后端返回的 `join.stuhelper.com/verify/<code>` 认证链接，通过后端 admission action SSE 下行流接收提醒、解禁、踢出和拉黑动作，执行后按 action ID 回写 ACK。`/sessions/pending` 拉取保留为低频 fallback，不再作为生产主路径。`koishi.yml` 的插件加载保持不变，不新增短链域名配置。

生产 NapCat 的 Koishi runtime platform 是 `onebot`，后端 admission 表中的 `platform` 是被验证账号的 subject platform。当前 admission MVP 验证的是 QQ 号，因此 `onebot` runtime 会显式映射为后端 `platform=qq`；未来若接入官方 QQ 机器人适配器，需要重新确认事件能力和 ID 语义，不能把 Koishi 适配器名直接写入 admission 业务记录。

生产 admission MVP 建议在 `stuhelper-group-guard` 下显式配置：

```yaml
scheduler:
  scanIntervalSeconds: 300
actionStream:
  reconnectDelaySeconds: 5
```

入群认证运行开关由 Koishi Console 的 StuHelper 群管中心“入群认证”页面保存到 `stuhelper_admission_runtime_settings`，并在运行时生效。默认值在 `@stuhelper/koishi-shared` 的 `DEFAULT_ADMISSION_RUNTIME_SETTINGS` 中维护：Action Stream、准入管理员命令、兜底扫描和群内认证提醒默认开启；公开命令、消息风控、新生材料转发和私聊/临时会话认证提醒默认关闭。公开命令和 admission 管理命令启动时始终注册，实际是否执行由 WebUI runtime setting 控制。入群认证管理员命令的执行权限由 Koishi Console“配置治理 / 命令策略”里的 CommandPolicy `admission-admin` 控制；如果还没有保存该策略，运行时会按默认 authority 4 兜底，避免误放开给普通成员。认证提醒的“群内提醒”和“私聊/临时会话提醒”是独立 runtime 开关，但 store 会拒绝两个渠道同时关闭。

QQ 绑定命令字和绑定流程提示由 Koishi Console 的 StuHelper 群管中心“全局设置 / QQ 绑定”页面保存到 `stuhelper_binding_runtime_settings`，默认值在 `@stuhelper/koishi-shared` 的 `DEFAULT_BINDING_RUNTIME_SETTINGS` 中维护。`stuhelper-binding` 原生插件配置只保留 `platform.baseUrl` 和 `platform.serviceToken`，不再包含 `binding.command`、绑定码 TTL 或提示文案。

管理员文本命令和新生审核命令的用户可见提示文案由 Koishi Console 的 StuHelper 群管中心“全局设置 / 管理员命令”页面保存到 `stuhelper_admin_runtime_settings`，默认值在 `@stuhelper/koishi-shared` 的 `DEFAULT_ADMIN_RUNTIME_SETTINGS` 中维护。`stuhelper-admin` 原生插件配置只保留 `platform.baseUrl` 和 `platform.serviceToken`，不再包含命令提示文案；命令执行开关仍由 `stuhelper_admission_runtime_settings.adminCommandsEnabled` 控制，命令权限由 CommandPolicy 控制。

骰子默认面数和抽禁言基础秒数、上限、保底阈值、保底秒数由 Koishi Console 的 StuHelper 群管中心“全局设置”页面保存到 `stuhelper_group_guard_behavior_settings`，默认值在 `@stuhelper/koishi-shared` 的 `DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS` 中维护。`stuhelper-group-guard` 原生插件配置不再包含 `fun` 字段，公开命令每次执行时读取该 runtime settings，因此 WebUI 修改后不需要重启 Koishi。

消息风控关键词规则由 Koishi Console 的 StuHelper 群管中心“全局设置 / 禁言关键词 / 群管关键词规则”页面保存到 `stuhelper_moderation_keyword_rule`，运行时只读取这张表。`stuhelper-group-guard` 原生插件配置不再包含 `moderation.keywordRules`，不要在 `koishi.yml` 中维护第二份关键词规则。

入群认证、公开命令、消息风控、举报和控制台操作的用户可见提示文案由 Koishi Console 的 StuHelper 群管中心“全局设置 / 群管提示”页面保存到 `stuhelper_group_guard_message_settings`，默认值在 `@stuhelper/koishi-shared` 的 `DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS` 中维护。`stuhelper-group-guard` 原生插件配置不再包含 `messages` 字段；运行时发送提醒、执行命令和记录群管事件时通过轻量 message provider 读取同一份 runtime settings，避免 WebUI 与插件配置重复维护文案。

举报 AI 审核的启用状态、HTTP 接口、模型和 API Key 由 Koishi Console 的 StuHelper 群管中心“全局设置 / AI 功能 / 举报 AI 审核”页面保存到 `stuhelper_group_guard_ai_settings`，默认值在 `@stuhelper/koishi-shared` 的 `DEFAULT_GROUP_GUARD_AI_SETTINGS` 中维护。`stuhelper-group-guard` 原生插件配置不再包含 `ai` 字段；运行时每次处理举报时读取同一份 runtime settings。WebUI 只显示 `apiKeyConfigured` 和脱敏后的 `apiKeyMasked`，保存时只发送 `newApiKey` 或 `clearApiKey`，不会把 API Key 明文回传到浏览器状态快照。

`platform.baseUrl`、`platform.serviceToken`、`scheduler.scanIntervalSeconds` 和 `actionStream.reconnectDelaySeconds` 仍是启动配置，只在 WebUI 脱敏或汇总展示，不做浏览器侧热改。`scheduler.fallbackScanEnabled`、`actionStream.enabled`、`moderation.enabled`、准入管理员命令开关和准入管理员命令权限不再是原生插件配置，分别由 WebUI runtime settings 与 CommandPolicy 控制。不要因此卸载或关闭旧 `student-query` 插件本身。若旧插件也在同一批目标群处理同一阶段入群验证，应在旧插件自身的目标群或功能开关中排除 admission 群，避免两个监听器双处理。

## 自动化验证

- 单元测试基于 Koishi 官方 `@koishijs/plugin-mock`，不需要连接真实 OneBot/NapCat。
- `test:unit` 会覆盖 `packages/` 与 `plugins/` 下的 Koishi 测试文件。
- 绑定插件测试会验证私聊绑定命令和群聊误用提示。
- 群管插件测试会验证 admission session 创建、入群禁言、认证链接、后端提醒/解禁/踢出/拉黑 action、材料转发、关键词处理、模板与同步绑定策略解析以及撤回留痕。
- 管理员命令测试会验证 QQ 管理群新生审核命令、操作者 QQ 上报、后端 capability 错误映射和黑名单解除。
- 控制台测试会验证高风险批量操作改走人工复核、复核执行、举报报表聚合，以及模板保存、同步绑定只读展示和命令策略写入 SQLite。
- 启动烟雾验证会真实拉起一次 Koishi，确认四个 StuHelper 插件、群管中心、Console API 与群守卫能力可启动，并固定监听 `5140`。
