import { readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

const sourcePath = resolve(
  process.cwd(),
  'src/views/users/admission-policy/index.vue',
);

const zhLocalePath = resolve(
  process.cwd(),
  'src/locales/langs/zh-CN/admin.json',
);

const enLocalePath = resolve(
  process.cwd(),
  'src/locales/langs/en-US/admin.json',
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

async function readAdmissionPolicyLocale(path: string) {
  const data = JSON.parse(await readFile(path, 'utf8')) as {
    users: { admissionPolicy: Record<string, unknown> };
  };
  return data.users.admissionPolicy;
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
      'successMessage:',
      ':loading="isActionPending(policy.id)"',
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

  it('sources operator-facing labels from i18n instead of raw API field names', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).toContain('policyFieldLabels.freshmanChannelEnabled');
    expect(source).toContain("$t('admin.users.admissionPolicy.fields.");
    expect(source).toContain("$t('admin.users.admissionPolicy.authorityNote')");
    expect(source).not.toContain('label="freshmanChannelEnabled"');

    const zh = await readAdmissionPolicyLocale(zhLocalePath);
    const zhFields = zh.fields as Record<string, string>;
    expect(zhFields.freshmanChannelEnabled).toBe('启用新生入群通道');
    expect(zhFields.guardEnabled).toBe('启用入群认证守卫');
    expect(zhFields.initialMuteDurationSeconds).toBe('入群初始禁言（秒）');
    expect(zhFields.managementGuildIDs).toBe('材料审核通知群号');
    expect(zhFields.forwardRawMaterialToQQ).toBe('转发原始材料到 QQ');
    expect(zh.authorityNote).toContain('Admin 是入群认证策略权威源');
    expect(zh.sectionMaterialNote).toContain('审核通知群只接收材料审核提醒');
  });

  it('keeps zh-CN and en-US admissionPolicy locales in key parity', async () => {
    const zh = await readAdmissionPolicyLocale(zhLocalePath);
    const en = await readAdmissionPolicyLocale(enLocalePath);

    expect(Object.keys(en).toSorted()).toEqual(Object.keys(zh).toSorted());
    expect(Object.keys(en.fields as object).toSorted()).toEqual(
      Object.keys(zh.fields as object).toSorted(),
    );
  });

  it('keeps user-facing copy out of the component source', async () => {
    const source = await readFile(sourcePath, 'utf8');
    const withoutComments = source
      .split('\n')
      .filter((line) => !/^\s*(?:\/\/|\*|\/\*)/.test(line))
      .join('\n');

    expect(withoutComments).not.toMatch(/[一-鿿]/);
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
      'joinHandlingStrategyHelp',
      "$t('admin.users.admissionPolicy.saveImpact'",
    ]) {
      expect(source).toContain(token);
    }
  });

  it('reports per-guild failures and retries only failed guilds (F029)', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).toContain('adminErrorMessage(error)');
    expect(source).toContain(
      "$t('admin.users.admissionPolicy.createPartialFailure'",
    );
    // 失败群号回填输入框，重试不会重复提交已创建的群
    expect(source).toContain('createPolicyForm.guildIDs = failures');
    expect(source).toContain(
      'if (succeeded || failures.length < guildIDs.length)',
    );
  });

  it('never writes client-side defaults back to the backend (F030)', async () => {
    const source = await readFile(sourcePath, 'utf8');

    expect(source).not.toContain('请先完成 StuHelper 学生认证后再申请入群');
    expect(source).toContain(
      "unverifiedJoinRejectReason: policy.unverifiedJoinRejectReason ?? ''",
    );
    expect(source).toContain(
      "$t('admin.users.admissionPolicy.unverifiedRejectPlaceholder')",
    );
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
      "$t('admin.users.memberBlacklist.createdFromColumn')",
      "$t('admin.users.memberBlacklist.createdByColumn')",
      "$t('admin.users.memberBlacklist.expiresAtColumn')",
      "$t('admin.users.memberBlacklist.releasedAtColumn')",
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
