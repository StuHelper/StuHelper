import { h, type Logger, type Session, type Universal } from 'koishi'

import {
  GuardPolicyStore,
  PlatformAPIError,
  type PlatformClient,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import type { GuardMemberRecord } from './model'
import type { GuardMemberStore } from './store'

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

    const verification = await this.deps.platform.getQQVerificationStatus(memberId)
    if (verification.verificationState === 'verified') {
      return
    }

    const record = createGuardMemberRecord(session, verification.verificationState, policy)
    await this.deps.guardStore.savePending(record)
    await muteGuardedMember(session.bot, record.guildId, record.memberId, policy.muteDurationSeconds)
    await this.deps.guardStore.markMuted(record.id, new Date())
    await sendReminder(session.bot, record.channelId, record.memberId, policy.reminderTemplate)
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
        verificationState: verification.verificationState,
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
    const botMap = createBotRuntimeMap(bots)
    const records = await this.deps.guardStore.listActive()
    for (const record of records) {
      const bot = botMap.get(createBotRuntimeID(record.platform, record.botSelfId))
      if (!bot) {
        const errorMessage = `guard bot not found: ${record.platform}:${record.botSelfId}`
        await this.deps.guardStore.markLastError(record.id, errorMessage, now)
        this.deps.logger.error(errorMessage, {
          guardRecordID: record.id,
          guildId: record.guildId,
          memberId: record.memberId,
        })
        continue
      }
      await this.handleSingleRecord(bot, record, now)
    }
  }

  private async handleSingleRecord(bot: GuardBotRuntime, record: GuardMemberRecord, now: Date) {
    try {
      const verification = await this.deps.platform.getQQVerificationStatus(record.memberId)
      if (verification.verificationState === 'verified') {
        await releaseGuardedMember(bot, record)
        await this.deps.guardStore.markReleased(record.id, now)
        await this.deps.moderationStore.appendEvent({
          platform: record.platform,
          botSelfId: record.botSelfId,
          guildId: record.guildId,
          channelId: record.channelId,
          memberId: record.memberId,
          type: 'join_released',
          level: 'info',
          summary: `已为 ${record.memberId} 解除禁言`,
          payload: null,
        })
        return
      }

      if (record.deadlineAt.getTime() <= now.getTime()) {
        await kickGuardedMember(bot, record)
        await this.deps.guardStore.markKicked(record.id, now)
        await this.deps.moderationStore.appendEvent({
          platform: record.platform,
          botSelfId: record.botSelfId,
          guildId: record.guildId,
          channelId: record.channelId,
          memberId: record.memberId,
          type: 'join_expired',
          level: 'high',
          summary: `${record.memberId} 认证超时，已自动移出群聊`,
          payload: null,
        })
      }
    } catch (error) {
      await this.deps.guardStore.markLastError(record.id, formatGuardError(error), now)
      this.deps.logger.warn('group guard scan failed for member', {
        guardRecordID: record.id,
        memberId: record.memberId,
        error: formatGuardError(error),
      })
    }
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
  verificationState: GuardMemberRecord['verificationState'],
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
    memberName: session.username || session.event.user?.nick || memberId,
    verificationState,
    joinedAt: now,
    deadlineAt: new Date(now.getTime() + policy.kickAfterMinutes * 60 * 1000),
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: now,
    updatedAt: now,
  }
}

function createBotRuntimeMap(bots: readonly GuardBotRuntime[]) {
  return new Map(bots.map((bot) => [createBotRuntimeID(bot.platform || '', bot.selfId), bot]))
}

function createBotRuntimeID(platform: string, botSelfId: string) {
  return `${platform}:${botSelfId}`
}

async function muteGuardedMember(bot: Universal.Methods, guildId: string, memberId: string, muteDurationSeconds: number) {
  await bot.muteGuildMember(guildId, memberId, muteDurationSeconds * 1000)
}

async function releaseGuardedMember(bot: Universal.Methods, record: GuardMemberRecord) {
  await bot.muteGuildMember(record.guildId, record.memberId, 0)
  await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已检测到你完成 StuHelper 学生认证，已自动解除禁言。`)
}

async function kickGuardedMember(bot: Universal.Methods, record: GuardMemberRecord) {
  await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 认证超时，机器人将自动移出群聊。`)
  await bot.kickGuildMember(record.guildId, record.memberId)
}

async function sendReminder(bot: Universal.Methods, channelId: string, memberId: string, template: string) {
  await bot.sendMessage(channelId, `${h.at(memberId)} ${template}`)
}

function formatGuardError(error: unknown) {
  if (error instanceof PlatformAPIError) {
    return `${error.status}:${error.message}`
  }
  return error instanceof Error ? error.message : String(error)
}
