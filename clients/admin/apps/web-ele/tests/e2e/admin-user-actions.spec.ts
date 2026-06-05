import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-05-24T04:00:00Z';

const capabilities = [
  'admission:session:read',
  'admission:session:manage',
  'user:identity:read',
  'user:identity:review',
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

const identityVerifications = [
  {
    userID: 11,
    docType: 'MAINLAND_ID',
    realName: '张三',
    verified: false,
    verifyMethod: 'manual',
    reviewedAt: null,
    verifiedAt: null,
    rejectionReason: null,
    createdAt: now,
    updatedAt: now,
  },
  {
    userID: 12,
    docType: 'PASSPORT',
    realName: '李四',
    verified: false,
    verifyMethod: 'manual',
    reviewedAt: null,
    verifiedAt: null,
    rejectionReason: null,
    createdAt: now,
    updatedAt: now,
  },
];

const studentVerifications = [
  {
    userID: 21,
    schoolID: 4_111_010_006,
    activeStudentID: '20260021',
    verificationStatus: 'pending',
    verificationMethod: 'manual',
    createdAt: now,
    updatedAt: now,
  },
  {
    userID: 22,
    schoolID: 4_111_010_006,
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
  schoolID: 4_111_010_006,
  qqID: '20001',
  applicantNameMasked: '王*',
  materialURL: 'https://example.com/material.jpg',
  failureCount: 2,
  createdAt: now,
};
const alternateFreshmanApplication = {
  ...freshmanApplication,
  id: 'freshman-action-2',
  qqID: '20002',
  applicantNameMasked: '李*',
  failureCount: 0,
};

const admissionSession = {
  id: 'admission-session-action-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: '178037297',
  channelID: '178037297',
  qqID: '1390191645',
  userID: '42',
  status: 'linked',
  tokenExpiresAt: '2026-05-24T05:00:00Z',
  tokenConsumedAt: '2026-05-24T04:10:00Z',
  linkWaitDeadlineAt: '2026-05-24T05:00:00Z',
  submissionWaitDeadlineAt: '2026-05-24T06:00:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-06-24T04:00:00Z',
  verifiedAt: null,
  cancelledAt: null,
  lastBotError: 'previous reminder failed',
  projectionPending: false,
  authURL: 'https://join.stuhelper.com/verify/action-token',
};

const alternateAdmissionSession = {
  ...admissionSession,
  id: 'admission-session-action-2',
  qqID: '1390191646',
  userID: '43',
  authURL: 'https://join.stuhelper.com/verify/action-token-2',
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

let nextStudentListErrorMessage: null | string = null;
let nextStudentReviewErrorMessage: null | string = null;
let nextStudentReviewDelay: null | Promise<void> = null;
let nextIdentityListErrorMessage: null | string = null;
let nextIdentityReviewErrorMessage: null | string = null;
let nextIdentityReviewDelay: null | Promise<void> = null;
let identityListTotal = identityVerifications.length;
let nextFreshmanListErrorMessage: null | string = null;
let nextFreshmanReviewErrorMessage: null | string = null;
let freshmanApplications = [freshmanApplication];
let nextFreshmanReviewDelay: null | Promise<void> = null;
let admissionSessions = [admissionSession];
let nextAdmissionSessionListErrorMessage: null | string = null;
let nextAdmissionSessionActionErrorMessage: null | string = null;
let nextAdmissionSessionActionDelay: null | Promise<void> = null;
let nextMemberBlacklistListErrorMessage: null | string = null;
let nextMemberBlacklistActionErrorMessage: null | string = null;
let nextMemberBlacklistActionDelay: null | Promise<void> = null;

interface CapturedMutation {
  body: unknown;
  method: string;
  path: string;
}

interface CapturedQuery {
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

function list<T>(items: T[], total = items.length) {
  return ok({ list: items, total });
}

function parseJsonBody(route: Route) {
  const postData = route.request().postData();
  return postData ? (JSON.parse(postData) as unknown) : null;
}

async function fulfillNextMemberBlacklistActionError(route: Route) {
  if (!nextMemberBlacklistActionErrorMessage) {
    return false;
  }
  const message = nextMemberBlacklistActionErrorMessage;
  nextMemberBlacklistActionErrorMessage = null;
  await route.fulfill(
    json({
      success: false,
      error: {
        code: 'A0090007',
        message,
      },
    }),
  );
  return true;
}

async function confirmPopconfirm(page: Page, title?: RegExp | string) {
  const popconfirm = title
    ? page.locator('.el-popper').filter({ hasText: title }).last()
    : page.locator('.el-popper').last();
  await expect(popconfirm).toBeVisible();
  await popconfirm.getByRole('button', { name: /确定|Confirm/ }).click();
}

async function mockAdminApi(
  page: Page,
  capturedMutations: CapturedMutation[],
  capturedQueries: CapturedQuery[],
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

    if (path === '/api/v1/admin/identities') {
      capturedQueries.push({
        path,
        method,
        query: Object.fromEntries(url.searchParams.entries()),
      });
      if (nextIdentityListErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090010',
              message: nextIdentityListErrorMessage,
            },
          }),
        );
        nextIdentityListErrorMessage = null;
        return;
      }
      await route.fulfill(list(identityVerifications, identityListTotal));
      return;
    }
    if (path.startsWith('/api/v1/admin/identities/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextIdentityReviewErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090011',
              message: nextIdentityReviewErrorMessage,
            },
          }),
        );
        nextIdentityReviewErrorMessage = null;
        return;
      }
      if (nextIdentityReviewDelay) {
        await nextIdentityReviewDelay;
        nextIdentityReviewDelay = null;
      }
      await route.fulfill(
        ok({ ...identityVerifications[0], verified: true, verifiedAt: now }),
      );
      return;
    }

    if (path === '/api/v1/admin/student-verifications') {
      if (nextStudentListErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090002',
              message: nextStudentListErrorMessage,
            },
          }),
        );
        nextStudentListErrorMessage = null;
        return;
      }
      await route.fulfill(list(studentVerifications));
      return;
    }
    if (path.startsWith('/api/v1/admin/student-verifications/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextStudentReviewErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090001',
              message: nextStudentReviewErrorMessage,
            },
          }),
        );
        nextStudentReviewErrorMessage = null;
        return;
      }
      if (nextStudentReviewDelay) {
        await nextStudentReviewDelay;
        nextStudentReviewDelay = null;
      }
      await route.fulfill(
        ok({ ...studentVerifications[0], verificationStatus: 'verified' }),
      );
      return;
    }

    if (path === '/api/v1/admin/freshman-verifications') {
      if (nextFreshmanListErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090004',
              message: nextFreshmanListErrorMessage,
            },
          }),
        );
        nextFreshmanListErrorMessage = null;
        return;
      }
      await route.fulfill(list(freshmanApplications));
      return;
    }
    if (path.startsWith('/api/v1/admin/freshman-verifications/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextFreshmanReviewErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090005',
              message: nextFreshmanReviewErrorMessage,
            },
          }),
        );
        nextFreshmanReviewErrorMessage = null;
        return;
      }
      if (nextFreshmanReviewDelay) {
        await nextFreshmanReviewDelay;
        nextFreshmanReviewDelay = null;
      }
      await route.fulfill(ok({ ...freshmanApplication, status: 'approved' }));
      return;
    }

    if (path === '/api/v1/admin/admission/sessions') {
      capturedQueries.push({
        path,
        method,
        query: Object.fromEntries(url.searchParams.entries()),
      });
      if (nextAdmissionSessionListErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090003',
              message: nextAdmissionSessionListErrorMessage,
            },
          }),
        );
        nextAdmissionSessionListErrorMessage = null;
        return;
      }
      await route.fulfill(list(admissionSessions));
      return;
    }
    if (path.startsWith('/api/v1/admin/admission/sessions/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextAdmissionSessionActionErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090006',
              message: nextAdmissionSessionActionErrorMessage,
            },
          }),
        );
        nextAdmissionSessionActionErrorMessage = null;
        return;
      }
      if (nextAdmissionSessionActionDelay) {
        await nextAdmissionSessionActionDelay;
        nextAdmissionSessionActionDelay = null;
      }
      if (path.endsWith('/regenerate')) {
        await route.fulfill(
          ok({
            authURL: 'https://join.stuhelper.com/verify/regenerated-token',
            session: {
              ...admissionSession,
              id: 'admission-session-action-2',
              authURL: 'https://join.stuhelper.com/verify/regenerated-token',
            },
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
      if (method === 'GET' && nextMemberBlacklistListErrorMessage) {
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'A0090005',
              message: nextMemberBlacklistListErrorMessage,
            },
          }),
        );
        nextMemberBlacklistListErrorMessage = null;
        return;
      }
      if (
        method === 'POST' &&
        (await fulfillNextMemberBlacklistActionError(route))
      ) {
        return;
      }
      capturedMutations.push({
        path,
        method,
        body: method === 'POST' ? parseJsonBody(route) : null,
      });
      if (method === 'POST' && nextMemberBlacklistActionDelay) {
        await nextMemberBlacklistActionDelay;
        nextMemberBlacklistActionDelay = null;
      }
      await route.fulfill(
        method === 'POST'
          ? ok({ ...blacklistEntry, id: 'entry-created' })
          : list([blacklistEntry]),
      );
      return;
    }
    if (path.startsWith('/api/v1/admin/member-blacklist/')) {
      if (await fulfillNextMemberBlacklistActionError(route)) {
        return;
      }
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextMemberBlacklistActionDelay) {
        await nextMemberBlacklistActionDelay;
        nextMemberBlacklistActionDelay = null;
      }
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
  let capturedQueries: CapturedQuery[];

  test.beforeEach(async ({ page }) => {
    capturedMutations = [];
    capturedQueries = [];
    nextIdentityListErrorMessage = null;
    nextIdentityReviewErrorMessage = null;
    nextIdentityReviewDelay = null;
    identityListTotal = identityVerifications.length;
    nextStudentListErrorMessage = null;
    nextStudentReviewErrorMessage = null;
    nextStudentReviewDelay = null;
    nextFreshmanListErrorMessage = null;
    nextFreshmanReviewErrorMessage = null;
    freshmanApplications = [freshmanApplication];
    nextFreshmanReviewDelay = null;
    admissionSessions = [admissionSession];
    nextAdmissionSessionListErrorMessage = null;
    nextAdmissionSessionActionErrorMessage = null;
    nextAdmissionSessionActionDelay = null;
    nextMemberBlacklistActionErrorMessage = null;
    nextMemberBlacklistListErrorMessage = null;
    nextMemberBlacklistActionDelay = null;
    await mockAdminApi(page, capturedMutations, capturedQueries);
  });

  test('identity review list failures show a persistent retry path', async ({
    page,
  }) => {
    nextIdentityListErrorMessage = '实名审核列表暂不可用';

    await page.goto('/users/identity-review');

    const loadError = page.locator('.admin-load-error', {
      hasText: '实名审核列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('张三')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('张三')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('identity review filters request the first page after pagination changes', async ({
    page,
  }) => {
    identityListTotal = 40;

    await page.goto('/users/identity-review');
    await expect(page.getByText('张三')).toBeVisible();

    await page
      .locator('.el-pagination .el-pager li')
      .filter({ hasText: /^2$/ })
      .click();
    await expect
      .poll(() =>
        capturedQueries.some(
          (request) =>
            request.path === '/api/v1/admin/identities' &&
            request.method === 'GET' &&
            request.query.page === '2' &&
            request.query.pageSize === '20',
        ),
      )
      .toBe(true);

    await page.getByRole('main').locator('.el-select').first().click();
    await page.getByRole('option', { name: /已认证|Verified/ }).click();

    await expect
      .poll(() => capturedQueries.at(-1))
      .toEqual(
        expect.objectContaining({
          path: '/api/v1/admin/identities',
          method: 'GET',
          query: expect.objectContaining({
            page: '1',
            pageSize: '20',
            status: 'verified',
          }),
        }),
      );
  });

  test('identity review buttons submit approve and reject decisions', async ({
    page,
  }) => {
    await page.goto('/users/identity-review');

    const approveRow = page.getByRole('row', { name: /张三/ });
    await expect(approveRow).toBeVisible();
    await approveRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);
    await expect(page.getByText('实名认证已通过')).toBeVisible();

    const rejectRow = page.getByRole('row', { name: /李四/ });
    await rejectRow.getByRole('button', { name: /驳回|Reject/ }).click();
    const rejectDialog = page.getByRole('dialog', {
      name: /驳回实名认证|Reject Identity Verification/,
    });
    await rejectDialog
      .getByPlaceholder(/请输入驳回原因|Enter rejection reason/)
      .fill('证件照片不清晰');
    await rejectDialog.getByRole('button', { name: /确定|Confirm/ }).click();
    await expect(page.getByText('实名认证已驳回')).toBeVisible();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/11',
        method: 'PUT',
        body: { approved: true },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/12',
        method: 'PUT',
        body: { approved: false, rejectionReason: '证件照片不清晰' },
      });
  });

  test('identity review action loading stays scoped to the active row', async ({
    page,
  }) => {
    let releaseReview!: () => void;
    nextIdentityReviewDelay = new Promise<void>((resolve) => {
      releaseReview = resolve;
    });

    await page.goto('/users/identity-review');

    const activeRow = page.getByRole('row', { name: /张三/ });
    const otherRow = page.getByRole('row', { name: /李四/ });
    await expect(activeRow).toBeVisible();
    await expect(otherRow).toBeVisible();

    const activeApprove = activeRow.locator('[data-action="approve"]');
    await activeApprove.click();
    await confirmPopconfirm(page);

    await expect(activeApprove).toBeDisabled();
    await expect(activeRow.locator('[data-action="reject"]')).toBeDisabled();
    await expect(otherRow.locator('[data-action="approve"]')).toBeEnabled();
    await expect(otherRow.locator('[data-action="reject"]')).toBeEnabled();

    releaseReview();
    await expect(page.getByText('实名认证已通过')).toBeVisible();
  });

  test('identity review failures preserve backend error detail and reject draft', async ({
    page,
  }) => {
    nextIdentityReviewErrorMessage = '实名审核已被其他管理员处理';

    await page.goto('/users/identity-review');

    const rejectRow = page.getByRole('row', { name: /李四/ });
    await expect(rejectRow).toBeVisible();
    await rejectRow.getByRole('button', { name: /驳回|Reject/ }).click();
    const rejectDialog = page.getByRole('dialog', {
      name: /驳回实名认证|Reject Identity Verification/,
    });
    const reasonInput = rejectDialog.getByPlaceholder(
      /请输入驳回原因|Enter rejection reason/,
    );
    await reasonInput.fill('证件照片不清晰');
    await rejectDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '实名审核已被其他管理员处理',
    });
    await expect(actionError).toBeVisible();
    await expect(rejectDialog).toBeVisible();
    await expect(reasonInput).toHaveValue('证件照片不清晰');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/12',
        method: 'PUT',
        body: { approved: false, rejectionReason: '证件照片不清晰' },
      });
  });

  test('student verification list failures show a persistent retry path', async ({
    page,
  }) => {
    nextStudentListErrorMessage = '学生认证列表暂不可用';

    await page.goto('/users/student-verification');

    const loadError = page.locator('.admin-load-error', {
      hasText: '学生认证列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('20260021')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('20260021')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('student verification buttons submit approve and reject decisions', async ({
    page,
  }) => {
    await page.goto('/users/student-verification');

    const approveRow = page.getByRole('row', { name: /20260021/ });
    await expect(approveRow).toBeVisible();
    await approveRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);
    await expect(page.getByText('学生认证已通过')).toBeVisible();

    const rejectRow = page.getByRole('row', { name: /20260022/ });
    await rejectRow.getByRole('button', { name: /驳回|Reject/ }).click();
    const rejectDialog = page.getByRole('dialog', {
      name: /驳回学生认证|Reject Student Verification/,
    });
    await rejectDialog
      .getByPlaceholder(/请输入驳回原因|Enter rejection reason/)
      .fill('学籍记录未匹配');
    await rejectDialog.getByRole('button', { name: /确定|Confirm/ }).click();
    await expect(page.getByText('学生认证已驳回')).toBeVisible();

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
        body: { approved: false, rejectionReason: '学籍记录未匹配' },
      });
  });

  test('student verification action loading stays scoped to the active row', async ({
    page,
  }) => {
    let releaseReview!: () => void;
    nextStudentReviewDelay = new Promise<void>((resolve) => {
      releaseReview = resolve;
    });

    await page.goto('/users/student-verification');

    const activeRow = page.getByRole('row', { name: /20260021/ });
    const otherRow = page.getByRole('row', { name: /20260022/ });
    await expect(activeRow).toBeVisible();
    await expect(otherRow).toBeVisible();

    const activeApprove = activeRow.locator('[data-action="approve"]');
    await activeApprove.click();
    await confirmPopconfirm(page);

    await expect(activeApprove).toBeDisabled();
    await expect(activeRow.locator('[data-action="reject"]')).toBeDisabled();
    await expect(otherRow.locator('[data-action="approve"]')).toBeEnabled();
    await expect(otherRow.locator('[data-action="reject"]')).toBeEnabled();

    releaseReview();
    await expect(page.getByText('学生认证已通过')).toBeVisible();
  });

  test('student verification review failures preserve backend error detail', async ({
    page,
  }) => {
    nextStudentReviewErrorMessage = '学生认证已被其他管理员处理';

    await page.goto('/users/student-verification');

    const approveRow = page.getByRole('row', { name: /20260021/ });
    await expect(approveRow).toBeVisible();
    await approveRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);

    const actionError = page.locator('.admin-load-error', {
      hasText: '学生认证已被其他管理员处理',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/student-verifications/21',
        method: 'PUT',
        body: { approved: true },
      });
  });

  test('student verification reject failures keep the rejection dialog draft', async ({
    page,
  }) => {
    nextStudentReviewErrorMessage = '学生认证驳回失败';

    await page.goto('/users/student-verification');

    const rejectRow = page.getByRole('row', { name: /20260022/ });
    await expect(rejectRow).toBeVisible();
    await rejectRow.getByRole('button', { name: /驳回|Reject/ }).click();
    const rejectDialog = page.getByRole('dialog', {
      name: /驳回学生认证|Reject Student Verification/,
    });
    const reasonInput = rejectDialog.getByPlaceholder(
      /请输入驳回原因|Enter rejection reason/,
    );
    await reasonInput.fill('学籍记录未匹配');
    await rejectDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '学生认证驳回失败',
    });
    await expect(actionError).toBeVisible();
    await expect(rejectDialog).toBeVisible();
    await expect(reasonInput).toHaveValue('学籍记录未匹配');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/student-verifications/22',
        method: 'PUT',
        body: { approved: false, rejectionReason: '学籍记录未匹配' },
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
    await expect(
      page.getByText('已通过新生审核并设置临时认证期限'),
    ).toBeVisible();

    await page.getByRole('textbox', { name: '驳回原因' }).fill('材料不清晰');
    await page.locator('[data-action="reject"]:visible').click();
    await expect(page.getByText('已驳回新生审核')).toBeVisible();

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

  test('freshman verification action loading stays scoped to the active row', async ({
    page,
  }) => {
    freshmanApplications = [freshmanApplication, alternateFreshmanApplication];
    let releaseReview!: () => void;
    nextFreshmanReviewDelay = new Promise<void>((resolve) => {
      releaseReview = resolve;
    });

    await page.goto('/users/freshman-verification');

    const activeRow = page.getByRole('row', { name: /王\*/ });
    const otherRow = page.getByRole('row', { name: /李\*/ });
    await expect(activeRow).toBeVisible();
    await expect(otherRow).toBeVisible();

    const activeApprove = activeRow.locator('[data-action="approve"]');
    await activeApprove.click();

    await expect(activeApprove).toBeDisabled();
    await expect(
      activeRow.locator('[data-action="approveWithDays"]'),
    ).toBeDisabled();
    await expect(activeRow.locator('[data-action="reject"]')).toBeDisabled();
    await expect(otherRow.locator('[data-action="approve"]')).toBeEnabled();
    await expect(
      otherRow.locator('[data-action="approveWithDays"]'),
    ).toBeEnabled();
    await expect(otherRow.locator('[data-action="reject"]')).toBeEnabled();

    releaseReview();
    await expect(page.getByText('已通过新生审核')).toBeVisible();
  });

  test('freshman verification review failures preserve backend error detail', async ({
    page,
  }) => {
    nextFreshmanReviewErrorMessage = '新生认证已被其他管理员处理';

    await page.goto('/users/freshman-verification');

    const freshmanRow = page.getByRole('row', { name: /王\*/ });
    await expect(freshmanRow).toBeVisible();
    await freshmanRow.locator('[data-action="approve"]').click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '新生认证已被其他管理员处理',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/freshman-verifications/freshman-action-1',
        method: 'PUT',
        body: { action: 'approve' },
      });
  });

  test('freshman verification reject failures preserve backend error detail and draft', async ({
    page,
  }) => {
    nextFreshmanReviewErrorMessage = '新生认证驳回失败';

    await page.goto('/users/freshman-verification');

    const freshmanRow = page.getByRole('row', { name: /王\*/ });
    await expect(freshmanRow).toBeVisible();
    const reasonInput = page.getByRole('textbox', { name: '驳回原因' });
    await reasonInput.fill('材料不清晰');
    await freshmanRow.locator('[data-action="reject"]').click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '新生认证驳回失败',
    });
    await expect(actionError).toBeVisible();
    await expect(reasonInput).toHaveValue('材料不清晰');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/freshman-verifications/freshman-action-1',
        method: 'PUT',
        body: { action: 'reject', reason: '材料不清晰' },
      });
  });

  test('freshman verification list failures show a persistent retry path', async ({
    page,
  }) => {
    nextFreshmanListErrorMessage = '新生认证列表暂不可用';

    await page.goto('/users/freshman-verification');

    const loadError = page.locator('.admin-load-error', {
      hasText: '新生认证列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('王*')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('王*')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('admission sessions filter, copy commands, and queue bot actions', async ({
    page,
  }) => {
    await page.addInitScript(() => {
      const clipboardState = globalThis as unknown as {
        __clipboardWrites: string[];
      };
      clipboardState.__clipboardWrites = [];
      Object.defineProperty(navigator, 'clipboard', {
        configurable: true,
        value: {
          writeText: (text: string) => {
            clipboardState.__clipboardWrites.push(text);
            return Promise.resolve();
          },
        },
      });
    });

    await page.goto('/users/admission-sessions');

    const sessionRow = page.getByRole('row', { name: /1390191645/ });
    await expect(sessionRow).toBeVisible();
    await expect(sessionRow).toContainText('已绑定账号');
    await expect(sessionRow).toContainText('previous reminder failed');

    await page.getByPlaceholder('QQ 号').fill(' 1390191645 ');
    await page.getByPlaceholder('群号').fill(' 178037297 ');
    await page.getByPlaceholder('Bot QQ').fill(' 2118785781 ');
    await page.getByRole('button', { name: '查询' }).click();

    await expect
      .poll(() => capturedQueries.at(-1))
      .toEqual(
        expect.objectContaining({
          path: '/api/v1/admin/admission/sessions',
          method: 'GET',
          query: expect.objectContaining({
            page: '1',
            pageSize: '20',
            platform: 'qq',
            qqID: '1390191645',
            guildID: '178037297',
            botSelfID: '2118785781',
          }),
        }),
      );

    await sessionRow.locator('[data-action="copyAuthURL"]').click();
    await expect(page.getByText('认证链接已复制')).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(
          () =>
            (globalThis as unknown as { __clipboardWrites?: string[] })
              .__clipboardWrites ?? [],
        ),
      )
      .toContain(admissionSession.authURL);

    await page.evaluate(() => {
      const state = globalThis as unknown as {
        __clipboardFallbackWrites?: string[];
      };
      state.__clipboardFallbackWrites = [];
      const browser = globalThis as unknown as {
        document: {
          body: {
            querySelector: (selector: string) => null | { value?: string };
          };
          execCommand?: (command: string) => boolean;
        };
        navigator: {
          clipboard?: unknown;
        };
      };
      const originalExecCommand = browser.document.execCommand?.bind(
        browser.document,
      );
      Object.defineProperty(browser.navigator, 'clipboard', {
        configurable: true,
        value: {
          writeText: async () => {
            throw new Error('clipboard denied');
          },
        },
      });
      Object.defineProperty(browser.document, 'execCommand', {
        configurable: true,
        value: (command: string) => {
          if (command !== 'copy') {
            return originalExecCommand?.(command) ?? false;
          }
          state.__clipboardFallbackWrites?.push(
            browser.document.body.querySelector('textarea')?.value ?? '',
          );
          return true;
        },
      });
    });
    await sessionRow.locator('[data-action="copyReissueCommand"]').click();
    await expect(page.getByText('重生命令已复制')).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(
          () =>
            (globalThis as unknown as { __clipboardFallbackWrites?: string[] })
              .__clipboardFallbackWrites ?? [],
        ),
      )
      .toContain('重新生成认证链接 1390191645');

    await sessionRow.locator('[data-action="requestResend"]').click();
    await expect(page.getByText('已加入机器人重发队列')).toBeVisible();

    await sessionRow.locator('[data-action="requestRegenerate"]').click();
    await confirmPopconfirm(page, '重新生成会取消当前未完成会话');
    await expect(
      page.getByText('已重新生成并加入机器人提醒队列'),
    ).toBeVisible();

    await sessionRow.locator('[data-action="requestCancel"]').click();
    await confirmPopconfirm(page, '确认取消该入群认证会话');
    await expect(page.getByText('认证会话已取消')).toBeVisible();

    await expect
      .poll(() => capturedMutations)
      .toEqual(
        expect.arrayContaining([
          {
            path: '/api/v1/admin/admission/sessions/admission-session-action-1/resend',
            method: 'POST',
            body: null,
          },
          {
            path: '/api/v1/admin/admission/sessions/admission-session-action-1/regenerate',
            method: 'POST',
            body: null,
          },
          {
            path: '/api/v1/admin/admission/sessions/admission-session-action-1/cancel',
            method: 'POST',
            body: null,
          },
        ]),
      );
  });

  test('admission session action loading stays scoped to the active row', async ({
    page,
  }) => {
    admissionSessions = [admissionSession, alternateAdmissionSession];
    let releaseAction!: () => void;
    nextAdmissionSessionActionDelay = new Promise<void>((resolve) => {
      releaseAction = resolve;
    });

    await page.goto('/users/admission-sessions');

    const activeRow = page.getByRole('row', { name: /1390191645/ });
    const otherRow = page.getByRole('row', { name: /1390191646/ });
    await expect(activeRow).toBeVisible();
    await expect(otherRow).toBeVisible();

    const activeResend = activeRow.locator('[data-action="requestResend"]');
    await activeResend.click();

    await expect(activeResend).toBeDisabled();
    await expect(
      activeRow.locator('[data-action="requestRegenerate"]'),
    ).toBeDisabled();
    await expect(
      activeRow.locator('[data-action="requestCancel"]'),
    ).toBeDisabled();
    await expect(
      otherRow.locator('[data-action="requestResend"]'),
    ).toBeEnabled();
    await expect(
      otherRow.locator('[data-action="requestRegenerate"]'),
    ).toBeEnabled();
    await expect(
      otherRow.locator('[data-action="requestCancel"]'),
    ).toBeEnabled();

    releaseAction();
    await expect(page.getByText('已加入机器人重发队列')).toBeVisible();
  });

  test('admission sessions status and page size pass query params', async ({
    page,
  }) => {
    await page.goto('/users/admission-sessions');

    const sessionRow = page.getByRole('row', { name: /1390191645/ });
    await expect(sessionRow).toBeVisible();

    await page.locator('.el-select[data-field="status"]').click();
    await page.getByRole('option', { name: '材料待审' }).click();

    await expect
      .poll(() => capturedQueries.at(-1))
      .toEqual(
        expect.objectContaining({
          path: '/api/v1/admin/admission/sessions',
          method: 'GET',
          query: expect.objectContaining({
            page: '1',
            pageSize: '20',
            platform: 'qq',
            status: 'material_submitted',
          }),
        }),
      );

    await page.locator('.admin-embedded-pagination .el-select').click();
    await page.getByRole('option', { name: /50/ }).click();

    await expect
      .poll(() => capturedQueries.at(-1))
      .toEqual(
        expect.objectContaining({
          path: '/api/v1/admin/admission/sessions',
          method: 'GET',
          query: expect.objectContaining({
            page: '1',
            pageSize: '50',
            platform: 'qq',
            status: 'material_submitted',
          }),
        }),
      );
  });

  test('admission session action failures preserve backend error detail', async ({
    page,
  }) => {
    nextAdmissionSessionActionErrorMessage = '机器人重发队列暂不可用';

    await page.goto('/users/admission-sessions');

    const sessionRow = page.getByRole('row', { name: /1390191645/ });
    await expect(sessionRow).toBeVisible();
    await sessionRow.locator('[data-action="requestResend"]').click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '机器人重发队列暂不可用',
    });
    await expect(actionError).toBeVisible();
    await expect(sessionRow).toBeVisible();
    await expect(sessionRow).toContainText('previous reminder failed');
    await expect(
      sessionRow.locator('[data-action="requestResend"]'),
    ).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/admission/sessions/admission-session-action-1/resend',
        method: 'POST',
        body: null,
      });
  });

  test('admission session regenerate failures keep the current session visible', async ({
    page,
  }) => {
    nextAdmissionSessionActionErrorMessage = '认证链接重新生成失败';

    await page.goto('/users/admission-sessions');

    const sessionRow = page.getByRole('row', { name: /1390191645/ });
    await expect(sessionRow).toBeVisible();
    await sessionRow.locator('[data-action="requestRegenerate"]').click();
    await confirmPopconfirm(page, '重新生成会取消当前未完成会话');

    const actionError = page.locator('.admin-load-error', {
      hasText: '认证链接重新生成失败',
    });
    await expect(actionError).toBeVisible();
    await expect(sessionRow).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/admission/sessions/admission-session-action-1/regenerate',
        method: 'POST',
        body: null,
      });
  });

  test('admission session cancel failures keep the current session visible', async ({
    page,
  }) => {
    nextAdmissionSessionActionErrorMessage = '入群认证会话取消失败';

    await page.goto('/users/admission-sessions');

    const sessionRow = page.getByRole('row', { name: /1390191645/ });
    await expect(sessionRow).toBeVisible();
    await sessionRow.locator('[data-action="requestCancel"]').click();
    await confirmPopconfirm(page, '确认取消该入群认证会话');

    const actionError = page.locator('.admin-load-error', {
      hasText: '入群认证会话取消失败',
    });
    await expect(actionError).toBeVisible();
    await expect(sessionRow).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/admission/sessions/admission-session-action-1/cancel',
        method: 'POST',
        body: null,
      });
  });

  test('admission session list failures show a persistent retry path', async ({
    page,
  }) => {
    nextAdmissionSessionListErrorMessage = '入群认证会话暂不可用';

    await page.goto('/users/admission-sessions');

    const loadError = page.locator('.admin-load-error', {
      hasText: '入群认证会话暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('1390191645')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('1390191645')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('member blacklist list failures show a persistent retry path', async ({
    page,
  }) => {
    nextMemberBlacklistListErrorMessage = '成员黑名单列表暂不可用';

    await page.goto('/users/member-blacklist');

    const loadError = page.locator('.admin-load-error', {
      hasText: '成员黑名单列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('too many failures')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('too many failures')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('member blacklist create failures preserve backend error detail and draft', async ({
    page,
  }) => {
    nextMemberBlacklistActionErrorMessage = '成员黑名单创建失败';

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

    const actionError = page.locator('.admin-load-error', {
      hasText: '成员黑名单创建失败',
    });
    await expect(actionError).toBeVisible();
    await expect(createDialog).toBeVisible();
    await expect(
      createDialog.getByRole('textbox', { name: '主体 ID（QQ 号）' }),
    ).toHaveValue('30002');
    await expect(page.getByText('too many failures')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('member blacklist create submit disables while the request is pending', async ({
    page,
  }) => {
    let releaseAction!: () => void;
    nextMemberBlacklistActionDelay = new Promise<void>((resolve) => {
      releaseAction = resolve;
    });

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

    const createButton = createDialog.getByRole('button', { name: '创建' });
    await createButton.click();

    await expect(createButton).toBeDisabled();
    await expect
      .poll(
        () =>
          capturedMutations.filter(
            (mutation) =>
              mutation.path === '/api/v1/admin/member-blacklist' &&
              mutation.method === 'POST',
          ).length,
      )
      .toBe(1);

    releaseAction();
    await expect(page.getByText('已将 30002 加入黑名单')).toBeVisible();
  });

  test('member blacklist release failures preserve backend error detail and draft', async ({
    page,
  }) => {
    nextMemberBlacklistActionErrorMessage = '成员黑名单解除失败';

    await page.goto('/users/member-blacklist');
    await expect(page.getByText('too many failures')).toBeVisible();

    await page
      .getByRole('row', { name: /30001/ })
      .getByRole('button', { name: '解除' })
      .click();

    const releaseDialog = page.getByRole('dialog', { name: '解除成员黑名单' });
    await releaseDialog
      .getByRole('textbox', { name: '备注（可选）' })
      .fill('人工复核通过');
    await releaseDialog.getByRole('button', { name: '解除' }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '成员黑名单解除失败',
    });
    await expect(actionError).toBeVisible();
    await expect(releaseDialog).toBeVisible();
    await expect(
      releaseDialog.getByRole('textbox', { name: '备注（可选）' }),
    ).toHaveValue('人工复核通过');
    await expect(page.getByText('too many failures')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
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
