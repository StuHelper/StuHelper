import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-05-24T04:00:00Z';

const capabilities = ['admin:reviews:manage', 'admin:reports:manage'];

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
});
