import type { Page } from '@playwright/test';

import { expect, test } from '@playwright/test';

async function mockAnonymousSession(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 'A1000401',
        message: 'unauthorized',
        success: false,
      }),
    });
  });

  await page.route('**/api/v1/auth/login**', async (route) => {
    const loginUrl = new URL(
      '/admin/auth/login',
      page.url() || 'http://127.0.0.1:4174',
    );
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          state: 'e2e-state',
          url: loginUrl.toString(),
        },
      }),
    });
  });
}

test('admin login route initiates OIDC login', async ({ page }) => {
  await mockAnonymousSession(page);
  const loginRequest = page.waitForRequest('**/api/v1/auth/login**');

  await page.goto('/auth/login');
  const request = await loginRequest;
  const requestURL = new URL(request.url());

  await expect(page).toHaveTitle(/StuHelper Admin/i);
  expect(requestURL.searchParams.get('app')).toBe('admin');
  expect(requestURL.searchParams.get('redirect')).toContain('/analytics');
});

test('root route initiates OIDC login when unauthenticated', async ({
  page,
}) => {
  await mockAnonymousSession(page);
  const loginRequest = page.waitForRequest('**/api/v1/auth/login**');
  await page.goto('/');
  const request = await loginRequest;
  const requestURL = new URL(request.url());

  await expect
    .poll(() => requestURL.searchParams.get('redirect'))
    .toContain('/analytics');
});
