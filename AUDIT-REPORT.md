# StuHelper 全库审计报告

> **归档状态（2026-08-01）**：本文件应项目 owner 要求从 Git 历史恢复，用于保留 Claude
> 两轮审计的原始汇总、编号和取证。它不是当前任务清单，也不能作为当前严重度、架构决策或
> 完成状态的权威来源；统一后的复核结论、最小修复边界和当前状态以
> [`AUDIT-FINDINGS.md`](AUDIT-FINDINGS.md) 为准。
>
> 本报告包含已经被后续 owner 决策取代的表述，例如“Casdoor 只做身份”“双管理员
> bootstrap”“最后一名管理员保护”和 `super_admin` MFA reset 双人复核。当前决定见
> [`ADR-0009`](docs/adr/0009-casdoor-organization-admin-super-admin-authority.md)：目标 Casdoor
> organization 用户对象的 `IsAdmin` 是 StuHelper `super_admin` 管理权威；项目不要求两个
> 管理员，MFA reset 不要求另一名 `super_admin` 确认。阅读下文时必须把这些旧内容视为历史
> 证据，不得据此恢复旧实现。

本报告的全部内容来自两轮多 agent 审计的执行记录（workflow journal 与 agent transcript），
不依赖 git 历史——该 worktree 有其他 agent 并行修改，git 历史无法可靠归属。

按[文档治理模型](docs/design/documentation-governance.md)，审计报告不属于 `docs/`：
本文件是工作产物，条目应转为 GitHub Issues 与 PR 描述，不作为项目文档提交。

## 1. 结论

两轮共产出 117 条原始发现，经独立对抗验证后 **确认 82 条**，证伪驳回 32 条（驳回率 28%）。

| | 原始发现 | 确认 | 证伪 | 未完成验证 |
|---|---|---|---|---|
| 第一轮 | 61 | 43 | 18 | 0 |
| 第二轮 | 56 | 39 | 14 | 3 |
| **合计** | **117** | **82** | **32** | **3** |

确认问题严重度：P0 × 1、P1 × 20、P2 × 41、P3 × 20。

## 2. 方法与覆盖

### 2.1 机制

审计 agent 按维度并行扫描，每条发现再交由**独立的对抗验证 agent** 复核。验证方的提示词要求
**默认证伪**——只有在代码明确证明缺陷存在时才判定成立。P0/P1 使用两个不同视角复核
（能否真实复现 / 诊断与修复方案是否正确，含「是否已有守卫或测试覆盖」），P2/P3 使用一个。
多数票决定去留，验证方可修正严重度并给出自己的修复方案。

这套机制驳回了 32 条发现，其中包含若干原报 P1 的条目。这些若直接采纳修改，属于无效改动甚至引入回归。

### 2.2 第一轮的覆盖缺口

第一轮 12 个维度中 **3 个零产出**，其 finder agent 反复重试后全部失败：

| 维度 | 尝试 | 结果 |
|---|---|---|
| 后端认证与授权 | 3 次 | 全部失败 |
| Web 客户端 UI | 3 次 | 全部失败 |
| UniAppX + Koishi Web UI | 2 次 | 全部失败 |

这些 agent 的 transcript 是全部 agent 中最大的（0.89–1.37 MB），原因是提示词要求无界穷举
（「列出每个页面的每个控件并逐个追踪」「把路由表每一条路由与中间件链交叉核对」），
agent 读到上下文耗尽而终止。对照组 Admin 维度文件数更多（1299 个）却成功，差别在其任务分层且有优先级。

### 2.3 第二轮的补救

把失败的 3 个维度拆成 **10 个有界子范围**，每个子范围：给定明确文件清单并禁止越界遍历；
发现上限降至 6 条；附上第一轮已知条目要求寻找不同问题。

结果：**10 个子范围全部产出**，新增确认 39 条。有界拆分是这类大仓审计的必要条件。

### 2.4 确认问题分布

| 区域 | P0 | P1 | P2 | P3 | 合计 |
|---|---|---|---|---|---|
| Web 前端 |  | 3 | 9 | 6 | 18 |
| Koishi 机器人 |  | 3 | 7 |  | 10 |
| 评课审核 |  | 2 | 6 | 2 | 10 |
| 入群认证 |  | 1 | 4 | 1 | 6 |
| Admin 前端 | 1 |  | 1 | 3 | 5 |
| 基础设施与运维 |  | 3 | 2 |  | 5 |
| UniAppX |  | 2 | 1 | 1 | 4 |
| 文档 |  | 1 |  | 3 | 4 |
| 中间件 |  |  | 3 |  | 3 |
| 后端公共包 |  |  | 2 | 1 | 3 |
| CI/CD |  |  | 2 |  | 2 |
| OpenAPI 契约 |  |  | 1 | 1 | 2 |
| 外部数据源 |  |  | 2 |  | 2 |
| 后端平台层 |  | 1 |  |  | 1 |
| 开放平台 |  | 1 |  |  | 1 |
| 用户 |  | 1 |  |  | 1 |
| 认证 |  | 1 |  |  | 1 |
| 资源 |  | 1 |  |  | 1 |
| 课程 |  |  | 1 |  | 1 |
| OIDC |  |  |  | 1 | 1 |
| OpenFGA |  |  |  | 1 | 1 |

## 3. 确认问题明细

每条含：位置、验证票数、证据、失败场景、经验证的修复方案。

### P0（1 项）

#### 1. Admin landing page requires admin:dashboard:view, so every non-super_admin role lands on a full-page 404 after login

`clients/admin/apps/web-ele/src/preferences.ts:12`

| | |
|---|---|
| 区域 | Admin 前端 |
| 类别 | broken-core-flow |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
preferences.ts:12 → `defaultHomePath: '/analytics',`
router/routes/modules/dashboard.ts:24 → `authority: [ADMIN_DASHBOARD_VIEW],` on the `/analytics` route
api/core/user.ts:31 → `homePath: preferences.app.defaultHomePath,` (no role awareness)
router/routes/core.ts:34 → Root `path: '/'` has `redirect: preferences.app.defaultHomePath`
router/guard.ts:178-186 → `const redirectPath = (from.query.redirect ?? (to.path === preferences.app.defaultHomePath ? userInfo.homePath || preferences.app.defaultHomePath : to.fullPath)); return { ...router.resolve(decodeURIComponent(redirectPath)), replace: true };`
packages/utils/src/helpers/generate-routes-frontend.ts:14-16 → `const finalRoutes = filterTree(routes, (route) => hasAuthority(route, roles));` (unauthorized routes are REMOVED, not 403-ed, because none set `menuVisibleWithForbidden`)
server/internal/pkg/capability/catalog.go:38-73 → only `super_admin` is granted `AdminDashboardView`; `school_admin`/`section_admin`/`section_moderator` are not.
```

**失败场景**

A `school_admin` (capabilities: admin:reviews:manage, admin:reports:manage, user:student:read/review, user:school:read/update) signs in. `/auth/me` returns canAccessAdmin=true (admin:reviews:manage is in AdminEntryCapabilities), so the session is accepted. login.vue:20-27 sends the OIDC redirect to `preferences.app.defaultHomePath` = `/analytics`. The access guard generates routes, `/analytics` is filtered out because the user lacks `admin:dashboard:view`, so `router.resolve('/analytics')` falls through to the top-level catch-all `/:path(.*)*` → `views/_core/fallback/not-found.vue`. That route is registered outside BasicLayout, so the user gets a bare full-page 404 with no sidebar/menu, and the Fallback component's "back" button uses `homePath: '/'` (packages/effects/common-ui/src/ui/fallback/fallback.vue:20,121) which redirects to `/analytics` again — an infinite 404 loop. The admin is only reachable if the operator hand-types e.g. `/content/reviews`. Same for `section_admin` and `section_moderator`.

**修复方案**

Make the landing path capability-derived, and add a guard-level safety net. Do NOT "fix" it by widening `authority` on dashboard.ts or by adding `meta.menuVisibleWithForbidden`: `GET /admin/stats` is gated on GLOBAL `admin:dashboard:view` (server/internal/app/admin_authorizers.go:72 → `rbac.RequireGlobalCapability(capability.AdminDashboardView)`), so a school_admin would render the dashboard shell and then eat a 403 — a different broken page.

1. New helper `clients/admin/apps/web-ele/src/router/resolve-home-path.ts`: `resolveHomePath(capabilities: string[]): string`. Derive it from `accessRoutes` itself (walk the tree ordered by `meta.order`, reuse `hasAuthority` from `@vben/utils`, pick the first authorized leaf that has a `component` and is not `hideInMenu`) so it can never drift from the route table; fall back to `preferences.app.defaultHomePath` only when nothing matches.

2. `clients/admin/apps/web-ele/src/api/core/user.ts:31`: `homePath: resolveHomePath(me.capabilities)`. `mapMeToUserInfo` already receives the whole `MeResult`, so no signature change. Update `src/api/core/user.test.ts` accordingly.

3. `clients/admin/apps/web-ele/src/router/guard.ts:178-186`: after `accessStore.setIsAccessChecked(true)`, resolve the candidate first and reject a dead target — if `resolved.name === fallbackNotFoundRouteName` (or `resolved.matched` contains only the catch-all), retry with `userInfo.homePath`, then with the first leaf path found in `accessibleMenus`, and only surface the 404 if all of those fail. Apply the same net at guard.ts:100-107 (the already-logged-in-hits-/auth/lo …


### P1（20 项）

#### 2. ensureAdminCommandAccess returns "allowed" when the guild id is empty, and `群审复核` then dumps every guild's review queue

`bots/koishi/plugins/stuhelper-admin/src/command-access.ts:29`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | authorization |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
command-access.ts:27-33
```ts
  const targetGuildId = input.targetGuildId ?? session?.guildId
  const guildId = targetGuildId
  if (!session || !guildId) return          // <- undefined == access granted
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    store.getMemberRoles(guildId, session.userId),
  ])
```
`resolveGuildId` yields `''` off-guild (command-access.ts:45: `return guildId?.trim() || session?.guildId || ''`), and `registerReviewListCommand` passes it straight through (commands.ts:95-107) then calls `deps.moderationStore.listPendingReviews(targetGuildId)`. store.ts:173-177 treats an empty id as "all guilds":
```ts
  async listPendingReviews(guildId?: string) {
    const query = guildId ? { guildId } : {}
```
```

**失败场景**

An operator whose account has Koishi authority >= 3 (e.g. bound to the console admin, authority 5) but who is deliberately excluded from the `guardReviews` command policy DMs the bot `群审复核` with no argument. `session.guildId` is undefined in a private chat, so `targetGuildId` is `''`; `ensureAdminCommandAccess` returns before ever reading the `guardReviews` policy or the caller's member roles, and `listPendingReviews('')` returns the pending review queue for **every** guild — member IDs, action types and free-text reasons — instead of just the guilds the operator governs.

**修复方案**

Three edits; deliberately do NOT apply two parts of the original proposal.

1. `bots/koishi/plugins/stuhelper-admin/src/commands.ts` — `registerReviewListCommand` (lines 91-108). Load-bearing fix for the leak. After the `if (denial) return denial` block, mirror `registerWarningCommand:80` and reuse the existing message key (no schema change needed, `guardWarningMissingContext` = '请在群聊中执行，或显式传入群号和成员 ID。'):
```ts
      if (denial) {
        return denial
      }
      if (!targetGuildId) {
        return adminMessage(messages, 'guardWarningMissingContext')
      }
      return formatPendingReviews(await deps.moderationStore.listPendingReviews(targetGuildId), messages)
```

2. `bots/koishi/plugins/stuhelper-admin/src/command-access.ts` lines 27-33. Close the policy bypass so the policy is always authoritative. Replace the permissive early return with a deny-on-no-session plus an empty-roles evaluation (keeps `minAuthority` enforced off-guild instead of skipping the whole check, and avoids regressing the friendlier group-only messages that `registerBatchMuteCommand`/`createReviewRequest` emit after the access call):
```ts
  if (!session) {
    return renderMessageTemplate(resolveAdminMessages(input.messages).commandAccessDenied)
  }
  const guildId = (input.targetGuildId ?? session.guildId ?? '').trim()
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    guildId ? store.getMemberRoles(guildId, session.userId) : Promise.resolve<string[]>([]),
  ])
```
(the existing `canExecuteCommand` / `allowed ? undefined : denied` tail is unchanged). …

#### 3. Koishi console admission runtime switches are bot-wide but have no global-scope check, unlike every equivalent stuhelper-core endpoint

`bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts:99`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | authz-bypass |
| 验证票数 | 2/2 |
| 严重度 | 原报 P0 → 验证方修正为 P1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
ctx.console.addListener(ADMISSION_RUNTIME_SETTINGS_EVENT, async (input) => {
    await deps.runtimeSettings.saveSettings(parseRuntimeSettingsInput(input))
    await deps.onRuntimeSettingsChanged?.()
    return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'admissionConsoleSettingsSaved')
  }, { authority: CONSOLE_AUTHORITY })

// contrast, stuhelper-core/src/core/api/group-guard-behavior-settings-api.ts:36
//   const scope = await api.resolveConsoleScope(this)
//   assertGlobalConsoleScope(scope, 'group guard behavior settings')
```

**失败场景**

A guild moderator is given Koishi console authority 4 plus a stuhelperGroupCenter role whose guildIds = ['111111'] (the exact configuration console-guild-scope.ts exists to support). They open /stuhelper -> 入群认证 and flip the 消息风控 switch off. AdmissionView.submitRuntimeSetting sends {moderationEnabled:false} to stuhelperGroupGuard/action/save-admission-runtime-settings, which only checks authority:4 and never resolves the guild scope. AdmissionRuntimeSettingsStore.saveSettings merges it into the single global settings row, so keyword filtering, repeat detection and anti-recall stop for EVERY guild the bot serves, not just 111111. The same control set also lets them turn off 群审命令 (adminCommandsEnabled), 兜底扫描 (fallbackScanEnabled) and Action Stream bot-wide. Trying the same thing through the sibling 设置 page is correctly refused with 'group guard behavior settings requires global console scope'.

**修复方案**

Do not try to resolve the scope from inside group-guard's own config — the plugin has no roles source (`inject = { required: ['database'], optional: ['console'] }`, `index.ts:46`; roles live in core's `AuthService.getRoles()`/`getUserRoleIds()`, `plugins/stuhelper-core/src/core/services/auth.service.ts:181,192`). Instead:

1. Add an optional `resolveConsoleScope?: (client: unknown) => Promise<ConsoleGuildScope>` to `AdmissionConsoleAPIDeps` in `admission-console-api.ts`, and wire it in `plugins/stuhelper-group-guard/src/index.ts:150` from the core service when present (`ctx.stuhelperGroupCenter` is declared on `Context`, `stuhelper-group-center.service.ts:41-45`), building it exactly like `plugins/stuhelper-core/src/core/api/index.ts:47-51` does (`resolveRequiredConsoleGuildScope(client, { roles: service.auth.getRoles(), getUserRoleIds, listBindingsByAuthId: aid => ctx.database.get('binding', { aid }) })`). Move the scope helpers (`console-guild-scope.ts`) into `@stuhelper/koishi-shared` so both plugins import one implementation rather than duplicating it.
2. Change the settings listener to a non-arrow function so `this` (the console client) is available, and fail closed:
   `ctx.console.addListener(ADMISSION_RUNTIME_SETTINGS_EVENT, async function (input) { const scope = await requireScope(deps, this); assertGlobalConsoleScope(scope, 'admission runtime settings'); ... })`, where `requireScope` throws (`admission runtime settings requires console scope resolution`) if `deps.resolveConsoleScope` was not supplied — never default to `{kind:'all'}`, since these listeners only ex …

#### 4. Admission runtime page and member actions are not guild-scoped, exposing and mutating guard records outside the operator's guilds

`bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts:91`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | authz-bypass |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
ctx.console.addListener(ADMISSION_RUNTIME_PAGE_EVENT, async () => {
    return buildAdmissionRuntimePageData(ctx, deps)
  }, { authority: CONSOLE_AUTHORITY })

  ctx.console.addListener(ADMISSION_RUNTIME_ACTION_EVENT, async function (input) {
    return handleAdmissionRuntimeAction(ctx, deps, input, this as ConsoleActionClient)
  }, { authority: CONSOLE_AUTHORITY })

// handleAdmissionRuntimeAction (line 232-237) only does:
//   const record = await deps.guardStore.getActiveByID(parsed.recordId)
//   if (!record) throw ... -- no assertConsoleGuildAccess(scope, record.guildId, ...)
```

**失败场景**

Same guilds-scoped console operator (role guildIds=['111111'], authority 4). buildAdmissionRuntimePageData calls deps.guardStore.listActive() with no filter, so the 受限成员队列 table renders memberId, memberName, guildId, admissionSessionID and lastError for pending members of every other guild — data that stuhelperGroupCenter/page/identity deliberately filters via buildScopedIdentityPageData. Worse, each of those foreign rows renders enabled action buttons (availableActionsForMember always returns query/reset-failures/release-blacklist, plus regenerate/skip). Clicking 跳过 on a row from guild 222222 sends {recordId:'<foreign id>', action:'skip'}; skipAdmissionSession cancels the backend admission session, marks the record released and calls bot.muteGuildMember(guildId, memberId, 0), letting an unverified member into a guild the operator has no authority over. 解拉黑 on the same row releases that member's blacklist entry, which stuhelperGroupCenter/blacklist/remove refuses for out-of-scope entries via assertVisibleBlacklistRelease.

**修复方案**

Enforce the existing console guild scope on both group-guard admission listeners, failing closed.

1. Make the scope resolver reachable from group-guard. Either move `console-guild-scope.ts` into `@stuhelper/koishi-shared` (exporting `resolveRequiredConsoleGuildScope`, `hasConsoleGuildAccess`, `assertConsoleGuildAccess`), or — better, matching bots/koishi/README.md which says stuhelper-core owns the WebUI — expose a `resolveConsoleGuildScope(client)` method on the `stuhelperGroupCenter` service and add it to group-guard's `inject.optional`.

2. In `registerAdmissionConsoleAPI`, change both listeners to `async function (…)` so `this` (the console client) is available, resolve the scope, and pass it down:
   - Page: `buildAdmissionRuntimePageData(ctx, deps, scope)`. When `scope.kind === 'guilds'`, filter `activeMembers` by `scope.guildIds.has(record.guildId)` and `bindings` by `hasConsoleGuildAccess(scope, binding.guildId)` before slicing/serializing, and compute `stats.*` (activeMemberCount, backendSyncPendingCount, membersWithAdmissionSessionCount, membersWithLastErrorCount, bindingCount, enabledBindingCount) from the filtered arrays so the counters stop leaking global totals. Templates are global objects — keep them, but derive `stats.templateCount` from templates actually referenced by in-scope bindings, and gate the global `platform` / `scheduler` / `bots` blocks behind `scope.kind === 'all'` (return nulls otherwise).
   - Action: immediately after `const record = await deps.guardStore.getActiveByID(parsed.recordId)` and the not-found check, add `assertConsoleGuildAccess …

#### 5. All four tabBar icons are missing from the build output (static/ lives outside uni-app's input dir)

`clients/uniappx/src/pages.json:100`

| | |
|---|---|
| 区域 | UniAppX |
| 类别 | build-asset-resolution |
| 验证票数 | 2/2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
pages.json tabBar entries reference build-relative assets:
  "iconPath": "static/tabbar/home.png", "selectedIconPath": "static/tabbar/home-active.png"
(and course/review/user, lines 100-119).
The PNGs actually live at clients/uniappx/static/tabbar/ — outside the uni-app input dir, which is `src/` (pages.json, manifest.json, main.ts, App.vue are all under src/). `ls src/static` -> "没有那个文件或目录".
The committed H5 build proves the consequence:
  $ grep -ro "static/tabbar/[a-z-]*\.png" dist/build/h5  ->  8 hits inside assets/index-B6ZxmmQD.js
  $ find dist/build/h5 -name "*.png"                    ->  only assets/home-DJS_ZZ7B.png (the index.html favicon)
  $ ls dist/build/h5                                    ->  assets/  index.html   (no static/ directory at all)
```

**失败场景**

Open the built H5 app at any route. The runtime requests /static/tabbar/home.png, course.png, review.png, user.png plus the four -active variants; every one 404s because the build never copied clients/uniappx/static. The bottom tab bar — the only global navigation in the app — renders four broken/blank image slots on every one of the 13 pages, and the selected-state icon swap does nothing. Same failure for the mp-weixin build, where a missing tabBar iconPath is a hard compile error.

**修复方案**

1. `git mv clients/uniappx/static clients/uniappx/src/static` — uni-app copies `<inputDir>/static` verbatim for every platform, and the dev-server middleware already resolves `/static/**` against UNI_INPUT_DIR, so dev keeps working and pages.json needs no change.

2. Same commit, update clients/uniappx/index.html:6 favicon from `href="/static/tabbar/home.png"` to `href="/src/static/tabbar/home.png"`. This is mandatory, not cosmetic: Vite resolves root-relative HTML asset URLs against `root` (= clients/uniappx), so leaving it pointing at the now-nonexistent root static/ makes the html asset unresolvable at build time. (Cleaner alternative if you dislike a /src/ URL in index.html: set `publicDir: 'public'` in vite.config.ts — vite-plugin-uni honours `config.publicDir || '__static__'` — and put a favicon.png in clients/uniappx/public/.)

3. Close the test gap, which is the real root cause of this shipping. `expectTabBarIconsAvailable` in tests/e2e/surface.spec.ts only ever exercises the dev server, where Vite's root static handler masks the miss. Add a build-output assertion instead, e.g. a script invoked after `build:h5` that fails unless all 8 files exist under dist/build/h5/static/tabbar/, and wire it into the CI job that already runs the uniappx checks. Optionally add a second Playwright project whose webServer previews the built output rather than the dev server, so asset-pipeline regressions are caught by the existing assertion.

#### 6. Review draft from a different course is silently loaded into the post form and can be submitted under the wrong course

`clients/uniappx/src/pages/review/post.vue:70`

| | |
|---|---|
| 区域 | UniAppX |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
const draftResult = await api.draft.getDraft()
const draft = unwrapOptionalData<components['schemas']['ReviewDraft']>(draftResult)
if (draft) {
  form.value.teacherID = draft.teacherID || 0
  form.value.termID = draft.termID || form.value.termID
  form.value.title = draft.title || ''
  form.value.content = draft.content || ''
  form.value.grade = draft.grade || ''
  form.value.ratings = (draft.ratings ?? {}) as ReviewRatings
}
draft.courseID is never compared to the page's courseID, even though the schema carries it (clients/shared/src/types/api.gen.ts:3746) and the endpoint is a single per-user draft with no course filter (clients/shared/src/api/draft.ts:23 `getDraft: () => client.GET('/api/v1/course/review/drafts')`). submitReview then posts `courseID: course.value.id` (the page's course) together with `teacherID: form.value.teacherID` (the draft's). The web client does not do this — PostReviewPage.vue:845 switches the page to `draft.courseID` and prompts the user before restoring.
```

**失败场景**

User saves a draft for course A (content + teacher T_A + ratings), leaves, then opens pages/review/post?courseID=B. loadPage() shows course B's name but fills the body with course A's text and sets teacherID = T_A, a teacher who does not teach course B (the teacher picker renders '选择教师' because find() misses, hiding the mismatch). Tapping 发布 sends {courseID: B, teacherID: T_A, content: <course A's draft>} — either a 4xx the user cannot explain, or a published review that attributes course A's teacher and text to course B. The subsequent deleteDraft() then destroys the course A draft either way.

**修复方案**

In clients/uniappx/src/pages/review/post.vue loadPage(), gate the draft restore on the course matching the page, mirroring the web client's semantics but without the dialog:

const draftResult = await api.draft.getDraft()
const draft = unwrapOptionalData<components['schemas']['ReviewDraft']>(draftResult)
// 单用户仅有一条草稿，可能属于其他课程；仅在课程一致（或草稿未绑定课程）时恢复。
if (draft && (!draft.courseID || draft.courseID === courseID.value)) {
  form.value.termID = draft.termID || form.value.termID
  form.value.title = draft.title || ''
  form.value.content = draft.content || ''
  form.value.grade = draft.grade || ''
  form.value.ratings = (draft.ratings ?? {}) as ReviewRatings
  // teacherID 只在草稿确实绑定到本课程时恢复，避免跨课程教师错配。
  form.value.teacherID = draft.courseID === courseID.value ? (draft.teacherID || 0) : 0
}

Also make the post-submit cleanup non-destructive for other courses: in submitReview, only call api.draft.deleteDraft() when the draft that was loaded belonged to this course. Track the loaded draft's courseID in a ref (e.g. `const loadedDraftCourseID = ref<number | null>(null)` set in loadPage) and guard the cleanup with `if (loadedDraftCourseID.value === null || loadedDraftCourseID.value === course.value.id)`; otherwise leave course A's draft intact.

This breaks no other caller: getDraft/deleteDraft are used only here in uniappx (verified via clients/uniappx/src grep — post.vue is the sole consumer), and the shared package and web client are untouched.

Optional hardening, separate from the client fix: TeacherBelongsToCourseSchoolTx (server/internal/modules/course/review/repository.go:64) valid …

#### 7. OAuth profile-completion sends the user to /account/profile to supply a missing username/email/avatar, but that page is entirely read-only — the authorization can never be completed

`clients/web/src/modules/open-platform/views/ProfileCompletionPage.vue:81`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | broken-flow |
| 验证票数 | 2/2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
ProfileCompletionPage.vue:70-88 renders one row per missing field whose only control is:
  <a class="completion-action" :href="profileCompletionActionURL(field.actionURL)" target="_blank" ...>
    {{ t('common.openPlatformProfileCompletion.openAction') }}

Backend action targets (server/internal/modules/openplatform/service_completion.go:26-33):
  ProfileFieldUsername: {..., ActionURL: "/account/profile"}
  ProfileFieldEmail:    {..., ActionURL: "/account/profile"}
  ProfileFieldAvatar:   {..., ActionURL: "/account/profile"}

But clients/web/src/modules/user/views/AccountProfilePage.vue contains no writable control at all — `grep -n '<input|<button|@click' views/AccountProfilePage.vue` yields only the two `<router-link>`s (lines 16 and 102). Every field is rendered through the read-only `ProfileField` / `DisclosureField` render functions, and the page's own copy says so (en-US/user.ts: "Review the account details, contact fields, verification status..."). No link to the Casdoor account page exists anywhere in the user or open-platform modules either.
```

**失败场景**

A third-party app requests `profile.basic.read` + `email.read` from a user whose Casdoor account has no email. The authorization endpoint redirects to /complete-profile?token=..., which lists "邮箱" / "Email" with an "Open" link to /account/profile. The user clicks it, lands on a read-only summary that just says "Email — Missing", finds no field, form, or outbound link to set it, returns to the completion tab, clicks "Continue" (continueProfileCompletion), and the backend returns the same still-missing state. The OAuth login for that app is a permanent dead end. Identical for a missing username or avatar.

**修复方案**

Two-part fix; both cheap because the plumbing already exists.

1. Backend — server/internal/modules/openplatform/service_completion.go:26-33: stop hardcoding `/account/profile` for identity-provider-owned fields. Turn `completionFieldCatalog` into a Service-scoped lookup that takes the configured public account-settings base (reuse the exact config path behind `buildAccountSettingsURL` / `accountSettingsURLBase` in server/internal/modules/auth/handler_userinfo.go:102-116, i.e. public account URL with Casdoor issuer fallback) and set `ActionURL` for `profile.email` and `profile.avatar` (and `profile.username`, harmless though unreachable) to that absolute `<accountBase>` URL. Leave `/user/phone-binding`, `/user/identity-verification`, `/user/student-verification` untouched — those pages are genuinely writable. Update server/internal/modules/openplatform/service_completion_test.go:32-34 accordingly. The web side needs no change to render it: `accountCenterURLForHref` returns null for a foreign origin and ProfileCompletionPage.vue:225 falls back to the raw absolute URL, which is a valid href with the existing `target="_blank" rel="noreferrer"`.

2. Web — clients/web/src/modules/user/views/AccountProfilePage.vue: render the already-parsed but unused `authStore.user.accountSettingsUrl` as an explicit outbound "Manage at identity provider" link in the profile card header (`target="_blank" rel="noreferrer"`, shown only when present), with a new key in both clients/web/src/locales/en-US/user.ts and zh-CN/user.ts. That makes the page a valid destination for users who arrive from the …

#### 8. Resource upload sends the browser MIME type verbatim; server requires it to equal the sniffed type, so every Office/CSV/legacy-doc upload fails with a generic "try again later"

`clients/web/src/modules/resource/resourceForm.ts:62`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | integration-gap |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
resourceForm.ts:56-64 —
  export async function buildCreateResourcePayload(form, file) {
    return { ...buildResourceMetadataPayload(form),
      filename: file.name,
      contentType: file.type,          // <- raw browser MIME, never validated
      dataBase64: await readFileAsDataURL(file) }
  }

The only client-side gate is size (ResourceEditPage.vue:126 `if (file.size <= 0 || file.size > MAX_RESOURCE_UPLOAD_SIZE)`) and the file input carries no `accept` attribute (ResourceEditPage.vue:317 `<input type="file" class="sr-only" @change="handleFileChange" />`). The i18n hint only mentions size (`resource.ts:79 fileHint: 'Maximum file size is 10 MB'`).

Server, server/internal/modules/resource/service.go:239-244 —
  detectedType := http.DetectContentType(content)
  if provided := strings.TrimSpace(contentType); provided != "" {
      if !resourceMediaTypesMatch(provided, detectedType) {
          return nil, "", ErrResourceContentTypeMismatch
      }
  }
and :248-255 `resourceMediaTypesMatch` compares the *bare* media types for equality.
```

**失败场景**

A student opens /resource/new, fills the form, and picks `lecture-notes.docx`. Chrome sets `file.type = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"`; the client passes it through unchanged. Go's `http.DetectContentType` sees the ZIP magic `PK\x03\x04` and returns `"application/zip"`, so `resourceMediaTypesMatch` is false and the request is rejected with ErrResourceContentTypeMismatch. `submitForm` catches everything into one bucket and renders `resource.form.saveFailed` = "Failed to save the resource. Please try again later." The user retries forever and can never publish .docx/.xlsx/.pptx/.doc, nor .csv (`text/csv` vs sniffed `text/plain`), nor .zip on Windows (`application/x-zip-compressed` vs `application/zip`). Server tests only ever exercise `text/plain` (server/internal/modules/resource/service_integration_test.go) and the Playwright spec mocks the API, so nothing catches it.

**修复方案**

Server-side is the required fix (the client cannot know Go's sniff behavior). In server/internal/modules/resource/service.go: (1) replace the strict equality in resourceMediaTypesMatch with a compatibility check — after mime.ParseMediaType on both sides, accept when bare types are equal, OR when the sniffed type is application/zip and the declared type is a known ZIP container (application/vnd.openxmlformats-officedocument.*, application/vnd.oasis.opendocument.*, application/x-zip-compressed, application/epub+zip, application/java-archive), OR when the sniffed type is text/plain and the declared type is any text/* or application/json (JSON/CSV/Markdown refinements), OR when the sniffed type is application/octet-stream (no signature matched, so it carries no information and must not veto the declaration). Reject only genuine contradictions (e.g. declared image/png with sniffed application/pdf). (2) Change decodePayload to return the effective content type — the declared type when it is a compatible refinement of the sniffed type, otherwise the sniffed type — and pass that to s.storage.Put at service.go:88, so a .docx is stored and later served (service.go:127) as the real Office MIME rather than application/zip. (3) Add table-driven unit tests over resourceMediaTypesMatch covering docx/xlsx/pptx/doc/csv/md/json/zip-on-Windows accept cases plus at least one true-contradiction reject case. Client-side, as defense in depth and for UX: add an explicit accept="..." allowlist to the file input (views/ResourceEditPage.vue:316-320), mirror it in validateSelectedFile(), and stop coll …

#### 9. Web review parser strips userVote, so every vote button renders un-voted and re-clicking silently un-votes while the UI shows +1

`clients/web/src/modules/review/reviewListPayload.ts:163`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
readReviewPayload() rebuilds the Review object field by field and never copies `userVote`:
```ts
    ratings: readReviewRatings(payload.ratings, message),
    likeCount: readInteger(payload, 'likeCount', message),
    dislikeCount: readInteger(payload, 'dislikeCount', message),
    replyCount: readInteger(payload, 'replyCount', message),
```
Every review list in the module flows through this reader (`clients/web/src/api/review.ts` -> `getReviewsPage` / `getLatestReviewsPage` / `searchReviewsPage` / `getBatchCourseReviewsPage` all call `readReviewPagePayload`). The consumers all depend on that field: `modules/review/useReviewVoting.ts:40` `return r.userVote ?? null`, `components/business/review/useReviewVote.ts:22` `ref(reviewGetter().userVote ?? null)`. The backend does send it — `Review.userVote` is in the OpenAPI contract (server/api/components/schemas/review.yaml:63) and `service.go:403/490/525` call `populateUserVotes` on the course-reviews, latest and search endpoints, all of which are registered with `optionalAuthMiddleware` (handler.go:95-98).
```

**失败场景**

User likes review R (likeCount goes 5 -> 6 server-side, vote row = like). User reloads /courses/12/reviews. The API returns `userVote:"like"`, but the parser drops it, so the heart is unfilled and `aria-pressed=false`. The user clicks like again: `applyOptimisticVote` sees userVote=null, so the UI shows likeCount 7 and a filled heart, while `VoteReview` (service_review_write.go:272 `case params.VoteType:`) deletes the vote and decrements to 5. No error is returned, so nothing rolls back — the card shows 7 likes and "liked" while the DB holds 5 and no vote.

**修复方案**

In clients/web/src/modules/review/reviewListPayload.ts, add the missing field to the object returned by readReviewPayload (next to dislikeCount, mirroring contract order). Because the Go side uses `json:"userVote,omitempty"` on a *string, the key is omitted rather than null when there is no vote, so readOptionalEnum suffices — but tolerate null defensively:

  const VOTE_TYPE_VALUES = new Set(['like', 'dislike'])

  function readOptionalVote(record: Record<string, unknown>, key: string, message: string) {
    if (record[key] === null) return undefined
    return readOptionalEnum<NonNullable<Review['userVote']>>(record, key, VOTE_TYPE_VALUES, message)
  }

  // inside readReviewPayload:
  userVote: readOptionalVote(payload, 'userVote', message),

Then extend clients/web/src/modules/review/__tests__/reviewListPayload.test.ts with (a) a case asserting `userVote: 'like'` survives readReviewPayload/readReviewPagePayload, (b) a case asserting an absent key yields undefined, and (c) a case asserting an invalid value ('LIKE') throws, matching the module's existing strict-reader convention. Also worth adding a guard test that every key of components['schemas']['Review'] appears in the reader's output, so the next contract field added is not silently dropped the same way.

#### 10. Bot action SSE stream has no shutdown release, so every SIGTERM stalls 30s and the process exits 1

`server/internal/modules/admission/handler_bot_queries.go:137`

| | |
|---|---|
| 区域 | 入群认证 |
| 类别 | graceful-shutdown |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
handleStreamBotAdmissionActions loops forever with the request context as its only exit:

```go
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-keepalive.C: ...
		case <-ticker.C:
			if !h.writeQueuedAdmissionActions(c, filter) { return }
		}
	}
```

Unlike the notification stream, nothing releases it at shutdown. `server/internal/app/modules.go:78` registers the *only* shutdown hook in the process — `rt.addShutdownHook(notifHub.Stop)` — and `admissionHandler` (modules.go:205-206) registers none. `server/internal/app/server.go:56-59` documents the intent that is not met here: "先停止后台取件并释放 SSE 等长连接… http.Server.Shutdown 本身不会主动取消活动中的长连接处理器". No `http.Server.RegisterOnShutdown` / `BaseContext` is set anywhere (`grep -rn 'RegisterOnShutdown|BaseContext' internal/` → no hits). Go 1.26 `net/http/server.go:3150` Shutdown only polls `closeIdleConns()`; it never cancels an active handler's context. I reproduced it with a minimal serve …
```

**失败场景**

A Koishi bot holds GET /api/v1/bot/admission/actions/stream open (bots/koishi/packages/shared/src/platform/index.ts:45 dials exactly this path). Operator sends SIGTERM for a rolling deploy. server.go:52 computes shutdownTimeout = DB_QUERY_TIMEOUT(5)*3 + 15s = 30s. `srv.Shutdown` cannot drain because the SSE handler is still active, so after 30s it returns context.DeadlineExceeded → server.go:60 logs "Server forced to shutdown" → server.go:71 `errors.Join` returns a non-nil error → app.Run returns it → cmd/stuhelper/main.go:19-20 prints "Application error" and calls os.Exit(1). Result: every clean shutdown takes 30 extra seconds and reports failure (non-zero exit → k8s/compose treats a normal deploy as a crash). Worse, during those 30s Shutdown has already closed the listeners while the SSE handler keeps claiming actions every 2s and pushing them to the bot; the bot performs the kick/release in QQ but its POST /api/v1/bot/admission/actions/{id}/events ack is refused (listener closed, idle keep-alive conns closed), so the action stays `dispatched`, is re-claimed 30s later on the new instance, and the same user is kicked/released twice.

**修复方案**

Give the admission handler a shutdown release symmetric to `notification.Hub`, reusing the `bgCtx` the runtime already cancels first in `beginShutdown`.

1. `server/internal/modules/admission/handler.go`: add `streamStop <-chan struct{}` to the `Handler` struct plus an option `WithStreamShutdown(ctx context.Context) HandlerOption` that stores `ctx.Done()`. Leave it optional — a nil channel blocks forever, preserving current behavior for `handler_errors_test.go:15` and `handler_user_test.go:176`.

2. `server/internal/app/modules.go:199`: pass `admission.WithStreamShutdown(bgCtx)` to `admission.NewHandler`. `router.go:37-39` already creates `bgCtx` and `runtime.go:172-175` cancels it before `shutdownHTTPServer` runs, so no new lifecycle wiring is needed. (Do NOT set `srv.BaseContext` to that context instead — it would cancel every in-flight ordinary request mid-write and defeat draining.)

3. `server/internal/modules/admission/handler_bot_queries.go:137` and `server/internal/modules/admission/handler_user.go:215`: add `case <-h.streamStop:` to both select loops, emitting `c.SSEvent("end", "shutdown")` + `c.Writer.Flush()` before `return` so the bot sees a clean close and reconnects to the new instance rather than logging a stream error.

4. `server/internal/modules/admission/handler_bot_queries.go`: add a hard max-lifetime timer to the bot stream (e.g. `time.NewTimer(10 * time.Minute)`, matching `handler_user.go:212`) that ends the stream with `end`/`timeout`; the plugin's existing reconnect path (`admission-action-stream.ts` `handleStreamError`/`scheduleReconnect`) handles i …

#### 11. No code or ops path ever writes school#admin or section#section_admin/section_moderator tuples, so every scoped admin role is dead

`server/internal/platform/authorization/role_scope_resolver.go:89`

> **最终处置：已实施，原代码位置和原方案已过时。** 原缺口成立，但影响是 scoped role
> fail-closed/不可用，不是越权。当前由 `server/internal/modules/authorization/` 管理固定
> `super_admin`、`school_admin` 与三类 section grant；PostgreSQL desired state、审计和
> outbox 同事务写入，OpenFGA exact projection/readback 后才首次激活。受 step-up MFA 的
> 管理 API、Admin 页面、全量重建、定时漂移修复和 bootstrap 已补齐。下面内容仅保留 Claude
> 当时的历史证据，不能再用其中的 Casdoor role 或旧 resolver 行号描述当前实现。

| | |
|---|---|
| 区域 | 后端平台层 |
| 类别 | missing-provisioning-integration |
| 验证票数 | 2/2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
objects, err := r.scopeReader.ListObjects(ctx, subject, openFGASchoolAdminRelation, openFGASchoolType)
...
sections, err := r.scopeReader.ListObjects(ctx, subject, role, openFGASectionType)
```

**失败场景**

Grepping every `Relation: "..."` literal in non-test Go code yields only: school/section/author/owner/parent tuples (fga/relation_writer.go, user/external_sync.go, cmd/fga-setup/school_tuples.go) and the ecosystem super_admin tuple (repository_auth_sync.go:81). The `admin` relation on `school` and the direct `section_admin`/`section_moderator` assignments on `section` are never written by the server, by fga-setup, by casdoor-bootstrap (which only creates the flat Casdoor role names, cmd/casdoor-bootstrap/settings.go:204-213), or by any infra/ops script. Concretely: give user 7 the school_admin role in Casdoor and log in. ListObjects(user:7, effective_admin, school) returns [] -> resolveSchoolAdminScopes skips the entry -> orgScopedRoles is nil -> capability.ExpandRoleGrants (capability.go:46-50) `continue`s because len(schoolIDs)==0 -> the user has zero capabilities -> adminAuthorizers.Entry (RequireAnyCapability) 403s on GET /api/v1/course/review/admin/reviews. Even if that were bypassed, review can_hide/can_admin_edit Checks deny because school_admin_proxy needs a school#admin tuple. The school_admin and section_* roles cannot be made to work at all.

**修复方案**

Make the app DB the authority for scoped admin grants and project it into OpenFGA, reusing the existing projection pattern rather than inventing a new one:

1. Migration: `scoped_admin_grants(user_id, role, school_id, section_id, granted_by, created_at)` with a CHECK that role='school_admin' implies school_id NOT NULL and section_id IS NULL, and role IN ('section_admin','section_moderator') implies section_id NOT NULL. Unique on (user_id, role, COALESCE(school_id,0), COALESCE(section_id,'')).

2. Projection: on insert/delete, enqueue an outbox job on the existing `iam_openfga_tuple_sync` stream (new job_type e.g. `scoped_admin_grant_projection`). The handler mirrors server/internal/modules/user/external_sync.go:283-345: ReadTuples for the desired object/relation, WriteMissingTuples for additions, DeleteTuples for stale rows — so revocation actually removes `school#admin` / `section#section_admin` / `section#section_moderator`. Reuse fga.ReviewModerationSectionID so projected section IDs satisfy validateReviewModerationSections (role_scope_resolver.go:118-126), otherwise ResolveRoleScopes hard-errors the request.

3. Grant/revoke surface: add endpoints in the user-admin module behind `rbac.RequireGlobalCapability(capability.UserSystemUpdate)` + step-up MFA, strictly Handler -> Service -> Repository with SQL only in the repository and responses only via response.* helpers, plus an audit_events record per grant/revoke. Extend cmd/fga-setup (or a small cmd/scoped-admin-grant) to seed/reconcile from the table so bootstrap and drift repair are covered.

4. Short-term, low-risk mi …

#### 12. pg_basebackup invocation is unconditionally rejected, so physical base backups and PITR never exist

`infra/ops/backup-postgres.sh:63`

| | |
|---|---|
| 区域 | 基础设施与运维 |
| 类别 | backup-integrity |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
compose run --rm --no-deps -T \
    postgres-client \
    pg_basebackup \
      --dbname "${replication_url}" \
      --format=tar \
      --gzip \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=- \
    >"$output_file"
```

**失败场景**

PostgreSQL refuses `-X stream` when tar output goes to stdout. Verified against the exact pinned image used in production (`cgr.dev/chainguard/postgres:latest@sha256:dc2f04037c1044a22af76cee4de70b9111885b17c561b939d7ed70103d100759`): `pg_basebackup: error: cannot stream write-ahead logs in tar mode to stdout` (exit 1). This is argument validation, so it fires before any connection — it can never succeed. Consequences: the weekly `stuhelper-postgres-basebackup.timer` installed by `infra/ops/install-backup-timers.sh:78-88` runs `run-scheduled-backup.sh basebackup`, which aborts at line 36 under `set -e`, so `backups/postgres/base/` only ever accumulates 0-byte `.tar.gz` files created by the `>` redirect (and the 15-minute sync timer then uploads those empty files to object storage). `infra/ops/restore-postgres-basebackup.sh` therefore has nothing to restore and PITR is impossible. Nothing catches it: `infra/ops/postgres-backup-evidence.sh:170-190` only inspects `${logical_dir}` for `*.dump` and never looks at the base directory, and `remote-preflight.sh:150-160` only warns about the logical directory, so `make prod-backup-evidence` and the deploy-time gate stay green.

**修复方案**

Three changes:

1. `infra/ops/backup-postgres.sh` — fix the invocation and stop leaving poisoned artifacts.
   In `run_basebackup()` (lines 53-67) replace `--wal-method=stream` with `--wal-method=fetch`, which is the only WAL method PostgreSQL permits with `--format=tar --pgdata=-` (verified: `fetch` passes arg validation on the pinned image; `stream` is fatally rejected). Keep `--format=tar --gzip --checkpoint=fast --pgdata=-` so `restore-postgres-basebackup.sh`'s `tar -xzf` stays compatible (fetch writes the required WAL into base.tar, which extracts into `pg_wal/`).
   Also make both `run_dump` and `run_basebackup` redirect to `"$output_file.partial"` and `mv` it into place only after the command succeeds (or `trap`-remove the partial on failure), so an aborted run never leaves a 0-byte `.tar.gz`/`.dump` in `backups/postgres/{base,logical}` for the 15-minute sync timer to upload. Optionally pass `--slot`/`--create-slot` (or ensure a non-zero `wal_keep_size`) so a long `fetch`-mode backup cannot lose WAL to recycling.

2. `infra/ops/postgres-backup-evidence.sh` — close the blind spot so this class of failure cannot stay green.
   Add `base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"`, call the fetcher for the base artifacts as well as `logical`, and run the existing `verify_sha256_sidecar` + a freshness check (e.g. mtime within `BACKUP_BASE_RETENTION_DAYS`/8 days, matching the weekly `Sun 03:45` timer) against `latest_file "${base_dir}" '*.tar.gz'` both locally and fetched. Emit `localBaseBackup`/`fetchedBaseBackup` in the JSON bundle. `verify_sha256_side …

#### 13. Deploy bundle strips .env.example / .env.prod.example, which every remote deploy and rollback requires

`infra/ops/build-deploy-bundle.sh:33`

| | |
|---|---|
| 区域 | 基础设施与运维 |
| 类别 | deployment-correctness |
| 验证票数 | 2/2 |
| 严重度 | 原报 P0 → 验证方修正为 P1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
tar \
    --exclude='.git' \
    --exclude='.claude' \
    --exclude='.run' \
    --exclude='.tools' \
    --exclude='.env*' \
    --exclude='.env' \
... (GNU tar matches --exclude patterns against unanchored path suffixes, so '.env*' also drops '.env.example' and '.env.prod.example')
```

**失败场景**

Verified empirically. (1) tar behaviour: building a fixture tree with `.env.example`, `.env.prod.example`, `normal.txt` and running the exact exclude list produces an archive containing only `./normal.txt` — both template files are gone. (2) Consumers require them: `infra/ops/lib/common.sh:753-761` `ensure_env_file()` does `[[ -f "${template_file}" ]] || die "missing env template: ${template_file}"` unconditionally (before it checks whether ENV_FILE already exists), and ENV_TEMPLATE_FILE defaults to `${REPO_ROOT}/.env.example` (common.sh:7); `.deploy/remote.env` written by `init-remote-deploy-config.sh` never sets ENV_TEMPLATE_FILE. (3) `infra/ops/validate-runtime-image-scan.py:229-230` calls `parse_env_file(repo_root / env_path)` for every image whose `env_files` is `[".env.example", ".env.prod.example"]` (all 17 registry images), and `parse_env_file` raises PolicyError on OSError. Reproduced: a repo root containing only `infra/security/runtime-images.json` + the validator gives `[runtime-image-scan][error] cannot read /tmp/bundleroot/.env.example: [Errno 2] No such file or directory` (exit 1). `infra/ops/bootstrap-ubuntu2404.sh:58-84` only `install -d`s directories and `touch`es `.env.prod.*`; it never clones the repo, so the bundle is the only source of repo files on the host. Net effect: `.github/workflows/deploy.yml:134` `remote-preflight.sh` dies at `load_env` with "missi …

**修复方案**

Three coordinated edits.

1) /home/wztxy/Code/StuHelper/infra/ops/build-deploy-bundle.sh — delete line 33 (`--exclude='.env*' \`). The nine explicit exclusions already present at lines 34-42 (`.env`, `.env.generated`, `.env.generated.secrets`, `.env.prod.local`, `.env.prod.shared`, `.env.prod.secrets`, `.env.prod.secrets.local`, `.env.prod.generated`, `.env.prod.generated.secrets`) cover every real root runtime env file. Add the ones the enumeration is missing, all of which are gitignored and therefore invisible to the clean-worktree gate:
    --exclude='.env.casdoor-bootstrap.local' \
    --exclude='.env.local' \
    --exclude='**/.env.local' \
    --exclude='**/.env.development.local' \
    --exclude='**/.env.production.local' \
Do NOT try to re-add the templates as extra tar operands — GNU tar applies --exclude to command-line names too, so that silently does nothing.

2) Same file, after the `tar` subshell (before `mv "${tmpfile}" "${OUTPUT_FILE}"` at line 63) — add a fail-closed contents assertion so this can never regress in either direction:
  bundle_entries="$(tar -tzf "${tmpfile}")"
  for required in ./.env.example ./.env.prod.example; do
    grep -Fxq -- "${required}" <<<"${bundle_entries}" \
      || die "deployment bundle is missing required env template: ${required}"
  done
  if leaked="$(grep -E '(^|/)\.env($|\.)' <<<"${bundle_entries}" | grep -v '\.env\.example$' | grep -v '\.env\.prod\.example$')"; then
    die "deployment bundle contains real env files: ${leaked}"
  fi
Rationale for the second half: `.env.example` / `.env.prod.example` are the only git-trac …

#### 14. Calendar-expiring image pin reviews hard-block production deploy and make rollback to older releases impossible

`infra/security/runtime-images.json:109`

| | |
|---|---|
| 区域 | 基础设施与运维 |
| 类别 | release-engineering |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
"id": "CASDOOR_DEV",
      "image": "casbin/casdoor:latest@sha256:d7658640...",
      "scope": "optional loopback-only local identity-provider development fixture",
      "pin_review": {
        "verified_on": "2026-07-29",
        "review_by": "2026-08-05",   <-- 6 days from today (2026-07-30)
```

**失败场景**

`infra/ops/validate-runtime-image-scan.py:236-246` enforces `require(end >= today, f"{end_field} expired on ...")` for every moving-tag pin inside `validate_policy`, i.e. also under `--policy-only`. Both `infra/ops/prod-deploy.sh:38-41` and `infra/ops/remote-preflight.sh:56-59` run `--policy-only`. Reproduced: `validate-runtime-image-scan.py --policy-only --today 2026-08-13` → `[runtime-image-scan][error] images[0].pin_review.review_by expired on 2026-08-12` (exit 1). Two concrete failures: (a) On 2026-08-05 the *development-only* Casdoor fixture pin lapses and every production deploy aborts, even though `casdoor` is `profiles: [dev-full]` and never runs in prod (7 pins total lapse by 2026-08-12; the CASDOOR_DEV exception at line 299-302 expires the same day). (b) `.github/workflows/rollback.yml:52-56` checks out `inputs.commit_sha` (the OLD release commit) and `build-deploy-bundle.sh` packages that commit's `infra/security/runtime-images.json`. Because `max_pin_review_days` is capped at 30, rolling back to any release older than ~30 days is permanently blocked: during an incident, `prod-rollback.sh` → `prod-deploy.sh:38` fails with an expired-review error and there is no override flag.

**修复方案**

Split the calendar-freshness control away from the deploy-time immutability control.

1. `infra/ops/validate-runtime-image-scan.py`: add `--review-windows {enforce,report}` (default `enforce`). Thread an `enforce_freshness: bool` through `validate_policy` into `validate_review_window` and make ONLY the `require(end >= today, ...)` assertion at line 110 conditional — when `report`, print `[runtime-image-policy][warn] {end_field} expired on {end}` to stderr and continue. Keep every other assertion unconditional in both modes: `start <= today`, `end >= start`, the `(end - start).days <= maximum_days` cap, `DIGEST_REF_RE` immutability, the `.env.example`/`.env.prod.example` exact-match check, `upstream_evidence`, and `validate_effective_environment`.

2. `infra/ops/prod-deploy.sh:38-41` and `infra/ops/remote-preflight.sh:56-59`: append `--review-windows report`. Keep `--policy-only --effective-environment production` so digest immutability and the production env-match gate still fail closed and `infra/ops/tests/runtime-image-security-contract.sh:135-136` still passes.

3. Keep `enforce` (the default) in `infra/ops/scan-runtime-images.sh:74-77` and `:144-148`, i.e. the CI `runtime-image-security` job stays the hard gate for stale pins.

4. Give CI a chance to catch the lapse before deploy time: add a `schedule: - cron: "0 1 * * *"` trigger to `.github/workflows/ci.yml` and include it in the `if:` at lines 557-560 for `runtime-image-security`, so a window lapsing on the calendar fails a scheduled run instead of only surfacing when someone tries to ship.

5. Decouple dev-only fixt …

#### 15. Open-platform phone.read disclosure can never succeed: users.phone_enc holds a masked phone, but the disclosure path requires 11 consecutive digits

`server/internal/modules/openplatform/service_disclosure.go:598`

| | |
|---|---|
| 区域 | 开放平台 |
| 类别 | integration-gap |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
service_disclosure.go:594-604
	phone, err := s.phoneCipher.Decrypt(projection.PhoneEnc)
	...
	normalized, ok := normalizeCasdoorMainlandPhone(phone)
	if !ok {
		return fmt.Errorf("%w: phone projection is unavailable", ErrDisclosureUnavailable)
	}
	out["phone"] = normalized

with `var mainlandPhoneDigitsPattern = regexp.MustCompile(`1[3-9]\d{9}`)` (line 14).

The only writer of users.phone_enc encrypts the MASKED value — server/internal/modules/user/service_phone.go:29-32:
	func (s *Service) buildPhoneProjection(phone string) (string, []byte, string, error) {
		trimmed := strings.TrimSpace(phone)
		masked := phoneutil.Mask(trimmed)
		phoneEnc, err := s.docCipher.Encrypt(masked)

Both sides use the same cipher (internal/app/modules.go:307 `user.NewService(userRepo, crypto.GetHMACKey(), piiCipher, ...)` and internal/app/modules_openplatform.go:39 `openplatform.WithPhoneDecryptor(piiCipher)`), and openplatform/repository_projection.go:18 reads `u.phone_enc` from the same `users` table. `grep -rn "phone_enc"` confirms no other writer.
```

**失败场景**

User binds 13812345678 via POST /api/v1/user/profile/bind-phone → users.phone_enc = Encrypt("138****5678"). An approved third-party app with an approved+consented `phone.read` scope calls GET /api/v1/open-platform/phone. addPhonePayload decrypts "138****5678"; mainlandPhoneDigitsPattern finds no 11-digit run, so ok=false and the request returns ErrDisclosureUnavailable → HTTP 503 "open platform disclosure unavailable", plus a bogus `open_platform.disclosure.denied` audit event with reason `payload_unavailable`. The OpenAPI contract (server/api/openapi.bundled.yaml:937 getOpenPlatformPhone) promises a 200 DisclosureResponse. The same failure hits the OIDC id_token path (UserInfoForIdentityToken with the `phone` scope). Profile completion does not catch it: RequiredProfileFields only checks `projection.PhoneVerified` (service_completion.go:83-86), which is true. No test exercises addPhonePayload — no test seeds phone_enc.

**修复方案**

Implement the documented behavior (real-time Casdoor read) rather than patching the symptom. Concretely: (1) server/internal/modules/openplatform/service.go:48 - add `type phoneProvider interface { GetPhone(ctx context.Context, subject string) (string, error) }` and a `WithPhoneProvider` ServiceOption; drop `phoneDecryptor`/`WithPhoneDecryptor` once unused. (2) server/internal/modules/openplatform/models.go:326 - add `CasdoorSubject string` to UserProjection, and select `u.casdoor_subject` in repository_projection.go:12-33 (column confirmed to exist: user/repository_academic.go:148). (3) service_disclosure.go:586-606 - rewrite addPhonePayload: keep the `!projection.PhoneVerified` early return as `phoneVerified:false` (200), then call `s.phoneProvider.GetPhone(ctx, projection.CasdoorSubject)` and pass the result to the existing normalizeCasdoorMainlandPhone (it already tolerates Casdoor's "+86" prefix via FindString); on provider error or unnormalizable value stay fail-closed with ErrDisclosureUnavailable per docs/design/security-model.md:46, but return `phoneVerified:false` + 200 when Casdoor reports no phone, so "unbound" is not reported as "service unavailable". Delete the phone_enc decrypt - the mask carries no usable information. (4) server/internal/app/modules_openplatform.go:39 - add a `GetPhone` method to casdoorUserProfileGateway (modules_openplatform.go:270-283) forwarding to platformcasdoor.UserProfileClient.GetPhone (platform/casdoor/user_profile.go:43) and pass it via WithPhoneProvider; the gateway is already constructed for the user module, so wiring cost is ne …

#### 16. Migration guides instruct editing the baseline schema file, which silently drops schema changes on any migrated database

`docs/guides/database-migrations.md:20`

| | |
|---|---|
| 区域 | 文档 |
| 类别 | doc-accuracy |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
docs/guides/database-migrations.md:18-21
  - 当前项目按绿地 schema 管理，不保留增量迁移兼容链路。
  - 结构变更直接更新 `000001_initial_schema.up.sql`。
docs/guides/database-migrations.md:31  # 仅本地可用：删除当前 schema baseline
  DATABASE_URL='postgres://...' make migrate-down-one
docs/guides/backend-development.md:73-74
  - 结构变更直接更新 `server/migrations/000001_initial_schema.up.sql`
  - `server/migrations/000001_initial_schema.up.sql` 是唯一 schema 权威来源
docs/README.md:70  | 某张表的列 / 索引 | `server/migrations/000001_initial_schema.up.sql` |

Reality: server/migrations/ holds 000001 through 000019 (…000019_notification_idempotency.up.sql adds notifications.idempotency_key). server/Makefile:19 uses golang-migrate v4 and server/cmd/migrate-runtime calls m.Up(). docs/reference/database.md:11 says the opposite: "`000001_initial_schema.up.sql` 只是基线，不代表后续演进后的完整 schema".
```

**失败场景**

A backend dev follows docs/guides/backend-development.md:73 and adds a column by editing 000001_initial_schema.up.sql, then runs `make migrate-up` against dev/staging. schema_migrations already records version 19, so golang-migrate returns ErrNoChange and migrate-runtime exits 0 with no DDL applied. The developer sees a green migration, the column never exists, and the first query against it fails at runtime in every already-provisioned environment (dev, staging, production). The doc-recommended rollback `make migrate-down-one` is also described as "删除当前 schema baseline" but actually only reverts 000019.

**修复方案**

Docs-only change, plus one Makefile help correction. Do NOT "fix" this by regenerating 000001 into a full schema dump — 000017 and 000019 use bare ADD COLUMN without IF NOT EXISTS, so a fattened baseline would make fresh installs fail with duplicate-column and mark schema_migrations dirty.

1. docs/guides/database-migrations.md
   - Line 5 frontmatter: `authoritative-source: server/migrations/` (not the 000001 file). Bump last-verified.
   - Lines 13-14: describe 000001 as the initial baseline only, explicitly noting it does not reflect later evolution; keep 000001.down.sql as local-only baseline teardown.
   - Lines 19-21 维护规则, replace wholesale with: schema changes are added as the next numbered up/down pair (currently 000020_*); 000001 and every already-applied migration is immutable, because golang-migrate does not checksum migration files — editing an applied file is silently ignored (m.Up() returns ErrNoChange and migrate-runtime exits 0) so the change never reaches any provisioned database; every up must ship a matching down; prefer additive DDL and use IF NOT EXISTS / DROP CONSTRAINT IF EXISTS so re-runs are safe.
   - Line 34 comment and line 80 rollback text: change "删除当前 schema baseline" to "回滚最新一个 migration（仅本地）", and note it reverts one version only (today: 000019, which drops notifications.idempotency_key plus its constraint and unique index).
   - Lines 46, 49, 67: replace "应用当前 baseline schema" with "按版本顺序应用 server/migrations/ 中全部未应用的 migration".

2. docs/guides/backend-development.md:73-74 — replace both bullets with: 结构变更新增下一个编号的 up/down migration；`server/ …

#### 17. Demoting a super_admin never deletes the ecosystem super_admin tuple, so platform-wide OpenFGA power is permanent

`server/internal/modules/user/repository_auth_sync.go:75`

> **最终处置：根因已由控制面切换消除，原 claim-driven 修复不再采用。** Casdoor role
> membership 和 token claim 现在都不能授予、恢复或撤销 StuHelper 权限；相关 parser、
> catalog、credential、worker 与运行配置已经退役。`super_admin` 只由 DB grant 管理，
> revoke 提交后 access snapshot 立即排除，随后 worker 对完整 direct tuple 执行
> higher-consistency exact read、`on_missing=ignore` delete 和 absence verify。下面的证据与
> 修复建议是过渡架构历史记录，不是当前运行模型。

| | |
|---|---|
| 区域 | 用户 |
| 类别 | stale-tuple-privilege-retention |
| 验证票数 | 1/1 |
| 严重度 | 原报 P0 → 验证方修正为 P1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
type roleFGAClient interface {
	WriteMissingTuples(ctx context.Context, desired []fga.Tuple) error
}
...
func (r *UserSyncRepository) syncGlobalRoleRelations(ctx context.Context, userID int64, roles []string) error {
	if r.roleFGA == nil || !hasSyncRole(roles, superAdminRoleName) {
		return nil
	}
	err := r.roleFGA.WriteMissingTuples(ctx, []fga.Tuple{{
		User:     "user:" + strconv.FormatInt(userID, 10),
		Relation: superAdminRoleName,
		Object:   "ecosystem:stuhelper",
	}})
```

**失败场景**

User 42 is a super_admin; login writes ecosystem:stuhelper#super_admin@user:42. An operator removes the super_admin role in Casdoor (user 42 now has only school_admin). On the next login UpsertUser -> syncGlobalRoleRelations takes the early `return nil` branch and no delete happens — the roleFGAClient interface has no delete method at all, and nothing else in the repo deletes ecosystem tuples (grep of DeleteTuples: only user/external_sync.go user_profile projection and openplatform resource grants). Because school.effective_admin = admin or super_admin from parent, ListObjects(user:42, effective_admin, school) still returns every school, so role_scope_resolver hands user 42 school_admin scope over the whole platform, and every FGA Check (can_hide / can_restore / can_admin_delete / can_admin_edit / report can_process) on every review of every school still returns allowed. Revoking the role in the IdP has zero effect on resource authorization.

**修复方案**

1) Extend roleFGAClient in server/internal/modules/user/repository_auth_sync.go to `WriteMissingTuples(ctx, []fga.Tuple) error` + `DeleteTuples(ctx, []fga.Tuple) error` (both already exist on *fga.Client, so the existing wiring at app/modules_auth.go:28 keeps compiling; the openplatform/user interfaces are separate and unaffected).
2) Make syncGlobalRoleRelations a reconcile instead of an early return:
   - build tuple := fga.Tuple{User: "user:"+id, Relation: "super_admin", Object: "ecosystem:stuhelper"}
   - if hasSyncRole(roles, superAdminRoleName): WriteMissingTuples (current behavior)
   - else: DeleteTuples(ctx, []fga.Tuple{tuple}) treating not-found as success (fga.Client.DeleteTuples already tolerates absent tuples per server/internal/pkg/fga/client_edge_test.go:53-56 — verify, and swallow the not-found error explicitly if not).
3) Guard against non-authoritative role projections, otherwise this fix itself revokes real super_admins: only take the delete branch when the roles claim was actually present. Concretely, thread a flag (e.g. usersync.Input.RolesProjected, set from oidc.Claims where ParseProviderRolesFromRaw found the claim key) and skip the delete when it is false/len(roles)==0; log a warning so a misconfigured application token is visible instead of silently skipping reconciliation.
4) Emit an audit record on the delete path (revocation of a platform-wide grant must be traceable) and keep the write path unchanged.
5) Add unit tests for the three cases (role present -> write, role absent with projected roles -> delete, roles unprojected -> no-op) — the funct …

#### 18. Access-token blacklist TTL comes from local TOKEN_ACCESS_TTL, not the token's own exp, so logout leaves a usable Bearer token after ~5 minutes

`server/internal/modules/auth/service.go:266`

| | |
|---|---|
| 区域 | 认证 |
| 类别 | revocation-bypass |
| 验证票数 | 2/2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
service.go:265-274 — `if accessToken != "" { if blErr := s.tokenService.GetBlacklist().Add(ctx, accessToken, s.tokenService.GetAccessTokenTTL()); ... }`
session.go:233-236 — `if data.AccessTokenHash != "" { if blErr := blacklist.AddByHash(ctx, data.AccessTokenHash, accessTTL); ... }` with `accessTTL = s.tokenService.GetAccessTokenTTL()` (service.go:253).
`GetAccessTokenTTL()` (pkg/token/service.go:62-65) returns `TOKEN_ACCESS_TTL` (default 300s, config.go:468) possibly overridden by systemconfig. Nothing in pkg/token, pkg/oidc or modules/auth ever parses the access token's `exp` claim to size the blacklist entry — grep for `exp` in those trees only hits the dead self-signed `JWTClaims.Exp`. The access credential is the Casdoor-issued raw ID token (handler_login.go:205, 667); its real lifetime is set by the Casdoor application ("Token expire", default 168h), completely independent of TOKEN_ACCESS_TTL.
```

**失败场景**

Native client logs in via POST /api/v1/auth/exchange-native and receives `accessToken` = raw ID token whose Casdoor `exp` is 7 days out. The token leaks (device backup, HTTP log, analytics SDK). User calls POST /api/v1/auth/logout: `RevokeSession` blacklists the token hash with TTL=300s and deletes the session. 301 seconds later the Redis blacklist key expires. The attacker replays `Authorization: Bearer <old ID token>`: `resolveToken` (middleware/auth.go:74-102) finds no blacklist entry, Casdoor introspection still reports `active:true` (only the refresh token was sent to the revocation endpoint), the app-id check passes, and — because the Bearer branch performs no session lookup — the deleted session is never consulted. The attacker holds full authenticated access as that user for the remaining ~7 days. The same 300s window applies to LogoutAll / admin-driven `RevokeAllSessions`, so incident-response "kill all sessions" also only holds for 5 minutes.

**修复方案**

Two parts; the second is the durable fix.

1. Size blacklist entries from the token's real expiry instead of TOKEN_ACCESS_TTL.
 - Add `AccessTokenExpiresAt int64` to `token.SessionData` (server/internal/pkg/token/session.go:84) and to `SessionTouchUpdate`, populated at session create/rotate from the already-verified ID token claim (`claims`/`idToken.Expiry` in handler_login.go and handler_refresh_oidc.go) — no new parsing on the hot path.
 - In `revokeLoadedSession` (session.go:230) and `Service.RevokeSession` (service.go:266), compute `ttl := clamp(time.Until(exp)+skew, minBlacklistTTL, maxBlacklistTTL)` and fall back to `max(GetAccessTokenTTL(), providerAccessTTL)` only when no exp is known.

2. Bind the Bearer path to the session store so revocation stops depending on TTL arithmetic. Require native clients to send `X-Stuhelper-Session-ID` on authenticated requests (they already do on refresh/logout) and, in the bearer branch of `resolveToken` (middleware/auth.go:76), load the session and verify `sessionAccessTokenMatches(tokenString, session.AccessTokenHash)` exactly as `resolveOIDCCookieToken` does. Then a deleted session immediately invalidates the token, with the blacklist as defence in depth.

Supporting hygiene: have casdoor-bootstrap/config validation reject a Casdoor application `ExpireInHours` that exceeds TOKEN_ACCESS_TTL (or emit a startup drift warning), and correct docs/design/auth-and-session.md:79, which currently claims natural expiry is 5 minutes when the provisioned provider expiry is 1 hour.

#### 19. GET /course/review/reviews/{reviewID}/replies has no optional-auth middleware, so isOwner is always false and users lose the delete button on their own replies

`server/internal/modules/course/review/handler.go:116`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
handler.go:116 — no auth middleware at all:
	r.GET("/reviews/:reviewID/replies", h.GetReplies)

compared with the sibling read routes, e.g. handler.go:95:
	r.GET("/courses/:courseID/reviews", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourseReviews)

review_read/review_reply.go:21 calls h.resolveOptionalUserHash(c), which is request_identity.go:26-30:
	userID := middleware.GetUserID(c)
	if userID == "" { return "", true }

middleware.GetUserID reads only CtxKeyUserID, which is set exclusively by setClaimsToContext in the auth middleware (pkg/middleware/auth_context.go:122,150). service_interaction.go:434-437 then never sets ownership:
	if params.UserHash != "" { list[i].IsOwner = list[i].UserHash == params.UserHash }

The group is created without .Use(): internal/app/modules_course_metrics.go:74 `api.Group("/course/review")`.
```

**失败场景**

A logged-in student posts a reply (POST .../replies returns Reply{IsOwner: true}, so the delete button appears). They reload the page; the client calls GET /api/v1/course/review/reviews/{id}/replies. GetUserID returns "" because no auth middleware ran, so every reply comes back with isOwner:false. clients/web/src/components/business/review/ReplyCard.vue:13 renders the delete control under `v-if="reply.isOwner"`, so the user can never delete their own reply after a refresh — even though DELETE /replies/{replyID} would succeed. The contract encodes the same bug: server/api/openapi.bundled.yaml:4152 declares `security: []` for getReplies while requiring `isOwner` in the response, whereas optional-auth endpoints use `security: [{}, cookieAuth: [], bearerAuth: []]` (line 3687).

**修复方案**

1. server/internal/modules/course/review/handler.go:116 — put the replies list route on the same optional-auth chain as its siblings:
   `r.GET("/reviews/:reviewID/replies", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetReplies)`
   This makes `setClaimsToContext` populate CtxKeyUserID for authenticated callers, so `resolveOptionalUserHash` returns a hash and service_interaction.go:436 sets IsOwner correctly; anonymous callers still pass through (auth.go:178-182) and get isOwner:false.

2. server/api/paths/review-reply.yaml:6 — replace `security: []` on getReplies with the optional-auth triple used by getCourseReviews (review-crud.yaml:6-9):
   security:
     - {}
     - cookieAuth: []
     - bearerAuth: []
   Optionally add `'503': $ref: '../components/responses/common.yaml#/ErrorResponse'` to getReplies' responses, since RequireHealthyOptionalAuth can now return 503 (auth.go:243).

3. Regenerate the contract — required or CI drift checks fail: `cd server && make bundle-spec` (regenerates api/openapi.bundled.yaml, guarded by `make check-bundled-drift`). Security-only edits do not change api.gen.ts or internal/api/gen types, but running `make generate` is harmless and keeps gen artifacts consistent. Do not hand-edit openapi.bundled.yaml or api.gen.ts.

4. Add a route-level regression test (e.g. in server/internal/modules/course/review/route_contract_test.go) that registers routes with an instrumented optional-auth stub and asserts the stub runs for `GET /api/v1/course/review/reviews/:reviewID/replies` — a stub setting middleware.CtxKeyUserID plus an …

#### 20. ProcessReport applies review status changes with no state-transition guard, letting a user-deleted review be resurrected and later republished

`server/internal/modules/course/review/service_report.go:188`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
service_report.go:182-198 (the "hide" branch) reads the current status but never validates the transition:
	case "hide":
		reportStatus = ReportStatusResolved
		currentStatus, currentCourseID, currentTeacherID, err := s.repo.GetReviewStatusCourseTeacherTx(ctx, tx, report.ReviewID)
		if err != nil { return err }
		if err := s.repo.UpdateReviewStatus(ctx, tx, report.ReviewID, StatusHidden); err != nil { return err }

The sibling entry point for the same mutation does guard it — service_admin.go:27-31 / 161-168:
	var validTransitions = map[string]map[string]bool{
		"hide":    {StatusPublished: true, StatusPendingReview: true},
		"restore": {StatusHidden: true, StatusPendingReview: true},
		...
	if !allowed[currentStatus] { return "", fmt.Errorf("%w: cannot %s from %s", ErrInvalidTransition, action, currentStatus) }

repository_review_query.go:162 has no status filter:
	_, err := tx.Exec(ctx, `UPDATE reviews SET status = $2, updated_at = NOW() WHERE id = $1`, reviewID, status)
and reviewCoreFieldsBaseQuery (repository_review_query.go:15) selects by id only.
```

**失败场景**

1) User A publishes a review; User B reports it (ReportReview requires status='published', so the report row exists with status=pending). 2) User A deletes their own review → reviews.status='deleted'. 3) A moderator calls PUT /admin/reports/{reportID} with action="hide". ProcessReport only checks `report.Status != ReportStatusPending`, so it proceeds and flips the review from 'deleted' to 'hidden'. 4) Any moderator then calls PUT /admin/reviews/{reviewID} with action="restore": validTransitions["restore"] permits StatusHidden, so the review is set back to 'published' and IncrementCourseReviewCount runs — content the user deliberately deleted is publicly visible again and counted in the course review total. The direct PUT /admin/reviews path correctly rejects both hide-from-deleted and restore-from-deleted.

**修复方案**

Primary fix in server/internal/modules/course/review/service_report.go, inside the ProcessReport transaction (lines 179-216):

1. Hoist the review-state read out of the switch so both "hide" and "delete" read `currentStatus, currentCourseID, currentTeacherID` from GetReviewStatusCourseTeacherTx once (it already takes FOR UPDATE), and wrap pgx.ErrNoRows as ErrReviewNotFound for consistency with applyAdminReviewActionTx.

2. For action "hide": if `currentStatus == StatusDeleted`, do NOT touch the review at all — set `reportStatus = ReportStatusResolved` and fall through to UpdateReport. The reported content is already gone, so the report is legitimately resolved; this preserves the author's deletion. For every other source status, call `validateAdminReviewTransition("hide", currentStatus)` (service_admin.go:161) and return its ErrInvalidTransition error unchanged so respondProcessReportError (http_errors.go:182) maps it to the same 4xx the direct PUT /admin/reviews path returns.

3. For action "delete": same shape — if `currentStatus == StatusDeleted`, skip SoftDeleteReview and just resolve the report (avoids pointless updated_at churn); otherwise validate with validateAdminReviewTransition("delete", currentStatus).

4. Defense in depth in server/internal/modules/course/review/repository_review_query.go:162 — change UpdateReviewStatus to `UPDATE reviews SET status = $2, updated_at = NOW() WHERE id = $1 AND status <> 'deleted'`. I verified this is safe for the other callers: service_admin.go:209 (hide) and :230 (restore) both go through validateAdminReviewTransition, whose whi …

#### 21. Resource list issues 2 nested queries per row while the outer cursor still holds a pooled connection (N+1 + pool starvation)

`server/internal/modules/resource/repository.go:228`

| | |
|---|---|
| 区域 | 资源 |
| 类别 | efficiency |
| 验证票数 | 2/2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
func (r *Repository) scanItems(ctx context.Context, rows pgx.Rows) ([]Item, int, error) {
	items := make([]Item, 0)
	total := 0
	for rows.Next() {
		item, err := scanItem(rows, &total)
		...
		item.Tags, err = r.loadTags(ctx, item.ID)         // r.db.Query -> acquires another pool conn
		...
		item.Bindings, err = r.loadBindings(ctx, item.ID) // r.db.Query -> acquires another pool conn
```

**失败场景**

GET /api/v1/resources?pageSize=100 (public, optionalAuth, handler.go:30) -> ListResources (repository.go:62) holds one pooled connection for the outer cursor for the entire scan and issues 2 more queries per row = 200 extra round trips. Confirmed in pgx v5 pgxpool/pool.go: Pool.Query does p.Acquire(ctx) and only releases the connection on Rows.Close(), so every request needs 2 pool connections at once. With DB_MAX_CONNS default 20 (config.go:370), 20 concurrent /resources requests each pin a connection on their outer cursor, then every loadTags() blocks in Acquire until the 5s DB_QUERY_TIMEOUT expires -> all requests return 500 'failed to list resources' and the entire DB pool is deadlocked for 5 seconds. GetResourceByID, UpdateResource and CreateResource all funnel through the same scanItems.

**修复方案**

Rewrite `scanItems` in /home/wztxy/Code/StuHelper/server/internal/modules/resource/repository.go (lines 220-299) as a drain-then-batch, mirroring the pattern already used in /home/wztxy/Code/StuHelper/server/internal/modules/openplatform/repository_apps.go (ListApps + loadScopeRequests/loadRedirectURIRequests).

1. Drain the outer cursor first, collecting ids and an index map, and check `rows.Err()` BEFORE issuing any nested query (so we never attach children to a partially-read result set):

```go
func (r *Repository) scanItems(ctx context.Context, rows pgx.Rows) ([]Item, int, error) {
	items := make([]Item, 0)
	ids := make([]int64, 0)
	index := make(map[int64]int)
	total := 0
	for rows.Next() {
		item, err := scanItem(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		index[item.ID] = len(items)
		ids = append(ids, item.ID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Cursor is exhausted here, so pgxpool has already returned the outer
	// connection (pgxpool/rows.go: Next() closes on !n). The batch loads below
	// therefore reuse a single pool connection instead of needing a second one.
	if err := r.attachTags(ctx, ids, items, index); err != nil {
		return nil, 0, err
	}
	if err := r.attachBindings(ctx, ids, items, index); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
```

2. Replace `loadTags` (line 260) and `loadBindings` (line 278) with batched versions keyed on `resource_id` (they have no other callers, verified by grep, so delete the old ones):

```go
func (r *Repository) attachTags(ctx c …


### P2（41 项）

#### 22. Student verification method labels miss school_email_otp and school_sso, so the review queue renders the raw i18n key

`clients/admin/apps/web-ele/src/locales/langs/zh-CN/admin.json:432`

| | |
|---|---|
| 区域 | Admin 前端 |
| 类别 | i18n-coverage |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
zh-CN/admin.json:432-435 → `"method": { "ldap": "LDAP", "manual": "人工审核" }` (en-US/admin.json:432-435 is identical in coverage).
views/users/student-verification/index.vue:200-206 → `const verificationMethodLabel = (method) => method ? $t(`admin.users.studentVerification.method.${method}`) : $t('admin.common.notSet');` — no fallback for unknown values (contrast content/reports/index.vue:101-108, which does `return label === key ? reason : label`).
clients/shared/src/types/api.gen.ts:5536 → `verificationMethod?: "ldap" | "manual" | "school_email_otp" | "school_sso" | null;`
server/internal/modules/user/service_student_email_otp.go:225-231 → `method := VerifyMethodSchoolEmailOTP; profile := &Profile{ ..., VerificationStatus: StatusPending, VerificationMethod: &method, ... }`
server/internal/modules/admission/repository_verified_profile.go:160-169 → `profileVerificationMethodForCredential` returns `"school_email_otp"` / `"school_sso"`.
```

**失败场景**

A student at a school whose `approvalPolicy` is `manual` verifies via school-email OTP (`POST /api/v1/user/profile/school-email/verify-otp`). service_student_email_otp.go writes `verification_status='pending', verification_method='school_email_otp'` (the auto-approve branch at line 239 is skipped for manual schools). The row appears in the admin 学生认证 pending queue, and the 认证方式 column renders the literal string `admin.users.studentVerification.method.school_email_otp` instead of a label (vue-i18n returns the key path on miss). Same for `school_sso` rows produced by the admission SSO credential path. Both locales are affected.

**修复方案**

Three edits.

1. clients/admin/apps/web-ele/src/locales/langs/zh-CN/admin.json:432-435 — extend the `users.studentVerification.method` map to cover the full enum, reusing the wording already used elsewhere in the same file (schoolConfig.schoolEmailOtp = "学校邮箱 OTP", schoolConfig.schoolSso = "学校 SSO") so the two screens stay consistent:
  "method": { "ldap": "LDAP", "manual": "人工审核", "school_email_otp": "学校邮箱 OTP", "school_sso": "学校 SSO" }

2. clients/admin/apps/web-ele/src/locales/langs/en-US/admin.json:432-435 — same, matching its own schoolConfig entries ("School Email OTP" / "School SSO"):
  "method": { "ldap": "LDAP", "manual": "Manual Review", "school_email_otp": "School Email OTP", "school_sso": "School SSO" }

3. clients/admin/apps/web-ele/src/views/users/student-verification/index.vue:200-206 — make the lookup degrade to the raw enum value instead of leaking the key path, mirroring `reasonLabel` in clients/admin/apps/web-ele/src/views/content/reports/index.vue:101-108:

const verificationMethodLabel = (
  method: StudentVerification['verificationMethod'],
) => {
  if (typeof method !== 'string' || method.trim() === '') {
    return $t('admin.common.notSet');
  }
  const key = `admin.users.studentVerification.method.${method}`;
  const label = $t(key);
  return label === key ? method : label;
};

This keeps the existing `notSet` behavior for null/undefined (and now also for an empty string), and any future enum addition on the server shows `school_xyz` rather than `admin.users.studentVerification.method.school_xyz`.

Optional, not required: `statusLabel` at index.vue: …

#### 23. No CI path filter covers scripts/ or tools/, so the custom Semgrep security rules can be changed with zero CI

`.github/workflows/ci.yml:63`

| | |
|---|---|
| 区域 | CI/CD |
| 类别 | ci-coverage-gap |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
docs:
              - 'AGENTS.md'
              - 'README.md'
              - 'docs/**'
              - 'scripts/check-docs-hygiene*'
...
  sast:
    if: >-
      github.event_name == 'workflow_dispatch' ||
      needs.changes.outputs.clients == 'true' ||
      needs.changes.outputs.backend == 'true' ||
      needs.changes.outputs.workflows == 'true'
```

**失败场景**

The seven filters (ci.yml:42-76) match only `server/**`, `clients/**`, `bots/koishi/**`, `docs/**`, `infra/**`, `.github/**`, `Makefile`, `docker-compose*.yml`, `.env.example`, `.env.prod.example`, `.gitleaksignore`, `AGENTS.md`, `README.md` and `scripts/check-docs-hygiene*`. Nothing matches `tools/**`, `scripts/lib/**`, `scripts/check-semgrep-custom-rules.sh`, `scripts/check-vue-ui-contracts.mjs`, `.node-version` or `.nvmrc`. Concretely: a PR that deletes the `stuhelper.go.raw-phone-log-field` rule from `tools/semgrep/stuhelper-security.yml` (the only file `scripts/check-semgrep-custom-rules.sh:5` scans `server/internal` with) sets every filter output to `false`, so `repository-policy`, `backend`, `contract`, `clients`, `client-e2e`, `koishi`, `infra`, `runtime-image-security` and `sast` all skip. `required` (ci.yml:652-659) only fails on `*failure*`/`*cancelled*`, so `CI / Required` — the sole non-CodeQL required check per docs/guides/github-migration.md:128 — reports success and the weakened security rule is never executed. Same for `scripts/lib/docs-hygiene-lib.mjs`, which holds all of `validateDocsTree()` while only `scripts/check-docs-hygiene.mjs` matches the `docs` filter; and for `.node-version`, where a Node major bump gets zero build/test coverage even though every job uses `node-version-file: .node-version`.

**修复方案**

Two changes, both mechanical.

A) `.github/workflows/ci.yml` — make guard code and the toolchain pin trigger the jobs that execute them.
1. Add a `guards` output at line 30 area: `guards: ${{ steps.filter.outputs.guards }}` (alongside `backend`..`workflows`, lines 24-30).
2. Add a `guards` filter in the `filters:` block (lines 42-76):
   ```
   guards:
     - 'scripts/**'
     - 'tools/**'
     - '.node-version'
     - '.nvmrc'
   ```
3. Extend the `docs` filter (line 59-63) with `- 'scripts/lib/**'` so `repository-policy` fires when `validateDocsTree()` itself changes (keep the existing `scripts/check-docs-hygiene*` line).
4. Add `needs.changes.outputs.guards == 'true'` to the `if:` of `repository-policy` (ci.yml:81-85) and `sast` (ci.yml:598-603). `sast` is the only executor of `scripts/check-semgrep-custom-rules.sh`, so this is the line that actually closes the `tools/semgrep/**` hole.
5. Add `.node-version` and `.nvmrc` to the `clients` (45-48), `contract` (49-58) and `koishi` (71-73) filters — every one of those jobs resolves Node via `node-version-file: .node-version`, so a major bump must build and test.

B) `infra/ops/tests/ci-and-drift-contract.sh` — assert the coverage so a future guard cannot be added outside a filter. Next to the existing `assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- '\.github/\*\*'$"`, add:
   ```
   assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- 'scripts/\*\*'$"
   assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- 'tools/\*\*'$"
   assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- '\.node-version'$"
   assert_contains "${GITHUB …

#### 24. The infra filter omits the Dockerfiles and Koishi sources that infra contracts assert, so supply-chain pinning gates skip the very change they guard

`.github/workflows/ci.yml:64`

| | |
|---|---|
| 区域 | CI/CD |
| 类别 | ci-coverage-gap |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
infra:
              - '.env.example'
              - '.env.prod.example'
              - '.github/**'
              - 'Makefile'
              - 'docker-compose*.yml'
              - 'infra/**'
```

**失败场景**

`infra/ops/tests/run-infra-contracts.sh` (only run by the `infra` job) executes 74 contracts that assert on files outside `infra/**`: `dockerfile-supply-chain-contract.sh:23-44` requires digest-pinned `ARG *_IMAGE=` and rejects mutable base tags in `server/Dockerfile`, `server/Dockerfile.dev`, `clients/web/Dockerfile` and `clients/admin/scripts/deploy/Dockerfile`; `ci-and-drift-contract.sh:111-142` asserts on those Dockerfiles plus `clients/.dockerignore`, `clients/package.json`, `clients/pnpm-workspace.yaml` and `server/Makefile`; `koishi-stuhelper-package-contract.sh` validates `bots/koishi` packaging. A PR that changes `clients/web/Dockerfile` from `ARG NGINX_IMAGE=nginx:1.30.4-alpine@sha256:...` to `nginx:latest` matches only the `clients` filter, so the `clients` job runs lint/type-check/test/build (which never touch Docker) while `infra` is skipped — the mutable-base-image gate does not run and `CI / Required` is green. The violation then surfaces later, on an unrelated `infra/**` PR by a different author. Same for `bots/koishi/**` changes versus the Koishi packaging contract.

**修复方案**

Apply the finding's second option, not the filter-extension option.

1. In .github/workflows/ci.yml, add a new job with NO `if:` gate on `needs.changes` (both scripts use only grep/sed/find — no apt, pnpm, Playwright, or Koishi setup — so this costs ~20s):

  static-contracts:
    name: Static file contracts
    runs-on: ubuntu-latest
    steps:
      - name: Check out the repository
        uses: actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803 # v6
        with:
          persist-credentials: false
      - name: Assert supply-chain pinning and CI wiring
        run: |
          bash infra/ops/tests/dockerfile-supply-chain-contract.sh
          bash infra/ops/tests/ci-and-drift-contract.sh

Leave run-infra-contracts.sh untouched; these two re-running inside the infra job costs milliseconds and keeps `make check-infra-contracts` complete for local use.

2. Add `- static-contracts` to the `required` job's `needs:` list (ci.yml:637-649) so a failure turns `CI / Required` red. Without this the new job is advisory only, since `required` is what branch protection observes.

3. Extend the `infra` filter at ci.yml:64 with `'scripts/**'` and `'bots/koishi/**'`. `scripts/**` is the bigger hole: scripts/check-secrets.sh, check-semgrep-custom-rules.sh and check-uniappx-shadow-files.sh currently match no filter whatsoever, so a PR touching only one of them runs zero gated jobs. `bots/koishi/**` is needed because koishi-stuhelper-package-contract.sh exercises infra/ops/package-koishi-stuhelper-packages.sh against real `corepack yarn build` output and cannot move into the cheap …

#### 25. Repeat detection loads a guild's entire message ledger into memory on every message, and the ledger is never pruned

`bots/koishi/packages/moderation-core/src/store.ts:84`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | efficiency |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
store.ts:83-86
```ts
  async listRecentMessages(guildId: string, limit: number) {
    const records = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { guildId })
    return records.sort(sortByCreatedDesc).slice(0, limit)
  }
```
Called once per message from message-guard.ts:232 (`const records = await this.deps.store.listRecentMessages(input.guildId, moderation.repeatWindowSize)`), right after message-guard.ts:63 inserts a new row via `saveMessage`. The model has no index on `guildId` and no ordering/limit pushdown (models.ts:88-101, `{ primary: 'messageId' }`), and the ledger content columns are `'text'`. A repo-wide grep for `database.remove` finds deletions only for keyword rules and reports — there is no retention job for `moderation_message_ledger`.
```

**失败场景**

An operator enables moderation in the Koishi WebUI (`moderationEnabled`). In a busy 2000-member QQ group at ~5k msgs/day, after two months the ledger holds ~300k rows for that guild. Every subsequent incoming message then executes `SELECT * FROM moderation_message_ledger WHERE guildId = ?` with no index, materializes ~300k JS objects (each carrying full `content` + `normalizedContent`), sorts them, and throws away all but `repeatWindowSize` (default ~10). Message handling latency grows linearly with history until the Koishi process stalls and eventually OOMs, taking down admission reminders and the action stream with it.

**修复方案**

Three changes, all in the koishi workspace.

1) Push ordering + limit into the query — bots/koishi/packages/moderation-core/src/store.ts:83-86. Use the cursor overload (not `select()`) so the store's dependency surface stays `database.get`, keeping the existing test double in store.test.ts:34-46 valid:

  async listRecentMessages(guildId: string, limit: number) {
    return this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { guildId }, {
      sort: { createdAt: 'desc' },
      limit,
    }) as Promise<MessageLedgerRecord[]>
  }

Keep `limit` exactly as passed (do NOT use limit + 1): message-guard.ts:233-234 filters the just-saved current message out of the window, so today's effective window is limit-1, and changing that would silently shift repeat-detection thresholds. Drop the now-unused JS sort for this method only (`sortByCreatedDesc` is still used by listRecentEvents).

2) Add the supporting index — bots/koishi/packages/moderation-core/src/models.ts:88-101, third argument of `ctx.model.extend`:
  { primary: 'messageId', indexes: [{ keys: { guildId: 'asc', createdAt: 'desc' } }] }
@minatojs/driver-sqlite creates this on migration (createIndex, lib/index.cjs:558); the live DB currently has only the primary-key autoindex.

3) Add config-driven retention. Add `ledgerRetentionDays` to GroupGuardModerationSettings in bots/koishi/packages/shared/src/guard/behavior-settings.ts (parse via the existing positiveIntegerOrDefault path at :217, add it to GROUP_GUARD_MODERATION_SETTING_KEYS at :260, default 7 to match the existing antiRecall retentionDays default, and surface i …

#### 26. Dashboard overview loads the whole moderation-event and guard-member tables to compute counters

`bots/koishi/packages/moderation-core/src/store.ts:61`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | efficiency |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
store.ts:60-63
```ts
  async listRecentEvents(limit = 20) {
    const records = await this.ctx.database.get(MODERATION_EVENT_TABLE, {})
    return records.sort(sortByCreatedDesc).slice(0, limit)
  }
```
store.ts:356-363
```ts
    const [events, reviews, reports, warnings, guards] = await Promise.all([
      this.listRecentEvents(20),
      ...
      this.ctx.database.get(GUARD_MEMBER_TABLE, {}),
    ])
```
`appendEvent` (store.ts:53-58) inserts a row for every join, kick, warn, keyword hit, repeat hit, recall and report, and nothing ever deletes from `MODERATION_EVENT_TABLE`. `buildAdmissionRuntimePageData` similarly calls `deps.guardStore.listActive()` and `deps.moderationStore.listAllKeywordRules()` unbounded (admission-console-api.ts:107-114).
```

**失败场景**

After a few months of normal operation the moderation event table holds hundreds of thousands of rows (every `guild-member-added` writes one). Opening the StuHelper console dashboard triggers `getOverview`, which SELECTs the entire event table plus the entire guard-member table (including long-since released/kicked rows) into JS objects and sorts them, just to show a 20-row 'recent events' list and four integers. The console request takes seconds to tens of seconds and spikes RSS by hundreds of MB, and repeated refreshes can OOM the bot process.

**修复方案**

Five concrete changes; items 1-2 are the finding's real substance, 3-4 replace its incorrect suggestions.

1. bots/koishi/packages/moderation-core/src/store.ts:60-63 — push the sort and limit into the driver:
   async listRecentEvents(limit = 20) {
     return this.ctx.database.get(MODERATION_EVENT_TABLE, {}, {
       sort: { createdAt: 'desc' },
       limit,
     }) as Promise<ModerationEventRecord[]>
   }
   Keep the `sortByCreatedDesc` helper — `listRecentMessages` (store.ts:83-86) still uses it; apply the same cursor treatment there (`{ guildId }` + `sort`/`limit`), since message-guard.ts:232 calls it on EVERY message and the ledger grows per message (message-guard.ts:63) with no retention. That hot path is worse than the dashboard one.

2. bots/koishi/packages/moderation-core/src/models.ts:52-67 — add an index so the new ORDER BY/LIMIT does not degenerate into a full scan + sort in SQLite:
   }, { primary: 'id', indexes: [{ keys: { createdAt: 'desc' } }] })
   Do the same for MODERATION_MESSAGE_LEDGER_TABLE on `{ guildId: 'asc', createdAt: 'desc' }`.

3. bots/koishi/plugins/stuhelper-core/src/core/api/page-api-runtime.ts:181-184 — filter at the DB instead of in JS, mirroring GuardMemberStore.listActive:
   async function listActiveGuardMembers(ctx: Context) {
     return ctx.database.get(GUARD_MEMBER_TABLE, { releasedAt: null, kickedAt: null }) as Promise<GuardMemberRecord[]>
   }
   This stops the 30s dashboard poll from materializing every historical released/kicked row. (Do NOT touch admission-console-api.ts:108 — `listActive()` is already correctly filtered and it …

#### 27. Batch-mute payload parser silently drops all but the first member id when ids are space-separated, and the error hint names a command that does not exist

`bots/koishi/plugins/stuhelper-admin/src/commands.ts:320`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
commands.ts:315-330
```ts
function parseBatchMutePayload(payload: string | undefined) {
  const source = (payload || '').trim()
  ...
  const [secondsText, memberIdsText] = source.split(/\s+/, 2)
```
JavaScript's `String.prototype.split(sep, limit)` truncates the result array — it does not append the remainder — so `'600 111 222 333'.split(/\s+/, 2)` is `['600','111']`. `parseMemberIds` (commands.ts:308-313) nonetheless splits on `/[\s,，]+/`, i.e. it is written to accept whitespace-separated ids. The hint the operator is shown on a parse failure points at a non-existent command: `guardBatchMuteInvalidPayload: '请提供禁言秒数和成员 ID 列表，例如：群审批量禁言 120 10001,10002'` (packages/shared/src/message-template.ts:36) while the registered command is `'群审禁言 <payload:text>'` (commands.ts:112).
```

**失败场景**

An operator runs `群审禁言 600 10001 10002 10003` (whitespace-separated, which `parseMemberIds` implies is supported). `memberIdsText` is `'10001'`, so only member 10001 is muted and only one `action_executed` moderation event is written; 10002 and 10003 are never muted and no warning is emitted. If the operator instead gets the format wrong, the bot tells them to type `群审批量禁言 …`, which resolves to no command at all.

**修复方案**

Three changes, all in tracked source (no generated files touched):

1. `bots/koishi/plugins/stuhelper-admin/src/commands.ts:320` — stop truncating the payload. Replace

    const [secondsText, memberIdsText] = source.split(/\s+/, 2)

with

    const [secondsText, ...rest] = source.split(/\s+/)
    const memberIdsText = rest.join(' ')

Behavior is preserved for every currently-working input (`'120 10011,10012'` still parses to two ids) and for the reject paths: payload `'600'` gives `rest === []` -> `memberIdsText === ''` -> `parseMemberIds` returns `[]` -> `null` -> `guardBatchMuteInvalidPayload`, and a non-numeric first token still fails the `Number.isInteger` check at 322. `parseMemberIds`'s existing `/[\s,，]+/` split then makes `600 10001 10002`, `600 10001, 10002`, and `600 10001，10002` all resolve fully, so its `\s` branch stops being dead code. No change needed at the call site (128-158).

2. `bots/koishi/packages/shared/src/message-template.ts:36` — point the hint at the real command and show both accepted separators:

    guardBatchMuteInvalidPayload: '请提供禁言秒数和成员 ID 列表，例如：群审禁言 120 10001,10002（也可用空格分隔）',

3. `bots/koishi/plugins/stuhelper-core/client/components/SettingsView.vue:1248` — update the duplicated WebUI default to the identical string, otherwise an operator who opens and saves the settings form re-persists the old wrong hint into `AdminRuntimeSettingsStore`.

Regression test: extend `bots/koishi/plugins/stuhelper-admin/src/index.test.ts` alongside the existing case at line 300 with `await client.shouldReply('群审禁言 120 10011 10012', '已批量禁言 2 名成员。')` and one c …

#### 28. Settings page save fans out to seven independent console endpoints with no rollback and no reload on failure, leaving the form contradicting the server

`bots/koishi/plugins/stuhelper-core/client/components/SettingsView.vue:1971`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | error-handling |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
await settingsApi.update(buildSettingsUpdatePayload())
    await bindingSettingsApi.update(buildBindingSettingsPayload())
    await adminSettingsApi.update(buildAdminSettingsPayload())
    await groupGuardAISettingsApi.update(buildGroupGuardAISettingsPayload())
    await groupGuardBehaviorSettingsApi.update(buildGroupGuardBehaviorSettingsPayload())
    if (groupGuardMessageResetPending.value) { await groupGuardMessageSettingsApi.reset() } else { await groupGuardMessageSettingsApi.update(...) }
    await saveKeywordRules()
    message.success('设置已保存')
    await loadSettings()
  } catch (cause) {
    setActionError('保存失败', cause, '保存设置失败')
```

**失败场景**

An operator edits the OpenAI key, the group-guard repeat threshold and deletes one keyword rule, then presses 保存. settingsApi.update and bindingSettingsApi.update succeed; groupGuardBehaviorSettingsApi.update rejects (e.g. repeatWindowSize out of range, or the websocket drops mid-flight). The catch shows a generic '保存失败' banner and, crucially, never calls loadSettings(), so originalSettings still holds the pre-save snapshot: the form keeps showing every edit as unsaved even though the plugin settings and binding messages are already persisted. Pressing 放弃更改 then restores the pre-save values into the form, so the UI now displays values that contradict the server, and the operator has no way to tell which of the seven writes landed. saveKeywordRules has the same shape internally — a mid-loop delete failure leaves some rules deleted and zero upserts applied.

**修复方案**

Do not auto-`loadSettings()` in the catch (that silently destroys the operator's unsaved form on a transient failure). Instead, in `SettingsView.vue`:
1. Attribute the failure: run each sub-save through a helper that tags it with its section name, e.g. `await step('AI 设置', () => groupGuardAISettingsApi.update(...))`, and have the catch report `保存失败（AI 设置）: <server message>` so the operator knows exactly where the sequence stopped and which sections already landed.
2. Commit the baseline incrementally so a retry is safe: after each sub-save resolves, update the corresponding slice of the `originalSettings` snapshot (critically the `keywordRules` slice inside `saveKeywordRules`, right after each successful `delete`/`upsert`). This removes the retry dead-end where an already-deleted rule is deleted again.
3. Belt-and-braces on the server: make `stuhelperGroupCenter/keyword-rules/delete` (`src/core/api/keyword-rule-api.ts:83-92`) idempotent — when `getKeywordRule(id)` returns nothing, return `success({success:true})` instead of throwing `keyword rule not found` (keep the scope assertion for existing rules).
4. Replicate the server-only `assertSafeKeywordRegex` check in `validateKeywordRuleDraft` so ReDoS-unsafe patterns are rejected at staging time rather than mid-save.
5. Add an explicit "重新加载服务器设置" action to the error banner so re-syncing is the operator's choice, and update the pinned assertions in `client/component-contract.test.ts:530,681` plus add a test that a mid-sequence rejection leaves the keyword-rule baseline advanced.

#### 29. Console listeners are never disposed, so privileged admission actions stay callable after the plugin is unloaded

`bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts:91`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | lifecycle |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
admission-console-api.ts:91-103 registers three listeners and discards the registration entirely:
```ts
  ctx.console.addListener(ADMISSION_RUNTIME_ACTION_EVENT, async function (input) {
    return handleAdmissionRuntimeAction(ctx, deps, input, this as ConsoleActionClient)
  }, { authority: CONSOLE_AUTHORITY })
```
The installed console implements `addListener` as a bare assignment with no disposer (node_modules/@koishijs/console/lib/index.js:242): `addListener(event, callback, options) { this.listeners[event] = { callback, ...options }; }`. Neither this file nor `stuhelper-core/src/core/api/page-api.ts:30-44`, `governance-actions.ts:62/69`, `review-actions.ts:28/36` register a `ctx.on('dispose', ...)` or `ctx.effect` to unregister. `console` is an *optional* injection (index.ts:46-49), and cordis only restarts scopes for **required** services (`if (!runtime.inject[name]?.required) continue` — @cordisjs/core/lib/index.cjs:459/470), so the stuhelper plugins are never reloaded when the console is.
```

**失败场景**

An admin disables `stuhelper-group-guard` from the Koishi WebUI plugin list. The scope is disposed (streams closed, `ctx.on` handlers removed), but `stuhelperGroupGuard/action/admission-member` remains in `console.listeners` bound to the disposed plugin's closure. Any authority-4 console client can still invoke `skip` / `release-blacklist` / `regenerate`, executing real backend admission mutations and `muteGuildMember` calls through the dead plugin's `platform` client and `guardStore`, while the WebUI reports the plugin as off. The closure also pins the whole plugin graph in memory. Symmetrically, reloading the console plugin creates a fresh `listeners` object and permanently kills the StuHelper console pages until the stuhelper plugins are manually reloaded.

**修复方案**

Two-part fix; part 1 is the high-value one.

PART 1 — make console a required injection for the registration sub-scope (fixes the reload/never-registered half, matches the pattern stuhelper-core already uses).
In `bots/koishi/plugins/stuhelper-group-guard/src/index.ts:150`, replace the bare call with:
```ts
ctx.inject(['console'], (consoleCtx) => {
  registerAdmissionConsoleAPI(consoleCtx, { config, platform, runtimeSettings, behaviorSettings, messageProvider, guardStore, policyStore, moderationStore, admissionSubjectCoordinator, onRuntimeSettingsChanged: () => actionStreams.refresh() })
})
```
This mirrors `plugins/stuhelper-core/src/setup/register-console-api.ts:27` and `register-console-entry.ts:11`, and makes the `if (!ctx.console) return` bail at `admission-console-api.ts:87-89` dead code — delete it (or keep as a type narrow only). Effect: the three `stuhelperGroupGuard/*` listeners are re-registered whenever the console service reappears, so reloading `@koishijs/plugin-console` from the WebUI config editor no longer leaves the admission page permanently answering `{"error":"not implemented"}`.

PART 2 — add a real disposer so the listeners die with the scope (fixes the stale-privileged-action half).
Add a shared helper, e.g. `bots/koishi/packages/shared/src/console/disposable-listener.ts`:
```ts
export function addDisposableConsoleListener<K extends keyof ConsoleEvents>(
  ctx: Context, event: K, callback: ConsoleEvents[K], options?: DataService.Options,
) {
  ctx.effect(() => {
    const console = ctx.console!
    console.addListener(event, callback, options)
    re …

#### 30. Admission restricted-member queue is silently capped at 100 rows with no pagination, filter, or truncation notice

`bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts:115`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
const ACTIVE_MEMBER_LIMIT = 100
...
  const sortedMembers = [...activeMembers]
    .sort((left, right) => left.deadlineAt.getTime() - right.deadlineAt.getTime())
    .slice(0, ACTIVE_MEMBER_LIMIT)
...
    stats: { activeMemberCount: activeMembers.length, ... },
    activeMembers: sortedMembers.map(serializeGuardMember),

// AdmissionView.vue:227-231
//   <WorkspaceSection title="受限成员队列" :meta="`${model.activeMembers.length} 条`" ...>
```

**失败场景**

A freshman intake wave leaves 350 guard records unreleased. The 受限成员 metric card reads 350 (it uses data.stats.activeMemberCount), but the 受限成员队列 section header reads '100 条' and the table renders exactly 100 rows. AdmissionView has no search box, guild filter, status filter or pager, so the remaining 250 members are unreachable from the console: an operator cannot 重发 a link or 跳过 anyone past the first 100 by deadline, and there is no on-screen text explaining the gap — the two numbers just silently disagree. The only remedy is waiting for the first 100 to age out of listActive().

**修复方案**

Minimal, safe fix in two lines of code plus one template block:

1. Return the true total and the applied cap from the page API so the UI can be honest. In `/home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts` (buildAdmissionRuntimePageData, around line 115-192) add to the payload:
   `activeMembersTruncated: activeMembers.length > ACTIVE_MEMBER_LIMIT,`
   `activeMemberLimit: ACTIVE_MEMBER_LIMIT,`
   (keep `stats.activeMemberCount` as the untruncated count).

2. Mirror the new fields in `AdmissionRuntimePageData` in `/home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-core/client/models/admission-runtime.ts` (interface at lines 1-53).

3. In `/home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-core/client/components/AdmissionView.vue` change the 受限成员队列 section meta (line 230) from `${model.activeMembers.length} 条` to something that cannot contradict the metric card, e.g. `${model.activeMembers.length} / ${data.stats.activeMemberCount} 条`, and render a hint inside the section when `data.activeMembersTruncated` is true: "仅显示最早到期的 {{ data.activeMemberLimit }} 条，共 {{ data.stats.activeMemberCount }} 条；其余请在 复核工作台 检索或使用群内认证命令处理。"

Optional follow-up (not required to close this): accept `{ offset, limit, guildId }` on `stuhelperGroupGuard/page/admission-runtime` and add a pager plus guild filter, matching the 类型/检索 controls ReviewView already has. Also worth a separate ticket: `GuardMemberStore.listActive()` filters only on `releasedAt/kickedAt` null and nothing ever terminalizes a record whose kick keeps failing (member-guard …

#### 31. One unforwardable freshman application permanently blocks all later material forwards

`bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts:471`

| | |
|---|---|
| 区域 | Koishi 机器人 |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
member-guard.ts:469-476
```ts
    const items = await this.deps.platform.listPendingFreshmanForwards()
    const messages = await this.getMessages()
    for (const item of items) {
      const bot = resolveFreshmanForwardBot(forwardBots, item)
      await forwardFreshmanMaterial(bot, item, messages)
      await this.deps.platform.markFreshmanForwarded(item.application.id)
    }
```
No try/catch inside the loop, and `scanPendingMembers` (member-guard.ts:215) does not guard the call either — the only catch is the scheduler's `.catch(...)` in events.ts:59-61. Both `resolveFreshmanForwardBot` (freshman-forward.ts:18/22) and `forwardFreshmanMaterial` (freshman-forward.ts:31/34/50) throw. The backend returns the queue oldest-first and only clears items on ACK: `repository_bot_scan.go:96-100` → `WHERE app.status = 'pending' AND app.forwarded_at IS NULL ... ORDER BY app.created_at ASC`.
```

**失败场景**

`forward_raw_material_to_qq` and `freshmanForward.enabled` are turned on. The oldest pending application belongs to a policy whose `management_guild_ids` includes a group the bot was since removed from, so `bot.sendMessage(guildID, ...)` throws and `forwardFreshmanMaterial` raises an `AggregateError` before `markFreshmanForwarded` runs. `forwarded_at` stays NULL, so this item is returned first on every subsequent scan and throws again — every newer freshman application behind it is never forwarded to the review group, and the only symptom is one 'group guard scheduled scan failed' log per scan interval. Freshman review silently stops for all applicants.

**修复方案**

Isolate each queue item WITHOUT swallowing the batch error, in `forwardFreshmanMaterials` at /home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts:471-475:

```ts
const failures: unknown[] = []
for (const item of items) {
  try {
    const bot = resolveFreshmanForwardBot(forwardBots, item)
    await forwardFreshmanMaterial(bot, item, messages)
    await this.deps.platform.markFreshmanForwarded(item.application.id)
  } catch (error) {
    failures.push(error)
    this.deps.logger.warn('group guard freshman forward failed', {
      applicationID: item.application.id,
      error: formatAdmissionActionError(error), // already imported at line 21
    })
  }
}
if (failures.length === 1) throw failures[0]
if (failures.length) throw new AggregateError(failures, 'freshman forward batch failed')
```

Rethrowing the single failure unchanged keeps freshman-forward.test.ts:73-88 and :109-118 green (Node matches a RegExp validator against `String(error)`, so the original message must survive) while every other application in the batch is still attempted. Add a regression test with an old poison item plus a newer healthy item asserting the healthy one is sent and marked.

Note this only stops starvation *within* a batch; the poison item is still re-served first on every tick and still logs each interval. To actually drain the queue, add server-side failure tracking: a `forward_attempt_count`/`last_forward_error`/`last_forward_attempt_at` set of columns on `freshman_verification_applications`, a bot endpoint to report a forward failure (mirroring `POST / …

#### 32. Batch reviews `courseIDs` array serialization is incompatible on all three legs (spec explode:false vs. client explode:true vs. handler reading only first value)

`server/api/paths/review-crud.yaml:195`

| | |
|---|---|
| 区域 | OpenAPI 契约 |
| 类别 | contract-mismatch |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
Spec (server/api/paths/review-crud.yaml:185-196):
      - name: courseIDs
        in: query
        required: true
        schema:
          type: array
        style: form
        explode: false

Backend (server/internal/modules/course/review/review_read.go:178-184):
	idsStr := c.Query("courseIDs")   // gin returns only the FIRST repeated value
	parts := strings.Split(idsStr, ",")

Shared client (clients/shared/src/api/client.ts:28-32) creates openapi-fetch with no `querySerializer`, so the library default (`style: form, explode: true`) applies. Verified empirically with the installed openapi-fetch:
  client.GET('/api/v1/course/review/reviews/batch', { params: { query: { courseIDs: [1,2,3], pageSize: 5 } } })
  -> 'http://x/api/v1/course/review/reviews/batch?courseIDs=1&courseIDs=2&courseIDs=3&pageSize=5'
The uniappx transport has the same behaviour (clients/shared/src/api/session-client.ts:144-149 pushes one pair per array item).
```

**失败场景**

`api.review.getBatchCourseReviews([101, 202, 303])` (clients/shared/src/api/reviews.ts:42) sends `?courseIDs=101&courseIDs=202&courseIDs=303`. `c.Query("courseIDs")` yields "101", so the handler fetches reviews for course 101 only and returns `{"101": {...}}`; courses 202 and 303 are silently missing with HTTP 200 — the exact N+1 avoidance the endpoint exists for is defeated with no error. The one wrapper that consumes it, clients/web/src/api/review.ts:42, then feeds the course-keyed map into `readReviewPagePayload`, which requires top-level `list`/`total` (clients/web/src/modules/review/reviewListPayload.ts:187-195) and therefore throws `Invalid review list response` for every successful call. No view calls it yet, so the break is latent but 100% reproducible the moment it is wired up.

**修复方案**

Apply the spec+handler side only; do NOT touch the shared client's serializer.

1. `server/api/paths/review-crud.yaml:195` — change `explode: false` to `explode: true` so the published contract matches what every generated client actually emits, mirroring the already-working `scope` param at `server/api/paths/open-platform.yaml:641`. Then regenerate (`server/api/openapi.bundled.yaml`, `server/internal/api/gen/`, `clients/shared/src/types/api.gen.ts`) via the normal codegen task — do not hand-edit generated files.

2. `server/internal/modules/course/review/review_read.go:178-184` — replace the single-value read with an array read that still accepts the legacy comma form, so both the new client format and any existing spec-conformant third-party caller keep working:

    raw := c.QueryArray("courseIDs")
    if len(raw) == 0 {
        response.BadRequest(c, "courseIDs is required")
        return
    }
    parts := make([]string, 0, len(raw))
    for _, v := range raw {
        parts = append(parts, strings.Split(v, ",")...)
    }
    if len(parts) > 20 { ... }

   Keep the existing per-item TrimSpace / ParseInt / id<=0 validation and the `len(courseIDs)==0` guard unchanged.

3. `clients/web/src/api/review.ts:41-44` — `getBatchCourseReviewsPage` must stop calling `readReviewPagePayload` on a course-keyed map. Either change its contract to return `Record<string, PaginatedResult<Review>>` and map each entry through `readReviewPagePayload`, or drop the adapter entirely until a view needs it. As written it throws `Invalid review list response` on every successful response.

4. Add …

#### 33. Pages declared with navigationStyle "custom" have no status-bar/safe-area padding, so their headers render under the status bar and WeChat capsule

`clients/uniappx/src/pages.json:13`

| | |
|---|---|
| 区域 | UniAppX |
| 类别 | layout |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
pages.json sets `"navigationStyle": "custom"` for pages/home/index (line 13), pages/auth/login (line 80) and pages/auth/callback (line 87), which removes the native navigation bar and makes the page start at y=0 on App and mini-program targets. Nothing compensates: `grep -rn "safe-area\|status-bar-height\|statusBarHeight\|getWindowInfo" src/` returns zero layout hits (only i18n's getSystemInfoSync for locale). The first painted element on home is `.hero-card { margin: 24rpx; padding: 36rpx; }` (pages/home/index.vue:210) — 12px from the top edge — and login.vue's `.login-page { padding: 48rpx }` with `.login-card { margin-top: 80rpx }` is measured from y=0 as well.
```

**失败场景**

Run the App or mp-weixin build on any device with a >=20px status bar (or a notch). On the home tab the purple hero card starts 12px below the physical top, so 'StuHelper' and the greeting sit behind the clock/battery/signal indicators, and on WeChat the 胶囊按钮 floats over the top-right of the card. There is no way to scroll the header out from under the status bar because the card is the first element of the scroll-view.

**修复方案**

None of the three pages actually implements a custom navigation bar, so the simplest correct fix is to delete `"navigationStyle": "custom"` from all three entries in clients/uniappx/src/pages.json (pages/home/index line 13, pages/auth/login line 80, pages/auth/callback line 87). Each already has a navigationBarTitleText and globalStyle supplies the bar colours, so the native bar renders correctly and handles the status bar, notch and capsule on every platform; it also restores the back button on pages/auth/callback, which is entered via uni.navigateTo. If the home hero is meant to bleed to the top edge, keep `custom` on pages/home/index only and reserve the system chrome in src/pages/home/index.vue by prepending a spacer inside the scroll-view (`<view class="status-bar-spacer" />` with `height: var(--status-bar-height, 20px)`, which uni-app injects on App and mp-weixin) plus `padding-top: env(safe-area-inset-top)` on `.home-page` for iOS H5, and enough extra top offset (~44px on mp-weixin) that .hero-card content clears the capsule button.

#### 34. ParticleBackground leaks 50 infinite gsap tweens on every resize frame and only kills the last batch on unmount

`clients/web/src/components/animated/ParticleBackground.vue:155`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | memory-leak |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
const handleResize = () => {
  if (resizeFrame !== null) return
  resizeFrame = requestAnimationFrame(() => {
    resizeFrame = null
    resizeCanvas()
    createParticles()          // <- discards `particles`, never kills their tweens
  })
}
// createParticles: particles = []; ... gsap.to(particle, { ..., repeat: -1, yoyo: true })
onUnmounted(() => { ... gsap.killTweensOf(particles) })   // only the current array
```

**失败场景**

ParticleBackground is live on the landing page (HomePage -> HeroSection.vue:25). `createParticles()` reassigns `particles` to a fresh array of 50 objects each with an infinitely repeating gsap tween, and the previous 50 tweens are never killed — gsap keeps their targets alive and ticking. The rAF guard only coalesces to one rebuild per frame, so dragging a desktop window edge for ~3 s (~180 resize frames) leaves ~9000 live infinite tweens; mobile URL-bar collapse / orientation change / soft-keyboard open also fire `resize`. `gsap.killTweensOf(particles)` at unmount kills only the newest batch, so after the user leaves the home page the orphaned tweens keep driving the gsap ticker at 60 fps for the entire session — growing memory and janking every later page.

**修复方案**

Two changes in clients/web/src/components/animated/ParticleBackground.vue:

1. Make `createParticles()` self-cleaning so no batch can be orphaned (this also makes the existing `onUnmounted` kill correct, since only one batch will ever be live):

const createParticles = () => {
  gsap.killTweensOf(particles)   // no-op on the initial empty array
  particles = []
  ...
}

2. Stop rebuilding on resize — clamp existing particles into the new viewport instead, which removes the churn entirely and preserves visual continuity:

const handleResize = () => {
  if (resizeFrame !== null) return
  resizeFrame = requestAnimationFrame(() => {
    resizeFrame = null
    resizeCanvas()
    if (!particles.length) { createParticles() ; return }
    for (const p of particles) {
      p.x = Math.min(p.x, window.innerWidth)
      p.y = Math.min(p.y, window.innerHeight)
    }
    if (prefersReducedMotion) drawFrame()   // canvas was cleared by resizeCanvas()
  })
}

Note that clamping `p.x` fights the running tween's cached start/end values, so if exact positioning after a large resize matters, prefer the alternative: keep the tween handles (`let tweens: gsap.core.Tween[] = []`, push each `gsap.to(...)` return value, `tweens.forEach(t => t.kill()); tweens = []` at the top of `createParticles`) and keep calling `createParticles()` from the rAF callback. Either way step 1 is the mandatory part; step 2 is the throughput improvement.

#### 35. Own-review delete fires the DELETE immediately on a single icon click, with no confirmation and no undo

`clients/web/src/components/business/review/useReviewDelete.ts:23`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | destructive-action |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
ReviewCard.vue:196-206 binds the trash icon straight to the handler:
```html
<button type="button" v-if="props.isOwnReview" :aria-label="t('review.review.deleteBtn')" :disabled="deleting" @click="handleDeleteOwn">
```
and `useReviewDelete.handleDeleteOwn` has no guard at all:
```ts
async function handleDeleteOwn() {
  deleting.value = true
  try {
    const review = reviewGetter()
    await api.review.deleteReview(review.id)
```
The sibling control for the much cheaper reply delete *does* have a two-step confirm (ReplyCard.vue:24-52 `requestDelete` -> `confirmingDelete` panel -> `confirmDelete`), so this is an inconsistency inside the same component family, not a project-wide convention.
```

**失败场景**

On the My Reviews list (the only place `is-own-review` is set) the edit (pencil) and delete (trash) icons sit adjacent with identical 16px styling. One mis-click on the trash icon issues `DELETE /api/v1/course/review/reviews/{id}` at once; the row is removed via `emit('deleted')` and the user's multi-paragraph review is gone with no confirm prompt and no restore path in the UI.

**修复方案**

Add the two-step confirm in clients/web/src/components/business/review/ReviewCard.vue only, leaving useReviewDelete's public shape (deleting, handleDeleteOwn) unchanged so the existing mock in __tests__/ReviewCard.locked.test.ts:96-101 keeps working.

1. Local state in <script setup>: `const confirmingDelete = ref(false)`, `requestDelete = () => { confirmingDelete.value = true }`, `cancelDelete = () => { confirmingDelete.value = false }`, `confirmDelete = async () => { confirmingDelete.value = false; await handleDeleteOwn() }`.
2. Change the trash button at line 203 from `@click="handleDeleteOwn"` to `@click="requestDelete"`, and extend its condition to `v-if="props.isOwnReview && !confirmingDelete"` so the icon swaps for the panel.
3. Mirror ReplyCard.vue:23-43: render a `v-if="confirmingDelete"` inline panel with role="alertdialog", `:aria-label="t('review.review.deleteConfirm')"`, the prompt `{{ t('review.review.deleteConfirm') }}` — finally consuming the dead translated key at src/i18n/locales/en-US/review.ts:80 and zh-CN/review.ts:65 — plus a cancel button (t('common.actions.cancel'), @click="cancelDelete") and a danger-styled confirm button (t('common.actions.confirm'), :disabled="deleting", @click="confirmDelete").
4. Add data-testid hooks matching ReplyCard's style (`review-delete-${review.id}`, `review-delete-confirm-${review.id}`) and a component test asserting clicking the trash icon does not call api.review.deleteReview until confirm is clicked.

No other caller is affected: ReviewCard.vue is the sole consumer of useReviewDelete, and is-own-review is passed only …

#### 36. ErrorBoundary swallows every Vue component error in production — no console output and no report to the existing frontend-error endpoint

`clients/web/src/components/common/ErrorBoundary.vue:37`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | observability |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
onErrorCaptured((err: Error) => {
  if (import.meta.env.DEV) {
    console.error('[ErrorBoundary] Captured error:', err)
  }
  error.value = err
  // 返回 false 阻止错误继续向上传播
  return false
})
```

**失败场景**

ErrorBoundary wraps the whole app (App.vue:3-17). Returning `false` stops propagation, so `app.config.errorHandler` (main.ts:56, itself DEV-only logging) never runs, and because Vue handled the exception the `window.addEventListener("error", ...)` beacon in src/utils/observability.ts:60 never fires either. Concrete case: a course-detail component throws a TypeError on a malformed API payload in production — the user sees the generic "出错了" card, the browser console is empty, and nothing is POSTed to /api/v1/metrics/frontend-errors, so the crash is invisible to operators. The existing frontend-error telemetry channel receives zero Vue render/setup errors by construction.

**修复方案**

Report and log before suppressing, and make the new kind actually acceptable end-to-end:

1. `clients/web/src/utils/observability.ts`: export the reporter — `export function reportFrontendError(kind: FrontendErrorKind, ...)` — and widen `type FrontendErrorKind = "error" | "unhandledrejection" | "vue-error"`. Guard it so it no-ops when observability was not initialized (`shouldInitObservability` is false for the E2E stub / bare join host), otherwise E2E runs start emitting beacons.

2. `clients/web/src/components/common/ErrorBoundary.vue:37-44`: inside `onErrorCaptured`, call `console.error('[ErrorBoundary]', err)` unconditionally (drop the `import.meta.env.DEV` gate) and `reportFrontendError('vue-error', err?.message, instance-name-or-info)`; wrap both in try/catch so a failure in reporting cannot re-enter the boundary. Keep `error.value = err` and `return false` so the fallback UI still owns the render.

3. `clients/web/src/main.ts:56-62`: do the same in `app.config.errorHandler` (log unconditionally + `reportFrontendError('vue-error', ...)`), since it is the handler for anything outside the boundary and is itself a prod no-op today that also suppresses Vue's built-in `logError`.

4. Backend, required or step 2 is a silent drop: add `"vue-error": true` to `allowedFrontendErrorKinds` in `server/internal/pkg/metrics/frontend_errors.go:14`, add `vue-error` to the enum at `server/api/openapi.yaml:541`, regenerate (do not hand-edit `server/internal/api/gen/` or `clients/shared/src/types/api.gen.ts`), and extend `server/internal/pkg/metrics/metrics_test.go` with a case asserting …

#### 37. No skip-to-content link: every page forces ~11 header tab stops before reaching main content

`clients/web/src/components/layout/AppShell.vue:5`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | accessibility |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
<div class="min-h-screen relative z-0">
    <AppHeader />

    <main class="app-shell-main" :class="mainPaddingClass">
      <slot />
    </main>
```

**失败场景**

`grep -rni "skip.*content|skip-link|#main"` over clients/web/src returns nothing, and `<main>` here has no `id`/`tabindex` to target. On /courses a keyboard or switch-access user must Tab through logo, 4 nav links, the InlineSearch combobox, the write-review button, LocaleSwitcher, ThemeSwitcher, NotificationBell and the user-menu button — 11+ stops — on every single page load and after every route change before reaching the first result link. This is a WCAG 2.4.1 (Bypass Blocks, Level A) failure in the shared shell, i.e. it affects every authenticated page of the product.

**修复方案**

In clients/web/src/components/layout/AppShell.vue:

1. Add a first-in-DOM skip link above `<AppHeader />`, using i18n rather than a hardcoded string:
   `<a href="#app-main" class="sr-only focus-visible:not-sr-only focus-visible:fixed focus-visible:top-2 focus-visible:left-2 focus-visible:z-[calc(var(--z-sticky)+1)] focus-visible:rounded-lg focus-visible:bg-bg-card focus-visible:px-4 focus-visible:py-2 focus-visible:text-sm focus-visible:font-semibold focus-visible:text-text-primary focus-visible:shadow-md" @click="focusMain">{{ t('nav.skipToContent') }}</a>`
   (Tailwind v4 is in use — `sr-only` / `not-sr-only` are available; confirmed tailwindcss ^4.0.0 in clients/web/package.json.)

2. Add `id="app-main"` and `tabindex="-1"` to the existing `<main>` at line 5, plus `scroll-mt-[var(--navbar-height)]` (and `max-tablet:scroll-mt-[var(--mobile-header-height)]` on /courses, mirroring `mainPaddingClass`) so the jump target is not occluded by the `fixed` AppHeader. Also add `outline-none` so the programmatic `-1` focus does not paint a stray ring.

3. Add the key to BOTH locale namespaces: `skipToContent: '跳到主要内容'` in src/i18n/locales/zh-CN/nav.ts and `skipToContent: 'Skip to main content'` in src/i18n/locales/en-US/nav.ts.

4. Optional, and only if scoped to real navigations (not initial mount): in AppShell, `watch(() => route.fullPath, ...)` and on change call `document.getElementById('app-main')?.focus({ preventScroll: true })` guarded by a "first run" flag. Do not focus on initial load. This is secondary — item 1+2 alone removes the barrier.

#### 38. Toasts created by a component that then navigates away stay on screen forever

`clients/web/src/composables/useToast.ts:88`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
onScopeDispose(() => {
  for (const id of instanceTimerIDs) {
    const timer = timers.get(id)
    if (timer) {
      clearTimeout(timer)
      timers.delete(id)
    }
  }
})

// the toast itself is never removed from the module-level list:
const toasts = ref<ToastItem[]>([])   // line 16, global
function show(...) { toasts.value = [...toasts.value, { id, type, message, duration }] }
```

**失败场景**

`toasts` is module-global and rendered by a permanently mounted `<Toast />` in App.vue:16. When the scope that created a toast is disposed, the dismiss timer is cleared but the entry is never removed from `toasts`, so the toast is displayed forever. Concrete flow: PostReviewPage.vue:1057-1060 does `toast.success(t('review.post.success'))` and then `router.push({ name: 'course-reviews' ... })`; the page unmounts before the 3000 ms timer fires, the timer is cancelled by onScopeDispose, and the green "发布成功" toast is pinned under the header for the rest of the SPA session (only manual click on × removes it). Every further navigate-with-toast stacks another permanent toast. Verified by a temporary vitest case: after unmounting the creator and advancing fake timers 60 s, `toasts.value` still contained `["posted!"]` (expected `[]`).

**修复方案**

Do not "fix" this by calling `remove(id)` in `onScopeDispose` — that would delete the toast the instant the creating page unmounts, so the user would never see the "发布成功" message at all. The dispose hook is the bug, not the cure: every piece of state the timer callback touches (`toasts`, `timers`, `nextID`) is already module-level, so a pending timeout after unmount leaks nothing.

Apply in clients/web/src/composables/useToast.ts:
1. Delete the entire `onScopeDispose(...)` block (lines 87-96) and the now-unused `instanceTimerIDs` array (line 34) plus its `push` at line 46, and drop `onScopeDispose` from the vue import.
2. Hoist `show/remove/clearAll/success/error/info/warning` to module scope and have `useToast()` just return the stable singleton object, making toast lifetime provably independent of any component scope.
3. Add a regression test in clients/web/src/composables/__tests__/ (node-env safe, no `mount`): run `useToast().success('x')` inside an `effectScope()`, `scope.stop()`, `vi.advanceTimersByTime(3000)`, assert `toasts.value` is `[]`; also assert `timers.size === 0` indirectly by re-showing and checking ids do not collide.

#### 39. A single transient /auth/me failure during projection polling throws the just-verified user onto the generic "无法打开认证" error panel

`clients/web/src/modules/admission/views/AdmissionPage.vue:735`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | error-handling |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
waitForAdmissionProjection({
    refreshAuth: auth.fetchUser,
    signal: projectionRefreshAbort.signal,
  })
    .then((ready) => { if (ready) pageState.value = 'approved'; else projectionRefreshTimedOut.value = true })
    .catch((error) => { if (!isAbortError(error)) applyError(error) })
// projectionRefresh.ts: for (const delay of retryDelays) { await wait(delay); const user = await refreshAuth() }  // any rejection aborts the whole loop
```

**失败场景**

The user's old-student email OTP or freshman review just succeeded, so pageState is 'projectionPending' and the page polls /auth/me at 1s, 2s, 4s, 8s, 16s. If any single one of those five calls times out or returns 5xx (mobile network blip, backend rolling restart), the loop rejects, `applyError` maps the non-admission error to 'error', and the panel switches to "无法打开认证 / 认证链接暂时无法打开，请稍后重试。" — telling a user whose verification actually passed that their link is broken. The ProjectionPendingNotice retry button is gone because `retryProjectionRefresh` returns early unless pageState === 'projectionPending'. For a non-network ApiError, `auth.fetchUser` additionally runs `clearAuth()` and sets `user.value = null` (stores/auth.ts fetchUser catch), so the user also appears logged out until a window-focus event re-runs the load.

**修复方案**

In clients/web/src/modules/admission/projectionRefresh.ts, wrap the refreshAuth() call in try/catch inside the loop: re-throw AbortError, re-throw hard auth failures (ApiError with status 401/403) so the page can route to needsLogin, and swallow everything else (network/timeout/5xx) to continue to the next backoff delay; return false if all attempts fail. Then in AdmissionPage.vue scheduleProjectionRefresh (line 743), change the catch to set projectionRefreshTimedOut.value = true for non-abort, non-auth errors — keeping the projectionPending panel and its 重新检查状态 retry button — and only call applyError for the auth-failure case. Add a projectionRefresh.test.ts case asserting that a single mid-loop rejection does not abort the remaining attempts.

#### 40. Search results ignore ReviewCard's moderated/deleted/updated events, so admin actions leave stale cards

`clients/web/src/modules/review/views/SearchPage.vue:332`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | stale-ui |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
```html
<ReviewCard
  v-for="(review, idx) in resultReviews"
  :key="review.id"
  :review="review"
  class="stagger-item"
  :style="{ animationDelay: `${Math.min(idx, 8) * 60}ms` }"
/>
```
ReviewCard declares `moderated`, `deleted` and `updated` emits (ReviewCard.vue:343-347) and fires `moderated` after every hide/restore/admin-edit (useReviewModeration.ts). Both sibling lists subscribe — ReviewFeed.vue:45 `@moderated="() => loadReviews(true)"`, TeacherProfilePage.vue:185 `@moderated="() => fetchTeacherReviews(true)"` — SearchPage is the only consumer that drops them, and it has no other refresh trigger (`handleSearch` only runs on submit or route-query change).
```

**失败场景**

An admin searches reviews, clicks the hide (EyeOff) icon on a result and confirms with a reason. The PUT succeeds and a success toast appears, but the card still renders as `published`: the hide icon is still offered (so a second click re-issues `action:'hide'`), the content is still visible, and the restore/edit icons never appear until the admin re-runs the search by hand.

**修复方案**

In /home/wztxy/Code/StuHelper/clients/web/src/modules/review/views/SearchPage.vue:332, add the single listener matching the established sibling pattern:

```html
<ReviewCard
  v-for="(review, idx) in resultReviews"
  :key="review.id"
  :review="review"
  class="stagger-item"
  :style="{ animationDelay: `${Math.min(idx, 8) * 60}ms` }"
  @moderated="() => handleSearch({ restart: true })"
/>
```

Do NOT add @deleted/@updated: their trigger buttons require `props.isOwnReview`, which SearchPage never passes, so those handlers would be unreachable.

Caveat on the refetch: handleSearch({ restart: true }) clears resultCourses/resultReviews, resets pagination and calls scrollSearchPageToTop(), so an admin who used "load more" loses position. There is no single-review GET in clients/shared (only reviews.updateReview/deleteReview and admin.updateReview), so an in-place patch would require widening the emit to carry the new state (e.g. `moderated: [id: string, status: ReviewStatus]`) and patching resultReviews in place. If the jump-to-top is unacceptable, prefer that; otherwise the one-line refetch is the consistent, low-risk change.

#### 41. A failed "load more" on the teacher profile replaces the already-loaded reviews with an error box

`clients/web/src/modules/review/views/TeacherProfilePage.vue:157`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | error-state |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
The error branch precedes the list branch in the v-if chain, and it is keyed on the same ref that append failures set:
```html
<div v-if="teacherReviewsLoading && teacherReviews.length === 0"> ...skeleton... </div>
<div v-else-if="teacherReviewsError" role="alert"> ... <button @click="fetchTeacherReviews(true)">retry</button></div>
<div v-else-if="teacherReviews.length === 0"> ...empty... </div>
<div v-else class="flex flex-col gap-4"><ReviewCard v-for=... /></div>
```
and line 491-493: `catch { ... teacherReviewsError.value = t('teaching.profile.reviewsLoadFailed') }` — the same field for both reset and append loads, with `teacherReviews` left untouched. CourseDetailPage handles the same case correctly (it keeps `reviews` and only shows a retry row, CourseDetailPage.vue:405-437).
```

**失败场景**

On /teachers/42 the first 6 reviews render. The user clicks "Load more"; the page-2 request fails (offline blip / 500). `teacherReviewsError` is set, so the 6 rendered reviews disappear and the section collapses to a single error box. The retry button calls `fetchTeacherReviews(true)`, which clears the array and starts again at page 1, discarding scroll position.

**修复方案**

Split the append error from the initial-load error, matching the existing pattern in TeacherHubPage.vue / MyReviewsTab.vue.

1. Add `const teacherReviewsLoadMoreError = ref('')` next to the other refs (near line 268), and clear it in `showTeacherNotFound()` alongside `teacherReviewsError`.
2. In `fetchTeacherReviews` (line 472-493): clear both refs at the start; in the catch, branch on the load kind —
   `if (reset || teacherReviews.value.length === 0) { teacherReviewsError.value = t('teaching.profile.reviewsLoadFailed') } else { teacherReviewsLoadMoreError.value = t('teaching.profile.reviewsLoadFailed') }`
   Also clear `teacherReviewsLoadMoreError` on success.
3. In the template, leave the `v-else-if="teacherReviewsError"` block for the empty/initial case only, and inside the list `v-else` block render the append error next to the Load-more control, e.g. change line 187 to `v-if="teacherReviewsHasMore || teacherReviewsLoadMoreError"` and add above the button:
   `<p v-if="teacherReviewsLoadMoreError" role="alert" class="m-0 text-sm text-danger">{{ teacherReviewsLoadMoreError }}</p>`
   with the button's retry calling `fetchTeacherReviews(false)` so the next page is re-requested without discarding the loaded pages or scroll position.

#### 42. AccountProfilePage swallows a failed status fetch and renders every verification/contact row as "Missing / Unverified" with no loading or error state

`clients/web/src/modules/user/views/AccountProfilePage.vue:408`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | missing-error-state |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
AccountProfilePage.vue:406-415 —
  onMounted(() => {
    if (authStore.isAuthenticated) {
      void verificationStore.fetchStatus().catch((error) => {
        if (import.meta.env.DEV) {
          console.warn('[AccountProfilePage] failed to fetch verification status', error)
        }
      })
    }
  })

The template has no `v-if="loading"` and no error branch anywhere, and every badge falls back to a negative state when the store refs stay null: `phoneBadge` (line ~313) -> `status.unverified`, `qqBadge` -> `status.unbound`, `identityStatus` -> `identity.unverified`, `studentStatus` -> `student.unverified`.
```

**失败场景**

A verified student with a bound phone and QQ opens /account/profile while the API is briefly 500ing (or their session cookie has just been rotated). `fetchStatus()` rejects, the rejection is discarded in production, and the page paints instantly and confidently: "Real-name verification — Unverified", "Student verification — Unverified", "Phone — Unverified", "QQ — Unbound", plus every third-party disclosure row as unavailable. There is no spinner, no error banner, and no retry button, so the user believes their verifications were lost and re-enters the identity-verification flow (uploading ID photos again).

**修复方案**

In AccountProfilePage.vue, replace the silent catch with real state, reusing what already exists:

1. Add `const loadError = ref(false)` and a `loadStatus()` function:
   `async function loadStatus() { loadError.value = false; try { await verificationStore.fetchStatus() } catch { loadError.value = true; toast.error(t('common.loadFailed')) } }`
   Call it from `onMounted` behind the existing `authStore.isAuthenticated` guard. This matches IdentityVerificationPage.vue:640-644 / JoinStartPage.vue:222-238.

2. Bind the existing store `loading` ref (verification.ts:86 — already exported) and add `v-if`/`v-else` branches around the contact card, the verification-items grid and the disclosure grid: show a lightweight skeleton while `verificationStore.loading` is true and the first fetch has not resolved.

3. When `loadError` is true, render an error banner in place of the badge grids with a retry button wired to `loadStatus()` using `t('common.loadFailed')` and `t('common.actions.retry')` (both keys already exist in src/i18n/locales/{en-US,zh-CN}/common.ts).

4. Stop deriving negative statuses from `null`: track a `loaded` ref set to true only after a successful `fetchStatus()`, and require `loaded` before rendering `phoneBadge` / `qqBadge` / `identityStatus` / `studentStatus` and the `disclosureFields` badges, so "Unverified"/"Unbound" is only ever shown for data the server actually returned.

5. Add a component test under clients/web/src/modules/user/__tests__/ asserting that a rejected `fetchStatus()` renders the error banner and does not render an "Unverified" identity/student …

#### 43. 50 ms blacklist lookup is bound to the client request context and every timeout/cancellation trips one global fail-closed circuit breaker

`server/internal/pkg/middleware/auth.go:59`

| | |
|---|---|
| 区域 | 中间件 |
| 类别 | availability |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
auth.go:21 `const authBlacklistLookupTimeout = 50 * time.Millisecond`; auth.go:59-69:
```
blacklistCtx, cancel := context.WithTimeout(c.Request.Context(), authBlacklistLookupTimeout)
...
isBlacklisted, err := tokenService.GetBlacklist().IsBlacklisted(blacklistCtx, tokenString)
if err != nil { ...; return nil, errBlacklistFail }
```
blacklist.go:279-292: `exists, err := b.rdb.Exists(ctx, blacklistPrefix+hash).Result(); if err != nil { b.cb.RecordFailure(); ...; return true, fmt.Errorf("blacklist service unavailable") }`. The breaker is a single process-wide instance (`NewNamed("token_blacklist", {FailureThreshold: 5, Timeout: 30s})`, blacklist.go:65-69) shared by every token. Once open, `IsBlacklisted` returns `(true, err)` for any hash with no local cache entry (blacklist.go:265-277), which `resolveToken` maps to `errBlacklistFail` → `authBackendUnavailable` → 503 (auth.go:144-145).
```

**失败场景**

A `context.Canceled` from a user navigating away mid-request, or Redis p99 latency crossing 50 ms during a traffic spike, is recorded as a breaker failure rather than a caller-side abort. Five such results without an interleaved success open the breaker for 30 seconds; during that window every authenticated request whose token hash is not in the 30 s local revocation cache (i.e. essentially all of them, since only revoked hashes are cached) receives HTTP 503 from `AuthMiddleware`. A brief latency blip or a burst of aborted requests therefore escalates into a 30-second full authentication outage.

**修复方案**

1. In `blacklist.go`, stop treating caller-driven aborts as backend health signals — before `b.cb.RecordFailure()` at line 281, skip the failure record when the error is a parent cancellation:
```go
exists, err := b.rdb.Exists(ctx, blacklistPrefix+hash).Result()
if err != nil {
    if !errors.Is(err, context.Canceled) { // 客户端主动断开不代表 Redis 故障
        b.cb.RecordFailure()
    }
    ...
}
```
(keep RecordFailure for `DeadlineExceeded` and genuine Redis errors, and keep the fail-closed return).
2. In `auth.go:59` (and `auth_cookie.go:55`), use the repo's existing helper so a client abort cannot even reach the breaker: `blacklistCtx, cancel := ctxutil.DetachedTimeout(c.Request.Context(), authBlacklistLookupTimeout)` — `internal/pkg/ctxutil/context.go:27` already provides this and it is the established pattern (outbox/worker.go:211, admission/service_freshman.go:408).
3. Raise the lookup budget above realistic Redis p99 + scheduler jitter (e.g. 150-200 ms) so a brief stall does not register as a backend failure at all; making it env-configurable is optional.
4. Reduce the amplification of a transient blip: give the `token_blacklist` breaker a shorter open window (e.g. 3-5 s instead of 30 s) so a 100 ms Redis hiccup cannot cost 30 s of global 503 on the authentication path; fail-closed semantics stay unchanged.
5. Add a regression test asserting that a canceled request context does not increment the breaker's failure count (`CircuitBreakerMetrics()` failures stays 0) while a Redis error does.

#### 44. Bearer introspection result is accepted without checking sub or token type, so a refresh token authenticates as an access token

`server/internal/pkg/middleware/auth.go:84`

| | |
|---|---|
| 区域 | 中间件 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
middleware/auth.go:78-102 — the whole Bearer acceptance test is:
```
if !result.Active { return nil, errTokenRevoked }
if oidcClient.ApplicationKeyForClientID(result.GetAppID()) == "" { ... ErrInvalidAudience }
return &authResult{ userID: result.Sub, ... }, nil
```
`IntrospectionResult` (oidc/introspection.go:24-36) has no `token_type` field and `parseIntrospectionResult` / `decorateIntrospectionResult` never validate one; `result.Sub` is never required to be non-empty. Unlike the cookie path (`resolveOIDCCookieToken`, auth_cookie.go:28-47) there is no session lookup and no `AccessTokenHash` comparison, so nothing else narrows which of the user's credentials may be presented. docs/design/auth-and-session.md:56 states the opposite guarantee: "access / refresh token 区分 `typ`，refresh 不会被当作 access 验证".
```

**失败场景**

A native client receives `{accessToken, refreshToken, sessionID}` from POST /api/v1/auth/exchange-native (handler_login.go:666-671) and stores the 7-day refresh token on device. Sending `Authorization: Bearer <refreshToken>` to any authenticated endpoint introspects as active for a configured client, so the middleware admits it and `setClaimsToContext` populates the full identity — the refresh token becomes a long-lived general API credential instead of a rotation-only credential, contradicting the documented design and widening the blast radius of a refresh-token leak. Symmetrically, an introspection response that is active but carries no `sub` (e.g. a non-user grant for one of the three configured client IDs) passes the middleware with `userID == ""`, so downstream handlers run with an empty subject instead of being rejected.

**修复方案**

In the Bearer branch of resolveToken (server/internal/pkg/middleware/auth.go:78-102):
1. Reject empty subject: after the Active/audience checks, `if strings.TrimSpace(result.Sub) == "" { return nil, errTokenMalformed }` so an active non-user grant can never populate an identity with `userID == ""`.
2. Discriminate token type locally, not from the introspection body. Parse the presented bearer JWT's Casdoor type claims (`tokenType` / `TokenType`, cf. casdoor-go-sdk casdoorsdk/jwt.go:27-34 `IsRefreshToken()`) and reject when it is `"refresh-token"`. Best done by adding to server/internal/pkg/oidc a small `RejectNonAccessToken(raw string) error` that decodes the JWT payload (no signature trust needed for a reject-only decision) and returns an error when the refresh marker is present; even stronger, run the resolved application's existing JWKS verifier (`VerifyIDTokenForApplication`) on the bearer token in addition to introspection, which both authenticates the token locally and gives access to its claims.
3. Do NOT gate on the introspection response's `token_type` field, and do not make `token_type_hint=access_token` the enforcement mechanism — Casdoor returns `token_type: "Bearer"` for both access and refresh matches, so that check either no-ops or breaks all Bearer auth. Sending the hint is fine as an optimization only.
4. Add middleware tests alongside auth_bearer_integration_test.go: (a) introspection returns `{"active":true,"client_id":"client-id"}` with no `sub` -> 401 ErrTokenInvalid; (b) a Casdoor-shaped refresh-token JWT presented as Bearer, with introspection returni …

#### 45. Routes whose sole credential is a path parameter have that credential written verbatim into every request log

`server/internal/pkg/middleware/logging.go:93`

| | |
|---|---|
| 区域 | 中间件 |
| 类别 | credential-leak |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
middleware/logging.go:73,75,93-94:
	path := c.Request.URL.Path
	query := maskSensitiveQueryParams(c.Request.URL.RawQuery)
	...
		zap.String("path", path),
		zap.String("query", query),

middleware/logging.go:32-41 redacts `token`, `code`, `secret`, `internal_key`… but only in the query string. There is no path redaction.

The affected routes authenticate *solely* by that path segment (admission/handler.go:70,78-80):
	admission.GET("/sessions/:token", h.handlePreviewAdmissionSession)
	admission.GET("/freshman/mobile-camera-handoffs/:token", h.handlePreviewFreshmanCameraHandoff)
	admission.POST("/freshman/mobile-camera-handoffs/:token/camera-capture", h.handleUploadFreshmanCameraHandoffCapture)
	admission.POST("/freshman/mobile-camera-handoffs/:token/continue", h.handleChooseFreshmanCameraHandoffContinuation)

That the raw value is a bearer secret is confirmed by the service, which only ever stores the hash — admission/service_freshman.go:219,233: `s.repo.GetFreshmanCameraHandoffByTokenHash(ctx, s.hashToken(token))`.
```

**失败场景**

An applicant scans the QR code and their phone calls GET /api/v1/admission/freshman/mobile-camera-handoffs/9f3c…d1/. RequestLogger emits `http_request path=/api/v1/admission/freshman/mobile-camera-handoffs/9f3c…d1`. Anyone with read access to the log pipeline (ops staff, the log-shipping vendor, an SRE debugging a 500, or an attacker who obtains a log dump) copies that path and, before the handoff expires, replays POST …/9f3c…d1/camera-capture with their own base64 image — no cookie, no bearer, no CSRF token needed — and the image is stored as that applicant's admission material. The same log line for GET /api/v1/admission/sessions/<token> yields a live admission-invite token, which any authenticated attacker can then POST to /sessions/<token>/link to bind the invite to their own account. Note the platform already treats this exact secret as loggable-forbidden when it travels as ?token=…, so the protection is strictly weaker for these routes than for their query-string equivalents.

**修复方案**

Redact only sensitive path parameters, keeping raw-path fidelity everywhere else, and apply it to both log sites.

In server/internal/pkg/middleware/logging.go add:

    // sensitivePathParams 路由参数名黑名单（这些路径段的值会被脱敏）
    var sensitivePathParams = map[string]bool{
        "token": true, "code": true, "secret": true, "key": true,
        "credential": true, "session": true, "invite": true,
    }

    // maskSensitivePathParams 将路径中命中黑名单的路由参数值替换为 [REDACTED]。
    // 注意：c.Params 在路由匹配后即可用（中间件进入时已填充），
    // 未匹配路由（404）时 Params 为空，直接返回原始路径以保留排障信息。
    func maskSensitivePathParams(c *gin.Context, path string) string {
        for _, p := range c.Params {
            if p.Value == "" || !sensitivePathParams[strings.ToLower(p.Key)] {
                continue
            }
            path = strings.Replace(path, p.Value, "[REDACTED]", 1)
            if esc := url.PathEscape(p.Value); esc != p.Value {
                path = strings.Replace(path, esc, "[REDACTED]", 1)
            }
        }
        return path
    }

Then in RequestLogger, drop the pre-Next `path := c.Request.URL.Path` capture and compute it after c.Next() (params and route are resolved by then):

    zap.String("path", maskSensitivePathParams(c, c.Request.URL.Path)),

and in Recovery (logging.go:146) replace
    zap.String("path", c.Request.URL.Path),
with
    zap.String("path", maskSensitivePathParams(c, c.Request.URL.Path)),

Why this shape rather than c.FullPath(): FullPath() is "" for unmatched routes (see internal/pkg/metrics/metrics_test.go:109), so using it in the access log would erase 404 diagnostics and strip …

#### 46. Admission school-email endpoints fan out to the external Oracle source with no per-user rate limit

`server/internal/modules/admission/handler.go:81`

| | |
|---|---|
| 区域 | 入群认证 |
| 类别 | missing-throttle |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
admission.POST("/school-email/academic-match", authMW, h.handleMatchSchoolEmailAcademicStudent)
admission.POST("/school-email/request-otp", authMW, h.handleRequestSchoolEmailOTP)
// compare server/internal/modules/user/handler.go:114-115
// user.POST("/profile/school-email/academic-match", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, ...), ...)
// verifyRateLimitPerMinute = 5
```

**失败场景**

The admission Handler has no limiter fields at all, while the equivalent user routes are capped at 5 req/min/user. MatchSchoolEmailAcademicStudent -> resolveAcademicStudentEmail -> GetAcademicInfo issues one Oracle query per call, and RequestSchoolEmailOTP performs the same lookup before reserveEmailOTPCooldown, so the 60s OTP cooldown does not throttle it either. One authenticated user with a linked admission session loops academic-match up to the only remaining cap (API_IP_RATE_LIMIT, default 100/min since .env.prod.example does not set it). With EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS=4, concurrent loops saturate the pool; genuine lookups then block on pool acquisition until the 3s ctx expires, returning DeadlineExceeded, which is recorded as a breaker failure (line 220) -> breaker opens -> 30s of school-wide 503s. The school's DBA also sees unbounded query volume from StuHelper.

**修复方案**

Apply the limiter half only; drop the cooldown reorder.

1. `server/internal/modules/admission/handler.go` — add a limiter field and a functional option, mirroring the user module:
   - add `schoolEmailLimiter *middleware.RedisRateLimiter` to the `Handler` struct (line 17-22);
   - add `func WithSchoolEmailRateLimiter(l *middleware.RedisRateLimiter) HandlerOption` next to `WithAdminAuthorizers` (avoids changing the 3-arg `NewHandler` signature; alternatively accept `*redis.Client` as `user.NewHandler` does and build the limiter internally);
   - in `RegisterRoutes`, replace lines 81-83 with the nil-safe branch used at `user/handler.go:112-122`: when `h.schoolEmailLimiter != nil`, wrap each route in `middleware.EndpointRateLimitMiddleware(h.schoolEmailLimiter, "admission-school-email-academic-match" | "-request-otp" | "-verify-otp")`; otherwise register as today. Distinct endpoint labels matter — the key is `"rl:endpoint:"+endpoint+":user:"+userID`, so a shared label would make the three routes share one bucket.
   - Keep `authMW` first so `GetUserID(c)` is populated and the limiter keys per-user rather than falling back to per-IP.

2. `server/internal/app/modules.go:199-204` — pass `admission.WithSchoolEmailRateLimiter(middleware.NewRedisRateLimiter(rt.redisClient.GetClient(), 5, time.Minute))`, matching `verifyRateLimitPerMinute = 5` in `user/handler.go:46`. Define the constant in the admission package (e.g. `schoolEmailRateLimitPerMinute = 5`) rather than hardcoding it at the call site.

3. Test — add to `server/internal/modules/admission/` a handler test that drives 6 re …

#### 47. One failing row discards the whole claimed bot-action batch, permanently dead-lettering kick/release actions

`server/internal/modules/admission/service_bot_actions.go:53`

| | |
|---|---|
| 区域 | 入群认证 |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
```go
	rows, err := s.repo.ClaimDueBotActions(ctx, normalized, now)   // already flipped every row to 'dispatched', attempt_count+1
	...
	actions := make([]AdmissionPendingAction, 0, len(rows))
	for i := range rows {
		action, stale, err := s.pendingActionFromQueuedRow(ctx, &rows[i], now)
		if err != nil {
			return nil, err          // <-- discards rows[0..i-1] that were already claimed and are already ack-pending
		}
		if stale {
			if err := s.repo.MarkBotActionStale(ctx, rows[i].ID, now); err != nil {
				return nil, err      // <-- same loss
			}
			continue
		}
```

The claim already consumed the lease (repository_bot_action_outbox.go:139-147): `SET status = 'dispatched', attempt_count = attempt_count + 1, next_attempt_at = $5`. There is no per-row recovery: the batch is up to `maxPendingActionLimit = 200` (repository_bot_scan.go:12) and spans every guild that bot serves. At `admissionBotActionMaxAttempts = 5` (repository_bot_action_outbox.go:14) the claim SQL's `terminal` CTE (lines 110-124) flips them to `dead_letter`. That state is terminal: `QueueBotActionTx`'s ON CONFLICT …
```

**失败场景**

Postgres is slow for ~3 minutes (a vacuum, failover, or connection-pool saturation). `pendingActionFromQueuedRow` issues 2 queries per row (see the N+1 finding), each bounded by DB_QUERY_TIMEOUT=5s, so at least one row in every claim errors. Each 2s SSE tick claims up to 200 queued kick/remind/release rows, burns one attempt on every one of them, then returns HTTP 500 / an SSE `error` frame and delivers none. After 5 such rounds (~2.5 min, spaced by next_attempt_at = now+30s) every queued action is `dead_letter`. Concretely: a student who just finished verification has a `BotActionRelease` row keyed `<sessionID>:release` (repository_bot_action_outbox.go:281-282); once it dead-letters the bot never un-mutes them, and because the key is stable and ON CONFLICT preserves `dead_letter`, neither `MarkVerified` nor `ProjectStudentVerification` re-queueing can ever revive it — the verified student stays muted in the QQ group forever without manual DB surgery.

**修复方案**

1) server/internal/modules/admission/service_bot_actions.go (ClaimQueuedAdmissionActions, lines 49-63): make the loop per-row fault-isolated, mirroring outbox.ProcessBatch (server/internal/pkg/outbox/worker.go:104-137). On pendingActionFromQueuedRow error: logger.L().Warn with action_id/session_id/error, call the new repo lease-release below, then continue. On MarkBotActionStale error: log and continue instead of returning. Only propagate an error when ClaimDueBotActions itself fails. Add a `if ctx.Err() != nil` check at the top of each iteration that breaks out and releases the leases of the remaining rows using a fresh short context derived from context.Background() (as outbox.abandonClaimedJobs/finalizeContext does), so a client disconnect mid-batch does not burn attempts.

2) server/internal/modules/admission/repository_bot_action_outbox.go: add a lease-release method modeled on outbox.MarkJobAbandoned (repository.go:348-361), fenced on the claimed attempt so a concurrent newer claim is never clobbered:
   func (r *Repository) AbandonBotActionLease(ctx context.Context, actionID int64, dispatchAttempt int, now time.Time) error, executing
   UPDATE admission_bot_action_outbox SET status = 'failed', attempt_count = GREATEST(attempt_count - 1, 0), next_attempt_at = $2, updated_at = $2 WHERE id = $1 AND status = 'dispatched' AND attempt_count = $3
   (use withDBTable(ctx, admissionBotActionOutboxTable); treat RowsAffected()==0 as a lost lease that is logged, not returned, so it cannot re-abort the batch).

3) Close the terminal-state trap for this table: add a requeue mirror …

#### 48. Bot action claim issues 2 extra DB queries per claimed row instead of one batched lookup

`server/internal/modules/admission/service_bot_actions.go:137`

| | |
|---|---|
| 区域 | 入群认证 |
| 类别 | efficiency |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
```go
func (s *Service) pendingActionFromQueuedRow(ctx context.Context, row *AdmissionBotActionOutboxRow, now time.Time) (...) {
	...
	seeds := pendingActionSeeds([]AdmissionSession{session}, now)
	contexts, err := s.pendingActionContexts(ctx, []AdmissionSession{session})   // per row!
```

`pendingActionContexts` (service_bot_action_contexts.go:35-49) always runs two queries — `ListPoliciesByGuildKeys` and `ListAdmissionFailuresByKeys` — and it is called once per claimed row from the loop at service_bot_actions.go:50-62. The sibling read path shows the intended batched shape: `ListPendingAdmissionActions` (service_bot_actions.go:26) calls `pendingActionContexts(ctx, sessions)` exactly once for the whole slice, and the repository helpers are already written to accept key slices (`unnest($1::text[], $2::text[])`, repository_bot_contexts.go:57-72).
```

**失败场景**

A bot connected to /api/v1/bot/admission/actions/stream ticks every 2s (handler_bot_queries.go:132). After a mass join event the queue holds 200 due actions (maxPendingActionLimit). One tick then executes 1 claim transaction + 400 point queries, each taking a pooled connection with a 5s timeout, and the next tick fires 2s later. With DB_MAX_CONNS at its default this alone saturates the pool, slows every user-facing request, and — because a single one of those 400 queries timing out aborts the whole batch (see the batch-discard finding) — it is also the trigger that burns the retry budget on all 200 actions.

**修复方案**

Hoist the context lookup out of the loop in /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_bot_actions.go, mirroring `ListPendingAdmissionActions`.

In `ClaimQueuedAdmissionActions`, after the `len(rows) == 0` early return (line 46-48):

    sessions := make([]AdmissionSession, len(rows))
    for i := range rows {
        sessions[i] = rows[i].Session
    }
    seeds := pendingActionSeeds(sessions, now)
    contexts, err := s.pendingActionContexts(ctx, sessions)
    if err != nil {
        return nil, err
    }
    actions := make([]AdmissionPendingAction, 0, len(rows))
    for i := range rows {
        action, stale, err := s.pendingActionFromQueuedRow(&rows[i], &sessions[i], seeds[i], contexts)
        // ... unchanged stale / MarkBotActionStale / append handling

Then change the signature to drop `ctx` and `now` and accept the shared data:

    func (s *Service) pendingActionFromQueuedRow(
        row *AdmissionBotActionOutboxRow,
        session *AdmissionSession,
        seed pendingActionSeed,
        contexts pendingActionContexts,
    ) (AdmissionPendingAction, bool, error) {
        if row == nil || session == nil {
            return AdmissionPendingAction{}, true, nil
        }
        if !sessionCanDispatchQueuedBotAction(session) {
            return AdmissionPendingAction{}, true, nil
        }
        action, err := s.pendingActionFromSession(session, seed, contexts)
        if err != nil {
            return AdmissionPendingAction{}, false, err
        }
        if !queuedActionCanDispatch(row.Action, action.Action) {
            return …

#### 49. ClaimQueuedAdmissionActions reloads policies and failure counters once per claimed row instead of batching

`server/internal/modules/admission/service_bot_actions.go:137`

| | |
|---|---|
| 区域 | 入群认证 |
| 类别 | efficiency |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
func (s *Service) pendingActionFromQueuedRow(ctx context.Context, row *AdmissionBotActionOutboxRow, now time.Time) (...) {
	...
	seeds := pendingActionSeeds([]AdmissionSession{session}, now)
	contexts, err := s.pendingActionContexts(ctx, []AdmissionSession{session})  // 2 queries, per row
// invoked from the ClaimQueuedAdmissionActions loop at service_bot_actions.go:50-62
```

**失败场景**

The Koishi guard polls POST /api/v1/bot/admission/actions/claim. ClaimDueBotActions returns up to filter.Limit rows (maxPendingActionLimit = 200, repository_bot_scan.go:12). For each row the loop calls pendingActionContexts, which runs ListPoliciesByGuildKeys and ListAdmissionFailuresByKeys (repository_bot_contexts.go:17 and :41), so a full batch costs 400 extra round trips per poll instead of 2, re-reading the same handful of policy rows 200 times. The sibling read path ListPendingAdmissionActions (service_bot_actions.go:26) already batches this correctly via pendingActionContexts(ctx, sessions), so the claim path is an unintended divergence.

**修复方案**

Batch the context load once per claim in /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_bot_actions.go, mirroring `ListPendingAdmissionActions`.

1. In `ClaimQueuedAdmissionActions` (lines 33-64), after the `len(rows) == 0` early return at line 46, build the session slice and load contexts once:

    sessions := make([]AdmissionSession, 0, len(rows))
    for i := range rows {
        sessions = append(sessions, rows[i].Session)
    }
    contexts, err := s.pendingActionContexts(ctx, sessions)
    if err != nil {
        return nil, err
    }

Building the slice from all rows (rather than pre-filtering with `sessionCanDispatchQueuedBotAction`) is fine and simpler: `pendingActionLookupKeys` dedupes both key sets through maps, so a few extra keys cost nothing and it stays a single query.

2. Change the helper signature at line 124 from
   `func (s *Service) pendingActionFromQueuedRow(ctx context.Context, row *AdmissionBotActionOutboxRow, now time.Time)`
   to
   `func (s *Service) pendingActionFromQueuedRow(row *AdmissionBotActionOutboxRow, contexts pendingActionContexts, now time.Time)`
   and delete lines 137-140 (the per-row `s.pendingActionContexts` call and its error branch). The `context.Context` parameter becomes unused — drop it. Keep the per-row `pendingActionSeeds([]AdmissionSession{session}, now)` at line 136 as-is (it is pure and does no I/O), or index into a batched `pendingActionSeeds(sessions, now)` for symmetry with `pendingActionsFromSessions`.

3. Update the call site at line 51 to `s.pendingActionFromQueuedRow(&rows[i], contexts, now)`. …

#### 50. A panic in any job handler permanently kills its outbox worker for the process lifetime

`server/internal/pkg/outbox/worker.go:112`

| | |
|---|---|
| 区域 | 后端公共包 |
| 类别 | resilience |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
`ProcessBatch` invokes the job handler with no recover, and `RunPollingWorker` wraps nothing:

```go
		if err := process(ctx, job); err != nil {
```

The only recover is at the goroutine root, `server/internal/app/runtime.go:151-169`:

```go
	go func() {
		defer rt.bgWg.Done()
		defer func() {
			if r := recover(); r != nil { ... logger.L().Error("background task panicked", fields...) }
		}()
		...
		run(ctx)          // never re-invoked
	}()
```

A panic therefore unwinds the entire `for { ... }` poll loop inside `RunPollingWorker`, logs one line, and the goroutine exits for good. This governs all six workers started via `startBackgroundTask`: review notification (service_notification_outbox.go:100), review FGA sync (service_fga_sync.go:94), teacher projection (teacher_projection_worker.go:12), user external sync (external_sync.go:173), resource cleanup (background_cleanup.go:49), plus the two hand-rolled loops `runFreshmanExpiryWorker` / `runMemberBlacklistExpiryWorker` (service_expiry.go:25-49). outbox/worker_test.go has no panic-recovery test.
```

**失败场景**

`processExternalSyncJob` → `syncVerifiedStudentRole` (or the Casdoor/FGA client beneath it) nil-derefs on one malformed record — e.g. a role-sync payload for a user whose Casdoor subject was deleted, so a nil `*RoleSyncClient` field is dereferenced. The panic kills the "user external sync worker" goroutine. The process keeps serving HTTP and passes /health, but from that moment no Casdoor role sync, no FGA tuple projection and no admission-verification projection is ever processed again: `domain_event_outbox` grows unbounded and every newly verified student silently fails to receive the `verified_student` role until someone notices the single "background task panicked" log line and restarts the pod. The claimed job is stuck in `processing` and only becomes re-claimable after LockStaleAfter (2 min) — by a replica that will panic on it too.

**修复方案**

Two changes, both narrow, plus a test. The proposed fix in the finding is the right shape; I would apply it with a corrected rationale (poison-pill dead-lettering and restoring the metric signal, not the fictional nil-RoleSyncClient deref).

1. server/internal/pkg/outbox/worker.go - convert a handler panic into an ordinary job error so the EXISTING retry/dead-letter accounting applies. Add (importing "runtime/debug"):

    func safeProcess[T any](ctx context.Context, process ProcessFunc[T], job T) (err error) {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("job handler panicked: %v\n%s", r, debug.Stack())
            }
        }()
        return process(ctx, job)
    }

and change line 112 to `if err := safeProcess(ctx, process, job); err != nil {`. This is the load-bearing half: it routes the panic through markFailure -> reachedMaxAttempts (terminal after IAMWorkerMaxAttempts=5) so a poison-pill job dead-letters instead of killing the worker, AND it makes metrics.ObserveOutboxJobFailure fire so existing alerting sees the problem. It also stops the 2-minute LockStaleAfter thrash where each replica re-claims and re-panics on the same row.

2. server/internal/app/runtime.go:145-170 - make startBackgroundTask supervise instead of exiting. Restructure so the recover wraps a single invocation and the goroutine retries while the context is live: `for ctx.Err() == nil { runOnce(ctx); if ctx.Err() != nil { break }; sleep(backoff) }`, where runOnce holds the `defer recover()`. Use capped backoff (1s doubling to 30s, reset after a s …

#### 51. Refresh tokens revoked by logout are indistinguishable from reused ones, so a post-logout refresh retry revokes every session the user has

`server/internal/pkg/token/session.go:239`

| | |
|---|---|
| 区域 | 后端公共包 |
| 类别 | correctness |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
session.go:238-248 (`revokeLoadedSession`, reached from Logout via Service.RevokeSession → SessionStore.Revoke):
```
if data.RefreshTokenHash != "" {
    if err := s.RememberRefreshTokenHash(ctx, data.RefreshTokenHash, RefreshTokenRef{SessionID: ..., UserID: ...}, refreshTTL); ...
    if blErr := blacklist.AddByHash(ctx, data.RefreshTokenHash, refreshTTL); ...
}
```
handler_session.go:137-145: `blacklisted, err := ...IsBlacklisted(...); if blacklisted { h.rejectRefreshReuse(c, refreshTokenStr); return }`
handler_refresh_reuse.go:19-41: `ref, err := h.lookupRefreshTokenRef(...)`; `if ref == nil { 401 revoked; return }` … `metrics.ObserveRefreshTokenReuse(...)`; `h.svc.RevokeAllSessions(c.Request.Context(), ref.UserID)`.
The ref written at logout time is exactly what makes `ref != nil`, so a logout-revoked token takes the family-revocation branch instead of the benign `ref == nil` branch. There is no flag distinguishing "revoked by logout" from "rotated and reused".
```

**失败场景**

User is logged in on the web (session A) and on the phone app (session B). On the phone they tap Logout: session B is revoked and B's refresh token hash is both blacklisted and remembered as a refresh ref. The phone still has the refresh token in storage; a queued API call finishes with 401 and the app's interceptor fires POST /api/v1/auth/refresh with that token (same for a browser whose refresh request was already in flight when logout completed). The handler sees `blacklisted == true`, `rejectRefreshReuse` finds the remembered ref, and calls `RevokeAllSessions(userID)` — silently killing the user's web session A as well, incrementing the refresh-reuse metric, and writing a false `refresh_reuse_detected` security audit record with Result=failure.

**修复方案**

Discriminate revoked-by-logout from rotated-and-reused before doing family revocation, and close the cookie-path gap:

1. In `rejectRefreshReuse` (server/internal/modules/auth/handler_refresh_reuse.go:18), after resolving `ref`, load the tracked session by `ref.SessionID` via `loadTrackedSession`. If it returns `token.ErrSessionNotFound` (session was deleted by Logout/LogoutAll/admin revoke), respond `response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)` and return — no `RevokeAllSessions`, no `metrics.ObserveRefreshTokenReuse`, no `refresh_reuse_detected` audit. Only when the session still exists and its current `RefreshTokenHash` differs from the presented token's hash (i.e. the token was genuinely superseded by a rotation) keep the existing family-revocation + metric + audit branch. On a Redis error from that lookup, keep the current fail-closed 503.

2. Optionally reinforce with an explicit marker: extend `token.RefreshTokenRef` with `Reason string` (`"rotated"` vs `"revoked"`), set `Reason: "revoked"` in `revokeLoadedSession` (server/internal/pkg/token/session.go:239) and `"rotated"` in `Service.RotateSession` (service.go:212), and treat `Reason == "revoked"` as the plain-401 branch. This is belt-and-braces; the session-existence check in (1) is the load-bearing part because refs are also written by CreateSession/Touch.

3. Mirror the native tracked-session existence check on the cookie path (drop the `if !fromBody { return true }` short-circuit in `requireTrackedNativeRefreshSession`, or add an equivalent `requireTrackedSession` call for cookie refr …

#### 52. Ansible deploy playbook builds the bundle with playbook-relative paths through ansible.builtin.command, which never resolves them

`infra/ansible/playbooks/deploy.yml:22`

| | |
|---|---|
| 区域 | 基础设施与运维 |
| 类别 | deployment-correctness |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
- name: Build deploy bundle on control node
      delegate_to: localhost
      ansible.builtin.command:
        cmd: ../../ops/build-deploy-bundle.sh ../../generated/deploy/stuhelper-deploy-bundle.tar.gz
```

**失败场景**

`ansible.builtin.command` executes a raw process with no `chdir`, so relative paths resolve against the controller process's working directory — never against the playbook directory (only file-lookup modules like `script`, `copy`, `template` do that). `Makefile:233` runs `cd infra/ansible && ansible-playbook -i inventory/production.ini playbooks/deploy.yml`, so the cwd is `<repo>/infra/ansible` and `../../ops/build-deploy-bundle.sh` resolves to `<repo>/ops/build-deploy-bundle.sh`, which does not exist → the task fails immediately with "No such file or directory", so `make ansible-deploy-prod` / `make ansible-deploy-staging` never reach the upload step. The neighbouring lines prove the inconsistency: line 6 uses `lookup('pipe', 'git -C ../..')` (correct for cwd `infra/ansible`) while lines 22 and 32 use `../../ops/...` and `../../generated/...` (only correct for cwd `infra/ansible/playbooks`, which is what `ansible.builtin.copy` on line 32 actually uses). Even if the second arg were reachable it would write the tarball to `<repo>/generated/deploy/`, where the `copy` task on line 32 (resolving to `infra/generated/deploy/`) would not find it.

**修复方案**

1) infra/ansible/playbooks/deploy.yml:19-22 - anchor the command and drop the redundant output argument, since infra/ops/build-deploy-bundle.sh already self-anchors via REPO_ROOT (infra/ops/lib/common.sh:5) to exactly the path the upload task reads:

    - name: Build deploy bundle on control node
      delegate_to: localhost
      ansible.builtin.command:
        argv:
          - "{{ playbook_dir }}/../../ops/build-deploy-bundle.sh"
      changed_when: true

(Equivalent acceptable form: `chdir: "{{ playbook_dir }}/../.."` plus `cmd: ./ops/build-deploy-bundle.sh ./generated/deploy/stuhelper-deploy-bundle.tar.gz` - note that anchor is <repo>/infra, not repo root.)

2) infra/ansible/playbooks/deploy.yml:32 - make the upload src explicitly playbook-anchored so build output and upload input are provably the same file: `src: "{{ playbook_dir }}/../../generated/deploy/stuhelper-deploy-bundle.tar.gz"`.

3) Add regression coverage, which is absent today: create infra/ops/tests/ansible-playbook-path-contract.sh (auto-discovered by infra/ops/tests/run-infra-contracts.sh, which globs *.sh) that, for every infra/ansible/playbooks/*.yml, (a) fails if any `command`/`shell` task uses a `cmd:`/`argv:` path beginning with `./` or `../` without a `chdir:`, and (b) resolves every `{{ playbook_dir }}`-anchored path plus every `ansible.builtin.script:` / `copy: src:` relative path against infra/ansible/playbooks and asserts the target exists on disk. That catches both the missing script and any future build/upload path drift without needing ansible installed in CI.

#### 53. Academic student import violates the sfzjh enc/hash pair constraint, aborting the documented fallback import

`infra/ops/import-buaa-academic-students.sh:232`

| | |
|---|---|
| 区域 | 基础设施与运维 |
| 类别 | data-integrity |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
INSERT INTO academic.buaa_students (
  xh, xm, sfzjlxdm, sfzjh_hash, yxdm, zydm, bjdm, xznj, rxnj, pyccdm,
  xslbdm, sjh, dzxx, xjztdm, sfzx, sfzj, synced_at
)
...
  NULLIF(btrim(sfzjh_hash), ''),
...
ON CONFLICT (xh) DO UPDATE
SET ...
    sfzjh_hash = EXCLUDED.sfzjh_hash,
```

**失败场景**

server/migrations/000001_initial_schema.up.sql:71 enforces CONSTRAINT chk_buaa_students_sfzjh_secure_pair CHECK ((sfzjh_enc IS NULL AND sfzjh_hash IS NULL) OR (sfzjh_enc IS NOT NULL AND sfzjh_hash IS NOT NULL)). The script's own usage text (line 45) advertises sfzjh_hash as a supported optional column while line 48 states sfzjh_enc is deliberately never imported, so the INSERT always leaves sfzjh_enc NULL. An operator following docs/guides/production-go-live.md:180 runs BUAA_ACADEMIC_STUDENTS_TSV=<tsv with a populated sfzjh_hash column> ./infra/ops/import-buaa-academic-students.sh: validate-only passes, then psql (-v ON_ERROR_STOP=1) aborts the whole transaction with 'new row for relation "buaa_students" violates check constraint "chk_buaa_students_sfzjh_secure_pair"' and zero rows are imported. The ON CONFLICT branch has the mirror-image bug: for any existing row that has both enc and hash, re-importing a TSV without sfzjh_hash sets sfzjh_hash = NULL against a non-NULL sfzjh_enc and fails the same constraint.

**修复方案**

Apply option (a) from the proposal - it matches the script's own stated contract that encrypted identity columns are not loaded from this TSV. In infra/ops/import-buaa-academic-students.sh: (1) remove "sfzjh_hash" from the Python `columns` list at line 121, and add a fail-fast right after `header` is built (around line 145) that raises SystemExit if the header contains sfzjh_hash or sfzjh_enc, e.g. `forbidden = [c for c in ("sfzjh_hash", "sfzjh_enc") if c in header]` -> `raise SystemExit(f"BUAA academic TSV must not supply {', '.join(forbidden)}: academic.buaa_students enforces chk_buaa_students_sfzjh_secure_pair, so encrypted identity columns must be written as a pair by the dedicated encrypted identity sync path")`; (2) delete `sfzjh_hash text,` from the TEMP TABLE at line 214, delete sfzjh_hash from the \copy column list at line 229, delete it from the INSERT column list at line 232, and delete the `NULLIF(btrim(sfzjh_hash), ''),` select item at line 239 - all four must change together so the normalized header and the copy list stay aligned; (3) delete `sfzjh_hash = EXCLUDED.sfzjh_hash,` from the ON CONFLICT SET list at line 257 so a re-import can never null a hash that is paired with an existing sfzjh_enc (simpler and strictly safer than the proposed COALESCE); (4) drop `sfzjh_hash` from the "Supported optional columns" usage text at line 45 and extend the note at lines 48-49 to say both sfzjh_enc and sfzjh_hash are never imported here because the schema enforces chk_buaa_students_sfzjh_secure_pair. Also remove `sfzjh_hash` from the 可选列 list in docs/guides/production-go …

#### 54. Caller context cancellation is recorded as an external-source failure and trips the circuit breaker

`server/internal/modules/externaldata/oracle_student_directory.go:218`

| | |
|---|---|
| 区域 | 外部数据源 |
| 类别 | resilience-correctness |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
queryCtx, cancel := withOptionalTimeout(ctx, d.queryTimeout)
defer cancel()

rows, err := d.db.QueryContext(queryCtx, d.query, normalizedID)
if err != nil {
	d.breaker.RecordFailure()
	return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
}
```

**失败场景**

queryCtx is derived from the inbound Gin request context (admission/handler_user.go:370 passes c.Request.Context()). net/http cancels that context when the client disconnects. Five students on flaky mobile networks close the page / their app times out mid-request on POST /api/v1/admission/school-email/academic-match: QueryContext returns context.Canceled, so RecordFailure() runs 5 times. circuitbreaker.RecordFailure has no decay window (failures only reset on RecordSuccess, circuitbreaker.go:203-205), so with no successful lookup in between the breaker opens for EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS=30. Every subsequent lookup for that school then fails fast with "circuit breaker open" -> ErrAcademicLookupUnavailable -> HTTP 503 "academic student lookup is temporarily unavailable" for all students, while the Oracle source is perfectly healthy. Half-open admits only one probe, so one more abandoned request re-opens it for another 30s.

**修复方案**

All in `server/internal/modules/externaldata/oracle_student_directory.go`:

1. Add a helper that checks the PARENT ctx (deliberately, so the directory's own `queryTimeout` expiry still counts as a source failure, since that leaves `ctx.Err() == nil`):

```go
// callerAborted reports whether err came from the inbound caller (client
// disconnect or the caller's own deadline) rather than from the Oracle source.
// The parent ctx is checked on purpose: d.queryTimeout expiring leaves
// ctx.Err() == nil and must keep feeding the breaker.
func callerAborted(ctx context.Context, err error) bool {
	if ctx.Err() == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
```

2. Funnel every failure path through one method instead of calling `RecordFailure` inline:

```go
func (d *OracleStudentDirectory) failSource(ctx context.Context, operation string, err error) error {
	if callerAborted(ctx, err) {
		return err // caller went away: says nothing about source health
	}
	d.breaker.RecordFailure()
	return wrapOracleStudentSourceFailure(operation, err)
}
```

Apply at line 219-222 (`return nil, d.failSource(ctx, "lookup oracle student record", err)`), line 239-242, line 243-246, and inside the deferred close at 231-237 (keeping the existing "only overwrite when err == nil" semantics). Do the same in `Probe` at 172-176 and 182-189. Leave all `RecordSuccess()` calls unchanged, and keep the `sql.ErrNoRows` branch at 183-186 as is.

3. Stop labelling caller aborts as dependency errors, since `infra/observability/prometheus/rules/appli …

#### 55. Per-record data-integrity rejection is reported as source unavailability and opens the shared breaker

`server/internal/modules/externaldata/oracle_student_directory.go:238`

| | |
|---|---|
| 区域 | 外部数据源 |
| 类别 | resilience-correctness |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
record, err = scanOracleStudentRecords(rows, d.schoolCode, normalizedID)
if err != nil {
	d.breaker.RecordFailure()
	return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
}
// scanOracleStudentRecords (line 462):
// if id == "" || id != expectedStudentID || !schoolauth.IsValidStudentID(id) ||
//	!name.Valid || !schoolauth.IsValidAcademicName(studentName) {
//	return nil, ErrStudentSourceInvalidRecord
// }
```

**失败场景**

One row in USR_JWBIZ.T_XS_JBXX (a ~130k-row table per infra/ops/tests/provision-external-student-source-oracle-readonly-contract.sh:64) has XM NULL, or a name containing a zero-width/format character, or >80 runes. The student whose XH matches that row calls POST /api/v1/admission/school-email/academic-match: scanOracleStudentRecords returns ErrStudentSourceInvalidRecord, which is wrapped as ErrStudentSourceUnavailable -> 503 "temporarily unavailable", so the user retries. Each retry adds a breaker failure (a permanent per-row data defect can never produce a success to reset the counter), so after 5 retries the breaker opens and every other student of that school gets 503 for 30s. One malformed external row plus one retrying user takes the whole school's verification flow offline.

**修复方案**

Separate "source unhealthy" from "this row is unusable", and keep the API contract stable.

1. server/internal/modules/externaldata/oracle_student_directory.go — split the sentinel first, so a bind-parameter violation stays a source failure:
   - In scanOracleStudentRecords (line 462), split the `id != expectedStudentID` condition out into a new `ErrStudentSourceRecordIdentityMismatch` (the source ignored the bind variable — keep this as RecordFailure + ErrStudentSourceUnavailable, it is a real source-integrity defect).
   - Keep ErrStudentSourceInvalidRecord for name-level data problems (!name.Valid, !IsValidAcademicName, empty id) and ErrStudentSourceAmbiguousRecord for conflicting duplicates.
   - At LookupStudent line 238-242:
       record, err = scanOracleStudentRecords(rows, d.schoolCode, normalizedID)
       if err != nil {
           if errors.Is(err, ErrStudentSourceInvalidRecord) || errors.Is(err, ErrStudentSourceAmbiguousRecord) {
               d.breaker.RecordSuccess() // the round trip succeeded; only the row is unusable
               return nil, err           // do NOT wrap in ErrStudentSourceUnavailable
           }
           d.breaker.RecordFailure()
           return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
       }
     (The existing deferred closeRows at line 231-237 stays inert because err != nil, so rows are still closed and no second RecordFailure fires.)
   - Add observability so a malformed row is not invisible: a dedicated counter (e.g. metrics counter external_student_source_invalid_records_total{school_code,reas …

#### 56. review_preview_content_chars / review_preview_content_percent are validated, parsed and cached but never applied; content previews are truncated with the title budget

`server/internal/modules/course/review/access.go:241`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | integration-gap |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
access.go:236-259 — both preview branches pass the TITLE budget and hardcode percent=100:
	case !facts.Authenticated:
		result[i].Content = previewFirstContentLine(result[i].Content, facts.PreviewTitleRunes)
		result[i].Title = ""
	case !facts.CanViewFull:
		result[i].Content = previewFirstContentLine(result[i].Content, facts.PreviewTitleRunes)
	...
	func previewFirstContentLine(value string, maxRunes int) string {
		for _, line := range strings.Split(value, "\n") {
			preview := previewText(line, maxRunes, 100)

facts.PreviewContentRunes / facts.PreviewContentPct are populated (access.go:180-181) from admin config (access.go:136-147) and validated on write (user/service_admin.go:470-477), but `grep -rn "PreviewContentRunes\|PreviewContentPct"` shows zero non-test production readers. previewText's percent branch (`percent > 0 && percent < 100`, access.go:268) is therefore dead in production.
```

**失败场景**

An operator sets review_preview_content_chars=400 and review_preview_content_percent=30 via PUT /api/v1/admin/system-configs/{key} to widen previews for unverified visitors. The value passes validation, is stored, InvalidateReviewAccessPolicySnapshot fires, and buildReviewAccessPolicy loads it into the snapshot — yet GET /course/review/courses/{id}/reviews still truncates content at PreviewTitleRunes (default 24) runes with no percentage applied. The knob has no observable effect at any value, and content previews are 5x shorter than the configured/documented content budget of 120.

**修复方案**

Stop shipping a half-wired config surface. Pick one of the two directions below; do not leave the current state.

Direction A (wire the knobs, preserving the single-line safety from 58243689) — preferred:
1. server/internal/modules/course/review/access.go:251 — change to `func previewFirstContentLine(value string, maxRunes int, percent int) string` and pass `previewText(line, maxRunes, percent)` instead of the hardcoded 100. Keep the first-non-empty-line loop so multi-line content can never leak past line 1.
2. access.go:243-245 (`!facts.CanViewFull`) — use `previewFirstContentLine(result[i].Content, facts.PreviewContentRunes, facts.PreviewContentPct)`. This matches the docs tier "已登录未认证 → 评课正文有限制" (docs/product-specs/course-review.md:133) and restores the pre-58243689 semantics without reintroducing multi-line exposure.
3. access.go:240-242 (`!facts.Authenticated`) — decide explicitly with product. Safest default: keep `facts.PreviewTitleRunes` here (guest teaser stays tight at 24) and document that `review_preview_title_chars` governs the guest teaser; otherwise use the content budget here too, but then note that guest exposure widens from 24 to 120 runes by default.
4. Fix the title knob's documented meaning: either re-apply `result[i].Title = previewText(result[i].Title, facts.PreviewTitleRunes, 100)` in the `!facts.CanViewFull` branch, or update the seeded description of `review_preview_title_chars` in a new migration so it says "游客正文预览最大字符数" and no longer claims to truncate titles.
5. Tests in server/internal/modules/course/review/access_policy_test.go: add a case ass …

#### 57. Scope-bound admin:reviews:manage is read as a global flag in review access facts

`server/internal/modules/course/review/access.go:190`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | authz-scope-bypass |
| 验证票数 | 2/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
facts.CanManageReviews = capability.Has(capabilities, capability.AdminReviewsManage)
...
facts.CanViewFull = facts.CanManageReviews || (canViewFull && facts.StudentVerified)
```

**失败场景**

capability.ExpandRoleGrants binds school_admin/section_admin/section_moderator grants to a school/section scope, but BuildUserAccessSnapshot still lists the bare name in Capabilities (the package's own test asserts GlobalCapabilities is empty while Capabilities contains AdminReviewsManage — capability_test.go:89-105). middleware.GetCapabilities(c) returns that flat list, so a section_moderator scoped only to section `review-moderation:1001` — who is not student-verified and whose school is not in `review_access_school_ids` — gets CanManageReviews=true and therefore CanViewFull=true for every school. Calling GET /api/v1/course/review/reviews/search?q=x or /courses/:id/reviews returns full review bodies for all schools, while a genuinely verified student of a non-allowlisted school gets only the truncated preview. The same flag also disables the StatusHidden redaction branch in stripReviewsForResponse (access.go:237) for that user on any path that surfaces non-published rows.

**修复方案**

Stop deriving a platform-wide manage flag from the flat capability list. Concretely:

1. Change `Service.ResolveAccessFacts` (server/internal/modules/course/review/access.go:172) to take `grants []capability.Grant` instead of (or in addition to) `capabilities []string`, and have `resolveReviewAccessFactsForRequest` (access.go:218-222) pass `middleware.GetCapabilityGrants(c)`.
2. Replace `facts.CanManageReviews = capability.Has(capabilities, capability.AdminReviewsManage)` (access.go:190) with `capability.HasGlobalGrant(grants, capability.AdminReviewsManage)`, and keep the derived `facts.CanViewFull` (access.go:210) based on that global-only flag.
3. Add scoped school ids to the facts (e.g. `ManageSchoolIDs map[int64]struct{}`), populated from the grants using the existing mapping helpers in admin_scope.go (`ScopeSchoolIDs` for `school_admin`, plus `schoolIDFromReviewModerationSectionID` for `ScopeSectionIDs`), then make `stripReviewsForResponse` (access.go:231) decide per row: a row is shown in full if the user has the global grant, or the row's `SchoolID` (already present on `Review`, model.go:59) is in `ManageSchoolIDs`, or the existing `canViewFull && StudentVerified` condition holds. Do the same in `response_contract.go:33`'s grouped path.
4. Add a unit test asserting that `section_moderator` scoped to `school_4111010006_review_moderation`, not student-verified, gets truncated content for a review whose `school_id` is a different school, and full content for a review in 4111010006.

#### 58. admin:reports:manage is enforced nowhere; moderation routes authorize on raw JWT role strings

`server/internal/modules/course/review/admin_scope.go:28`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | capability-model-drift |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
func requireModerationRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !hasAnyModerationRole(middleware.GetRoles(c)) { ... }

func hasAnyModerationRole(roles []string) bool {
	return hasRole(roles, roleSuperAdmin) || hasRole(roles, roleSchoolAdmin) ||
		hasRole(roles, roleSectionAdmin) || hasRole(roles, roleSectionModerator)
}
```

**失败场景**

capability.AdminReportsManage is declared (catalog.go:6), granted to four roles (catalog.go:40,53,58,61) and listed in AdminEntryCapabilities (catalog.go:78), yet a repo-wide grep shows no rbac.RequireCapability/RequireGlobalCapability ever references it — the only consumers are the catalog itself. GET /admin/reports, PUT /admin/reports/:reportID, GET/PUT /admin/reviews, PATCH /admin/reviews/batch and the content-flag routes (handler.go:168-188) are gated by requireModerationRole()/requireSchoolAdminRole(), which read middleware.GetRoles(c) and never consult capabilities or grants. Concretely: an operator removes AdminReportsManage from "section_moderator" in catalog.go — the documented single control point (docs/design/authorization-model.md:21) — and section moderators keep full access to report listing and processing, because the role name still matches. Conversely section_reviewer, listed as a first-class role in the docs, has an empty capability set (catalog.go:63) and is missing from hasAnyModerationRole, so it grants literally nothing anywhere.

**修复方案**

Make capabilities the server-side gate for the review moderation surface, preserving today's effective permissions exactly:

1. Extend `review.AdminAuthorizers` (used by `server/internal/app/admin_authorizers.go:68-77`) with `ReportsManage`, `ReviewsModerate`, and `ReviewsEditContent`, and wire them:
   - `ReportsManage: rbac.RequireCapability(capability.AdminReportsManage)`
   - `ReviewsModerate: rbac.RequireCapability(capability.AdminReviewsManage)` (scoped grants must pass — do NOT use `RequireGlobalCapability` here, or scoped school/section admins lose moderation)
   - `ReviewsEditContent: rbac.RequireCapability(capability.AdminReviewsEditContent)` — a NEW capability granted only to `super_admin` and `school_admin` in `roleCapabilities`, so `POST /admin/reviews/:reviewID/edit` keeps its current narrower audience instead of being widened to `section_*`.
2. In `handler.go:168-188` replace `requireModerationRole()` with `h.adminAuthorizers.ReportsManage` on the two `/reports` routes and `h.adminAuthorizers.ReviewsModerate` on `/reviews`, `/reviews/:reviewID`, `/reviews/batch`, `/content-flags*`; replace `requireSchoolAdminRole()` with `h.adminAuthorizers.ReviewsEditContent`. Delete `requireModerationRole`/`requireSchoolAdminRole`/`hasAnyModerationRole`/`hasAnyContentEditRole`. Keep the per-resource OpenFGA checks untouched.
3. Rebuild `moderationScope` from `middleware.GetCapabilityGrants(c)` — take the grant for the capability the route requires and read `Global`, `ScopeSchoolIDs`, `ScopeSectionIDs` — instead of `middleware.GetOrgScopedRoles(c)` role names, so the gate an …

#### 59. Sensitive-word admin mutations never invalidate the moderation filter, and Filter.Refresh is dead code, so new block rules take up to 5 minutes to apply

`server/internal/modules/course/review/handler_sensitive_word_admin.go:60`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | cache-invalidation |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
handler_sensitive_word_admin.go:60-74 (create), :103-114 (update), :125-136 (delete) all mutate the rule set and then only log — no filter refresh and no cache invalidation:
	w, err := h.service.CreateSensitiveWord(c.Request.Context(), req.Word, req.Category, req.Level)
	...
	h.logAdminOp(c, "create_sensitive_word", "sensitive_word", w.ID, ...)
	response.Created(c, w)

filter.go:52-55 fixes a purely time-based TTL:
	return &Filter{ repo: repo, refreshTTL: 5 * time.Minute }
and filter.go:110-118 only reloads when `time.Since(f.lastRefresh) > f.refreshTTL`. The exported escape hatch is unused: `grep -rn "filter.Refresh|\.Refresh(ctx)"` (excluding tests) returns nothing, i.e. Filter.Refresh (filter.go:72) has no production caller. Contrast the module's established invalidation pattern for every other admin mutation: handler.go:266-277 invalidateCachePrefixes / cache.InvalidateByVersion.
```

**失败场景**

A moderator responding to a live brigading incident adds a `block`-level sensitive word via POST /api/v1/course/review/admin/sensitive-words. For up to 5 minutes each API replica keeps serving from its in-process matcher set, so PostReview and CreateReply continue to accept and publish content containing the newly blocked term. The same window applies in reverse: deactivating a false-positive word keeps rejecting legitimate reviews for 5 minutes. There is no mechanism at all to shorten it — no version key, no pub/sub, no call to Refresh.

**修复方案**

Do it in two layers, keeping invalidation in the Service (which owns `filter`), not the Handler.

1. Local (serving-replica) consistency — the high-value, low-risk half:
   - server/internal/modules/course/review/filter.go: add `func (f *Filter) Invalidate()` next to `Refresh` (filter.go:72) that takes `f.mu.Lock()` and sets `f.lastRefresh = time.Time{}`, so the next `ensureFresh` (filter.go:110) reloads. Cheap, no I/O, cannot fail, and cannot make a request error out.
   - server/internal/modules/course/review/service.go: call `s.filter.Invalidate()` after the successful repo call in `CreateSensitiveWord` (after line 693), `UpdateSensitiveWord` (after line 723) and `DeleteSensitiveWord` (after line 730). Handlers at handler_sensitive_word_admin.go:60/103/125 need no change.

2. Cross-replica consistency:
   - Give `Filter` an optional version source. Add a `NewFilterWithVersion(repo, *cache.Helper)` (or a functional option on `NewService`, wired from server/internal/app/modules_course_metrics.go:40 where the Redis client / `cache.Helper` is already available). On mutation, bump `review:sensitive_words` via the existing `cache.Helper.InvalidateByVersion` used at handler.go:230/239/270; in `ensureFresh`, read that version and force a reload when it differs from the version captured at last load.
   - Guard the hot path: memoize the version probe for a few seconds (e.g. 2-5s) so a burst of posts is not one Redis GET each, and on any Redis error log-and-fall-back to the existing TTL path — content posting must never fail because Redis is down. Do not let this new dependency wi …

#### 60. Review write/moderation errors return generic error codes, so users see the wrong localized message

`server/internal/modules/course/review/http_errors.go:17`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | correctness |
| 验证票数 | 1/2 |
| 严重度 | 原报 P1 → 验证方修正为 P2 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
reviewModerationErrorMappings = []response.ErrorMapping{
	response.MatchError(ErrTitleEmpty, 400, "title cannot be empty"),
	response.MatchError(ErrTitleTooLong, 400, "title is too long", errs.ErrParamOutOfRange),
	response.MatchError(ErrDangerousContent, 400, "content contains potentially dangerous elements"),   // <- no code arg
	response.MatchError(ErrSensitiveContent, 400, "content contains sensitive words", errs.ErrSensitiveContent),
	...
	response.MatchError(ErrContentTooShort, 400, "content is too short", errs.ErrParamOutOfRange),
	response.MatchError(ErrContentTooLong, 400, "content is too long", errs.ErrParamOutOfRange),
}
reviewWriteValidationErrorMappings = []response.ErrorMapping{
	...
	response.MatchError(ErrRatingRequired, 400, "at least one rating dimension is required"),   // <- no code arg
	response.MatchError(ErrInvalidRating, 400, "rating must be between 1 and 5"),               // <- no code arg
}

// response/mapped_error.go:19 -> omitted code falls back to defaultErrorCodeForStatus(400) = errs.ErrBadRequest (A0000400)
// errs/codes.go:182-183,193,203-204 define …
```

**失败场景**

User posts a review at POST /api/v1/reviews with a 5-character body. review/service.go:296 returns ErrContentTooShort; respondPostReviewError -> reviewModerationErrorMappings maps it to HTTP 400 with code errs.ErrParamOutOfRange = "A0000403". PostReviewPage.vue:1062 calls getErrorMessage(error, ...), which looks up errors.A0000403 and renders "参数超出范围" instead of the existing string errors.A0110003 = "测评内容过短". Same for over-long content, dangerous content (returns A0000400 "请求参数错误" instead of A0110300 "内容包含危险元素"), missing rating dimension and out-of-range rating. The correct localized strings are already in both zh-CN and en-US bundles and are unreachable.

**修复方案**

Apply the narrow, non-breaking subset only.

1. server/internal/modules/course/review/http_errors.go — add explicit codes ONLY where the sentinel is unambiguous:
   - line 17: `response.MatchError(ErrDangerousContent, 400, "content contains potentially dangerous elements", errs.ErrDangerousContent)` (field-agnostic message, safe for review/draft/reply/report).
   - line 21: `ErrContentTooShort` -> `errs.ErrReviewContentTooShort` (only produced by validateReviewTextLengths, service.go:296 — review create + admin edit).
   - line 56: `ErrRatingRequired` -> `errs.ErrRatingDimensionMissing`.
   - line 57: `ErrInvalidRating` -> `errs.ErrRatingInvalid` (both only produced by validateRatingValues, service.go:322/338/341/344).
   - DO NOT change line 22 (`ErrContentTooLong`) in the shared table. Either leave it at `errs.ErrParamOutOfRange`, or do it properly: add distinct sentinels `ErrReplyContentTooLong` (service_interaction.go:405) and `ErrReportDescriptionTooLong` (service_report.go:118), keep those on `errs.ErrParamOutOfRange`, and map the review/draft-body `ErrContentTooLong` to `errs.ErrReviewContentTooLong` in a review-only mapping group consumed by respondPostReviewError / respondUpdateReviewError / respondSaveDraftError / respondAdminEditReviewError but NOT by respondCreateReplyError / respondReportReviewError.

2. server/internal/modules/course/review/http_errors_test.go:26 — MUST update the pinned expectation from `code: errs.ErrBadRequest` to `code: errs.ErrRatingInvalid`, otherwise the package tests fail. Add cases: ErrContentTooShort -> A0110003, ErrDangerousContent …

#### 61. REFRESH MATERIALIZED VIEW CONCURRENTLY runs under the global 5s DB_QUERY_TIMEOUT, so the 000018 projection can permanently stop updating

`server/internal/modules/course/review/repository_teacher_public.go:106`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
func (r *Repository) RefreshTeacherPublicStats(ctx context.Context) error {
	ctx = withDBTable(ctx, "mv_teacher_public_stats")
	_, err := r.db.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY mv_teacher_public_stats`)
// db.Exec (db.go:267) does: ctx, cancel := d.withTimeout(ctx)  == WithTimeout(ctx, DB_QUERY_TIMEOUT)
// DB_QUERY_TIMEOUT defaults to 5s (config.go:374) and validation.go:50 caps it at 60s
```

**失败场景**

mv_teacher_public_stats aggregates teachers LEFT JOIN departments LEFT JOIN reviews (000001_initial_schema.up.sql:973). Once that CONCURRENTLY refresh (which builds a full temp copy and diffs it) takes longer than DB_QUERY_TIMEOUT, every teacher_public_stats_refresh job claimed by runTeacherPublicStatsRefreshWorker fails with context deadline exceeded. ListPublicTeachers and ListHotTeachers read only the materialized view, so the public teacher list permanently serves stale avg_rating / review_count and omits newly created teachers, with no config escape hatch (raising DB_QUERY_TIMEOUT raises it for every query and is capped at 60s). It also amplifies: markRetrySQL (pkg/outbox/repository.go:137) resets attempt_count to 0 whenever locked_revision != revision, and the 000018 triggers bump revision on every reviews write, so the failing job never reaches dead_letter and the 2s-poll worker retries whole-view rebuilds indefinitely.

**修复方案**

Give the refresh its own budget, keep it cancellable, and stop the hot retry loop locally.

1. server/internal/pkg/db/db.go — add a long-operation entry point next to Exec (db.go:267), e.g.
   `func (d *DB) ExecWithTimeout(ctx context.Context, timeout time.Duration, sql string, args ...any) (pgconn.CommandTag, error)`
   that does `ctx, cancel := context.WithTimeout(ctxutil.Normalize(ctx), timeout)` (fall back to d.timeout when timeout <= 0) and reuses the exact same span + ObserveDBQueryDuration/ObserveDBQueryTotal path as Exec, with no automatic retry. Do NOT use ctxutil.DetachedTimeout — parent cancellation must still abort the statement so graceful shutdown works. Add a unit test asserting the explicit timeout wins over d.timeout and that parent cancel still propagates.

2. server/internal/pkg/config/config.go + validation.go — add `REVIEW_TEACHER_STATS_REFRESH_TIMEOUT` (seconds, default 60, validated e.g. 5..600, env-only per project law), and factor it into the shutdown budget in server/internal/app/server.go:52 (or document that the worker ctx cancel aborts it) so a mid-flight refresh cannot outlive shutdown.

3. Plumb the value into review.NewRepository (server/internal/app/modules_course*.go wiring) and change repository_teacher_public.go:104-110 to
   `_, err := r.db.ExecWithTimeout(ctx, r.statsRefreshTimeout, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_teacher_public_stats")`.
   SQL stays in the Repository; the timeout is config, not a hardcoded constant.

4. Add a cadence floor so a slow/failing refresh cannot spin every 2s: in server/internal/modules/course/rev …

#### 62. Nullable courses columns are scanned into non-nullable Go fields; one NULL department_id/code/credits 500s every course endpoint

`server/internal/modules/course/repository.go:175`

| | |
|---|---|
| 区域 | 课程 |
| 类别 | schema-mismatch |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count
FROM courses c
LEFT JOIN departments d ON d.id = c.department_id
WHERE c.id = $1
// scanned into model.go:19 DepartmentID int64, model.go:21 Code string, model.go:23 Credits float64
// live schema: department_id bigint NULL, code character varying(50) NULL, credits numeric(4,1) NULL
```

**失败场景**

Reproduced against the live dev database (migrations at version 19): inserting one row with INSERT INTO courses (school_id, name, code, department_id, credits, category) VALUES ($1,'NULL-AUDIT',NULL,NULL,NULL,'x') and running the exact GetCourseByID query with the exact Scan destinations returns "can't scan into dest[2] (col: department_id): cannot scan NULL into *int64". Because no Go code inserts into courses (catalog rows come from operator SQL import), a single imported course lacking a department, course code, or credit value makes GET /courses/:id, GET /courses (line 131), GET /courses/search (line 154) and ListCoursesGroupedByDepartment (line 284) all return 500 — the list endpoints break for every user, not just that one course. The LEFT JOIN d.name into DepartmentName string fails the same way, while sibling repositories in the same module do COALESCE it (course/review/repository_interaction.go:90 uses COALESCE(c.name, '')).

**修复方案**

Normalize NULLs in SQL (do NOT switch to pointer types — that breaks the generated TS contract).

1. `server/internal/modules/course/repository.go` — in all four SELECTs (lines 131, 154, 175, 284) replace the projection
   `c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count`
   with
   `c.id, c.school_id, COALESCE(c.department_id, 0), COALESCE(d.name, ''), COALESCE(c.code, ''), c.name, COALESCE(c.credits, 0), c.category, c.review_count`.
   Leave the WHERE/ORDER BY clauses untouched (they already handle 0 as the "no department filter" sentinel at lines 96, 136).

2. `server/internal/modules/course/review/repository_interaction.go:267` (`ListFavorites`) — same defect, missed by the finding. Change
   `SELECT c.id, c.name, c.code, c.credits, c.department_id, d.name, ...`
   to
   `SELECT c.id, c.name, c.code, COALESCE(c.credits, 0), COALESCE(c.department_id, 0), d.name, ...`.
   `Code`/`DepartmentName` are already `*string` in `review/model.go:144,147` so they need no COALESCE.

3. `server/internal/modules/course/model.go:23` — drop `omitempty` from `Credits float64 \`json:"credits,omitempty"\`` → `json:"credits"`. Required because `server/api/components/schemas/course.yaml:3` marks `credits` as **required**; COALESCEing NULL→0 with `omitempty` still present would silently drop a required field and produce a response that violates the OpenAPI contract (`api.gen.ts:3608` types it `credits: number`). `DepartmentID` already lacks `omitempty` — correct as-is. Keep `omitempty` on `Code`/`DepartmentName`; both are optional in the schem …


### P3（20 项）

#### 63. AdminContentLayout silently drops the `description` prop two pages pass to it

`clients/admin/apps/web-ele/src/views/shared/AdminContentLayout.vue:3`

| | |
|---|---|
| 区域 | Admin 前端 |
| 类别 | dead-prop |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
AdminContentLayout.vue:2-5 → `defineProps<{ title: string; total?: number; }>();` — no `description` prop and no description slot anywhere in the template (lines 8-37).
views/users/admission-policy/index.vue:312 → `description="控制目标群入群处理方式、入群后等待时长和学生认证审核行为。"`
views/dashboard/workspace/index.vue:162 → `:description="$t('admin.dashboard.summary.title')"`
```

**失败场景**

Opening /users/admission-policy shows only the title "入群认证策略" — the explanatory sentence the author wrote never renders. Because the component does not declare the prop and does not set `inheritAttrs: false`, the value instead lands on the root `<section>` as a non-standard `description="控制目标群…"` HTML attribute, so the guidance text is present in the DOM but invisible to users and meaningless to assistive tech. Same on /workspace.

**修复方案**

1) clients/admin/apps/web-ele/src/views/shared/AdminContentLayout.vue — add the prop and render it as a real subtitle:
- Extend defineProps (lines 2-5) to `defineProps<{ title: string; description?: string; total?: number; }>();`.
- Do NOT drop the <p> straight into .admin-content-page__heading: that div is `display: flex; align-items: center` (lines 63-68) and would place the description inline next to the <h1> and total badge. Instead wrap the existing `<h1>` + total `<span>` (lines 12-18) in an inner row element (e.g. `<div class="admin-content-page__heading-row">`), make `.admin-content-page__heading` `flex-direction: column; align-items: flex-start;`, move `gap: 10px; align-items: center;` onto the new row class, and add `<p v-if="description" class="admin-content-page__description">{{ description }}</p>` after the row.
- Add CSS: `.admin-content-page__description { margin: 4px 0 0; font-size: 13px; line-height: 1.5; color: var(--el-text-color-secondary); }` and change `.admin-content-page__header`'s `align-items: center` (line 57) to `flex-start` so the #actions buttons stay top-aligned once the header is two lines tall.
- Optionally also expose a `#description` slot for rich copy, mirroring the existing $slots.actions pattern.

2) clients/admin/apps/web-ele/src/views/dashboard/workspace/index.vue:162 — do not just start rendering the current value. `:description="$t('admin.dashboard.summary.title')"` resolves to "统计概览", a section heading (it is used as an <h2>/<p> label in dashboard/analytics/index.vue:187 and :245), so as a page subtitle under "工作台" it reads wrong. …

#### 64. PersistentAdminTableColumn clobbers a caller-supplied min-width with the undefined defaultMinWidth

`clients/admin/apps/web-ele/src/views/shared/admin-table/PersistentAdminTableColumn.vue:64`

| | |
|---|---|
| 区域 | Admin 前端 |
| 类别 | component-contract |
| 验证票数 | 1/1 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
PersistentAdminTableColumn.vue:61-67 →
```
<ElTableColumn
  v-bind="$attrs"
  :column-key="columnKey"
  :min-width="defaultMinWidth"
  resizable
  :width="width"
>
```
`:min-width` is merged AFTER `v-bind="$attrs"`, so the later (undefined) value wins.
views/open-platform/consents/index.vue:223,234,246,257,273,284,296 → columns declare raw `min-width="120"` … `min-width="280"` instead of `:default-min-width`.
```

**失败场景**

On /open-platform/consents every column passes `min-width="…"` through `$attrs`, but the component overwrites it with `defaultMinWidth === undefined`, so ElTableColumn falls back to its 80px floor. The 用户ID / 应用 / 授权范围 / 授权时间 / 最近使用 / 操作 columns collapse well below their intended widths on narrow viewports, truncating client IDs and scope tag lists that the page was laid out to show.

**修复方案**

Apply both halves; either alone leaves a trap.

1. clients/admin/apps/web-ele/src/views/shared/admin-table/PersistentAdminTableColumn.vue (lines 61-67) - move the min-width binding ABOVE the attrs spread so an explicitly passed attr overrides the default instead of being erased:

<ElTableColumn
  :min-width="defaultMinWidth"
  v-bind="$attrs"
  :column-key="columnKey"
  resizable
  :width="width"
>

`:column-key`, `resizable` and `:width` must stay AFTER `v-bind="$attrs"` - the persisted width from `table.columnWidth(columnKey)` has to win over any caller attr, and the column key is the persistence identity. Only `min-width` moves. `vue/attributes-order` is off, so this ordering will not fail lint. (The alternative, `:min-width="defaultMinWidth ?? ($attrs['min-width'] as number | string | undefined)"` via `useAttrs()`, works too but is more code for the same behavior.)

2. clients/admin/apps/web-ele/src/views/open-platform/consents/index.vue lines 223, 234, 246, 257, 275, 284, 296 - replace `min-width="120"` / `"220"` / `"110"` / `"280"` / `"170"` / `"170"` / `"260"` with `:default-min-width="120"` etc., matching the convention already used by the other 14 admin table pages. This is what actually restores the intended layout today; parseMinWidth accepts both the number and the string form.

3. clients/admin/apps/web-ele/src/views/shared/admin-table/PersistentAdminTable.test.ts - add `minWidth: { type: [Number, String], default: undefined }` to ElTableColumnStub, emit it as `data-min-width`, and add a regression test asserting that a column declared with `min-width="120"` (a …

#### 65. Freshman verification shows enabled 通过/带天数通过/驳回 buttons on already-reviewed rows; every click 409s

`clients/admin/apps/web-ele/src/views/users/freshman-verification/index.vue:310`

| | |
|---|---|
| 区域 | Admin 前端 |
| 类别 | dead-control |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
index.vue:299-357 — the action cell is unconditional: `<div class="freshman-action-group">` with `<ElButton data-action="approve" :disabled="rowReviewing(row)" @click="approve(row)">`, `data-action="approveWithDays"`, and `data-action="reject"`. The only disable condition is `rowReviewing(row)` (an in-flight request), never `row.status`.
index.vue:45-49 — the toolbar filter offers `pending | approved | rejected`, so reviewed rows are routinely on screen.
server/internal/modules/admission/service_operator.go:104-107 → `if app.Status != FreshmanApplicationPending { return ErrAdmissionInvalidStatus }`
server/internal/modules/admission/handler_errors.go:55-56 → `case errors.Is(err, ErrAdmissionInvalidStatus): response.Conflict(c, "admission session status invalid")`
Compare identity-review/index.vue:429 (`v-if="isPending(row)"`) and student-verification/index.vue:350 (`v-if="row.verificationStatus === 'pending'"`), which do gate on status.
```

**失败场景**

An operator switches the status filter to 已通过 and clicks 通过 (or 驳回, after typing a reason) on any listed row. `PUT /api/v1/admin/freshman-verifications/{id}` reaches `reviewFreshmanApplication`, which sees `app.Status == approved`, aborts the transaction and returns HTTP 409 with the raw English body `admission session status invalid`. The admin sees a red toast plus a persistent `.admin-load-error` alert containing that untranslated string, with no indication that the action was never valid for that row.

**修复方案**

In /home/wztxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/users/freshman-verification/index.vue (action cell, lines 299-358):
1. Keep the 材料预览 ElButton rendered for every row (read-only evidence viewing stays legitimate).
2. Wrap only the mutation controls - approve ElButton, the extension-days ElInputNumber, approveWithDays ElButton, the reason ElInput, and the reject ElButton - in a container with v-if="row.status === 'pending'", and render <span v-else class="admin-cell-muted">—</span>, mirroring identity-review/index.vue:429 and student-verification/index.vue:350. Reduce the column :default-width accordingly is optional; leaving 420 is fine.
3. Add defensive early returns in the script so a row that went stale between render and click cannot fire a doomed request: at the top of approve() and reject() (or inside handleReview(), alongside the existing rowReviewing check at line 120) add `if (row.status !== 'pending') return false;`. This mirrors identity-review/index.vue:136 and :187.
4. Extend tests/e2e/admin-user-actions.spec.ts with an approved-status fixture (e.g. { ...freshmanApplication, id: 'freshman-action-3', status: 'approved' }) and assert row.locator('[data-action="approve"]') has count 0 while [data-material-preview] is visible. All current fixtures are status 'pending', so existing e2e and index.test.ts assertions keep passing.

Do NOT apply the shared-result.ts part of the original proposal: ErrAdmissionInvalidStatus is emitted by response.Conflict without an explicit code (server/internal/modules/admission/handler_errors.go:55-56), so it carries t …

#### 66. ES256 is whitelisted by the local alg pre-check but the verifier only ever accepts RS256, and discovery-advertised algs are ignored

`server/internal/pkg/oidc/jwt_alg.go:35`

| | |
|---|---|
| 区域 | OIDC |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
jwt_alg.go:33-40 — `switch algorithm { case "RS256", "ES256": return true ... }`, used as the gate in verify.go:32-34. But verifier.go:42 builds the verifier with `gooidc.NewVerifier(cfg.Issuer, keySet, &gooidc.Config{SkipClientIDCheck: true})` — `SupportedSigningAlgs` is left empty and `NewVerifier` (unlike `provider.VerifierContext`, go-oidc v3.17.0 verify.go:134-138) does not merge the provider's `id_token_signing_alg_values_supported`. go-oidc verify.go:206-213: "If no algorithms were specified by both the config and discovery, default to the one mandatory algorithm RS256", so `jose.ParseSigned` is given `[RS256]` only.
```

**失败场景**

An operator configures the Casdoor application certificate as ECDSA (Casdoor supports ES256/ES384/ES512). Every ID token then carries `alg: ES256`; `validateJWTSigningAlgorithm` happily accepts it, then `c.verifier.Verify` fails with "oidc: malformed jwt: unexpected signature algorithm". `verifyIDToken` returns a non-`ErrProviderUnavailable` error, so `handleWebCallback` responds 500 "authentication failed" on every login and `resolveCookieToken` 401s every request — a total auth outage with a misleading error, while the code's own allow-list advertises ES256 as supported.

**修复方案**

In `server/internal/pkg/oidc/verifier.go:42`, make the verifier's accepted algorithm set explicit and identical to the local allow-list, instead of relying on go-oidc's implicit RS256-only default:

```go
return gooidc.NewVerifier(cfg.Issuer, keySet, &gooidc.Config{
    SkipClientIDCheck:    true,
    SupportedSigningAlgs: allowedJWTSigningAlgorithms(), // []string{"RS256", "ES256"}
}), nil
```

and in `jwt_alg.go` expose the list once so the two cannot drift again:

```go
func allowedJWTSigningAlgorithms() []string { return []string{"RS256", "ES256"} }

func isAllowedJWTSigningAlgorithm(algorithm string) bool {
    return slices.Contains(allowedJWTSigningAlgorithms(), algorithm)
}
```

Keep `validateJWTSigningAlgorithm` as the pre-JWKS-fetch gate (it still gives the clear `errDisallowedJWTAlgorithm` error and avoids a JWKS round trip for HS*/none). Do NOT seed `SupportedSigningAlgs` from discovery via `provider.VerifierContext` without intersecting with the local list — a compromised/misconfigured discovery document could otherwise widen the accepted set, which is exactly what the current hardcoded list is defending against.

Add a unit/integration test mirroring `client_integration_test.go`'s RS256 fixture but signing with an ECDSA P-256 key and `jose.ES256`, asserting `VerifyIDToken` succeeds. That test fails today and passes after the change.

If the team instead decides RS256-only is the intended posture, the equivalent-cost fix is to drop `"ES256"` from `isAllowedJWTSigningAlgorithm` and correct `docs/design/iam-architecture.md:877-878` (which currently states 默认 RS25 …

#### 67. School SSO admission endpoints are declared `security: []` (public) but are registered behind authMW and return an undeclared 401

`server/api/paths/admission.yaml:463`

| | |
|---|---|
| 区域 | OpenAPI 契约 |
| 类别 | contract-mismatch |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
Spec (server/api/paths/admission.yaml:458-485) for `/api/v1/admission/school-sso/{schoolCode}/login`:
    operationId: startAdmissionSchoolSSO
    security: []
    responses:
      '302': ...
      '400': ...      # no 401
Same `security: []` at line 492 for `/callback`.

Code (server/internal/modules/admission/handler.go:84-85):
	admission.GET("/school-sso/:schoolCode/login", authMW, h.handleStartSchoolSSO)
	admission.GET("/school-sso/:schoolCode/callback", authMW, h.handleCompleteSchoolSSO)
Both handlers then call `h.resolveAdmissionUserAndSchool(c)` -> `middleware.ResolveRequiredInternalUserID`, which does `response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)` (server/internal/pkg/middleware/internal_user.go:27-30).
```

**失败场景**

A user whose StuHelper session expired while at the school IdP is redirected back to `GET /api/v1/admission/school-sso/4111010006/callback?code=...&state=...`; the auth middleware returns 401 JSON. For `/login` the same 401 is possible but is not in the declared response set at all, so generated clients/error handling have no branch for it. Worse for integration: the contract states these two operations need no credentials, so any SDK, gateway or API-doc consumer generated from the spec will call them anonymously and always get 401 instead of the documented 302.

**修复方案**

Spec-only change, no Go/TS hand-edits. In server/api/paths/admission.yaml: (1) delete `security: []` at line 463 (startAdmissionSchoolSSO) and line 492 (completeAdmissionSchoolSSO) so both inherit the global `cookieAuth`/`bearerAuth` requirement (equivalently write the explicit two-item list, matching the style used by every other authenticated operation in the file). (2) Extend startAdmissionSchoolSSO's responses to match what respondAdmissionError/authMW actually return - add '401', '403', '404', '409' and '503' all as `$ref: '../components/responses/common.yaml#/ErrorResponse'` (401 = authMW/no subject; 403 = unprovisioned user; 404 = school not found; 409 = ErrAdmissionLinkedSessionRequired; 503 = ErrAdmissionSSONotConfigured / ErrAdmissionRedisUnavailable); for completeAdmissionSchoolSSO add the missing '403', '404' and '409' (401 and 503 are already declared). (3) Same pass, fix the inverse drift: add `security: []` to the three operations `GET /api/v1/admission/freshman/mobile-camera-handoffs/{token}`, `POST .../camera-capture` and `POST .../continue`, which server/internal/modules/admission/handler.go:78-80 registers with no authMW and are therefore anonymous, token-scoped endpoints. Then run `cd /home/wztxy/Code/StuHelper/server && make bundle-spec generate` (targets exist at server/Makefile:100 and :112) and commit the regenerated server/api/openapi.bundled.yaml, server/internal/api/gen/server.gen.go (embedded spec) and clients/shared/src/types/api.gen.ts so `make check-bundled-drift`/`check-drift-*` stay green. Runtime behavior is unchanged because the OpenAPI re …

#### 68. ReadTuples drops the OpenFGA continuation token, so WriteMissingTuples re-writes already-present tuples and the write fails

`server/internal/pkg/fga/client.go:231`

| | |
|---|---|
| 区域 | OpenFGA |
| 类别 | pagination-correctness |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
body := client.ClientReadRequest{
		Object:   openfga.PtrString(object),
		Relation: openfga.PtrString(relation),
	}
	resp, err := c.fga.Read(ctx).Body(body).Execute()
...
	result := make([]Tuple, 0, len(resp.Tuples))
	for _, tuple := range resp.Tuples { ... }
	return result, nil
```

**失败场景**

No PageSize is set and resp.ContinuationToken is never followed, so ReadTuples returns only the OpenFGA server's first page (default 50). WriteMissingTuples (relation_writer.go:46-52) computes `missing` against that truncated set, so any tuple on page 2+ is classified as missing and passed to WriteTuples. OpenFGA's Write API rejects a tuple that already exists with HTTP 400 write_failed_due_to_invalid_input, so the whole batch fails. Concrete: once the platform has more than ~50 super_admins, ReadTuples("ecosystem:stuhelper", "super_admin") truncates, and every subsequent super_admin login fails UpsertUser -> syncGlobalRoleRelations with a write error, blocking login user-sync. Same defect hits openplatform writeResourceTuplesForRollback for a user_profile with >50 app read grants.

**修复方案**

Two changes in server/internal/pkg/fga:

1. client.go — make ReadTuples exhaustive. Set an explicit page size and loop on the continuation token:

    const readTuplesPageSize = 100
    var token string
    result := make([]Tuple, 0, readTuplesPageSize)
    for {
        opts := client.ClientReadOptions{PageSize: openfga.PtrInt32(readTuplesPageSize)}
        if token != "" { opts.ContinuationToken = openfga.PtrString(token) }
        resp, err := c.fga.Read(ctx).Body(body).Options(opts).Execute()
        ... append resp.Tuples ...
        token = resp.GetContinuationToken()
        if token == "" { break }
    }

   Keep the metrics/span observation once per HTTP call (or observe the loop as a whole) and add a page cap (e.g. 100 pages) that returns an error rather than spinning forever if the server keeps handing back tokens.

2. Add a user-filtered existence read and use it for presence checks so the hot path does not enumerate a whole relation:

    func (c *Client) hasTuple(ctx context.Context, t Tuple) (bool, error) {
        // Read with User+Relation+Object set: returns 0 or 1 tuple, no paging concern.
    }

   In relation_writer.go WriteMissingTuples, use hasTuple per desired tuple instead of ReadTuples(group.object, group.relation) + MissingTuples. That fixes the ecosystem:stuhelper#super_admin path (the only group that grows with data) independently of paging and removes the per-login read of up to a page of unrelated tuples.

Add a unit test against the existing httptest-based fake (client_http_test.go) that serves two pages with a continuation_token and asserts …

#### 69. Native SSO callback discards the redirect the user was sent to login from and always reLaunches to the profile page

`clients/uniappx/src/pages/auth/callback.vue:40`

| | |
|---|---|
| 区域 | UniAppX |
| 类别 | navigation |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
callback.vue after a successful exchange:
  setTimeout(() => { uni.reLaunch({ url: '/pages/user/index' }) }, 500)
The redirect is computed and threaded all the way to the login page but never comes back: stores/auth.ts:208 `uni.navigateTo({ url: `/pages/auth/login?redirect=${encodeURIComponent(buildCurrentRouteRedirect())}` })`; login.vue:38 `redirect.value = normalizeRedirectOption(options?.redirect)`; login.vue:48 `api.auth.login(redirectPath, platform, 'uniapp')`. On the native path the deep link only carries code+state (App.vue:57-70), and callback.vue never reads or restores `redirect`. reLaunch also destroys the page stack, so the originating page is gone.
```

**失败场景**

On the App build a logged-out user browsing /pages/course/detail?id=42 taps 写评课. requireAuth pushes /pages/auth/login?redirect=%2Fpages%2Fcourse%2Fdetail%3Fid%3D42, SSO opens in the system browser, the stuhelper:// deep link returns to the callback page, and the user is reLaunched to 个人中心 with an empty stack. They land on an unrelated page and must search for course 42 again to write the review they started.

**修复方案**

Persist the redirect in its own storage slot next to the SSO state and consume it in the callback (do not reuse SSO_STATE_STORAGE_KEY — its normalizeState regex `/^[A-Za-z0-9_-]+$/` rejects paths).

1. In clients/uniappx/src/auth/sso-state.ts add `export const SSO_REDIRECT_STORAGE_KEY = 'stuhelper:sso-redirect'` plus `persistSSORedirect(path)` / `readStoredSSORedirect(): string | null` / `clearStoredSSORedirect()`, and move login.vue's `normalizeRedirectOption` here (export it) so both pages share one validator (`/pages/` prefix, reject `//`, bounded length, double-decode guard). Storage writes must not be fatal for redirect (wrap in try/catch and ignore) — losing the redirect must never block login.
2. clients/uniappx/src/pages/auth/login.vue: in the `isNativeApp` branch, after `persistSSOState(data.state)`, call `persistSSORedirect(redirectPath)`.
3. clients/uniappx/src/pages/auth/callback.vue: after `await authStore.exchangeNativeCode(code, state)`, resolve `const target = normalizeRedirectOption(readStoredSSORedirect())` (defaults to `/pages/user/index`), clear the slot, and `uni.reLaunch({ url: target })`. Also clear the slot on the error path and in `handleRetry` so a stale redirect from an abandoned attempt cannot leak into a later login.

Using storage (rather than `uni.navigateBack({ delta: 2 })`) is the right choice because the deep link may cold-start the app with an empty page stack; keep reLaunch so tab-bar and non-tab-bar targets both work. Add a vitest case asserting callback reLaunches to the stored path and falls back to /pages/user/index for a missing/host …

#### 70. Edit/delete-own-review controls are dead in the whole course/review module because the backend Review payload carries no ownership field

`clients/web/src/components/business/review/ReviewCard.vue:186`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | ownership-gating |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
Ownership is driven purely by a caller-supplied boolean:
```html
<button v-if="!props.isOwnReview" ... @click="toggleReportMenu">      <!-- report -->
<button v-if="props.isOwnReview && !editing" ... @click="startEditing">  <!-- edit -->
<button v-if="props.isOwnReview" ... @click="handleDeleteOwn">          <!-- delete -->
```
`isOwnReview?: boolean` (line 340) is passed by exactly one call site in the repo: `modules/user/views/MyReviewsTab.vue:28 :is-own-review="true"` (hardcoded). SearchPage.vue:332, ReviewFeed.vue:39 and TeacherProfilePage.vue:179 all render `<ReviewCard :review="review">` with no such prop, and they cannot compute it: the OpenAPI `Review` schema (server/api/components/schemas/review.yaml:14-80) has no `isOwner`/`userHash` field — unlike `Reply`, which does expose `isOwner` (replyPageResponse.ts:82).
```

**失败场景**

A verified student opens the course page for a review they wrote themselves (or finds it via /search). They see a Report (flag) button on their own review and no edit or delete control, so the only way to fix a typo is to hunt for the review in the user center. Conversely `MyReviewsTab` marks every row own, so if that list ever returns another user's review the edit/delete buttons would be shown for it.

**修复方案**

1. server/api/components/schemas/review.yaml: add `isOwner: {type: boolean}` to the Review schema (optional, not in `required`, mirroring Reply at line 174), then re-bundle and regenerate Go + TS types (never hand-edit gen/ or api.gen.ts).
2. server/internal/modules/course/review/model.go: keep `UserHash` as json:"-" and add `IsOwner bool `json:"isOwner"``; populate it in the existing per-user enrichment step alongside populateUserVotes (service.go:403/446/490/525) as `r.IsOwner = params.UserHash != "" && r.UserHash == params.UserHash` (false for anonymous). Populate before stripReviewsForResponse so preview stripping does not drop it. Do not add it to any respondWithCachedData path.
3. clients/web/src/components/business/review/ReviewCard.vue: replace the three raw `props.isOwnReview` checks with `const isOwn = computed(() => props.isOwnReview ?? props.review.isOwner === true)` so MyReviewsTab keeps working unchanged while search/hub/teacher-profile become correct; keep the prop as an override only.
4. server/internal/modules/course/review/service_report.go ReportReview: after the existence check, reject when reporter hash == review author hash (new ErrCannotReportOwnReview -> 400) so the backend closes self-reporting regardless of UI state.
5. Add a Vue test asserting report is hidden and edit/delete shown when `review.isOwner === true` with no prop passed, plus a Go test for the self-report rejection.

#### 71. Inline review edit accepts 1-9 character content that the API rejects with a generic 400

`clients/web/src/components/business/review/useReviewEdit.ts:33`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | validation-mismatch |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
Client-side the only check is non-empty (and the Save button mirrors it: ReviewCard.vue:248 `:disabled="saving || !editContent.trim()"`):
```ts
const trimmed = editContent.value.trim()
if (!trimmed) return
...
await api.review.updateReview(review.id, { content: trimmed, ratings: review.ratings })
```
The server requires 10 chars: `server/internal/modules/course/review/review.go:121` `Content *string \`json:"content" binding:"omitempty,min=10,max=5000"\`` and the contract agrees (`UpdateReviewRequest.content minLength: 10`). On failure the handler answers `response.BadRequest(c, "invalid request parameters")` (review.go:145-147), i.e. no field-level detail. Note the create form in the same module *does* enforce it (PostReviewPage.vue:980 `trimmed.length < CONTENT_MIN`), so the edit path is the outlier.
```

**失败场景**

User edits their review down to "讲得很好" (4 chars), the Save button is enabled, the PUT returns 400 "invalid request parameters", and the toast shows the generic `review.review.editFailed` text. The textarea keeps the edit with no hint about the 10-character minimum, so the user retries and fails repeatedly.

**修复方案**

In clients/web/src/components/business/review/useReviewEdit.ts, import REVIEW_CONTENT_MIN_LENGTH / REVIEW_CONTENT_MAX_LENGTH from @stuhelper/shared/constants and change the guard in handleSaveEdit to reject out-of-range lengths with a specific toast, e.g. `if (trimmed.length < REVIEW_CONTENT_MIN_LENGTH) { toast.error(t2('review.validation.contentTooShort', { min: REVIEW_CONTENT_MIN_LENGTH })); return }` plus the symmetric max check (the `t` param currently has signature `(key: string) => string`, so widen it to accept an interpolation payload, matching how PostReviewPage.vue:981 calls it). Then in ReviewCard.vue: add `:maxlength="REVIEW_CONTENT_MAX_LENGTH"` to the edit textarea, expose a computed `editContentTooShort`, change Save to `:disabled="saving || editContentTooShort"`, and render the existing `review.validation.contentTooShort` message plus a `{{ editContent.trim().length }}/{{ REVIEW_CONTENT_MAX_LENGTH }}` counter below the textarea — the same pattern already used at PostReviewPage.vue:349. No new i18n keys are required.

#### 72. Admission login / signup / re-login buttons have no loading state, so repeated taps rotate the server oidc_state and the callback dead-ends on a raw 400 JSON page

`clients/web/src/modules/admission/views/AdmissionPage.vue:34`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | double-submit |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
<button class="primary-button" type="button" @click="startLogin">登录</button>
<button v-if="!consumedTokenNeedsLogin" class="secondary-button" type="button" @click="startSignup">注册</button>
<button class="primary-button" type="button" @click="startAccountSwitch">重新登录</button>
// no :disabled binding anywhere, although the same store flag is used elsewhere:
// LoginPage.vue:  :disabled="loading"   JoinStartPage.vue: :disabled="auth.loading"
```

**失败场景**

On a slow mobile network (this page is opened from QQ), tapping 登录 gives zero feedback: the label never changes and the button stays active while `api.auth.login` is in flight. A second tap issues a second GET /api/v1/auth/login; each response re-writes the HttpOnly `oidc_state` cookie (handler_login.go:533 setOIDCStateCookie) and sets `window.location.href`. If the navigation started by the first response wins while the second response's Set-Cookie has already landed, Casdoor comes back with state A while the cookie holds state B, `validateOIDCStateCookie` fails and the backend answers /api/v1/auth/callback with the raw JSON 400 "invalid or expired state parameter" — a dead end with no link back to /verify/:code. `startAccountSwitch` is worse: two concurrent logout + upstream SSO logout + login chains.

**修复方案**

In clients/web/src/modules/admission/views/AdmissionPage.vue add a local in-flight ref and guard all three handlers, rather than relying only on the shared `auth.loading` (which drops to false mid-`switchAccount`):

const authRedirecting = ref(false)
const authActionBusy = computed(() => authRedirecting.value || auth.loading)

async function runAuthRedirect(start: (returnURL: string) => Promise<void>) {
  if (authActionBusy.value) return
  authRedirecting.value = true
  try { await start(currentAdmissionURL()) }
  catch { authRedirecting.value = false }   // keep it true on success: page is navigating away
}
function startLogin() { void runAuthRedirect(auth.login) }
function startSignup() { void runAuthRedirect(auth.signup) }
function startAccountSwitch() { void runAuthRedirect(auth.switchAccount) }

Then bind on lines 34, 37-44, 63 and 87: `:disabled="authActionBusy"` plus a busy label, e.g. `{{ authActionBusy ? '正在跳转…' : '登录' }}` (and '正在退出…' / '重新登录' for the account-switch button). AdmissionPage.css:39-40 already styles `:disabled`, so no CSS change is needed. Extend clients/web/src/modules/admission/__tests__/admissionPageStates.test.ts with a case asserting a second click on 登录 in the needsLogin state does not issue a second auth.login call.

Optional, and the durable fix for every caller: make `startLoginFlow`/`startSignupFlow`/`switchAccount` in clients/web/src/stores/auth.ts early-return (or reuse an in-flight promise) while a redirect is already pending, and stop `logout()`'s `finally` from clearing `loading` when it is invoked from `switchAccount`.

#### 73. School-email OTP resend ignores cooldownSeconds, so the resend button re-enables instantly and the retry surfaces raw English server text

`clients/web/src/modules/admission/views/OldStudentVerificationFlow.vue:276`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
const result = await admissionApi.requestSchoolEmailOTP({...})
    email.value = result.email
    successMessage.value = selectedSchoolRequiresAcademicEmail.value
      ? '学号和姓名已匹配，验证码已发送到学号邮箱。'
      : '验证码已发送。'
  } catch (error) { ... errorMessage.value = readErrorMessage(error, '验证码发送失败。') }
// api.ts:400 parses cooldownSeconds, and nothing in the module reads it:
//   cooldownSeconds: readInteger(payload, 'cooldownSeconds', message),
```

**失败场景**

On /verify/:code the user clicks 校验并发送验证码; the request resolves and `requestingOTP` flips back to false, so the button is immediately clickable again with no countdown. A user who does not see the mail within a few seconds clicks again inside the 60s window (server side `admissionEmailOTPCooldown = time.Minute`, service_student.go:27). The backend returns 429 with the message "please wait before requesting a new code" (handler_errors.go:147), and `readErrorMessage` renders that raw English string inside the otherwise all-Chinese join page, with no indication of how long to wait. PhoneBindingPage.vue:159-205 implements exactly the countdown that is missing here.

**修复方案**

Two small edits in clients/web/src/modules/admission/views/OldStudentVerificationFlow.vue:

1. Delete the module-local `readErrorMessage` (lines 473-475) and use the shared i18n-aware helper instead: `import { getErrorMessage } from '@/api/errors'`, then call it at lines 285, 308 and 434. That alone turns the 429 into '请求过于频繁，请稍后重试' via the existing `errors.A0000429` key and fixes every other English backend string this component echoes (e.g. 'admission student record not found'). Consider the same substitution for the copies at AdmissionPage.vue:1004 and FreshmanCameraFlow.vue:555.

2. Add the countdown, mirroring PhoneBindingPage.vue:159-205: `const cooldown = ref(0)` plus a 1s `setInterval` seeded from `result.cooldownSeconds` (fall back to 60 when the value is <= 0) after a successful request; clear the interval in the existing `onBeforeUnmount`; add `|| cooldown.value > 0` to the button's `:disabled`, and show the remaining seconds in the label (e.g. `重新发送（{{ cooldown }}s）`). Extend oldStudentFlow.test.ts to assert the button is disabled right after a successful send and re-enabled after advancing fake timers 60s.

#### 74. QQ binding panel polls the binding endpoint every 3s with no expiry, visibility, or attempt limit — it never stops until the component unmounts

`clients/web/src/modules/user/views/QQBindingPanel.vue:310`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | resource-leak |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
QQBindingPanel.vue:308-313 —
  function startStatusPolling() {
    stopStatusPolling()
    statusPollTimer = setInterval(() => {
      void refreshQQBindingStatus({ silent: true })
    }, QQ_BINDING_STATUS_POLL_INTERVAL_MS)   // = 3000 (line 153)
  }

`stopStatusPolling()` is only ever reached from `onUnmounted`, from the 409 branch of `onCreateCode`, or once `fetchQQBinding()` returns a binding (refreshQQBindingStatus:262-266). The code's `expiresAt` is displayed (template line 122 `formatTime(qqBindingCode.expiresAt)`) but never compared against the clock, and there is no `document.visibilityState` check or maximum attempt count.
```

**失败场景**

A user clicks "生成绑定码", then switches tabs or walks away without messaging the bot. The interval keeps firing `GET /api/v1/identity/qq-binding` every 3 s — 20 authenticated requests/minute, 1200/hour — indefinitely, long after the binding code has expired server-side and can no longer succeed. Leaving the tab open overnight produces ~10k pointless authenticated requests from one idle browser tab, and the request rate multiplies by the number of users who leave the page open.

**修复方案**

In QQBindingPanel.vue, give the poller a deadline and a visibility guard instead of leaving it open-ended:

1. Add module state next to `statusPollTimer`: `let pollDeadline = 0` and `const codeExpired = ref(false)`.
2. In `startStatusPolling()`, compute the deadline from the freshly issued code with a hard fallback ceiling, e.g. `const parsed = Date.parse(qqBindingCode.value?.expiresAt ?? '')` then `pollDeadline = Number.isNaN(parsed) ? Date.now() + 10 * 60 * 1000 : parsed`, and reset `codeExpired.value = false`.
3. In the interval callback, before polling: `if (Date.now() >= pollDeadline) { stopStatusPolling(); codeExpired.value = true; return }` and `if (typeof document !== 'undefined' && document.visibilityState === 'hidden') { return }` (skip the tick, do not kill the timer, so the poll resumes when the tab is refocused).
4. Surface `codeExpired` in the template near line 121-123 (e.g. an amber "绑定码已过期，请重新生成" line, and disable/relabel the copy + manual-refresh controls) so the user is not handed a dead command; clear it in `onCreateCode` on success.
5. Keep the manual `onRefreshStatus` button working regardless of `codeExpired` so a user who already messaged the bot can still confirm.

Optionally also stop the timer when a poll fails with 401 (`getErrorStatus(error) === 401`) instead of swallowing it silently, so an expired session does not keep hammering the endpoint.

#### 75. Login is hard-blocked when sessionStorage is unavailable, for an OAuth state value that no deployed callback ever reads

`clients/web/src/stores/auth.ts:340`

| | |
|---|---|
| 区域 | Web 前端 |
| 类别 | correctness |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
const data = readLoginURLPayload(res.data?.data);
if (!storeOAuthState(data.state)) {
    throw new Error("OAuth state storage unavailable");
}
window.location.href = data.url;
// browserStorage.ts: safeSetStorageItem returns false whenever sessionStorage is missing/throws
// the only consumer is AuthCallbackPage.vue:86 consumeOAuthState(state), but every documented config points
// Casdoor at the backend: .env.prod.example:81 CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
// and handler_login.go:730 returns origin + "/api/v1/auth/callback"
```

**失败场景**

A user opens /verify/:code (or /login) in a browser where sessionStorage writes fail — iOS Safari private/lockdown mode, an embedded QQ/WeChat webview with partitioned or disabled storage, or a full storage quota. `storeOAuthState` returns false, `startLoginFlow` throws before navigating, and the user gets only "登录失败"/toast; every subsequent tap fails identically, so login (and therefore the whole admission flow) is impossible. The check protects nothing in practice: the authoritative one-time state lives in Redis plus the HttpOnly `oidc_state` cookie, and the SPA page that would verify the stored copy is never the configured redirect target.

**修复方案**

In clients/web/src/stores/auth.ts, make the SPA state copy best-effort in both startLoginFlow (:339) and startSignupFlow (:367): replace the `throw` with a non-fatal record, e.g.

    if (!storeOAuthState(data.state)) {
        // 会话存储不可用（webview 关闭 DOM storage / 配额耗尽）时不阻断登录：
        // state 的权威校验在后端 Redis + HttpOnly oidc_state cookie（handler_login.go consumeOIDCState）。
        reportClientWarning?.("auth.oauth_state_storage_unavailable");
    }
    window.location.href = data.url;

Keep the return value observable (telemetry or console.warn in DEV) so the condition stays diagnosable. Then update clients/web/src/stores/__tests__/authAuthorizeFlow.test.ts:246-262 to assert the inverted contract: when storeOAuthState returns false, navigation to data.url still happens and login does not reject.

Leave AuthCallbackPage.vue and consumeOAuthState untouched: they only run if a deployment actually points Casdoor at the SPA /auth/callback, and that path still ends at the backend callback, which enforces the cookie + one-time Redis state. Do not add a "fail closed only when the SPA is the redirect target" branch — the frontend cannot see CASDOOR_REDIRECT_URI, so that conditional would need new config surface for no security gain.

#### 76. Unauthenticated image-upload route contradicts its own OpenAPI security declaration and has no endpoint rate limit

`server/internal/modules/admission/handler.go:79`

| | |
|---|---|
| 区域 | 入群认证 |
| 类别 | contract-mismatch |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
admission/handler.go:78-80 registers three routes with no authMW:
	admission.GET("/freshman/mobile-camera-handoffs/:token", h.handlePreviewFreshmanCameraHandoff)
	admission.POST("/freshman/mobile-camera-handoffs/:token/camera-capture", h.handleUploadFreshmanCameraHandoffCapture)
	admission.POST("/freshman/mobile-camera-handoffs/:token/continue", h.handleChooseFreshmanCameraHandoffContinuation)

But in server/api/openapi.bundled.yaml these three operations declare no `security` key, so they inherit the document-level `security: [{cookieAuth: []}, {bearerAuth: []}]`. Every other genuinely public route in the spec opts out explicitly with `security: []` — including the directly adjacent token route GET /api/v1/admission/sessions/{token} and GET /api/v1/admission/school-sso/{schoolCode}/login. So the three mobile-handoff operations are the only routes in the whole spec that claim to require a credential while the server registers them fully open. The mismatch is not caught at runtime: middleware/openapi_validation.go:35 wires `openapi3filter.NoopAuthenticationFunc`, i.e. the validator ch …
```

**失败场景**

A security review or an automated spec-driven scanner enumerates public endpoints from openapi.bundled.yaml, sees these three operations as authenticated, and never tests them anonymously — so an anonymous base64 image-upload path into object storage stays off the attack-surface inventory. Concretely, an attacker who obtains one handoff token (see the logging finding) can POST …/camera-capture repeatedly; because the only guard is the shared per-IP /api/v1 limiter, they can burn object-storage writes across a rotating IP set with no per-token or per-endpoint ceiling, and the generated TypeScript client in clients/shared also believes these calls need a session it will not have on the unauthenticated mobile page.

**修复方案**

Spec only. In server/api/paths/admission.yaml add `security: []` to the three operations previewFreshmanMobileCameraHandoff (line ~261), uploadFreshmanMobileCameraCapture (~290) and chooseFreshmanMobileCameraContinuation (~331), matching the adjacent public previewAdmissionSession, then regenerate server/api/openapi.bundled.yaml, server/internal/api/gen/server.gen.go and clients/shared/src/types/api.gen.ts through the normal generation task (never hand-edit generated files). Do NOT add the proposed per-token EndpointRateLimitMiddleware to /camera-capture: uploads are already capped at one per handoff by ensureFreshmanCameraHandoffUsable + MarkFreshmanCameraHandoffUploaded, request bodies by MaxBodySize plus policy.MaxMaterialBytes, and the existing limiter middleware is IP/user-keyed so a per-token variant would need new keying plumbing for no added protection.

#### 77. Cache version falls back to "0" and caches it on Redis error, so invalidated payloads can be re-served

`server/internal/pkg/cache/cache.go:268`

| | |
|---|---|
| 区域 | 后端公共包 |
| 类别 | cache-correctness |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
```go
		version, err := h.client.Get(ctx, vk).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return "0", nil
			}
			// 非 key-not-found 错误，记录日志并返回默认值
			logger.L().Warn("failed to get cache version from redis", zap.String("key", vk), zap.Error(err))
			return "0", nil
		}
```

A transport error is treated identically to "version key absent", and the caller then *caches* that answer for `versionLocalTTL` (cache.go:280-288). `BuildVersionedKey` (cache.go:294-297) therefore builds `prefix:v0:key`, and callers both read and write at that key (e.g. course.go:125/151, review/cache_response.go:26/46). `InvalidateByVersion` (cache.go:332-359) only ever bumps the counter forward, so v0 entries are never invalidated — they persist for their full `cache.DefaultTTL` (5 min) with `JitteredTTL`. The singleflight wrapper amplifies it: the flight captures the first caller's `ctx` (cache.go:249-271), so one client disconnect makes `Get` return context.Canceled and every request sharing that flight gets version "0".
```

**失败场景**

Redis has intermittent timeouts under load. Request A hits a timeout, GetVersion returns "0", the handler loads from Postgres and writes the payload to `review:course:v0:<key>` with a 5-minute TTL. Seconds later an admin hides an abusive review; `InvalidateByVersion("review:course")` bumps the real version to 7, which does nothing to the v0 entry. A subsequent Redis blip makes GetVersion return "0" again, the handler reads `review:course:v0:<key>` and serves the pre-hide payload — the hidden review reappears to users for up to 5 minutes with only a Warn log to explain it. The same mechanism serves any invalidated course/review list from the orphaned v0 namespace.

**修复方案**

Apply in server/internal/pkg/cache/cache.go, cheapest-first:

1. Stop one caller from poisoning the shared flight. Inside the GetVersion singleflight loader (cache.go:249-271), do not use the captured caller ctx for the Redis GET. Use the existing helper: `loaderCtx, cancel := ctxutil.DetachedTimeout(ctx, 200*time.Millisecond); defer cancel()` (server/internal/pkg/ctxutil/context.go:26, already used in audit.go:165 and oidc/verifier.go:47) and call `h.client.Get(loaderCtx, vk)`. This removes the most likely trigger: a client disconnecting mid-flight no longer makes healthy waiters — and the 1s process-wide version cache — see version "0".

2. Distinguish absent from unavailable. In the loader, keep `return "0", nil` only for `errors.Is(err, redis.Nil)`; for any other error log the warning and `return "", err`. Then in GetVersion, when DoValue returns an error, return the unknown signal and do NOT execute the local-cache write at cache.go:280-288, so a transport blip is never memoized for 1s.

3. Make "unknown version" bypass the cache instead of aliasing the live v0 namespace. Have GetVersion expose an ok/err form (e.g. add `GetVersionOK(ctx, prefix) (string, bool)`; keep GetVersion as a thin wrapper for the existing tests) and have BuildVersionedKey return `""` when the version is unknown. Add an empty-key guard at the top of GetRaw (cache.go:129), GetAs (cache.go:145) and Set (cache.go:166) so an empty key is a miss / no-op. That makes all five production call sites — course.go:125, 165, 201, 246 and review/cache_response.go:26 — fall through to Postgres for that request …

#### 78. authorization-model.md diverges from the Go capability catalog it declares authoritative

`docs/design/authorization-model.md:15`

| | |
|---|---|
| 区域 | 文档 |
| 类别 | docs-drift |
| 验证票数 | 1/1 |
| 来源 | 第二轮：覆盖缺口补扫 |

**证据**

```
1. **角色** — Casdoor JWT claims：`super_admin` / `school_admin` / `section_admin` / `section_moderator` / `section_reviewer` / `verified_student` / `user`
...
代码：`server/internal/pkg/capability/capability.go`
```

**失败场景**

The role list omits `freshman_provisional`, which capability_test.go:52-68 pins as one of the eight catalog roles and which is granted the full review write set (review:create / review:edit:own / review:delete:own / review:list:full, catalog.go:67-69) — a reader auditing who may post reviews from this doc concludes only verified_student can. The doc also points at capability.go for the role->capability table (the constants and roleCapabilities map now live in catalog.go), and its capability inventory (lines 28-34) omits all eight admission:*/member_blacklist:* capabilities even though every one of them is an AdminEntryCapabilities member that opens the admin surface.

**修复方案**

Edit docs/design/authorization-model.md only:
1. Line 15: add `freshman_provisional` to the role enumeration (super_admin / school_admin / section_admin / section_moderator / section_reviewer / verified_student / freshman_provisional / user), with a parenthetical that freshman_provisional is the time-limited freshman role whose capability set equals verified_student's (review:list:full / review:create / review:edit:own / review:delete:own) and that is revoked on credential expiry -- consistent with docs/design/koishi-admission-verification.md:58,296.
2. Line 21: repoint `代码：server/internal/pkg/capability/capability.go` to `server/internal/pkg/capability/catalog.go`（角色到能力表）plus `capability.go`（scope 展开逻辑）.
3. Add one sentence noting that `section_reviewer` currently expands to an empty capability set (catalog.go:63) and, being section-scoped, grants nothing even when a section scope is present -- it exists only as a Casdoor projection placeholder.
4. Bump `last-verified` to the edit date.
Skip the proposed expansion of the 典型能力 list: it is explicitly labelled non-exhaustive, so listing admission:* / member_blacklist:* is optional. If added anyway, label them as admin-entry capabilities (AdminEntryCapabilities, catalog.go:90-97) rather than implying the list is complete.
Optional recurrence guard: extend the existing TestRoleCapabilities_UsesCasdoorV2RoleCatalog with an assertion that every role name in roleCapabilities appears in docs/design/authorization-model.md.

#### 79. Two docs/design files are migration plans with pre-change narrative, one describing storage that no longer exists

`docs/design/member-blacklist-unification.md:19`

| | |
|---|---|
| 区域 | 文档 |
| 类别 | doc-accuracy |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
docs/design/member-blacklist-unification.md — status: current, `authoritative-source: server/api/openapi.yaml and server/migrations/000001_initial_schema.up.sql after implementation`.
  L12-21 "## 背景 / 当前项目存在多套成员黑名单真源" table lists "当前存储 `blacklist.json`" for three Koishi paths. `git grep blacklist.json` matches only this file and one archived exec-plan — the identifier exists nowhere in bots/, clients/ or server/.
  L226-229, L301 "`blacklist.json` 不再作为写入真源。" / "控制台黑名单页面改为调用后端 list/create/release API。" / "Koishi 不再直接修改 `blacklist.json`" — task-list wording for work already shipped (server/internal/modules/admission/repository_blacklist.go + member_blacklist_entries in migrations).

docs/design/iam-v2-casdoor.md — status: draft, `supersedes: 2026-05-01-casdoor-open-platform-iam-design.md`.
  L23 "> 本 spec 取代 `2026-05-01-…-design.md`（commit 8295a1e7）…已被本文从架构上修正。"
  L21 "**迁移性质**：绿地架构，不做兼容数据迁移；历史 Zitadel external subject、session、token 全部失效"
  L751-752 per-line change orders: "| `…/middleware/auth.go:74-76` | 修改注释 | \"Zitadel introspection\" 改为 \"Casdoor introspection\" |"
  L884 "项目已采纳 …
```

**失败场景**

A new maintainer opens docs/design/member-blacklist-unification.md (linked from docs/README.md's design list, marked status: current) to learn how member blacklisting works. The 背景 table tells them Koishi writes bans to a `blacklist.json` file and that the console still owns a local JSON source of truth. They go looking for that file, find nothing, and cannot tell which half of the document is current architecture and which half is a completed 2026-05 work plan. iam-v2-casdoor.md compounds it: a reader following its "修改注释" change table would try to apply edits that were applied months ago.

**修复方案**

Two docs, plus one durable guard. Do not delete the rationale sections — fix the tense and remove the completed to-do lists.

A. /home/wztxy/Code/StuHelper/docs/design/member-blacklist-unification.md
1. L5: `authoritative-source: server/api/openapi.yaml + server/migrations/000001_initial_schema.up.sql` — drop " after implementation".
2. L12-23: keep the rationale but make it unambiguously historical. Retitle `## 背景` -> `## 为什么统一`, change L14 to past tense ("统一前项目存在多套成员黑名单真源"), and change the L16 table header from "| 来源 | 当前存储 | 当前用途 |" to "| 来源 | 统一前存储 | 统一前用途 |". This keeps the design rationale legal under documentation-governance.md:22 while removing the false present-tense claim about a file that does not exist.
3. L198-202: DELETE the entire "待移除旧路由" block. Verified gone: `git grep -n "admission/qq-users\|admission/blacklist" -- server/` returns nothing.
4. L226-231 and L301: rewrite the "改为 / 不再" task bullets as present-tense statements of shipped behavior, e.g. L227 "`blacklist.json` 不再作为写入真源。" -> delete (the file does not exist, so there is nothing to negate); L228 -> "控制台黑名单页面调用后端 list/create/release API。"; L226 -> "`event-handlers.ts` 的入群申请黑名单判断调用后端统一准入接口。"
5. L42: "不保留 Koishi `blacklist.json` 作为长期写入路径。" -> drop this 非目标 bullet (satisfied, and it is the last remaining reference to the identifier).
6. Bump `last-verified` to the re-verification date.

B. /home/wztxy/Code/StuHelper/docs/design/iam-v2-casdoor.md
1. DELETE §11 "迁移：从 Zitadel 到 Casdoor" (L738-901) and §17 "后续实施阶段" (L1107-1120). The history already has a home in docs/internal/exec-plans/completed/; if you …

#### 80. docs/guides/ mixes dated audit snapshots and "now changed to" narrative into status:current guides, plus two stale structure claims

`docs/guides/github-migration.md:170`

| | |
|---|---|
| 区域 | 文档 |
| 类别 | doc-accuracy |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
docs/guides/github-migration.md (type: guide, status: current)
  L168-170 "## 当前就绪状态 / 以下状态于 2026-07-30 通过 GitHub API 重新核验：" followed by an audit table of 已验证 / 部分验证 / 未就绪 / 未验证 rows.
  L36 "## 历史与秘密基线" … "曾提交的真实部署环境文件" … "已删除的内部工具缓存和内部安全审查导出"
  L146 "远端部署脚本当前仍执行 registry login。迁移 GHCR 时，需要…"
docs/guides/automation.md:165 "远端机器不再由 CI / Ansible 在每次发布时下发 `deploy.remote.env`。现在改为：" (also L142 "…把旧版由开发模板带入的 `localhost` / `http` … 重写回生产占位符")
docs/guides/release-runbook.md:25 "…已完成备份（注：`prod-deploy.sh` 现已自动在迁移前执行 `backup-postgres.sh`）。"
docs/guides/frontend-development.md:31,39 the workspace tree lists `clients/web/src/constants/` and `clients/web/src/types/`; neither directory exists (actual children: api, components, composables, design-system, directives, i18n, modules, router, stores, styles, utils).
docs/guides/frontend-development.md:47 "当前主要分为" lists 7 modules; clients/web/src/modules/ actually holds 10 (admission, open-platform and resource are missing).
docs/guides/backend-development.md:69 "`modules/rbac/` 仅保留 `middleware.go`（capability 中间件），不再是完整 RBAC 模块。" — server/internal/modu …
```

**失败场景**

A maintainer picks up the repo after 2026-07-30 and reads github-migration.md's readiness table as current fact: it says GHCR has no published packages and both environments have empty secrets. Once someone publishes images or fills in DEPLOY_* secrets, the table is silently wrong but still labelled 当前 and status: current, so the maintainer either re-does completed setup or skips a step believing it is already done. Separately, a frontend dev following frontend-development.md:31 creates shared constants under clients/web/src/constants/ — a directory the guide invents — instead of the real home (clients/shared/src/constants/, which is the documented single source for capability constants), and never learns that admission, open-platform and resource modules exist.

**修复方案**

Apply only the verified parts; skip the backend/runbook/registry churn.

1. `docs/guides/frontend-development.md` (highest value): delete the `│   ├── constants/` line (L31) and `│   ├── types/` line (L39) from the workspace tree — neither directory exists and both were deliberately removed (see docs/internal/exec-plans/completed/2026-04/2026-04-17-audit-closed-items.md:276,366). Add one sentence under the tree: 共享常量与共享类型只放在 `clients/shared/src/constants/` 与 `clients/shared/src/types/`，不要在 `clients/web/src/` 下重建本地影子层. Then fix L47: either list all 10 modules (admission, auth, common, course, errors, home, open-platform, resource, review, user) or replace the enumeration with a pointer to `clients/web/src/router/` as the authoritative list. Bump `last-verified` to the date the fix lands.

2. `docs/guides/github-migration.md`: move the whole `## 当前就绪状态` section (L168 through the "因此，代码与仓库治理可进入 PR 审核…" paragraph) into `docs/internal/github-migration-2026-07-29.md` (or a new `docs/internal/` snapshot with `status: snapshot`), where staleness is explicitly permitted. In the guide, keep only the steady-state requirement already in `## 仓库治理验收` and add a single line pointing at the internal snapshot for the latest dated verification. Leave L36 `## 历史与秘密基线` in place — its requirements are present-tense and normative ("所有可达提交都必须持续满足…不得包含…"); at most reword the two past-tense bullets ("曾提交的真实部署环境文件", "已删除的内部工具缓存和内部安全审查导出") as categories rather than history. Leave L146 alone — it is factually accurate.

3. `docs/guides/automation.md:165`: restate in present tense — replace "远端机器不再由 CI …

#### 81. Bulk admin review export writes no audit record, unlike every other sensitive admin read/write in the module

`server/internal/modules/course/review/admin_export.go:36`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | audit-coverage |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
admin_export.go:36-53 — full-table export with no audit.LogFromGin and no h.logAdminOp:
	func (h *Handler) ExportReviews(c *gin.Context) {
		format := c.DefaultQuery("format", "json")
		status := c.DefaultQuery("status", "all")
		...
		if format == "csv" { h.exportCSVStream(c, status); return }
		h.exportNDJSONStream(c, status)
	}

Every sibling admin route records one: admin.go:173 `h.logAdminOp(c, req.Action, "review", reviewID, ...)`, admin.go:89, handler_content_flag.go:50, handler_sensitive_word_admin.go:70/112/134. The audit package even defines the event type for this: pkg/audit/audit.go:33 `EventDataExport EventType = "data.export"` — `grep -rn EventDataExport` shows it is declared and never used anywhere. The module's own precedent for auditing a sensitive admin *read* is user/handler_admin.go:64 (audit.EventDataAccess on identity review material). The exported stream (repository_operation_log.go:130-149, status="all") includes hidden and deleted review titles/content plus moderation_reason.
```

**失败场景**

An admin with the global AdminReviewsManage capability calls GET /api/v1/course/review/admin/export?format=csv&status=all and downloads every review in the platform, including hidden and soft-deleted content and moderation reasons. Nothing lands in audit_events or admin_operation_logs, so a later incident review of "who exfiltrated the review corpus and when" has no record — while a single hide of one review is fully logged. No page/limit bound applies either, so the size of the exfiltration is also unrecorded.

**修复方案**

In server/internal/modules/course/review/admin_export.go, emit exactly ONE audit event from the export path — do not also call h.logAdminOp (that helper writes the same audit_events table via repository_operation_log.go:30-47 and would produce a duplicate row typed data.update, since eventTypeForOperationAction("export") hits the default branch).

Concretely:
1. Change exportNDJSONStream / exportCSVStream to return (rowCount int, err error), incrementing a counter inside the StreamExportReviews callback.
2. In ExportReviews (admin_export.go:36-53), after the chosen stream returns, call:
   audit.LogFromGin(c, audit.Event{
     Type: audit.EventDataExport,
     Category: "admin_operation",   // required so the row appears in GET /api/v1/course/review/admin/logs (ListAdminOperations filters category='admin_operation') and inherits the 90-day CleanupAdminOperations retention
     ActorType: "admin",
     Resource: "review", ResourceType: "review", ResourceID: "bulk",
     Action: "export",
     Result: "success" | "failure",
     Reason: streamErr.Error() on failure,
     Details: map[string]any{"format": format, "status": status, "row_count": n, "row_limit": 10000},
   })
   Emit the failure variant (with rows written so far) when StreamExportReviews errors, since the response body is already partially streamed and the client keeps a truncated file.
3. Add a handler test asserting the event fires for both format=csv and format=ndjson, success and stream-error paths (the existing service_admin_integration_test.go:285-303 already drives both formats, so extend there).

Optional …

#### 82. Three DELETE operations declare 204 No Content but the handlers return 200 with a JSON envelope

`server/internal/modules/course/review/review_interaction.go:84`

| | |
|---|---|
| 区域 | 评课审核 |
| 类别 | contract-mismatch |
| 验证票数 | 1/1 |
| 严重度 | 原报 P2 → 验证方修正为 P3 |
| 来源 | 第一轮：全维度扫描 |

**证据**

```
Spec declares only 204 (no content):
  server/api/paths/review-favorite.yaml:70  '204': description: 取消收藏成功   (removeFavorite)
  server/api/paths/review-admin.yaml:563    '204': description: 删除成功       (deleteTeacher)
  server/api/paths/review-admin.yaml:705    '204': description: 删除成功       (deleteSensitiveWord)

Handlers all return 200 + body:
  review_interaction.go:84            response.Success(c, gin.H{"message": "favorite removed successfully"})
  handler_teacher_admin.go:122        response.Success(c, gin.H{"message": "teacher deleted successfully"})
  handler_sensitive_word_admin.go:136 response.Success(c, gin.H{"message": "sensitive word deleted"})

Generated types encode the wrong shape (clients/shared/src/types/api.gen.ts:9251-9258): `responses: { 204: { content?: never }, 401: ..., 403: ..., 500: ... }` — 200 is not a declared outcome at all.
```

**失败场景**

`adminApi.deleteTeacher(7)` returns HTTP 200 with `{"success":true,"data":{"message":"teacher deleted successfully"}}`, but the generated TS type says the only success is 204 with `content?: never`, so `result.data` is typed as never/undefined and the message is unreadable without a cast. Any consumer that branches on `response.status === 204` (or an OpenAPI response validator / contract test, which the repo already has the machinery for via kin-openapi) classifies every successful delete as a contract violation. Current callers only survive because `unwrapVoid` (clients/admin/apps/web-ele/src/api/shared-result.ts:73-82) checks a 200-299 range.

**修复方案**

Fix the SPEC, not the handlers (opposite of the finding's stated preference).

1. server/api/paths/review-favorite.yaml:70 — replace the bare `'204': description: 取消收藏成功` with a 200 response identical in shape to the sibling `addFavorite` at lines 37-52:
     '200':
       description: 取消收藏成功
       content:
         application/json:
           schema:
             allOf:
               - $ref: '../components/schemas/common.yaml#/SuccessResponse'
               - type: object
                 required: [data]
                 properties:
                   data:
                     $ref: '../components/schemas/common.yaml#/MessageData'
2. server/api/paths/review-admin.yaml:563 (deleteTeacher) and :705 (deleteSensitiveWord) — same replacement, description 删除成功. Keep the existing 401/403/404/500 refs untouched.
3. Regenerate and commit all three generated artifacts: `make bundle-spec` in server/ (redocly -> server/api/openapi.bundled.yaml), then `go generate ./internal/api/gen` (oapi-codegen) and `pnpm api:generate` in clients/ (openapi-typescript -> clients/shared/src/types/api.gen.ts). Do not hand-edit api.gen.ts. `make` has a bundle-sync check at server/Makefile:124 and clients has `check:api-drift`, so both will verify the result.

Rationale: this aligns the three outliers with the other five DELETE operations (all 200 + MessageData), makes `result.data.message` correctly typed for future consumers, and changes no server behavior — so no risk to the shipped web/uniappx/admin clients. Do NOT switch the handlers to `response.NoContent`; that would change wire behavior on …


## 4. 经验证驳回的发现

以下条目经复核判定不成立，**不应修改**。列出以避免重复提出。

### 第一轮：全维度扫描

#### ~~renovate.json coexists with .github/dependabot.yml, giving two competing dependency bots for the same ecosystems~~

原报 P3　`renovate.json:1`　票数 0/1

> **驳回**（high 置信度）：The surface observation is accurate, but every consequence claimed in the failure scenario is either impossible in the current state or factually wrong.

WHAT CHECKS OUT
- Both files exist: `renovate.json` (658 B) and `.github/dependabot.yml` (1461 B).
- `docs/guides/github-migration.md` really does name Dependabot the single authority ("依赖更新由 `.github/dependabot.yml` 每周检查 GitHub Actions、Go modules、三个 JavaScript workspace 和 Docker 基础镜像。"), and never mentions Renovate.
- `renovate.json` is an unreferenced pre-migration leftover: added 2026-04-09 in `4acafde8 chore: checkpoint current workspace changes`, whereas `dependabot.yml` arrived 2026-07-28 in `c63c428f ci(github): prepare public monorepo migration`.

WHY IT IS STILL NOT A DEFECT

1. Renovate is not installed, so `renovate.json` is inert dead config, not a "competing bot". Renovate only acts as an installed GitHub App or via a self-hosted workflow. Evidence: `gh pr list --state all` returns 21 PRs, 20 authored by `app/dependabot` and 1 by the human — zero Renovate PRs; zero `renovate/*` branches among 21 remote branches; `.githu …

#### ~~Redis outage is reported to users as an OTP cooldown; the error branch is dead code~~

原报 P2　`server/internal/modules/admission/service_student.go:320`　票数 0/1

> **驳回**（high 置信度）：The code-level observation is accurate, but the severity-justifying failure scenario cannot occur — it is already handled by two independent fail-closed guards on the same Redis client, so this does not deserve to be reported as a P2 error-handling finding.

WHAT IS TRUE. At /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_student.go:314-327 the ordering is inverted: `if errors.Is(err, redis.Nil) || result != "OK" { return ErrAdmissionOTPCooldown }` precedes `if err != nil`. go-redis `StatusCmd.Result()` is `return cmd.val, cmd.err` (/home/wztxy/go/pkg/mod/github.com/redis/go-redis/v9@v9.18.0/command.go:821-823) and `cmd.val` is never assigned on failure, so on any non-Nil error `result == ""` and line 320 wins; lines 323-325 are unreachable. The twin at /home/wztxy/Code/StuHelper/server/internal/modules/user/service_student_email_otp.go:399-411 is identical, and /home/wztxy/Code/StuHelper/server/internal/modules/auth/otp.go:243-256 does use the correct ordering. So the dead branch is real.

WHY THE FAILURE SCENARIO IS REFUTED (the finding's whole basis for P2).
1 …

#### ~~Runtime payload-guard helpers are hand-copied into 19 web files while clients/shared already exports isRecord~~

原报 P2　`clients/web/src/modules/course/coursePayload.ts:17`　票数 0/1

> **驳回**（high 置信度）：The raw duplication count checks out, but the two load-bearing claims in the finding -- "the bodies are already identical / this is a pure move" and the failure scenario -- are both factually false against the current code, so the finding as written should not be acted on.

WHAT IS TRUE (verified):
- 19 private `function isRecord` definitions exist in non-test clients/web files, exactly the list given (grep count = 19).
- 15 web files define read*/readOptional* families whose bodies are semantically identical modulo formatting (/home/wztxy/Code/StuHelper/clients/web/src/stores/auth.ts, stores/user.ts, modules/review/reviewListPayload.ts, modules/open-platform/pagePayload.ts all have byte-equivalent `readString`/`readInteger`/`readBoolean`/`readNumber` bodies; the stores use 4-space/double-quote/semicolon formatting, the modules use 2-space/single-quote).
- /home/wztxy/Code/StuHelper/clients/shared/src/api/errors.ts:15 does export `isRecord`, re-exported at /home/wztxy/Code/StuHelper/clients/shared/src/api/index.ts:17, and `@stuhelper/shared/api` is a real subpath export in clients/sh …

#### ~~RefreshCourseRatingStatsTx and RefreshTeacherRatingStatsTx are ~90 verbatim-duplicated lines differing by four tokens~~

原报 P2　`server/internal/modules/course/review/repository_rating.go:247`　票数 0/1

> **驳回**（high 置信度）：The duplication itself is factually accurate, but the finding does not survive as a P2 defect.

WHAT I CONFIRMED (the finding's factual core is right):
Cited location is correct. `diff` of `server/internal/modules/course/review/repository_rating.go:247-336` against `server/internal/modules/course/review/repository_rating_stats.go:103-192` yields exactly 90 lines each, differing only in the 4 spots claimed: `r.teacher_id`/`r.course_id`, the DELETE table, the INSERT table + column list, and the `fmt.Errorf` label. Same base/stats/dist CTE, same local `statRow`, same scan loop, same DELETE-then-multi-VALUES-INSERT with 7 placeholders per row. No test pins the two to equivalent behavior (`service_review_stats_test.go:31,36` only counts rows in each table).

WHY I REFUTE ANYWAY:

1. The stated failure scenario is provably impossible. `ReviewRatings` is `map[string]int` (`model.go:10`), and `validateRatingValues` (`service.go:319-346`) rejects any `v < 1 || v > 5` plus unknown/ill-formed dimension keys. The only production writers of `reviews.ratings` are `CreateReturning` (`repository.go: …

#### ~~Error-code reference documents 49 codes the server never emits and omits the 6 admission codes it does emit~~

原报 P2　`docs/reference/error-codes.md:136`　票数 0/1

> **驳回**（high 置信度）：The raw facts are largely accurate, but the finding fails on failure scenario, framing, and fix quality.

WHAT CHECKS OUT
- I scripted qualified-reference analysis over all 116 `ErrorCode` constants in `/home/wztxy/Code/StuHelper/server/internal/pkg/errs/codes.go`: 51 (not 49) have no non-test `errs.X` emission site. Close enough.
- `/home/wztxy/Code/StuHelper/server/internal/modules/admission/handler_errors.go:14-19` really does declare 6 dotted codes outside `errs/codes.go`, and `handler_errors.go:48` really returns HTTP 410 + `admission.token_expired`. They are absent from `docs/reference/error-codes.md`.

WHY THE FAILURE SCENARIO CANNOT OCCUR
Every real consumer already handles those 6 codes by exact string, with tests:
- `/home/wztxy/Code/StuHelper/clients/web/src/modules/admission/admissionToken.ts:11-22` defines TERMINAL/INVALID/EXPIRED code sets containing all of them; `mapAdmissionApiError()` returns `'expired'` for `admission.token_expired`. It is wired into `AdmissionPage.vue:524`, `OldStudentVerificationFlow.vue:281`, `FreshmanCameraFlow.vue:488`, `FreshmanMobileCameraPag …

#### ~~Six exported helpers in server/internal/pkg have zero call sites, and three of them invite context-losing audit writes~~

原报 P3　`server/internal/pkg/audit/audit.go:105`　票数 0/1

> **驳回**（high 置信度）：The reference counts are factually accurate — I confirmed all seven functions (the title says "six" but the evidence lists seven) have zero call sites in Go code including _test.go, and I checked the one aliased import (`auditpkg` at server/internal/modules/course/review/repository_operation_log.go:9) so no alias hid a caller. But an accurate "unreferenced" count is not the same as a defect, and the finding's severity narrative is affirmatively contradicted by project law.

1. THE CENTRAL CLAIM IS CONTRADICTED BY THE PROJECT'S OWN WRITTEN GUARDRAIL. All the claimed weight of this finding comes from the assertion that audit.Log/LogSuccess/LogFailure are "an active trap" a developer could autocomplete into. docs/design/iam-implementation-guardrails.md, section "审计写入上下文", settles this in the opposite direction:
  - line 66: "请求链路内的审计事件必须使用 `audit.LogContext(ctx, event)`、`audit.LogFromGin(c, event)`、`audit.LogSuccessContext(...)` 或 `audit.LogFailureContext(...)`" — the Context variants are mandatory inside a request path;
  - line 70: "后台启动、bootstrap 等没有请求上下文的调用可以继续使用 `audit.Log`，但有业务 `c …

#### ~~QueryTimeout also bounds connection establishment, so the configured 5s Oracle connect timeout is unreachable~~

原报 P2　`server/internal/modules/externaldata/oracle_student_directory.go:215`　票数 0/1

> **驳回**（high 置信度）：The mechanical claim is accurate, but it is not a defect and the proposed fix would be a regression.

What I confirmed (mechanism is real):
- server/internal/modules/externaldata/oracle_student_directory.go:215-218 wraps the lookup in `withOptionalTimeout(ctx, d.queryTimeout)` and then calls `d.db.QueryContext(queryCtx, ...)`. `withOptionalTimeout` (lines 484-489) always installs a deadline because `normalizeOracleStudentDirectoryConfig` clamps QueryTimeout to 1-60s with a 3s default (lines 24, 306-311).
- go-ora v2.9.0 really does turn the DSN option into the dialer timeout: configurations/connect_config.go:251-258 maps `CONNECTION TIMEOUT` -> `SessionInfo.ConnectTimeout`, and network/session.go:519-542 dials via `net.Dialer{Timeout: ConnectTimeout}.DialContext(ctx)`. connection.go:459-462 additionally arms `session.StartContext(ctx)`, whose watchdog breaks/disconnects the session on ctx.Done, so the TLS + negotiation + logon phases are capped by the outer ctx too.
- Defaults match the claim: config.go:600-601 = connect 5s / query 3s, MaxIdleConns 1 (:603), ConnMaxIdleTime 60s (:605 …

#### ~~Least-privilege gate never ties the runtime Oracle account to the provisioned read-only account~~

原报 P3　`infra/ops/lib/common.sh:399`　票数 0/1

> **驳回**（high 置信度）：REFUTED. The code fact (no gate ties EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME to EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME) is true, but both halves of the stated failure scenario fail verification, and the proposed fix is not a clear improvement.

(1) Cited line is off by one: the gate is infra/ops/lib/common.sh:400-401, not :399.

(2) The SYS/SYSTEM half is already handled, and specifically as a PRE-DEPLOY die — the opposite of what the finding claims. server/internal/modules/externaldata/oracle_student_directory.go:285-287 calls isDisallowedOracleRuntimeUsername (defined :396-403, rejecting SYS/SYSBACKUP/SYSDG/SYSKM/SYSRAC/SYSTEM) inside normalizeOracleStudentDirectoryConfig, which runs from NewOracleStudentDirectory — invoked by server/cmd/external-student-source-smoke/main.go:99. So infra/ops/external-student-source-smoke.sh, which infra/ops/admission-student-source-go-live.sh:87 runs BEFORE admission-production-readiness.sh, exits non-zero with "oracle runtime username must be a dedicated non-administrative account". EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME=SYSTEM therefor …

#### ~~Migration 000018 triggers funnel every reviews/teachers/departments write through one outbox row, serializing all review writes~~

原报 P2　`server/migrations/000018_teacher_public_stats_projection.up.sql:32`　票数 0/1

> **驳回**（high 置信度）：The cited code is quoted accurately and the Postgres mechanism is real, but this is a deliberate, documented, test-locked design whose expensive work is already kept outside the lock, and both proposed fixes are regressions.

CONFIRMED MECHANICS: server/migrations/000018_teacher_public_stats_projection.up.sql:22 hardcodes dedupe_key='teacher_public_stats'; line 32 is ON CONFLICT (stream, dedupe_key); domain_event_outbox_stream_dedupe_idx is UNIQUE on (stream, dedupe_key) at 000001_initial_schema.up.sql:2786. So INSERT ... ON CONFLICT DO UPDATE does take a row-exclusive lock on one tuple, held to commit, and concurrent qualifying statements serialize. PostReview (service_review_write.go:170 CreateReturning -> 194 IncrementCourseReviewCount -> 197 refreshReviewTargetTx -> 200 enqueueReviewFGASyncTx) and BatchUpdateReviews (service_admin.go:376-394, maxBatchSize=100 at admin.go:20) do keep working after the trigger fires. That much is not in dispute.

WHY IT IS REFUTED:

1. The singleton dedupe key IS the coalescing device, and it is locked by an existing test. repository_teacher_public …

#### ~~academics.ReplaceSnapshot upserts one row per round trip inside a transaction hard-bounded to DB_QUERY_TIMEOUT x 3~~

原报 P3　`server/internal/modules/academics/repository_import.go:119`　票数 0/1

> **驳回**（high 置信度）：The mechanical claims are factually accurate, but the failure scenario cannot occur in the current code, and the proposed fix is not a clear improvement. Refuting on both grounds.

WHAT I CONFIRMED AS TRUE
1. /home/wztxy/Code/StuHelper/server/internal/modules/academics/repository_import.go:117-131 is indeed a per-row `tx.QueryRow(... RETURNING id)` loop, and the same row-at-a-time shape repeats at lines 137, 157, 183, 237, 245 and 271.
2. /home/wztxy/Code/StuHelper/server/internal/pkg/db/db.go:387 `txTimeout := d.timeout * txTimeoutMultiplier` with `txTimeoutMultiplier = 3` (db.go:30), and `DB_QUERY_TIMEOUT` defaults to 5 (config.go:374, .env.example:39), so the whole `WithTx` body really is bounded at 15s by default. db.go:431-438 also rejects the commit if the ctx expired, so an overrun does roll the import back, exactly as claimed.

WHY IT IS STILL REFUTED
3. The failure scenario has no reachable input. `ReplaceSnapshot` has exactly one caller: `TriggerImport` (service.go:62), whose only caller is the admin handler (handler.go:90). No scheduler, cron, or CLI touches it (grepped se …

#### ~~Identity review approves on URL presence only — a deleted/never-uploaded or expired-signature photo still enables Approve~~

原报 P1　`clients/admin/apps/web-ele/src/views/users/identity-review/index.vue:73`　票数 0/2

> **驳回**（high 置信度）：REFUTED. The finding's frontend observations are accurate as code descriptions, but the load-bearing backend claim ("`PresignGetURL` is a local signing operation and never touches the network... there is no HEAD/Stat existence check") is factually wrong for the production code path, and that breaks both failure cases.

What the audit missed: `s.photoStore` is NOT an `*objectstorage.Store`. Production wiring at /home/wztxy/Code/StuHelper/server/internal/app/modules.go:292 is `user.WithIdentityPhotoStore(newIdentityPhotoStorageAdapter(storageService, storage.DefaultMountKey))`. That adapter (/home/wztxy/Code/StuHelper/server/internal/app/identity_photo_storage_adapter.go:33-36) implements `PresignGetURL` as:
  `url, err := a.service.GetDownloadURLByMountKey(ctx, a.mountKey, key)`
and `storage.Service.GetDownloadURLByMountKey` (/home/wztxy/Code/StuHelper/server/internal/modules/storage/service.go:223-238) does:
  `if err := ensureObjectExists(ctx, driver, mount, objectKey); err != nil { return "", err }`
with `ensureObjectExists` (service.go:239-247) calling `driver.Stat(...)` -> `s3Dri …

> **驳回**（high 置信度）：REFUTED — the core evidence claim is wrong, and the scenario fail-closes end to end.

1) "resolveIdentityReviewPhoto ... there is no HEAD/Stat existence check" is false for the production wiring. `photoStore` is never a raw `*objectstorage.Store`; the sole non-test injection is server/internal/app/modules.go:292 → `newIdentityPhotoStorageAdapter(storageService, storage.DefaultMountKey)`. That adapter's PresignGetURL (server/internal/app/identity_photo_storage_adapter.go:34-37) calls `storage.Service.GetDownloadURLByMountKey`, which at server/internal/modules/storage/service.go:231-234 runs `ensureObjectExists(ctx, driver, mount, objectKey)` BEFORE presigning; that calls `driver.Stat` (storage/driver.go:92-105) → `objectstorage.Store.Stat` → S3 HeadObject (objectstorage/store.go:165-181). The audit's grep was scoped to internal/modules/user/*.go and missed the check one layer down.

2) Case A (S3 object deleted/expired/never uploaded) therefore cannot produce an approvable record. Missing key → HeadObject 404 → classifyErrorKind → ErrorKindNotFound (objectstorage/errors.go:77-80) → no …

#### ~~Operation-logs pagination never binds page-size: page count is computed from 10 while requests use 20, and the size selector is dead~~

原报 P1　`clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue:170`　票数 0/2

> **驳回**（high 置信度）：REFUTED — the finding's core claim is contradicted by the current code; its evidence quote is the real template block with the decisive line deleted.

1. The binding exists. clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue:170-179 reads:
   <ElPagination
     v-model:current-page="query.page"        (171)
     v-model:page-size="query.pageSize"       (172)  <-- PRESENT; omitted from the finding's quote
     background                               (173)
     layout="prev, pager, next, sizes, total" (174)
     :page-sizes="[10, 20, 50, 100]"          (175)
     :total="total"                           (176)
     @current-change="refreshPage"            (177)
     @size-change="refreshPage(1)"            (178)
   />                                         (179)
   The finding's 8-line quote is byte-identical to the real 9-line block minus line 172, i.e. the "missing" line was dropped from the evidence.

2. Not a stale snapshot. `git status --porcelain` and `git diff HEAD` on the file are both empty (worktree == HEAD), and `git log -S'v-model:page-size="query.pageSi …

> **驳回**（high 置信度）：The finding's central evidence is factually false for the current code. All three sub-claims fail.

(1) MISQUOTED EVIDENCE — `v-model:page-size` IS bound. clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue:170-179 actually reads:
  <ElPagination
    v-model:current-page="query.page"
    v-model:page-size="query.pageSize"     <-- line 172
    background
    layout="prev, pager, next, sizes, total"
    :page-sizes="[10, 20, 50, 100]"
    :total="total"
    @current-change="refreshPage"
    @size-change="refreshPage(1)"
  />
Line 172 is silently absent from the finding's quoted block. This is committed at HEAD (`git show HEAD:clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue`), `git status --porcelain` on the file is clean, and `find` confirms exactly one operation-logs/index.vue exists in the repo — so the auditor was not looking at a different worktree or a stale copy of a duplicated file. The claim "No page-size / v-model:page-size is passed" is simply wrong.

(2) THE innerPageSize=10 FALLBACK IS UNREACHABLE. element-plus 2.14.3 pagination.mjs:3 …

#### ~~Both stuhelper-core and stuhelper-group-guard register `guild-member-request`, so every join request is decided and answered twice~~

原报 P1　`bots/koishi/plugins/stuhelper-group-guard/src/events.ts:25`　票数 0/2

> **驳回**（high 置信度）：REFUTED. The finding's central premise -- that stuhelper-core also registers `guild-member-request` -- is false at runtime, so there is exactly one listener and the race cannot occur.

1) Core's listener is unreachable dead code. `bots/koishi/plugins/stuhelper-core/src/core/modules/event-handlers.ts:33` sits inside `registerEventListeners(host)`, which is called only from `EventModule.init()` (core/modules/event.module.ts:55). `EventModule` is only constructed by `eventRuntimeModule` (event.module.ts:70-78), which is only listed in `MODULE_REGISTRATIONS` (runtime/registry.ts:45), which is only read by `getRuntimeModules()` (registry.ts:52). Grepping core's src for `getRuntimeModules` returns ONLY its own definition plus `registry.test.ts` -- no production caller. The actual plugin entry (`src/index.ts:37`) calls `registerRuntimeModules`, which is a no-op stub: `bots/koishi/plugins/stuhelper-core/src/setup/register-runtime-modules.ts:7-9` -> `export function registerRuntimeModules(_ctx, _config) { logger.info('stuhelper-core 旧群管运行时模块已停用，仅注册 WebUI 与 Console API') }` ("legacy group-mana …

> **驳回**（high 置信度）：The finding's central premise is false: stuhelper-core does NOT register a `guild-member-request` listener at runtime, so nothing is handled twice.

Trace of core's listener, end to end:
- `/home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-core/src/core/modules/event-handlers.ts:33` (`host.ctx.on('guild-member-request', ...)`) lives inside `registerEventListeners(host)`.
- The only non-test caller of `registerEventListeners` is `EventModule.init()` (`src/core/modules/event.module.ts:55`).
- `EventModule` is only constructed by `eventRuntimeModule` (`event.module.ts:70`), which is only referenced by `src/runtime/registry.ts:9,45`.
- `registry.ts`'s `getRuntimeModules()` has ZERO non-test callers (`grep -rn "getRuntimeModules"` → only registry.ts:52 definition + registry.test.ts). `StuhelperGroupCenterService.registerModule` / `initModules` (`src/core/services/stuhelper-group-center.service.ts:102,128`) likewise have zero non-test callers, so `_modules` is always empty.
- The production entry `src/index.ts:32-38` calls `registerRuntimeModules(ctx, config)`, and `src/setup/regist …

#### ~~Disabling both reminder delivery channels makes the bot report reminders as successfully delivered~~

原报 P3　`bots/koishi/plugins/stuhelper-group-guard/src/admission-actions.ts:93`　票数 0/1

> **驳回**（high 置信度）：The mechanics the finding describes are real, but they are the deliberately chosen, documented, and tested product behavior — not a defect — and the proposed fix would reintroduce a constraint the team removed two days ago plus create new failure loops.

What I confirmed as mechanically accurate:
- /home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/admission-reminder-delivery.ts:84-87 returns `{}` (no `cancelled`) when both channels are off.
- /home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/admission-actions.ts:93-96 therefore ACKs `{action:'remind', success:true}`, mark `'reminder'`.
- Backend on that ACK: /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_session.go:890-898 -> `applySuccessfulBotEventTx` -> `MarkReminderSentTx` (/home/wztxy/Code/StuHelper/server/internal/modules/admission/repository_bot_queries.go:94-98: `SET last_reminded_at = $2, next_reminder_at = $3, last_bot_error = NULL`) plus `queueNextReminderTx`, and `MarkBotActionSucceededTx` clears `last_error`.
- `admissionReminderDeliveryDisabled` (/home/ …

#### ~~429 is not declared on four endpoints that carry EndpointRateLimitMiddleware~~

原报 P2　`server/api/paths/user-identity.yaml:90`　票数 0/1

> **驳回**（high 置信度）：The factual observation is accurate, but the stated failure scenario is impossible and the P2 severity rests entirely on a fabricated mechanism.

WHAT CHECKS OUT
- /home/wztxy/Code/StuHelper/server/api/paths/user-identity.yaml:90-95 — uploadIdentityPhoto declares only 201/400/401/500. Confirmed.
- The routes really do carry the limiter: server/internal/modules/user/handler.go:101-105, 113, 114 and server/internal/modules/auth/handler.go:201; EndpointRateLimitMiddleware returns 429 via response.RateLimitExceeded (server/internal/pkg/middleware/ratelimit.go:241-245 -> server/internal/pkg/response/response.go:136-142). Confirmed.
- The internal inconsistency is real: exchangeNative (auth.yaml:448) declares 429 on the same refreshLimiter; requestStudentEmailOTP (:252) and verifyStudentEmailOTP (:326) declare 429 on the same verifyLimiter; review write ops declare 429 (review-crud.yaml:269,313,347,389,431).
- The finding actually UNDERCOUNTS: searchReviews (review-crud.yaml:111) and getBatchCourseReviews (:177) carry ProgressiveEndpointRateLimitMiddleware (handler.go:96-107), which also e …

#### ~~`capturedAt` is part of the declared camera-capture request and is sent by the web client, but the Go DTO has no such field so it is silently discarded~~

原报 P2　`server/internal/modules/admission/handler_user.go:28`　票数 0/1

> **驳回**（high 置信度）：The mechanical claim is accurate, but the failure scenario is factually wrong and the field is inert, so this is a P3-at-most spec-hygiene nit, not a defect worth changing.

WHAT I CONFIRMED AS TRUE:
- /home/wztxy/Code/StuHelper/server/api/components/schemas/admission.yaml:500-511 declares `capturedAt` (type: string, format: date-time) as an OPTIONAL property (`required: [contentType, imageBase64]`).
- /home/wztxy/Code/StuHelper/server/internal/modules/admission/handler_user.go:28-31 binds only ContentType/ImageBase64, and both handlers (handler_user.go:137-142 and 307-311) build `CameraCaptureInput` / `FreshmanCameraHandoffCaptureInput` without any timestamp. models.go:249-254 and 286-290 have no CapturedAt field. `capturedAt` reaches nothing below the handler; buildFreshmanMaterial (service_freshman.go:593-609) and newFreshmanMaterialRecord (service_freshman.go:642-659) never see a time.
- /home/wztxy/Code/StuHelper/clients/web/src/modules/admission/cameraCapture.ts:102-107 does send `capturedAt: new Date().toISOString()`.

WHY IT IS NOT A REAL DEFECT:

1. The failure scenario's us …

#### ~~Review list responses return `page`/`pageSize` that the spec's data schema does not declare~~

原报 P3　`server/internal/modules/course/review/response_contract.go:5`　票数 0/1

> **驳回**（high 置信度）：The literal observation is factually accurate but the defect and its failure scenario are not.

VERIFIED TRUE (facts): `/home/wztxy/Code/StuHelper/server/internal/modules/course/review/response_contract.go:5-10` does emit `page`/`pageSize`, used at `review_read.go:70,114,173` and `response_contract.go:34`; and `/home/wztxy/Code/StuHelper/server/api/paths/review-crud.yaml` declares only `list`/`total` for all four data objects (lines 39-51, 88-100, 145-157, and 216-226 for the batch group), so `clients/shared/src/types/api.gen.ts:8299-8303` types `data` as `{ list: Review[]; total: number }`.

WHY IT IS REFUTED:
1. Not a contract violation. None of those response `data` objects sets `additionalProperties: false`, so extra members are permitted by the spec; the spec is merely under-declared. The runtime OpenAPI middleware validates requests only (`server/internal/pkg/middleware/openapi_validation.go:87` -> `openapi3filter.ValidateRequest`), so nothing rejects, logs, or breaks on the extra fields.
2. The stated failure scenario cannot occur. The validator is mounted on the whole `/api/v …

#### ~~`IdentityPhotoUploadResult` declares three fields (rejectionReason, createdAt, updatedAt) that the upload handler never returns~~

原报 P3　`server/api/components/schemas/user-system.yaml:165`　票数 0/1

> **驳回**（high 置信度）：The finding's raw observation is factually accurate, but it is not a defect and the claimed failure scenario is impossible. Refuted on four independent grounds.

1) It is NOT a contract mismatch. The three fields sit outside `required` (server/api/components/schemas/user-system.yaml:167 declares only `required: [key]`), so they are optional. A response of `{"key":"identity/...jpg"}` fully validates against `IdentityPhotoUploadResult`. Nothing is violated — no schema validation fails, and the generated code faithfully mirrors the spec. The generated Go type even renders them as `*time.Time` / `*string` with `,omitempty` (server/internal/api/gen/server.gen.go:6905-6912), i.e. the contract explicitly anticipates their absence. The category label "contract-mismatch" is wrong; this is at most spec over-declaration.

2) The failure scenario cannot occur. The claim is that a client "reads `undefined` forever" and silently shows a blank 'uploaded at ...'. But the generated TS marks all three optional (clients/shared/src/types/api.gen.ts:5400-5408: `rejectionReason?: string | null; createdAt? …


### 第二轮：覆盖缺口补扫

#### ~~A11yButton fires its handler twice on every keyboard Enter/Space, so toggle controls never open~~

原报 P1　`clients/uniappx/src/components/A11yButton.vue:26`　票数 0/2

> **驳回**（high 置信度）：REFUTED by direct execution against the real H5 dev build in Chromium (Playwright), not by reading alone.

What I verified in code
- /home/wztxy/Code/StuHelper/clients/uniappx/src/App.vue:8-18,97 does register a document capture-phase keydown listener that matches `.a11y-button[role="button"]`, calls `event.preventDefault()` and `target.click()`.
- /home/wztxy/Code/StuHelper/clients/uniappx/src/components/A11yButton.vue:26-30,40-41 does also bind `@keydown="handleKeydown"` with no `defaultPrevented` guard.
- The rendered element really is `<uni-button class="a11y-button" role="button" tabindex="0">` (probe: PROBE3_TAG UNI-BUTTON, PROBE3_ROLE "button"), so the App.vue selector does match and both handlers really do run.

So the two-handler overlap is real — but the claimed user-visible failure is not.

Empirical results (temporary probe specs, since reverted; toggle = `uni-review-replies-<id>` on /pages/course/detail):
- Focus toggle, press Enter: `.replies-section` count 0 -> 1, `aria-expanded` becomes "true". Panel OPENS. Second Enter closes it. Third opens again. Keyboard toggling …

> **驳回**（high 置信度）：REFUTED — I reproduced the exact scenario in a real browser and the toggle behaves correctly: one Enter press opens the reply panel and it stays open.

What I ran (Playwright, desktop-chromium, probe specs appended to a copy of clients/uniappx/tests/e2e/surface.spec.ts so they could reuse mockUniApi/gotoUniPage; all probe files deleted afterwards, repo left clean):

1. Enter on `uni-review-replies-<id>` on /#/pages/course/detail: `{"clicks":1,"keys":1,"aria-expanded":"true","panelVisible":true}`; a second Enter gives `"false"`. A MutationObserver on `aria-expanded` recorded exactly ONE change ("true"). If toggleReplies had run twice in the same tick the final committed state would have been `false` with the panel closed. It is not.
2. Order probe on `uni-course-search-submit`: log = `["click:trusted=false","keydown-capture:prevented=true","keydown-bubble:prevented=true"]`, i.e. App.vue's document-capture listener does fire and does dispatch a synthetic click, and the keydown does keep propagating to the element. So the finding's premise about event flow is right.
3. The decisive expe …

#### ~~SSO login is unreachable on the mp-weixin target: handleSSOLogin requires either `plus` or `window`~~

原报 P1　`clients/uniappx/src/pages/auth/login.vue:59`　票数 0/1

> **驳回**（high 置信度）：I read the cited code and then empirically tested the premise the finding rests on.

The code quotation is accurate. /home/wztxy/Code/StuHelper/clients/uniappx/src/pages/auth/login.vue:13 is `const isNativeApp = typeof plus !== 'undefined'`, and line 59 is `if (typeof window === 'undefined') throw new Error(t('auth.login.ssoInitFailed'))` with no third branch. The author demonstrably knows about uni-app conditional compilation elsewhere: /home/wztxy/Code/StuHelper/clients/uniappx/src/pages/user/index.vue:50-70 uses `// #ifdef H5` / `// #ifndef H5` with a plus.runtime.openURL fallback. So the shape of the finding is plausible on paper.

What refutes it is the premise that mp-weixin is a real, buildable target. It is not:

1. `@dcloudio/uni-mp-weixin` (the mini-program platform compiler) is neither in clients/uniappx/package.json nor installed in node_modules. Only `@dcloudio/uni-h5`, `@dcloudio/uni-h5-vue`, `@dcloudio/uni-app`, `@dcloudio/uni-components`, `@dcloudio/vite-plugin-uni` are present, and vite-plugin-uni does not depend on it (I enumerated its deps).
2. clients/uniappx/vite …

#### ~~GET /admin/stats is registered before admin.Use(...), skipping MFA context and RequirePrivilegedMFA~~

原报 P2　`server/internal/modules/course/review/handler.go:154`　票数 0/1

> **驳回**（high 置信度）：The mechanism described is factually correct but the "gap" is a deliberate, tested design carve-out, and the proposed fix would regress it.

1. Cited code is accurate. server/internal/modules/course/review/handler.go:153-167 registers `admin.GET("/stats", ...)` before `admin.Use(adminRouteMiddlewares...)`, and gin does snapshot group.Handlers at registration time, so `adminMiddlewares` (in prod: `[user.MFAContextMiddleware(repo), rbac.RequirePrivilegedMFA()]` from internal/app/modules_auth.go:126-134, wired at internal/app/modules.go:70,82 -> modules_course_metrics.go:74) does not apply to /admin/stats.

2. It is intentional, not an oversight. `git show 497d2624` ("fix: keep review admin dashboard out of step-up gate") is exactly the commit that MOVED `/admin/stats` out of the group and re-listed authMiddleware + Entry + DashboardView explicitly. It also added a regression test, server/internal/modules/course/review/route_contract_test.go:76-108 `TestReviewAdminStatsDoesNotUseGroupStepUpGate`, which passes a blocking gate as `adminMiddlewares` and asserts `/admin/stats` returns 200 w …

#### ~~Academic import-job admin routes are gated by the school-scoped user:school:* capabilities~~

原报 P3　`server/internal/app/admin_authorizers.go:30`　票数 0/1

> **驳回**（high 置信度）：The cited code is quoted accurately (/home/wztxy/Code/StuHelper/server/internal/app/admin_authorizers.go:28-33), and the observation that academic sources/import jobs are platform-level (no school_id anywhere in academics.Source / academics.ImportJob, /home/wztxy/Code/StuHelper/server/internal/modules/academics/model.go:3-25) while storage uses user:system:* is factually correct. But it is a naming-consistency observation with no reachable defect:

1. Zero behavioral difference today. In /home/wztxy/Code/StuHelper/server/internal/pkg/capability/catalog.go:38-56 the only roles holding user:school:read/update are super_admin (unscoped) and school_admin (school-scoped). RequireGlobalCapability -> authorizeGlobalCapability -> capability.HasGlobalGrant requires grant.Global (/home/wztxy/Code/StuHelper/server/internal/platform/authorization/service.go:88; /home/wztxy/Code/StuHelper/server/internal/pkg/capability/capability.go:217-224). So global user:school:update and global user:system:update resolve to exactly the same principal set: super_admin only. Swapping the constants changes nothi …

#### ~~Review admin /stats is registered before admin.Use(), so it is the only route in that group without the MFA chain~~

原报 P2　`server/internal/modules/course/review/handler.go:154`　票数 0/1

> **驳回**（high 置信度）：The mechanical claim is accurate (Gin snapshots the chain, so `/admin/stats` at handler.go:154-162 does not receive the `admin.Use(...)` middlewares added at line 164-166), but the finding's framing, failure scenario, and fix are all wrong.

1. It is an explicit, tested decision, not accidental ordering. `git log -S` shows commit 497d2624 "fix: keep review admin dashboard out of step-up gate", whose diff literally moves `admin.GET("/stats", ...)` from inside the group block to above `admin.Use(...)` and adds a regression test: `server/internal/modules/course/review/route_contract_test.go:76 TestReviewAdminStatsDoesNotUseGroupStepUpGate`, which registers a blocking gate as `adminMiddlewares` and asserts `/admin/stats` returns 200 while `/admin/logs` returns 428. The proposed fix (move `admin.Use` up, drop the per-route chain) would revert that commit and fail that existing test. So "the gap is created by statement ordering rather than by an explicit choice" is factually false.

2. The concrete failure scenario cannot occur as written. It claims a `school_admin` (or `admin`) with unenr …

#### ~~/internal/sms/send is mounted on the engine root, outside every rate limiter that protects /api/v1~~

原报 P2　`server/internal/app/modules.go:258`　票数 0/1

> **驳回**（high 置信度）：The mechanical claim is accurate but the failure scenario cannot occur in the shipped deployment, and half the proposed fix would break production.

What checks out:
- /home/wztxy/Code/StuHelper/server/internal/app/modules.go:255-259 does register the route on the *gin.Engine, so its chain is only registerGlobalMiddleware + CORS (server/internal/app/router.go:60-72), and the two Redis limiters really are installed only on the api group (modules.go:221-224). server/internal/pkg/sms/handler.go:61-66 really does accept the shared key via `?internal_key=`.

Why the exploit path is closed (the finding's premise fails):
1. The endpoint is never routed by the public ingress. `grep -n internal infra/nginx/*.conf` returns nothing. In infra/nginx/baota-stuhelper.conf the only backend proxy_pass targets are `^~ /api/`, `/health`, `/metrics`, `/docs/`; the catch-all `location / { proxy_pass http://127.0.0.1:18000; }` (line 134-135) sends `POST /internal/sms/send` to the *web SPA container*, and the join.stuhelper.com block ends in `location / { return 404; }` (line 214-215). Same in infra/nginx/ …

#### ~~One unrecognized section from ListObjects makes every authenticated request from that user return 503~~

原报 P1　`server/internal/platform/authorization/role_scope_resolver.go:124`　票数 0/1

> **驳回**（medium 置信度）：The mechanism is real but the finding's premise and its proposed fix are both wrong, and applying the fix would open an authz hole.

1) Mechanism confirmed (only part that holds). /home/wztxy/Code/StuHelper/server/internal/platform/authorization/role_scope_resolver.go:115-131 returns a hard error, /home/wztxy/Code/StuHelper/server/internal/pkg/middleware/auth_context.go:51 wraps it as errRoleScopeUnavailable, auth.go:226-231 makes authBackendUnavailable() true, and modules_auth.go:60/63 wires that resolver into the GLOBAL authMW/optionalAuthMW. So a non-parsable section ID does produce 503 on every authenticated route for that user.

2) The trigger is not "legitimate", it is pure operator misconfiguration, and only for users who ALSO hold the section_admin/section_moderator Casdoor claim (needsResolvedScopes, role_scope_resolver.go:74-78, plus containsRole gate at :107). No section object other than the synthetic review-moderation one exists anywhere in the system: infra/openfga/model.fga type section is only ever instantiated by fga.WriteReviewModerationSection / WriteReviewRelation …

#### ~~Section moderators' SQL data boundary is parsed out of the FGA object-ID string, not read from the authoritative section#school tuple~~

原报 P2　`server/internal/platform/authorization/role_scope_resolver.go:126`　票数 0/1

> **驳回**（high 置信度）：Refuted on three independent grounds.

(a) The finding's stated premise is factually wrong. It asserts "nothing in the codebase writes section tuples ... they are entirely operator-authored". They are written by application code, always from the same schoolID variable that generates the synthetic name: server/cmd/fga-setup/school_tuples.go:79-86 writes `{User: "school:"+schoolID, Relation: "school", Object: "section:"+fga.ReviewModerationSectionID(schoolID)}`, and server/internal/pkg/fga/relation_writer.go:8-12 (WriteReviewModerationSection) writes the identical tuple. A section whose name says school 5 but whose `school` tuple points at school:9 cannot be produced by the application at all.

(b) The failure scenario is mechanically inverted; the two boundaries cannot disagree. The per-item boundary is derived from the *same* naming convention as the list boundary. WriteReviewRelations (relation_writer.go:17,24) writes `review:<id>#section @ section:school_<review.school_id>_review_moderation`, i.e. a review's section is computed from its own school_id. infra/openfga/model.fga define …

#### ~~GetAdminStats has no school scoping and caches under a scope-free key, unlike every other admin list~~

原报 P2　`server/internal/modules/course/review/repository_admin.go:86`　票数 0/1

> **驳回**（high 置信度）：The failure scenario cannot occur in current code, and the codebase contains an explicit structural guard that blocks it even under the change the finding anticipates.

1. Route gate — server/internal/modules/course/review/handler.go:154-162 registers `httputil.RouteHandlers(h.GetAdminStats, authMiddleware, h.adminAuthorizers.Entry, h.adminAuthorizers.DashboardView)`. This is the only registration of GetAdminStats in the repo (grep found handler.go:157 plus tests only). The only production construction of review.AdminAuthorizers is server/internal/app/admin_authorizers.go:69-79, where `DashboardView: rbac.RequireGlobalCapability(capability.AdminDashboardView)` is non-nil, so the nil-skipping in httputil.AppendRouteMiddlewares (routes.go:14-20) cannot drop it.

2. RequireGlobalCapability requires a GLOBAL (unscoped) grant, not just possession of the capability. server/internal/modules/rbac/middleware.go:59-61 documents exactly this ("作用域能力（例如 school_admin 的 school-scoped grant）不会通过此检查"), and server/internal/platform/authorization/service.go:83-90 implements it as `capability.HasGlobal …

#### ~~RatingBar exposes the rating only through bar width and colour — no text, no progressbar semantics~~

原报 P2　`clients/web/src/components/common/RatingBar.vue:4`　票数 0/1

> **驳回**（high 置信度）：Refuted on three independent grounds.

1) The cited component is dead code, so the stated failure scenario cannot occur. `RatingBar.vue`'s only consumer is `/home/wztxy/Code/StuHelper/clients/web/src/components/business/review/DimensionBars.vue`, and a repo-wide grep (excluding node_modules/.git) finds exactly one reference to `DimensionBars` anywhere: line 29 of the auto-generated `/home/wztxy/Code/StuHelper/clients/web/src/components.d.ts` (unplugin-vue-components type shim, which is emitted from scanning the components dir, not from usage). No template, route, or test renders `<DimensionBars>` / `<dimension-bars>`. Therefore no screen-reader user "on a course page" ever encounters RatingBar. The live dimension-bar surfaces are hand-rolled duplicates that do not use RatingBar at all: `/home/wztxy/Code/StuHelper/clients/web/src/modules/review/views/CourseDetailPage.vue:121-141` and `/home/wztxy/Code/StuHelper/clients/web/src/components/business/review/SemesterStatsGrid.vue:23-41`. The finding's location and its causal chain to user impact are both wrong.

2) The "sighted user must e …

#### ~~OtpCodeInput leaves a rejected non-digit character visible in the cell~~

原报 P3　`clients/web/src/components/common/OtpCodeInput.vue:103`　票数 0/1

> **驳回**（high 置信度）：The finding's core premise — "Because the bound `:value="displayValue(index - 1)"` is unchanged, Vue skips patching the DOM property" — is false. Vue's renderer deliberately special-cases the `value` prop and force-syncs it against the live DOM on every update, precisely so that input rejected by the handler is reverted.

Verified in the installed Vue source (/home/wztxy/Code/StuHelper/bots/koishi/node_modules/@vue/runtime-core, version 3.5.32; clients/web/package.json pins "vue": "^3.5.13", same semantics, and this behavior has existed since Vue 3.0):

1. Optimized PROPS fast path, dist/runtime-core.cjs.js:5758-5766 — `if (next !== prev || key === "value") { hostPatchProp(...) }`. The `|| key === "value"` clause patches value even when the vnode prop did not change.
2. Full-props path, dist/runtime-core.cjs.js:5831-5841 — the generic loop explicitly skips value (`if (next !== prev && key !== "value")`) and then unconditionally calls `hostPatchProp(el, "value", ...)` under `if ("value" in newProps)`.
3. @vue/runtime-dom dist/runtime-dom.cjs.js:572-582 (patchDOMProp) compares against …

#### ~~Join/admission hostnames are hardcoded in the router with no env override, so the whole admission flow 404s on any other domain~~

原报 P2　`clients/web/src/router/join-domain.ts:1`　票数 0/1

> **驳回**（high 置信度）：The code reads as the finding describes, but the behavior is deliberate, documented, and contract-tested — not accidental config hardcoding — and the proposed fix would break a production smoke contract.

1. The literals cover every supported host. /home/wztxy/Code/StuHelper/clients/web/src/router/join-domain.ts:1-4 pins `join.stuhelper.com` (prod) and `join.localhost` (dev). Dev bootstrap points the backend link generator at exactly that dev host: /home/wztxy/Code/StuHelper/infra/ops/init-dev-env.sh:464 `ensure_dev_default "ADMISSION_PUBLIC_BASE_URL" ... "http://join.localhost:3000"`, with matching CORS_ORIGINS (:367) and CASDOOR_ADDITIONAL_REDIRECT_URIS (:414) entries for `join.localhost`. In both supported environments `currentHostname()` IS in the Set, so no QQ admission link 404s.

2. The 404-outside-join-host is an intentional, production-asserted requirement. /home/wztxy/Code/StuHelper/docs/design/admission-flow.md:19 "群内公开链接只使用 https://join.stuhelper.com/verify/<code>" and :49 "入群流程只在 join.stuhelper.com 闭环". The negative direction is explicitly smoke-tested: /home/wztxy/Code/ …

#### ~~"继续入群认证" login button does not force re-authentication, so an existing SSO session silently signs the user in as the wrong account~~

原报 P3　`clients/web/src/modules/admission/views/AdmissionPage.vue:964`　票数 0/1

> **驳回**（high 置信度）：Read the cited code and its callers.

1) The cited line is accurate: `clients/web/src/modules/admission/views/AdmissionPage.vue:964-966` is `function startLogin() { void auth.login(currentAdmissionURL()) }`, and `auth.login` -> `startLoginFlow(redirect)` with `prompt`/`maxAge` undefined (`clients/web/src/stores/auth.ts:386-390`, options forwarded at :335-337). So the factual claim "no prompt=login" is true. Everything after that is where the finding falls apart.

2) The primary case this state exists to serve is the *same* user returning to a consumed link. `consumedTokenNeedsLogin` is set when the token is already bound AND the local StuHelper session probe failed (AdmissionPage.vue:694-697 and :843-846). The overwhelmingly common cause is an expired/cleared local StuHelper session in the same browser whose Casdoor SSO session still belongs to the original account. In that case a plain authorize silently resumes account A, `resumeConsumedTokenSession` succeeds, and the user continues authentication with zero friction — exactly what `docs/design/admission-flow.md:50` prescribes ("消费后 …

#### ~~Developer Connect page falls back to the web app's own origin (then a hardcoded sso.stuhelper.com) as the OIDC issuer, publishing endpoint URLs that point at the wrong host~~

原报 P3　`clients/web/src/modules/open-platform/connectEndpoints.ts:34`　票数 0/1

> **驳回**（high 置信度）：The code is as quoted (clients/web/src/modules/open-platform/connectEndpoints.ts:1 defines DEFAULT_SSO_ISSUER; :33-40 falls back configured -> currentOrigin -> DEFAULT_SSO_ISSUER; both callers pass window.location.origin as second candidate: views/ConnectPage.vue:47-51, components/ConnectEndpointsPanel.vue:62-67). But the failure premise -- "any build/deploy where VITE_SSO_URL is unset or empty" -- does not exist here. Every supported build/run path injects it with empty-safe defaults: clients/web/Dockerfile:9,16 `ARG VITE_SSO_URL=https://sso.stuhelper.com` + `ENV VITE_SSO_URL=${VITE_SSO_URL}`; docker-compose.yml:532 `VITE_SSO_URL=${WEB_VITE_SSO_URL:-http://localhost:8085}` (`:-` also covers empty); .github/workflows/publish-images.yml:119 `VITE_SSO_URL=${{ vars.WEB_VITE_SSO_URL || 'https://sso.stuhelper.com' }}` (empty GH var is falsy -> default); .env.example:328 and .env.prod.example:246 ship the key, and infra/ops/init-prod-env.sh:417 / infra/ops/init-dev-env.sh:466 actively ensure_*_default it (contract tests assert the rewrite), so a generated env file cannot lack it; CI (.gith …


## 5. 主对话独立发现

审计 agent 未覆盖，由主对话追查得出。

### I-1. 评课审核范围 fail-open：无授权 school_admin 可见全部学校数据

`server/internal/modules/course/review/admin_scope.go:136`　P1　authorization

**证据**

```
schoolIDs()          super_admin -> nil ；无授权 admin -> []int64{}（空但非 nil）
repository_admin.go:53        if len(schoolIDs) > 0 { ... WHERE r.school_id = ANY($1) }
repository_content_flag.go:87 if len(schoolIDs) > 0 { ... }
admin_cache_key.go:17-20     schoolIDs == nil -> "all" ；len == 0 -> "none"   ← 此处区分了两者
```

**失败场景**

用户 roles claim 含 `school_admin`，但 OpenFGA 无任何 `effective_admin` school tuple。
`ResolveRoleScopes` 因 `len(scopes)==0` 返回 nil，`orgScopedRoles` 保持 nil，`schoolIDs()` 返回空切片；repository 的 `len() > 0` 为假，**不加 school 过滤**。
`requireModerationRole()` 只检查角色名不检查作用域，该用户因此可列出全部学校的 review / report / flagged 内容。影响 `admin.go:34`、`admin.go:118`、`handler_content_flag.go:19`。

同文件 cache key 把 nil 映射 `"all"`、空切片映射 `"none"`，证明预期语义是「空 = 无权限」，仅 repository 层未遵守。

**修复方案**

让「不限范围」与「无范围」在类型上不可混淆：`schoolIDs()` 返回 `([]int64, bool)` 或显式 unrestricted 标记。非 super_admin 且集合为空时 repository 必须返回空结果集，而非省略过滤条件。
补集成测试断言无授权 moderation 角色返回 0 条。

> 说明：此问题正属于第一轮失败的 backend-authz 维度，是覆盖缺口造成实际漏报的直接证据。

### I-2. 21 个后端读取的环境变量未出现在任何 env 模板

`.env.example` / `.env.prod.example`　P2　configuration-surface

`server/internal/pkg/config` 读取 187 个环境变量，其中 21 个两个模板都未定义：

```
AWS_CA_BUNDLE                          CASDOOR_ADMIN_SCOPES
CASDOOR_APP_PROVISIONING_CERTIFICATE   CASDOOR_BOOTSTRAP_MODE
CASDOOR_INTROSPECTION_ENDPOINT         CASDOOR_ROLE_SYNC_CERTIFICATE
CASDOOR_UNIAPP_SCOPES                  CASDOOR_USER_LOOKUP_CERTIFICATE
CASDOOR_USER_PROFILE_CERTIFICATE       CASDOOR_WEB_SCOPES
DB_SSL_CERT                            DB_SSL_KEY
EMAIL_PROVIDER_POLICY                  FGA_SKIP_SCHOOL_TUPLES
GIN_MODE                               LOG_ENVIRONMENT
LOG_SERVICE_NAME                       LOG_SERVICE_VERSION
REDIS_TLS_CERT                         REDIS_TLS_KEY
STUHELPER_REDIS_INTEGRATION
```

复核结论：`EXTERNAL_POSTGRES_ALLOW_PLAINTEXT` 与 `FGA_SKIP_SCHOOL_TUPLES` **已有正确守卫**
（前者在 `validation.go:248` 于 production 被拒绝；后者仅作用于 `cmd/fga-setup` 引导工具，
不进入运行时授权路径）。因此这是配置面完备性问题，不是安全漏洞。

**修复方案**：补入模板并注明用途与默认值；增加 CI 检查比对 `config` 包读取集合与模板定义集合。

### I-3. 契约测试锁死了被测代码的缺陷

`infra/ops/tests/deploy-bundle-contract.sh`　P1　test-design

旧契约测试断言 `assert_contains "--exclude='\.env\*'"`——即断言那行**导致 P1 缺陷的代码存在**。
部署包缺失 env 模板的问题因此在 CI 中永远是绿的：测试把缺陷锁住了。

这是文本匹配型测试的固有风险。已重写为行为测试：实际打包并断言两个模板在、其他 `.env*` 不在。

## 6. 未完成验证

第二轮有 3 条发现未获验证结论（验证 agent 未返回）。这些条目**未经确认，不应直接采纳**，
需要单独复核后再决定。

## 7. 关于修复实施

本 worktree 有其他 agent 并行修改，因此本报告不对实施状态做断言——git 历史无法可靠归属到
具体执行者。可确认的事实：

- 本次审计的两轮 subagent **未执行任何 `git commit`**（transcript 中改动性 git 命令共 2 条：
  一条 `git checkout --` 撤销自身实验性改动，一条 `git stash list` 只读）。
- subagent 的写操作全部指向 `/tmp/` 探针或仓库内 `zz*` / `*.tmp.*` 一次性验证文件。
  这些临时文件需要清理，不应进入提交。
- 主对话直接修复了 3 项：Admin 落地页 404 循环、部署包 env 模板缺失、上述契约测试反模式。

每条确认问题的修复方案已随条目列出，且经过验证 agent 复核（含「修复是否会破坏其他调用方」这一视角）。实施时应以此为依据，并为每条补上能捕获该缺陷的回归测试——I-3 说明缺少行为测试会让缺陷在 CI 中长期隐形。
