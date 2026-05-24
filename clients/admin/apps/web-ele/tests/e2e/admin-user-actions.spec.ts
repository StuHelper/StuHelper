import type { Page, Route } from '@playwright/test';

import { expect, test } from '@playwright/test';

const now = '2026-05-24T04:00:00Z';

const capabilities = [
  'user:student:review',
  'user:system:read',
  'user:system:update',
  'admission:freshman:review',
  'member_blacklist:read',
  'member_blacklist:manage',
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

const studentVerifications = [
  {
    userID: 21,
    schoolID: 1001,
    activeStudentID: '20260021',
    verificationStatus: 'pending',
    verificationMethod: 'manual',
    createdAt: now,
    updatedAt: now,
  },
  {
    userID: 22,
    schoolID: 1001,
    activeStudentID: '20260022',
    verificationStatus: 'pending',
    verificationMethod: 'manual',
    createdAt: now,
    updatedAt: now,
  },
];

const freshmanApplication = {
  id: 'freshman-action-1',
  status: 'pending',
  schoolID: 1001,
  qqID: '20001',
  applicantNameMasked: '王*',
  materialURL: 'https://example.com/material.jpg',
  failureCount: 2,
  createdAt: now,
};

const blacklistEntry = {
  id: 'entry-active',
  platform: 'qq',
  subjectType: 'qq_user',
  subjectID: '30001',
  scopeType: 'guild',
  guildID: 'guild-1',
  source: 'admission_failure',
  reasonCode: 'admission_timeout_limit',
  reasonText: 'too many failures',
  createdFrom: 'admin_console',
  createdByType: 'admin_user',
  createdByID: 'admin-1',
  createdAt: now,
  updatedAt: now,
  expiresAt: null,
  releasedAt: null,
  releasedByType: null,
  releasedByID: null,
  releaseReasonCode: null,
  releaseReason: null,
  metadata: {},
};

const systemConfig = {
  key: 'review.retention_days',
  value: '365',
  description: '评课保留天数',
  updatedAt: now,
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

async function mockAdminApi(page: Page, capturedMutations: CapturedMutation[]) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    if (path === '/api/v1/auth/me') {
      await route.fulfill(ok(adminUser));
      return;
    }

    if (path === '/api/v1/admin/student-verifications') {
      await route.fulfill(list(studentVerifications));
      return;
    }
    if (path.startsWith('/api/v1/admin/student-verifications/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(
        ok({ ...studentVerifications[0], verificationStatus: 'verified' }),
      );
      return;
    }

    if (path === '/api/v1/admin/freshman-verifications') {
      await route.fulfill(list([freshmanApplication]));
      return;
    }
    if (path.startsWith('/api/v1/admin/freshman-verifications/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(ok({ ...freshmanApplication, status: 'approved' }));
      return;
    }

    if (path === '/api/v1/admin/member-blacklist') {
      capturedMutations.push({
        path,
        method,
        body: method === 'POST' ? parseJsonBody(route) : null,
      });
      await route.fulfill(
        method === 'POST'
          ? ok({ ...blacklistEntry, id: 'entry-created' })
          : list([blacklistEntry]),
      );
      return;
    }
    if (path.startsWith('/api/v1/admin/member-blacklist/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(
        ok({
          ...blacklistEntry,
          releasedAt: now,
          releaseReasonCode: 'manual_pardon',
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/system-configs') {
      await route.fulfill(ok([systemConfig]));
      return;
    }
    if (path.startsWith('/api/v1/admin/system-configs/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(ok({ ...systemConfig, value: '180' }));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked admin user action request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin user-system actions', () => {
  let capturedMutations: CapturedMutation[];

  test.beforeEach(async ({ page }) => {
    capturedMutations = [];
    await mockAdminApi(page, capturedMutations);
  });

  test('student verification buttons submit approve and reject decisions', async ({
    page,
  }) => {
    await page.goto('/users/student-verification');

    const approveRow = page.getByRole('row', { name: /20260021/ });
    await expect(approveRow).toBeVisible();
    await approveRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);

    const rejectRow = page.getByRole('row', { name: /20260022/ });
    await rejectRow.getByRole('button', { name: /驳回|Reject/ }).click();
    await confirmPopconfirm(page);

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/student-verifications/21',
        method: 'PUT',
        body: { approved: true },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/student-verifications/22',
        method: 'PUT',
        body: { approved: false },
      });
  });

  test('freshman verification actions submit expiry and rejection payloads', async ({
    page,
  }) => {
    await page.goto('/users/freshman-verification');

    const freshmanRow = page.getByRole('row', { name: /王\*/ });
    await expect(freshmanRow).toBeVisible();

    for (let index = 0; index < 7; index += 1) {
      await page.getByRole('button', { name: '增加数值' }).click();
    }
    await page.locator('[data-action="approveWithDays"]:visible').click();

    await page.getByRole('textbox', { name: '驳回原因' }).fill('材料不清晰');
    await page.locator('[data-action="reject"]:visible').click();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/freshman-verifications/freshman-action-1',
        method: 'PUT',
        body: { action: 'approve', expiresInDays: 7 },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/freshman-verifications/freshman-action-1',
        method: 'PUT',
        body: { action: 'reject', reason: '材料不清晰' },
      });
  });

  test('member blacklist form creates and releases entries', async ({
    page,
  }) => {
    await page.goto('/users/member-blacklist');

    await expect(page.getByText('too many failures')).toBeVisible();
    await page.getByRole('button', { name: '新增黑名单' }).click();

    const createDialog = page.getByRole('dialog', { name: '新增成员黑名单' });
    await createDialog
      .getByRole('textbox', { name: '主体 ID（QQ 号）' })
      .fill('30002');
    await createDialog.getByRole('textbox', { name: '群号' }).fill('guild-2');
    await createDialog
      .getByRole('textbox', { name: '原因（必填）' })
      .fill('人工风控加入');
    await createDialog.getByRole('button', { name: '创建' }).click();

    await page
      .getByRole('row', { name: /30001/ })
      .getByRole('button', { name: '解除' })
      .click();

    const releaseDialog = page.getByRole('dialog', { name: '解除成员黑名单' });
    await releaseDialog
      .getByRole('textbox', { name: '备注（可选）' })
      .fill('人工复核通过');
    await releaseDialog.getByRole('button', { name: '解除' }).click();

    await expect
      .poll(
        () =>
          capturedMutations.find(
            (mutation) =>
              mutation.path === '/api/v1/admin/member-blacklist' &&
              mutation.method === 'POST',
          )?.body,
      )
      .toEqual(
        expect.objectContaining({
          platform: 'qq',
          subjectType: 'qq_user',
          subjectID: '30002',
          scopeType: 'guild',
          guildID: 'guild-2',
          source: 'manual_admin',
          reasonCode: 'manual_blacklist',
          reasonText: '人工风控加入',
        }),
      );
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/member-blacklist/entry-active/release',
        method: 'POST',
        body: {
          releaseReasonCode: 'manual_pardon',
          releaseReason: '人工复核通过',
        },
      });
  });

  test('system config edit dialog saves the updated value', async ({
    page,
  }) => {
    await page.goto('/users/system-config');

    const configRow = page.getByRole('row', { name: /review\.retention_days/ });
    await expect(configRow).toBeVisible();
    await configRow.getByRole('button', { name: /编辑|Edit/ }).click();

    const dialog = page.getByRole('dialog', { name: /编辑配置|Edit/ });
    await dialog.getByPlaceholder('请输入配置值').fill('180');
    await dialog.getByRole('button', { name: /保存|Save/ }).click();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/system-configs/review.retention_days',
        method: 'PUT',
        body: { value: '180' },
      });
  });
});
