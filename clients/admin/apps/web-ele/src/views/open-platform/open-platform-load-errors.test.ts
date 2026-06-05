import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const openPlatformViewPaths = [
  'src/views/open-platform/apps/index.vue',
  'src/views/open-platform/audit-events/index.vue',
  'src/views/open-platform/consents/index.vue',
  'src/views/open-platform/token-probe-evidence/index.vue',
  'src/views/open-platform/disclosure-report/index.vue',
];

const dashboardViewPaths = [
  'src/views/dashboard/analytics/index.vue',
  'src/views/dashboard/workspace/index.vue',
];

describe('open platform and dashboard load error contract', () => {
  it('keeps open platform loads visible, retryable, and scoped to the latest request', async () => {
    for (const viewPath of openPlatformViewPaths) {
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
      expect(source).toContain("$t('admin.common.retry')");
      expect(source).toContain('function adminErrorMessage(error: unknown)');
    }
  });

  it('keeps dashboard stat loads scoped to the latest request', async () => {
    for (const viewPath of dashboardViewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      expect(source).toContain('ElAlert');
      expect(source).toContain("const loadError = ref('')");
      expect(source).toContain('let fetchRequestSeq = 0;');
      expect(source).toContain('const requestSeq = ++fetchRequestSeq;');
      expect(source).toContain('const data = await getAdminStats()');
      expect(source).toContain('if (requestSeq !== fetchRequestSeq) return;');
      expect(source).toContain('stats.value = data');
      expect(source).toContain('if (requestSeq === fetchRequestSeq) {');
      expect(source).toContain('@click="fetchStats"');
      expect(source).toContain("$t('admin.common.retry')");
    }
  });

  it('invalidates resource grant dialog loads when the dialog closes', async () => {
    const source = await readFile(
      resolve(
        process.cwd(),
        'src/views/open-platform/apps/ResourceGrantsDialog.vue',
      ),
      'utf8',
    );

    expect(source).toContain("const loadError = ref('')");
    expect(source).toContain('let fetchRequestSeq = 0;');
    expect(source).toContain('fetchRequestSeq += 1;');
    expect(source).toContain('const requestSeq = ++fetchRequestSeq;');
    expect(source).toContain('if (requestSeq !== fetchRequestSeq) return;');
    expect(source).toContain('if (requestSeq === fetchRequestSeq) {');
    expect(source).toContain('grants.value = result.grants');
    expect(source).toContain('loadError.value = adminErrorMessage(error)');
    expect(source).toContain('@click="fetchGrants"');
    expect(source).toContain(':title="loadError"');
  });

  it('resets app list pagination before applying status filters', async () => {
    const source = await readFile(
      resolve(process.cwd(), 'src/views/open-platform/apps/index.vue'),
      'utf8',
    );

    expect(source).toContain('function resetPageAndFetch()');
    expect(source).toContain('query.page = 1;');
    expect(source).toContain('void fetchData();');
    expect(source).toContain('@change="resetPageAndFetch"');
    expect(source).toContain('@click="resetPageAndFetch"');
    expect(source).toContain('@current-change="fetchData"');
    expect(source).not.toContain(
      '<ElButton type="primary" @click="fetchData">',
    );
  });
});
