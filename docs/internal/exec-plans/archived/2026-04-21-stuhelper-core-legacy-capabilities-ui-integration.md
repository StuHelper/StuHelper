---
type: internal
audience: maintainers, backend-dev, frontend-dev
status: archived
authoritative-source: this file
last-verified: 2026-04-21
---

# StuHelper Core Legacy Capabilities UI Integration Implementation Plan

> 归档说明：本计划已由后续 core console 与 Koishi 重构工作吸收，保留为历史实施记录。下方 checklist 已闭环，不再作为当前项目待办来源。

**Goal:** 在 `stuhelper-core` 中把旧 StuHelper 的身份认证、处置中心、治理配置能力并入当前 grouphelper 风格控制台，并提供可刷新、可深链、可测试的页面域接口与 UI。

**Architecture:** 后端新增页面域聚合 service 和 console listener，前端用 query-synced 顶栏壳层承载 `Dashboard / Config / Identity / Review` 升级。纯数据映射、导航解析、状态格式化优先拆成可测试的 TypeScript 模块，再由 Vue 页面消费，避免把页面逻辑重新堆进单文件组件。

**Tech Stack:** Koishi plugin console, Vue 3 `<script setup lang="ts">`, TypeScript, `@koishijs/client`, Koishi database, Node `node:test`.

---

### Task 1: 建立页面域类型与导航状态基线

**Files:**
- Create: `bots/koishi/plugins/stuhelper-core/client/models/navigation.ts`
- Create: `bots/koishi/plugins/stuhelper-core/client/models/views.ts`
- Create: `bots/koishi/plugins/stuhelper-core/client/models/formatters.ts`
- Test: `bots/koishi/plugins/stuhelper-core/client/models/navigation.test.ts`

- [x] **Step 1: Write the failing test**

```ts
import assert from 'node:assert/strict'
import test from 'node:test'

import { createConsoleQuery, parseConsoleQuery } from './navigation'

test('parseConsoleQuery restores page workspace and entity context', () => {
  const params = new URLSearchParams('view=review&workspace=pending&guildId=1001&memberId=2002&itemId=rv-1&keyword=kick')
  const state = parseConsoleQuery(params)

  assert.deepEqual(state, {
    view: 'review',
    workspace: 'pending',
    guildId: '1001',
    memberId: '2002',
    itemId: 'rv-1',
    tab: null,
    keyword: 'kick',
  })
})

test('createConsoleQuery omits empty optional fields', () => {
  const params = createConsoleQuery({
    view: 'dashboard',
    workspace: null,
    guildId: null,
    memberId: null,
    itemId: null,
    tab: null,
    keyword: '',
  })

  assert.equal(params.toString(), 'view=dashboard')
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `node --test bots/koishi/plugins/stuhelper-core/client/models/navigation.test.ts`

Expected: FAIL with `Cannot find module './navigation'`

- [x] **Step 3: Write minimal implementation**

```ts
export type ConsoleViewId = 'dashboard' | 'config' | 'warns' | 'blacklist' | 'identity' | 'review' | 'roles' | 'logs' | 'chat' | 'subscriptions' | 'settings'

export interface ConsoleNavigationState {
  view: ConsoleViewId
  workspace: string | null
  guildId: string | null
  memberId: string | null
  itemId: string | null
  tab: string | null
  keyword: string
}

export function parseConsoleQuery(params: URLSearchParams): ConsoleNavigationState {
  return {
    view: isView(params.get('view')) ? params.get('view')! : 'dashboard',
    workspace: params.get('workspace'),
    guildId: params.get('guildId'),
    memberId: params.get('memberId'),
    itemId: params.get('itemId'),
    tab: params.get('tab'),
    keyword: params.get('keyword') ?? '',
  }
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `node --test bots/koishi/plugins/stuhelper-core/client/models/navigation.test.ts`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add bots/koishi/plugins/stuhelper-core/client/models/navigation.ts bots/koishi/plugins/stuhelper-core/client/models/views.ts bots/koishi/plugins/stuhelper-core/client/models/formatters.ts bots/koishi/plugins/stuhelper-core/client/models/navigation.test.ts
git commit -m "feat(stuhelper-core): add console navigation model"
```

### Task 2: 新增后端页面域聚合 service 与 listener

**Files:**
- Create: `bots/koishi/plugins/stuhelper-core/src/core/services/page-types.ts`
- Create: `bots/koishi/plugins/stuhelper-core/src/core/services/dashboard-page.service.ts`
- Create: `bots/koishi/plugins/stuhelper-core/src/core/services/identity-page.service.ts`
- Create: `bots/koishi/plugins/stuhelper-core/src/core/services/review-page.service.ts`
- Create: `bots/koishi/plugins/stuhelper-core/src/core/services/config-governance.service.ts`
- Create: `bots/koishi/plugins/stuhelper-core/src/core/api/page-api.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/services/index.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/api/index.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/index.ts`
- Test: `bots/koishi/plugins/stuhelper-core/src/core/services/dashboard-page.service.test.ts`
- Test: `bots/koishi/plugins/stuhelper-core/src/core/services/identity-page.service.test.ts`
- Test: `bots/koishi/plugins/stuhelper-core/src/core/services/review-page.service.test.ts`
- Test: `bots/koishi/plugins/stuhelper-core/src/core/services/config-governance.service.test.ts`

- [x] **Step 1: Write the failing tests**

```ts
test('buildReviewPageData maps reviews, guard members and reports into ReviewWorkItem rows', async () => {
  const service = createReviewPageServiceWithFixtures()

  const data = await service.getPageData()

  assert.equal(data.items[0].kind, 'review')
  assert.ok(data.items.some((item) => item.kind === 'admission'))
  assert.ok(data.items.some((item) => item.kind === 'report'))
})

test('buildConfigGovernanceData includes template source and command policy list', async () => {
  const service = createConfigGovernanceServiceWithFixtures()

  const data = await service.getPageData()

  assert.equal(data.workspaces[0].id, 'guild-config')
  assert.equal(data.templates[0].source.label, '模板库')
  assert.equal(data.commandPolicies[0].commandId, 'report')
})
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
node --test \
  bots/koishi/plugins/stuhelper-core/src/core/services/dashboard-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/identity-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/review-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/config-governance.service.test.ts
```

Expected: FAIL with missing service modules

- [x] **Step 3: Write minimal implementation**

```ts
export class ReviewPageService {
  constructor(
    private readonly ctx: Context,
    private readonly moderationStore = new ModerationStore(ctx),
  ) {}

  async getPageData(): Promise<ReviewPageData> {
    const [reviews, reports, guardMembers, events] = await Promise.all([
      this.moderationStore.listPendingReviews(),
      this.moderationStore.listOpenReports(),
      listActiveGuardMembers(this.ctx),
      this.moderationStore.listRecentEvents(50),
    ])

    return {
      generatedAt: new Date().toISOString(),
      items: [
        ...reviews.map(toReviewItem),
        ...guardMembers.map(toAdmissionItem),
        ...reports.map((report) => toReportItem(report, events)),
      ].sort(compareWorkItemTimeDesc),
      events: events.map(serializeEvent),
    }
  }
}
```

- [x] **Step 4: Register console listeners**

```ts
ctx.console.addListener('stuhelperGroupCenter/page/dashboard', () => dashboardService.getPageData())
ctx.console.addListener('stuhelperGroupCenter/page/identity', (query) => identityService.getPageData(parseIdentityQuery(query)))
ctx.console.addListener('stuhelperGroupCenter/page/review', (query) => reviewService.getPageData(parseReviewQuery(query)))
ctx.console.addListener('stuhelperGroupCenter/page/config-governance', () => configGovernanceService.getPageData())
```

- [x] **Step 5: Run tests to verify they pass**

Run:

```bash
node --test \
  bots/koishi/plugins/stuhelper-core/src/core/services/dashboard-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/identity-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/review-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/config-governance.service.test.ts
```

Expected: PASS

- [x] **Step 6: Commit**

```bash
git add bots/koishi/plugins/stuhelper-core/src/core/services bots/koishi/plugins/stuhelper-core/src/core/api/page-api.ts bots/koishi/plugins/stuhelper-core/src/index.ts
git commit -m "feat(stuhelper-core): add page-domain console services"
```

### Task 3: 重写前端 API 层与顶栏 query-synced 壳层

**Files:**
- Create: `bots/koishi/plugins/stuhelper-core/client/composables/use-console-navigation.ts`
- Create: `bots/koishi/plugins/stuhelper-core/client/composables/use-console-pages.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/client/api.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/client/pages/index.vue`
- Modify: `bots/koishi/plugins/stuhelper-core/client/index.ts`
- Test: `bots/koishi/plugins/stuhelper-core/client/composables/use-console-navigation.test.ts`

- [x] **Step 1: Write the failing navigation test**

```ts
test('selectView writes navigation state and preserves query-backed context', () => {
  const history = createMockHistory('https://example.test/stuhelper?view=dashboard')
  const nav = createConsoleNavigation(history)

  nav.selectView('identity', { guildId: '1001', memberId: '2002' })

  assert.equal(history.url.searchParams.get('view'), 'identity')
  assert.equal(history.url.searchParams.get('guildId'), '1001')
  assert.equal(history.url.searchParams.get('memberId'), '2002')
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `node --test bots/koishi/plugins/stuhelper-core/client/composables/use-console-navigation.test.ts`

Expected: FAIL with missing composable

- [x] **Step 3: Replace old local `currentView` shell**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useConsoleNavigation } from '../composables/use-console-navigation'
import { useConsolePages } from '../composables/use-console-pages'

const navigation = useConsoleNavigation(window)
const pages = useConsolePages(navigation.state)
const activeComponent = computed(() => pages.resolve(navigation.state.value.view))
</script>
```

- [x] **Step 4: Add overflow-aware top nav**

```ts
const PRIMARY_MENU_IDS: ConsoleViewId[] = ['dashboard', 'config', 'warns', 'blacklist', 'identity', 'review', 'roles', 'logs']
const OVERFLOW_MENU_IDS: ConsoleViewId[] = ['chat', 'subscriptions', 'settings']
```

- [x] **Step 5: Run test to verify it passes**

Run:

```bash
node --test bots/koishi/plugins/stuhelper-core/client/composables/use-console-navigation.test.ts
```

Expected: PASS

- [x] **Step 6: Commit**

```bash
git add bots/koishi/plugins/stuhelper-core/client/api.ts bots/koishi/plugins/stuhelper-core/client/composables bots/koishi/plugins/stuhelper-core/client/pages/index.vue bots/koishi/plugins/stuhelper-core/client/index.ts
git commit -m "feat(stuhelper-core): add query-synced console shell"
```

### Task 4: 升级 Dashboard 与 Config 页面

**Files:**
- Create: `bots/koishi/plugins/stuhelper-core/client/models/dashboard.ts`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/dashboard/DashboardTodoPanel.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/dashboard/DashboardShortcutPanel.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/config/ConfigWorkspaceTabs.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/config/GuardTemplatesPanel.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/config/GuardBindingsPanel.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/config/CommandPoliciesPanel.vue`
- Modify: `bots/koishi/plugins/stuhelper-core/client/components/DashboardView.vue`
- Modify: `bots/koishi/plugins/stuhelper-core/client/components/ConfigView.vue`
- Test: `bots/koishi/plugins/stuhelper-core/client/models/dashboard.test.ts`

- [x] **Step 1: Write the failing dashboard model test**

```ts
test('buildDashboardModel exposes todo rows and jump targets', () => {
  const model = buildDashboardModel(createDashboardFixture())

  assert.equal(model.todoRows[0].target.view, 'review')
  assert.equal(model.shortcuts[1].target.view, 'identity')
  assert.equal(model.statusBand.length > 0, true)
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `node --test bots/koishi/plugins/stuhelper-core/client/models/dashboard.test.ts`

Expected: FAIL with missing model

- [x] **Step 3: Implement Dashboard summary panels**

```vue
<DashboardTodoPanel
  :rows="dashboardModel.todoRows"
  @select="navigation.jumpTo($event)"
/>
<DashboardShortcutPanel
  :items="dashboardModel.shortcuts"
  @select="navigation.jumpTo($event)"
/>
```

- [x] **Step 4: Split Config into workspaces**

```ts
const configWorkspaces = [
  { id: 'guild-config', label: '群配置' },
  { id: 'templates', label: '模板库' },
  { id: 'bindings', label: '群绑定' },
  { id: 'command-policies', label: '命令策略' },
] as const
```

- [x] **Step 5: Run tests to verify they pass**

Run:

```bash
node --test bots/koishi/plugins/stuhelper-core/client/models/dashboard.test.ts
```

Expected: PASS

- [x] **Step 6: Commit**

```bash
git add bots/koishi/plugins/stuhelper-core/client/models/dashboard.ts bots/koishi/plugins/stuhelper-core/client/components/dashboard bots/koishi/plugins/stuhelper-core/client/components/config bots/koishi/plugins/stuhelper-core/client/components/DashboardView.vue bots/koishi/plugins/stuhelper-core/client/components/ConfigView.vue
git commit -m "feat(stuhelper-core): upgrade dashboard and config workspaces"
```

### Task 5: 新增 Identity 与 Review 主页面

**Files:**
- Create: `bots/koishi/plugins/stuhelper-core/client/models/identity.ts`
- Create: `bots/koishi/plugins/stuhelper-core/client/models/review.ts`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/IdentityView.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/ReviewView.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/identity/IdentityMemberTable.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/identity/IdentityMemberDetail.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/review/ReviewWorkItemTable.vue`
- Create: `bots/koishi/plugins/stuhelper-core/client/components/review/ReviewInspector.vue`
- Test: `bots/koishi/plugins/stuhelper-core/client/models/identity.test.ts`
- Test: `bots/koishi/plugins/stuhelper-core/client/models/review.test.ts`

- [x] **Step 1: Write the failing tests**

```ts
test('buildIdentityModel groups members by guild-first scope and resolves selected member detail', () => {
  const model = buildIdentityModel(createIdentityFixture(), {
    guildId: '1001',
    memberId: '2002',
  })

  assert.equal(model.selectedGuildId, '1001')
  assert.equal(model.selectedMember?.memberId, '2002')
  assert.equal(model.detail.cards[0].label, '认证状态')
})

test('buildReviewModel preserves selected item when filters narrow the list', () => {
  const model = buildReviewModel(createReviewFixture(), {
    status: 'pending',
    keyword: 'kick',
    itemId: 'rv-1',
  })

  assert.equal(model.selectedItem?.id, 'rv-1')
  assert.equal(model.filteredItems.some((item) => item.id === 'rv-1'), true)
})
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
node --test \
  bots/koishi/plugins/stuhelper-core/client/models/identity.test.ts \
  bots/koishi/plugins/stuhelper-core/client/models/review.test.ts
```

Expected: FAIL with missing model modules

- [x] **Step 3: Implement the two new top-level pages**

```vue
<IdentityView
  v-if="navigation.state.value.view === 'identity'"
  :page-data="pages.identity.data.value"
  :navigation="navigation"
/>

<ReviewView
  v-if="navigation.state.value.view === 'review'"
  :page-data="pages.review.data.value"
  :navigation="navigation"
/>
```

- [x] **Step 4: Wire review actions through existing moderation listeners**

```ts
await runReviewAction({ reviewId: item.id, action: 'execute', note })
await reloadReviewPage()
navigation.replaceContext({ itemId: nextItemId ?? null })
```

- [x] **Step 5: Run tests to verify they pass**

Run:

```bash
node --test \
  bots/koishi/plugins/stuhelper-core/client/models/identity.test.ts \
  bots/koishi/plugins/stuhelper-core/client/models/review.test.ts
```

Expected: PASS

- [x] **Step 6: Commit**

```bash
git add bots/koishi/plugins/stuhelper-core/client/models/identity.ts bots/koishi/plugins/stuhelper-core/client/models/review.ts bots/koishi/plugins/stuhelper-core/client/components/IdentityView.vue bots/koishi/plugins/stuhelper-core/client/components/ReviewView.vue bots/koishi/plugins/stuhelper-core/client/components/identity bots/koishi/plugins/stuhelper-core/client/components/review
git commit -m "feat(stuhelper-core): add identity and review pages"
```

### Task 6: 完整验证、运行时烟雾测试与收尾清理

**Files:**
- Modify: `bots/koishi/plugins/stuhelper-core/client/pages/index.vue`
- Modify: `bots/koishi/plugins/stuhelper-core/client/styles/components.css`
- Modify: `bots/koishi/plugins/stuhelper-core/client/styles/variables.css`
- Modify: `bots/koishi/plugins/stuhelper-core/src/browser-entry.test.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/runtime-contract.test.ts`

- [x] **Step 1: Add regression tests for browser entry and runtime menu registration**

```ts
test('stuhelper-core browser page remains registered at /stuhelper', async () => {
  const content = await readFile(new URL('../client/index.ts', import.meta.url), 'utf8')
  assert.match(content, /path:\s*'\/stuhelper'/)
})
```

- [x] **Step 2: Run targeted unit tests**

Run:

```bash
node --test \
  bots/koishi/plugins/stuhelper-core/client/models/navigation.test.ts \
  bots/koishi/plugins/stuhelper-core/client/models/dashboard.test.ts \
  bots/koishi/plugins/stuhelper-core/client/models/identity.test.ts \
  bots/koishi/plugins/stuhelper-core/client/models/review.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/dashboard-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/identity-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/review-page.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/core/services/config-governance.service.test.ts \
  bots/koishi/plugins/stuhelper-core/src/browser-entry.test.ts \
  bots/koishi/plugins/stuhelper-core/src/runtime-contract.test.ts
```

Expected: PASS

- [x] **Step 3: Run package build**

Run: `cd bots/koishi && corepack yarn workspace koishi-plugin-stuhelper-core build`

Expected: exit 0 and regenerated `dist/index.js` / `dist/style.css`

- [x] **Step 4: Run startup smoke**

Run: `cd bots/koishi && node scripts/startup-smoke.mjs`

Expected: listens on `5140`, console page entry available, no startup crash

- [x] **Step 5: Commit**

```bash
git add bots/koishi/plugins/stuhelper-core docs/internal/exec-plans/archived/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration.md
git commit -m "feat(stuhelper-core): integrate legacy capabilities into core console"
```

## Self-Review

### Spec coverage

- 顶栏模块化菜单与 `More` 溢出策略：Task 3
- `Dashboard` 变成“看板 + 待办 + 分发台”：Task 4
- `Config` 四工作区：Task 4
- `Identity` 群优先视角：Task 5
- `ReviewWorkItem` 联合模型与动作上下文：Task 2 + Task 5
- query-synced `ConsoleNavigationState`：Task 1 + Task 3
- 页面域聚合 service：Task 2
- 错误显式暴露与动作后上下文保持：Task 2 + Task 5
- 模型测试、交互测试、启动验证：Task 1/2/4/5/6

### Placeholder scan

- 未使用 `TODO` / `TBD` / “类似 Task N”
- 每个任务都给出文件、测试命令、最小实现轮廓和提交点

### Type consistency

- 导航状态统一使用 `ConsoleNavigationState`
- 处置页主模型统一使用 `ReviewWorkItem`
- 页面查询 listener 与前端 API 都按页面域命名：`dashboard` / `identity` / `review` / `config-governance`
