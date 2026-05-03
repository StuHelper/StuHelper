import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

const sourcePath = fileURLToPath(new URL('./index.vue', import.meta.url));

describe('freshman verification admin view contract', () => {
  it('covers review list fields, material preview, and approval actions', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const token of [
      'data-field="status"',
      'data-field="schoolID"',
      'data-field="qqID"',
      'data-field="createdAt"',
      'data-field="failureCount"',
      'data-action="approve"',
      'data-action="approveWithDays"',
      'data-action="reject"',
      'data-state="pendingReview"',
      'data-material-preview',
    ]) {
      expect(source).toContain(token);
    }
  });
});
