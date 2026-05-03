import { h } from 'koishi'

import type { FreshmanForwardItem } from '@stuhelper/koishi-shared'

const MINUTE_MS = 60 * 1000
const UNKNOWN_FIELD = '未提供'

const MATERIAL_TYPE_LABELS = {
  admission_notice: '录取通知书',
  admission_certificate: '录取证明',
} as const

export interface AdmissionReminderInput {
  readonly memberId: string
  readonly authURL: string
  readonly deadlineAt: Date
  readonly now?: Date
}

export function formatAdmissionReminder(input: AdmissionReminderInput) {
  const minutes = minutesUntil(input.deadlineAt, input.now || new Date())
  return [
    `${h.at(input.memberId)} 请在 ${minutes} 分钟内完成 StuHelper 学生身份认证：`,
    input.authURL,
    '通过后自动解除禁言，超时将移出群聊。',
  ].join('\n')
}

function minutesUntil(deadlineAt: Date, now: Date) {
  return Math.max(1, Math.ceil((deadlineAt.getTime() - now.getTime()) / MINUTE_MS))
}

export function formatFreshmanForwardSummary(item: FreshmanForwardItem) {
  const application = item.application
  return [
    `新生材料审核 ${application.id}`,
    `姓名：${application.applicantNameMasked}`,
    `学校：${item.schoolName || application.schoolID}`,
    `专业：${application.departmentOrMajor || UNKNOWN_FIELD}`,
    `QQ：${item.qqID || UNKNOWN_FIELD}`,
    `材料：${MATERIAL_TYPE_LABELS[application.materialType]}`,
    `临时身份过期：${application.provisionalExpiresAt || UNKNOWN_FIELD}`,
    `通过：新生审核通过 ${application.id}`,
    `驳回：新生审核驳回 ${application.id} <原因>`,
  ].join('\n')
}
