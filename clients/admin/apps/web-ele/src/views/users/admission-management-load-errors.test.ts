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
      expect(source).toContain('v-if="loadError"');
      expect(source).toContain(':title="loadError"');
      expect(source).toContain('@click="fetchData"');
      expect(source).toContain("$t('admin.common.retry')");

      // 加载/错误状态机收口在共享 composable（由 use-admin-list.test.ts 锁定），
      // 视图不得回退到旧的内联复制粘贴实现。
      expect(source).not.toContain('let fetchRequestSeq');
      expect(source).not.toContain('function adminErrorMessage');
    }
  });

  it('keeps mutation failures visible after toast dismissal', async () => {
    for (const viewPath of viewPaths) {
      const source = await readFile(resolve(process.cwd(), viewPath), 'utf8');

      // 失败提示的唯一 toast 层在 useAdminAction（由 use-admin-action.test.ts
      // 锁定），视图保留可驻留的内联 Alert 以便 toast 消失后仍可见。
      expect(source).toContain('useAdminAction');
      expect(source).toContain('v-if="actionError"');
      expect(source).toContain(':title="actionError"');
      expect(source).toContain('@close="clearActionError"');
      expect(source).not.toContain('ElMessage.error(actionError.value)');
    }
  });
});
