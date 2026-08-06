import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-08-05T08:00:00Z';

const capabilities = [
  'admission:session:read',
  'admission:session:manage',
  'user:system:read',
  'user:system:update',
  'member_blacklist:read',
  'member_blacklist:manage',
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

const admissionSession = {
  id: 'admission-session-action-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: '178037297',
  channelID: '178037297',
  qqID: '1390191645',
  userID: '42',
  status: 'awaiting_requirements',
  tokenExpiresAt: '2026-08-05T09:00:00Z',
  tokenConsumedAt: '2026-08-05T08:10:00Z',
  linkWaitDeadlineAt: '2026-08-05T09:00:00Z',
  submissionWaitDeadlineAt: '2026-08-05T10:00:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-08-05T08:30:00Z',
  verifiedAt: null,
  cancelledAt: null,
  lastBotError: 'previous reminder failed',
  eligibilityRevision: null,
  eligibilityEvaluatedAt: null,
  requirementsStatus: 'awaiting_requirements',
  projectionPending: false,
  authURL: 'https://join.stuhelper.com/verify/action-token',
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

interface CapturedRequest {
  body: unknown;
  method: string;
  path: string;
  query: Record<string, string>;
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

async function confirmPopconfirm(page: Page, title: string) {
  const popconfirm = page
    .locator('.el-popper')
    .filter({ hasText: title })
    .last();
  await expect(popconfirm).toBeVisible();
  await popconfirm.getByRole('button', { name: /确定|Confirm/ }).click();
}

async function mockAdminApi(page: Page, captured: CapturedRequest[]) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    if (path === '/api/v1/auth/me') {
      await route.fulfill(ok(adminUser));
      return;
    }

    if (path === '/api/v1/admin/admission/sessions' && method === 'GET') {
      captured.push({
        path,
        method,
        query: Object.fromEntries(url.searchParams.entries()),
        body: null,
      });
      await route.fulfill(list([admissionSession]));
      return;
    }

    if (path.startsWith('/api/v1/admin/admission/sessions/')) {
      captured.push({
        path,
        method,
        query: {},
        body: parseJsonBody(route),
      });
      if (path.endsWith('/regenerate')) {
        await route.fulfill(
          ok({
            authURL: 'https://join.stuhelper.com/verify/regenerated-token',
            session: admissionSession,
            token: 'redacted-regenerated-token',
          }),
        );
        return;
      }
      await route.fulfill(
        ok({
          ...admissionSession,
          status: path.endsWith('/cancel')
            ? 'cancelled'
            : admissionSession.status,
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/member-blacklist') {
      captured.push({
        path,
        method,
        query: Object.fromEntries(url.searchParams.entries()),
        body: method === 'POST' ? parseJsonBody(route) : null,
      });
      await route.fulfill(
        method === 'POST'
          ? ok({ ...blacklistEntry, id: 'entry-created' })
          : list([blacklistEntry]),
      );
      return;
    }

    if (path === '/api/v1/admin/member-blacklist/entry-active/release') {
      captured.push({ path, method, query: {}, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...blacklistEntry, releasedAt: now }));
      return;
    }

    if (path === '/api/v1/admin/system-configs') {
      await route.fulfill(ok([systemConfig]));
      return;
    }

    if (path === '/api/v1/admin/system-configs/review.retention_days') {
      captured.push({ path, method, query: {}, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...systemConfig, value: '180' }));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked admin user request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin current admission and account controls', () => {
  let captured: CapturedRequest[];

  test.beforeEach(async ({ page }) => {
    captured = [];
    await mockAdminApi(page, captured);
  });

  test('admission session uses the target state machine and queues durable actions', async ({
    page,
  }) => {
    await page.goto('/users/admission-sessions');

    const row = page.getByRole('row', { name: /1390191645/ });
    await expect(row).toBeVisible();
    await expect(row).toContainText('等待满足要求');
    await expect(row).toContainText('当前学生资格尚未满足群策略');

    await page.getByPlaceholder('QQ 号').fill(' 1390191645 ');
    await page.getByPlaceholder('群号').fill(' 178037297 ');
    await page.getByRole('button', { name: '查询' }).click();

    await expect
      .poll(() => captured.findLast((item) => item.method === 'GET'))
      .toEqual(
        expect.objectContaining({
          path: '/api/v1/admin/admission/sessions',
          query: expect.objectContaining({
            qqID: '1390191645',
            guildID: '178037297',
            platform: 'qq',
          }),
        }),
      );

    await row.locator('[data-action="requestResend"]').click();
    await expect(page.getByText('已加入机器人重发队列')).toBeVisible();

    await row.locator('[data-action="requestRegenerate"]').click();
    await confirmPopconfirm(page, '重新生成会取消当前未完成会话');
    await expect(
      page.getByText('已重新生成并加入机器人提醒队列'),
    ).toBeVisible();

    await row.locator('[data-action="requestCancel"]').click();
    await confirmPopconfirm(page, '确认取消该入群认证会话');
    await expect(page.getByText('认证会话已取消')).toBeVisible();

    await expect
      .poll(() => captured.map((item) => `${item.method} ${item.path}`))
      .toEqual(
        expect.arrayContaining([
          'POST /api/v1/admin/admission/sessions/admission-session-action-1/resend',
          'POST /api/v1/admin/admission/sessions/admission-session-action-1/regenerate',
          'POST /api/v1/admin/admission/sessions/admission-session-action-1/cancel',
        ]),
      );
  });

  test('member blacklist mutations preserve their explicit audit payloads', async ({
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
          captured.find(
            (item) =>
              item.path === '/api/v1/admin/member-blacklist' &&
              item.method === 'POST',
          )?.body,
      )
      .toEqual(
        expect.objectContaining({
          platform: 'qq',
          subjectID: '30002',
          guildID: 'guild-2',
          reasonText: '人工风控加入',
        }),
      );
    await expect
      .poll(() => captured)
      .toContainEqual(
        expect.objectContaining({
          path: '/api/v1/admin/member-blacklist/entry-active/release',
          method: 'POST',
          body: {
            releaseReasonCode: 'manual_pardon',
            releaseReason: '人工复核通过',
          },
        }),
      );
  });

  test('system configuration changes still use the dedicated update contract', async ({
    page,
  }) => {
    await page.goto('/users/system-config');

    const row = page.getByRole('row', { name: /review\.retention_days/ });
    await expect(row).toBeVisible();
    await row.getByRole('button', { name: /编辑|Edit/ }).click();
    const dialog = page.getByRole('dialog', { name: /编辑配置|Edit/ });
    await dialog.getByPlaceholder('请输入配置值').fill('180');
    await dialog.getByRole('button', { name: /保存|Save/ }).click();

    await expect
      .poll(() => captured)
      .toContainEqual(
        expect.objectContaining({
          path: '/api/v1/admin/system-configs/review.retention_days',
          method: 'PUT',
          body: { value: '180' },
        }),
      );
  });
});
