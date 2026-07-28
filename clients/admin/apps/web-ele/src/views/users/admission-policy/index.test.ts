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
      'guardEnabled',
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
      'createAdmissionPolicy',
      'sourcePolicyID',
      'parseCreateGuildIDs',
      'POLICY_DATETIME_FORMAT',
      ':value-format="POLICY_DATETIME_FORMAT"',
      'ElMessage.success',
      ':loading="savingPolicyIDs[policy.id]"',
    ]) {
      expect(source).toContain(token);
    }
  });

  it('normalizes missing management guild arrays before rendering', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).toContain('function normalizeManagementGuildIDs');
    expect(source).toContain('Array.isArray(values)');
    expect(source).toContain('const data = await listAdmissionPolicies()');
    expect(source).toContain('data.map((policy) => normalizePolicy(policy))');
  });

  it('renders operator-facing Chinese labels instead of raw API field names', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const label of [
      'Admin 是入群认证策略权威源',
      '启用新生入群通道',
      '启用入群认证守卫',
      '新生通道关闭时间',
      '默认临时认证到期时间',
      '入群初始禁言（秒）',
      '材料审核通知群号',
      '目标认证群',
      '新增目标认证群',
      '目标认证群号',
      '转发原始材料到 QQ',
      '审核通知群只接收材料审核提醒',
      '验证码等待（秒）',
      'Koishi 同步后会按该值创建群内验证码挑战的超时踢出期限',
      'Koishi 会在下次同步后显示执行态',
    ]) {
      expect(source).toContain(label);
    }

    expect(source).toContain('policyFieldLabels.freshmanChannelEnabled');
    expect(source).not.toContain('label="freshmanChannelEnabled"');
  });

  it('keeps Admin as policy authority and exposes save impact summary', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const token of [
      'data-policy-sync-source="admin"',
      'data-policy-summary',
      'data-policy-save-impact',
      'function guardSyncLabel',
      'function saveImpactSummary',
      'function managementGuildCount',
      'function usesStudentVerificationFlow',
      'function linkWaitLabel',
      'joinHandlingStrategyHelp',
      'Admin 是入群认证策略权威源',
      'Koishi WebUI 只显示同步后的执行态和现场队列',
    ]) {
      expect(source).toContain(token);
    }
  });

  it('keeps post-join time-code controls scoped to applicable fields', async () => {
    const source = await readFile(sourcePath, 'utf8');

    for (const token of [
      'isPostJoinTimeCodeStrategy(policy)',
      '验证码等待：{{ policy.linkWaitSeconds }} 秒',
      'v-if="usesStudentVerificationFlow(policy)"',
      ':label="linkWaitLabel(policy)"',
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
