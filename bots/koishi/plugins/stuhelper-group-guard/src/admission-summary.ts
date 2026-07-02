import type { AdmissionSession, PlatformAPIError, PlatformClient } from '@stuhelper/koishi-shared'
import { PlatformAPIError as PlatformAPIErrorClass } from '@stuhelper/koishi-shared'

import { groupGuardMessage, type GroupGuardMessages } from './group-guard-message-provider'

export interface AdmissionSubject {
  readonly platform: string
  readonly guildID: string
  readonly qqID: string
}

/**
 * 统一的 PlatformAPIError 形状判断：优先 instanceof；当同一类被打包成
 * 两个实例时回退到 name + status 双重检查（取自 console 版的更严格实现）。
 */
export function asPlatformAPIError(error: unknown): PlatformAPIError | undefined {
  if (error instanceof PlatformAPIErrorClass) {
    return error
  }
  const candidate = error as PlatformAPIError | undefined
  if (candidate?.name === 'PlatformAPIError' && typeof candidate.status === 'number') {
    return candidate
  }
  return undefined
}

export function isAdmissionInvalidStateError(error: unknown) {
  return asPlatformAPIError(error)?.status === 409
}

/**
 * 跳过入群认证：后端返回 409（invalid state）时回查会话，
 * 若已是 cancelled 则视为跳过成功复用该会话。
 */
export async function skipAdmissionSessionOrUseCancelled(
  platform: Pick<PlatformClient, 'skipAdmissionSessionForMember' | 'getAdmissionSessionByMember'>,
  subject: AdmissionSubject,
  operatorQQID: string,
) {
  try {
    return await platform.skipAdmissionSessionForMember({
      ...subject,
      operatorQQID,
    })
  } catch (error) {
    if (!isAdmissionInvalidStateError(error)) {
      throw error
    }
    const session = await platform.getAdmissionSessionByMember(subject)
    if (session.status === 'cancelled') {
      return session
    }
    throw error
  }
}

export function formatAdmissionSessionSummary(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  return compactRenderedMessage(groupGuardMessage(messages, 'admissionQuerySummary', {
    qqID: session.qqID,
    statusLabel: statusLabel(session.status, messages),
    sessionID: session.id,
    qqLinkedLabel: isQQLinked(session)
      ? groupGuardMessage(messages, 'admissionQueryQQLinked')
      : groupGuardMessage(messages, 'admissionQueryQQUnlinked'),
    studentVerificationLabel: studentVerificationLabel(session, messages),
    deadlineLine: describeDeadline(session, messages) ?? '',
    nextStep: nextAdmissionStep(session, messages),
    lastBotErrorLine: session.lastBotError
      ? groupGuardMessage(messages, 'admissionQueryLastBotError', { lastBotError: session.lastBotError })
      : '',
  }))
}

export function reminderDeadline(session: AdmissionSession) {
  switch (session.status) {
    case 'linked':
      return new Date(session.submissionWaitDeadlineAt)
    case 'material_submitted':
      return new Date(session.manualReviewDeadlineAt || session.submissionWaitDeadlineAt)
    default:
      return new Date(session.linkWaitDeadlineAt)
  }
}

export function describeDeadline(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  switch (session.status) {
    case 'joined_muted':
      return groupGuardMessage(messages, 'admissionQueryDeadlineLink', { deadlineAt: session.linkWaitDeadlineAt })
    case 'linked':
      return groupGuardMessage(messages, 'admissionQueryDeadlineSubmission', { deadlineAt: session.submissionWaitDeadlineAt })
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionQueryDeadlineManualReview', {
        deadlineAt: session.manualReviewDeadlineAt || groupGuardMessage(messages, 'admissionQueryDeadlineUnset'),
      })
    default:
      return undefined
  }
}

export function isQQLinked(session: AdmissionSession) {
  return hasLinkedUser(session) ||
    session.status === 'linked' ||
    session.status === 'material_submitted' ||
    session.status === 'verified'
}

export function hasLinkedUser(session: AdmissionSession) {
  return session.userID !== undefined && session.userID !== null && String(session.userID).trim() !== ''
}

export function studentVerificationLabel(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  switch (session.status) {
    case 'verified':
      return groupGuardMessage(messages, 'admissionQueryStudentVerified')
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionQueryStudentFreshmanPending')
    default:
      return groupGuardMessage(messages, 'admissionQueryStudentUnverified')
  }
}

export function nextAdmissionStep(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  switch (session.status) {
    case 'joined_muted':
      return groupGuardMessage(messages, 'admissionNextStepJoinedMuted')
    case 'linked':
      return groupGuardMessage(messages, 'admissionNextStepLinked')
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionNextStepMaterialSubmitted')
    case 'verified':
      return session.lastBotError
        ? groupGuardMessage(messages, 'admissionNextStepVerifiedWithBotError')
        : groupGuardMessage(messages, 'admissionNextStepVerified')
    case 'expired_kicked':
      return hasLinkedUser(session)
        ? groupGuardMessage(messages, 'admissionNextStepExpiredKickedLinked')
        : groupGuardMessage(messages, 'admissionNextStepExpiredKicked')
    case 'cancelled':
      return groupGuardMessage(messages, 'admissionNextStepCancelled')
    default:
      return groupGuardMessage(messages, 'admissionNextStepDefault')
  }
}

export function statusLabel(
  status: AdmissionSession['status'],
  messages: GroupGuardMessages,
) {
  switch (status) {
    case 'joined_muted':
      return groupGuardMessage(messages, 'admissionStatusJoinedMuted')
    case 'linked':
      return groupGuardMessage(messages, 'admissionStatusLinked')
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionStatusMaterialSubmitted')
    case 'verified':
      return groupGuardMessage(messages, 'admissionStatusVerified')
    case 'expired_kicked':
      return groupGuardMessage(messages, 'admissionStatusExpiredKicked')
    case 'cancelled':
      return groupGuardMessage(messages, 'admissionStatusCancelled')
    default:
      return status
  }
}

export function compactRenderedMessage(message: string) {
  return message
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line.trim())
    .join('\n')
}
