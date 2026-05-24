import type { Page } from './fixtures';

import { expect, test } from './fixtures';

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

const now = '2026-05-24T04:00:00Z';

function createPendingApp() {
  return {
    app: {
      id: 42,
      clientID: 'campus-connector',
      displayName: 'Campus Connector',
      description: 'Campus integration client',
      homepageURL: 'https://connector.example.com',
      privacyPolicyURL: 'https://connector.example.com/privacy',
      redirectURIs: ['https://connector.example.com/callback'],
      status: 'pending',
      createdAt: now,
      updatedAt: now,
    },
    scopes: [
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

async function mockAdminSession(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: adminUser,
      }),
    });
  });
}

test('open platform apps page renders pending reviews and approves a scope', async ({
  page,
}) => {
  await mockAdminSession(page);

  const appList = [createPendingApp()];
  let listRequestURL: null | URL = null;
  let listRequestCount = 0;
  let approvedScopePath = '';

  await page.route('**/api/v1/admin/open-platform/apps?*', async (route) => {
    listRequestCount += 1;
    listRequestURL = new URL(route.request().url());
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          list: appList,
          total: appList.length,
        },
      }),
    });
  });

  await page.route(
    '**/api/v1/admin/open-platform/apps/42/scopes/email.read/approve',
    async (route) => {
      approvedScopePath = new URL(route.request().url()).pathname;
      appList[0].scopes = appList[0].scopes.map((scope) =>
        scope.scope === 'email.read'
          ? {
              ...scope,
              status: 'approved',
              reviewerUserID: 99,
              reviewedAt: now,
              decisionNote: '',
              updatedAt: now,
            }
          : scope,
      );
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { scopes: appList[0].scopes },
        }),
      });
    },
  );

  await page.goto('/open-platform/apps');

  await expect(page.getByText('Campus Connector')).toBeVisible();
  await expect(page.getByText('campus-connector')).toBeVisible();
  await expect(page.getByText('Email address')).toBeVisible();
  expect(listRequestURL?.searchParams.get('status')).toBe('pending');

  await page.getByRole('button', { name: /批准权限|Approve Scope/ }).click();
  await page
    .getByRole('button', { name: /确定|Confirm/ })
    .last()
    .click();

  await expect
    .poll(() => approvedScopePath)
    .toContain('/api/v1/admin/open-platform/apps/42/scopes/email.read/approve');
  await expect.poll(() => listRequestCount).toBeGreaterThan(1);
  await expect(
    page.getByRole('button', { name: /批准应用|Approve App/ }),
  ).toBeVisible();
});
