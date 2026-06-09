import { h } from 'koishi'

import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type FreshmanForwardItem,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

const MINUTE_MS = 60 * 1000

export interface AdmissionReminderInput {
  readonly memberId: string
  readonly authURL: string
  readonly deadlineAt: Date
  readonly failureCount?: number
  readonly remainingRetryCount?: number
  readonly willBlacklistOnTimeout?: boolean
  readonly now?: Date
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
}

export function formatAdmissionReminder(input: AdmissionReminderInput) {
  const minutes = minutesUntil(input.deadlineAt, input.now || new Date())
  const messages = resolveGroupGuardMessages(input.messages)
  return renderMessageTemplate(messages.admissionReminder, {
    at: h.at(input.memberId),
    memberId: input.memberId,
    minutes,
    authURL: input.authURL,
    timeoutLine: admissionTimeoutLine(input, messages),
  })
}

function admissionTimeoutLine(
  input: AdmissionReminderInput,
  messages = resolveGroupGuardMessages(input.messages),
) {
  const failureCount = normalizedCount(input.failureCount)
  const remainingRetryCount = normalizedCount(input.remainingRetryCount)
  const variables = { failureCount, remainingRetryCount }
  if (input.willBlacklistOnTimeout) {
    return renderMessageTemplate(messages.admissionTimeoutBlacklist, variables)
  }
  if (failureCount > 0) {
    return renderMessageTemplate(messages.admissionTimeoutWithFailures, variables)
  }
  return renderMessageTemplate(messages.admissionTimeoutNormal, variables)
}

function normalizedCount(value: number | undefined) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? Math.floor(value) : 0
}

function minutesUntil(deadlineAt: Date, now: Date) {
  return Math.max(1, Math.ceil((deadlineAt.getTime() - now.getTime()) / MINUTE_MS))
}

export function formatFreshmanForwardSummary(
  item: FreshmanForwardItem,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
) {
  const application = item.application
  const resolvedMessages = resolveGroupGuardMessages(messages)
  const unknownField = renderMessageTemplate(resolvedMessages.freshmanForwardUnknownField)
  return renderMessageTemplate(resolvedMessages.freshmanForwardSummary, {
    applicationId: application.id,
    applicantName: application.applicantNameMasked,
    schoolName: item.schoolName || application.schoolID,
    departmentOrMajor: application.departmentOrMajor || unknownField,
    qqID: item.qqID || unknownField,
    materialType: freshmanMaterialTypeLabel(application.materialType, resolvedMessages),
    provisionalExpiresAt: application.provisionalExpiresAt || unknownField,
  })
}

function freshmanMaterialTypeLabel(
  materialType: FreshmanForwardItem['application']['materialType'],
  messages: ReturnType<typeof resolveGroupGuardMessages>,
) {
  if (materialType === 'admission_notice') {
    return renderMessageTemplate(messages.freshmanMaterialTypeAdmissionNotice)
  }
  return renderMessageTemplate(messages.freshmanMaterialTypeAdmissionCertificate)
}
