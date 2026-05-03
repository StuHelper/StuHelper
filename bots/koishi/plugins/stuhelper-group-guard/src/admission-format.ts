import { h } from 'koishi'

const MINUTE_MS = 60 * 1000

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
