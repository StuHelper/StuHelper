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

function ok(data: unknown) {
  return {
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ success: true, data }),
  };
}

async function mockAdminSession(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill(ok(adminUser));
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
