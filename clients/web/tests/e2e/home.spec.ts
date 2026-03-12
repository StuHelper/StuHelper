import { test, expect } from '@playwright/test'

test('home page renders shell and brand', async ({ page }) => {
  await page.goto('/')

  await expect(page).toHaveTitle(/StuHelper/i)
  await expect(page.getByRole('link', { name: /StuHelper/i }).first()).toBeVisible()
})
