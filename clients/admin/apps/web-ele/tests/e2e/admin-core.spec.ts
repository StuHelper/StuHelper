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

const userSystemReadOnlyCapabilities = [
  'user:school:read',
  'user:system:read',
  'member_blacklist:read',
];

const userSystemReadOnlyAdminUser = {
  ...adminUser,
  capabilities: userSystemReadOnlyCapabilities,
  globalCapabilities: userSystemReadOnlyCapabilities,
  capabilityGrants: userSystemReadOnlyCapabilities.map((name) => ({
    name,
    global: true,
    scopeRoles: [],
    scopeSchoolIDs: [],
    scopeSectionIDs: [],
  })),
};

const stats = {
  totalReviews: 128,
  todayReviews: 7,
  weekReviews: 31,
  publishedReviews: 113,
  hiddenReviews: 4,
  deletedReviews: 11,
  totalReports: 18,
  pendingReports: 3,
};

const schoolConfig = {
  approvalPolicy: 'manual',
  schoolID: 4_111_010_006,
  schoolCode: '4111010006',
  schoolName: '只读大学',
  verificationMethod: 'ldap',
  enabled: true,
  schoolSsoEnabled: false,
  schoolEmailOtpEnabled: false,
  academicDbTable: 'academic_students',
  consentText: '只读认证授权说明',
  ldapConfig: {
    url: 'ldap://readonly.example.com',
    baseDN: 'dc=readonly,dc=example',
    systemBindDN: 'cn=reader,dc=readonly,dc=example',
    useTLS: true,
  },
};

const systemConfig = {
  key: 'review.retention_days',
  value: '365',
  description: '评课保留天数',
  updatedAt: '2026-05-26T04:00:00Z',
};

const blacklistEntry = {
  id: 'entry-readonly',
  platform: 'qq',
  subjectType: 'qq_user',
  subjectID: '30001',
  scopeType: 'guild',
  guildID: 'guild-readonly',
  source: 'admission_failure',
  reasonCode: 'admission_timeout_limit',
  reasonText: 'read-only visible reason',
  createdFrom: 'admin_console',
  createdByType: 'admin_user',
  createdByID: 'admin-1',
  createdAt: '2026-05-26T04:00:00Z',
  updatedAt: '2026-05-26T04:00:00Z',
  expiresAt: null,
  releasedAt: null,
  releasedByType: null,
  releasedByID: null,
  releaseReasonCode: null,
  releaseReason: null,
  metadata: {},
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

async function mockAdminStats(page: Page) {
  await page.route('**/api/v1/course/review/admin/stats', async (route) => {
    await route.fulfill(ok(stats));
  });
}

async function mockReadOnlyUserSystemApi(
  page: Page,
  mutatingRequests: string[],
) {
  await mockAdminSession(page, userSystemReadOnlyAdminUser);

  await page.route('**/api/v1/admin/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    if (path === '/api/v1/admin/school-configs' && method === 'GET') {
      await route.fulfill(ok([schoolConfig]));
      return;
    }

    if (path === '/api/v1/admin/system-configs' && method === 'GET') {
      await route.fulfill(ok([systemConfig]));
      return;
    }

    if (path === '/api/v1/admin/member-blacklist' && method === 'GET') {
      await route.fulfill(
        ok({
          items: [blacklistEntry],
          list: [blacklistEntry],
          total: 1,
        }),
      );
      return;
    }

    mutatingRequests.push(`${method} ${path}`);
    await route.fulfill(ok({}));
  });
}

test.describe('Admin core shell routes', () => {
  test.beforeEach(async ({ page }) => {
    await mockAdminSession(page);
  });

  test('profile route renders real account data and external account settings', async ({
    page,
  }) => {
    await page.goto('/profile');

    const main = page.locator('main');

    await expect(main.getByText('Platform Admin').first()).toBeVisible();
    await expect(page.getByRole('tab', { name: '账号资料' })).toBeVisible();
    await expect(page.getByText('用户名')).toBeVisible();
    await expect(
      main.getByRole('cell', { name: 'platform-admin' }),
    ).toBeVisible();
    await expect(
      main.getByRole('cell', { name: 'admin@example.com' }),
    ).toBeVisible();
    await expect(main.getByText('platform_admin')).toBeVisible();
    await expect(page.getByRole('tab', { name: '安全设置' })).toHaveCount(0);
    await expect(page.getByRole('tab', { name: '新消息提醒' })).toHaveCount(0);

    await page.getByRole('tab', { name: '账户设置' }).click();
    await expect(page.getByText('由身份提供商管理')).toBeVisible();
  });

  test('profile route load failures show a persistent retry path', async ({
    page,
  }) => {
    let authMeCalls = 0;
    await page.route('**/api/v1/auth/me', async (route) => {
      authMeCalls += 1;
      if (authMeCalls === 2) {
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: false,
            error: {
              code: 'E2E_PROFILE_UNAVAILABLE',
              message: 'profile temporarily unavailable',
            },
          }),
        });
        return;
      }

      await route.fulfill(ok(adminUser));
    });

    await page.goto('/profile');

    const loadError = page.locator('.admin-load-error', {
      hasText: 'profile temporarily unavailable',
    });
    await expect(loadError).toBeVisible();
    await expect(
      page.getByRole('cell', { name: 'platform-admin' }),
    ).toHaveCount(0);

    await loadError.getByRole('button', { name: /重试|Retry/ }).click();

    await expect(
      page.getByRole('cell', { name: 'platform-admin' }),
    ).toBeVisible();
    await expect(loadError).toHaveCount(0);
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

test.describe('Admin user-system read-only capability boundaries', () => {
  test('read-only user can view data but cannot see mutating controls', async ({
    page,
  }) => {
    const mutatingRequests: string[] = [];
    await mockReadOnlyUserSystemApi(page, mutatingRequests);

    await page.goto('/users/school-config');
    await expect(page.getByText('只读大学')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('人工审核')).toBeVisible();
    await expect(page.getByText('未接入')).toBeVisible();
    await expect(page.getByText('ldap://readonly.example.com')).toBeVisible();
    await expect(page.getByRole('button', { name: /编辑|Edit/ })).toHaveCount(
      0,
    );

    await page.goto('/users/system-config');
    await expect(page.getByText('review.retention_days')).toBeVisible();
    await expect(page.getByText('评课保留天数')).toBeVisible();
    await expect(page.getByRole('button', { name: /编辑|Edit/ })).toHaveCount(
      0,
    );

    await page.goto('/users/member-blacklist');
    await expect(
      page.getByRole('heading', { name: '成员黑名单' }),
    ).toBeVisible();
    await expect(page.getByText('read-only visible reason')).toBeVisible();
    await expect(page.getByRole('button', { name: '新增黑名单' })).toHaveCount(
      0,
    );
    await expect(page.getByRole('button', { name: '解除' })).toHaveCount(0);

    expect(mutatingRequests).toEqual([]);
  });
});

test.describe('Admin capability route filtering', () => {
  test.beforeEach(async ({ page }) => {
    await mockAdminSession(page, dashboardOnlyAdminUser);
    await mockAdminStats(page);
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

  test('limited admin dashboard hides shortcuts for inaccessible routes', async ({
    page,
  }) => {
    await page.goto('/analytics');

    await expect(
      page.getByRole('heading', { name: /分析页|Analytics/ }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('button', { name: '教师管理' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '举报管理' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '评课管理' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '实名审核' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '学生认证' })).toHaveCount(0);

    await page.goto('/workspace');
    await expect(page.getByText('评课总量')).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByRole('button', { name: /待处理举报/ })).toHaveCount(
      0,
    );
    await expect(page.getByRole('button', { name: /隐藏评课/ })).toHaveCount(0);
    await expect(
      page.getByRole('button', { name: /本周新增评课/ }),
    ).toHaveCount(0);
    await expect(page.getByRole('button', { name: '成员黑名单' })).toHaveCount(
      0,
    );
  });
});
