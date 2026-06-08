import type { Logger, Session, Universal } from 'koishi'

import {
  type GuardPolicyStore,
  type GuardMemberStore,
  type GuardMemberRecord,
  type AdmissionPendingAction,
  PlatformAPIError,
  type PlatformClient,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import {
  assertAdmissionActionBoundary,
  isAdmissionActionPlatform,
  requireAdmissionActionPlatform,
} from './admission-action-boundary'
import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'
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
import type { AdmissionReminderDeduper } from './admission-reminder-deduper'
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

const POSITIVE_MUTE_DURATION_REQUIRED = 'admission session initialMuteUntil must be in the future'
const DUPLICATE_REMINDER_SUPPRESS_MS = 30_000

interface MemberGuardDeps {
  platform: PlatformClient
  guardStore: GuardMemberStore
  policyStore: GuardPolicyStore
  moderationStore: ModerationStore
  logger: Logger
  isFreshmanForwardEnabled?: () => Promise<boolean> | boolean
  reminderDeduper?: AdmissionReminderDeduper
  messages?: Partial<StuhelperGroupGuardMessageConfig>
}

export class MemberGuardService {
  private readonly activeJoinSubjects = new Set<string>()

  constructor(private readonly deps: MemberGuardDeps) {}

  async handleGuildMemberRequest(session: Session) {
    const guildId = resolveGuildID(session)
    const memberId = requireMemberID(session)
    if (!guildId) {
      return
    }
    const admissionPlatform = resolveAdmissionSubjectPlatform(session.platform)
    if (!admissionPlatform) {
      return
    }
    const policy = await this.deps.policyStore.resolvePolicy(admissionPlatform, guildId)
    if (!policy || policy.exemptUsers.includes(memberId)) {
      return
    }
    try {
      await this.resolveAndApplyJoinRequestDecision(session, admissionPlatform, guildId, memberId)
    } catch (error) {
      if (isAdmissionPolicyNotFound(error)) {
        this.deps.logger.warn('group guard admission join request policy not found', {
          platform: admissionPlatform,
          guildId,
          memberId,
        })
        return
      }
      throw error
    }
  }

  async handleGuildMemberAdded(session: Session) {
    const guildId = resolveGuildID(session)
    const memberId = requireMemberID(session)
    if (!guildId) {
      return
    }
    const admissionPlatform = resolveAdmissionSubjectPlatform(session.platform)
    if (!admissionPlatform) {
      return
    }

    const subjectKey = joinSubjectKey(admissionPlatform, session.selfId, guildId, memberId)
    if (this.activeJoinSubjects.has(subjectKey)) {
      return
    }
    this.activeJoinSubjects.add(subjectKey)
    try {
      const policy = await this.deps.policyStore.resolvePolicy(admissionPlatform, guildId)
      if (!policy || policy.exemptUsers.includes(memberId)) {
        return
      }

      const existing = await this.deps.guardStore.findActiveBySubject({
        platform: admissionPlatform,
        botSelfId: session.selfId,
        guildId,
        memberId,
      })
      if (existing) {
        return
      }

      const admission = await this.createAdmissionSessionForJoin(session, policy, admissionPlatform)
      if (!admission) {
        return
      }
      if (admission.session.status === 'verified') {
        await this.reportAlreadyVerifiedJoin(session, admission, policy, admissionPlatform)
        return
      }
      const record = createGuardMemberRecord(session, admission, admissionPlatform)
      await this.deps.guardStore.savePending(record)
      await muteGuardedMember({
        bot: session.bot,
        guildId: record.guildId,
        memberId: record.memberId,
        muteDurationMs: muteDurationMs(new Date(admission.session.initialMuteUntil)),
      })
      await this.deps.guardStore.markMuted(record.id, new Date())
      const messageID = await sendAdmissionReminder(
        session.bot,
        record,
        admission.authURL,
        admission.session,
        this.deps.messages,
      )
      this.deps.reminderDeduper?.remember(admission.session.id)
      await this.deps.guardStore.markReminderSent(record.id, new Date())
      await this.recordAdmissionReminderSent(admission.session.id, messageID)
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
    } finally {
      this.activeJoinSubjects.delete(subjectKey)
    }
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

  async handleQueuedAdmissionAction(bot: GuardBotRuntime, action: AdmissionPendingAction) {
    await this.handleAdmissionAction(bot, action, new Date())
  }

  private async createAdmissionSessionForJoin(
    session: Session,
    policy: EffectiveGuardPolicy,
    platform: NonNullable<ReturnType<typeof resolveAdmissionSubjectPlatform>>,
  ): Promise<AdmissionSessionCreateResult | null> {
    try {
      return await this.deps.platform.createAdmissionSession(createAdmissionSessionRequest(session, platform))
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
      await this.failClosedBackendUnavailableJoin(session, policy, platform, error)
      return null
    }
  }

  private async resolveAndApplyJoinRequestDecision(
    session: Session,
    platform: NonNullable<ReturnType<typeof resolveAdmissionSubjectPlatform>>,
    guildId: string,
    memberId: string,
  ) {
    const requestID = requestIDOf(session)
    const rawEvent = rawRequestEvent(session)
    const decision = await this.deps.platform.resolveJoinRequestDecision({
      platform,
      guildID: guildId,
      qqID: memberId,
      requestID,
      rawEvent,
    })
    const approve = decision.decision === 'approve'
    try {
      await session.bot.handleGuildMemberRequest(
        requestID,
        approve,
        approve ? undefined : decision.reason,
      )
      await this.recordAdmissionJoinRequest({
        platform,
        guildId,
        memberId,
        requestID,
        decision: decision.decision,
        success: true,
        rawEvent,
      })
    } catch (error) {
      await this.recordAdmissionJoinRequest({
        platform,
        guildId,
        memberId,
        requestID,
        decision: decision.decision,
        success: false,
        error: errorMessage(error),
        rawEvent,
      })
      throw error
    }
  }

  private async reportAlreadyVerifiedJoin(
    session: Session,
    admission: AdmissionSessionCreateResult,
    policy: EffectiveGuardPolicy,
    platform: NonNullable<ReturnType<typeof resolveAdmissionSubjectPlatform>>,
  ) {
    const guildId = resolveGuildID(session)
    if (!guildId) {
      return
    }
    const memberId = requireMemberID(session)
    await this.deps.moderationStore.appendEvent({
      platform,
      botSelfId: session.selfId,
      guildId,
      channelId: session.channelId || guildId,
      memberId,
      type: 'join_guarded',
      level: 'low',
      summary: `已识别 ${memberId} 为完成学生认证的成员，跳过入群禁言`,
      payload: {
        admissionSessionID: admission.session.id,
        policySource: policy.source,
        templateId: policy.templateId,
      },
    })
  }

  private async failClosedBackendUnavailableJoin(
    session: Session,
    policy: EffectiveGuardPolicy,
    platform: NonNullable<ReturnType<typeof resolveAdmissionSubjectPlatform>>,
    error: unknown,
  ) {
    const now = new Date()
    const message = formatAdmissionActionError(error)
    const record = createBackendPendingGuardMemberRecord({
      session,
      policy,
      platform,
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
    await sendBackendPendingReminder(session.bot, record, policy.reminderTemplate, this.deps.messages)
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
    const input = {
      platform,
      botSelfID: bot.selfId,
    }
    const actions = this.deps.platform.claimQueuedAdmissionActions
      ? await this.deps.platform.claimQueuedAdmissionActions(input)
      : await this.deps.platform.listPendingAdmissionActions(input)
    for (const action of actions) {
      await this.handleAdmissionAction(bot, action, now)
    }
  }

  private async syncBackendPendingMembers(bot: GuardBotRuntime, now: Date) {
    const platform = resolveAdmissionSubjectPlatform(bot.platform)
    if (!platform) {
      return
    }
    const records = await this.deps.guardStore.listBackendSyncPending(platform, bot.selfId)
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
        botSelfID: record.botSelfId,
      })
      if (admission.session.status === 'verified') {
        await bot.muteGuildMember(record.guildId, record.memberId, 0)
        const released = await this.deps.guardStore.markReleased(record.id, now)
        if (released === false) return
        await this.deps.moderationStore.appendEvent({
          platform: record.platform,
          botSelfId: record.botSelfId,
          guildId: record.guildId,
          channelId: record.channelId,
          memberId: record.memberId,
          type: 'join_guarded',
          level: 'low',
          summary: `后端同步识别 ${record.memberId} 已完成学生认证，已解除本地兜底禁言`,
          payload: {
            admissionSessionID: admission.session.id,
          },
        })
        return
      }
      const update = backendSyncUpdate(admission)
      const synced = await this.deps.guardStore.markBackendSynced(record.id, update)
      if (synced === false) return
      const messageID = await sendAdmissionReminder(
        bot,
        { ...record, ...update },
        admission.authURL,
        admission.session,
        this.deps.messages,
      )
      this.deps.reminderDeduper?.remember(admission.session.id, now)
      await this.deps.guardStore.markReminderSent(record.id, now)
      await this.recordAdmissionReminderSent(admission.session.id, messageID)
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
      if (this.isRecentDuplicateReminder(action, record, now)) {
        await this.recordAdmissionActionResult(action, {
          action: 'remind',
          success: true,
        })
        await this.markActionComplete(record, 'reminder', now)
        return
      }
      const result = await executeAdmissionAction(bot, action, record ?? null, this.deps.messages)
      if (action.action === 'remind') {
        this.deps.reminderDeduper?.remember(action.sessionID, now)
      }
      await this.recordAdmissionActionResult(action, result.event)
      await this.markActionComplete(record, result.mark, now)
    } catch (error) {
      await this.reportActionFailure({ action, record, error, now })
    }
  }

  private async recordAdmissionReminderSent(sessionID: string, messageID: string | undefined) {
    try {
      await this.deps.platform.recordAdmissionEvent(sessionID, {
        action: 'remind',
        success: true,
        ...(messageID ? { messageID } : {}),
      })
    } catch (error) {
      this.deps.logger.warn('group guard admission reminder state sync failed', {
        sessionID,
        error: formatAdmissionActionError(error),
      })
    }
  }

  private async recordAdmissionActionResult(action: AdmissionPendingAction, event: Parameters<PlatformClient['recordAdmissionEvent']>[1]) {
    if (action.actionID) {
      await this.deps.platform.recordAdmissionActionEvent(action.actionID, event)
      return
    }
    await this.deps.platform.recordAdmissionEvent(action.sessionID, event)
  }

  private async recordAdmissionJoinRequest(input: AdmissionJoinRequestRecordInput) {
    await this.deps.platform.recordJoinRequestEvent({
      platform: input.platform,
      guildID: input.guildId,
      qqID: input.memberId,
      requestID: input.requestID,
      decision: input.decision,
      success: input.success,
      rawEvent: input.rawEvent,
      ...(input.error ? { error: input.error } : {}),
    })
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
    await this.recordAdmissionActionResult(action, {
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
    if (this.deps.isFreshmanForwardEnabled && !await this.deps.isFreshmanForwardEnabled()) {
      return
    }
    const forwardBots = bots.filter(isAdmissionActionPlatform)
    if (!forwardBots.length) {
      return
    }

    const items = await this.deps.platform.listPendingFreshmanForwards()
    for (const item of items) {
      const bot = resolveFreshmanForwardBot(forwardBots, item)
      await forwardFreshmanMaterial(bot, item, this.deps.messages)
      await this.deps.platform.markFreshmanForwarded(item.application.id)
    }
  }

  private isRecentDuplicateReminder(
    action: AdmissionPendingAction,
    record: GuardMemberRecord | undefined,
    now: Date,
  ) {
    if (action.action !== 'remind') {
      return false
    }
    if (this.deps.reminderDeduper?.wasRecentlySent(action.sessionID, now)) {
      return true
    }
    return isRecentDuplicateReminder(action, record, now)
  }
}

interface AdmissionJoinRequestRecordInput {
  readonly platform: NonNullable<ReturnType<typeof resolveAdmissionSubjectPlatform>>
  readonly guildId: string
  readonly memberId: string
  readonly requestID: string
  readonly decision: 'approve' | 'reject'
  readonly success: boolean
  readonly error?: string
  readonly rawEvent: Record<string, unknown>
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

function isRecentDuplicateReminder(
  action: AdmissionPendingAction,
  record: GuardMemberRecord | undefined,
  now: Date,
) {
  if (action.action !== 'remind' || !record?.reminderSentAt) {
    return false
  }
  return now.getTime() - record.reminderSentAt.getTime() <= DUPLICATE_REMINDER_SUPPRESS_MS
}

function joinSubjectKey(platform: string, botSelfId: string, guildId: string, memberId: string) {
  return JSON.stringify([platform, botSelfId, guildId, memberId])
}

function requestIDOf(session: Session) {
  if (!session.messageId) {
    throw new Error('admission join request event missing message id')
  }
  return session.messageId
}

function rawRequestEvent(session: Session) {
  return (session.event as { _data?: Record<string, unknown> } | undefined)?._data || {}
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

function isAdmissionPolicyNotFound(error: unknown) {
  return error instanceof PlatformAPIError && error.status === 404
}
