import { expect, test } from '@playwright/test'

/**
 * P0a smoke：仅验证 Koishi Console 端口可达且无前端运行时错误。
 * 不涉及 admin 登录与 11 view 导航——那是 P0b 的职责。
 */

test('Koishi console root is reachable without page errors', async ({ page }) => {
  const errors: Error[] = []
  page.on('pageerror', (error) => {
    errors.push(error)
  })

  const response = await page.goto('/', { waitUntil: 'domcontentloaded' })
  expect(response?.ok(), 'console root should respond 2xx').toBe(true)

  await expect(page).toHaveTitle(/Koishi/i, { timeout: 10_000 })

  expect(errors, `unexpected pageerror: ${errors.map((error) => error.message).join('; ')}`)
    .toHaveLength(0)
})
