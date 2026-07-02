import type {
  ListMemberBlacklistParams,
  MemberBlacklistEntry,
} from '#/api/admin';

import { $t } from '#/locales';

import { formatAdminDateTime } from '../../shared/display';

export type ScopeType = 'global' | 'guild';
export type StatusFilter = 'active' | 'all' | 'expired' | 'released';
export type SourceFilter =
  | ''
  | NonNullable<NonNullable<ListMemberBlacklistParams>['source']>;
export type ReleaseReasonCode =
  | 'admission_appeal_passed'
  | 'manual_pardon'
  | 'release_only';
export type DisplayStatus = 'active' | 'expired' | 'released';

const SOURCE_LABEL_KEYS: Record<string, string> = {
  admission_failure: 'admin.users.memberBlacklist.source.admissionFailure',
  kick_blacklist: 'admin.users.memberBlacklist.source.kickBlacklist',
  manual_admin: 'admin.users.memberBlacklist.source.manualAdmin',
  migration_admission_failure:
    'admin.users.memberBlacklist.source.migrationAdmissionFailure',
  migration_legacy_koishi:
    'admin.users.memberBlacklist.source.migrationLegacyKoishi',
  moderation_action: 'admin.users.memberBlacklist.source.moderationAction',
};

const CREATED_FROM_LABEL_KEYS: Record<string, string> = {
  admin_console: 'admin.users.memberBlacklist.createdFrom.adminConsole',
  admission_worker: 'admin.users.memberBlacklist.createdFrom.admissionWorker',
  koishi_console: 'admin.users.memberBlacklist.createdFrom.koishiConsole',
  migration_script: 'admin.users.memberBlacklist.createdFrom.migrationScript',
  moderation_review: 'admin.users.memberBlacklist.createdFrom.moderationReview',
  qq_command: 'admin.users.memberBlacklist.createdFrom.qqCommand',
};

/** 惰性求值：语言包异步加载，模块级常量会把 key 固化成首屏语言。 */
export function releaseReasonOptions(): Array<{
  label: string;
  value: ReleaseReasonCode;
}> {
  return [
    {
      label: $t('admin.users.memberBlacklist.releaseReason.manualPardon'),
      value: 'manual_pardon',
    },
    {
      label: $t('admin.users.memberBlacklist.releaseReason.releaseOnly'),
      value: 'release_only',
    },
    {
      label: $t('admin.users.memberBlacklist.releaseReason.appealPassed'),
      value: 'admission_appeal_passed',
    },
  ];
}

export function statusOptions(): Array<{
  label: string;
  value: StatusFilter;
}> {
  return [
    { label: $t('admin.users.memberBlacklist.status.active'), value: 'active' },
    {
      label: $t('admin.users.memberBlacklist.status.released'),
      value: 'released',
    },
    {
      label: $t('admin.users.memberBlacklist.status.expired'),
      value: 'expired',
    },
    { label: $t('admin.users.memberBlacklist.status.all'), value: 'all' },
  ];
}

export function scopeOptions(): Array<{
  label: string;
  value: '' | ScopeType;
}> {
  return [
    { label: $t('admin.users.memberBlacklist.scopeFilter.all'), value: '' },
    {
      label: $t('admin.users.memberBlacklist.scopeFilter.guild'),
      value: 'guild',
    },
    {
      label: $t('admin.users.memberBlacklist.scopeFilter.global'),
      value: 'global',
    },
  ];
}

export function sourceOptions(): Array<{
  label: string;
  value: SourceFilter;
}> {
  return [
    { label: $t('admin.users.memberBlacklist.sourceFilter.all'), value: '' },
    {
      label: $t('admin.users.memberBlacklist.sourceFilter.manualAdmin'),
      value: 'manual_admin',
    },
    {
      label: $t('admin.users.memberBlacklist.sourceFilter.kickBlacklist'),
      value: 'kick_blacklist',
    },
    {
      label: $t('admin.users.memberBlacklist.sourceFilter.moderationAction'),
      value: 'moderation_action',
    },
    {
      label: $t('admin.users.memberBlacklist.sourceFilter.admissionFailure'),
      value: 'admission_failure',
    },
  ];
}

export function entryStatus(entry: MemberBlacklistEntry): DisplayStatus {
  if (entry.releasedAt) return 'released';
  if (entry.expiresAt && Date.parse(entry.expiresAt) <= Date.now()) {
    return 'expired';
  }
  return 'active';
}

export function statusType(
  status: DisplayStatus,
): 'danger' | 'info' | 'success' {
  if (status === 'active') return 'danger';
  if (status === 'expired') return 'info';
  return 'success';
}

export function statusLabel(status: DisplayStatus): string {
  return $t(`admin.users.memberBlacklist.status.${status}`);
}

export function scopeLabel(entry: MemberBlacklistEntry): string {
  return entry.scopeType === 'global'
    ? $t('admin.users.memberBlacklist.scopeGlobal')
    : $t('admin.users.memberBlacklist.scopeGuildEntry', {
        guild: entry.guildID ?? '—',
      });
}

export function sourceLabel(entry: MemberBlacklistEntry): string {
  const key = SOURCE_LABEL_KEYS[entry.source];
  return key ? $t(key) : entry.source;
}

export function createdFromLabel(entry: MemberBlacklistEntry): string {
  const key = CREATED_FROM_LABEL_KEYS[entry.createdFrom];
  return key ? $t(key) : entry.createdFrom;
}

export function createdByLabel(entry: MemberBlacklistEntry): string {
  return `${entry.createdByType} · ${entry.createdByID}`;
}

export function formatDateTime(value?: null | string): string {
  return formatAdminDateTime(value);
}

export function toIsoString(value: Date | string): string | undefined {
  if (!value) return undefined;
  if (value instanceof Date) return value.toISOString();
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return undefined;
  return parsed.toISOString();
}
