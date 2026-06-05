import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-05-24T04:00:00Z';

const adminUser = {
  id: '99',
  name: 'platform-admin',
  displayName: 'Platform Admin',
  email: 'admin@example.com',
  avatar: '',
  roles: ['platform_admin'],
  capabilities: ['open_platform:manage'],
  globalCapabilities: ['open_platform:manage'],
  capabilityGrants: [
    {
      name: 'open_platform:manage',
      global: true,
      scopeSchoolIDs: [],
      scopeSectionIDs: [],
      scopeRoles: [],
    },
  ],
  canAccessAdmin: true,
  isPlatformAdmin: true,
};

const baseApp = {
  id: 42,
  clientID: 'campus-connector',
  displayName: 'Campus Connector',
  description: 'Campus integration client',
  homepageURL: 'https://connector.example.com',
  privacyPolicyURL: 'https://connector.example.com/privacy',
  redirectURIs: ['https://connector.example.com/callback'],
  status: 'approved',
  createdAt: now,
  updatedAt: now,
};

const approvedScopes = [
  {
    id: 1,
    scope: 'profile.basic.read',
    displayName: 'Basic profile',
    sensitivity: 'low',
    fields: ['name', 'avatar'],
    reason: 'Show the signed-in user',
    status: 'approved',
    reviewerUserID: 99,
    reviewedAt: now,
    decisionNote: 'standard profile access',
    createdAt: now,
    updatedAt: now,
  },
  {
    id: 3,
    scope: 'resource.read',
    displayName: 'Resource read',
    sensitivity: 'medium',
    fields: ['resourceID'],
    reason: 'Read assigned campus resources',
    status: 'approved',
    reviewerUserID: 99,
    reviewedAt: now,
    decisionNote: 'resource access',
    createdAt: now,
    updatedAt: now,
  },
  {
    id: 4,
    scope: 'resource.write',
    displayName: 'Resource write',
    sensitivity: 'high',
    fields: ['resourceID'],
    reason: 'Write assigned campus resources',
    status: 'approved',
    reviewerUserID: 99,
    reviewedAt: now,
    decisionNote: 'resource write access',
    createdAt: now,
    updatedAt: now,
  },
];

function createPendingApp() {
  return {
    app: {
      ...baseApp,
      status: 'pending',
    },
    scopes: [
      approvedScopes[0],
      {
        id: 2,
        scope: 'email.read',
        displayName: 'Email address',
        sensitivity: 'medium',
        fields: ['email'],
        reason: 'Send account notifications',
        status: 'pending',
        reviewerUserID: null,
        reviewedAt: null,
        decisionNote: null,
        createdAt: now,
        updatedAt: now,
      },
    ],
    redirectURIRequests: [
      {
        id: 7,
        redirectURIs: ['https://connector.example.com/oauth/callback'],
        reason: 'Move callback to the OAuth path',
        status: 'pending',
        reviewerUserID: null,
        reviewedAt: null,
        decisionNote: null,
        createdAt: now,
        updatedAt: now,
      },
    ],
  };
}

function createApprovedApp() {
  return {
    app: baseApp,
    scopes: approvedScopes,
    redirectURIRequests: [],
  };
}

function createSuspendedApp() {
  return {
    app: {
      ...baseApp,
      status: 'suspended',
    },
    scopes: approvedScopes,
    redirectURIRequests: [],
  };
}

const existingResourceGrant = {
  resourceType: 'resource_item',
  resourceID: 'existing_resource',
  action: 'read',
  relation: 'can_read_by_app',
};

const consent = {
  userID: 12,
  app: baseApp,
  scopes: [
    {
      scope: 'profile.basic.read',
      displayName: 'Basic profile',
      grantedAt: now,
      lastUsedAt: now,
    },
  ],
};

interface CapturedMutation {
  body: unknown;
  method: string;
  path: string;
}

let nextAppListErrorMessage = '';
let nextConsentListErrorMessage = '';
let nextOpenPlatformActionErrorMessage = '';
let nextResourceGrantListErrorMessage = '';
let nextResourceGrantActionErrorMessage = '';

function json(data: unknown, status = 200) {
  return {
    status,
    contentType: 'application/json',
    body: JSON.stringify(data),
  };
}

function ok(data: unknown) {
  return json({ success: true, data });
}

function list<T>(items: T[], total = items.length) {
  return ok({ list: items, total });
}

function parseJsonBody(route: Route) {
  const postData = route.request().postData();
  return postData ? (JSON.parse(postData) as unknown) : null;
}

async function fulfillNextOpenPlatformActionError(route: Route) {
  if (!nextOpenPlatformActionErrorMessage) {
    return false;
  }
  const message = nextOpenPlatformActionErrorMessage;
  nextOpenPlatformActionErrorMessage = '';
  await route.fulfill(
    json({
      success: false,
      error: {
        code: 'E2E_OPEN_PLATFORM_ACTION_FAILED',
        message,
      },
    }),
  );
  return true;
}

async function fulfillNextResourceGrantActionError(route: Route) {
  if (!nextResourceGrantActionErrorMessage) {
    return false;
  }
  const message = nextResourceGrantActionErrorMessage;
  nextResourceGrantActionErrorMessage = '';
  await route.fulfill(
    json({
      success: false,
      error: {
        code: 'E2E_OPEN_PLATFORM_RESOURCE_GRANT_ACTION_FAILED',
        message,
      },
    }),
  );
  return true;
}

async function confirmPopconfirm(page: Page) {
  await page
    .getByRole('button', { name: /确定|Confirm/ })
    .last()
    .click();
}

async function waitForAppListRequest(
  page: Page,
  matches: (url: URL) => boolean,
  trigger: () => Promise<void>,
) {
  const responsePromise = page.waitForResponse((response) => {
    const request = response.request();
    const url = new URL(response.url());
    return (
      request.method() === 'GET' &&
      url.pathname === '/api/v1/admin/open-platform/apps' &&
      matches(url) &&
      response.status() < 400
    );
  });

  await trigger();
  await responsePromise;
}

async function submitMessageBoxPrompt(page: Page, value: string) {
  const messageBox = page.locator('.el-message-box').last();
  await messageBox.locator('input').fill(value);
  await messageBox.getByRole('button', { name: /确定|确认|Confirm/ }).click();
}

async function mockOpenPlatformApi(
  page: Page,
  apps: unknown[],
  capturedMutations: CapturedMutation[],
  appListTotal = apps.length,
) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    if (path === '/api/v1/auth/me') {
      await route.fulfill(ok(adminUser));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps') {
      if (nextAppListErrorMessage) {
        const message = nextAppListErrorMessage;
        nextAppListErrorMessage = '';
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_OPEN_PLATFORM_APPS_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(list(apps, appListTotal));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/import-casdoor') {
      if (await fulfillNextOpenPlatformActionError(route)) {
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(
        ok({
          app: {
            ...baseApp,
            id: 77,
            clientID: 'legacy-campus',
            displayName: 'Legacy Campus',
            homepageURL: 'https://legacy.example.com',
            privacyPolicyURL: 'https://legacy.example.com/privacy',
            redirectURIs: ['https://legacy.example.com/callback'],
            status: 'approved',
          },
          clientSecretSource: 'provided',
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/open-platform/consents') {
      if (nextConsentListErrorMessage) {
        const message = nextConsentListErrorMessage;
        nextConsentListErrorMessage = '';
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_OPEN_PLATFORM_CONSENTS_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(list([consent]));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/42/resource-grants') {
      if (method === 'GET') {
        if (nextResourceGrantListErrorMessage) {
          const message = nextResourceGrantListErrorMessage;
          nextResourceGrantListErrorMessage = '';
          await route.fulfill(
            json({
              success: false,
              error: {
                code: 'E2E_OPEN_PLATFORM_RESOURCE_GRANTS_UNAVAILABLE',
                message,
              },
            }),
          );
          return;
        }
        await route.fulfill(ok({ grants: [existingResourceGrant] }));
        return;
      }
      if (await fulfillNextResourceGrantActionError(route)) {
        return;
      }
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(
        ok({
          grants: [
            existingResourceGrant,
            {
              resourceType: 'resource_item',
              resourceID: 'course_42',
              action: 'read',
              relation: 'can_read_by_app',
            },
          ],
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/42/resource-grants/revoke') {
      if (await fulfillNextResourceGrantActionError(route)) {
        return;
      }
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(ok({ grants: [] }));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/42/approve') {
      if (await fulfillNextOpenPlatformActionError(route)) {
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(
        ok({
          app: { ...baseApp, status: 'approved' },
          clientSecret: 'secret-once',
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/42/secret/rotate') {
      if (await fulfillNextOpenPlatformActionError(route)) {
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(
        ok({ app: baseApp, clientSecret: 'rotated-secret-once' }),
      );
      return;
    }

    if (
      path === '/api/v1/admin/open-platform/apps/42/suspend' ||
      path === '/api/v1/admin/open-platform/apps/42/resume' ||
      path === '/api/v1/admin/open-platform/apps/42/revoke' ||
      path === '/api/v1/admin/open-platform/apps/42/consents/revoke' ||
      path.includes('/api/v1/admin/open-platform/apps/42/scopes/') ||
      path.includes(
        '/api/v1/admin/open-platform/apps/42/redirect-uri-requests/',
      )
    ) {
      if (await fulfillNextOpenPlatformActionError(route)) {
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ app: baseApp }));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked open platform action request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Open Platform admin actions', () => {
  let capturedMutations: CapturedMutation[];

  test.beforeEach(() => {
    capturedMutations = [];
    nextAppListErrorMessage = '';
    nextConsentListErrorMessage = '';
    nextOpenPlatformActionErrorMessage = '';
    nextResourceGrantListErrorMessage = '';
    nextResourceGrantActionErrorMessage = '';
  });

  test('app list failures show a persistent retry path', async ({ page }) => {
    nextAppListErrorMessage = '开放平台应用列表暂时不可用';
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    const loadError = page.locator('.admin-load-error', {
      hasText: '开放平台应用列表暂时不可用',
    });
    await expect(loadError).toBeVisible();

    await loadError.getByRole('button', { name: /重试|Retry/ }).click();

    await expect(page.getByText('Campus Connector')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('app list query button requests the first page after pagination changes', async ({
    page,
  }) => {
    await mockOpenPlatformApi(
      page,
      [createPendingApp()],
      capturedMutations,
      40,
    );

    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await waitForAppListRequest(
      page,
      (url) =>
        url.searchParams.get('page') === '2' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page
          .locator('.el-pagination .el-pager li')
          .filter({ hasText: /^2$/ })
          .click();
      },
    );

    await waitForAppListRequest(
      page,
      (url) =>
        url.searchParams.get('status') === 'pending' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('button', { name: /查询|Query/ }).click();
      },
    );
  });

  test('consent list failures show a persistent retry path', async ({
    page,
  }) => {
    nextConsentListErrorMessage = '开放平台授权列表暂时不可用';
    await mockOpenPlatformApi(page, [], capturedMutations);

    await page.goto('/open-platform/consents');
    await page.locator('input[placeholder="按应用 ID"]').fill('42');
    await page.getByRole('button', { name: /查询|Query/ }).click();
    const loadError = page.locator('.admin-load-error', {
      hasText: '开放平台授权列表暂时不可用',
    });
    await expect(loadError).toBeVisible();

    await loadError.getByRole('button', { name: /重试|Retry/ }).click();

    await expect(page.getByText('Campus Connector')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('resource grant list failures show a persistent retry path', async ({
    page,
  }) => {
    nextResourceGrantListErrorMessage = '资源授权列表暂时不可用';
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await page
      .getByRole('button', { name: /资源授权|Resource Grants/ })
      .click();

    const dialog = page.getByRole('dialog', {
      name: /Campus Connector 的资源授权/,
    });
    const loadError = dialog.locator('.open-platform-resource-grants__hint', {
      hasText: '资源授权列表暂时不可用',
    });
    await expect(loadError).toBeVisible();

    await loadError.getByRole('button', { name: /重试|Retry/ }).click();

    await expect(dialog.getByText('existing_resource')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('app review buttons submit scope, redirect, and app decisions', async ({
    page,
  }) => {
    await mockOpenPlatformApi(page, [createPendingApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /驳回权限|Reject Scope/ }).click();
    await submitMessageBoxPrompt(page, '用途说明不足');

    await page
      .getByRole('button', { name: /批准回调|Approve Redirect/ })
      .click();
    await confirmPopconfirm(page);

    await page
      .getByRole('button', { name: /驳回回调|Reject Redirect/ })
      .click();
    await submitMessageBoxPrompt(page, '回调地址不可用');

    await page.getByRole('button', { name: /批准应用|Approve App/ }).click();
    await confirmPopconfirm(page);

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/scopes/email.read/reject',
        method: 'POST',
        body: { decisionNote: '用途说明不足' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/redirect-uri-requests/7/approve',
        method: 'POST',
        body: null,
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/redirect-uri-requests/7/reject',
        method: 'POST',
        body: { decisionNote: '回调地址不可用' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/approve',
        method: 'POST',
        body: null,
      });
  });

  test('app review failures preserve backend error detail', async ({
    page,
  }) => {
    nextOpenPlatformActionErrorMessage = '开放平台应用审批写入失败';
    await mockOpenPlatformApi(page, [createPendingApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /批准应用|Approve App/ }).click();
    await confirmPopconfirm(page);

    await expect(
      page.locator('.admin-load-error', {
        hasText: '开放平台应用审批写入失败',
      }),
    ).toBeVisible();
    await expect(page.getByText('Campus Connector')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('legacy Casdoor import dialog submits normalized app metadata and scopes', async ({
    page,
  }) => {
    await mockOpenPlatformApi(page, [], capturedMutations);

    await page.goto('/open-platform/apps');
    await page.getByRole('button', { name: /导入 Casdoor 应用/ }).click();

    const dialog = page.getByRole('dialog', {
      name: /导入 legacy Casdoor 应用/,
    });
    await dialog
      .getByPlaceholder('请输入 Casdoor application name')
      .fill('legacy-campus');
    await dialog
      .getByPlaceholder('可选；为空时使用 Casdoor 展示名或应用名')
      .fill('Legacy Campus');
    await dialog
      .getByPlaceholder('可选；为空时使用 Casdoor 应用描述')
      .fill('Imported app');
    await dialog
      .getByPlaceholder('可选；为空时使用 Casdoor 主页 URL')
      .fill('https://legacy.example.com');
    await dialog
      .getByPlaceholder('必填；用于 id 授权页展示')
      .fill('https://legacy.example.com/privacy');
    await dialog
      .getByPlaceholder('可选；每行一个 URI。为空时使用 Casdoor redirect URI')
      .fill(
        'https://legacy.example.com/callback\nhttps://legacy.example.com/callback\nhttps://legacy.example.com/alt',
      );
    await dialog
      .getByPlaceholder('可选；用于保留原 legacy secret，提交后不回显')
      .fill('legacy-secret');
    await dialog
      .getByPlaceholder('用途说明，写入 scope 审核记录')
      .first()
      .fill('同步基础资料');
    await dialog.getByRole('button', { name: /添加权限/ }).click();
    await dialog
      .getByPlaceholder('用途说明，写入 scope 审核记录')
      .nth(1)
      .fill('同步邮箱');
    await dialog.getByRole('button', { name: /导入 Casdoor 应用/ }).click();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/import-casdoor',
        method: 'POST',
        body: {
          casdoorApplicationName: 'legacy-campus',
          clientSecret: 'legacy-secret',
          description: 'Imported app',
          displayName: 'Legacy Campus',
          homepageURL: 'https://legacy.example.com',
          privacyPolicyURL: 'https://legacy.example.com/privacy',
          redirectURIs: [
            'https://legacy.example.com/callback',
            'https://legacy.example.com/alt',
          ],
          scopes: [
            {
              reason: '同步基础资料',
              scope: 'profile.basic.read',
            },
            {
              reason: '同步邮箱',
              scope: 'email.read',
            },
          ],
        },
      });
  });

  test('legacy Casdoor import failures preserve backend error detail and draft input', async ({
    page,
  }) => {
    nextOpenPlatformActionErrorMessage = 'Casdoor 应用导入失败';
    await mockOpenPlatformApi(page, [], capturedMutations);

    await page.goto('/open-platform/apps');
    await page.getByRole('button', { name: /导入 Casdoor 应用/ }).click();

    const dialog = page.getByRole('dialog', {
      name: /导入 legacy Casdoor 应用/,
    });
    await dialog
      .getByPlaceholder('请输入 Casdoor application name')
      .fill('legacy-campus');
    await dialog
      .getByPlaceholder('必填；用于 id 授权页展示')
      .fill('https://legacy.example.com/privacy');
    await dialog
      .getByPlaceholder('用途说明，写入 scope 审核记录')
      .first()
      .fill('同步基础资料');

    await dialog.getByRole('button', { name: /导入 Casdoor 应用/ }).click();

    await expect(
      page.locator('.admin-load-error', {
        hasText: 'Casdoor 应用导入失败',
      }),
    ).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(
      dialog.getByPlaceholder('请输入 Casdoor application name'),
    ).toHaveValue('legacy-campus');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('approved app lifecycle buttons submit audited reasons', async ({
    page,
  }) => {
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /轮换密钥|Rotate Secret/ }).click();
    await submitMessageBoxPrompt(page, '定期轮换');

    await page.getByRole('button', { name: /暂停应用|Suspend App/ }).click();
    await submitMessageBoxPrompt(page, '异常流量复核');

    await page.getByRole('button', { name: /吊销应用|Revoke App/ }).click();
    await submitMessageBoxPrompt(page, '应用下线');

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/secret/rotate',
        method: 'POST',
        body: { reason: '定期轮换' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/suspend',
        method: 'POST',
        body: { reason: '异常流量复核' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/revoke',
        method: 'POST',
        body: { reason: '应用下线' },
      });
  });

  test('app lifecycle failures preserve backend error detail', async ({
    page,
  }) => {
    nextOpenPlatformActionErrorMessage = '开放平台密钥轮换失败';
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /轮换密钥|Rotate Secret/ }).click();
    await submitMessageBoxPrompt(page, '定期轮换');

    await expect(
      page.locator('.admin-load-error', {
        hasText: '开放平台密钥轮换失败',
      }),
    ).toBeVisible();
    await expect(page.getByText('Campus Connector')).toBeVisible();
    await expect(page.getByText('rotated-secret-once')).toHaveCount(0);
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('suspended app resume button submits audited reason', async ({
    page,
  }) => {
    await mockOpenPlatformApi(page, [createSuspendedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /恢复应用|Resume App/ }).click();
    await submitMessageBoxPrompt(page, '风险解除');

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/resume',
        method: 'POST',
        body: { reason: '风险解除' },
      });
  });

  test('resource grants dialog grants and revokes resource access', async ({
    page,
  }) => {
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await page
      .getByRole('button', { name: /资源授权|Resource Grants/ })
      .click();

    const dialog = page.getByRole('dialog', {
      name: /Campus Connector 的资源授权/,
    });
    await expect(dialog.getByText('existing_resource')).toBeVisible();

    await dialog.getByPlaceholder(/请输入资源 ID/).fill('course_42');
    await dialog.getByPlaceholder(/请输入原因/).fill('授予课程读取');
    await dialog
      .getByRole('button', { name: /授予资源权限|Grant Resource/ })
      .click();

    await dialog
      .getByRole('button', { name: /撤销|Revoke/ })
      .first()
      .click();
    await submitMessageBoxPrompt(page, '撤销测试授权');

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/resource-grants',
        method: 'POST',
        body: {
          actions: ['read'],
          reason: '授予课程读取',
          resourceID: 'course_42',
          resourceType: 'resource_item',
        },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/resource-grants/revoke',
        method: 'POST',
        body: {
          actions: ['read'],
          reason: '撤销测试授权',
          resourceID: 'existing_resource',
          resourceType: 'resource_item',
        },
      });
  });

  test('resource grant action failures remain visible in the dialog', async ({
    page,
  }) => {
    nextResourceGrantActionErrorMessage = 'OpenFGA 资源授权写入失败';
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await page
      .getByRole('button', { name: /资源授权|Resource Grants/ })
      .click();

    const dialog = page.getByRole('dialog', {
      name: /Campus Connector 的资源授权/,
    });
    await expect(dialog.getByText('existing_resource')).toBeVisible();

    await dialog.getByPlaceholder(/请输入资源 ID/).fill('course_42');
    await dialog.getByPlaceholder(/请输入原因/).fill('授予课程读取');
    await dialog
      .getByRole('button', { name: /授予资源权限|Grant Resource/ })
      .click();

    await expect(
      dialog.locator('.open-platform-resource-grants__hint', {
        hasText: 'OpenFGA 资源授权写入失败',
      }),
    ).toBeVisible();
    await expect(dialog.getByText('existing_resource')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('resource grant revoke failures keep the existing grant visible', async ({
    page,
  }) => {
    nextResourceGrantActionErrorMessage = 'OpenFGA 资源授权撤销失败';
    await mockOpenPlatformApi(page, [createApprovedApp()], capturedMutations);

    await page.goto('/open-platform/apps');
    await page
      .getByRole('button', { name: /资源授权|Resource Grants/ })
      .click();

    const dialog = page.getByRole('dialog', {
      name: /Campus Connector 的资源授权/,
    });
    await expect(dialog.getByText('existing_resource')).toBeVisible();

    await dialog
      .getByRole('button', { name: /撤销|Revoke/ })
      .first()
      .click();
    await submitMessageBoxPrompt(page, '撤销测试授权');

    await expect(
      dialog.locator('.open-platform-resource-grants__hint', {
        hasText: 'OpenFGA 资源授权撤销失败',
      }),
    ).toBeVisible();
    await expect(dialog.getByText('existing_resource')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('consent management revokes one scope and all app grants', async ({
    page,
  }) => {
    await mockOpenPlatformApi(page, [], capturedMutations);

    await page.goto('/open-platform/consents');
    await page.locator('input[placeholder="按应用 ID"]').fill('42');
    await page.getByRole('button', { name: /查询|Query/ }).click();
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /撤销 Scope|Revoke Scope/ }).click();
    await submitMessageBoxPrompt(page, '撤销单个 scope');

    await page.getByRole('button', { name: /撤销全部|Revoke All/ }).click();
    await submitMessageBoxPrompt(page, '撤销全部授权');

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/consents/revoke',
        method: 'POST',
        body: {
          reason: '撤销单个 scope',
          scopes: ['profile.basic.read'],
          userID: 12,
        },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/open-platform/apps/42/consents/revoke',
        method: 'POST',
        body: {
          reason: '撤销全部授权',
          userID: 12,
        },
      });
  });

  test('consent revoke failures preserve backend error detail', async ({
    page,
  }) => {
    nextOpenPlatformActionErrorMessage = '开放平台授权撤销失败';
    await mockOpenPlatformApi(page, [], capturedMutations);

    await page.goto('/open-platform/consents');
    await page.locator('input[placeholder="按应用 ID"]').fill('42');
    await page.getByRole('button', { name: /查询|Query/ }).click();
    await expect(page.getByText('Campus Connector')).toBeVisible();

    await page.getByRole('button', { name: /撤销 Scope|Revoke Scope/ }).click();
    await submitMessageBoxPrompt(page, '撤销单个 scope');

    await expect(
      page.locator('.admin-load-error', {
        hasText: '开放平台授权撤销失败',
      }),
    ).toBeVisible();
    await expect(page.getByText('Campus Connector')).toBeVisible();
    await expect(
      page.locator('.open-platform-consent-scopes .el-tag__content', {
        hasText: 'profile.basic.read',
      }),
    ).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });
});
