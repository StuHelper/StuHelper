import axe from 'axe-core'

import { expect, test, type Page } from './fixtures'

type AxeViolation = {
  help: string
  id: string
  impact: string | null
  nodes: Array<{
    failureSummary?: string
    target: string[]
  }>
}

async function mockPublicShell(page: Page) {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: 'A0010100', message: 'login required' },
      }),
    }),
  )
  await page.route('**/api/v1/course/stats', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { courseCount: 120, departmentCount: 8 },
      }),
    }),
  )
  await page.route('**/api/v1/course/review/stats', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          courseCount: 120,
          reviewCount: 580,
          departmentCount: 8,
          userCount: 230,
        },
      }),
    }),
  )
}

async function expectNoWcagAAViolations(page: Page) {
  await page.addScriptTag({ content: axe.source })
  const violations = await page.evaluate(async () => {
    const axeRuntime = (
      window as unknown as {
        axe: {
          run(
            context: Document,
            options: {
              resultTypes: string[]
              runOnly: { type: string; values: string[] }
            },
          ): Promise<{ violations: AxeViolation[] }>
        }
      }
    ).axe
    const results = await axeRuntime.run(document, {
      runOnly: {
        type: 'tag',
        values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'],
      },
      resultTypes: ['violations'],
    })
    return results.violations.map((violation) => ({
      help: violation.help,
      id: violation.id,
      impact: violation.impact,
      nodes: violation.nodes.map((node) => ({
        failureSummary: node.failureSummary,
        target: node.target,
      })),
    }))
  })

  expect(
    violations,
    `WCAG A/AA violations:\n${JSON.stringify(violations, null, 2)}`,
  ).toEqual([])
}

test.describe('WCAG A/AA browser baseline', () => {
  test.beforeEach(async ({ page }) => {
    await mockPublicShell(page)
  })

  for (const scenario of [
    { name: 'home', path: '/' },
    { name: 'about', path: '/about' },
    { name: 'privacy', path: '/privacy' },
    { name: 'terms', path: '/terms' },
    { name: 'not found', path: '/this-route-does-not-exist' },
  ]) {
    test(`${scenario.name} page has no detectable WCAG A/AA violations`, async ({
      page,
    }) => {
      await page.goto(scenario.path)
      await page.waitForLoadState('networkidle')

      await expectNoWcagAAViolations(page)
    })
  }

  test('dark home page has no detectable WCAG A/AA violations', async ({
    page,
  }) => {
    await page.addInitScript(() => {
      window.localStorage.setItem('theme-mode', 'dark')
    })
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')

    await expectNoWcagAAViolations(page)
  })
})
