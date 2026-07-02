import type {
  AdmissionSession,
  ListAdmissionSessionsParams,
} from '#/api/admin';

import { $t } from '#/locales';

import { formatAdminDateTime, formatNullableText } from '../../shared/display';

export type StatusFilter =
  | ''
  | NonNullable<NonNullable<ListAdmissionSessionsParams>['status']>;

export type AdmissionSessionAction = 'cancel' | 'regenerate' | 'resend';

const STATUS_LABEL_KEYS: Record<string, string> = {
  cancelled: 'admin.users.admissionSessions.status.cancelled',
  expired_kicked: 'admin.users.admissionSessions.status.expiredKicked',
  joined_muted: 'admin.users.admissionSessions.status.joinedMuted',
  linked: 'admin.users.admissionSessions.status.linked',
  material_submitted: 'admin.users.admissionSessions.status.materialSubmitted',
  verified: 'admin.users.admissionSessions.status.verified',
};

const STATUS_HINT_KEYS: Record<string, string> = {
  cancelled: 'admin.users.admissionSessions.hint.cancelled',
  expired_kicked: 'admin.users.admissionSessions.hint.expiredKicked',
  joined_muted: 'admin.users.admissionSessions.hint.joinedMuted',
  linked: 'admin.users.admissionSessions.hint.linked',
  material_submitted: 'admin.users.admissionSessions.hint.materialSubmitted',
  verified: 'admin.users.admissionSessions.hint.verified',
};

/** 惰性求值：语言包异步加载，模块级常量会把 key 固化成首屏语言。 */
export function statusOptions(): Array<{ label: string; value: StatusFilter }> {
  return [
    { label: $t('admin.users.admissionSessions.status.all'), value: '' },
    {
      label: $t('admin.users.admissionSessions.status.joinedMuted'),
      value: 'joined_muted',
    },
    {
      label: $t('admin.users.admissionSessions.status.linked'),
      value: 'linked',
    },
    {
      label: $t('admin.users.admissionSessions.status.materialSubmitted'),
      value: 'material_submitted',
    },
    {
      label: $t('admin.users.admissionSessions.status.verified'),
      value: 'verified',
    },
    {
      label: $t('admin.users.admissionSessions.status.expiredKicked'),
      value: 'expired_kicked',
    },
    {
      label: $t('admin.users.admissionSessions.status.cancelled'),
      value: 'cancelled',
    },
  ];
}

export function statusLabel(status: AdmissionSession['status']): string {
  const key = STATUS_LABEL_KEYS[status];
  return key ? $t(key) : status;
}

export function statusOperationHint(
  status: AdmissionSession['status'],
): string {
  return $t(
    STATUS_HINT_KEYS[status] ?? 'admin.users.admissionSessions.hint.unknown',
  );
}

export function statusTagType(
  status: AdmissionSession['status'],
): 'danger' | 'info' | 'primary' | 'success' | 'warning' {
  if (status === 'verified') return 'success';
  if (status === 'expired_kicked' || status === 'cancelled') return 'info';
  if (status === 'material_submitted') return 'warning';
  if (status === 'linked') return 'primary';
  return 'danger';
}

export function formatDateTime(value?: null | string): string {
  return formatAdminDateTime(value);
}

export function formatText(value?: null | string): string {
  return formatNullableText(value);
}

export function boolLabel(value: boolean): string {
  return $t(value ? 'admin.common.yes' : 'admin.common.no');
}

export function admissionReissueCommand(session: AdmissionSession): string {
  // 这是发给 Koishi 机器人的命令字符串，命令字由 bot 侧 runtime settings
  // 定义为中文，不随管理界面语言变化，因此不走 i18n。
  return `重新生成认证链接 ${session.qqID}`;
}

export function botErrorLabel(session: AdmissionSession): string {
  return $t(
    session.lastBotError
      ? 'admin.users.admissionSessions.botErrorPresent'
      : 'admin.users.admissionSessions.botErrorNone',
  );
}

export function canManageAdmissionSession(
  status: AdmissionSession['status'],
): boolean {
  return !['cancelled', 'expired_kicked', 'verified'].includes(status);
}
