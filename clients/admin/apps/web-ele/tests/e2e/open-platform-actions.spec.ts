import type { Page, Route } from '@playwright/test';

import { expect, test } from '@playwright/test';

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

function list<T>(items: T[]) {
  return ok({ list: items, total: items.length });
}

function parseJsonBody(route: Route) {
  const postData = route.request().postData();
  return postData ? (JSON.parse(postData) as unknown) : null;
}

async function confirmPopconfirm(page: Page) {
  await page
    .getByRole('button', { name: /确定|Confirm/ })
    .last()
    .click();
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
      await route.fulfill(list(apps));
      return;
    }

    if (path === '/api/v1/admin/open-platform/consents') {
      await route.fulfill(list([consent]));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/42/resource-grants') {
      if (method === 'GET') {
        await route.fulfill(ok({ grants: [existingResourceGrant] }));
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
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(ok({ grants: [] }));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps/42/approve') {
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
});
