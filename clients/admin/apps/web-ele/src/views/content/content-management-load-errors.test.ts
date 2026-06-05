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
  it('keeps list loading failures visible, retryable, and scoped to the latest request', async () => {
    for (const viewPath of listViewPaths) {
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

  it('keeps mutation failures visible after toast dismissal', async () => {
    for (const viewPath of mutableViewPaths) {
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

  it('resets pagination before applying content filters', async () => {
    for (const viewPath of mutableViewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('function resetPageAndFetch()');
      expect(source).toContain('query.page = 1;');
      expect(source).toContain('void fetchData();');
      expect(source).toContain('@click="resetPageAndFetch"');
      expect(source).toContain('@current-change="fetchData"');
    }

    for (const viewPath of [
      'src/views/content/reviews/index.vue',
      'src/views/content/reports/index.vue',
      'src/views/content/sensitive-words/index.vue',
    ]) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('@change="resetPageAndFetch"');
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
