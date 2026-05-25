import type { Logger, Session, Universal } from 'koishi'

import {
  type GuardPolicyStore,
  type AdmissionPendingAction,
  type PlatformClient,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import {
  assertAdmissionActionBoundary,
  isAdmissionActionPlatform,
  requireAdmissionActionPlatform,
} from './admission-action-boundary'
import {
  executeAdmissionAction,
  formatAdmissionActionError,
} from './admission-actions'
import {
  isMemberBlacklistedError,
  kickBlacklistedJoin,
  kickBlacklistedPendingMember,
} from './member-blacklist-rejection'
import {
  forwardFreshmanMaterial,
  resolveFreshmanForwardBot,
} from './freshman-forward'
import {
  muteGuardedMember,
  sendAdmissionReminder,
  sendBackendPendingReminder,
} from './member-guard-actions'
import {
  backendSyncUpdate,
  createAdmissionSessionRequest,
  createBackendPendingGuardMemberRecord,
  createGuardMemberRecord,
  requireMemberID,
  resolveGuildID,
  type AdmissionSessionCreateResult,
  type EffectiveGuardPolicy,
} from './member-records'
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

    const admission = await this.createAdmissionSessionForJoin(session, policy)
    if (!admission) {
      return
    }
    const record = createGuardMemberRecord(session, admission)
    await this.deps.guardStore.savePending(record)
    await muteGuardedMember({
      bot: session.bot,
      guildId: record.guildId,
      memberId: record.memberId,
      muteDurationMs: muteDurationMs(new Date(admission.session.initialMuteUntil)),
    })
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
      await this.syncBackendPendingMembers(bot, now)
      await this.scanBotAdmissionActions(bot, now)
    }
    await this.forwardFreshmanMaterials(bots)
  }

  private async createAdmissionSessionForJoin(
    session: Session,
    policy: EffectiveGuardPolicy,
  ): Promise<AdmissionSessionCreateResult | null> {
    try {
      return await this.deps.platform.createAdmissionSession(createAdmissionSessionRequest(session))
    } catch (error) {
      if (isMemberBlacklistedError(error)) {
        await kickBlacklistedJoin({
          session,
          moderationStore: this.deps.moderationStore,
          logger: this.deps.logger,
          error,
        })
        return null
      }
      await this.failClosedBackendUnavailableJoin(session, policy, error)
      return null
    }
  }

  private async failClosedBackendUnavailableJoin(
    session: Session,
    policy: EffectiveGuardPolicy,
    error: unknown,
  ) {
    const now = new Date()
    const message = formatAdmissionActionError(error)
    const record = createBackendPendingGuardMemberRecord({
      session,
      policy,
      lastError: message,
      now,
    })
    await this.deps.guardStore.savePending(record)
    await muteGuardedMember({
      bot: session.bot,
      guildId: record.guildId,
      memberId: record.memberId,
      muteDurationMs: policy.muteDurationSeconds * 1000,
    })
    await this.deps.guardStore.markMuted(record.id, now)
    await sendBackendPendingReminder(session.bot, record, policy.reminderTemplate)
    await this.deps.guardStore.markReminderSent(record.id, now)
    await this.reportBackendUnavailableJoin(record, policy, message)
  }

  private async reportBackendUnavailableJoin(
    record: GuardMemberRecord,
    policy: EffectiveGuardPolicy,
    message: string,
  ) {
    await this.deps.moderationStore.appendEvent({
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId: record.guildId,
      channelId: record.channelId,
      memberId: record.memberId,
      type: 'join_guarded',
      level: 'high',
      summary: `后端 admission session 创建失败，已对 ${record.memberId} 执行本地兜底禁言`,
      payload: {
        backendSyncPending: true,
        policySource: policy.source,
        templateId: policy.templateId,
        error: message,
      },
    })
    this.deps.logger.warn('group guard admission backend unavailable; member muted locally', {
      guardRecordID: record.id,
      error: message,
    })
  }

  private async scanBotAdmissionActions(bot: GuardBotRuntime, now: Date) {
    if (!isAdmissionActionPlatform(bot)) {
      return
    }

    const platform = requireAdmissionActionPlatform(bot)
    const actions = await this.deps.platform.listPendingAdmissionActions({
      platform,
      botSelfID: bot.selfId,
    })
    for (const action of actions) {
      await this.handleAdmissionAction(bot, action, now)
    }
  }

  private async syncBackendPendingMembers(bot: GuardBotRuntime, now: Date) {
    const records = await this.deps.guardStore.listBackendSyncPending(bot.platform, bot.selfId)
    for (const record of records) {
      await this.syncBackendPendingMember(bot, record, now)
    }
  }

  private async syncBackendPendingMember(bot: GuardBotRuntime, record: GuardMemberRecord, now: Date) {
    try {
      const admission = await this.deps.platform.createAdmissionSession({
        platform: record.platform,
        guildID: record.guildId,
        channelID: record.channelId,
        qqID: record.memberId,
        qqNickname: record.memberName,
        botSelfID: record.botSelfId,
      })
      const update = backendSyncUpdate(admission)
      await this.deps.guardStore.markBackendSynced(record.id, update)
      await sendAdmissionReminder(bot, { ...record, ...update }, admission.authURL)
      await this.deps.guardStore.markReminderSent(record.id, now)
    } catch (error) {
      if (isMemberBlacklistedError(error)) {
        await kickBlacklistedPendingMember({
          bot,
          record,
          guardStore: this.deps.guardStore,
          moderationStore: this.deps.moderationStore,
          logger: this.deps.logger,
          error,
          now,
        })
        return
      }
      const message = formatAdmissionActionError(error)
      await this.deps.guardStore.markLastError(record.id, message, now)
      this.deps.logger.warn('group guard admission backend sync failed', {
        guardRecordID: record.id,
        error: message,
      })
    }
  }

  private async handleAdmissionAction(bot: GuardBotRuntime, action: AdmissionPendingAction, now: Date) {
    const record = await this.deps.guardStore.findActiveByAdmissionSessionID(action.sessionID)
    try {
      await assertAdmissionActionBoundary({
        bot,
        action,
        record: record ?? null,
        policyStore: this.deps.policyStore,
      })
      const result = await executeAdmissionAction(bot, action, record ?? null)
      await this.deps.platform.recordAdmissionEvent(action.sessionID, result.event)
      await this.markActionComplete(record, result.mark, now)
    } catch (error) {
      await this.reportActionFailure({ action, record, error, now })
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

  private async reportActionFailure(input: {
    readonly action: AdmissionPendingAction
    readonly record: GuardMemberRecord | undefined
    readonly error: unknown
    readonly now: Date
  }) {
    const { action, record, error, now } = input
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

  private async forwardFreshmanMaterials(bots: readonly GuardBotRuntime[]) {
    const forwardBots = bots.filter(isAdmissionActionPlatform)
    if (!forwardBots.length) {
      return
    }

    const items = await this.deps.platform.listPendingFreshmanForwards()
    for (const item of items) {
      const bot = resolveFreshmanForwardBot(forwardBots, item)
      await forwardFreshmanMaterial(bot, item)
      await this.deps.platform.markFreshmanForwarded(item.application.id)
    }
  }
}

export interface GuardBotRuntime extends Universal.Methods {
  platform?: string
  selfId: string
  sid: string
}

function muteDurationMs(initialMuteUntil: Date) {
  const duration = initialMuteUntil.getTime() - Date.now()
  if (!Number.isFinite(duration) || duration <= 0) {
    throw new Error(POSITIVE_MUTE_DURATION_REQUIRED)
  }
  return duration
}
