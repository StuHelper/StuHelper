import type { Page, Route } from '@playwright/test';

import { expect, test } from '@playwright/test';

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
      await route.fulfill(list([pendingReview, hiddenReview]));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/reviews/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(ok({ ...pendingReview, status: 'published' }));
      return;
    }

    if (path === '/api/v1/course/review/admin/reports') {
      await route.fulfill(list([pendingReport]));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/reports/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
      await route.fulfill(ok({ ...pendingReport, status: 'rejected' }));
      return;
    }

    if (path === '/api/v1/admin/identities') {
      await route.fulfill(list(identityReviews));
      return;
    }
    if (path.startsWith('/api/v1/admin/identities/')) {
      capturedMutations.push({
        path,
        method,
        body: parseJsonBody(route),
      });
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
    await mockAdminApi(page, capturedMutations);
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

  test('identity review buttons submit approve and reject decisions', async ({
    page,
  }) => {
    await page.goto('/users/identity-review');

    const approveRow = page.getByRole('row', { name: /张三/ });
    await expect(approveRow).toBeVisible();
    await approveRow.getByRole('button', { name: /通过|Approve/ }).click();
    await confirmPopconfirm(page);

    const rejectRow = page.getByRole('row', { name: /李四/ });
    await rejectRow.getByRole('button', { name: /驳回|Reject/ }).click();
    await confirmPopconfirm(page);

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
        body: { approved: false },
      });
  });
});
