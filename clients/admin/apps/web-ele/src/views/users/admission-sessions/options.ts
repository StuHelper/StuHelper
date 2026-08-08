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
  action_pending: '放行动作待执行',
  admitted: '已准入',
  awaiting_account_link: '等待绑定账号',
  awaiting_requirements: '等待满足要求',
  cancelled: '已取消',
  created: '会话已创建',
  eligible: '资格已满足',
  expired: '已过期',
  pending_manual_review: '等待人工认证审核',
  rejected: '已拒绝',
  released: '已解除限制',
};

export const STATUS_OPERATION_HINTS: Record<string, string> = {
  cancelled: '管理员已取消，此会话不再处理。',
  action_pending: '资格 revision 已锁定，等待 Koishi 幂等执行放行动作。',
  admitted: '后端已确认本次入群准入完成。',
  awaiting_account_link: '等待用户将平台账号绑定到 StuHelper。',
  awaiting_requirements: '账号已关联，但当前学生资格尚未满足群策略。',
  created: '后端已创建独立准入会话。',
  eligible: '当前资格和策略匹配，准备创建带 revision 的放行动作。',
  expired: '认证会话已过期，用户需要重新发起流程。',
  pending_manual_review: '用户的独立学生认证申请正在人工审核。',
  rejected: '本次群准入已被拒绝。',
  released: 'Koishi 已完成解除限制动作。',
};

export const STATUS_OPTIONS: Array<{ label: string; value: StatusFilter }> = [
  { label: '全部状态', value: '' },
  { label: '会话已创建', value: 'created' },
  { label: '等待绑定账号', value: 'awaiting_account_link' },
  { label: '等待满足要求', value: 'awaiting_requirements' },
  { label: '等待人工审核', value: 'pending_manual_review' },
  { label: '资格已满足', value: 'eligible' },
  { label: '放行动作待执行', value: 'action_pending' },
  { label: '已准入', value: 'admitted' },
  { label: '已解除限制', value: 'released' },
  { label: '已拒绝', value: 'rejected' },
  { label: '已过期', value: 'expired' },
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
  if (status === 'eligible' || status === 'admitted' || status === 'released')
    return 'success';
  if (status === 'expired' || status === 'cancelled') return 'info';
  if (status === 'pending_manual_review' || status === 'action_pending')
    return 'warning';
  if (status === 'awaiting_account_link' || status === 'awaiting_requirements')
    return 'primary';
  if (status === 'rejected') return 'danger';
  return 'info';
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
  return !['admitted', 'cancelled', 'expired', 'rejected', 'released'].includes(
    status,
  );
}
