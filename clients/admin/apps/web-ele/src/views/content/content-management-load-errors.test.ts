import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const listViewPaths = [
  'src/views/content/operation-logs/index.vue',
  'src/views/content/reviews/index.vue',
  'src/views/content/reports/index.vue',
  'src/views/content/teachers/index.vue',
  'src/views/content/sensitive-words/index.vue',
];

const mutableViewPaths = [
  'src/views/content/reviews/index.vue',
  'src/views/content/reports/index.vue',
  'src/views/content/teachers/index.vue',
  'src/views/content/sensitive-words/index.vue',
];

describe('content management views error contract', () => {
  it('delegates list loading to useAdminList instead of local boilerplate', async () => {
    for (const viewPath of listViewPaths) {
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
    for (const viewPath of listViewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('ElAlert');
      expect(source).toContain('v-if="loadError"');
      expect(source).toContain(':title="loadError"');
      expect(source).toContain('@click="fetchData"');
      expect(source).toContain("$t('admin.common.retry')");
    }
  });

  it('runs mutations through useAdminAction with row-level locks and single-layer feedback', async () => {
    for (const viewPath of mutableViewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain(
        "import { useAdminAction } from '#/composables/use-admin-action'",
      );
      expect(source).toContain('runAction(');
      expect(source).toContain('isActionPending(row.id)');
      expect(source).toContain('v-if="actionError"');
      expect(source).toContain(':title="actionError"');
      expect(source).toContain('@close="clearActionError"');
      // The composable owns the failure toast; views must not double-toast.
      expect(source).not.toContain('ElMessage.error');
    }
  });

  it('uses the shared sized pagination layout', async () => {
    for (const viewPath of listViewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('ADMIN_PAGINATION_LAYOUT');
      expect(source).toContain('ADMIN_PAGE_SIZES');
      expect(source).toContain('ADMIN_DEFAULT_PAGE_SIZE');
      expect(source).toContain(':layout="ADMIN_PAGINATION_LAYOUT"');
      expect(source).toContain(':page-sizes="ADMIN_PAGE_SIZES"');
      expect(source).toContain('@current-change="fetchData"');
      expect(source).toContain('@size-change="resetPageAndFetch"');
    }
  });

  it('resets pagination before applying content filters', async () => {
    for (const viewPath of [
      'src/views/content/reviews/index.vue',
      'src/views/content/reports/index.vue',
      'src/views/content/sensitive-words/index.vue',
    ]) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('@change="resetPageAndFetch"');
      expect(source).toContain('@click="resetPageAndFetch"');
    }

    for (const viewPath of [
      'src/views/content/teachers/index.vue',
      'src/views/content/sensitive-words/index.vue',
    ]) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('@clear="resetPageAndFetch"');
      expect(source).toContain('@keyup.enter="resetPageAndFetch"');
    }
  });
});
