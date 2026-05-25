import type { Page } from './fixtures';

import { expect, test } from './fixtures';

const capabilities = [
  'admin:dashboard:view',
  'admin:reviews:manage',
  'admin:reports:manage',
  'admin:teachers:manage',
  'admin:sensitive_words:manage',
  'admin:logs:view',
  'user:identity:read',
  'user:identity:review',
  'user:student:review',
  'user:school:read',
  'user:school:update',
  'user:system:read',
  'user:system:update',
  'admission:freshman:review',
  'admission:policy:update',
  'member_blacklist:read',
  'member_blacklist:manage',
  'open_platform:manage',
];

const adminUser = {
  id: '99',
  name: 'platform-admin',
  displayName: 'Platform Admin',
  email: 'admin@example.com',
  avatar: '',
  roles: ['platform_admin'],
  capabilities,
  globalCapabilities: capabilities,
  capabilityGrants: capabilities.map((name) => ({
    name,
    global: true,
    scopeRoles: [],
    scopeSchoolIDs: [],
    scopeSectionIDs: [],
  })),
  canAccessAdmin: true,
  isPlatformAdmin: true,
};

const dashboardOnlyCapabilities = ['admin:dashboard:view'];

const dashboardOnlyAdminUser = {
  ...adminUser,
  capabilities: dashboardOnlyCapabilities,
  globalCapabilities: dashboardOnlyCapabilities,
  capabilityGrants: dashboardOnlyCapabilities.map((name) => ({
    name,
    global: true,
    scopeRoles: [],
    scopeSchoolIDs: [],
    scopeSectionIDs: [],
  })),
};

function ok(data: unknown) {
  return {
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ success: true, data }),
  };
}

async function mockAdminSession(page: Page, user = adminUser) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill(ok(user));
  });
}

test.describe('Admin core shell routes', () => {
  test.beforeEach(async ({ page }) => {
    await mockAdminSession(page);
  });

  test('profile route renders account tabs from the authenticated session', async ({
    page,
  }) => {
    await page.goto('/profile');

    const main = page.locator('main');

    await expect(main.getByText('Platform Admin').first()).toBeVisible();
    await expect(page.getByRole('tab', { name: '基本设置' })).toBeVisible();
    await expect(page.getByText('用户名')).toBeVisible();
    await expect(page.getByRole('textbox', { name: '用户名' })).toHaveValue(
      'platform-admin',
    );

    await page.getByRole('tab', { name: '安全设置' }).click();
    await expect(page.getByText('当前密码强度：强')).toBeVisible();

    await page.getByRole('tab', { name: '修改密码' }).click();
    await expect(page.getByText('由身份提供商管理')).toBeVisible();

    await page.getByRole('tab', { name: '新消息提醒' }).click();
    await expect(page.getByText('系统消息', { exact: true })).toBeVisible();
  });

  test('user dropdown confirms logout and starts admin SSO login', async ({
    page,
  }) => {
    let sessionActive = true;
    const observed: {
      loginRequest: null | {
        app: null | string;
        method: string;
        path: string;
        redirect: null | string;
      };
      logoutRequest: null | { method: string; path: string };
    } = {
      loginRequest: null,
      logoutRequest: null,
    };

    await page.route('**/api/v1/auth/me', async (route) => {
      if (!sessionActive) {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 'A1000401',
            message: 'unauthorized',
            success: false,
          }),
        });
        return;
      }

      await route.fulfill(ok(adminUser));
    });
    await page.route('**/api/v1/auth/logout', async (route) => {
      const url = new URL(route.request().url());
      observed.logoutRequest = {
        method: route.request().method(),
        path: url.pathname,
      };
      sessionActive = false;
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });
    await page.route('**/api/v1/auth/login**', async (route) => {
      const url = new URL(route.request().url());
      observed.loginRequest = {
        app: url.searchParams.get('app'),
        method: route.request().method(),
        path: url.pathname,
        redirect: url.searchParams.get('redirect'),
      };
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            state: 'logout-state',
            url: 'about:blank',
          },
        }),
      });
    });

    await page.goto('/profile');

    const avatar = page.getByRole('banner').getByRole('button', {
      name: 'IN',
    });
    await expect(avatar).toBeVisible();

    await avatar.click();
    await page.getByRole('menuitem', { name: /退出登录/ }).click();
    await expect(page.getByText('是否退出登录？')).toBeVisible();

    await page.getByRole('button', { name: '确认' }).click();

    await expect
      .poll(() => observed.logoutRequest)
      .toEqual({
        method: 'POST',
        path: '/api/v1/auth/logout',
      });
    await expect
      .poll(() => observed.loginRequest)
      .toMatchObject({
        app: 'admin',
        method: 'GET',
        path: '/api/v1/auth/login',
      });
    expect(observed.loginRequest?.redirect).toContain('/profile');

    await expect(page).toHaveURL('about:blank');
  });

  test('unknown authenticated route renders the admin 404 fallback', async ({
    page,
  }) => {
    await page.goto('/admin-route-that-does-not-exist');

    await expect(page.getByText('哎呀！未找到页面')).toBeVisible();
    await expect(
      page.getByText('抱歉，我们无法找到您要找的页面。'),
    ).toBeVisible();
  });
});

test.describe('Admin capability route filtering', () => {
  test.beforeEach(async ({ page }) => {
    await mockAdminSession(page, dashboardOnlyAdminUser);
  });

  test('limited admin cannot reach Open Platform route or trigger its API', async ({
    page,
  }) => {
    const openPlatformRequests: string[] = [];

    await page.route('**/api/v1/admin/open-platform/**', async (route) => {
      openPlatformRequests.push(new URL(route.request().url()).pathname);
      await route.fulfill(
        ok({
          list: [],
          total: 0,
          page: 1,
          pageSize: 20,
        }),
      );
    });

    await page.goto('/open-platform/apps');

    await expect(page.getByText('哎呀！未找到页面')).toBeVisible({
      timeout: 10_000,
    });
    await expect(
      page.getByText('抱歉，我们无法找到您要找的页面。'),
    ).toBeVisible();
    expect(openPlatformRequests).toEqual([]);
  });
});
