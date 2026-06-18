import { h, type Logger, type Session } from 'koishi'

import {
  resolveGroupGuardMessages,
  type GuardMemberStore,
  type GuardPolicyStore,
  type EffectiveGuardPolicy,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import {
  createTimeCodeGuardMemberRecord,
  requireMemberID,
  resolveGuildID,
} from './member-records'
import type { AdmissionSubjectPlatform } from './admission-subject-platform'
import { groupGuardMessage } from './group-guard-message-provider'

const BEIJING_OFFSET_MS = 8 * 60 * 60 * 1000
const TIME_CODE_WINDOW_BEFORE_MINUTES = 2
const TIME_CODE_WINDOW_AFTER_MINUTES = 1

export const POST_JOIN_TIME_CODE_STRATEGY = 'post_join_time_code' as const

export function isPostJoinTimeCodeStrategy(strategy?: string) {
  return strategy === POST_JOIN_TIME_CODE_STRATEGY
}

export interface PostJoinTimeCodeStrategyInput {
  readonly session: Session
  readonly platform: AdmissionSubjectPlatform
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
  readonly moderationStore: ModerationStore
  readonly logger: Logger
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
  readonly now?: () => Date
  readonly policy?: EffectiveGuardPolicy
}

export async function applyPostJoinTimeCodeStrategy(input: PostJoinTimeCodeStrategyInput) {
  const guildId = resolveGuildID(input.session)
  if (!guildId) {
    return
  }
  const memberId = requireMemberID(input.session)
  const policy = input.policy ?? await input.policyStore.resolvePolicy(input.platform, guildId)
  if (!policy || !isPostJoinTimeCodeStrategy(policy.joinHandlingStrategy) || policy.exemptUsers.includes(memberId)) {
    return
  }

  const existing = await input.guardStore.findActiveBySubject({
    platform: input.platform,
    botSelfId: input.session.selfId,
    guildId,
    memberId,
  })
  if (existing) {
    return
  }

  const now = input.now?.() ?? new Date()
  const record = createTimeCodeGuardMemberRecord({
    session: input.session,
    policy,
    platform: input.platform,
    now,
  })
  await input.guardStore.savePending(record)
  const messages = resolveGroupGuardMessages(input.messages)
  await input.session.bot.sendMessage(
    record.channelId,
    groupGuardMessage(messages, 'admissionTimeCodeReminder', {
      at: h.at(memberId),
      memberId,
      minutes: policy.kickAfterMinutes,
    }),
  )
  await input.guardStore.markReminderSent(record.id, now)
  await input.moderationStore.appendEvent({
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    type: 'join_guarded',
    level: 'medium',
    summary: groupGuardMessage(messages, 'admissionTimeCodeJoinGuardedEventSummary', { memberId: record.memberId }),
    payload: {
      joinHandlingStrategy: POST_JOIN_TIME_CODE_STRATEGY,
      deadlineAt: record.deadlineAt.toISOString(),
      timeWindowBeforeMinutes: TIME_CODE_WINDOW_BEFORE_MINUTES,
      timeWindowAfterMinutes: TIME_CODE_WINDOW_AFTER_MINUTES,
    },
  })
}

export async function handlePostJoinTimeCodeMessage(input: {
  readonly session: Session
  readonly platform: AdmissionSubjectPlatform
  readonly guardStore: GuardMemberStore
  readonly moderationStore: ModerationStore
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
  readonly now?: () => Date
}) {
  const guildId = resolveGuildID(input.session)
  const memberId = requireMemberID(input.session)
  const content = input.session.content?.trim()
  if (!guildId || !memberId || !content) {
    return false
  }
  const record = await input.guardStore.findActiveBySubject({
    platform: input.platform,
    botSelfId: input.session.selfId,
    guildId,
    memberId,
  })
  if (!record || record.backendSyncPending || record.releasedAt || record.kickedAt || record.admissionSessionID) {
    return false
  }

  const now = input.now?.() ?? new Date()
  if (new Date(record.deadlineAt).getTime() <= now.getTime()) {
    return false
  }
  const expected = allowedAdmissionTimeCodes(memberId, now)
  if (!isValidAdmissionTimeCode(content, expected)) {
    if (hasAdmissionTimeCodeCandidate(content)) {
      const messages = resolveGroupGuardMessages(input.messages)
      await input.session.bot.sendMessage(
        record.channelId,
        groupGuardMessage(messages, 'admissionTimeCodeInvalid', {
          at: h.at(memberId),
          memberId,
        }),
      )
    }
    return false
  }

  const released = await input.guardStore.markReleased(record.id, now)
  if (released === false) {
    return false
  }
  const messages = resolveGroupGuardMessages(input.messages)
  await input.session.bot.sendMessage(
    record.channelId,
    groupGuardMessage(messages, 'admissionTimeCodeVerified', {
      at: h.at(memberId),
      memberId,
    }),
  )
  await input.moderationStore.appendEvent({
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    type: 'join_released',
    level: 'low',
    summary: groupGuardMessage(messages, 'admissionTimeCodeVerifiedEventSummary', { memberId }),
    payload: {
      joinHandlingStrategy: POST_JOIN_TIME_CODE_STRATEGY,
      codeMatched: true,
    },
  })
  return true
}

export async function kickExpiredPostJoinTimeCodeMembers(input: {
  readonly bot: { selfId: string, platform?: string, kickGuildMember: (guildId: string, memberId: string, permanent?: boolean) => Promise<void>, sendMessage: (channelId: string, content: string) => Promise<unknown> }
  readonly guardStore: GuardMemberStore
  readonly moderationStore: ModerationStore
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
  readonly now?: () => Date
}) {
  const now = input.now?.() ?? new Date()
  const store = input.guardStore as GuardMemberStore & {
    listActive?: GuardMemberStore['listActive']
  }
  if (typeof store.listActive !== 'function') {
    return
  }
  const records = await store.listActive()
  const messages = resolveGroupGuardMessages(input.messages)
  for (const record of records) {
    if (record.botSelfId !== input.bot.selfId) continue
    if (record.backendSyncPending || record.admissionSessionID || record.releasedAt || record.kickedAt) continue
    if (new Date(record.deadlineAt).getTime() > now.getTime()) continue
    try {
      let noticeError: unknown
      try {
        await input.bot.sendMessage(
          record.channelId,
          groupGuardMessage(messages, 'admissionTimeCodeKickTimeout', {
            at: h.at(record.memberId),
            memberId: record.memberId,
          }),
        )
      } catch (error) {
        noticeError = error
      }
      if (noticeError) {
        await input.guardStore.markLastError(
          record.id,
          noticeError instanceof Error ? noticeError.message : String(noticeError),
          now,
        )
      }
      await input.bot.kickGuildMember(record.guildId, record.memberId)
      const kicked = await input.guardStore.markKicked(record.id, now)
      if (kicked === false) continue
      await input.moderationStore.appendEvent({
        platform: record.platform,
        botSelfId: record.botSelfId,
        guildId: record.guildId,
        channelId: record.channelId,
        memberId: record.memberId,
        type: 'join_expired',
        level: 'low',
        summary: groupGuardMessage(messages, 'admissionTimeCodeExpiredEventSummary', { memberId: record.memberId }),
        payload: {
          joinHandlingStrategy: POST_JOIN_TIME_CODE_STRATEGY,
        },
      })
    } catch (error) {
      await input.guardStore.markLastError(record.id, error instanceof Error ? error.message : String(error), now)
    }
  }
}

export function admissionTimeCode(qqID: string, now = new Date()) {
  const lastDigit = lastQQDigit(qqID)
  if (lastDigit === null) return null
  const beijing = new Date(now.getTime() + BEIJING_OFFSET_MS)
  const hhmm = beijing.getUTCHours() * 100 + beijing.getUTCMinutes()
  return String(hhmm + lastDigit).padStart(4, '0')
}

export function allowedAdmissionTimeCodes(qqID: string, now = new Date()) {
  const codes = new Set<string>()
  for (let offset = -TIME_CODE_WINDOW_BEFORE_MINUTES; offset <= TIME_CODE_WINDOW_AFTER_MINUTES; offset += 1) {
    const candidate = admissionTimeCode(qqID, new Date(now.getTime() + offset * 60_000))
    if (candidate) {
      codes.add(candidate)
    }
  }
  return codes
}

export function isValidAdmissionTimeCode(content: string, codes: ReadonlySet<string>) {
  return admissionTimeCodeCandidates(content).some((code) => codes.has(code))
}

export function hasAdmissionTimeCodeCandidate(content: string) {
  return admissionTimeCodeCandidates(content).length > 0
}

function admissionTimeCodeCandidates(content: string) {
  return (content.match(/\d+/g) ?? []).filter((code) => code.length === 4)
}

function lastQQDigit(qqID: string) {
  for (let index = qqID.length - 1; index >= 0; index -= 1) {
    const char = qqID.charCodeAt(index)
    if (char >= 48 && char <= 57) {
      return char - 48
    }
  }
  return null
}
