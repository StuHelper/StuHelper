import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const sourcePath = resolve(
  process.cwd(),
  'src/views/users/admission-policy/index.vue',
);

const memberBlacklistPath = resolve(
  process.cwd(),
  'src/views/users/member-blacklist/index.vue',
);

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
    ]) {
      expect(source).toContain(token);
    }
  });

  it('no longer hosts the member blacklist release form', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const token of [
      'releaseMemberBlacklistBySubject',
      'data-action="releaseBlacklist"',
      'blacklistPlatform',
      'blacklistScope',
      'blacklistGuildID',
    ]) {
      expect(source).not.toContain(token);
    }
  });
});

describe('member blacklist admin view contract', () => {
  it('exposes list, create and release-by-id with required columns', async () => {
    const source = await readFile(memberBlacklistPath, 'utf8');

    for (const token of [
      'listMemberBlacklist',
      'createMemberBlacklist',
      'releaseMemberBlacklist',
      'releaseReasonCode',
      'manual_pardon',
      'release_only',
      'admission_appeal_passed',
      '创建入口',
      '创建人',
      '过期时间',
      '解除时间',
      'data-action="openCreate"',
      'data-action="release"',
      'data-action="submitCreate"',
      'data-action="submitRelease"',
      'data-field="releaseReasonCode"',
    ]) {
      expect(source).toContain(token);
    }
  });
});
