# StuHelper Admin（clients/admin/apps/web-ele）深度分析报告

---

## 1. 功能清单

### 1.1 路由结构

路由由 `import.meta.glob` 自动合并模块（`src/router/routes/index.ts:7-13`），核心路由（登录、404）不受权限拦截（`src/router/routes/core.ts:25-57`），动态路由按 `meta.authority`（capability 字符串）过滤。

| 路由组 | order | 页面 | 权限 (meta.authority) |
|---|---|---|---|
| Dashboard `/dashboard`（`src/router/routes/modules/dashboard.ts`） | -1 | `/analytics`、`/workspace` | `ADMIN_DASHBOARD_VIEW`（来自 `@stuhelper/shared/constants`，dashboard.ts:10） |
| 内容管理 `/content`（`modules/content.ts`） | 1 | reviews、reports、teachers、sensitive-words、logs | `admin:reviews:manage` 等（content.ts:11-17） |
| 用户系统 `/users`（`modules/user-system.ts`） | 2 | identity-review、student-verification、school-config、admission-sessions、freshman-verification、admission-policy、member-blacklist、system-config | `user:identity:review`、`admission:session:read`、`member_blacklist:read` 等（user-system.ts:11-23） |
| 开放平台 `/open-platform`（`modules/open-platform.ts`） | 3 | apps、audit-events、consents、token-probe-evidence、disclosure-report | 全部 `open_platform:manage`（open-platform.ts:8） |
| `_core` | — | `/auth/login`（OIDC 跳转/forbidden 页）、`/profile`、404/403 fallback | 无 |

### 1.2 页面功能与后端依赖

所有视图都不直接发请求，而是经 `src/api/` 薄封装调用 `@stuhelper/shared/api` 的工厂（`createAdminApi` / `createUserAdminApi` / `createAdmissionApi` / `createAuthApi`）。**全部端点清单**（来自 `clients/shared/src/api/*.ts`）：

**认证（`src/api/core/auth.ts`）**
- `GET /api/v1/auth/login?app=admin`（取 Casdoor 授权 URL，auth.ts:120-139）
- `GET /api/v1/auth/me`（会话探测 `tryGetMe`，auth.ts:145-147）
- `POST /api/v1/auth/refresh`、`POST /api/v1/auth/logout`
- `GET /api/v1/auth/step-up`、MFA 引导用 `/auth/me.accountSettingsUrl`（`src/api/shared-client.ts:122-166`）

**内容管理（`src/api/admin/content.ts`，对应 `/api/v1/course/review/admin/*`）**
- reviews 列表/单条处理/批量处理：`GET|PUT /reviews`、`/reviews/{id}`、`POST /reviews/batch`（content.ts:23-51）
- reports：`GET /reports`、`PUT /reports/{id}`（content.ts:59-74）
- teachers CRUD：`GET|POST /teachers`、`PUT|DELETE /teachers/{id}`（content.ts:83-110）
- sensitive-words CRUD（content.ts:112-150）
- 操作日志 `GET /logs`、统计 `GET /stats`（content.ts:152-161）
- 注：shared 层还有 `content-flags`、`export`、`reviews/{id}/edit` 端点，admin 前端未使用

**用户系统（`src/api/admin/user-system.ts`）**
- `GET /api/v1/admin/identities`、`PUT /identities/{userID}`（实名审核）
- `GET /api/v1/admin/student-verifications`、`PUT /{userID}`（学生认证审核）
- `GET|PUT /api/v1/admin/school-configs[/{schoolID}]`
- `GET|PUT /api/v1/admin/system-configs[/{key}]`

**入群认证/黑名单（`src/api/admin/admission.ts`）**
- 新生审核：`GET /api/v1/admin/freshman-verifications[/{id}]`、`PUT .../{id}`（review）
- 策略：`GET|POST /api/v1/admin/admission/policies`、`PUT .../{id}`
- 会话：`GET /api/v1/admin/admission/sessions`、`POST .../{id}/resend|regenerate|cancel`
- 黑名单：`GET|POST /api/v1/admin/member-blacklist`、`POST .../{id}/release`、`POST .../release-by-subject`

**开放平台（`src/api/admin/open-platform.ts`，对应 `/api/v1/admin/open-platform/*`）**
- apps 列表、approve、import-casdoor、scope approve/reject、redirect-uri-requests approve/reject、secret rotate、suspend/resume/revoke、resource-grants grant/revoke、consents 列表/revoke、audit-events、token-probe-evidence、disclosure-report（open-platform.ts:49-244）

**Dashboard**：analytics 与 workspace 共用 `GET /course/review/admin/stats`（`views/dashboard/analytics/index.vue:12`、`workspace/index.vue:12`），并按 accessCodes 过滤快捷入口（analytics/index.vue:64-70）。

---

## 2. 架构

### 2.1 Vben 集成

- **Bootstrap**（`src/bootstrap.ts:20-77`）：组件适配器 → VbenForm 适配 → i18n → pinia（`initStores`）→ `registerAccessDirective` → router → Motion。
- **Preferences**（`src/preferences.ts`）：`locale: 'zh-CN'`、`loginExpiredMode: 'page'`、`defaultHomePath: '/analytics'`。
- **适配层**：`src/adapter/component/index.ts`（333 行，Element Plus → Vben 通用组件映射）、`adapter/form.ts`、`adapter/vxe-table.ts`。**注意：没有任何视图使用 `useVbenForm`/`useVbenVxeGrid`**（grep 全 views 无结果），所有页面用裸 Element Plus 组件 + 自建 `PersistentAdminTable`。
- **请求层**：`src/api/request.ts` 仅暴露 `@vben/request` 的 `RequestClient`；真正管线在 `src/api/shared-client.ts` —— 用共享包的 `createSessionApiClient` 组装 transport（请求 → 401 时 refresh → 仍失败则 `redirectToOIDCLogin` 强制重新认证，shared-client.ts:247-256）。还有不带刷新的 `sharedBaseApiClient`（shared-client.ts:258-264）用于冷启动探测。

### 2.2 认证 / 会话（Casdoor OIDC）

- Cookie 会话 + CSRF 双提交：`buildSecurityHeaders` 注入 `csrf_token` cookie 值（shared-client.ts:57-65；`src/api/utils/csrf.ts:5`）。
- 冷启动：路由守卫无 token 时调 `authStore.initSession()` → `tryGetMe()`（baseClient，不触发刷新，`src/api/core/auth.ts:145-147`）→ 按 401/403/5xx 分类为 `unauthenticated/forbidden/retryable_error`（auth.ts:40-60）。`me.canAccessAdmin=false` 视为 forbidden（`src/store/auth.ts:164-168`）。
- Vben 兼容 hack：用占位 token `'cookie-session'` 通过 Vben 的"已登录"守卫（auth.ts:170-177，有注释说明）。
- 高级流：403+`A0010204` → 跳 MFA 注册；412/428+`A0010205` → 跳 step-up（shared-client.ts:168-183）。刷新返回 403+CSRF 错误码降级为 401 触发重登录（shared-client.ts:241-244）。
- 登录页（`views/_core/authentication/login.vue:25-29`）仅是 OIDC 跳板 + forbidden 提示页；`switchAccount` 用 `prompt=login&maxAge=0` 强制重认证（FORCE_REAUTH_LOGIN_OPTIONS，core/auth.ts:35-38）。

### 2.3 权限控制

- 三层：路由 `meta.authority` ↔ `accessStore.accessCodes`（即 `me.capabilities`，guard.ts:159-176）；按钮级直接 `accessCodes.includes('member_blacklist:manage')`（member-blacklist/index.vue:38-40、system-config/index.vue:39-40）；schoolID 作用域 grant 解析 `resolveScopedSchoolId`（store/auth.ts:77-101，仅 student-verification/index.vue:56 使用）。
- 守卫细节良好：未知路径直接 404 不强制登录（guard.ts:111-116），已知受保护路径仍走完整流程（guard.ts:49-63）。

---

## 3. 交互逻辑问题

1. **错误提示双重弹出（系统性缺陷）**：API 层 `unwrapData/unwrapListData/unwrapVoid` 失败时已经 `ElMessage.error(message)` 再 throw（`src/api/shared-result.ts:53-54、101-102`）；而每个视图的 `handleActionError` 又再 `ElMessage.error(actionError.value)`（如 open-platform/apps/index.vue:372-375、member-blacklist/index.vue:178-181）。同一次失败用户会看到**两个相同 toast + 一条内联 Alert**。toast 职责应只属于一层。
2. **加载错误同样重复**：fetchData 失败既有 API 层 toast，又设置 `loadError` 渲染 ElAlert（reviews/index.vue:170-181）。
3. **筛选/分页状态不进 URL**：所有列表页的 query 都是组件内 `reactive` 状态（reviews/index.vue:35-44、member-blacklist/index.vue:42-51），刷新/分享/返回都会丢失筛选条件。grep 确认 views 下无任何 `useRoute().query` 同步。
4. **分页布局不一致**：大多数页面 `layout="total, prev, pager, next"` 无 page-size 选择（reviews:305、consents:330 等 13 处），operation-logs 用 `"prev, pager, next, sizes, total"`（operation-logs/index.vue:174-175，唯一有 page-sizes 的），blacklist/admission 表又是 `"total, prev, pager, next, sizes"`（BlacklistTable.vue:159、AdmissionSessionTable.vue:258）。三种排列并存。
5. **空状态缺失**：`PersistentAdminTable` 仅依赖 ElTable 默认"暂无数据"（PersistentAdminTable.vue:140-150），没有区分"无数据"vs"筛选无结果"vs"首次加载"。对比 koishi WebUI 有专门 `EmptyState`/`ConsolePageSkeleton` 组件。仅 dashboard 用了 ElSkeleton（analytics/index.vue:10）。
6. **行级 vs 页级 action 锁不一致**：apps 页一个全局 `actionLoading` 禁掉整页所有按钮（open-platform/apps/index.vue:53、789）；admission-sessions 与 identity-review 则是按 id 的行级锁（admission-sessions/index.vue:27-29、identity-review/index.vue:37-39）。后者体验明显更好，前者一次操作会冻结所有行。
7. **确认流程不一致**：approve 用 `ElPopconfirm`（apps/index.vue:774）、reject/lifecycle 用 `ElMessageBox.prompt`（apps/index.vue:377-414）、blacklist release 用专门 Dialog 组件（ReleaseBlacklistDialog.vue）、review 的 delete/hide 走 ReviewActions 组件。同类破坏性操作的确认范式有 3 种。
8. **admission-policy 批量创建非原子**：`submitCreatePolicies` 在 for 循环里逐个 `await createAdmissionPolicy`（admission-policy/index.vue:214-220），中途失败会留下部分创建成功且无清单反馈。
9. **AdminContentLayout 不支持 `description` prop**（AdminContentLayout.vue:2-5 只声明 `title/total`），但 admission-policy/index.vue:274 传了 `description="控制新生入群验证…"` —— **该文案被静默丢弃，从不渲染**。
10. **consents 页强制条件查询**：appID/userID 均空时直接 warning 并清空表（consents/index.vue:54-62），首屏即一个 warning toast，应改为引导性的内联空态。
11. **通知中心是假的**：basic.vue 的 Notification 面板绑定本地空数组（layouts/basic.vue:25、117-126），无任何后端数据源，纯 UI 占位。

---

## 4. 耦合与代码质量

### 4.1 超大文件（>500 行）

| 文件 | 行数 |
|---|---|
| `views/open-platform/apps/index.vue` | **1025**（10+ 个 action handler + 全部表格模板 + 样式集中一处） |
| `views/users/admission-policy/index.vue` | 546 |
| `views/open-platform/disclosure-report/index.vue` | 536 |
| `views/open-platform/apps/ResourceGrantsDialog.vue` | 519 |

### 4.2 重复 CRUD 样板（最大的质量问题）

- `adminErrorMessage()` 一字不差地复制于 **22 个视图文件**（如 apps/index.vue:489-493、member-blacklist/index.vue:183-187、reviews/index.vue:120-124）。
- `fetchRequestSeq` 竞态守卫 + `loading/loadError` + try/catch/finally 的 fetchData 骨架复制于 **23 个视图**（如 reviews/index.vue:54-71、teachers/index.vue:45-67、consents/index.vue:51 起）。
- `handleActionError`、loadError/actionError 双 ElAlert 模板块（约 20 行）几乎每页复制一份（reviews/index.vue:170-191 = member-blacklist/index.vue:209-230 = apps/index.vue:589-610）。
- `copyToClipboard` + `copyTextWithFallback` 整段复制于 apps/index.vue:170-201 和 admission-sessions/index.vue:107-138（koishi 侧已有 `client/utils/clipboard.ts` 同类实现，三份拷贝）。
- 这是教科书级的 `useAdminList(fetcher)` / `useAdminAction()` composable 抽取场景，能删掉约 1500 行。

### 4.3 类型安全（总体优秀，少量瑕疵）

- 全 src 几乎无 `any`：仅 `adapter/component/index.ts:271、292`（来自 Vben 上游模板）和 `PersistentAdminTableColumn.vue:12`（slot row 类型）。
- 类型源头是 **openapi-typescript 生成的 `clients/shared/src/types/api.gen.ts`**（由 `server/api/openapi.yaml` 生成），admin api 层全部 `components['schemas'][...]` 引用（content.ts:10-15、open-platform.ts:10-47）。**没有手写类型脱节问题** —— 这是该项目做得很对的地方。
- 例外瑕疵：
  - `freshman-verification/index.vue:31-35` 定义 `FreshmanReviewRow` 重复加宽 `failureCount/materialURL/qqID`，并在 :64 `as FreshmanReviewRow[]` 断言 —— 但生成类型 `FreshmanApplication` 已含这三个可选字段（api.gen.ts:4594-4596），整个类型与断言是冗余的。
  - `core/auth.ts:63` 用 `(me as Record<string, unknown>).accountSettingsUrl` 取字段，说明 `UserInfo` schema 缺 `accountSettingsUrl` 声明，应补进 openapi 而非运行时探测。
  - `shared-client.ts:213` `error as TransportError` 断言（可接受，axios 错误形状）。

### 4.4 硬编码 / i18n 漂移

- 项目有完整 zh/en 语言包（`locales/langs/{zh-CN,en-US}/admin.json`）且大多数页面用 `$t`，但 **users/ 域大量回退到硬编码中文**：admission-policy/index.vue 63 行含中文字面量（:51-67 字段标签、:104 默认拒绝理由、:274-498 模板文案）、freshman-verification 23 行（:91、:99、:158、:172-174）、admission-sessions/index.vue（:100、:120、:143-157、:197）、member-blacklist/index.vue（:129、:167、:193）。en-US 用户在这些页面会看到中英混排。
- 其他硬编码：`STORAGE_PREFIX 'stuhelper.admin.tableColumns'`（PersistentAdminTable.vue:26，合理）；错误码字符串散布于 shared-client.ts:33-37 与 shared-result.ts:18-24、core/auth.ts:50-53（`'A00101'` 前缀匹配出现 3 处，应集中到 shared error-codes 模块）；system-config 的 `emailPolicyKey = 'email.delivery_policy'` 与其表单结构内联在视图中（system-config/index.vue:42-63）。

### 4.5 死代码 / 未使用导出

- **`views/content/logs/index.vue`（74 行）是死代码**：路由 `/content/logs` 指向 `operation-logs/index.vue`（content.ts:64-65），无任何引用；且它没有错误处理、没有竞态守卫，是旧版残留。注意 `router/access.ts:14` 的 `import.meta.glob('../views/**/*.vue')` 仍会把它打成 chunk。
- API 层未被任何视图使用的导出：`releaseMemberBlacklistBySubject`（admission.ts:129-135）、`getFreshmanVerification`（admission.ts:47-51）、`refreshTokenApi`（core/auth.ts:159-161）。
- `adapter/vxe-table.ts` 无任何 import 方（grep 无结果）；`adapter/form.ts` 仅被 bootstrap 初始化但视图零使用 —— 二者基本是 Vben 模板残留负担。

### 4.6 其他

- admission-policy 直接对 fetch 回来的 `policy` 对象做 v-model 就地修改（admission-policy/index.vue:362、397 等），无 dirty 跟踪：编辑一半刷新或保存失败后界面与服务器态可能不一致（fetchData 重置可缓解，但保存失败时不回滚）。
- 多大写连写 prop 名（`guildID/qqID/subjectID`）导致模板里出现 `v-model:guild-i-d`、`@copy-auth-u-r-l` 这类畸形 kebab 名（member-blacklist/index.vue:200-201、admission-sessions/index.vue:241-242），可读性差且易拼错。
- `adminLogger` 只是 console.warn 包装（`src/utils/admin-logger.ts:3-7`），没有接入 web 端已有的前端错误上报指标管线。

---

## 5. 与 Koishi WebUI 的功能重叠

| 功能 | Admin（web-ele） | Koishi WebUI（`bots/koishi/plugins/stuhelper-core/client`） | 差异 |
|---|---|---|---|
| **成员黑名单** | `/users/member-blacklist`，走 `/api/v1/admin/member-blacklist*`（admission.ts:106-135），capability `member_blacklist:read/manage`，可按 platform/scope/source/status 全量筛选 | `BlacklistView.vue`，经 console RPC → `member-blacklist-console-api.ts` → `MemberBlacklistBackend` → **`/api/v1/bot/member-blacklist*`**（bot 服务端点） | 同一份后端数据、不同端点族；koishi 侧按 console 角色 + guild 作用域裁剪可见性（member-blacklist-console-api.ts:63、170-173），admin 是全局管理面。增/释操作两边都能做 → 双写入口 |
| **入群认证（admission）** | 策略权威源 `/users/admission-policy`（写策略），会话管理 `/users/admission-sessions`（resend/regenerate/cancel，走 `/api/v1/admin/admission/sessions/*`），新生材料审核 `/users/freshman-verification` | `AdmissionView.vue` + `admission-runtime` 页面（page-api.ts:23-30），数据来自 group-guard 插件运行态，bot 走 `/api/v1/bot/admission/*`（sessions/member/resend、regenerate、skip、failures/reset 等） | 边界已被刻意声明：admission-policy 页面明确写"Admin 是入群认证策略权威源。Koishi WebUI 只显示同步后的执行态和现场队列"（admission-policy/index.vue:333）。但**会话级操作重叠**：admin 的 resend/regenerate/cancel 与 koishi 的 member/resend、regenerate、skip 是同一会话对象的两条操作路径（admin 经 `/admin/...` 队列化给 bot，koishi 直接 `/bot/...`） |
| 操作反馈/剪贴板/错误工具 | 各视图内联复制 | koishi 有 `use-action-feedback.ts`、`use-confirm.ts`、`utils/clipboard.ts`、`utils/error-message.ts` 等成形 composable | 同类问题两套前端成熟度不同，admin 落后于 koishi 侧的抽象 |

风险点：黑名单两边都可创建/释放，但 admin 创建成功提示等文案硬编码中文且不提示对 koishi 现场（踢人/拒入群）的传播时效；建议在 admin 黑名单/会话页补充类似 admission-policy 的"执行面归属"说明。

---

## 6. 改进机会（按优先级）

1. **抽取 `useAdminListPage` / `useAdminAction` composables**（新建 `src/composables/`）：封装 `fetchRequestSeq` 竞态守卫、loading/loadError/actionError、`adminErrorMessage`。理由：22 处逐字重复（§4.2），任何行为修正（如修复双 toast）目前要改 22 个文件。
2. **统一错误提示层级**：从 `src/api/shared-result.ts:53、69、85、101` 移除 `ElMessage.error`，让视图层（或新 composable）唯一负责展示。理由：当前每次失败双 toast + Alert（§3.1）。
3. **删除死代码**：`views/content/logs/index.vue`（无路由引用）、`adapter/vxe-table.ts`（无 import 方）、`admission.ts:129-135` 等未用导出。理由：`access.ts:14` 的 glob 会把死视图继续打包。
4. **补齐 AdminContentLayout 的 `description` prop**（AdminContentLayout.vue:2-5），或删掉 admission-policy/index.vue:274 的无效传参。理由：现状是静默丢失的说明文案，属于真实 bug。
5. **筛选/分页同步到 URL**：在 list composable 中接 `router.replace({ query })`。理由：管理后台深链/回退/分享是基本诉求，目前全部丢状态（§3.3）。
6. **users 域 i18n 收口**：把 admission-policy/freshman-verification/admission-sessions/member-blacklist 的中文字面量迁入 `locales/langs/*/admin.json`。理由：同一应用一半页面完整双语、一半硬编码（§4.4），`locales/locales.test.ts` 的 zh/en 对齐测试对这些字符串完全失效。
7. **拆分 `open-platform/apps/index.vue`（1025 行）**：把 scope 审批、redirect-uri 审批、生命周期操作、secret 展示各拆为子组件 + `useAppLifecycleActions` composable；现有 ReviewActions.vue、BlacklistTable.vue 已证明该仓库的拆法可行。
8. **统一分页布局与行级 action 锁**：分页统一为带 sizes 的布局并提供共享常量；apps 页全局 `actionLoading`（apps/index.vue:53）改为 admission-sessions 式按 id 锁（admission-sessions/index.vue:27-29）。
9. **修正冗余类型断言**：删除 `FreshmanReviewRow`（freshman-verification/index.vue:31-35、64）直接用生成类型；把 `accountSettingsUrl` 写入 `server/api/openapi.yaml` 的 UserInfo schema，删掉 `core/auth.ts:62-65` 的运行时探测。
10. **`admission-policy` 批量创建改后端批量端点或失败清单反馈**（admission-policy/index.vue:214-220），避免半成功状态。
11. **抽 `copyToClipboard` 到 `src/utils/clipboard.ts`**（消除 apps/index.vue:170-201 与 admission-sessions/index.vue:107-138 的复制），命名与 koishi 的 `client/utils/clipboard.ts` 对齐。
12. **错误码常量收口**：`A00101` 前缀、`A0010204/A0010205` 等分散在 shared-client.ts:33-37、shared-result.ts:30-37、core/auth.ts:50-53，应统一从 `@stuhelper/shared` 的 error-codes 模块导入。

**总体评价**：架构层（OIDC 会话管线、OpenAPI 生成类型贯通、capability 路由权限、竞态守卫意识、测试覆盖）质量明显高于平均；最大短板是视图层 22 份复制的 CRUD 样板、错误反馈双重弹出、users 域 i18n 失守，以及 1 个千行视图文件。
