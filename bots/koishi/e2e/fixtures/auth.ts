import { test as base, expect, type Page } from '@playwright/test'

/**
 * Koishi Console 登录 fixture。
 *
 * Koishi auth 通过 WebSocket 事件 `login/password(name, password)` 完成登录；
 * 登录态以 token 形式存到浏览器 localStorage 的 "auth" key。
 *
 * 这里走真实 UI 输入路径（`/login` 页面填表 + 点击"登录"按钮），原因：
 * - 验证登录页本身没坏（顺带是 P0b 的回归保护）
 * - 不依赖 Koishi 内部 socket 协议细节，UI 改动也不影响
 *
 * fixture 在每个 test 的 beforeEach 自动跑；登录成功后跳转到 `/profile`，
 * 该页存在与否表示登录成功。
 */

const ADMIN_USERNAME = 'admin'
const FALLBACK_PASSWORD = 'ui-smoke-password'

interface LoggedInFixtures {
  loggedInPage: Page
}

export const test = base.extend<LoggedInFixtures>({
  loggedInPage: async ({ page }, use) => {
    await loginAsAdmin(page)
    await use(page)
  },
})

export { expect }

async function loginAsAdmin(page: Page): Promise<void> {
  const password = process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD ?? FALLBACK_PASSWORD

  await page.goto('/login', { waitUntil: 'domcontentloaded' })

  // Koishi auth 登录页用 placeholder 标识 input；placeholder selector 比 nth-child 更稳
  await page.locator('input[placeholder="用户名"]').fill(ADMIN_USERNAME)
  await page.locator('input[placeholder="密码"]').fill(password)

  // <k-button> 渲染后是 native button，按可访问名称定位
  await page.getByRole('button', { name: '登录', exact: true }).click()

  // 登录成功后 store.user 被设置，watcher 跳转到 /profile
  await expect(page).toHaveURL(/\/profile($|\?)/, { timeout: 10_000 })
}
