import type { Session } from 'koishi'

import {
  type GuardPolicyStore,
  type PlatformClient,
} from '@stuhelper/koishi-shared'

import type { AdmissionSubjectPlatform } from './admission-subject-platform'
import type { GuardMemberRecord } from './model'

const MINUTE_MS = 60_000

export type EffectiveGuardPolicy = NonNullable<Awaited<ReturnType<GuardPolicyStore['resolvePolicy']>>>
export type AdmissionSessionCreateResult = Awaited<ReturnType<PlatformClient['createAdmissionSession']>>

export function createAdmissionSessionRequest(session: Session, platform: AdmissionSubjectPlatform) {
  return {
    platform,
    guildID: requireGuildID(session),
    channelID: resolveChannelID(session),
    qqID: requireMemberID(session),
    botSelfID: session.selfId,
  }
}

export function createGuardMemberRecord(
  session: Session,
  admission: AdmissionSessionCreateResult,
  platform: AdmissionSubjectPlatform,
): GuardMemberRecord {
  const now = new Date()
  return createBaseGuardMemberRecord(session, {
    platform,
    admissionSessionID: admission.session.id,
    backendSyncPending: false,
    deadlineAt: new Date(admission.session.linkWaitDeadlineAt),
    nextReminderAt: new Date(admission.session.linkWaitDeadlineAt),
    manualReviewDeadlineAt: parseOptionalDate(admission.session.manualReviewDeadlineAt),
    lastError: null,
    now,
  })
}

export function createBackendPendingGuardMemberRecord(input: {
  readonly session: Session
  readonly policy: EffectiveGuardPolicy
  readonly platform: AdmissionSubjectPlatform
  readonly lastError: string
  readonly now?: Date
}): GuardMemberRecord {
  const { session, policy, platform, lastError, now = new Date() } = input
  const deadlineAt = new Date(now.getTime() + policy.kickAfterMinutes * MINUTE_MS)
  return createBaseGuardMemberRecord(session, {
    platform,
    admissionSessionID: null,
    backendSyncPending: true,
    deadlineAt,
    nextReminderAt: deadlineAt,
    manualReviewDeadlineAt: null,
    lastError,
    now,
  })
}

export function backendSyncUpdate(admission: AdmissionSessionCreateResult) {
  return {
    admissionSessionID: admission.session.id,
    backendSyncPending: false,
    deadlineAt: new Date(admission.session.linkWaitDeadlineAt),
    nextReminderAt: new Date(admission.session.linkWaitDeadlineAt),
    manualReviewDeadlineAt: parseOptionalDate(admission.session.manualReviewDeadlineAt),
  } as const
}

export function resolveGuildID(session: Session) {
  return session.guildId || session.channelId
}

export function resolveChannelID(session: Session) {
  const channelId = session.channelId || session.guildId
  if (!channelId) {
    throw new Error('group guard requires guildId or channelId')
  }
  return channelId
}

export function requireMemberID(session: Session) {
  if (!session.userId) {
    throw new Error('group guard requires session.userId')
  }
  return session.userId
}

function createBaseGuardMemberRecord(
  session: Session,
  input: GuardMemberRecordBaseInput,
): GuardMemberRecord {
  const guildId = requireGuildID(session)
  const memberId = requireMemberID(session)
  return {
    id: `${input.platform}:${session.selfId}:${guildId}:${memberId}`,
    platform: input.platform,
    botSelfId: session.selfId,
    guildId,
    channelId: resolveChannelID(session),
    memberId,
    memberName: resolveMemberName(session) || memberId,
    verificationState: 'bound_unverified',
    admissionSessionID: input.admissionSessionID,
    backendSyncPending: input.backendSyncPending,
    joinedAt: input.now,
    deadlineAt: input.deadlineAt,
    nextReminderAt: input.nextReminderAt,
    manualReviewDeadlineAt: input.manualReviewDeadlineAt,
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: input.lastError,
    createdAt: input.now,
    updatedAt: input.now,
  }
}

function requireGuildID(session: Session) {
  const guildId = resolveGuildID(session)
  if (!guildId) {
    throw new Error('group guard requires guildId or channelId')
  }
  return guildId
}

function resolveMemberName(session: Session) {
  return session.username || session.event.user?.nick || undefined
}

function parseOptionalDate(value: string | null | undefined) {
  return value ? new Date(value) : null
}

interface GuardMemberRecordBaseInput {
  readonly platform: AdmissionSubjectPlatform
  readonly admissionSessionID: string | null
  readonly backendSyncPending: boolean
  readonly deadlineAt: Date
  readonly nextReminderAt: Date | null
  readonly manualReviewDeadlineAt: Date | null
  readonly lastError: string | null
  readonly now: Date
}
