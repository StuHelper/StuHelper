# StuHelper Koishi WebUI（stuhelper-core/client）+ Console 数据通道 + E2E 覆盖分析报告

代码规模：client 共 92 个文件、约 27,849 行（含测试与 CSS）。入口为 `client/index.ts:48-57`（`ctx.page()` 注册 `/stuhelper`，authority 4），页面壳为 `client/pages/index.vue` → `components/shell/AppShell.vue`。

---

## 1. 功能与视图清单

视图注册表：`client/composables/use-console-pages.ts:18-32`（13 个 view），view id/标题：`client/models/views.ts:1-35`。导航状态（view/workspace/guildId/memberId/itemId/tab/keyword）序列化到 URL hash：`client/models/navigation.ts:7-104`。

| 视图 | 文件（行数） | 功能与交互 | 数据通道（console 事件） |
|---|---|---|---|
| DashboardHub | `components/DashboardHubView.vue`（343） | 指标卡、4 张图表、待办/快捷入口/系统状态/最近事件，点击跳转到目标 view | 并行拉取 `stuhelperGroupCenter/page/dashboard` + `stats/dashboard` + `stats/charts`（`DashboardHubView.vue:256-260`） |
| Admission | `components/AdmissionView.vue`（584） | 运行开关（el-switch 即时保存）、同步目标群（只读）、模板表、受限成员队列行内动作（查询/重发/重建/跳过/清次数/解拉黑，带确认） | 跨插件命名空间：`stuhelperGroupGuard/page/admission-runtime`、`action/admission-member`、`action/save-admission-runtime-settings`（`client/page-api.ts:23-31`），由 `plugins/stuhelper-group-guard/src/admission-console-api.ts:91-99` 提供 |
| ConfigCenter | `components/ConfigCenterView.vue`（392） | 4 个 workspace tab（群配置/模板库/同步绑定/命令策略），脏表单切换确认，逻辑在 `composables/use-config-governance.ts` | `stuhelperGroupCenter/page/config-governance`、`action/save-guard-template`、`action/save-command-policy`（server：`src/core/api/governance-actions.ts:62-69`） |
| Config（legacy，嵌套在 ConfigCenter 的"群配置"tab） | `components/ConfigView.vue`（2686） | 群配置 CRUD、复制群号、重载、删除需输入群号二次确认、多 tab 编辑弹窗 | `stuhelperGroupCenter/config/*`（`client/api.ts:86-94`，server `src/core/api/config-api.ts`） |
| Warns | `components/WarnsView.vue`（608） | 按群聚合左右分栏，行内 el-input-number 改次数、清零确认、添加警告 Drawer | `stuhelperGroupCenter/warns/*`（`client/api.ts:97-105`） |
| Blacklist | `components/BlacklistView.vue`（395） | QueueTable 列表、添加 Drawer（guild/global 范围，global 二次确认）、解除/宽恕双动作 | `stuhelperGroupCenter/blacklist/*`（`client/api.ts:108-114`，server `src/core/api/member-blacklist-console-api.ts`） |
| Identity | `components/IdentityView.vue`（402） | 受限成员过滤 + 详情、最近自动解除、查询错误区；过滤/选中同步 URL | `stuhelperGroupCenter/page/identity`（server scope 过滤：`src/core/api/page-api.ts:54-67`） |
| Review | `components/ReviewView.vue`（466） | 复核/准入/举报统一工作项列表 + 右侧处置面板（备注 + 动作按钮 + 确认） | `stuhelperGroupCenter/page/review`、`action/work-item`、`action/review`（server `src/core/api/review-actions.ts:28-36`） |
| Roles | `components/RolesView.vue`（3070） | 角色 CRUD/克隆/拖拽排序、权限树勾选（分组滚动导航）、成员增删、三源批量导入（角色/authority/群管理员） | `stuhelperGroupCenter/auth/*` 共 12 个事件（`client/api.ts:469-486`，server `src/core/api/auth-api.ts`） |
| Logs | `components/LogsView.vue`（501） | 多条件检索 + 分页 + 行点击 Drawer 详情，deep-link 过滤 | `stuhelperGroupCenter/logs/search`（server 端分页：`src/core/api/logs-api.ts:86-91`） |
| Chat | `components/ChatView.vue`（2607） | 实时会话列表、消息流（文本/图片/引用）、群成员侧栏、@/回复/复制/转发/撤回右键菜单、粘贴图片发送 | 拉：`stuhelperGroupCenter/chat/*`（`client/api.ts:382-397`）；推：`receive('stuhelperGroupCenter/chat/message')`（`ChatView.vue:765`，server 广播 `src/core/api/chat-message-broadcast.ts` → `chat-delivery.ts`）；图片代理 `image/fetch`（`client/api.ts:410-424`） |
| Subscription | `components/SubscriptionView.vue`（547） | 订阅卡片网格、Drawer 编辑 6 类推送事件、按数组下标更新/删除 | `stuhelperGroupCenter/subscriptions/*`（`client/api.ts:117-122`） |
| Settings | `components/SettingsView.vue`（3182） | 15 个 section 的全局设置巨型表单（警告/禁言/AI/举报/绑定文案/管理员文案/群管提示/关键词规则编辑器），JSON 快照 diff 检测脏状态 | 一次并行 7 个 get（`SettingsView.vue:1891-1899`）；保存串行 7+ 个 update（`1942-1952`）：`settings/*`、`admin-settings/*`、`binding-settings/*`、`group-guard-ai-settings/*`、`group-guard-behavior-settings/*`、`group-guard-message-settings/*`、`keyword-rules/*` |
| System | `components/SystemView.vue`（259） | 缓存统计 + 强制刷新/清空（带确认） | `stuhelperGroupCenter/cache/*`（`client/api.ts:456-466`） |

壳层附加表面：
- **EntityOverlay**（`components/shell/EntityOverlay.vue`，616 行）：用户/群实体侧滑画像，事件 `stuhelperGroupCenter/page/entity-profile`，各 fact 区可跳转对应 view。
- **SearchPanel**（494 行）：⌘K 全站搜索（view/命令/实体）。
- **ChatDock**（`components/shell/ChatDock.vue:44-57`）：懒挂载 ChatView 的悬浮 dock。
- **Pulse**（`composables/use-pulse.ts:17,64-67`）：每 30s 轮询 dashboard 页面数据，驱动 CommandBar/NavRail 角标。

---

## 2. 架构

### 与 Koishi console 的集成
- **入口注册**：`src/setup/register-console-entry.ts:10-15` → `ctx.console.addEntry(resolveBrowserEntry())`，dev/prod 双路径在 `src/browser-entry.ts:10-15`。
- **API 注册**：`src/setup/register-console-api.ts:23-38`，在 `ctx.inject(['console','database','stuhelperGroupCenter','auth'])` 内依次注册 WebSocket 域 API（`src/core/api/index.ts` 的 `registerWebSocketAPI`）、页面域 API（`src/core/api/page-api.ts:23-45`）、处置动作 API、治理动作 API；启动时强校验 `STUHELPER_CONSOLE_ADMIN_PASSWORD`（`src/console-auth.ts:6-16`）。
- **事件协议**：全部走 `ctx.console.addListener(event, handler, { authority: 4 })`（`page-api.ts:21,30-44`）；页面域事件每次请求都通过 `resolveRequiredConsoleGuildScope` 做按群 scope 过滤（`page-api.ts:165-167`，`console-guild-scope.ts`）。这是清晰的"page = 只读聚合，action = 写"分层。
- **服务端推送**：仅聊天用 broadcast（`chat-message-broadcast.ts` → 客户端 `receive`，`ChatView.vue:765`）；其余页面无推送，靠手动刷新 + pulse 轮询。
- **类型贯通**：服务端 `src/augmentations.d.ts` 增广 console 事件表；client 的 `page-api.ts` 直接用类型化 `send`，而 `client/api.ts:54-56` 却 `send as ConsoleSend` 抹掉了事件表类型（见第 4 节）。契约由 `src/core/api/console-api.contract.test.ts`、`page-api.contract.test.ts`、client `component-contract.test.ts`（1193 行）兜底。

### 状态管理
- 无 Pinia/全局 store。三层：
  1. **URL 即状态**：`use-console-navigation.ts:31-112`（pushState/replaceState + popstate/hashchange 同步），配合纯函数 `models/navigation.ts`——这是该 client 最好的设计之一。
  2. **壳层 provide/inject**：`use-app-shell.ts:40-148`（rail/chat/entity/search 开闭互斥）。
  3. **视图本地 ref/reactive + models/ 纯函数建模**（`models/dashboard.ts`、`models/review.ts`、`models/identity.ts` 等，均有单测）。

### 组件分层
- `shell/`（AppShell/NavRail/CommandBar/SearchPanel/EntityOverlay/ChatDock）→ `views`（13 个）→ `primitives/`（WorkspaceHead/WorkspaceSection/QueueTable/Drawer/ConfirmDialog/EmptyState/SeverityTag/EntityChip/NoticeStack/ConsolePageSkeleton）→ `dashboard/`、`chat/` 子域组件。
- 视图切换用 `<keep-alive>` 全量缓存（`AppShell.vue:25-27`），navigation 经 props 逐层传入每个 view。

### 样式体系
- Design token：`styles/tokens.css`（`--sh-*` 色彩/间距/字号/动效）；全局原语类：`styles/primitives.css`（1174 行，`.sh-stat/.sh-lane/.sh-field/.sh-load-error` 等）；壳层：`styles/shell.css`（954 行）。Element Plus 组件 + 自定义 `.sh-*` 皮肤混用。
- 例外：RolesView/SettingsView/ConfigView/ChatView 各自带数百至上千行 scoped CSS，自建 `.save-bar/.modal-dialog/.dialog-card` 等，与 `.sh-*` 体系并存（e2e 中可见两套 dialog selector：`stuhelper-views.spec.ts:1604-1642`）。

---

## 3. 交互逻辑问题

**好的部分先说明**：所有视图都有 skeleton/错误空态/局部刷新错误条三态（如 `DashboardHubView.vue:30-48`）；请求竞态统一用 requestSeq 守卫；危险动作均有 ConfirmDialog；ConfigCenter 有脏表单拦截 + beforeunload（`use-config-governance.ts:112-116`）。缺陷集中在：

1. **keep-alive 导致数据陈旧**：13 个 view 全部 keep-alive（`AppShell.vue:25`），但只有 AppShell 自身用了 `onActivated`（`AppShell.vue:146`）；所有 view 仅在首挂载时 load（如 `AdmissionView.vue:376`、`ReviewView.vue:282`），重访不刷新。处置中心这类队列页面，陈旧数据会直接导致操作失败或误判。
2. **无乐观更新，全量回刷**：每个写操作后 `await loadData()/refresh()` 整页重拉（`AdmissionView.vue:424`、`BlacklistView.vue:332`、`WarnsView.vue:470,500`）；Warns 取消清零确认时甚至专门 `refresh()` 一次只为重置 input（`WarnsView.vue:491-493`）。
3. **Settings 保存非原子**：7 个串行 update + keyword 规则逐条 delete/upsert（`SettingsView.vue:1942-1952,1870-1883`），中途失败留下半保存状态，且错误信息不指明哪一段失败。
4. **RolesView「新建角色」立即落库**：点 + 即以 `Date.now()` 为 id 创建服务端记录（`RolesView.vue:845-855`），用户放弃编辑会留垃圾角色；正确做法是本地草稿、保存时创建。
5. **Subscription 以数组下标为标识**：`subscriptionApi.update(index)/remove(index)`（`client/api.ts:120-121`，`SubscriptionView.vue:361,380`），并发修改（双管理员）会改错/删错条目。
6. **表单校验不一致**：Settings 的 keyword 规则有完整校验（`SettingsView.vue:2077-2111`），而 Blacklist/Warns 的 ID 字段只做非空判断（`WarnsView.vue:381`），无数字格式校验；Chat connect 表单无任何校验（`ChatView.vue:877-879`）。
7. **可访问性**：壳层 aria 较完善（NavRail nav label、SearchPanel role=dialog、CommandBar aria-label），但：Logs 用 el-table `@row-click` 打开详情（`LogsView.vue:118`）无键盘路径；Chat 右键菜单（`ChatView.vue:576-581`）无键盘等价操作；EntityOverlay 的 `.sh-overlay__row` 是可点 `<li>` 而非 button（`EntityOverlay.vue:99-113`）；ChatDock 永远渲染 `role="dialog"` 仅靠 data-open 隐藏（`ChatDock.vue:3-9`），关闭时仍在可达树中（e2e 用 `getByRole('dialog')` count=0 断言侧面规避，`stuhelper-views.spec.ts:274`）。
8. **移动端**：rail scrim/breakpoint 处理完善（`AppShell.vue:65-69,91-98`，e2e `:579-599`），但 Admission 成员表 7 列 + fixed 操作列 260px（`AdmissionView.vue:280`）、Settings 15-section 表单在窄屏只有一个下拉切换（`SettingsView.vue:1339-1343`），小屏可用性差；`isCompact`/`isOverflowMode`（`use-console-navigation.ts:109-110`）定义了却几乎没有 view 消费。
9. **Chat 会话不持久**：sessions 全在组件内存（`ChatView.vue:434`），刷新即丢；且 `chat` 同时是 keep-alive view（`use-console-pages.ts:28`）和 ChatDock 懒挂载实例（`ChatDock.vue:30,46-56`）——深链 `#chat` + 打开 dock 会出现**两个互不同步的聊天状态、两个 `receive` 监听**（`ChatView.vue:764-768` 注册后从不 dispose）。

---

## 4. 耦合与代码质量

### 超大组件（>500 行）
| 文件 | 行数 |
|---|---|
| `components/SettingsView.vue` | 3182 |
| `components/RolesView.vue` | 3070 |
| `components/ConfigView.vue` | 2686 |
| `components/ChatView.vue` | 2607 |
| `components/shell/EntityOverlay.vue` | 616 |
| `components/WarnsView.vue` | 608 |
| `components/AdmissionView.vue` | 584 |
| `components/SubscriptionView.vue` | 547 |
| `components/LogsView.vue` | 501 |

四个巨型组件与其余 9 个"models + primitives + composable"风格的新视图形成明显代际断层（ConfigCenterView 392 行就是新风格样板，`ConfigCenterView.vue:290-339` script 仅 50 行）。

### 重复模式（量化）
1. **加载样板 13 份拷贝**：`requestSeq + loading + error + try/catch/finally` 出现在 `DashboardHubView.vue:250-274`、`AdmissionView.vue:382-400`、`BlacklistView.vue:235-254`、`WarnsView.vue:404-429`、`LogsView.vue:336-361`、`ReviewView.vue:294-313`、`IdentityView.vue:302-320`、`SubscriptionView.vue:293-312`、`SystemView.vue:163-180`、`RolesView.vue:685-710`、`ConfigView.vue:802-823`、`SettingsView.vue:1885-1930`、`use-config-governance.ts:118-135`，每份约 20 行，合计 ~260 行纯样板。
2. **navigation 同步逻辑 5 份拷贝**：`applyNavigationState`/`navigationStateKey`/`syncSelection` 在 `ReviewView.vue:315-351`、`IdentityView.vue:322-368`、`WarnsView.vue:522-530`、`LogsView.vue:316-334`、`ConfigView.vue:982-986` 几乎同构。
3. **错误横幅 CSS 重复**：`primitives.css:22` 已有 `.sh-load-error`，但 Blacklist/Warns/Subscription 又各自复制了 ~37 行硬编码 `rgba(248,81,73,…)`/`#ff8a80` 的同款样式（`BlacklistView.vue:358-394`、`WarnsView.vue:565-601`、`SubscriptionView.vue:392-427`）；该硬编码色值全 client 共出现 20 处（5 个文件）。
4. **header 双轨**：已有 `WorkspaceHead` primitive，但 Dashboard/Admission/Review/Identity/ConfigCenter 仍手写 `sh-workspace-head` 裸标记（`AdmissionView.vue:3-38` vs `BlacklistView.vue:3-21`）。
5. **ConfirmDialog 接线 9+ 份**：每个 view 重复 `useConfirm()` 解构 + 模板 8 行 props 透传（`AdmissionView.vue:303-313` 等）。

### Props drilling / 类型安全 / 硬编码
- `navigation` 以可选 prop 穿透 AppShell→每个 view（`use-console-pages` 不注入），所有 view 充斥 `props.navigation?.` 防御（`AdmissionView.vue:345-347`）；应像 AppShellController 一样 provide/inject。
- `client/api.ts:54-56` `const sendConsole = send as ConsoleSend` 抹掉 Koishi 事件表类型，全部返回类型靠手写 `call<T>` 泛型断言；与 `page-api.ts` 的类型化 `send` 双轨并存，且两者错误协议不同（api.ts 解 `ApiResponse` envelope，page-api 直抛）。
- QueueTable 的 cell 弱类型迫使 WarnsView 写运行时类型守卫并 throw（`WarnsView.vue:353-379`）——primitive 抽象反而制造了不安全。
- `GroupConfig.auto` 是字符串 'true'/'false'（`ConfigView.vue:793-796`）的遗留布尔。
- 巨型硬编码字典三处重复同一键集：api.ts `AdminMessageSettings` 54 键（`client/api.ts:148-202`）、SettingsView 默认值（`SettingsView.vue:1193-1256`）、SettingsView 标签表（`1435-1489`）；`groupGuardMessageLabels` 约 128 条（`1491-1619`）。新增一条文案要改 3 处。
- 脏检测用 `JSON.stringify` 全量比较（`SettingsView.vue:1353-1363`、`RolesView.vue:724-744`），对键序敏感且每次响应式变更都全序列化。
- RolesView 滚动监听经 `setTimeout(100ms)` 注册且**从不移除**（`RolesView.vue:712-721`，全文件无 onBeforeUnmount）。

---

## 5. 性能问题

1. **Pulse 轮询拉全量 dashboard**：`use-pulse.ts:39-55` 每 30s 调 `consolePageApi.dashboard()`，服务端并行加载 8 个数据集并序列化全部 pendingMembers/reviews/policies/templates/bindings（`src/core/api/page-api.ts:118-139`、`dashboard-page.service.ts:36-56`），而客户端只用 5 个 overview 数字。应提供轻量 `page/overview` 事件。
2. **keep-alive 全量缓存 13 个 view**（`AppShell.vue:25-27`）：访问过的巨型组件（Settings 3182 行 + Roles 3070 行 + Chat 2607 行）常驻内存且 watcher 持续活跃；Chat 消息数组无上限（`ChatView.vue:832` 只 push 不裁剪）。
3. **大列表无虚拟化、无分页**：全 client 无任何 virtual list（grep 0 命中）。Warns 一次拉全部记录并在内存分组过滤（`WarnsView.vue:409-414,537-548`）；Roles 权限树全量渲染 + 未节流 scroll handler 每次遍历所有分组 DOM 读 offsetTop（`RolesView.vue:630-649`，强制 layout）；Chat 消息 `v-for`（`ChatView.vue:65`）无窗口化。只有 Logs 做了服务端分页（`logs-api.ts:86-91`）。
4. **`fetchNames=true` 默认开启**（`WarnsView.vue:275`、`SubscriptionView.vue:253`、`ConfigView.vue:749`）：每次刷新都要求服务端逐项解析名称，放大列表接口成本。
5. **写后全量回刷**（见 3.2）叠加上面几条，Warns/Blacklist 每次行级操作的网络与渲染成本都是 O(全表)。
6. **Settings 加载即 7 个并行请求 + 保存 7+ 个串行请求**（`SettingsView.vue:1891-1899,1942-1952`），keyword 规则逐条网络往返（`1875-1882`）。

---

## 6. 改进机会（按收益排序）

1. **抽取 `useAsyncResource` composable**（新文件 `client/composables/use-async-resource.ts`）：封装 requestSeq/loading/error/data/refresh，消除第 4 节 13 份样板；现成参照是 `use-config-governance.ts:118-135` 的形态。可一并内置 `onActivated` 重新校验，顺带解决 keep-alive 陈旧数据问题（`AppShell.vue:25`）。
2. **抽取 `useViewNavigationSync(viewId, mapper)`**：统一 `ReviewView.vue:315-351`、`IdentityView.vue:322-368` 等 5 份 URL↔本地状态同步逻辑；`navigationStateKey` 的手工 join（`ReviewView.vue:323-333`）应改为对 `navigation.state` 的深 watch 或选择器。
3. **新增轻量 pulse 端点**：在 `src/core/api/page-api.ts` 增加 `stuhelperGroupCenter/page/overview` 只返回 5 个计数，`use-pulse.ts:42` 改调它；或服务端在写操作后 broadcast overview，彻底去掉轮询（broadcast 基础设施已在 `chat-message-broadcast.ts` 验证可用）。
4. **拆分四个巨型组件**（项目方针允许重写）：
   - SettingsView → 每个 section 一个子组件 + `useSettingsModel` composable；`AdminMessageSettings` 键集/默认值/标签三处重复（`client/api.ts:148-202`、`SettingsView.vue:1193-1256,1435-1489`）应收敛为单一 `models/settings-schema.ts`（key/label/default 同源），由服务端 shared 包导出更佳。
   - RolesView → RoleList/RoleEditor/PermissionTree/MemberPanel/ImportDialog；顺手修复未移除的 scroll 监听（`RolesView.vue:712-721`，改 IntersectionObserver）。
   - ChatView → SessionList/MessageList/Composer/MemberSidebar + `useChatSessions` **提升为 shell 级单例 store**，解决双实例双 `receive` 问题（`ChatView.vue:764-768` + `use-console-pages.ts:28` + `ChatDock.vue:30`），并给消息数组加上限或虚拟化。
5. **统一数据层为 page-api 风格**：`client/api.ts:54-56` 去掉 `as ConsoleSend` 改用增广后的类型化 `send`（事件表已在 `src/augmentations.d.ts`），错误协议统一（envelope 解包收敛到一处）；Subscription 服务端改为稳定 id 标识（`client/api.ts:120-121`、`src/core/api/subscriptions-api.ts`）。
6. **navigation 改 provide/inject**：在 `AppShell.vue:58` provide，view 内 `useConsoleNavigationContext()`，删除 13 个 `navigation?: ConsoleNavigationController` 可选 prop 与全部 `?.` 防御。
7. **统一确认对话框**：在 AppShell 挂一个全局 ConfirmDialog + `provideConfirm()`，删除 9 份 `ConfirmDialog` 模板接线（`AdmissionView.vue:303-313` 等）。
8. **删除三套重复错误横幅 CSS**：`BlacklistView.vue:358-394`、`WarnsView.vue:565-601`、`SubscriptionView.vue:392-427` 全部改用 `primitives.css:22` 的 `.sh-load-error`，色值换 token（`--sh-danger`）。
9. **RolesView 新建改为本地草稿**（`RolesView.vue:836-868`）：保存时才 `updateRole`，避免服务端垃圾角色。
10. **E2E 补缺**：`e2e/stuhelper-views.spec.ts` 已覆盖 13 view 渲染锚点、历史栈去重、深链 fallback、Identity/Admission/Review/Logs/Config 治理/订阅/黑名单/警告/缓存/设置/角色（含三源导入）真实 console action 路径与 ChatDock 全链路（收发/转发/图片/撤回）——质量很高。缺口：Settings 关键词规则编辑器（`SettingsView.vue:2010-2111`）、Roles 拖拽排序与权限勾选 tab、EntityOverlay 加载失败重试路径（`EntityOverlay.vue:41-45`）、键盘可访问性与多视口（规则要求 320/768/1440）视觉回归。

### 关键文件索引
- 客户端入口/壳：`client/index.ts`、`client/components/shell/AppShell.vue`
- 数据层：`client/api.ts`、`client/page-api.ts`、`client/page-types.ts`
- 服务端通道：`src/setup/register-console-entry.ts`、`src/setup/register-console-api.ts`、`src/core/api/page-api.ts`、`src/core/api/index.ts`、`src/core/api/console-guild-scope.ts`、`src/augmentations.d.ts`
- 跨插件 admission：`plugins/stuhelper-group-guard/src/admission-console-api.ts`
- E2E：`bots/koishi/e2e/stuhelper-views.spec.ts`、`e2e/fixtures/auth.ts`、`e2e/fixtures/diagnostics.ts`
