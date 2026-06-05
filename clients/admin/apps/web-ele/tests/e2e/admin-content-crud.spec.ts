import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

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

let nextSensitiveWordListErrorMessage: null | string = null;
let nextSensitiveWordActionErrorMessage: null | string = null;
let nextTeacherActionErrorMessage: null | string = null;
let nextTeacherDeleteErrorMessage: null | string = null;
let nextTeacherListErrorMessage: null | string = null;

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

async function fulfillNextSensitiveWordActionError(route: Route) {
  if (!nextSensitiveWordActionErrorMessage) {
    return false;
  }
  const message = nextSensitiveWordActionErrorMessage;
  nextSensitiveWordActionErrorMessage = null;
  await route.fulfill(
    json({
      success: false,
      error: {
        code: 'E2E_SENSITIVE_WORD_ACTION_FAILED',
        message,
      },
    }),
  );
  return true;
}

async function fulfillNextTeacherActionError(route: Route) {
  if (!nextTeacherActionErrorMessage) {
    return false;
  }
  const message = nextTeacherActionErrorMessage;
  nextTeacherActionErrorMessage = null;
  await route.fulfill(
    json({
      success: false,
      error: {
        code: 'E2E_TEACHER_ACTION_FAILED',
        message,
      },
    }),
  );
  return true;
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
        if (nextTeacherListErrorMessage) {
          const message = nextTeacherListErrorMessage;
          nextTeacherListErrorMessage = null;
          await route.fulfill(
            json({
              success: false,
              error: {
                code: 'E2E_TEACHER_LIST_UNAVAILABLE',
                message,
              },
            }),
          );
          return;
        }
        await route.fulfill(list([teacher]));
        return;
      }
      if (await fulfillNextTeacherActionError(route)) {
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...teacher, id: 8, name: '周老师' }));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/teachers/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      if (method === 'DELETE' && nextTeacherDeleteErrorMessage) {
        const message = nextTeacherDeleteErrorMessage;
        nextTeacherDeleteErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_TEACHER_DELETE_FAILED',
              message,
            },
          }),
        );
        return;
      }
      if (await fulfillNextTeacherActionError(route)) {
        return;
      }
      await route.fulfill(ok({ ...teacher, name: '李教授更新' }));
      return;
    }

    if (path === '/api/v1/course/review/admin/sensitive-words') {
      if (method === 'GET') {
        if (nextSensitiveWordListErrorMessage) {
          const message = nextSensitiveWordListErrorMessage;
          nextSensitiveWordListErrorMessage = null;
          await route.fulfill(
            json({
              success: false,
              error: {
                code: 'E2E_SENSITIVE_WORD_LIST_UNAVAILABLE',
                message,
              },
            }),
          );
          return;
        }
        await route.fulfill(list([sensitiveWord]));
        return;
      }
      if (await fulfillNextSensitiveWordActionError(route)) {
        return;
      }
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...sensitiveWord, id: 'word-2', word: '新词' }));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/sensitive-words/')) {
      if (await fulfillNextSensitiveWordActionError(route)) {
        return;
      }
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
    nextSensitiveWordActionErrorMessage = null;
    nextSensitiveWordListErrorMessage = null;
    nextTeacherActionErrorMessage = null;
    nextTeacherDeleteErrorMessage = null;
    nextTeacherListErrorMessage = null;
    await mockAdminApi(page, capturedMutations);
  });

  test('teacher list failures show a persistent retry path', async ({
    page,
  }) => {
    nextTeacherListErrorMessage = '教师列表暂时不可用';

    await page.goto('/content/teachers');

    const loadError = page.locator('.admin-load-error', {
      hasText: '教师列表暂时不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('李教授')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('李教授')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('teacher action failures remain visible without losing table data', async ({
    page,
  }) => {
    await page.goto('/content/teachers');
    await expect(page.getByText('李教授')).toBeVisible();

    nextTeacherDeleteErrorMessage = '教师删除失败，请稍后重试';
    const teacherRow = page.getByRole('row', { name: /李教授/ });
    await teacherRow.getByRole('button', { name: /删除|Delete/ }).click();
    await confirmPopconfirm(page);

    const actionError = page.locator('.admin-load-error', {
      hasText: '教师删除失败，请稍后重试',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('李教授')).toBeVisible();
  });

  test('teacher create failures keep the dialog draft visible', async ({
    page,
  }) => {
    await page.goto('/content/teachers');
    await expect(page.getByText('李教授')).toBeVisible();

    nextTeacherActionErrorMessage = '教师保存失败，请稍后重试';
    await page.getByRole('button', { name: /新增教师|Create/ }).click();

    const createDialog = page.getByRole('dialog', { name: /新增教师|Create/ });
    await createDialog.getByPlaceholder('请输入教师姓名').fill('保存失败教师');
    await createDialog.getByPlaceholder('院系 ID').fill('2');
    await createDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '教师保存失败，请稍后重试',
    });
    await expect(actionError).toBeVisible();
    await expect(createDialog).toBeVisible();
    await expect(createDialog.getByPlaceholder('请输入教师姓名')).toHaveValue(
      '保存失败教师',
    );
    await expect(page.getByText('李教授')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('teacher edit failures keep the dialog draft visible', async ({
    page,
  }) => {
    await page.goto('/content/teachers');
    const teacherRow = page.getByRole('row', { name: /李教授/ });
    await expect(teacherRow).toBeVisible();

    nextTeacherActionErrorMessage = '教师更新失败，请稍后重试';
    await teacherRow.getByRole('button', { name: /编辑|Edit/ }).click();

    const editDialog = page.getByRole('dialog', { name: /编辑教师|Edit/ });
    await editDialog.getByPlaceholder('请输入教师姓名').fill('李教授失败草稿');
    await editDialog.getByPlaceholder('院系 ID').fill('3');
    await editDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '教师更新失败，请稍后重试',
    });
    await expect(actionError).toBeVisible();
    await expect(editDialog).toBeVisible();
    await expect(editDialog.getByPlaceholder('请输入教师姓名')).toHaveValue(
      '李教授失败草稿',
    );
    await expect(teacherRow).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
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

  test('sensitive word list failures show a persistent retry path', async ({
    page,
  }) => {
    nextSensitiveWordListErrorMessage = '敏感词列表暂时不可用';

    await page.goto('/content/sensitive-words');

    const loadError = page.locator('.admin-load-error', {
      hasText: '敏感词列表暂时不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('违规词')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('违规词')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('sensitive word action failures remain visible without losing table data', async ({
    page,
  }) => {
    await page.goto('/content/sensitive-words');
    await expect(page.getByText('违规词')).toBeVisible();

    nextSensitiveWordActionErrorMessage = '敏感词删除失败，请稍后重试';
    const wordRow = page.getByRole('row', { name: /违规词/ });
    await wordRow.getByRole('button', { name: /删除|Delete/ }).click();
    await confirmPopconfirm(page);

    const actionError = page.locator('.admin-load-error', {
      hasText: '敏感词删除失败，请稍后重试',
    });
    await expect(actionError).toBeVisible();
    await expect(page.getByText('违规词')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('sensitive word create failures keep the dialog draft visible', async ({
    page,
  }) => {
    await page.goto('/content/sensitive-words');
    await expect(page.getByText('违规词')).toBeVisible();

    nextSensitiveWordActionErrorMessage = '敏感词保存失败，请稍后重试';
    await page.getByRole('button', { name: /新增敏感词|Create/ }).click();

    const createDialog = page.getByRole('dialog', {
      name: /新增敏感词|Create/,
    });
    await createDialog.getByPlaceholder('请输入敏感词').fill('失败新词');
    await createDialog.getByPlaceholder('请输入分类').fill('comment');
    await createDialog.locator('.el-select').click();
    await page.getByRole('option', { name: /警告|Warn/ }).click();
    await createDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '敏感词保存失败，请稍后重试',
    });
    await expect(actionError).toBeVisible();
    await expect(createDialog).toBeVisible();
    await expect(createDialog.getByPlaceholder('请输入敏感词')).toHaveValue(
      '失败新词',
    );
    await expect(page.getByText('违规词')).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('sensitive word edit failures keep the dialog draft visible', async ({
    page,
  }) => {
    await page.goto('/content/sensitive-words');
    const wordRow = page.getByRole('row', { name: /违规词/ });
    await expect(wordRow).toBeVisible();

    nextSensitiveWordActionErrorMessage = '敏感词更新失败，请稍后重试';
    await wordRow.getByRole('button', { name: /编辑|Edit/ }).click();

    const editDialog = page.getByRole('dialog', { name: /编辑敏感词|Edit/ });
    await editDialog.getByPlaceholder('请输入敏感词').fill('违规词失败草稿');
    await editDialog.getByPlaceholder('请输入分类').fill('review');
    await editDialog.locator('.el-select').click();
    await page.getByRole('option', { name: /复核|Review/ }).click();
    await editDialog.getByRole('button', { name: /确定|Confirm/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '敏感词更新失败，请稍后重试',
    });
    await expect(actionError).toBeVisible();
    await expect(editDialog).toBeVisible();
    await expect(editDialog.getByPlaceholder('请输入敏感词')).toHaveValue(
      '违规词失败草稿',
    );
    await expect(wordRow).toBeVisible();
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
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
