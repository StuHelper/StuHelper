import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const viewPaths = [
  'src/views/users/identity-review/index.vue',
  'src/views/users/student-verification/index.vue',
  'src/views/users/freshman-verification/index.vue',
];

describe('verification admin views error contract', () => {
  it('keeps list loading failures visible and retryable', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('ElAlert');
      expect(source).toContain("const loadError = ref('')");
      expect(source).toContain('let fetchRequestSeq = 0;');
      expect(source).toContain('const requestSeq = ++fetchRequestSeq;');
      expect(source).toContain("loadError.value = ''");
      expect(source).toContain('if (requestSeq !== fetchRequestSeq) return;');
      expect(source).toContain('loadError.value = adminErrorMessage(error)');
      expect(source).toContain('if (requestSeq === fetchRequestSeq) {');
      expect(source).toContain('v-if="loadError"');
      expect(source).toContain(':title="loadError"');
      expect(source).toContain('@click="fetchData"');
      expect(source).toContain("$t('admin.common.retry')");
      expect(source).toContain('function adminErrorMessage(error: unknown)');
    }
  });

  it('keeps review action failures visible after toast dismissal', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain("const actionError = ref('')");
      expect(source).toContain("actionError.value = ''");
      expect(source).toContain('actionError.value = adminErrorMessage(error)');
      expect(source).toContain('ElMessage.error(actionError.value)');
      expect(source).toContain('v-if="actionError"');
      expect(source).toContain(':title="actionError"');
      expect(source).toContain('@close="actionError = \'\'"');
    }
  });

  it('resets pagination before applying verification filters', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('function resetPageAndFetch()');
      expect(source).toContain('query.page = 1;');
      expect(source).toContain('void fetchData();');
      expect(source).toContain('@change="resetPageAndFetch"');
      expect(source).toContain('@click="resetPageAndFetch"');
      expect(source).toContain('@current-change="fetchData"');
    }

    const studentSource = await readFile(
      resolve(process.cwd(), 'src/views/users/student-verification/index.vue'),
      'utf8',
    );
    expect(studentSource).toContain('@clear="resetPageAndFetch"');
    expect(studentSource).toContain('@keyup.enter="resetPageAndFetch"');
  });

  it('keeps identity review loading scoped to the active evidence target', async () => {
    const source = await readFile(
      resolve(process.cwd(), 'src/views/users/identity-review/index.vue'),
      'utf8',
    );

    for (const token of [
      "type IdentityReviewAction = 'approve' | 'reject'",
      'reviewingActionsByUserId[userId] = action',
      'delete reviewingActionsByUserId[userId]',
      ':disabled="userReviewing(row.userID)"',
      'function detailTargetReviewing()',
      'function detailTargetActionLoading(action: IdentityReviewAction)',
      ':disabled="detailTargetReviewing()"',
      ':loading="detailTargetActionLoading(\'approve\')"',
      ':loading="detailTargetActionLoading(\'reject\')"',
      ':disabled="rejectTargetReviewing()"',
      ':loading="rejectTargetActionLoading(\'reject\')"',
    ]) {
      expect(source).toContain(token);
    }

    expect(source).not.toContain('@confirm="handleReview(row.userID, true)"');
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
      'reviewingActionsByUserId[userId] = action',
      'delete reviewingActionsByUserId[userId]',
      ':disabled="userReviewing(row.userID)"',
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
