import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const viewPaths = [
  'src/views/users/identity-review/index.vue',
  'src/views/users/student-verification/index.vue',
  'src/views/users/freshman-verification/index.vue',
];

describe('verification admin views error contract', () => {
  it('delegates list loading to useAdminList instead of local boilerplate', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain(
        "import { useAdminList } from '#/composables/use-admin-list'",
      );
      expect(source).toContain('useAdminList<');
      expect(source).toContain('resetPageAndFetch');
      // The race guard and error normalization live in the composable only.
      expect(source).not.toContain('fetchRequestSeq');
      expect(source).not.toContain('function adminErrorMessage');
    }
  });

  it('keeps list loading failures visible and retryable', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('ElAlert');
      expect(source).toContain('v-if="loadError"');
      expect(source).toContain(':title="loadError"');
      expect(source).toContain('@click="fetchData"');
      expect(source).toContain("$t('admin.common.retry')");
    }
  });

  it('runs review mutations through useAdminAction with single-layer feedback', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain(
        "import { useAdminAction } from '#/composables/use-admin-action'",
      );
      expect(source).toContain('runAction(');
      expect(source).toContain('v-if="actionError"');
      expect(source).toContain(':title="actionError"');
      expect(source).toContain('@close="clearActionError"');
      // The composable owns the failure toast; views must not double-toast.
      expect(source).not.toContain('ElMessage.error');
    }
  });

  it('uses the shared sized pagination layout', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain(':layout="ADMIN_PAGINATION_LAYOUT"');
      expect(source).toContain(':page-sizes="ADMIN_PAGE_SIZES"');
      expect(source).toContain('ADMIN_DEFAULT_PAGE_SIZE');
      expect(source).toContain('@current-change="fetchData"');
      expect(source).toContain('@size-change="resetPageAndFetch"');
    }
  });

  it('resets pagination before applying verification filters', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('@change="resetPageAndFetch"');
      expect(source).toContain('@click="resetPageAndFetch"');
    }

    const studentSource = await readFile(
      resolve(process.cwd(), 'src/views/users/student-verification/index.vue'),
      'utf8',
    );
    expect(studentSource).toContain('@clear="resetPageAndFetch"');
    expect(studentSource).toContain('@keyup.enter="resetPageAndFetch"');
  });

  it('keeps identity review loading scoped to the active user', async () => {
    const source = await readFile(
      resolve(process.cwd(), 'src/views/users/identity-review/index.vue'),
      'utf8',
    );

    for (const token of [
      "type IdentityReviewAction = 'approve' | 'reject'",
      ':disabled="isActionPending(row.userID)"',
      ':loading="userActionLoading(row.userID, \'approve\')"',
      ':loading="userActionLoading(row.userID, \'reject\')"',
      ':disabled="rejectTargetReviewing()"',
      ':loading="rejectTargetActionLoading(\'reject\')"',
    ]) {
      expect(source).toContain(token);
    }

    expect(source).not.toContain('const actionLoading = ref(false)');
    expect(source).not.toContain(':loading="actionLoading"');
  });

  it('keeps student verification review loading scoped to the active user', async () => {
    const source = await readFile(
      resolve(process.cwd(), 'src/views/users/student-verification/index.vue'),
      'utf8',
    );

    for (const token of [
      "type StudentReviewAction = 'approve' | 'reject'",
      ':disabled="isActionPending(row.userID)"',
      ':loading="userActionLoading(row.userID, \'approve\')"',
      ':loading="userActionLoading(row.userID, \'reject\')"',
      ':disabled="rejectTargetReviewing()"',
      ':loading="rejectTargetActionLoading(\'reject\')"',
    ]) {
      expect(source).toContain(token);
    }

    expect(source).not.toContain('const actionLoading = ref(false)');
    expect(source).not.toContain(':loading="actionLoading"');
  });
});
