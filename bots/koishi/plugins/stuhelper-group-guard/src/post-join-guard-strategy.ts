import type { Logger, Session } from 'koishi'

import type { AdmissionJoinHandlingStrategy } from '@stuhelper/koishi-shared'
import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type GuardMemberRecord,
  type GuardMemberStore,
  type GuardPolicyStore,
  type PlatformClient,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import { formatAdmissionActionError } from './admission-actions'
import type { AdmissionSubjectPlatform } from './admission-subject-platform'
import type {
  AdmissionSubjectCoordinator,
  AdmissionSubjectRef,
} from './admission-subject-coordinator'
import type { AdmissionReminderDeduper } from './admission-reminder-deduper'
import {
  isMemberBlacklistedError,
  kickBlacklistedJoin,
} from './member-blacklist-rejection'
import {
  muteGuardedMember,
  sendAdmissionReminder,
  sendBackendPendingReminder,
} from './member-guard-actions'
import {
  resolveAdmissionReminderDeliveryInput,
  type AdmissionReminderDeliveryConfigProvider,
} from './admission-reminder-delivery'
import {
  createAdmissionSessionRequest,
  createBackendPendingGuardMemberRecord,
  createGuardMemberRecord,
  requireMemberID,
  resolveGuildID,
  type AdmissionSessionCreateResult,
  type EffectiveGuardPolicy,
} from './member-records'

export const POST_JOIN_GUARD_STRATEGY: AdmissionJoinHandlingStrategy = 'post_join_guard'
const POSITIVE_MUTE_DURATION_REQUIRED = 'admission session initialMuteUntil must be in the future'

export function isPostJoinGuardStrategy(strategy?: AdmissionJoinHandlingStrategy) {
  return (strategy ?? POST_JOIN_GUARD_STRATEGY) === POST_JOIN_GUARD_STRATEGY
}

export interface PostJoinGuardStrategyInput {
  readonly session: Session
  readonly platform: AdmissionSubjectPlatform
  readonly platformClient: PlatformClient
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
  readonly moderationStore: ModerationStore
  readonly logger: Logger
  readonly admissionSubjectCoordinator?: AdmissionSubjectCoordinator
  readonly reminderDeduper?: AdmissionReminderDeduper
  readonly admissionReminderDelivery?: AdmissionReminderDeliveryConfigProvider
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
}

export function postJoinGuardSubjectKey(input: {
  readonly platform: AdmissionSubjectPlatform
  readonly botSelfId: string
  readonly guildId: string
  readonly memberId: string
}) {
  return JSON.stringify([input.platform, input.botSelfId, input.guildId, input.memberId])
}

export async function applyPostJoinGuardStrategy(input: PostJoinGuardStrategyInput) {
  const guildId = resolveGuildID(input.session)
  if (!guildId) {
    return
  }
  const memberId = requireMemberID(input.session)
  const policy = await input.policyStore.resolvePolicy(input.platform, guildId)
  if (!policy || policy.exemptUsers.includes(memberId)) {
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

  const admission = await createAdmissionSessionForJoin(input, policy)
  if (!admission) {
    return
  }
  if (admission.session.status === 'verified') {
    await reportAlreadyVerifiedJoin(input, admission, policy)
    return
  }

  const record = createGuardMemberRecord(input.session, admission, input.platform)
  await input.guardStore.savePending(record)
  if (await stopPostJoinIfAdmissionInactive(input, record, admission, false)) {
    return
  }
  await muteGuardedMember({
    bot: input.session.bot,
    guildId: record.guildId,
    memberId: record.memberId,
    muteDurationMs: muteDurationMs(new Date(admission.session.initialMuteUntil)),
  })
  await input.guardStore.markMuted(record.id, new Date())
  if (await stopPostJoinIfAdmissionInactive(input, record, admission, true)) {
    return
  }
  const messageID = await sendPostJoinReminderIfActive(input, record, admission)
  if (messageID === false) {
    return
  }
  await input.moderationStore.appendEvent({
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    type: 'join_guarded',
    level: 'medium',
    summary: message(input.messages, 'admissionJoinMutedEventSummary', { memberId: record.memberId }),
    payload: {
      admissionSessionID: admission.session.id,
      policySource: policy.source,
      templateId: policy.templateId,
    },
  })
}

async function stopPostJoinIfAdmissionInactive(
  input: PostJoinGuardStrategyInput,
  record: GuardMemberRecord,
  admission: AdmissionSessionCreateResult,
  muted: boolean,
) {
  if (await isPostJoinAdmissionActive(input, record, admission)) {
    return false
  }
  await input.guardStore.markReleased(record.id, new Date())
  if (muted) {
    await input.session.bot.muteGuildMember(record.guildId, record.memberId, 0)
  }
  return true
}

async function isPostJoinAdmissionActive(
  input: PostJoinGuardStrategyInput,
  record: GuardMemberRecord,
  admission: AdmissionSessionCreateResult,
) {
  if (input.admissionSubjectCoordinator?.isCancelled(record, admission.session.id)) {
    return false
  }
  const localActive = await isLocalGuardRecordActive(input.guardStore, record.id)
  if (!localActive) {
    return false
  }
  if (input.admissionSubjectCoordinator?.isCancelled(record, admission.session.id)) {
    return false
  }
  const platformClient = input.platformClient as PlatformClient & {
    getAdmissionSessionByMember?: PlatformClient['getAdmissionSessionByMember']
  }
  if (typeof platformClient.getAdmissionSessionByMember !== 'function') {
    return true
  }
  try {
    const current = await platformClient.getAdmissionSessionByMember({
      platform: record.platform,
      guildID: record.guildId,
      qqID: record.memberId,
    })
    if (input.admissionSubjectCoordinator?.isCancelled(record, admission.session.id)) {
      return false
    }
    return current.id === admission.session.id &&
      isInProgressAdmissionStatus(current.status)
  } catch (error) {
    input.logger.warn('group guard post-join admission status check failed', {
      guardRecordID: record.id,
      admissionSessionID: admission.session.id,
      error: formatAdmissionActionError(error),
    })
    return true
  }
}

async function sendPostJoinReminderIfActive(
  input: PostJoinGuardStrategyInput,
  record: GuardMemberRecord,
  admission: AdmissionSessionCreateResult,
) {
  const ref = admissionSubjectRef(record)
  const send = async () => {
    if (await stopPostJoinIfAdmissionInactive(input, record, admission, true)) {
      return false
    }
    const delivery = await resolveAdmissionReminderDeliveryInput(input.admissionReminderDelivery)
    if (await stopPostJoinIfAdmissionInactive(input, record, admission, true)) {
      return false
    }
    const messageID = await sendAdmissionReminder(
      input.session.bot,
      record,
      admission.authURL,
      admission.session,
      input.messages,
      delivery,
      () => !input.admissionSubjectCoordinator?.isCancelled(record, admission.session.id),
    )
    if (messageID === false) {
      return false
    }
    input.reminderDeduper?.remember(admission.session.id)
    await input.guardStore.markReminderSent(record.id, new Date())
    await recordAdmissionReminderSent(input, admission.session.id, messageID)
    return messageID
  }
  return input.admissionSubjectCoordinator
    ? input.admissionSubjectCoordinator.runExclusive(ref, send)
    : send()
}

function admissionSubjectRef(record: AdmissionSubjectRef): AdmissionSubjectRef {
  return {
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    memberId: record.memberId,
  }
}

async function isLocalGuardRecordActive(guardStore: GuardMemberStore, recordID: string) {
  const store = guardStore as GuardMemberStore & {
    getActiveByID?: (id: string) => Promise<GuardMemberRecord | undefined>
  }
  if (typeof store.getActiveByID !== 'function') {
    return true
  }
  return Boolean(await store.getActiveByID(recordID))
}

function isInProgressAdmissionStatus(status: string) {
  return status === 'joined_muted' || status === 'linked' || status === 'material_submitted'
}

async function createAdmissionSessionForJoin(
  input: PostJoinGuardStrategyInput,
  policy: EffectiveGuardPolicy,
): Promise<AdmissionSessionCreateResult | null> {
  try {
    return await input.platformClient.createAdmissionSession(
      createAdmissionSessionRequest(input.session, input.platform),
    )
  } catch (error) {
    if (isMemberBlacklistedError(error)) {
      await kickBlacklistedJoin({
        session: input.session,
        moderationStore: input.moderationStore,
        logger: input.logger,
        messages: input.messages,
        error,
      })
      return null
    }
    await failClosedBackendUnavailableJoin(input, policy, error)
    return null
  }
}

async function reportAlreadyVerifiedJoin(
  input: PostJoinGuardStrategyInput,
  admission: AdmissionSessionCreateResult,
  policy: EffectiveGuardPolicy,
) {
  const guildId = resolveGuildID(input.session)
  if (!guildId) {
    return
  }
  const memberId = requireMemberID(input.session)
  await input.moderationStore.appendEvent({
    platform: input.platform,
    botSelfId: input.session.selfId,
    guildId,
    channelId: input.session.channelId || guildId,
    memberId,
    type: 'join_guarded',
    level: 'low',
    summary: message(input.messages, 'admissionJoinAlreadyVerifiedEventSummary', { memberId }),
    payload: {
      admissionSessionID: admission.session.id,
      policySource: policy.source,
      templateId: policy.templateId,
    },
  })
}

async function failClosedBackendUnavailableJoin(
  input: PostJoinGuardStrategyInput,
  policy: EffectiveGuardPolicy,
  error: unknown,
) {
  const now = new Date()
  const errorMessage = formatAdmissionActionError(error)
  const record = createBackendPendingGuardMemberRecord({
    session: input.session,
    policy,
    platform: input.platform,
    lastError: errorMessage,
    now,
  })
  await input.guardStore.savePending(record)
  await muteGuardedMember({
    bot: input.session.bot,
    guildId: record.guildId,
    memberId: record.memberId,
    muteDurationMs: policy.muteDurationSeconds * 1000,
  })
  await input.guardStore.markMuted(record.id, now)
  await sendBackendPendingReminder(
    input.session.bot,
    record,
    policy.reminderTemplate,
    input.messages,
    await resolveAdmissionReminderDeliveryInput(input.admissionReminderDelivery),
  )
  await input.guardStore.markReminderSent(record.id, now)
  await reportBackendUnavailableJoin(input, record, policy, errorMessage)
}

async function reportBackendUnavailableJoin(
  input: PostJoinGuardStrategyInput,
  record: GuardMemberRecord,
  policy: EffectiveGuardPolicy,
  errorMessage: string,
) {
  await input.moderationStore.appendEvent({
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    type: 'join_guarded',
    level: 'high',
    summary: message(input.messages, 'admissionJoinBackendUnavailableEventSummary', { memberId: record.memberId }),
    payload: {
      backendSyncPending: true,
      policySource: policy.source,
      templateId: policy.templateId,
      error: errorMessage,
    },
  })
  input.logger.warn('group guard admission backend unavailable; member muted locally', {
    guardRecordID: record.id,
    error: errorMessage,
  })
}

async function recordAdmissionReminderSent(
  input: PostJoinGuardStrategyInput,
  sessionID: string,
  messageID: string | undefined,
) {
  try {
    await input.platformClient.recordAdmissionEvent(sessionID, {
      action: 'remind',
      success: true,
      ...(messageID ? { messageID } : {}),
    })
  } catch (error) {
    input.logger.warn('group guard admission reminder state sync failed', {
      sessionID,
      error: formatAdmissionActionError(error),
    })
  }
}

function message(
  messages: Partial<StuhelperGroupGuardMessageConfig> | undefined,
  key: keyof ReturnType<typeof resolveGroupGuardMessages>,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(resolveGroupGuardMessages(messages)[key], variables)
}

function muteDurationMs(initialMuteUntil: Date) {
  const duration = initialMuteUntil.getTime() - Date.now()
  if (!Number.isFinite(duration) || duration <= 0) {
    throw new Error(POSITIVE_MUTE_DURATION_REQUIRED)
  }
  return duration
}
