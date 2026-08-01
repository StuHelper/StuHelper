import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-05-24T04:00:00Z';

const capabilities = [
  'admin:reviews:manage',
  'admin:reports:manage',
  'user:identity:read',
  'user:identity:review',
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

const pendingReview = {
  id: 'review-pending',
  courseID: 42,
  courseName: '操作系统',
  teacherName: '陈老师',
  title: '待审评课标题',
  content: '这条评课需要管理员审核。',
  ratings: { clarity: 4, recommendation: 5 },
  status: 'pending_review',
  createdAt: now,
};

const hiddenReview = {
  ...pendingReview,
  id: 'review-hidden',
  title: '已隐藏评课标题',
  status: 'hidden',
};

const pendingReport = {
  id: 'report-pending',
  reviewID: pendingReview.id,
  review: pendingReview,
  reason: 'spam',
  description: '举报理由：疑似广告。',
  resolutionNote: null,
  status: 'pending',
  createdAt: now,
};

const identityReviews = [
  {
    userID: 12,
    realName: '张三',
    docType: 'MAINLAND_ID',
    verifyMethod: 'manual',
    verified: false,
    verifiedAt: null,
    reviewedAt: null,
    createdAt: now,
    updatedAt: now,
  },
  {
    userID: 13,
    realName: '李四',
    docType: 'MAINLAND_ID',
    verifyMethod: 'manual',
    verified: false,
    verifiedAt: null,
    reviewedAt: null,
    createdAt: now,
    updatedAt: now,
  },
];

let nextIdentityReviewListErrorMessage: null | string = null;
let nextIdentityReviewActionErrorMessage: null | string = null;
let nextIdentityReviewActionDelay: null | Promise<void> = null;
let nextIdentityReviewDetailErrorMessage: null | string = null;
let identityReviewDetailMissingSelfie = false;
let nextReportListErrorMessage: null | string = null;
let nextReportActionErrorMessage: null | string = null;
let nextReviewListErrorMessage: null | string = null;
let nextReviewActionErrorMessage: null | string = null;

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

async function openIdentityEvidence(page: Page, rowName: RegExp) {
  const row = page.getByRole('row', { name: rowName });
  await expect(row).toBeVisible();
  await row.getByRole('button', { name: /查看材料|Review Evidence/ }).click();
  const dialog = page.getByRole('dialog', {
    name: /核验实名认证材料|Review Identity Evidence/,
  });
  await expect(dialog).toBeVisible();
  await expect(
    dialog.getByRole('img', { name: /证件正面|Document Front/ }),
  ).toBeVisible();
  return dialog;
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

    if (path === '/api/v1/course/review/admin/reviews') {
      if (nextReviewListErrorMessage) {
        const message = nextReviewListErrorMessage;
        nextReviewListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_REVIEW_LIST_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(list([pendingReview, hiddenReview]));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/reviews/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextReviewActionErrorMessage) {
        const message = nextReviewActionErrorMessage;
        nextReviewActionErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_REVIEW_ACTION_REJECTED',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok({ ...pendingReview, status: 'published' }));
      return;
    }

    if (path === '/api/v1/course/review/admin/reports') {
      if (nextReportListErrorMessage) {
        const message = nextReportListErrorMessage;
        nextReportListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_REPORT_LIST_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(list([pendingReport]));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/reports/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextReportActionErrorMessage) {
        const message = nextReportActionErrorMessage;
        nextReportActionErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_REPORT_ACTION_REJECTED',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok({ ...pendingReport, status: 'rejected' }));
      return;
    }

    if (path === '/api/v1/admin/identities') {
      if (nextIdentityReviewListErrorMessage) {
        const message = nextIdentityReviewListErrorMessage;
        nextIdentityReviewListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_IDENTITY_REVIEW_LIST_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(list(identityReviews));
      return;
    }
    if (path.startsWith('/api/v1/admin/identities/')) {
      if (method === 'GET') {
        if (nextIdentityReviewDetailErrorMessage) {
          const message = nextIdentityReviewDetailErrorMessage;
          nextIdentityReviewDetailErrorMessage = null;
          await route.fulfill(
            json({
              success: false,
              error: {
                code: 'E2E_IDENTITY_REVIEW_DETAIL_UNAVAILABLE',
                message,
              },
            }),
          );
          return;
        }
        const userID = Number(path.split('/').at(-1));
        const item =
          identityReviews.find((identity) => identity.userID === userID) ??
          identityReviews[0];
        await route.fulfill(
          ok({
            ...item,
            docPhotoFrontURL:
              'data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=',
            docPhotoBackURL: null,
            docPhotoSelfieURL: identityReviewDetailMissingSelfie
              ? null
              : 'data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=',
          }),
        );
        return;
      }
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      if (nextIdentityReviewActionErrorMessage) {
        const message = nextIdentityReviewActionErrorMessage;
        nextIdentityReviewActionErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_IDENTITY_REVIEW_CONFLICT',
              message,
            },
          }),
        );
        return;
      }
      if (nextIdentityReviewActionDelay) {
        await nextIdentityReviewActionDelay;
        nextIdentityReviewActionDelay = null;
      }
      await route.fulfill(ok({ ...identityReviews[0], verified: true }));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked admin action request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin management actions', () => {
  let capturedMutations: CapturedMutation[];

  test.beforeEach(async ({ page }) => {
    capturedMutations = [];
    nextIdentityReviewListErrorMessage = null;
    nextIdentityReviewActionErrorMessage = null;
    nextIdentityReviewActionDelay = null;
    nextIdentityReviewDetailErrorMessage = null;
    identityReviewDetailMissingSelfie = false;
    nextReportListErrorMessage = null;
    nextReportActionErrorMessage = null;
    nextReviewListErrorMessage = null;
    nextReviewActionErrorMessage = null;
    await mockAdminApi(page, capturedMutations);
  });

  test('review list failures show a persistent retry path', async ({
    page,
  }) => {
    nextReviewListErrorMessage = '评课审核列表暂不可用';

    await page.goto('/content/reviews');

    const loadError = page.locator('.admin-load-error', {
      hasText: '评课审核列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('待审评课标题')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('待审评课标题')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('review moderation buttons send the expected update actions', async ({
    page,
  }) => {
    await page.goto('/content/reviews');

    const pendingRow = page.getByRole('row', { name: /待审评课标题/ });
    await expect(pendingRow).toBeVisible();
    await pendingRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);

    const hiddenRow = page.getByRole('row', { name: /已隐藏评课标题/ });
    await hiddenRow.getByRole('button', { name: /恢复|Restore/ }).click();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/reviews/review-pending',
        method: 'PUT',
        body: { action: 'restore' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/reviews/review-hidden',
        method: 'PUT',
        body: { action: 'restore' },
      });
  });

  test('review moderation failures preserve backend error detail', async ({
    page,
  }) => {
    nextReviewActionErrorMessage = '评课审核状态已被其他管理员更新';

    await page.goto('/content/reviews');

    const pendingRow = page.getByRole('row', { name: /待审评课标题/ });
    await expect(pendingRow).toBeVisible();
    await pendingRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);

    const actionError = page.locator('.admin-load-error', {
      hasText: '评课审核状态已被其他管理员更新',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('report list failures show a persistent retry path', async ({
    page,
  }) => {
    nextReportListErrorMessage = '举报列表暂不可用';

    await page.goto('/content/reports');

    const loadError = page.locator('.admin-load-error', {
      hasText: '举报列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('举报理由：疑似广告')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('举报理由：疑似广告')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('report handling buttons send reject and review disposition actions', async ({
    page,
  }) => {
    await page.goto('/content/reports');

    const reportRow = page.getByRole('row', { name: /举报理由：疑似广告/ });
    await expect(reportRow).toBeVisible();

    await reportRow.getByRole('button', { name: /驳回|Reject/ }).click();
    await confirmPopconfirm(page);

    await reportRow
      .getByRole('button', { name: /隐藏评课|Hide Review/ })
      .click();
    await confirmPopconfirm(page);

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/reports/report-pending',
        method: 'PUT',
        body: { action: 'reject' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/reports/report-pending',
        method: 'PUT',
        body: { action: 'hide' },
      });
  });

  test('report handling failures preserve backend error detail', async ({
    page,
  }) => {
    nextReportActionErrorMessage = '举报处理状态已被其他管理员更新';

    await page.goto('/content/reports');

    const reportRow = page.getByRole('row', { name: /举报理由：疑似广告/ });
    await expect(reportRow).toBeVisible();
    await reportRow.getByRole('button', { name: /驳回|Reject/ }).click();
    await confirmPopconfirm(page);

    const actionError = page.locator('.admin-load-error', {
      hasText: '举报处理状态已被其他管理员更新',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('identity review list failures show a persistent retry path', async ({
    page,
  }) => {
    nextIdentityReviewListErrorMessage = '实名认证审核列表暂不可用';

    await page.goto('/users/identity-review');

    const loadError = page.locator('.admin-load-error', {
      hasText: '实名认证审核列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('张三')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('张三')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('identity review detail failures keep a retry path inside the evidence dialog', async ({
    page,
  }) => {
    nextIdentityReviewDetailErrorMessage = '实名认证材料暂不可用';

    await page.goto('/users/identity-review');
    const row = page.getByRole('row', { name: /张三/ });
    await row.getByRole('button', { name: /查看材料|Review Evidence/ }).click();

    const dialog = page.getByRole('dialog', {
      name: /核验实名认证材料|Review Identity Evidence/,
    });
    await expect(dialog.getByText('实名认证材料暂不可用')).toBeVisible();
    await dialog.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(
      dialog.getByRole('img', { name: /证件正面|Document Front/ }),
    ).toBeVisible();
  });

  test('identity review cannot approve incomplete evidence', async ({
    page,
  }) => {
    identityReviewDetailMissingSelfie = true;

    await page.goto('/users/identity-review');
    const dialog = await openIdentityEvidence(page, /张三/);

    await expect(
      dialog.getByText(
        /必需材料不完整，无法通过审核|Required evidence is incomplete/,
      ),
    ).toBeVisible();
    await expect(dialog.locator('[data-action="approve"]')).toBeDisabled();
    await expect(dialog.locator('[data-action="reject"]')).toBeEnabled();
  });

  test('identity review buttons submit approve and reject decisions', async ({
    page,
  }) => {
    await page.goto('/users/identity-review');

    const approveDialog = await openIdentityEvidence(page, /张三/);
    await approveDialog.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);
    await expect(page.getByText('实名认证已通过')).toBeVisible();

    const rejectEvidenceDialog = await openIdentityEvidence(page, /李四/);
    await rejectEvidenceDialog
      .getByRole('button', { name: /驳回|Reject/ })
      .click();
    const rejectDialog = page.getByRole('dialog', {
      name: /驳回实名认证|Reject Identity Verification/,
    });
    await rejectDialog
      .getByPlaceholder(/请输入驳回原因|Enter rejection reason/)
      .fill('证件照片与姓名不一致');
    await rejectDialog.getByRole('button', { name: /确定|Confirm/ }).click();
    await expect(page.getByText('实名认证已驳回')).toBeVisible();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/12',
        method: 'PUT',
        body: { approved: true },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/13',
        method: 'PUT',
        body: {
          approved: false,
          rejectionReason: '证件照片与姓名不一致',
        },
      });
  });

  test('identity review action loading stays scoped to the active row', async ({
    page,
  }) => {
    let releaseReview!: () => void;
    nextIdentityReviewActionDelay = new Promise<void>((resolve) => {
      releaseReview = resolve;
    });

    await page.goto('/users/identity-review');

    const activeRow = page.getByRole('row', { name: /张三/ });
    const otherRow = page.getByRole('row', { name: /李四/ });
    await expect(activeRow).toBeVisible();
    await expect(otherRow).toBeVisible();

    const activeDialog = await openIdentityEvidence(page, /张三/);
    const activeApprove = activeDialog.locator('[data-action="approve"]');
    await activeApprove.click();
    await confirmPopconfirm(page);

    await expect(activeApprove).toBeDisabled();
    await expect(activeDialog.locator('[data-action="reject"]')).toBeDisabled();
    await expect(
      otherRow.locator('[data-action="review-evidence"]'),
    ).toBeEnabled();

    releaseReview();
    await expect(page.getByText('实名认证已通过')).toBeVisible();
  });

  test('identity review action failures preserve backend error detail', async ({
    page,
  }) => {
    nextIdentityReviewActionErrorMessage = '实名认证已被其他管理员处理';

    await page.goto('/users/identity-review');

    const approveDialog = await openIdentityEvidence(page, /张三/);
    await approveDialog.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);

    const actionError = page.locator('.admin-load-error', {
      hasText: '实名认证已被其他管理员处理',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/12',
        method: 'PUT',
        body: { approved: true },
      });
  });

  test('identity review reject failures keep the rejection dialog draft', async ({
    page,
  }) => {
    nextIdentityReviewActionErrorMessage = '实名认证驳回失败';

    await page.goto('/users/identity-review');

    const evidenceDialog = await openIdentityEvidence(page, /李四/);
    await evidenceDialog.getByRole('button', { name: /驳回|Reject/ }).click();
    const rejectDialog = page.getByRole('dialog', {
      name: /驳回实名认证|Reject Identity Verification/,
    });
    const reasonInput = rejectDialog.getByPlaceholder(
      /请输入驳回原因|Enter rejection reason/,
    );
    await reasonInput.fill('证件照片与姓名不一致');
    await rejectDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '实名认证驳回失败',
    });
    await expect(actionError).toBeVisible();
    await expect(rejectDialog).toBeVisible();
    await expect(reasonInput).toHaveValue('证件照片与姓名不一致');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/identities/13',
        method: 'PUT',
        body: {
          approved: false,
          rejectionReason: '证件照片与姓名不一致',
        },
      });
  });
});
