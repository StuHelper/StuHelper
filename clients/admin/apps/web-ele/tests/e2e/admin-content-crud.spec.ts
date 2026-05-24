import type { Page, Route } from '@playwright/test';

import { expect, test } from '@playwright/test';

const now = '2026-05-24T04:00:00Z';

const capabilities = ['admin:teachers:manage', 'admin:sensitive_words:manage'];

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

const teacher = {
  id: 7,
  name: '李教授',
  departmentID: 1,
  departmentName: '计算机学院',
  reviewCount: 12,
  createdAt: now,
};

const sensitiveWord = {
  id: 'word-1',
  word: '违规词',
  category: 'review',
  level: 'block',
  isActive: true,
  createdAt: now,
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

    if (path === '/api/v1/course/review/admin/teachers') {
      if (method === 'GET') {
        await route.fulfill(list([teacher]));
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...teacher, id: 8, name: '周老师' }));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/teachers/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...teacher, name: '李教授更新' }));
      return;
    }

    if (path === '/api/v1/course/review/admin/sensitive-words') {
      if (method === 'GET') {
        await route.fulfill(list([sensitiveWord]));
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...sensitiveWord, id: 'word-2', word: '新词' }));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/sensitive-words/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...sensitiveWord, word: '违规词更新' }));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked content CRUD request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin content CRUD actions', () => {
  let capturedMutations: CapturedMutation[];

  test.beforeEach(async ({ page }) => {
    capturedMutations = [];
    await mockAdminApi(page, capturedMutations);
  });

  test('teacher management creates, updates, and deletes teachers', async ({
    page,
  }) => {
    await page.goto('/content/teachers');

    await expect(page.getByText('李教授')).toBeVisible();
    await page.getByRole('button', { name: /新增教师|Create/ }).click();

    const createDialog = page.getByRole('dialog', { name: /新增教师|Create/ });
    await createDialog.getByPlaceholder('请输入教师姓名').fill('周老师');
    await createDialog.getByPlaceholder('院系 ID').fill('2');
    await createDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const teacherRow = page.getByRole('row', { name: /李教授/ });
    await teacherRow.getByRole('button', { name: /编辑|Edit/ }).click();

    const editDialog = page.getByRole('dialog', { name: /编辑教师|Edit/ });
    await editDialog.getByPlaceholder('请输入教师姓名').fill('李教授更新');
    await editDialog.getByPlaceholder('院系 ID').fill('3');
    await editDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    await teacherRow.getByRole('button', { name: /删除|Delete/ }).click();
    await confirmPopconfirm(page);

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/teachers',
        method: 'POST',
        body: { departmentID: 2, name: '周老师' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/teachers/7',
        method: 'PUT',
        body: { departmentID: 3, name: '李教授更新' },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/teachers/7',
        method: 'DELETE',
        body: null,
      });
  });

  test('sensitive word management creates, updates, and deletes entries', async ({
    page,
  }) => {
    await page.goto('/content/sensitive-words');

    await expect(page.getByText('违规词')).toBeVisible();
    await page.getByRole('button', { name: /新增敏感词|Create/ }).click();

    const createDialog = page.getByRole('dialog', {
      name: /新增敏感词|Create/,
    });
    await createDialog.getByPlaceholder('请输入敏感词').fill('新词');
    await createDialog.getByPlaceholder('请输入分类').fill('comment');
    await createDialog.locator('.el-select').click();
    await page.getByRole('option', { name: /警告|Warn/ }).click();
    await createDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const wordRow = page.getByRole('row', { name: /违规词/ });
    await wordRow.getByRole('button', { name: /编辑|Edit/ }).click();

    const editDialog = page.getByRole('dialog', { name: /编辑敏感词|Edit/ });
    await editDialog.getByPlaceholder('请输入敏感词').fill('违规词更新');
    await editDialog.getByPlaceholder('请输入分类').fill('review');
    await editDialog.locator('.el-select').click();
    await page.getByRole('option', { name: /复核|Review/ }).click();
    await editDialog.locator('.el-switch').click();
    await editDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    await wordRow.getByRole('button', { name: /删除|Delete/ }).click();
    await confirmPopconfirm(page);

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/sensitive-words',
        method: 'POST',
        body: {
          category: 'comment',
          isActive: true,
          level: 'warn',
          word: '新词',
        },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/sensitive-words/word-1',
        method: 'PUT',
        body: {
          category: 'review',
          isActive: false,
          level: 'review',
          word: '违规词更新',
        },
      });
    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/course/review/admin/sensitive-words/word-1',
        method: 'DELETE',
        body: null,
      });
  });
});
