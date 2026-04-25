import { test, expect } from './fixtures/auth'
import type { ConsoleMessage, Page } from '@playwright/test'

/**
 * P0b：StuHelper 群管中心 11 个 view 的端到端导航回归基线。
 *
 * 架构要点：
 * - 所有 test 共享一个 worker-scoped page（fixture 完成登录 + SPA warm-up）
 * - 每个 test 通过 click nav button 切 view，**不**用 page.goto——避免重载
 *   引发 Koishi page 注册异步竞态（P0b 第一版 40% flake 的根因）
 * - 第一个 test 验证 nav click 切换 URL 工作
 * - 11 个 per-view test 各自 click 目标 view 的 nav button，断言：
 *     - URL 落定后包含 view=<id>
 *     - view-specific anchor 元素出现（防"渲染错 view 但壳还在"）
 *     - 无 pageerror、无 console.error/warning（按 allowlist 过滤）
 *
 * 不使用 test.describe.configure({ mode: 'serial' })：workers:1 已保证顺序，
 * 任一 view fail 不阻塞后续，方便看到全量基线状态。
 */

interface ViewAnchor {
  /**
   * 该 view 渲染时一定存在的 selector。要求 view-specific——
   * 当 useConsolePages.resolve() 因未知 view ID 静默 fallback 到 dashboard 时，
   * 此 selector 不应出现在被错误渲染出来的组件里。
   */
  readonly selector: string
  /** 可选：要求 selector 元素的文本必须包含此字符串。无则只验存在性。 */
  readonly text?: string
}

interface ViewSpec {
  readonly id: string
  readonly label: string
  readonly anchor: ViewAnchor
}

/**
 * 11 个 view 的专属 anchor。来源：
 * - 大多数 view 用 `WorkspaceHead` primitive，H1 是 `.sh-workspace-head__title`，文本 == title prop
 * - dashboard 用自定义 `.sh-dashboard__title`（H1 文本"控制台总览"，与 nav label "仪表盘"不一致）
 * - chat / roles / settings 没用 WorkspaceHead，用它们的根容器 class
 *
 * anchor 选择标准：必须能在 dashboard fallback 场景下区分"是不是 view 真的渲染对了"。
 */
const VIEWS: readonly ViewSpec[] = [
  { id: 'dashboard', label: '仪表盘', anchor: { selector: '.sh-dashboard__title', text: '控制台总览' } },
  { id: 'config', label: '配置治理', anchor: { selector: '.sh-workspace-head__title', text: '配置治理' } },
  { id: 'warns', label: '警告记录', anchor: { selector: '.sh-workspace-head__title', text: '警告记录' } },
  { id: 'blacklist', label: '黑名单', anchor: { selector: '.sh-workspace-head__title', text: '黑名单' } },
  { id: 'identity', label: '身份认证', anchor: { selector: '.sh-workspace-head__title', text: '身份认证' } },
  { id: 'review', label: '处置中心', anchor: { selector: '.sh-workspace-head__title', text: '处置中心' } },
  { id: 'roles', label: '角色权限', anchor: { selector: '.roles-view-container' } },
  { id: 'logs', label: '日志检索', anchor: { selector: '.sh-workspace-head__title', text: '日志检索' } },
  { id: 'chat', label: '实时聊天', anchor: { selector: '.chat-view' } },
  { id: 'subscriptions', label: '订阅管理', anchor: { selector: '.sh-workspace-head__title', text: '订阅管理' } },
  { id: 'settings', label: '设置', anchor: { selector: '.settings-view' } },
]

/**
 * 当真实环境出现需要忽略的合法 console message 时，在这里加显式条目。
 * 规则：
 * - 必须带注释说明为什么忽略以及在哪个 Koishi/Element Plus 版本观察到
 * - 不允许泛化正则（如 /Vue Router/、/Element Plus/）
 * - 必须包含具体 message 片段，缩小范围
 */
const CONSOLE_ALLOWLIST: readonly RegExp[] = [
  // Koishi Console 通过 ctx.console.addEntry 把 stuhelper-core 的 /stuhelper page
  // 异步注入客户端 Vue Router。在 page 注册完成之前，浏览器对该路径的 navigation
  // 会触发 Vue Router 的 "No match found" 警告。fixture warm-up 已等到 page 渲染，
  // warning 不影响 view 实际渲染（spec 的 anchor 断言已验证）。
  //
  // 观察版本：@koishijs/plugin-console 5.30.4 + koishi 4.18.7（2026-04-25）。
  //
  // 范围：仅放过 path 等于 /stuhelper 或 /stuhelper?<query>。其他路径下同类
  // No match 仍会失败，保留对真实路由错误的灵敏度。
  /^\[Vue Router warn\]: No match found for location with path "\/stuhelper(\?[^"]*)?"$/,
]

test('TopNavigation click switches between views', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  // 先点击第一个 view（dashboard），让导航有一个稳定起点
  await clickNavTab(page, VIEWS[0].label)
  await expect(page).toHaveURL(/[?&]view=dashboard($|&)/, { timeout: 5_000 })

  // 切到第二个 view，验证 nav 真的能切
  await clickNavTab(page, VIEWS[1].label)
  await expect(page).toHaveURL(/[?&]view=config($|&)/, { timeout: 5_000 })

  tracker.assertClean()
})

for (const view of VIEWS) {
  test(`view "${view.id}" renders with view-specific anchor`, async ({ loggedInPage: page }) => {
    await using tracker = createTracker(page)

    await clickNavTab(page, view.label)

    // URL 断言抓"切换没成功"场景：如果 nav click 失败 URL 不会包含 view=<id>
    await expect(page).toHaveURL(new RegExp(`[?&]view=${view.id}($|&)`), { timeout: 5_000 })

    // view-specific anchor 抓"渲染错 view 但壳还在"场景：
    // useConsolePages.resolve() 对未知 view 静默 fallback 到 dashboard，
    // 各 view 的专属 selector 能区分真实渲染目标
    const anchorBase = page.locator(view.anchor.selector)
    const anchor = view.anchor.text
      ? anchorBase.filter({ hasText: view.anchor.text })
      : anchorBase
    await expect(anchor.first()).toBeVisible({ timeout: 10_000 })

    tracker.assertClean()
  })
}

/**
 * 通过 nav button 切到指定 view。
 * 优先点击直接可见的 .nav-tab；如果该 view 被折进 More 菜单，则展开 More 后点击 .more-menu__item。
 */
async function clickNavTab(page: Page, label: string): Promise<void> {
  const directTab = page.locator('.nav-tab', { hasText: label })
  if ((await directTab.count()) > 0 && (await directTab.first().isVisible())) {
    await directTab.first().click()
    return
  }
  // fallback：进 More 菜单
  await page.locator('.nav-tab', { hasText: 'More' }).first().click()
  await page.locator('.more-menu__item', { hasText: label }).first().click()
}

interface ConsoleIssue {
  readonly type: 'error' | 'warning'
  readonly text: string
  readonly url: string
}

interface Tracker extends AsyncDisposable {
  readonly issues: readonly ConsoleIssue[]
  readonly errors: readonly Error[]
  assertClean(): void
}

function createTracker(page: Page): Tracker {
  const issues: ConsoleIssue[] = []
  const errors: Error[] = []

  const onConsole = (message: ConsoleMessage) => {
    const type = message.type()
    if (type !== 'error' && type !== 'warning') return
    const text = message.text()
    if (CONSOLE_ALLOWLIST.some((pattern) => pattern.test(text))) return
    issues.push({ type, text, url: message.location().url })
  }
  const onPageError = (error: Error) => {
    errors.push(error)
  }

  page.on('console', onConsole)
  page.on('pageerror', onPageError)

  return {
    issues,
    errors,
    assertClean() {
      expect(
        errors,
        `unexpected pageerror:\n${errors.map((error) => `  ${error.message}`).join('\n')}`,
      ).toHaveLength(0)
      expect(
        issues,
        `unexpected console output:\n${issues
          .map((issue) => `  [${issue.type}] ${issue.text} (${issue.url})`)
          .join('\n')}`,
      ).toHaveLength(0)
    },
    async [Symbol.asyncDispose]() {
      page.off('console', onConsole)
      page.off('pageerror', onPageError)
    },
  }
}
