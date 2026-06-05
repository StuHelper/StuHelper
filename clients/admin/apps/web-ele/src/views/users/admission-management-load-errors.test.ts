import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const viewPaths = [
  'src/views/users/admission-sessions/index.vue',
  'src/views/users/admission-policy/index.vue',
  'src/views/users/member-blacklist/index.vue',
];

describe('admission management views error contract', () => {
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

  it('keeps mutation failures visible after toast dismissal', async () => {
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
});
