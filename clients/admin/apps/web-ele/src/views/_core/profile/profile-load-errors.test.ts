import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

describe('profile base setting load error contract', () => {
  it('keeps profile loading failures visible, retryable, and scoped to the latest request', async () => {
    const source = await readFile(
      resolve(process.cwd(), 'src/views/_core/profile/base-setting.vue'),
      'utf8',
    );

    expect(source).toContain('ElAlert');
    expect(source).toContain('ElButton');
    expect(source).toContain("const loadError = ref('')");
    expect(source).toContain('let fetchRequestSeq = 0;');
    expect(source).toContain('const requestSeq = ++fetchRequestSeq;');
    expect(source).toContain("loadError.value = ''");
    expect(source).toContain('if (requestSeq !== fetchRequestSeq) return;');
    expect(source).toContain('loadError.value = adminErrorMessage(error)');
    expect(source).toContain('if (requestSeq === fetchRequestSeq) {');
    expect(source).toContain('v-if="loadError"');
    expect(source).toContain(':title="loadError"');
    expect(source).toContain('@click="fetchProfile"');
    expect(source).toContain("$t('admin.common.retry')");
    expect(source).toContain('function adminErrorMessage(error: unknown)');
    expect(source).toContain('onMounted(fetchProfile)');
  });
});
