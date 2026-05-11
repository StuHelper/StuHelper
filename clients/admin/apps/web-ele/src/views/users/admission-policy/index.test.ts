import { readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const sourcePath = resolve(
  process.cwd(),
  'src/views/users/admission-policy/index.vue',
);

const memberBlacklistDir = resolve(
  process.cwd(),
  'src/views/users/member-blacklist',
);

async function readAllMemberBlacklistSources(): Promise<string> {
  const entries = await readdir(memberBlacklistDir);
  const targets = entries.filter(
    (name) => name.endsWith('.vue') || name.endsWith('.ts'),
  );
  const sources = await Promise.all(
    targets.map((name) => readFile(resolve(memberBlacklistDir, name), 'utf8')),
  );
  return sources.join('\n');
}

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

  it('normalizes missing management guild arrays before rendering', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).toContain('function normalizeManagementGuildIDs');
    expect(source).toContain('Array.isArray(values)');
    expect(source).toContain('await listAdmissionPolicies()).map(normalizePolicy)');
  });

  it('renders operator-facing Chinese labels instead of raw API field names', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const label of [
      '启用新生入群通道',
      '新生通道关闭时间',
      '默认临时认证到期时间',
      '入群初始禁言（秒）',
      '管理群号',
      '转发原始材料到 QQ',
    ]) {
      expect(source).toContain(label);
    }

    expect(source).toContain('policyFieldLabels.freshmanChannelEnabled');
    expect(source).not.toContain('label="freshmanChannelEnabled"');
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
    const source = await readAllMemberBlacklistSources();

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
      ':teleported="false"',
      '@change="emit(\'search\')"',
    ]) {
      expect(source).toContain(token);
    }
  });
});
