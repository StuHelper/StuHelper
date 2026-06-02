import type {
  AdmissionSession,
  ListAdmissionSessionsParams,
} from '#/api/admin';

import { formatAdminDateTime, formatNullableText } from '../../shared/display';

export type StatusFilter =
  | ''
  | NonNullable<NonNullable<ListAdmissionSessionsParams>['status']>;

export const STATUS_LABELS: Record<string, string> = {
  cancelled: '已取消',
  expired_kicked: '超时移出',
  joined_muted: '已入群禁言',
  linked: '已绑定账号',
  material_submitted: '材料待审',
  verified: '已通过',
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

export function canManageAdmissionSession(
  status: AdmissionSession['status'],
): boolean {
  return !['cancelled', 'expired_kicked', 'verified'].includes(status);
}
