import type {
  ListMemberBlacklistParams,
  MemberBlacklistEntry,
} from '#/api/admin';

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

export const SOURCE_LABELS: Record<string, string> = {
  admission_failure: '认证失败',
  kick_blacklist: '踢出拉黑',
  manual_admin: '管理员手动',
  migration_admission_failure: '迁移·认证失败',
  migration_legacy_koishi: '迁移·Koishi 旧库',
  moderation_action: '审核处置',
};

export const CREATED_FROM_LABELS: Record<string, string> = {
  admin_console: 'Admin 后台',
  admission_worker: 'Admission Worker',
  koishi_console: 'Koishi 控制台',
  migration_script: '迁移脚本',
  moderation_review: '审核流程',
  qq_command: 'QQ 命令',
};

export const RELEASE_REASON_OPTIONS: Array<{
  label: string;
  value: ReleaseReasonCode;
}> = [
  { label: '宽恕（重置 admission 失败计数）', value: 'manual_pardon' },
  { label: '仅解除（保留失败计数）', value: 'release_only' },
  { label: '申诉通过', value: 'admission_appeal_passed' },
];

export const STATUS_OPTIONS: Array<{ label: string; value: StatusFilter }> = [
  { label: '生效中', value: 'active' },
  { label: '已解除', value: 'released' },
  { label: '已过期', value: 'expired' },
  { label: '全部', value: 'all' },
];

export const SCOPE_OPTIONS: Array<{ label: string; value: '' | ScopeType }> = [
  { label: '全部范围', value: '' },
  { label: '单群', value: 'guild' },
  { label: '全局', value: 'global' },
];

export const SOURCE_OPTIONS: Array<{ label: string; value: SourceFilter }> = [
  { label: '全部来源', value: '' },
  { label: '管理员手动', value: 'manual_admin' },
  { label: 'QQ 踢出', value: 'kick_blacklist' },
  { label: '审核处置', value: 'moderation_action' },
  { label: '认证失败', value: 'admission_failure' },
];

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
  if (status === 'active') return '生效中';
  if (status === 'expired') return '已过期';
  return '已解除';
}

export function scopeLabel(entry: MemberBlacklistEntry): string {
  return entry.scopeType === 'global' ? '全局' : `群 ${entry.guildID ?? '—'}`;
}

export function sourceLabel(entry: MemberBlacklistEntry): string {
  return SOURCE_LABELS[entry.source] ?? entry.source;
}

export function createdFromLabel(entry: MemberBlacklistEntry): string {
  return CREATED_FROM_LABELS[entry.createdFrom] ?? entry.createdFrom;
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
