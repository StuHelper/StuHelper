import type { Page } from './fixtures';

import { expect, test } from './fixtures';

const oidcDestinationURL = 'about:blank';

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
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          state: 'e2e-state',
          url: oidcDestinationURL,
        },
      }),
    });
  });
}

async function mockForbiddenAdminSession(page: Page) {
  const observed = {
    loginRequests: 0,
    logoutRequests: 0,
  };

  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          id: '100',
          name: 'student-user',
          displayName: 'Student User',
          email: 'student@example.com',
          avatar: '',
          roles: ['verified_student'],
          capabilities: ['review:list:full'],
          globalCapabilities: ['review:list:full'],
          capabilityGrants: [],
          canAccessAdmin: false,
          isPlatformAdmin: false,
        },
      }),
    });
  });
  await page.route('**/api/v1/auth/logout', async (route) => {
    observed.logoutRequests += 1;
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true }),
    });
  });
  await page.route('**/api/v1/auth/login**', async (route) => {
    observed.loginRequests += 1;
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          state: 'forbidden-e2e-state',
          url: 'about:blank',
        },
      }),
    });
  });

  return observed;
}

test('admin login route initiates OIDC login', async ({ page }) => {
  await mockAnonymousSession(page);
  const loginRequest = page.waitForRequest('**/api/v1/auth/login**');
  const loginResponse = page.waitForResponse('**/api/v1/auth/login**');

  await page.goto('/auth/login');
  const request = await loginRequest;
  await loginResponse;
  const requestURL = new URL(request.url());

  await expect(page).toHaveURL(oidcDestinationURL);
  expect(requestURL.searchParams.get('app')).toBe('admin');
  expect(requestURL.searchParams.get('redirect')).toContain('/analytics');
});

test('root route initiates OIDC login when unauthenticated', async ({
  page,
}) => {
  await mockAnonymousSession(page);
  const loginRequest = page.waitForRequest('**/api/v1/auth/login**');
  const loginResponse = page.waitForResponse('**/api/v1/auth/login**');
  await page.goto('/');
  const request = await loginRequest;
  await loginResponse;
  const requestURL = new URL(request.url());

  await expect(page).toHaveURL(oidcDestinationURL);
  await expect
    .poll(() => requestURL.searchParams.get('redirect'))
    .toContain('/analytics');
});

test('authenticated user without admin access sees forbidden page before re-login', async ({
  page,
}) => {
  const observed = await mockForbiddenAdminSession(page);

  await page.goto('/analytics');

  await expect(page).toHaveURL(/\/auth\/login\?error=forbidden/);
  await expect(
    page.getByRole('heading', { name: /无权访问|Access Denied/ }),
  ).toBeVisible();
  await expect(
    page.getByText(/没有管理后台的访问权限|does not have permission/),
  ).toBeVisible();
  expect(observed.loginRequests).toBe(0);
  expect(observed.logoutRequests).toBe(0);

  await page
    .getByRole('button', { name: /使用其他账号登录|Try a Different Account/ })
    .click();

  await expect.poll(() => observed.logoutRequests).toBe(1);
  await expect.poll(() => observed.loginRequests).toBe(1);
  await expect(page).toHaveURL('about:blank');
});
