import { test as base, expect, type Page } from '@playwright/test'
import { createTracker } from './diagnostics'

/**
 * Koishi Console 共享登录 fixture。
 *
 * 设计要点：
 * - `consolePage` 是 **worker-scoped** fixture：整个 worker 生命周期只跑一次
 *   登录 + SPA 路由 warm-up，所有 test 共享同一个 page
 * - 这样规避了"每个 test 各自 page.goto 触发 page reload + SPA 异步路由
 *   注册竞态"——这是 P0b 第一版的核心 flake 来源
 * - test 通过 SPA 内 navigation（`history.pushState` 经由 nav button click）
 *   切换 view，不触发 page reload
 * - `loggedInPage` 是 test-scoped 别名，让 spec 与传统 Playwright 写法一致
 *
 * Koishi auth 通过 WebSocket 事件 `login/password(name, password)` 完成登录；
 * 走真实 UI 输入路径以保护登录页本身的回归。
 */

const ADMIN_USERNAME = 'admin'
const FALLBACK_PASSWORD = 'ui-smoke-password'

interface WorkerFixtures {
  consolePage: Page
}

interface TestFixtures {
  loggedInPage: Page
}

export const test = base.extend<TestFixtures, WorkerFixtures>({
  consolePage: [
    async ({ browser }, use) => {
      const context = await browser.newContext()
      const page = await context.newPage()
      try {
        await using tracker = createTracker(page)
        await loginAndWarmUp(page)
        tracker.assertClean()
        await use(page)
      } finally {
        await context.close()
      }
    },
    { scope: 'worker' },
  ],

  loggedInPage: async ({ consolePage }, use) => {
    await use(consolePage)
  },
})

export { expect }

async function loginAndWarmUp(page: Page): Promise<void> {
  const password = process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD ?? FALLBACK_PASSWORD

  await page.goto('/login', { waitUntil: 'domcontentloaded' })

  await page.locator('input[placeholder="用户名"]').fill(ADMIN_USERNAME)
  await page.locator('input[placeholder="密码"]').fill(password)

  await page.getByRole('button', { name: '登录', exact: true }).click()

  // 登录成功后 store.user 被设置，watcher 跳转到 /profile
  await expect(page).toHaveURL(/\/profile($|\?)/, { timeout: 10_000 })

  // SPA 路由 warm-up：Koishi 通过 ctx.console.addEntry 把 /stuhelper 入口异步推到
  // 客户端。先等待侧边栏 entry 出现，再通过 Koishi 自己的 router link 进入页面，
  // 避免直接 page.goto('/stuhelper') 抢在 entry 注册前触发 profile fallback。
  await page.locator('a[href="/stuhelper"]').first().click()
  await expect(page.locator('.stuhelperGroupCenter-app')).toBeVisible({ timeout: 15_000 })
  // 等 NavRail 挂载，证明 stuhelper-core 客户端 AppShell 已渲染
  await expect(page.locator('.sh-rail__item').first()).toBeVisible({ timeout: 10_000 })
}
