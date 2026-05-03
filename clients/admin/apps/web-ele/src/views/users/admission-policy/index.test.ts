import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

const sourcePath = fileURLToPath(new URL('./index.vue', import.meta.url));

describe('admission policy admin view contract', () => {
  it('covers every admission policy control required by the spec', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const token of [
      'freshmanChannelEnabled',
      'freshmanChannelClosesAt',
      'freshmanDefaultExpiresAt',
      'initialMuteDurationSeconds',
      'linkWaitSeconds',
      'submissionWaitSeconds',
      'manualReviewTimeoutSeconds',
      'reminderIntervalSeconds',
      'failedJoinLimit',
      'blacklistDurationSeconds',
      'maxMaterialBytes',
      'maxExtensionDays',
      'managementGuildIDs',
      'forwardRawMaterialToQQ',
      'releaseAdmissionBlacklist',
      'ElPopconfirm',
      'data-action="releaseBlacklist"',
    ]) {
      expect(source).toContain(token);
    }
  });
});
