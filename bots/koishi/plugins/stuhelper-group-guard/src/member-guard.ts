import { type Logger, type Session, type Universal } from 'koishi'

import {
  GuardPolicyStore,
  type AdmissionPendingAction,
  type PlatformClient,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import {
  executeAdmissionAction,
  formatAdmissionActionError,
} from './admission-actions'
import { formatAdmissionReminder } from './admission-format'
import type { GuardMemberRecord } from './model'
import type { GuardMemberStore } from './store'

const POSITIVE_MUTE_DURATION_REQUIRED = 'admission session initialMuteUntil must be in the future'

interface MemberGuardDeps {
  platform: PlatformClient
  guardStore: GuardMemberStore
  policyStore: GuardPolicyStore
  moderationStore: ModerationStore
  logger: Logger
}

export class MemberGuardService {
  constructor(private readonly deps: MemberGuardDeps) {}

  async handleGuildMemberAdded(session: Session) {
    const guildId = resolveGuildID(session)
    const memberId = requireMemberID(session)
    if (!guildId) {
      return
    }

    const policy = await this.deps.policyStore.resolvePolicy(session.platform, guildId)
    if (!policy || policy.exemptUsers.includes(memberId)) {
      return
    }

    const admission = await this.deps.platform.createAdmissionSession({
      platform: session.platform,
      guildID: guildId,
      channelID: resolveChannelID(session),
      qqID: memberId,
      qqNickname: resolveMemberName(session),
      botSelfID: session.selfId,
    })
    const record = createGuardMemberRecord(session, admission, policy)
    await this.deps.guardStore.savePending(record)
    await muteGuardedMember(
      session.bot,
      record.guildId,
      record.memberId,
      muteDurationMs(new Date(admission.session.initialMuteUntil)),
    )
    await this.deps.guardStore.markMuted(record.id, new Date())
    await sendAdmissionReminder(session.bot, record, admission.authURL)
    await this.deps.guardStore.markReminderSent(record.id, new Date())
    await this.deps.moderationStore.appendEvent({
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId: record.guildId,
      channelId: record.channelId,
      memberId: record.memberId,
      type: 'join_guarded',
      level: 'medium',
      summary: `已对 ${record.memberId} 执行入群待认证禁言`,
      payload: {
        admissionSessionID: admission.session.id,
        policySource: policy.source,
        templateId: policy.templateId,
      },
    })
  }

  async scanPendingMembers(bots: readonly GuardBotRuntime[]) {
    if (!bots.length) {
      return
    }

    const now = new Date()
    for (const bot of bots) {
      await this.scanBotAdmissionActions(bot, now)
    }
  }

  private async scanBotAdmissionActions(bot: GuardBotRuntime, now: Date) {
    const actions = await this.deps.platform.listPendingAdmissionActions({
      platform: bot.platform || '',
      botSelfID: bot.selfId,
    })
    for (const action of actions) {
      await this.handleAdmissionAction(bot, action, now)
    }
  }

  private async handleAdmissionAction(bot: GuardBotRuntime, action: AdmissionPendingAction, now: Date) {
    const record = await this.deps.guardStore.findActiveByAdmissionSessionID(action.sessionID)
    try {
      const result = await executeAdmissionAction(bot, action, record ?? null)
      await this.deps.platform.recordAdmissionEvent(action.sessionID, result.event)
      await this.markActionComplete(record, result.mark, now)
    } catch (error) {
      await this.reportActionFailure(action, record, error, now)
      throw error
    }
  }

  private async markActionComplete(
    record: GuardMemberRecord | undefined,
    mark: 'reminder' | 'released' | 'kicked',
    now: Date,
  ) {
    if (!record) return
    if (mark === 'reminder') {
      await this.deps.guardStore.markReminderSent(record.id, now)
      return
    }
    if (mark === 'released') {
      await this.deps.guardStore.markReleased(record.id, now)
      return
    }
    await this.deps.guardStore.markKicked(record.id, now)
  }

  private async reportActionFailure(
    action: AdmissionPendingAction,
    record: GuardMemberRecord | undefined,
    error: unknown,
    now: Date,
  ) {
    const message = formatAdmissionActionError(error)
    if (record) {
      await this.deps.guardStore.markLastError(record.id, message, now)
    }
    await this.deps.platform.recordAdmissionEvent(action.sessionID, {
      action: action.action,
      success: false,
      error: message,
    })
    this.deps.logger.warn('group guard admission action failed', {
      action: action.action,
      sessionID: action.sessionID,
      guardRecordID: record?.id,
      error: message,
    })
  }
}

export interface GuardBotRuntime extends Universal.Methods {
  platform?: string
  selfId: string
  sid: string
}

function resolveGuildID(session: Session) {
  return session.guildId || session.channelId
}

function resolveChannelID(session: Session) {
  const channelId = session.channelId || session.guildId
  if (!channelId) {
    throw new Error('group guard requires guildId or channelId')
  }
  return channelId
}

function requireMemberID(session: Session) {
  if (!session.userId) {
    throw new Error('group guard requires session.userId')
  }
  return session.userId
}

function createGuardMemberRecord(
  session: Session,
  admission: Awaited<ReturnType<PlatformClient['createAdmissionSession']>>,
  policy: Awaited<ReturnType<GuardPolicyStore['resolvePolicy']>>,
): GuardMemberRecord {
  if (!policy) {
    throw new Error('group guard policy is required')
  }
  const now = new Date()
  const guildId = resolveGuildID(session)
  const memberId = requireMemberID(session)
  if (!guildId) {
    throw new Error('group guard requires guildId or channelId')
  }
  return {
    id: `${session.platform}:${session.selfId}:${guildId}:${memberId}`,
    platform: session.platform,
    botSelfId: session.selfId,
    guildId,
    channelId: resolveChannelID(session),
    memberId,
    memberName: resolveMemberName(session) || memberId,
    verificationState: 'bound_unverified',
    admissionSessionID: admission.session.id,
    joinedAt: now,
    deadlineAt: new Date(admission.session.linkWaitDeadlineAt),
    nextReminderAt: new Date(admission.session.linkWaitDeadlineAt),
    manualReviewDeadlineAt: parseOptionalDate(admission.session.manualReviewDeadlineAt),
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: now,
    updatedAt: now,
  }
}

function resolveMemberName(session: Session) {
  return session.username || session.event.user?.nick || undefined
}

function parseOptionalDate(value: string | null | undefined) {
  return value ? new Date(value) : null
}

function muteDurationMs(initialMuteUntil: Date) {
  const duration = initialMuteUntil.getTime() - Date.now()
  if (!Number.isFinite(duration) || duration <= 0) {
    throw new Error(POSITIVE_MUTE_DURATION_REQUIRED)
  }
  return duration
}

async function muteGuardedMember(bot: Universal.Methods, guildId: string, memberId: string, muteDurationMs: number) {
  await bot.muteGuildMember(guildId, memberId, muteDurationMs)
}

async function sendAdmissionReminder(bot: Universal.Methods, record: GuardMemberRecord, authURL: string) {
  await bot.sendMessage(record.channelId, formatAdmissionReminder({
    memberId: record.memberId,
    authURL,
    deadlineAt: record.deadlineAt,
  }))
}
