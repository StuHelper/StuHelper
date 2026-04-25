import { test, expect } from './fixtures/auth'
import type { Page } from '@playwright/test'

/**
 * P0b：StuHelper 群管中心 11 个 view 的端到端导航回归基线。
 *
 * - 每个 test 自动通过 fixture 完成 admin 登录（fresh context）
 * - 第一个 test 验证 TopNavigation 点击切换走通（保护用户路径）
 * - 后续 11 个 view 走 URL 直跳 `?view=<id>`，断言主区域出现 + 无 pageerror
 *
 * 11 个 view ID 来自 stuhelper-core/client/composables/use-console-pages.ts。
 * 顺序与 stuhelper-core/client/pages/index.vue 的 allMenuItems 一致。
 */

interface ViewSpec {
  readonly id: string
  readonly label: string
}

const VIEWS: readonly ViewSpec[] = [
  { id: 'dashboard', label: '仪表盘' },
  { id: 'config', label: '配置治理' },
  { id: 'warns', label: '警告记录' },
  { id: 'blacklist', label: '黑名单' },
  { id: 'identity', label: '身份认证' },
  { id: 'review', label: '处置中心' },
  { id: 'roles', label: '角色权限' },
  { id: 'logs', label: '日志检索' },
  { id: 'chat', label: '实时聊天' },
  { id: 'subscriptions', label: '订阅管理' },
  { id: 'settings', label: '设置' },
]

test.describe.configure({ mode: 'serial' })

test('TopNavigation click switches between views', async ({ loggedInPage: page }) => {
  const errors = trackPageErrors(page)

  await page.goto('/stuhelper', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.stuhelperGroupCenter-app')).toBeVisible({ timeout: 10_000 })

  // 点击 dashboard tab（默认就是它，但点一次确保 nav 链路工作）
  await page.locator('.nav-tab', { hasText: VIEWS[0].label }).first().click()
  await expect(page).toHaveURL(/[?&]view=dashboard($|&)/, { timeout: 5_000 })

  // 切到第二个 view，验证 nav 真的能切
  await page.locator('.nav-tab', { hasText: VIEWS[1].label }).first().click()
  await expect(page).toHaveURL(/[?&]view=config($|&)/, { timeout: 5_000 })

  expectNoPageErrors(errors)
})

for (const view of VIEWS) {
  test(`view "${view.id}" loads without page errors`, async ({ loggedInPage: page }) => {
    const errors = trackPageErrors(page)

    await page.goto(`/stuhelper?view=${view.id}`, { waitUntil: 'domcontentloaded' })
    await expect(page.locator('.stuhelperGroupCenter-app')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.stuhelperGroupCenter-app .main-content')).toBeVisible({
      timeout: 10_000,
    })

    // 不断言 .nav-tab.active：第 9 之后的 view 会被 TopNavigation 折进 More 菜单，
    // active 状态挂在 More 按钮容器上而非 .nav-tab。nav 链路工作由第一个 test 覆盖。
    expectNoPageErrors(errors)
  })
}

function trackPageErrors(page: Page): Error[] {
  const errors: Error[] = []
  page.on('pageerror', (error) => {
    errors.push(error)
  })
  return errors
}

function expectNoPageErrors(errors: readonly Error[]): void {
  expect(
    errors,
    `unexpected pageerror: ${errors.map((error) => error.message).join('; ')}`,
  ).toHaveLength(0)
}
