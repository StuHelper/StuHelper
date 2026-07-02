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
      'extensionDaysById[row.id]',
      "type FreshmanReviewAction = 'approve' | 'approveWithDays' | 'reject'",
      'successMessage,',
      ':disabled="isActionPending(row.id)"',
      ':loading="rowActionLoading(row, \'approve\')"',
      ':loading="rowActionLoading(row, \'approveWithDays\')"',
      ':loading="rowActionLoading(row, \'reject\')"',
    ]) {
      expect(source).toContain(token);
    }

    expect(source).not.toContain('const actionLoading = ref(false)');
    expect(source).not.toContain(':loading="actionLoading"');
  });

  it('requires confirmation before approvals and gates days on positive integers', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).toContain('ElPopconfirm');
    expect(source).toContain(
      "$t('admin.users.freshmanVerification.confirmApprove')",
    );
    expect(source).toContain(
      "$t('admin.users.freshmanVerification.confirmApproveWithDays'",
    );
    expect(source).toContain('@confirm="approve(row)"');
    expect(source).toContain('@confirm="approveWithDays(row)"');
    // 天数 0/非法时禁用带天数通过，而不是静默退化为普通通过
    expect(source).toContain('rowExtensionDays(row) === undefined');
    expect(source).toContain('Number.isInteger(days) && days > 0');
  });

  it('keeps all user-facing copy in i18n', async () => {
    const source = await readFile(sourcePath, 'utf8');
    const withoutComments = source
      .split('\n')
      .filter((line) => !/^\s*(?:\/\/|\*|\/\*)/.test(line))
      .join('\n');

    expect(withoutComments).not.toMatch(/[一-鿿]/);
  });

  it('uses the generated FreshmanApplication type without a local row alias', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).toContain(
      "import type { FreshmanApplication } from '#/api/admin'",
    );
    expect(source).not.toContain('FreshmanReviewRow');
  });
});
