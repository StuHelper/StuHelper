import { test, expect } from './fixtures/auth'
import { createTracker } from './fixtures/diagnostics'
import type { Page } from '@playwright/test'

/**
 * P0b：StuHelper 群管中心 view 端到端导航回归基线（NavRail + Shell 重构后）。
 *
 * 架构要点：
 * - 所有 test 共享一个 worker-scoped page（fixture 完成登录 + SPA warm-up）。
 * - 切 view 通过点击 NavRail 的 `<button class="sh-rail__item" :title="...">`，
 *   不用 page.goto 避免 Koishi page 注册的异步竞态。
 * - 第一个 test 验证 NavRail click → URL hash 切换工作。
 * - 12 个 per-view test 各自 click 目标 view 的 NavRail button，断言：
 *     - URL 落定后 hash 包含 view id；
 *     - view-specific anchor 元素出现（防"渲染错 view 但壳还在"）；
 *     - 无 pageerror、无 console.error/warning（按 allowlist 过滤）。
 * - chat 不在 NavRail，单独覆盖 ChatDock 打开/关闭，以及实时消息接收、图片代理、
 *   群成员加载、发送含粘贴图片、右键撤回等真实 console action 路径。
 * - 配置治理、警告记录、黑名单、订阅管理与系统缓存覆盖真实操作路径，防止
 *   console action / WebSocket API 只在单元测试中通过、但浏览器 UI 断链。
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
 * NavRail 中可达的 12 个 view（chat 不在 rail 中，独立 dock 测）。
 *
 * label = NavRail 内显示文本 = button[title]；可能与 view 内部 H1 不同
 * （如 identity 的 nav label "限制中" vs 内部 H1 "身份认证"）。
 *
 * anchor 选择标准：必须能在 dashboard fallback 场景下区分"是不是 view 真的渲染对了"。
 */
const VIEWS: readonly ViewSpec[] = [
  { id: 'dashboard', label: '总览', anchor: { selector: '.sh-dashboard__title', text: '控制台总览' } },
  { id: 'admission', label: '入群认证', anchor: { selector: '.sh-workspace-head__title', text: '入群认证' } },
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
    if (view.id === 'dashboard') {
      await expect(page.getByText('已配置群组').first()).toBeVisible({ timeout: 10_000 })
      await expect(page.getByText('加载失败')).toHaveCount(0)
    }

    tracker.assertClean()
  })
}

test('identity view filters restricted members and renders verification context', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '限制中')
  await expect(page).toHaveURL(/#identity($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.sh-workspace-head__title', { hasText: '身份认证' }).first()).toBeVisible({
    timeout: 10_000,
  })

  const metric = (label: string) =>
    page
      .locator('.sh-dashboard-metrics .sh-stat')
      .filter({ has: page.locator('.sh-stat__label', { hasText: label }) })
      .locator('.sh-stat__value')
      .first()

  await expect(metric('待认证成员')).toHaveText('2')
  await expect(metric('已认证成员')).toHaveText('1')
  await expect(metric('待完善认证')).toHaveText('1')
  await expect(metric('最近释放')).toHaveText('1')

  await page.locator('.sh-identity-toolbar__field--wide input').fill('')
  await page.locator('.sh-identity-toolbar__field').first().locator('.el-select__wrapper').click()
  await page.locator('.el-select-dropdown__item:visible', { hasText: '全部群组' }).click()

  const restrictedSection = page.locator('.sh-section', { hasText: '当前受限成员' }).first()
  await expect(restrictedSection.locator('.sh-lane__row', { hasText: 'E2E 待认证成员' })).toBeVisible({
    timeout: 10_000,
  })
  await expect(restrictedSection.locator('.sh-lane__row', { hasText: 'E2E 待完善认证成员' })).toBeVisible()
  await expect(restrictedSection.locator('.sh-lane__row', { hasText: 'E2E 已释放成员' })).toHaveCount(0)

  await page.locator('.sh-identity-toolbar__field').first().locator('.el-select__wrapper').click()
  await page.locator('.el-select-dropdown__item:visible', { hasText: '1002 · 1 待处理' }).click()
  await page.locator('.sh-identity-toolbar__field--wide input').fill('待完善')

  await expect(restrictedSection.locator('.sh-lane__row')).toHaveCount(1)
  const filteredMember = restrictedSection.locator('.sh-lane__row', { hasText: 'E2E 待完善认证成员' }).first()
  await expect(filteredMember).toBeVisible()
  await expect(filteredMember).toContainText('bound_unverified')
  await filteredMember.click()

  const detailSection = page.locator('.sh-section', { hasText: '成员详情' }).first()
  await expect(detailSection).toContainText('200202')
  await expect(detailSection).toContainText('E2E 待完善认证成员')
  await expect(detailSection).toContainText('bound_unverified')

  const releaseSection = page.locator('.sh-section', { hasText: '最近自动解除' }).first()
  await expect(releaseSection).toContainText('E2E 已释放成员')
  await expect(releaseSection).toContainText('200203')
  await expect(releaseSection).toContainText('verified')
  await expect(page.locator('.sh-section', { hasText: '查询错误' })).toHaveCount(0)

  tracker.assertClean()
})

test('admission runtime saves switches and executes member actions through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '入群认证')
  await expect(page).toHaveURL(/#admission($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.sh-workspace-head__title', { hasText: '入群认证' }).first()).toBeVisible({
    timeout: 10_000,
  })

  const actionStreamRow = page.locator('.sh-lane__row', { hasText: 'Action Stream' }).first()
  await expect(actionStreamRow).toBeVisible({ timeout: 10_000 })
  await actionStreamRow.locator('.el-switch').click()
  await expect(page.getByText('已保存入群认证运行开关。')).toBeVisible({ timeout: 10_000 })
  await expect(actionStreamRow.locator('.el-switch')).not.toHaveClass(/is-checked/, { timeout: 10_000 })

  const memberRow = page.locator('.el-table__row', { hasText: '200201' }).first()
  await expect(memberRow).toBeVisible({ timeout: 10_000 })
  await memberRow.getByRole('button', { name: '查询', exact: true }).click()
  await expect(page.getByText(/QQ 200201 的入群认证状态：linked/)).toBeVisible({ timeout: 10_000 })

  await memberRow.getByRole('button', { name: '重发', exact: true }).click()
  const resendDialog = confirmDialog(page, '确认入群认证动作')
  await expect(resendDialog).toBeVisible({ timeout: 5_000 })
  await expect(resendDialog.getByText('确定要对 QQ 200201 执行「重发」吗？')).toBeVisible()
  await resendDialog.getByRole('button', { name: '重发', exact: true }).click()

  await expect(page.getByText('已重发 QQ 200201 的入群认证链接。')).toBeVisible({ timeout: 10_000 })
  const actions = await uiSmokeChatActions(page)
  expect(actions.sentMessages.at(-1)?.content).toContain('https://join.stuhelper.com/verify/session-ui-smoke-200201')

  tracker.assertClean()
})

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

  await page.locator('.sh-dock[data-open="true"] .sh-dock__action[title="关闭"]').click()
  await expect(page.locator('.sh-dock[data-open="true"]')).toHaveCount(0, { timeout: 5_000 })

  tracker.assertClean()
})

test('chat dock receives, sends image message, and recalls through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await page.evaluate(() => {
    const state = globalThis as unknown as { __chatClipboardWrites?: string[] }
    state.__chatClipboardWrites = []
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: async (text: string) => {
          state.__chatClipboardWrites?.push(text)
        },
      },
    })
  })

  await clickNavRail(page, VIEWS[0].label)
  await expect(page).toHaveURL(/#dashboard($|\?)/, { timeout: 5_000 })

  await page.locator('.sh-cmd__chat').first().click()
  const dock = page.locator('.sh-dock[data-open="true"]').first()
  await expect(dock.locator('.chat-view')).toBeVisible({ timeout: 10_000 })

  await uiSmokeChatPost(page, '/__stuhelper-ui-smoke/chat/incoming', {
    content: 'E2E incoming chat message',
    includeImage: true,
  })

  const sessionItem = dock.locator('.session-item', { hasText: 'E2E 聊天频道' }).first()
  await expect(sessionItem).toBeVisible({ timeout: 10_000 })
  await expect(sessionItem.locator('.badge')).toHaveText('1')
  await sessionItem.click()

  await expect(dock.locator('.chat-header', { hasText: 'E2E 聊天频道' })).toBeVisible({ timeout: 10_000 })
  await expect(dock.locator('.member-item', { hasText: 'E2E 群主' })).toBeVisible({ timeout: 10_000 })
  await expect(dock.locator('.member-item', { hasText: 'E2E 管理员' })).toBeVisible()
  await expect(dock.locator('.member-item', { hasText: 'E2E 聊天用户' })).toBeVisible()

  const incomingRow = dock.locator('.message-row', { hasText: 'E2E incoming chat message' }).first()
  await expect(incomingRow).toBeVisible({ timeout: 10_000 })
  await expect(incomingRow.locator('.username')).toHaveText('E2E 聊天用户')
  await expect(incomingRow.locator('img.msg-img[alt="聊天图片"]').first()).toHaveAttribute(
    'src',
    /^data:image\/png;base64,/,
    { timeout: 10_000 },
  )

  await incomingRow.click({ button: 'right' })
  const copyMenu = page.locator('.context-menu').first()
  await expect(copyMenu).toBeVisible({ timeout: 5_000 })
  await copyMenu.locator('.context-menu-item', { hasText: '复制' }).click()
  await expect(toastMessage(page, '已复制到剪贴板')).toBeVisible({ timeout: 10_000 })
  await expect
    .poll(() =>
      page.evaluate(() => {
        const state = globalThis as unknown as { __chatClipboardWrites?: string[] }
        return state.__chatClipboardWrites?.at(-1) ?? ''
      }),
    )
    .toContain('E2E incoming chat message')
  await expect
    .poll(() =>
      page.evaluate(() => {
        const state = globalThis as unknown as { __chatClipboardWrites?: string[] }
        return state.__chatClipboardWrites?.at(-1) ?? ''
      }),
    )
    .toContain('[图片:ui-smoke-image.png]')

  const input = dock.locator('.chat-input').first()
  await input.fill('E2E outbound chat text')
  await pasteTinyPng(page)
  await expect(dock.locator('.pending-image-item')).toBeVisible({ timeout: 5_000 })

  await dock.locator('.send-btn').click()
  await expect(input).toHaveValue('', { timeout: 10_000 })
  await expect(dock.locator('.pending-image-item')).toHaveCount(0, { timeout: 10_000 })

  const selfRow = dock.locator('.message-row.self', { hasText: 'E2E outbound chat text' }).first()
  await expect(selfRow).toBeVisible({ timeout: 10_000 })

  const actions = await uiSmokeChatActions(page)
  expect(actions.sentMessages.at(-1)?.content).toContain('E2E outbound chat text')
  expect(actions.sentMessages.at(-1)?.content).toContain('<img src="data:image/png;base64,')

  await selfRow.click({ button: 'right' })
  const contextMenu = page.locator('.context-menu').first()
  await expect(contextMenu).toBeVisible({ timeout: 5_000 })
  await contextMenu.locator('.context-menu-item.danger', { hasText: '撤回' }).click()

  await expect(toastMessage(page, '消息已撤回')).toBeVisible({ timeout: 10_000 })
  await expect(dock.locator('.message-row.self', { hasText: 'E2E outbound chat text' })).toHaveCount(0, {
    timeout: 10_000,
  })

  const recalled = await uiSmokeChatActions(page)
  expect(recalled.recalledMessages.at(-1)?.messageId).toBe(actions.sentMessages.at(-1)?.messageId)

  await dock.locator('.sh-dock__action[title="关闭"]').click()
  await expect(page.locator('.sh-dock[data-open="true"]')).toHaveCount(0, { timeout: 5_000 })

  tracker.assertClean()
})

test('global search opens from keyboard and navigates to a view result', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '总览')
  await expect(page).toHaveURL(/#dashboard($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.sh-cmd__search')).toHaveAttribute('title', '⌘K 全站搜索')

  await page.keyboard.press('Control+K')

  const searchDialog = page.getByRole('dialog', { name: '全站搜索' })
  await expect(searchDialog).toBeVisible({ timeout: 5_000 })
  await searchDialog.getByPlaceholder('输入用户 ID / 群号 / 视图名 / 命令…').fill('logs')
  await searchDialog.getByText('日志检索').click()

  await expect(page).toHaveURL(/#logs($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.sh-workspace-head__title', { hasText: '日志检索' }).first()).toBeVisible({
    timeout: 10_000,
  })
  await expect(searchDialog).toBeHidden({ timeout: 5_000 })

  tracker.assertClean()
})

test('global search opens entity overlay and entity jump updates navigation state', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '总览')
  await expect(page).toHaveURL(/#dashboard($|\?)/, { timeout: 5_000 })

  await page.locator('.sh-cmd__search').click()

  const searchDialog = page.getByRole('dialog', { name: '全站搜索' })
  await expect(searchDialog).toBeVisible({ timeout: 5_000 })
  await searchDialog.getByPlaceholder('输入用户 ID / 群号 / 视图名 / 命令…').fill('100000')
  await searchDialog.getByText('查看用户 100000').click()

  const overlay = page.locator('.sh-overlay[data-open="true"]')
  await expect(overlay).toBeVisible({ timeout: 10_000 })
  await expect(overlay.locator('.sh-overlay__kind')).toHaveText('USER')
  await expect(overlay.locator('.sh-overlay__title')).toHaveText('100000')
  await expect(overlay.getByText('此用户当前无任何记录。')).toBeVisible({ timeout: 10_000 })

  await overlay.locator('.sh-overlay__jump', { hasText: '警告记录' }).click()

  await expect(page).toHaveURL(/#warns\?keyword=100000/, { timeout: 5_000 })
  await expect(page.locator('.sh-workspace-head__title', { hasText: '警告记录' }).first()).toBeVisible({
    timeout: 10_000,
  })
  await expect(page.locator('.sh-overlay[data-open="true"]')).toHaveCount(0)

  tracker.assertClean()
})

test('review center dismisses a seeded report through real console action', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '处置中心')
  await expect(page).toHaveURL(/#review($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.sh-workspace-head__title', { hasText: '处置中心' }).first()).toBeVisible({
    timeout: 10_000,
  })

  await selectLabeledOption(page, '类型', '举报')
  await fillLabeledInput(page, '检索', 'dismiss-report-token')

  const row = page
    .locator('.sh-lane__row', { hasText: '200200' })
    .filter({ hasText: 'dismiss-report-token' })
    .first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await expect(row.getByText('举报', { exact: true })).toBeVisible()
  await expect(row.getByText('completed', { exact: true })).toBeVisible()
  await row.click()

  await expect(page.getByText('E2E 处置中心关联事件 report-related-token')).toBeVisible({ timeout: 10_000 })
  await page
    .locator('.sh-field', { hasText: '处理备注(可选)' })
    .locator('input')
    .first()
    .fill('E2E 浏览器驳回举报备注')
  await page.getByRole('button', { name: '驳回举报', exact: true }).click()

  const actionDialog = confirmDialog(page, '确认处置')
  await expect(actionDialog).toBeVisible({ timeout: 5_000 })
  await expect(actionDialog.getByText('确定要对 200200 执行「驳回举报」吗？')).toBeVisible()
  await actionDialog.getByRole('button', { name: '驳回举报', exact: true }).click()

  await expect(toastMessage(page, '已驳回举报：200200')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.sh-lane__row', { hasText: '200200' })).toHaveCount(0, { timeout: 10_000 })
  await expect(page.getByText('没有匹配的工作项')).toBeVisible({ timeout: 10_000 })

  tracker.assertClean()
})

test('log search filters seeded command log and opens detail drawer', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '日志检索')
  await expect(page).toHaveURL(/#logs($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.sh-workspace-head__title', { hasText: '日志检索' }).first()).toBeVisible({
    timeout: 10_000,
  })

  await fillLabeledInput(page, '命令', 'e2e.log-search')
  await fillLabeledInput(page, '用户 ID', '100000')
  await fillLabeledInput(page, '详情关键字', 'drawer-match-token')
  await page.getByRole('button', { name: '检索', exact: true }).click()

  const row = page
    .locator('.el-table__row', { hasText: 'e2e.log-search' })
    .filter({ hasText: '100000' })
    .filter({ hasText: 'drawer-match-token' })
    .first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await expect(row.getByText('成功', { exact: true })).toBeVisible()

  await row.click()

  const drawer = page.locator('.el-drawer.sh-drawer', { hasText: 'e2e.log-search' }).first()
  await expect(drawer).toBeVisible({ timeout: 5_000 })
  await expect(drawer.getByText('E2E 日志用户')).toBeVisible()
  await expect(drawer.getByText('E2E 日志群')).toBeVisible()
  await expect(drawer.getByText('42 ms')).toBeVisible()
  await expect(
    drawer.locator('.sh-logs__code', { hasText: 'E2E command log result drawer-match-token' }),
  ).toBeVisible()

  await drawer.getByRole('button', { name: '关闭', exact: true }).click()
  await expect(drawer).toBeHidden({ timeout: 5_000 })

  await page.getByRole('button', { name: '重置', exact: true }).click()
  await expect(page.locator('label', { hasText: '命令' }).locator('input').first()).toHaveValue('')
  await expect(page.locator('label', { hasText: '用户 ID' }).locator('input').first()).toHaveValue('')
  await expect(page.locator('label', { hasText: '详情关键字' }).locator('input').first()).toHaveValue('')

  tracker.assertClean()
})

test('config governance workspace tabs render and update navigation state', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '群组配置')
  await expect(page).toHaveURL(/#config($|\?)/, { timeout: 5_000 })

  const workspaceCases = [
    { label: '群配置', hash: /#config($|\?)/, anchor: '群组配置' },
    { label: '模板库', hash: /#config\?workspace=templates/, anchor: '编辑模板' },
    { label: '群绑定', hash: /#config\?workspace=bindings/, anchor: '新建 / 编辑绑定' },
    { label: '命令策略', hash: /#config\?workspace=command-policies/, anchor: '编辑命令策略' },
  ] as const

  for (const item of workspaceCases) {
    await page.getByRole('button', { name: item.label, exact: true }).click()
    await expect(page).toHaveURL(item.hash, { timeout: 5_000 })
    await expect(page.getByText(item.anchor).first()).toBeVisible({ timeout: 10_000 })
  }

  tracker.assertClean()
})

test('config governance confirms before discarding dirty workspace form', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '群组配置')
  await page.getByRole('button', { name: '模板库', exact: true }).click()
  await expect(page).toHaveURL(/#config\?workspace=templates/, { timeout: 5_000 })

  const templateIdInput = page.locator('label', { hasText: '模板 ID' }).locator('input').first()
  await templateIdInput.fill('dirty-template-draft')
  await expect(page.getByText('有未提交更改').first()).toBeVisible()

  await page.getByRole('button', { name: '群绑定', exact: true }).click()
  const cancelDialog = confirmDialog(page, '放弃未保存更改')
  await expect(cancelDialog).toBeVisible({ timeout: 5_000 })
  await expect(cancelDialog.getByText('当前有未保存的改动，继续会丢失这些修改。')).toBeVisible()
  await cancelDialog.getByRole('button', { name: '取消', exact: true }).click()

  await expect(cancelDialog).toBeHidden({ timeout: 5_000 })
  await expect(page).toHaveURL(/#config\?workspace=templates/, { timeout: 5_000 })
  await expect(templateIdInput).toHaveValue('dirty-template-draft')

  await page.getByRole('button', { name: '群绑定', exact: true }).click()
  const discardDialog = confirmDialog(page, '放弃未保存更改')
  await expect(discardDialog).toBeVisible({ timeout: 5_000 })
  await discardDialog.getByRole('button', { name: '放弃更改', exact: true }).click()

  await expect(discardDialog).toBeHidden({ timeout: 5_000 })
  await expect(page).toHaveURL(/#config\?workspace=bindings/, { timeout: 5_000 })
  await expect(page.getByText('新建 / 编辑绑定').first()).toBeVisible({ timeout: 10_000 })

  tracker.assertClean()
})

test('config governance template save failures stay in the form without replacing page data', async ({
  loggedInPage: page,
}) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '群组配置')
  await page.getByRole('button', { name: '模板库', exact: true }).click()
  await expect(page).toHaveURL(/#config\?workspace=templates/, { timeout: 5_000 })

  await fillLabeledInput(page, '模板 ID', 'e2e-invalid-template')
  await fillLabeledInput(page, '模板名称', 'E2E 失败模板')
  await fillLabeledInput(page, '禁言时长(秒)', '999999999')
  await fillLabeledInput(page, '踢出阈值(分钟)', '15')
  await fillLabeledInput(page, '提醒文案', 'E2E 保存失败后保留草稿')
  await page.getByRole('button', { name: '保存模板', exact: true }).click()

  const actionError = page.locator('.sh-config-governance__action-error').first()
  await expect(actionError).toBeVisible({ timeout: 10_000 })
  await expect(actionError).toContainText('保存模板失败')
  await expect(actionError).toContainText('禁言时长')
  await expect(page.locator('.sh-load-error', { hasText: '刷新配置治理数据失败' })).toHaveCount(0)
  await expect(page.locator('label', { hasText: '模板 ID' }).locator('input').first()).toHaveValue(
    'e2e-invalid-template',
  )

  await actionError.getByRole('button', { name: '关闭', exact: true }).click()
  await expect(actionError).toHaveCount(0)

  await fillLabeledInput(page, '禁言时长(秒)', '120')
  await page.getByRole('button', { name: '保存模板', exact: true }).click()
  await expect(page.getByText('已保存群模板：E2E 失败模板')).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('E2E 失败模板').first()).toBeVisible({ timeout: 10_000 })

  tracker.assertClean()
})

test('config governance binding and command policy validation failures keep drafts editable', async ({
  loggedInPage: page,
}) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '群组配置')
  await page.getByRole('button', { name: '群绑定', exact: true }).click()
  await expect(page).toHaveURL(/#config\?workspace=bindings/, { timeout: 5_000 })

  await fillLabeledInput(page, '平台', 'qq')
  await fillLabeledInput(page, '群号', '1002')
  await fillLabeledInput(page, '备注', '未选择模板时保留绑定草稿')
  await page.getByRole('button', { name: '保存绑定', exact: true }).click()

  let actionError = page.locator('.sh-config-governance__action-error').first()
  await expect(actionError).toBeVisible({ timeout: 10_000 })
  await expect(actionError).toContainText('保存绑定失败')
  await expect(actionError).toContainText('模板不能为空')
  await expect(page.locator('.sh-load-error', { hasText: '刷新配置治理数据失败' })).toHaveCount(0)
  await expect(page.locator('label', { hasText: '群号' }).locator('input').first()).toHaveValue('1002')

  await actionError.getByRole('button', { name: '关闭', exact: true }).click()
  await page.getByRole('button', { name: '命令策略', exact: true }).click()
  const discardDialog = confirmDialog(page, '放弃未保存更改')
  await expect(discardDialog).toBeVisible({ timeout: 5_000 })
  await discardDialog.getByRole('button', { name: '放弃更改', exact: true }).click()

  await expect(page).toHaveURL(/#config\?workspace=command-policies/, { timeout: 5_000 })
  await fillLabeledInput(page, '角色白名单(逗号分隔)', 'admin')
  await page.getByRole('button', { name: '保存策略', exact: true }).click()

  actionError = page.locator('.sh-config-governance__action-error').first()
  await expect(actionError).toBeVisible({ timeout: 10_000 })
  await expect(actionError).toContainText('保存策略失败')
  await expect(actionError).toContainText('命令不能为空')
  await expect(page.locator('.sh-load-error', { hasText: '刷新配置治理数据失败' })).toHaveCount(0)
  await expect(page.locator('label', { hasText: '角色白名单(逗号分隔)' }).locator('input').first()).toHaveValue(
    'admin',
  )

  await actionError.getByRole('button', { name: '关闭', exact: true }).click()
  await fillLabeledInput(page, '角色白名单(逗号分隔)', '')

  tracker.assertClean()
})

test('legacy group config opens plugin editor and saves normalized defaults', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '群组配置')
  await page.getByRole('button', { name: '群配置', exact: true }).click()
  await expect(page).toHaveURL(/#config($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.view-title', { hasText: '群组配置' })).toBeVisible({ timeout: 10_000 })

  await page.locator('.config-view .search-input').fill('190001')
  const row = page.locator('.config-view .list-row', { hasText: '190001' }).first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await expect(row.getByText('1 验证')).toBeVisible()
  await expect(row.getByText('1 违规')).toBeVisible()
  await row.getByRole('button', { name: '编辑', exact: true }).click()

  const dialog = configEditDialog(page, '190001')
  await expect(dialog).toBeVisible({ timeout: 5_000 })
  await expect(configTextarea(dialog, '验证关键词')).toHaveValue('legacy-pass')

  await dialog.locator('.sidebar-item', { hasText: '功能插件' }).click()
  const banmeCard = dialog.locator('.plugin-card', { hasText: '自我禁言' }).first()
  await expect(banmeCard).toBeVisible()
  await banmeCard.locator('.plugin-header').click()
  await expect(configInput(dialog, 'UP时长')).toHaveValue('24h')
  await expect(configInput(dialog, '歪时长')).toHaveValue('12h')

  await dialog.locator('.sidebar-item', { hasText: '入群设置' }).click()
  await configTextarea(dialog, '验证关键词').fill('legacy-pass, e2e-added-pass')
  await dialog.locator('.sidebar-item', { hasText: '违规管理' }).click()
  await configTextarea(dialog, '禁言关键词').fill('legacy-spam, e2e-added-spam')
  await dialog.getByRole('button', { name: '保存', exact: true }).click()

  await expect(toastMessage(page, '保存成功')).toBeVisible({ timeout: 10_000 })
  await expect(dialog).toHaveCount(0, { timeout: 10_000 })
  await expect(row.getByText('2 验证')).toBeVisible({ timeout: 10_000 })
  await expect(row.getByText('2 违规')).toBeVisible()

  tracker.assertClean()
})

test('legacy group config creates, copies, reloads, and deletes through real console actions', async ({
  loggedInPage: page,
}) => {
  await using tracker = createTracker(page)

  const guildId = '190099'
  await page.evaluate(() => {
    const state = globalThis as unknown as { __configClipboardWrites?: string[] }
    state.__configClipboardWrites = []
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: async (text: string) => {
          state.__configClipboardWrites?.push(text)
        },
      },
    })
  })

  await clickNavRail(page, '群组配置')
  await page.getByRole('button', { name: '群配置', exact: true }).click()
  await expect(page).toHaveURL(/#config($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.view-title', { hasText: '群组配置' })).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: '新建配置', exact: true }).click()
  const createDialog = page.locator('.config-view .dialog-card', { hasText: '新建群组配置' }).first()
  await expect(createDialog).toBeVisible({ timeout: 5_000 })
  await createDialog.locator('input').fill(guildId)
  await createDialog.getByRole('button', { name: '创建', exact: true }).click()

  await expect(toastMessage(page, '创建成功')).toBeVisible({ timeout: 10_000 })
  const editDialog = configEditDialog(page, guildId)
  await expect(editDialog).toBeVisible({ timeout: 10_000 })
  await editDialog.getByRole('button', { name: '取消', exact: true }).click()
  await expect(editDialog).toHaveCount(0, { timeout: 10_000 })

  await page.locator('.config-view .search-input').fill(guildId)
  const row = page.locator('.config-view .list-row', { hasText: guildId }).first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await row.getByRole('button', { name: '复制', exact: true }).click()
  await expect(toastMessage(page, '已复制群号')).toBeVisible({ timeout: 10_000 })
  await expect
    .poll(() =>
      page.evaluate(() => {
        const state = globalThis as unknown as { __configClipboardWrites?: string[] }
        return state.__configClipboardWrites ?? []
      }),
    )
    .toContain(guildId)

  await page.evaluate(() => {
    const state = globalThis as unknown as { __configFallbackCopies?: string[] }
    state.__configFallbackCopies = []
    const originalExecCommand = document.execCommand?.bind(document)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: async () => {
          throw new Error('clipboard denied')
        },
      },
    })
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: (command: string) => {
        if (command !== 'copy') {
          return originalExecCommand?.(command) ?? false
        }
        const activeElement = document.activeElement
        state.__configFallbackCopies?.push(
          activeElement instanceof HTMLTextAreaElement ? activeElement.value : '',
        )
        return true
      },
    })
  })
  await row.getByRole('button', { name: '复制', exact: true }).click()
  await expect(toastMessage(page, '已复制群号')).toBeVisible({ timeout: 10_000 })
  await expect
    .poll(() =>
      page.evaluate(() => {
        const state = globalThis as unknown as { __configFallbackCopies?: string[] }
        return state.__configFallbackCopies ?? []
      }),
    )
    .toContain(guildId)

  await page.getByRole('button', { name: '重载', exact: true }).click()
  await expect(toastMessage(page, '已重新加载')).toBeVisible({ timeout: 10_000 })
  await expect(row).toBeVisible({ timeout: 10_000 })

  await row.getByRole('button', { name: '删除', exact: true }).click()
  const deleteDialog = page.locator('.config-view .dialog-card', { hasText: '删除群组配置' }).first()
  await expect(deleteDialog).toBeVisible({ timeout: 5_000 })
  await deleteDialog.locator('input').fill(guildId)
  await deleteDialog.getByRole('button', { name: '删除', exact: true }).click()

  await expect(toastMessage(page, '删除成功')).toBeVisible({ timeout: 10_000 })
  await expect(row).toHaveCount(0, { timeout: 10_000 })

  tracker.assertClean()
})

test('config governance saves template, binding, and command policy through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '群组配置')
  await page.getByRole('button', { name: '模板库', exact: true }).click()
  await expect(page).toHaveURL(/#config\?workspace=templates/, { timeout: 5_000 })

  await fillLabeledInput(page, '模板 ID', 'e2e-template')
  await fillLabeledInput(page, '模板名称', 'E2E 模板')
  await fillLabeledInput(page, '禁言时长(秒)', '90')
  await fillLabeledInput(page, '踢出阈值(分钟)', '15')
  await fillLabeledInput(page, '提醒文案', 'E2E 自动化提醒')
  await fillLabeledInput(page, '豁免名单(逗号分隔)', '10001,10002')

  await page.getByRole('button', { name: '保存模板', exact: true }).click()

  await expect(page.getByText('已保存群模板：E2E 模板')).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('E2E 模板').first()).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: '群绑定', exact: true }).click()
  await expect(page).toHaveURL(/#config\?workspace=bindings/, { timeout: 5_000 })
  await fillLabeledInput(page, '平台', 'qq')
  await fillLabeledInput(page, '群号', '1001')
  await selectLabeledOption(page, '模板', 'E2E 模板 (e2e-template)')
  await fillLabeledInput(page, '备注', 'E2E 绑定验证')
  await page.getByRole('button', { name: '保存绑定', exact: true }).click()

  await expect(page.getByText('已保存群绑定：qq/1001')).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('E2E 绑定验证').first()).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: '命令策略', exact: true }).click()
  await expect(page).toHaveURL(/#config\?workspace=command-policies/, { timeout: 5_000 })
  await selectLabeledOption(page, '命令', 'report')
  await fillLabeledInput(page, '最小 authority', '4')
  await fillLabeledInput(page, '角色白名单(逗号分隔)', 'admin, moderator')
  await page.getByRole('button', { name: '保存策略', exact: true }).click()

  await expect(page.getByText('已保存命令策略。')).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('authority 4 · admin, moderator').first()).toBeVisible({ timeout: 10_000 })

  tracker.assertClean()
})

test('subscription management adds, edits, and deletes a target through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  const targetId = `e2e-sub-${Date.now()}`

  await clickNavRail(page, '推送订阅')
  await expect(page).toHaveURL(/#subscriptions($|\?)/, { timeout: 5_000 })

  await page.getByRole('button', { name: '添加订阅', exact: true }).click()
  await expect(page.locator('.el-drawer', { hasText: '添加订阅' }).first()).toBeVisible({ timeout: 5_000 })
  await fillLabeledInput(page, '目标 ID', targetId)
  await setFeatureChecked(page, '防撤回', true)
  await page.getByRole('button', { name: '添加', exact: true }).click()

  await expect(page.getByText('订阅已添加')).toBeVisible({ timeout: 10_000 })
  const card = page.locator('.sh-sub-card', { hasText: targetId }).first()
  await expect(card).toBeVisible({ timeout: 10_000 })
  await expect(card.locator('.sh-sub-card__feature', { hasText: '防撤回' })).toHaveAttribute('data-active', 'true')

  await card.click()
  await expect(page.locator('.el-drawer', { hasText: '编辑订阅' }).first()).toBeVisible({ timeout: 5_000 })
  await setFeatureChecked(page, '防撤回', false)
  await setFeatureChecked(page, '禁言解除', true)
  await page.getByRole('button', { name: '保存', exact: true }).click()

  await expect(page.getByText('订阅已更新')).toBeVisible({ timeout: 10_000 })
  await expect(card.locator('.sh-sub-card__feature', { hasText: '防撤回' })).toHaveAttribute('data-active', 'false')
  await expect(card.locator('.sh-sub-card__feature', { hasText: '禁言解除' })).toHaveAttribute('data-active', 'true')

  await card.click()
  await expect(page.locator('.el-drawer', { hasText: '编辑订阅' }).first()).toBeVisible({ timeout: 5_000 })
  await page.getByRole('button', { name: '删除订阅', exact: true }).click()
  await page.getByRole('button', { name: '删除', exact: true }).click()

  await expect(page.getByText('订阅已删除')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.sh-sub-card', { hasText: targetId })).toHaveCount(0, { timeout: 10_000 })

  tracker.assertClean()
})

test('blacklist management adds and releases a global entry through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  const userId = `200${Date.now()}`
  const reason = `E2E 全局拉黑 ${userId}`

  await clickNavRail(page, '黑名单')
  await expect(page).toHaveURL(/#blacklist($|\?)/, { timeout: 5_000 })

  await page.getByRole('button', { name: '添加用户', exact: true }).click()
  await expect(page.locator('.el-drawer', { hasText: '添加黑名单用户' }).first()).toBeVisible({ timeout: 5_000 })
  await fillLabeledInput(page, '用户 ID', userId)
  await selectLabeledOption(page, '范围', '全局')
  await fillLabeledInput(page, '原因', reason)
  await page.getByRole('button', { name: '添加', exact: true }).click()

  const globalAddDialog = confirmDialog(page, '添加全局黑名单')
  await expect(globalAddDialog).toBeVisible({ timeout: 5_000 })
  await expect(globalAddDialog.getByText(`确定要将 ${userId} 加入全局黑名单吗？该成员会被所有群拒绝。`)).toBeVisible()
  await globalAddDialog.getByRole('button', { name: '添加全局黑名单', exact: true }).click()

  await expect(page.getByText(`已将 ${userId} 加入黑名单`)).toBeVisible({ timeout: 10_000 })
  const row = page.locator('.el-table__row', { hasText: userId }).first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await expect(row.getByText('全局', { exact: true })).toBeVisible()
  await expect(row.getByText(reason)).toBeVisible()

  await row.getByRole('button', { name: '解除', exact: true }).click()
  const releaseDialog = confirmDialog(page, '解除黑名单成员')
  await expect(releaseDialog).toBeVisible({ timeout: 5_000 })
  await expect(releaseDialog.getByText(`确定要解除 ${userId} 的全局黑名单吗？认证失败计数会保留。`)).toBeVisible()
  await releaseDialog.getByRole('button', { name: '解除', exact: true }).click()

  await expect(page.getByText(`已从黑名单解除 ${userId}`)).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.el-table__row', { hasText: userId })).toHaveCount(0, { timeout: 10_000 })

  tracker.assertClean()
})

test('warn records add, reload, and clear through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  const suffix = Date.now().toString().slice(-9)
  const guildId = `g${suffix}`
  const userId = `u${suffix}`

  await clickNavRail(page, '警告记录')
  await expect(page).toHaveURL(/#warns($|\?)/, { timeout: 5_000 })

  await page.getByRole('button', { name: '添加警告', exact: true }).click()
  await expect(page.locator('.el-drawer', { hasText: '添加警告' }).first()).toBeVisible({ timeout: 5_000 })
  await fillLabeledInput(page, '群号', guildId)
  await fillLabeledInput(page, '用户 ID', userId)
  await page.getByRole('button', { name: '添加', exact: true }).click()

  await expect(page.getByText(`已在 ${guildId} 添加警告`)).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.sh-lane__row', { hasText: guildId }).first()).toBeVisible({ timeout: 10_000 })

  const row = page.locator('.el-table__row', { hasText: userId }).first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await expect(row.locator('.el-input-number input')).toHaveValue('1')

  await page.getByRole('button', { name: '重载', exact: true }).click()
  await expect(page.getByText('警告数据已重新加载')).toBeVisible({ timeout: 10_000 })
  await expect(row).toBeVisible({ timeout: 10_000 })

  await row.getByRole('button', { name: '清除', exact: true }).click()
  const clearDialog = confirmDialog(page, '清除警告记录')
  await expect(clearDialog).toBeVisible({ timeout: 5_000 })
  await expect(clearDialog.getByText('确定要清除这条警告记录吗？清零后该成员会从当前群组列表中移除。')).toBeVisible()
  await clearDialog.getByRole('button', { name: '清除', exact: true }).click()

  await expect(page.getByText('警告已清除')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.el-table__row', { hasText: userId })).toHaveCount(0, { timeout: 10_000 })
  await expect(page.getByText('暂无警告记录')).toBeVisible({ timeout: 10_000 })

  tracker.assertClean()
})

test('system cache actions refresh and clear through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '系统 / 缓存')
  await expect(page).toHaveURL(/#system($|\?)/, { timeout: 5_000 })

  await expect(page.getByText('群组缓存')).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('用户缓存')).toBeVisible()
  await expect(page.getByText('成员缓存')).toBeVisible()

  await page.getByRole('button', { name: '刷新统计', exact: true }).click()
  await expect(page.getByText('缓存操作')).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: '强制刷新', exact: true }).click()
  const refreshDialog = confirmDialog(page, '强制刷新缓存')
  await expect(refreshDialog).toBeVisible({ timeout: 5_000 })
  await expect(refreshDialog.getByText('确认从 Bot 重新拉取所有群组、用户、成员信息？这可能耗时数十秒。')).toBeVisible()
  await refreshDialog.getByRole('button', { name: '开始刷新', exact: true }).click()
  await expect(page.getByText('缓存刷新完成')).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: '清空缓存', exact: true }).click()
  const clearDialog = confirmDialog(page, '清空缓存')
  await expect(clearDialog).toBeVisible({ timeout: 5_000 })
  await expect(clearDialog.getByText('确认清空所有本地缓存？下次访问相关群 / 用户时会重新拉取。')).toBeVisible()
  await clearDialog.getByRole('button', { name: '清空', exact: true }).click()
  await expect(page.getByText('缓存已清空')).toBeVisible({ timeout: 10_000 })

  tracker.assertClean()
})

test('global settings discard, save, and restore defaults through real console actions', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  await clickNavRail(page, '全局设置')
  await expect(page).toHaveURL(/#settings($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.view-title', { hasText: '全局设置' })).toBeVisible({ timeout: 10_000 })

  const warnLimitInput = settingsInput(page, '警告次数限制')
  const banExpressionInput = settingsInput(page, '禁言时长表达式')

  await expect(warnLimitInput).toHaveValue('3')
  await fillSettingsInput(page, '警告次数限制', '7')
  await expect(page.locator('.save-bar', { hasText: '检测到未保存的修改' })).toBeVisible()
  await page.getByRole('button', { name: '放弃更改', exact: true }).click()

  const discardDialog = confirmDialog(page, '放弃更改')
  await expect(discardDialog).toBeVisible({ timeout: 5_000 })
  await expect(discardDialog.getByText('确定要放弃当前所有未保存的修改吗？')).toBeVisible()
  await discardDialog.getByRole('button', { name: '确认', exact: true }).click()

  await expect(page.getByText('已放弃更改')).toBeVisible({ timeout: 10_000 })
  await expect(warnLimitInput).toHaveValue('3')
  await expect(page.locator('.save-bar')).toHaveCount(0, { timeout: 5_000 })

  await fillSettingsInput(page, '警告次数限制', '5')
  await fillSettingsInput(page, '禁言时长表达式', '{t}h')
  await page.getByRole('button', { name: '保存更改', exact: true }).click()

  await expect(toastMessage(page, '设置已保存')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.save-bar')).toHaveCount(0, { timeout: 5_000 })

  await clickNavRail(page, '日志检索')
  await expect(page).toHaveURL(/#logs($|\?)/, { timeout: 5_000 })
  await clickNavRail(page, '全局设置')
  await expect(page).toHaveURL(/#settings($|\?)/, { timeout: 5_000 })
  await expect(warnLimitInput).toHaveValue('5', { timeout: 10_000 })
  await expect(banExpressionInput).toHaveValue('{t}h')

  await page.getByRole('button', { name: '恢复默认', exact: true }).click()
  const defaultDialog = confirmDialog(page, '恢复默认设置')
  await expect(defaultDialog).toBeVisible({ timeout: 5_000 })
  await expect(defaultDialog.getByText('确定要将所有设置恢复为默认值吗？此操作将覆盖当前所有设置，需要保存后才会生效。')).toBeVisible()
  await defaultDialog.getByRole('button', { name: '确认', exact: true }).click()

  await expect(page.getByText('已恢复默认设置，请保存以应用更改')).toBeVisible({ timeout: 10_000 })
  await expect(warnLimitInput).toHaveValue('3')
  await expect(banExpressionInput).toHaveValue('{t}^2h')
  await expect(page.locator('.save-bar', { hasText: '检测到未保存的修改' })).toBeVisible()
  await page.getByRole('button', { name: '保存更改', exact: true }).click()

  await expect(toastMessage(page, '设置已保存')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.save-bar')).toHaveCount(0, { timeout: 5_000 })

  tracker.assertClean()
})

test('role management creates, edits, assigns member, revokes member, and deletes a custom role', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  const suffix = Date.now().toString().slice(-8)
  const roleName = `E2E 角色 ${suffix}`
  const roleAlias = `e2e-${suffix}`
  const memberId = `30${suffix}`

  await page.evaluate(() => {
    const state = globalThis as unknown as { __roleClipboardWrites?: string[] }
    state.__roleClipboardWrites = []
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: async (text: string) => {
          state.__roleClipboardWrites?.push(text)
        },
      },
    })
  })

  await clickNavRail(page, '角色权限')
  await expect(page).toHaveURL(/#roles($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.roles-view-container')).toBeVisible({ timeout: 10_000 })

  await page.getByRole('button', { name: '＋', exact: true }).click()
  await expect(toastMessage(page, '角色创建成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.content-header h1', { hasText: '新角色' })).toBeVisible({ timeout: 10_000 })

  await fillRoleInput(page, '角色名称', roleName)
  await fillRoleInput(page, '角色别名', roleAlias)
  await fillRoleInput(page, '角色颜色', '#3b82f6')
  await expect(page.locator('.roles-view-container .save-bar', { hasText: '检测到未保存的修改' })).toBeVisible()
  await page.getByRole('button', { name: '保存更改', exact: true }).click()

  await expect(toastMessage(page, '保存成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.role-item', { hasText: roleName }).first()).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.roles-view-container .save-bar')).toHaveCount(0, { timeout: 5_000 })

  const roleId = (await page.locator('.role-id-code').textContent())?.trim()
  if (!roleId) {
    throw new Error('role id is not rendered')
  }
  await page.locator('.copy-btn').click()
  await expect(toastMessage(page, '角色 ID 已复制到剪贴板')).toBeVisible({ timeout: 10_000 })
  await expect
    .poll(() =>
      page.evaluate(() => {
        const state = globalThis as unknown as { __roleClipboardWrites?: string[] }
        return state.__roleClipboardWrites ?? []
      }),
    )
    .toContain(roleId)

  await page.locator('.tab-item', { hasText: '成员' }).click()
  await page.locator('.add-member input[placeholder="输入用户 ID 添加..."]').fill(memberId)
  await page.getByRole('button', { name: '添加成员', exact: true }).click()

  await expect(toastMessage(page, '添加成员成功')).toBeVisible({ timeout: 10_000 })
  const memberRow = page.locator('.member-item', { hasText: memberId }).first()
  await expect(memberRow).toBeVisible({ timeout: 10_000 })

  await memberRow.getByRole('button', { name: '移除', exact: true }).click()
  await expect(toastMessage(page, '移除成员成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.member-item', { hasText: memberId })).toHaveCount(0, { timeout: 10_000 })
  await expect(page.locator('.empty-tip', { hasText: '暂无成员（输入用户 QQ 号添加）' })).toBeVisible()

  await page.getByRole('button', { name: '删除角色', exact: true }).click()
  const deleteDialog = confirmDialog(page, '删除角色')
  await expect(deleteDialog).toBeVisible({ timeout: 5_000 })
  await expect(deleteDialog.getByText(`确定要删除角色"${roleName}"吗？此操作不可撤销。`)).toBeVisible()
  await deleteDialog.getByRole('button', { name: '确认', exact: true }).click()

  await expect(toastMessage(page, '删除成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.role-item', { hasText: roleName })).toHaveCount(0, { timeout: 10_000 })

  tracker.assertClean()
})

test('role management imports members from another custom role', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  const suffix = Date.now().toString().slice(-8)
  const sourceRoleName = `E2E 来源角色 ${suffix}`
  const targetRoleName = `E2E 目标角色 ${suffix}`
  const importedMemberId = `40${suffix}`

  await clickNavRail(page, '角色权限')
  await expect(page).toHaveURL(/#roles($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.roles-view-container')).toBeVisible({ timeout: 10_000 })

  await createNamedRole(page, sourceRoleName, `src-${suffix}`)
  await page.locator('.tab-item', { hasText: '成员' }).click()
  await page.locator('.add-member input[placeholder="输入用户 ID 添加..."]').fill(importedMemberId)
  await page.getByRole('button', { name: '添加成员', exact: true }).click()
  await expect(toastMessage(page, '添加成员成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.member-item', { hasText: importedMemberId }).first()).toBeVisible({
    timeout: 10_000,
  })

  await createNamedRole(page, targetRoleName, `dst-${suffix}`)
  await page.locator('.tab-item', { hasText: '成员' }).click()
  await expect(page.locator('.empty-tip', { hasText: '暂无成员（输入用户 QQ 号添加）' })).toBeVisible({
    timeout: 10_000,
  })

  await page.getByRole('button', { name: '导入成员', exact: true }).click()
  const importDialog = roleDialog(page, '导入成员')
  await expect(importDialog).toBeVisible({ timeout: 5_000 })
  await importDialog
    .locator('.form-group', { hasText: '选择角色' })
    .locator('select')
    .selectOption({ label: sourceRoleName })

  const previewRow = importDialog.locator('.preview-item', { hasText: importedMemberId }).first()
  await expect(previewRow).toBeVisible({ timeout: 10_000 })
  await expect(importDialog.locator('.preview-count')).toHaveText('已选 1 / 1')
  await importDialog.getByRole('button', { name: '导入 (1)', exact: true }).click()

  await expect(toastMessage(page, '成功导入 1 个成员')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.member-item', { hasText: importedMemberId }).first()).toBeVisible({
    timeout: 10_000,
  })

  await deleteCurrentRole(page, targetRoleName)
  await page.locator('.role-item', { hasText: sourceRoleName }).first().click()
  await deleteCurrentRole(page, sourceRoleName)

  tracker.assertClean()
})

test('role management imports members from authority and guild admin sources', async ({ loggedInPage: page }) => {
  await using tracker = createTracker(page)

  const suffix = Date.now().toString().slice(-8)
  const targetRoleName = `E2E 多源导入角色 ${suffix}`
  const authorityMemberId = '100003'
  const ownerMemberId = '100001'
  const adminMemberId = '100002'

  await clickNavRail(page, '角色权限')
  await expect(page).toHaveURL(/#roles($|\?)/, { timeout: 5_000 })
  await expect(page.locator('.roles-view-container')).toBeVisible({ timeout: 10_000 })

  await createNamedRole(page, targetRoleName, `multi-${suffix}`)
  await page.locator('.tab-item', { hasText: '成员' }).click()

  await page.getByRole('button', { name: '导入成员', exact: true }).click()
  let importDialog = roleDialog(page, '导入成员')
  await expect(importDialog).toBeVisible({ timeout: 5_000 })
  await importDialog.getByText('从 Authority 等级导入', { exact: true }).click()
  await importDialog
    .locator('.form-group', { hasText: '选择权限等级' })
    .locator('select')
    .selectOption('2')
  await expect(importDialog.locator('.preview-item', { hasText: authorityMemberId }).first()).toBeVisible({
    timeout: 10_000,
  })
  await expect(importDialog.locator('.preview-item', { hasText: 'E2E Authority 导入用户' }).first()).toBeVisible()
  await expect(importDialog.locator('.preview-count')).toHaveText('已选 1 / 1')
  await importDialog.getByRole('button', { name: '导入 (1)', exact: true }).click()

  await expect(toastMessage(page, '成功导入 1 个成员')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.member-item', { hasText: authorityMemberId }).first()).toBeVisible({
    timeout: 10_000,
  })

  await page.getByRole('button', { name: '导入成员', exact: true }).click()
  importDialog = roleDialog(page, '导入成员')
  await expect(importDialog).toBeVisible({ timeout: 5_000 })
  await importDialog.getByText('从群管理员导入', { exact: true }).click()
  await importDialog.locator('input[placeholder="请输入群号..."]').fill('1001')
  await importDialog.getByRole('button', { name: '获取', exact: true }).click()

  await expect(importDialog.locator('.preview-item', { hasText: ownerMemberId }).first()).toBeVisible({
    timeout: 10_000,
  })
  await expect(importDialog.locator('.preview-item', { hasText: adminMemberId }).first()).toBeVisible()
  await expect(importDialog.locator('.preview-item', { hasText: 'E2E 聊天用户' })).toHaveCount(0)
  await expect(importDialog.locator('.preview-count')).toHaveText('已选 2 / 2')
  await importDialog.getByRole('button', { name: '导入 (2)', exact: true }).click()

  await expect(toastMessage(page, '成功导入 2 个成员')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.member-item', { hasText: authorityMemberId }).first()).toBeVisible({
    timeout: 10_000,
  })
  await expect(page.locator('.member-item', { hasText: ownerMemberId }).first()).toBeVisible()
  await expect(page.locator('.member-item', { hasText: adminMemberId }).first()).toBeVisible()

  await deleteCurrentRole(page, targetRoleName)

  tracker.assertClean()
})

/**
 * 通过 NavRail 的 button[title=label] 切到指定 view。
 * NavRail 收起时 label 文本 opacity:0 不可读，但 button 元素本身可点击；
 * title attribute 在两种状态下都准确反映 view label，是稳定的 selector key。
 */
interface UiSmokeChatActions {
  sentMessages: Array<{
    channelId: string
    guildId?: string
    content: string
    messageId: string
  }>
  recalledMessages: Array<{
    channelId: string
    messageId: string
  }>
}

async function uiSmokeChatPost(page: Page, path: string, body: Record<string, unknown>): Promise<void> {
  await page.evaluate(async ({ path, body }) => {
    const response = await fetch(path, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(body),
    })
    const result = await response.json()
    if (!response.ok || !result.success) {
      throw new Error(result.error || `ui smoke chat seed failed: ${response.status}`)
    }
  }, { path, body })
}

async function uiSmokeChatActions(page: Page): Promise<UiSmokeChatActions> {
  return page.evaluate(async () => {
    const response = await fetch('/__stuhelper-ui-smoke/chat/actions')
    const result = await response.json()
    if (!response.ok || !result.success) {
      throw new Error(result.error || `ui smoke chat actions failed: ${response.status}`)
    }
    return {
      sentMessages: result.sentMessages ?? [],
      recalledMessages: result.recalledMessages ?? [],
    }
  })
}

async function pasteTinyPng(page: Page): Promise<void> {
  await page.evaluate(() => {
    const base64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII='
    const bytes = Uint8Array.from(atob(base64), (char) => char.charCodeAt(0))
    const file = new File([bytes], 'ui-smoke.png', { type: 'image/png' })
    const dataTransfer = new DataTransfer()
    dataTransfer.items.add(file)

    const input = document.querySelector<HTMLTextAreaElement>('.sh-dock[data-open="true"] .chat-input')
    if (!input) {
      throw new Error('chat input is not mounted')
    }

    const event = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', { value: dataTransfer })
    input.dispatchEvent(event)
  })
}

async function clickNavRail(page: Page, label: string): Promise<void> {
  const button = page.locator(`.sh-rail__item[title="${label}"]`)
  await expect(button.first()).toBeAttached({ timeout: 5_000 })
  await button.first().click()
}

async function fillLabeledInput(page: Page, label: string, value: string): Promise<void> {
  await page.locator('label', { hasText: label }).locator('input').first().fill(value)
}

async function selectLabeledOption(page: Page, label: string, option: string): Promise<void> {
  await page.locator('label', { hasText: label }).locator('.el-select__wrapper').first().click()
  await page.locator('.el-select-dropdown__item:visible', { hasText: option }).click()
}

async function setFeatureChecked(page: Page, label: string, checked: boolean): Promise<void> {
  const feature = page.locator('.sh-sub-feature', { hasText: label }).first()
  await expect(feature).toBeVisible({ timeout: 5_000 })

  const input = feature.locator('input[type="checkbox"]').first()
  if (await input.isChecked() !== checked) {
    await feature.locator('.el-checkbox').first().click()
  }

  if (checked) {
    await expect(input).toBeChecked()
  } else {
    await expect(input).not.toBeChecked()
  }
}

function confirmDialog(page: Page, title: string) {
  return page.locator('.el-dialog.sh-confirm:visible', { hasText: title }).first()
}

function settingsInput(page: Page, label: string) {
  return page
    .locator('.settings-view .form-row')
    .filter({ has: page.locator('.form-label', { hasText: label }) })
    .locator('input')
    .first()
}

async function fillSettingsInput(page: Page, label: string, value: string): Promise<void> {
  await settingsInput(page, label).fill(value)
}

function toastMessage(page: Page, text: string) {
  return page.locator('.el-message__content', { hasText: text }).last()
}

function configEditDialog(page: Page, guildId: string) {
  return page.locator('.config-view .edit-dialog', { hasText: `编辑群组配置 - ${guildId}` }).first()
}

function configFormGroup(dialog: ReturnType<typeof configEditDialog>, label: string) {
  return dialog.locator('.form-group', { hasText: label }).first()
}

function configInput(dialog: ReturnType<typeof configEditDialog>, label: string) {
  return configFormGroup(dialog, label).locator('input').first()
}

function configTextarea(dialog: ReturnType<typeof configEditDialog>, label: string) {
  return configFormGroup(dialog, label).locator('textarea').first()
}

function roleDialog(page: Page, title: string) {
  return page.locator('.roles-view-container .modal-dialog', { hasText: title }).first()
}

function roleInput(page: Page, label: string) {
  return page
    .locator('.roles-view-container .form-group')
    .filter({ has: page.locator('label', { hasText: label }) })
    .locator('input')
    .last()
}

async function fillRoleInput(page: Page, label: string, value: string): Promise<void> {
  await roleInput(page, label).fill(value)
}

async function createNamedRole(page: Page, name: string, alias: string): Promise<void> {
  await page.getByRole('button', { name: '＋', exact: true }).click()
  await expect(toastMessage(page, '角色创建成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.content-header h1', { hasText: '新角色' })).toBeVisible({ timeout: 10_000 })

  await fillRoleInput(page, '角色名称', name)
  await fillRoleInput(page, '角色别名', alias)
  await page.getByRole('button', { name: '保存更改', exact: true }).click()

  await expect(toastMessage(page, '保存成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.role-item', { hasText: name }).first()).toBeVisible({ timeout: 10_000 })
}

async function deleteCurrentRole(page: Page, roleName: string): Promise<void> {
  await page.getByRole('button', { name: '删除角色', exact: true }).click()
  const deleteDialog = confirmDialog(page, '删除角色')
  await expect(deleteDialog).toBeVisible({ timeout: 5_000 })
  await expect(deleteDialog.getByText(`确定要删除角色"${roleName}"吗？此操作不可撤销。`)).toBeVisible()
  await deleteDialog.getByRole('button', { name: '确认', exact: true }).click()

  await expect(toastMessage(page, '删除成功')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.role-item', { hasText: roleName })).toHaveCount(0, { timeout: 10_000 })
}
