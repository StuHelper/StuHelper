---
type: internal
audience: maintainers
status: current
authoritative-source: docs/reviews/2026-06-11-koishi-webui-admin-review/ + 当前仓库状态
last-verified: 2026-06-11
---

# 2026-06-11 Koishi 插件 / WebUI / Admin 全面优化执行计划

事实输入：`docs/reviews/2026-06-11-koishi-webui-admin-review/`（四份深度审查报告）。本计划是头脑风暴的收敛结果，按依赖关系和价值密度排成 4 个 wave。项目方针：active development、无兼容包袱、no compat shims、重写优于补丁。

## 总体诊断

| 面 | 健康度 | 一句话 |
|---|---|---|
| koishi 服务端插件 | 中 | 架构分层正确、测试基建好，但拖着 ~14.5k 行死代码，热路径每消息 5+ 次 DB 读，多处 fail-open 安全弱点 |
| koishi WebUI | 中 | 新一代视图（models+primitives+composable）质量高，但 4 个 2600-3200 行巨型组件、13 份加载样板、keep-alive 数据陈旧、Chat 双实例 bug |
| stuhelper admin | 中上 | 架构层（OIDC/OpenAPI 类型/capability 权限）优秀，视图层 22 份 CRUD 样板复制、错误双重弹出、users 域 i18n 失守 |
| 跨面对接 | 分化 | admin↔server 类型全生成最健康；koishi↔server 592 行手写类型已漂移 4 处；SSE 协议双侧手写无契约测试 |

## Wave 1 — 正确性、安全与系统性去重（并行 3 路）

### Lane K：koishi 服务端（顺序两步）

**K1 安全与正确性修复**
- F005 批量禁言 `split(/\s+/, 2)` 截断 bug（stuhelper-admin/commands.ts:320）
- F006 events.ts 异步事件处理器补 try/catch + 日志
- F047 warningThresholdExpression 保存时校验
- F019/F053/F054 命令策略 fail-closed 统一（含私聊/无 guildId 路径收紧）
- AI 审核响应 zod 严格校验，畸形响应按 failed 处理（report-service.ts:283-307）
- F055 QQ 绑定码 bot 侧限速 + 尝试上限
- F046 incrementWarning 原子化
- F048 的可落地半段：存量非法关键词规则逐条容错跳过 + 告警，不再让整群消息处理抛错
- F051 regenerate/释放路径 unmute 失败容忍对齐 skip
- F052 新生转发逐项容错 + 防部分成功重复转发
- 小项：cache.service 空 catch 补日志、json.store timer flush 包错、data.service 时区 hack 换 formatShanghaiTimestamp、chat-image-fetch 注册表 TTL/上限、F120 提醒投递失败日志

**K2 解耦与性能**
- §3.2 admission 展示层 10 组重复函数收编为共享模块（console api 与 admin commands 双降 500 行内）
- §3.3 console 范围解析统一（chat-delivery 复用 resolveConsoleGuildScope）+ authority=4 常量收口 + F050 fail-closed 范围
- settings store 全家族加读缓存 + save 失效（消除每消息 DB 读，F049）
- moderation-core 查询下推（listRecentMessages/listKeywordRules/listRecentEvents）+ message_ledger/event 保留期修剪任务（F004）
- SSE 读循环 inactivity timeout（platform/index.ts:330-361）
- §4.1 duck-typing 防御分支删除，测试 fake 补全接口

### Lane A：stuhelper admin（顺序两步）

**A1 系统性视图层重构**
- 错误提示单层化：shared-result.ts 移除 ElMessage，展示职责归视图层
- 新建 `useAdminListPage`/`useAdminAction` composables 并迁移全部 ~22 个列表视图（竞态守卫/loading/error/分页/行级 action 锁/URL query 同步一体化）
- AdminContentLayout 补 description prop（修静默丢文案 bug）
- 死代码删除：content/logs/index.vue、adapter/vxe-table.ts、未用 API 导出（F103/F106）
- clipboard 工具收口（apps + admission-sessions 两份拷贝）
- 错误码常量收口到 @stuhelper/shared error-codes（F027）
- 分页布局统一（带 sizes 的共享常量）

**A2 页面级修复**
- F028 teachers departmentID 空字符串绕过校验
- F107 敏感词 trim + category 默认
- F025 schoolId NaN 静默丢弃 → 显式校验
- F104 OIDC redirect 绝对 URL 防护
- F105 路由守卫日志 + 引导失败兜底
- F108 oxlint 非空断言修复
- F031 新生审核通过加确认 + 天数 0 显式语义
- F029/F030 admission-policy 批量创建失败清单反馈 + 默认值不静默覆盖后端
- F026 step-up 跳转异常回归 ApiCallResult 契约
- users 域 i18n 收口（admission-policy/freshman-verification/admission-sessions/member-blacklist 全部迁入 admin.json 双语）

### Lane S：Go server P0/P1（一步）

- F023 typed-nil verifier panic
- F001+F007 新生凭证过期投影回写 + join decision 过期条件（P0 数据正确性，同根因一起修，带回归测试）
- F020 跨校签发校验
- F060 join decision 检查 member blacklist

## Wave 2 — 契约统一与 WebUI 架构

- koishi 类型生成化：openapi-typescript 从 openapi.bundled.yaml 生成，592 行手写类型替换为别名，加 check:api-drift（跨面报告统一方案 #1）
- SSE 事件协议契约测试双侧锁定（统一方案 #3）
- F072 security 声明契约测试（统一方案 #2）
- WebUI `useAsyncResource`（13 份样板 + keep-alive onActivated 重校验）
- WebUI `useViewNavigationSync`（5 份拷贝）+ navigation provide/inject（13 个可选 prop 删除）
- WebUI 轻量 `page/overview` pulse 端点（30s 全量 dashboard 轮询 → 5 个计数）
- WebUI Chat 单例 store（修双实例双监听 bug）+ Subscription 稳定 id
- WebUI 错误横幅 CSS/确认对话框统一
- 删除 core/modules 死代码 ~14.5k 行（先迁出 log-redaction/CommandLogRecord）
- admission-admin-commands 测试补齐

## Wave 3 — 巨型组件拆分与深度性能

- WebUI：SettingsView(3182)/RolesView(3070)/ConfigView(2686)/ChatView(2607) 四件套拆分；settings-schema 单源化（54 键三处重复）；RolesView 草稿模式 + IntersectionObserver；大列表虚拟化/分页
- **WebUI 协同重构（从 Wave 2 并入，详见 Wave 2 续记录）**：项 4 useAsyncResource 落地全部 13 视图（composable 已建成待接入）+ keep-alive onActivated 重校验；项 5 navigation provide/inject + useViewNavigationSync；项 7 ChatView 单例 store + Subscription 稳定 id（随 ChatView 拆分）；项 8 错误横幅/确认对话框 CSS 收口为 primitive。**必须同步重写 `component-contract.test.ts` 的家族级断言**（内联竞态守卫 → composable 契约、精确 CSS 类名 → primitive 契约、navigation prop → inject 契约），否则契约测试会与新实现冲突。
- Admin：open-platform/apps/index.vue(1025) 拆分
- server 配合：qq-users verification 批量端点（消 N+1）；F058 outbox 清理任务；F062 auth_url 明文 token 治理
- 聊天广播缓存复用 CacheService（每消息远程调用消除）

## Wave 4 — 功能完善 brainstorm（按价值排序的新能力）

1. **Admin 通知中心真实化**：现为纯 UI 占位（basic.vue），接 SSE/轮询的审核待办（pending reviews/reports/freshman/scope 审批计数），与 koishi WebUI pulse 同型
2. **黑名单/会话页"执行面归属"提示**：admin 操作后显示 koishi 现场传播状态（outbox 动作状态查询端点已有数据）
3. **Admin 列表导出能力**：shared 层已有 export 端点未被 UI 使用，接到 reviews/operation-logs
4. **content-flags 与 review edit 端点 UI 化**（shared 层已有、前端未用）
5. **WebUI 写后 broadcast 增量更新**：用 chat-message-broadcast 已验证的通道替代写后全量回刷
6. **统一审计视图**：admin operation-logs 与 koishi moderation events 同屏（跨面只读聚合）
7. **F121 AI apiKey 加密存储**；F122 绑定日志 PII 脱敏
8. **F045 SAST 覆盖 bots/koishi**
9. **F038/F032 seed/migrate-verify 破坏性操作防护**
10. **WebUI 键盘可访问性补课**（Logs 行点击、Chat 右键菜单、EntityOverlay li→button）+ 320/768/1440 视觉回归

## 验收原则

- 每个 lane 完成后必须通过该 workspace 全量验证：koishi `yarn build && yarn test:unit && yarn test:startup`；admin `pnpm test && pnpm check:type && pnpm lint`；server `go build ./... && go test ./...`
- 涉及 OpenAPI 的改动先改 `server/api/openapi.yaml` 再生成类型
- 行为契约（测试断言的文案/选择器/数据钩子）改动必须同步更新测试并在提交信息中声明
- 不引入兼容 shim；所有调用点直接改

## Wave 1 完成记录（2026-06-13）

全部 lane 完成并通过全量验证（koishi build + 607 单测 + startup；admin 174 测试 + typecheck + lint 0 错；server go build + 全量 go test）。

补充实现说明：

- **F049 settings 读缓存**：`packages/shared/src/settings-cache.ts` 的 `SettingsReadCache` 按 (database, 表名) 在模块级共享——同一张表被插件运行时实例与 WebUI console API 实例各自构造的 store 消费，按实例缓存会让 WebUI 保存对热路径不可见（真实 bug，已修）。cordis 服务代理每次访问返回新 Proxy，身份锚点用 `database.tables`。测试里写 settings 必须走 store（裸 DB 写不会被缓存观察到），message-guard/commands 测试的 helper 已改为 store 保存。
- **F004**：listRecentMessages/listRecentEvents/listKeywordRules 下推 sort/limit/$in；新增 `pruneExpired` + group-guard 配置 `retention`（消息 14 天/事件 30 天默认）每日修剪。
- **SSE inactivity timeout**：`readWithInactivityTimeout` 60s（服务端 15s keepalive ×4）。
- **§3.3 chat-delivery**：复用 `resolveConsoleGuildScope`/`hasConsoleGuildAccess`，删除 ~70 行重复范围展开。
- **F030**：unverifiedJoinRejectReason 留空不再注入前端默认值（placeholder 提示，后端落默认）。
- **F029**：批量创建逐群容错，失败群号回填输入框，重试只补失败项。
- **users 域 i18n**：admission-policy/freshman-verification/admission-sessions/member-blacklist 全量迁入 zh-CN/en-US admin.json；`admissionReissueCommand` 是发给 bot 的命令字符串，保持中文不走 i18n；挂载类测试统一 `vi.mock('#/locales')` 返回 key。
- **G6 边界**：open-platform 5 视图列表状态层已迁 useAdminList/useAdminLoad；apps/index.vue 的 9 个 action 处理器仍为手写 actionLoading 模式，**留给 Wave 3 的 apps 拆分任务一并转 useAdminAction**（流程含 issuedSecret 捕获与多步 prompt，拆分时重构更合适）。
- 工作区保持未提交（按要求不做 git add/commit）。

## Wave 2 进度（2026-06-13）

- **项 9 死代码删除 ✅**：净删 15,405 行 / 115 文件。迁出活件至 core/data/（log-redaction.ts、command-log-records.ts 含 CommandLogRecord 类型）与 core/settings/（report-prompts.ts——review 清单遗漏的活依赖，utils/execute-command.ts 同遗漏已修）；整删 core/modules/、runtime/、register-runtime-modules、runtime-contract.test；连带清理 service 模块注册表、warm-cache 死链（initModules 无调用方，预热从未执行）、moduleStates/systemStatus 全链（server page API + stats/modules 端点 + WebUI Dashboard 系统状态区块 + client api/types）；log-module-lookup 化简为唯一 data.commandLogs 来源；startup-smoke 的"旧模块已停用"断言改为 WebSocket API 注册标记。验证：build + 452/452 + startup 全绿。
- **项 10 admission-admin-commands 测试 ✅**：新增 14 个测试（fake ctx 捕获 action 直调），覆盖 6 命令 + 去重窗口/失败 forget 重试（resend/regenerate/skip）、skip 的 cancelSubject→cancel→runExclusive→clearSubjectCancellation 协调顺序与失败清理、409 已取消会话复用、unmute 失败容忍、权限 fail-closed、WebUI 停用开关。466/466 全绿。
- **项 1 类型生成化 — 阶段 1 ✅**：openapi-typescript@7.13 + `yarn api:generate`（→ packages/shared/src/types/api.gen.ts，12,157 行 209 schema）+ `yarn check:api-drift`。
- **项 1 类型生成化 — 阶段 2 ✅**：types/index.ts 与 types/member-blacklist.ts 的 API 契约段全部替换为 `components['schemas']`/`operations` 别名（同名直映；bot 端请求映 Bot 前缀 schema；FreshmanForwardItem、MemberBlacklistListResult 从 paths 内联响应提取；9 个枚举 union 改为生成类型派生；re-export components/operations/paths）。Stuhelper*Config 等 koishi 内部类型保持手写。漂移核实：生产代码全为读取方，必填化/收窄兼容，4 插件 `tsc --noEmit` 干净（注意 yakumo build 对插件只跑 esbuild 不查类型，需手动 tsc 验证）；测试 fixture 修 6 处 userID 漂移（number→string、删 null）。验证：build + 466/466 + startup + drift 全绿。
- 其余 Wave 2 项（SSE 契约测试、F072、WebUI useAsyncResource/导航/pulse/Chat 单例/横幅统一）未开始。

### Wave 2 续（2026-06-14）

- **项 2 SSE 事件协议契约测试 ✅**：新建共享 fixture `server/api/contracts/admission-action-stream.json`（开流注释 + action/keepalive/error 字节级帧 + actionPayload 全字段样例）。server `handler_bot_queries.go` 发射点收口为协议常量 + 三个窄函数（writeAdmissionActionEvent/Keepalive/StreamError）；`stream_contract_test.go` 锁字段双向集合关系（DisallowUnknownFields ⊆ + 字节比对 ⊇）、拒绝未知事件名、校验 actionPayload 字段全在 openapi schema。koishi `action-stream-contract.test.ts` 回放同一 fixture 字节，断言 action 分发 / keepalive+error+注释忽略 / 关流转重连错误。补 openapi stream 描述的 `event:error` 说明并 redocly 回归 bundle + api:generate + drift。验证：koishi build+468/468+startup；server build + admission 全测试。
- **项 3 F072 security 契约测试 ✅**：先修 spec 实际漂移——admission.yaml 5 操作 security 与代码对齐（mobile-camera-handoffs ×3 token 凭证补 `security:[]`；school-sso login/callback 删误标 `security:[]` 回归会话鉴权）。新建 `admission/security_contract_test.go` 行为式三分类：sentinel authMW（abort 799）探会话鉴权挂载、非 nil fakeVerifier 使 bot 路由无凭证→401、静默 CustomRecovery 把公开路由 nil-service panic 转 500；按 `apigen.GetSwagger()` effective security 分类每个 admission/bot 操作并断言匿名状态码，三类计数 Positive 守卫防退化。漂移双向可捕获。redocly bundle + go generate 再生 server.gen.go（仅 gzip 嵌入 blob churn）。验证：redocly lint + admission 全包 + vet + build + app 路由契约 + koishi drift 全绿。

- **项 6 page/overview pulse 端点 ✅**：use-pulse 原每 30s 轮询全量 dashboard（加载并序列化 7 集合明细）但只用 5 计数。dashboard-page.service 抽共享 `buildDashboardOverview`（dashboard 与 pulse 唯一计数来源，保证不漂移；highRiskEvents 是最近窗口计数非全表，故不可改全表 count）+ `getOverviewData()` 只返回 `{generatedAt, overview}`；page-scope 抽 `applyScopeToDashboardInput` 供 scoped dashboard/overview 共用；注册 `page/overview` 监听 + scope 感知 handler；augmentations + client 类型/api 补 OverviewPageData。消除每 30s 的序列化与负载体积（DB 读因 parity 保留）。新增 getOverviewData↔getPageData.overview 计数 deepEqual parity 测试。验证：build + 469/469 + startup。

### Wave 2 WebUI 项（4/5/7/8）重新归入 Wave 3（2026-06-14 决定）

**事实依据**：`plugins/stuhelper-core/client/component-contract.test.ts`（1194 行）把 13 个视图的内联竞态守卫样板、错误可见性、不隐藏陈旧数据、精确 CSS 类名、navigation prop 形态作为**家族级行为契约逐字锁定**；其中 4 个（SettingsView/RolesView/ConfigView/ChatView）是 Wave 3 拆分目标且含多 loader 复杂守卫。client 无 vue-tsc、无 per-view 测试。

- **项 4 useAsyncResource**：composable + 6 单测已建成并验证；但正确落地需一次性迁移全部 13 视图（含 4 巨型件）+ 协同重写约 15 个契约测试块。部分迁移会让契约测试自相矛盾（已实测打破 4 项 → 已回滚到一致状态）。
- **项 7 Chat 单例 store**：聊天状态全部内联在 ChatView.vue（2607 行）内，无独立 store；单例化即 Wave 3 ChatView 拆分的一部分。
- **项 5 navigation provide/inject、项 8 错误横幅/确认对话框统一**：同样横跨契约锁定的视图家族（含巨型件）。

**结论**：项 4/5/7/8 与 Wave 3「巨型组件拆分」强耦合，应作为一次协同的 WebUI 重构执行（巨型件重建 + 契约测试连同重写 + composable 落地 + keep-alive 重校验 + 单例 store + CSS primitive 收口）。拆散为 Wave 2 piecemeal 会退化契约一致性。**Wave 2 的可独立、可强验证项（项 1/2/3/6/9/10）已全部完成。**

## Wave 3 进度（server 侧独立项，2026-06-14）

巨型 Vue 拆分（含并入的 Wave 2 项 4/5/7/8）受 vue-tsc 缺失 + 家族级契约测试约束，留待专注会话。先完成 server 侧可独立强验证项：

- **F058 admission outbox 清理 ✅**：admission_bot_action_outbox 终态行（succeeded/dead_letter/stale）此前无限膨胀。迁移 000016（终态分区索引）+ repository.PruneTerminalBotActions + service 保留期 7 天/每小时 cleanup worker + StartBackgroundJobs 注册。非终态一律不删。2 个真实 PG 测试。验证：build + vet + admission 全包 + 迁移契约。
- **聊天广播复用 CacheService ✅**：chat-message-broadcast 的群信息/at 名片富集改走 deps.api.service.cache.getGuildInfo/getMemberInfo（TTL 缓存），消除每消息远程 N+1（外加 QQ 头像回退）。验证：build + 469/469 + startup。
- **F062 auth_url 明文 token 加密存储 ✅**：迁移 000017（auth_url text→bytea，active-dev drop+add）；Repository 注入既有 pii.EncryptDecryptor，加解密收口在唯一 DB 边界（encryptAuthURL 空→NULL / decryptAuthURL NULL→""；2 个 INSERT 加密写入；3 个 scan 自由函数转 Repository 方法解密 + outbox scan，~21 调用点加 r. 前缀）；app/modules.go 注入 piiCipher；测试 newTestAuthURLCipher 助手 + 4 调用点。安全回归测试：裸列字节 ≠ 明文、不含 token / 不含 /verify/、读回解密还原。API 契约不变（bot 仍收明文 authURL）。验证：go build + go vet ./...（含全测试编译）+ admission 全包真实 PG 38.5s + app 路由契约。
- 其余 server 配合项（qq-users verification 批量端点消 N+1）待续。
