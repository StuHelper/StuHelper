import { test, expect } from './fixtures/auth'
import type { ConsoleMessage, Page, Request, Response } from '@playwright/test'

/**
 * P0b：StuHelper 群管中心 view 端到端导航回归基线（NavRail + Shell 重构后）。
 *
 * 架构要点：
 * - 所有 test 共享一个 worker-scoped page（fixture 完成登录 + SPA warm-up）。
 * - 切 view 通过点击 NavRail 的 `<button class="sh-rail__item" :title="...">`，
 *   不用 page.goto 避免 Koishi page 注册的异步竞态。
 * - 第一个 test 验证 NavRail click → URL hash 切换工作。
 * - 11 个 per-view test 各自 click 目标 view 的 NavRail button，断言：
 *     - URL 落定后 hash 包含 view id；
 *     - view-specific anchor 元素出现（防"渲染错 view 但壳还在"）；
 *     - 无 pageerror、无 console.error/warning（按 allowlist 过滤）。
 * - chat 不在 NavRail，单独一个 test：通过 CommandBar 的 ⌘/ 按钮打开 ChatDock
 *   浮窗，断言 `.chat-view` 出现在 dock body 内。
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
  /** NavRail 中显示的 label（同时是 button 的 title 属性）。 */
  readonly label: string
  readonly anchor: ViewAnchor
}

/**
 * NavRail 中可达的 11 个 view（chat 不在 rail 中，独立 dock 测）。
 *
 * label = NavRail 内显示文本 = button[title]；可能与 view 内部 H1 不同
 * （如 identity 的 nav label "限制中" vs 内部 H1 "身份认证"）。
 *
 * anchor 选择标准：必须能在 dashboard fallback 场景下区分"是不是 view 真的渲染对了"。
 */
const VIEWS: readonly ViewSpec[] = [
  { id: 'dashboard', label: '总览', anchor: { selector: '.sh-dashboard__title', text: '控制台总览' } },
  { id: 'review', label: '处置中心', anchor: { selector: '.sh-workspace-head__title', text: '处置中心' } },
  { id: 'identity', label: '限制中', anchor: { selector: '.sh-workspace-head__title', text: '身份认证' } },
  { id: 'warns', label: '警告记录', anchor: { selector: '.sh-workspace-head__title', text: '警告记录' } },
  { id: 'blacklist', label: '黑名单', anchor: { selector: '.sh-workspace-head__title', text: '黑名单' } },
  { id: 'config', label: '群组配置', anchor: { selector: '.sh-workspace-head__title', text: '配置治理' } },
  { id: 'roles', label: '角色权限', anchor: { selector: '.roles-view-container' } },
  { id: 'settings', label: '全局设置', anchor: { selector: '.settings-view' } },
  { id: 'logs', label: '日志检索', anchor: { selector: '.sh-workspace-head__title', text: '日志检索' } },
  { id: 'subscriptions', label: '推送订阅', anchor: { selector: '.sh-workspace-head__title', text: '订阅管理' } },
  { id: 'system', label: '系统 / 缓存', anchor: { selector: '.sh-workspace-head__title', text: '系统 / 缓存' } },
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
  // 范围：仅放过 path 等于 /stuhelper 或 /stuhelper?<query>。hash 不参与 path
  // 匹配；其他路径下同类 No match 仍会失败，保留对真实路由错误的灵敏度。
  /^\[Vue Router warn\]: No match found for location with path "\/stuhelper(\?[^"]*)?"$/,
]

const CRITICAL_RESOURCE_TYPES = new Set(['document', 'font', 'image', 'script', 'stylesheet'])

test('NavRail click switches between views', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  // 先点击第一个 view（dashboard），让导航有一个稳定起点
  await clickNavRail(page, VIEWS[0].label)
  await expect(page).toHaveURL(/#dashboard($|\?)/, { timeout: 5_000 })

  // 切到下一个 view，验证 NavRail 真的能切
  await clickNavRail(page, VIEWS[1].label)
  await expect(page).toHaveURL(new RegExp(`#${VIEWS[1].id}($|\\?)`), { timeout: 5_000 })

  tracker.assertClean()
})

for (const view of VIEWS) {
  test(`view "${view.id}" renders with view-specific anchor`, async ({ loggedInPage: page }) => {
    await using tracker = createTracker(page)

    await clickNavRail(page, view.label)

    // URL 断言抓"切换没成功"场景：如果 NavRail click 失败 URL hash 不会包含 view id
    await expect(page).toHaveURL(new RegExp(`#${view.id}($|\\?)`), { timeout: 5_000 })

    // view-specific anchor 抓"渲染错 view 但壳还在"场景
    const anchorBase = page.locator(view.anchor.selector)
    const anchor = view.anchor.text
      ? anchorBase.filter({ hasText: view.anchor.text })
      : anchorBase
    await expect(anchor.first()).toBeVisible({ timeout: 10_000 })

    tracker.assertClean()
  })
}

test('chat dock opens via CommandBar and renders ChatView', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  // 确保从已知 view 起步，避免 dock 状态被前序 test 残留
  await clickNavRail(page, VIEWS[0].label)
  await expect(page).toHaveURL(/#dashboard($|\?)/, { timeout: 5_000 })

  // 打开 ChatDock（CommandBar 右侧 ⌘/ 按钮）
  await page.locator('.sh-cmd__chat').first().click()

  // dock 内部应该挂出 ChatView
  await expect(page.locator('.sh-dock[data-open="true"] .chat-view').first()).toBeVisible({
    timeout: 10_000,
  })

  tracker.assertClean()
})

/**
 * 通过 NavRail 的 button[title=label] 切到指定 view。
 * NavRail 收起时 label 文本 opacity:0 不可读，但 button 元素本身可点击；
 * title attribute 在两种状态下都准确反映 view label，是稳定的 selector key。
 */
async function clickNavRail(page: Page, label: string): Promise<void> {
  const button = page.locator(`.sh-rail__item[title="${label}"]`)
  await expect(button.first()).toBeAttached({ timeout: 5_000 })
  await button.first().click()
}

interface ConsoleIssue {
  readonly type: 'error' | 'warning'
  readonly text: string
  readonly url: string
}

interface ResourceIssue {
  readonly method: string
  readonly resourceType: string
  readonly url: string
  readonly status?: number
  readonly statusText?: string
  readonly failure?: string
}

interface Tracker extends AsyncDisposable {
  readonly issues: readonly ConsoleIssue[]
  readonly errors: readonly Error[]
  readonly resourceIssues: readonly ResourceIssue[]
  assertClean(): void
}

function createTracker(page: Page): Tracker {
  const issues: ConsoleIssue[] = []
  const errors: Error[] = []
  const resourceIssues: ResourceIssue[] = []

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
  const onRequestFailed = (request: Request) => {
    if (!CRITICAL_RESOURCE_TYPES.has(request.resourceType())) return
    resourceIssues.push({
      method: request.method(),
      resourceType: request.resourceType(),
      url: request.url(),
      failure: request.failure()?.errorText ?? 'failed',
    })
  }
  const onResponse = (response: Response) => {
    const request = response.request()
    if (!CRITICAL_RESOURCE_TYPES.has(request.resourceType()) || response.status() < 400) {
      return
    }
    resourceIssues.push({
      method: request.method(),
      resourceType: request.resourceType(),
      url: response.url(),
      status: response.status(),
      statusText: response.statusText(),
    })
  }

  page.on('console', onConsole)
  page.on('pageerror', onPageError)
  page.on('requestfailed', onRequestFailed)
  page.on('response', onResponse)

  return {
    issues,
    errors,
    resourceIssues,
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
      expect(
        resourceIssues,
        `unexpected critical resource failures:\n${resourceIssues
          .map((issue) => {
            const status = issue.status ? ` HTTP ${issue.status} ${issue.statusText ?? ''}` : ''
            const failure = issue.failure ? ` ${issue.failure}` : ''
            return `  ${issue.resourceType} ${issue.method} ${issue.url}${status}${failure}`
          })
          .join('\n')}`,
      ).toHaveLength(0)
    },
    async [Symbol.asyncDispose]() {
      page.off('console', onConsole)
      page.off('pageerror', onPageError)
      page.off('requestfailed', onRequestFailed)
      page.off('response', onResponse)
    },
  }
}
