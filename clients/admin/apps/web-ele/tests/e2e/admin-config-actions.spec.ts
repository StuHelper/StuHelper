import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const capabilities = [
  'user:system:read',
  'user:system:update',
  'admission:policy:read',
  'admission:policy:update',
];

const adminUser = {
  id: '99',
  name: 'platform-admin',
  displayName: 'Platform Admin',
  email: 'admin@example.com',
  avatar: '',
  roles: ['super_admin'],
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

const systemConfig = {
  key: 'review.retention_days',
  value: '90',
  description: '评课数据保留天数',
  updatedAt: '2026-08-05T00:00:00Z',
};

const admissionPolicy = {
  id: 'policy-qq-1',
  platform: 'qq',
  guildID: 'guild-1',
  guardEnabled: true,
  joinHandlingStrategy: 'post_join_guard',
  autoApproveJoin: true,
  autoApproveVerifiedJoin: true,
  autoApproveUnverifiedJoin: true,
  unverifiedJoinRejectReason: '请先完成 StuHelper 学生认证后再申请入群。',
  initialMuteDurationSeconds: 60,
  linkWaitSeconds: 300,
  submissionWaitSeconds: 600,
  manualReviewTimeoutSeconds: 3600,
  reminderIntervalSeconds: 120,
  failedJoinLimit: 3,
  blacklistDurationSeconds: 86_400,
  allowTemporaryFreshman: true,
};

interface CapturedMutation {
  body: unknown;
  method: string;
  path: string;
}

let nextSystemConfigListErrorMessage: null | string = null;
let nextSystemConfigUpdateErrorMessage: null | string = null;
let nextAdmissionPolicyListErrorMessage: null | string = null;
let nextAdmissionPolicyUpdateErrorMessage: null | string = null;

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

function parseJsonBody(route: Route) {
  const postData = route.request().postData();
  return postData ? (JSON.parse(postData) as unknown) : null;
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

    if (path === '/api/v1/admin/system-configs') {
      if (nextSystemConfigListErrorMessage) {
        const message = nextSystemConfigListErrorMessage;
        nextSystemConfigListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: { code: 'E2E_SYSTEM_CONFIG_UNAVAILABLE', message },
          }),
        );
        return;
      }
      await route.fulfill(ok([systemConfig]));
      return;
    }

    if (path === '/api/v1/admin/system-configs/review.retention_days') {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      if (nextSystemConfigUpdateErrorMessage) {
        const message = nextSystemConfigUpdateErrorMessage;
        nextSystemConfigUpdateErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: { code: 'E2E_SYSTEM_CONFIG_REJECTED', message },
          }),
        );
        return;
      }
      await route.fulfill(ok({ ...systemConfig, value: '180' }));
      return;
    }

    if (path === '/api/v1/admin/admission/policies' && method === 'POST') {
      const body = parseJsonBody(route) as null | { guildID?: string };
      capturedMutations.push({ path, method, body });
      await route.fulfill(
        ok({
          ...admissionPolicy,
          id: `policy-${body?.guildID ?? 'new'}`,
          guildID: body?.guildID ?? 'guild-new',
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/admission/policies') {
      if (nextAdmissionPolicyListErrorMessage) {
        const message = nextAdmissionPolicyListErrorMessage;
        nextAdmissionPolicyListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: { code: 'E2E_ADMISSION_POLICY_UNAVAILABLE', message },
          }),
        );
        return;
      }
      await route.fulfill(ok([admissionPolicy]));
      return;
    }

    if (path === '/api/v1/admin/admission/policies/policy-qq-1') {
      const body = parseJsonBody(route);
      capturedMutations.push({ path, method, body });
      if (nextAdmissionPolicyUpdateErrorMessage) {
        const message = nextAdmissionPolicyUpdateErrorMessage;
        nextAdmissionPolicyUpdateErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: { code: 'E2E_ADMISSION_POLICY_REJECTED', message },
          }),
        );
        return;
      }
      await route.fulfill(ok(admissionPolicy));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked config request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin current configuration actions', () => {
  let capturedMutations: CapturedMutation[];

  test.beforeEach(async ({ page }) => {
    capturedMutations = [];
    nextSystemConfigListErrorMessage = null;
    nextSystemConfigUpdateErrorMessage = null;
    nextAdmissionPolicyListErrorMessage = null;
    nextAdmissionPolicyUpdateErrorMessage = null;
    await mockAdminApi(page, capturedMutations);
  });

  test('system configuration failures keep a retry and preserve the draft', async ({
    page,
  }) => {
    nextSystemConfigListErrorMessage = '系统配置列表暂不可用';
    await page.goto('/users/system-config');

    const loadError = page.locator('.admin-load-error', {
      hasText: '系统配置列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('review.retention_days')).toBeVisible();

    nextSystemConfigUpdateErrorMessage = '系统配置保存被后台拒绝';
    const row = page.getByRole('row', { name: /review\.retention_days/ });
    await row.getByRole('button', { name: /编辑|Edit/ }).click();
    const dialog = page.getByRole('dialog', { name: /编辑配置|Edit/ });
    await dialog.getByPlaceholder('请输入配置值').fill('180');
    await dialog.getByRole('button', { name: /保存|Save/ }).click();

    await expect(
      page.locator('.admin-load-error', {
        hasText: '系统配置保存被后台拒绝',
      }),
    ).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(dialog.getByPlaceholder('请输入配置值')).toHaveValue('180');
  });

  test('admission policy saves only target qualification and timing controls', async ({
    page,
  }) => {
    await page.goto('/users/admission-policy');
    const form = page.locator('form').filter({ hasText: 'QQ 群 guild-1' });
    await expect(form).toBeVisible();

    await form
      .getByRole('spinbutton', { name: '学生认证链接等待（秒）' })
      .fill('450');
    await form.getByRole('spinbutton', { name: '失败入群上限' }).fill('5');
    await form
      .locator('.el-form-item', { hasText: '允许临时新生凭据入群' })
      .locator('.el-switch')
      .click();
    await form.getByText('申请时学生认证', { exact: true }).click();
    await form
      .getByRole('textbox', { name: '未认证拒绝理由' })
      .fill('请先完成 StuHelper 学生认证后再申请入群');
    await form.getByRole('button', { name: '保存' }).click();

    await expect
      .poll(() => capturedMutations.at(-1))
      .toMatchObject({
        path: '/api/v1/admin/admission/policies/policy-qq-1',
        method: 'PUT',
        body: expect.objectContaining({
          guardEnabled: true,
          joinHandlingStrategy: 'join_request_review',
          autoApproveJoin: false,
          autoApproveVerifiedJoin: true,
          autoApproveUnverifiedJoin: false,
          unverifiedJoinRejectReason: '请先完成 StuHelper 学生认证后再申请入群',
          linkWaitSeconds: 450,
          failedJoinLimit: 5,
          allowTemporaryFreshman: false,
        }),
      });
  });

  test('admission policy creates new target groups from an existing policy', async ({
    page,
  }) => {
    await page.goto('/users/admission-policy');
    await page.getByRole('button', { name: '新增目标认证群' }).click();
    const dialog = page.getByRole('dialog', { name: '新增目标认证群' });
    await dialog
      .getByPlaceholder('每行一个需要开启入群认证的 QQ 群号')
      .fill('guild-2\nguild-3');
    await dialog.getByRole('button', { name: '创建' }).click();

    await expect
      .poll(() =>
        capturedMutations.filter(
          (item) =>
            item.path === '/api/v1/admin/admission/policies' &&
            item.method === 'POST',
        ),
      )
      .toMatchObject([
        {
          body: {
            sourcePolicyID: 'policy-qq-1',
            platform: 'qq',
            guildID: 'guild-2',
          },
        },
        {
          body: {
            sourcePolicyID: 'policy-qq-1',
            platform: 'qq',
            guildID: 'guild-3',
          },
        },
      ]);
  });

  test('admission policy list failures expose an explicit retry', async ({
    page,
  }) => {
    nextAdmissionPolicyListErrorMessage = '入群认证策略暂不可用';
    await page.goto('/users/admission-policy');
    const error = page.locator('.admin-load-error', {
      hasText: '入群认证策略暂不可用',
    });
    await expect(error).toBeVisible();
    await error.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('QQ 群 guild-1')).toBeVisible();
  });
});
