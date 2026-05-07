import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const sourcePath = resolve(
  process.cwd(),
  'src/views/users/freshman-verification/index.vue',
);

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
