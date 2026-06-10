import type {
  AdmissionSession,
  ListAdmissionSessionsParams,
} from '#/api/admin';

import { formatAdminDateTime, formatNullableText } from '../../shared/display';

export type StatusFilter =
  | ''
  | NonNullable<NonNullable<ListAdmissionSessionsParams>['status']>;

export type AdmissionSessionAction = 'cancel' | 'regenerate' | 'resend';

export const STATUS_LABELS: Record<string, string> = {
  cancelled: '已取消',
  expired_kicked: '超时移出',
  joined_muted: '已入群禁言',
  linked: '已绑定账号',
  material_submitted: '材料待审',
  verified: '已通过',
};

export const STATUS_OPERATION_HINTS: Record<string, string> = {
  cancelled: '管理员已取消，此会话不再处理。',
  expired_kicked: '认证超时，成员需要重新入群或重新生成链接。',
  joined_muted: '已入群并临时禁言，等待用户打开链接或机器人重发提醒。',
  linked: '账号已绑定，等待学生认证或材料提交。',
  material_submitted: '新生材料已进入后台审核队列。',
  verified: '后端已通过，等待或已完成 Koishi 解除禁言同步。',
};

export const STATUS_OPTIONS: Array<{ label: string; value: StatusFilter }> = [
  { label: '全部状态', value: '' },
  { label: '已入群禁言', value: 'joined_muted' },
  { label: '已绑定账号', value: 'linked' },
  { label: '材料待审', value: 'material_submitted' },
  { label: '已通过', value: 'verified' },
  { label: '超时移出', value: 'expired_kicked' },
  { label: '已取消', value: 'cancelled' },
];

export function statusLabel(status: AdmissionSession['status']): string {
  return STATUS_LABELS[status] ?? status;
}

export function statusOperationHint(
  status: AdmissionSession['status'],
): string {
  return STATUS_OPERATION_HINTS[status] ?? '未知状态，请检查后端会话。';
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
  return value ? '是' : '否';
}

export function admissionReissueCommand(session: AdmissionSession): string {
  return `重新生成认证链接 ${session.qqID}`;
}

export function botErrorLabel(session: AdmissionSession): string {
  return session.lastBotError ? '存在 Bot 执行错误' : '暂无 Bot 错误';
}

export function canManageAdmissionSession(
  status: AdmissionSession['status'],
): boolean {
  return !['cancelled', 'expired_kicked', 'verified'].includes(status);
}
