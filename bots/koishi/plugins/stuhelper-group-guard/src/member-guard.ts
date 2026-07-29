import type { Logger, Session, Universal } from 'koishi'

import {
  type GuardPolicyStore,
  type GuardMemberStore,
  type GuardMemberRecord,
  type AdmissionPendingAction,
  PlatformAPIError,
  type PlatformClient,
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
  kickBlacklistedPendingMember,
} from './member-blacklist-rejection'
import {
  forwardFreshmanMaterial,
  resolveFreshmanForwardBot,
} from './freshman-forward'
import {
  applyJoinRequestReviewStrategy,
  type JoinRequestReviewRecordInput,
} from './join-request-review-strategy'
import {
  applyPostJoinGuardStrategy,
  postJoinGuardSubjectKey,
} from './post-join-guard-strategy'
import {
  applyPostJoinTimeCodeStrategy,
  handlePostJoinTimeCodeMessage,
  isPostJoinTimeCodeStrategy,
  kickExpiredPostJoinTimeCodeMembers,
} from './post-join-time-code-strategy'
import { sendAdmissionReminder } from './member-guard-actions'
import {
  resolveAdmissionReminderDeliveryInput,
  type AdmissionReminderDeliveryConfigProvider,
} from './admission-reminder-delivery'
import type { AdmissionReminderDeduper } from './admission-reminder-deduper'
import type {
  AdmissionSubjectCoordinator,
  AdmissionSubjectRef,
} from './admission-subject-coordinator'
import {
  backendSyncUpdate,
  requireMemberID,
  resolveGuildID,
} from './member-records'
import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
} from './group-guard-message-provider'

const DUPLICATE_REMINDER_SUPPRESS_MS = 30_000

interface MemberGuardDeps {
  platform: PlatformClient
  guardStore: GuardMemberStore
  policyStore: GuardPolicyStore
  moderationStore: ModerationStore
  logger: Logger
  isFreshmanForwardEnabled?: () => Promise<boolean> | boolean
  isTimeCodeReminderEnabled?: () => Promise<boolean> | boolean
  admissionSubjectCoordinator?: AdmissionSubjectCoordinator
  reminderDeduper?: AdmissionReminderDeduper
  admissionReminderDelivery?: AdmissionReminderDeliveryConfigProvider
  messageProvider?: GroupGuardMessageProvider
  now?: () => Date
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
    try {
      await applyJoinRequestReviewStrategy({
        session,
        platform: admissionPlatform,
        guildId,
        memberId,
        platformClient: this.deps.platform,
        recordJoinRequest: (input) => this.recordAdmissionJoinRequest(input),
      })
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

    const subjectKey = postJoinGuardSubjectKey({
      platform: admissionPlatform,
      botSelfId: session.selfId,
      guildId,
      memberId,
    })
    if (this.activeJoinSubjects.has(subjectKey)) {
      return
    }
    this.activeJoinSubjects.add(subjectKey)
    try {
      const messages = await this.getMessages()
      const policy = await this.deps.policyStore.resolvePolicy(admissionPlatform, guildId)
      if (isPostJoinTimeCodeStrategy(policy?.joinHandlingStrategy)) {
        await applyPostJoinTimeCodeStrategy({
          session,
          platform: admissionPlatform,
          guardStore: this.deps.guardStore,
          policyStore: this.deps.policyStore,
          moderationStore: this.deps.moderationStore,
          logger: this.deps.logger,
          messages,
          now: this.deps.now,
          policy,
          reminderEnabled: await this.isTimeCodeReminderEnabled(),
        })
        return
      }
      await applyPostJoinGuardStrategy({
        session,
        platform: admissionPlatform,
        platformClient: this.deps.platform,
        guardStore: this.deps.guardStore,
        policyStore: this.deps.policyStore,
        moderationStore: this.deps.moderationStore,
        logger: this.deps.logger,
        admissionSubjectCoordinator: this.deps.admissionSubjectCoordinator,
        reminderDeduper: this.deps.reminderDeduper,
        admissionReminderDelivery: () => this.getAdmissionReminderDelivery(),
        messages,
        policy,
      })
    } finally {
      this.activeJoinSubjects.delete(subjectKey)
    }
  }

  async handleMessage(session: Session) {
    const guildId = session.guildId || session.event.guild?.id
    if (!guildId || !session.userId || !session.content?.trim()) {
      return false
    }
    const admissionPlatform = resolveAdmissionSubjectPlatform(session.platform)
    if (!admissionPlatform) {
      return false
    }
    const messages = await this.getMessages()
    return handlePostJoinTimeCodeMessage({
      session,
      platform: admissionPlatform,
      guardStore: this.deps.guardStore,
      moderationStore: this.deps.moderationStore,
      messages,
      now: this.deps.now,
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
      await kickExpiredPostJoinTimeCodeMembers({
        bot,
        guardStore: this.deps.guardStore,
        moderationStore: this.deps.moderationStore,
        messages: await this.getMessages(),
        now: this.deps.now,
      })
    }
    await this.forwardFreshmanMaterials(bots)
  }

  async handleQueuedAdmissionAction(bot: GuardBotRuntime, action: AdmissionPendingAction) {
    await this.handleAdmissionAction(bot, action, new Date())
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
    const messages = await this.getMessages()
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
          summary: groupGuardMessage(messages, 'admissionJoinBackendVerifiedEventSummary', { memberId: record.memberId }),
          payload: {
            admissionSessionID: admission.session.id,
          },
        })
        return
      }
      const update = backendSyncUpdate(admission)
      const synced = await this.deps.guardStore.markBackendSynced(record.id, update)
      if (synced === false) return
      const messageID = await this.sendBackendSyncReminderIfActive(
        bot,
        { ...record, ...update },
        admission,
        messages,
        now,
      )
      if (messageID === false) return
    } catch (error) {
      if (isMemberBlacklistedError(error)) {
        await kickBlacklistedPendingMember({
          bot,
          record,
          guardStore: this.deps.guardStore,
          moderationStore: this.deps.moderationStore,
          logger: this.deps.logger,
          messages,
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
    const ref = this.resolveAdmissionActionSubjectRef(bot, action, record)
    const run = () => this.handleAdmissionActionWithRecord(bot, action, record, ref, now)
    if (action.action === 'remind' && ref && this.deps.admissionSubjectCoordinator) {
      await this.deps.admissionSubjectCoordinator.runExclusive(ref, run)
      return
    }
    await run()
  }

  private async handleAdmissionActionWithRecord(
    bot: GuardBotRuntime,
    action: AdmissionPendingAction,
    record: GuardMemberRecord | undefined,
    ref: AdmissionSubjectRef | null,
    now: Date,
  ) {
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
      if (await this.isInactiveLocalReminder(action, record)) {
        await this.recordAdmissionActionResult(action, {
          action: 'remind',
          success: true,
        })
        return
      }
      const result = await executeAdmissionAction(
        bot,
        action,
        record ?? null,
        await this.getMessages(),
        await this.getAdmissionReminderDelivery(),
        action.action === 'remind' && ref && this.deps.admissionSubjectCoordinator
          ? () => !this.deps.admissionSubjectCoordinator!.isCancelled(ref, action.sessionID)
          : undefined,
      )
      if (result.mark === 'reminder') {
        this.deps.reminderDeduper?.remember(action.sessionID, now)
      }
      await this.recordAdmissionActionResult(action, result.event)
      if (result.mark !== 'none') {
        await this.markActionComplete(record, result.mark, now)
      }
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
      const dispatchAttempt = action.dispatchAttempt
      if (!Number.isInteger(dispatchAttempt) || dispatchAttempt! <= 0) {
        throw new Error(`admission action ${action.sessionID} missing valid dispatchAttempt`)
      }
      await this.deps.platform.recordAdmissionActionEvent(action.actionID, {
        ...event,
        dispatchAttempt: dispatchAttempt!,
      })
      return
    }
    await this.deps.platform.recordAdmissionEvent(action.sessionID, event)
  }

  private async recordAdmissionJoinRequest(input: JoinRequestReviewRecordInput) {
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
    const messages = await this.getMessages()
    for (const item of items) {
      const bot = resolveFreshmanForwardBot(forwardBots, item)
      await forwardFreshmanMaterial(bot, item, messages)
      await this.deps.platform.markFreshmanForwarded(item.application.id)
    }
  }

  private async getMessages() {
    return getGroupGuardMessages(this.deps.messageProvider)
  }

  private async isTimeCodeReminderEnabled() {
    return await this.deps.isTimeCodeReminderEnabled?.() ?? true
  }

  private getAdmissionReminderDelivery() {
    return resolveAdmissionReminderDeliveryInput(this.deps.admissionReminderDelivery)
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

  private async isInactiveLocalReminder(
    action: AdmissionPendingAction,
    record: GuardMemberRecord | undefined,
  ) {
    if (action.action !== 'remind') {
      return false
    }
    if (record && this.deps.admissionSubjectCoordinator?.isCancelled(record, action.sessionID)) {
      return true
    }
    const guardStore = this.deps.guardStore as GuardMemberStore & {
      findByAdmissionSessionID?: (sessionID: string) => Promise<GuardMemberRecord | undefined>
    }
    if (typeof guardStore.findByAdmissionSessionID !== 'function') {
      return false
    }
    const localRecord = await guardStore.findByAdmissionSessionID(action.sessionID)
    if (localRecord && this.deps.admissionSubjectCoordinator?.isCancelled(localRecord, action.sessionID)) {
      return true
    }
    return Boolean(localRecord?.releasedAt || localRecord?.kickedAt || record?.releasedAt || record?.kickedAt)
  }

  private async sendBackendSyncReminderIfActive(
    bot: GuardBotRuntime,
    record: GuardMemberRecord,
    admission: Awaited<ReturnType<PlatformClient['createAdmissionSession']>>,
    messages: Awaited<ReturnType<MemberGuardService['getMessages']>>,
    now: Date,
  ) {
    const send = async () => {
      if (this.deps.admissionSubjectCoordinator?.isCancelled(record, admission.session.id)) {
        await this.deps.guardStore.markReleased(record.id, now)
        return false
      }
      if (!await this.isGuardRecordActive(record.id)) {
        return false
      }
      const messageID = await sendAdmissionReminder(
        bot,
        record,
        admission.authURL,
        admission.session,
        messages,
        await this.getAdmissionReminderDelivery(),
        async () => !this.deps.admissionSubjectCoordinator?.isCancelled(record, admission.session.id) &&
          await this.isGuardRecordActive(record.id),
      )
      if (messageID === false) {
        return false
      }
      this.deps.reminderDeduper?.remember(admission.session.id, now)
      await this.deps.guardStore.markReminderSent(record.id, now)
      await this.recordAdmissionReminderSent(admission.session.id, messageID)
      return messageID
    }
    return this.deps.admissionSubjectCoordinator
      ? this.deps.admissionSubjectCoordinator.runExclusive(record, send)
      : send()
  }

  private resolveAdmissionActionSubjectRef(
    bot: GuardBotRuntime,
    action: AdmissionPendingAction,
    record: GuardMemberRecord | undefined,
  ): AdmissionSubjectRef | null {
    const platform = record?.platform ?? action.platform ?? resolveAdmissionSubjectPlatform(bot.platform)
    const botSelfId = record?.botSelfId ?? action.botSelfID ?? bot.selfId
    const guildId = record?.guildId ?? action.guildID
    const memberId = record?.memberId ?? action.qqID
    if (!platform || !botSelfId || !guildId || !memberId) {
      return null
    }
    return {
      platform,
      botSelfId,
      guildId,
      memberId,
    }
  }

  private async isGuardRecordActive(recordID: string) {
    const guardStore = this.deps.guardStore as GuardMemberStore & {
      getActiveByID?: (id: string) => Promise<GuardMemberRecord | undefined>
    }
    if (typeof guardStore.getActiveByID !== 'function') {
      return true
    }
    return Boolean(await guardStore.getActiveByID(recordID))
  }
}

export interface GuardBotRuntime extends Universal.Methods {
  platform?: string
  selfId: string
  sid: string
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

function isAdmissionPolicyNotFound(error: unknown) {
  return error instanceof PlatformAPIError && error.status === 404
}
