import type { Context, Session } from 'koishi'

import {
  type AdmissionRuntimeSettingsStore,
  type AdminRuntimeSettingsStore,
  type GuardMemberAdminRecord,
  type GuardMemberAdminStore,
  type PlatformClient,
  renderMessageTemplate,
  resolveAdminMessages,
} from '@stuhelper/koishi-shared'
import {
  COMMAND_POLICY_IDS,
  type ModerationStore,
  type ReviewActionType,
} from '@stuhelper/koishi-moderation-core'

import {
  ensureAdminCommandAccess,
  resolveGuildId,
} from './command-access'

interface AdminCommandDeps {
  guardMembers: GuardMemberAdminStore
  moderationStore: ModerationStore
  platform: PlatformClient
  runtimeSettings?: AdmissionRuntimeSettingsStore
  adminSettings?: AdminRuntimeSettingsStore
}

type AdminMessages = ReturnType<typeof resolveAdminMessages>

export function registerAdminCommands(ctx: Context, deps: AdminCommandDeps) {
  registerStatusCommand(ctx, deps)
  registerWarningCommand(ctx, deps)
  registerReviewListCommand(ctx, deps)
  registerBatchMuteCommand(ctx, deps)
  registerKickReviewCommand(ctx, deps)
  registerBlockReviewCommand(ctx, deps)
}

function registerStatusCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审状态 [guildId:text]', adminCommandDescription('guardStatusCommandDescription'), { authority: 3 })
    .action(async ({ session }, guildId) => {
      const messages = await getAdminMessages(deps)
      const targetGuildId = resolveGuildId(session, guildId)
      const denial = await ensureAdminCommandAccess({
        store: deps.moderationStore,
        session,
        commandId: COMMAND_POLICY_IDS.guardStatus,
        targetGuildId,
        runtimeSettings: deps.runtimeSettings,
        messages,
      })
      if (denial) {
        return denial
      }
      if (!targetGuildId) {
        return adminMessage(messages, 'guardGuildContextRequired')
      }
      return formatPendingMembers(await deps.guardMembers.listActiveByGuild({ guildId: targetGuildId }), messages)
    })
}

function registerWarningCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审警告 <memberId:text> [guildId:text]', adminCommandDescription('guardWarningCommandDescription'), { authority: 3 })
    .action(async ({ session }, memberId, guildId) => {
      const messages = await getAdminMessages(deps)
      const targetGuildId = resolveGuildId(session, guildId)
      const denial = await ensureAdminCommandAccess({
        store: deps.moderationStore,
        session,
        commandId: COMMAND_POLICY_IDS.guardWarnings,
        targetGuildId,
        runtimeSettings: deps.runtimeSettings,
        messages,
      })
      if (denial) {
        return denial
      }
      if (!targetGuildId || !memberId?.trim()) {
        return adminMessage(messages, 'guardWarningMissingContext')
      }
      return formatWarningCounter(
        await deps.moderationStore.getWarningCounter(targetGuildId, memberId.trim()),
        memberId.trim(),
        messages,
      )
    })
}

function registerReviewListCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审复核 [guildId:text]', adminCommandDescription('guardReviewListCommandDescription'), { authority: 3 })
    .action(async ({ session }, guildId) => {
      const messages = await getAdminMessages(deps)
      const targetGuildId = resolveGuildId(session, guildId)
      const denial = await ensureAdminCommandAccess({
        store: deps.moderationStore,
        session,
        commandId: COMMAND_POLICY_IDS.guardReviews,
        targetGuildId,
        runtimeSettings: deps.runtimeSettings,
        messages,
      })
      if (denial) {
        return denial
      }
      if (!targetGuildId) {
        return adminMessage(messages, 'guardGuildContextRequired')
      }
      return formatPendingReviews(await deps.moderationStore.listPendingReviews(targetGuildId), messages)
    })
}

function registerBatchMuteCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审禁言 <payload:text>', adminCommandDescription('guardBatchMuteCommandDescription'), { authority: 3 })
    .action(async ({ session }, payload) => {
      const messages = await getAdminMessages(deps)
      const denial = await ensureAdminCommandAccess({
        store: deps.moderationStore,
        session,
        commandId: COMMAND_POLICY_IDS.guardMute,
        runtimeSettings: deps.runtimeSettings,
        messages,
      })
      if (denial) {
        return denial
      }
      if (!session?.guildId || !session.channelId) {
        return adminMessage(messages, 'guardBatchMuteGroupOnly')
      }
      const parsed = parseBatchMutePayload(payload)
      if (!parsed) {
        return adminMessage(messages, 'guardBatchMuteInvalidPayload')
      }
      const targets = await deps.guardMembers.listActiveByGuild({
        guildId: session.guildId,
        memberIds: parsed.memberIds,
      })
      if (!targets.length) {
        return adminMessage(messages, 'guardBatchMuteNoTargets')
      }
      for (const target of targets) {
        const mutedAt = new Date()
        await session.bot.muteGuildMember(target.guildId, target.memberId, parsed.seconds * 1000)
        await deps.moderationStore.appendEvent({
          platform: session.platform,
          botSelfId: session.selfId,
          guildId: target.guildId,
          channelId: target.channelId,
          memberId: target.memberId,
          type: 'action_executed',
          level: 'high',
          summary: adminMessage(messages, 'guardBatchMuteEventSummary', { memberId: target.memberId }),
          payload: {
            seconds: parsed.seconds,
            reason: adminMessage(messages, 'guardBatchMuteEventReason'),
          },
        })
        await deps.guardMembers.tryMarkActiveMuted({ record: target, mutedAt })
      }
      return adminMessage(messages, 'guardBatchMuteSuccess', { count: targets.length })
    })
}

function registerKickReviewCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审踢人申请 <memberId> <reason:text>', adminCommandDescription('guardKickReviewCommandDescription'), { authority: 4 })
    .action(async ({ session }, memberId, reason) => {
      const messages = await getAdminMessages(deps)
      const denial = await ensureAdminCommandAccess({
        store: deps.moderationStore,
        session,
        commandId: COMMAND_POLICY_IDS.guardKickRequest,
        runtimeSettings: deps.runtimeSettings,
        messages,
      })
      if (denial) {
        return denial
      }
      return createReviewRequest({
        store: deps.moderationStore,
        session,
        memberId,
        reason,
        actionType: 'kick',
        actionLabel: adminMessage(messages, 'guardReviewKickActionLabel'),
        messages,
      })
    })
}

function registerBlockReviewCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审拉黑申请 <memberId> <reason:text>', adminCommandDescription('guardBlockReviewCommandDescription'), { authority: 4 })
    .action(async ({ session }, memberId, reason) => {
      const messages = await getAdminMessages(deps)
      const denial = await ensureAdminCommandAccess({
        store: deps.moderationStore,
        session,
        commandId: COMMAND_POLICY_IDS.guardBlockRequest,
        runtimeSettings: deps.runtimeSettings,
        messages,
      })
      if (denial) {
        return denial
      }
      return createReviewRequest({
        store: deps.moderationStore,
        session,
        memberId,
        reason,
        actionType: 'kick_and_block',
        actionLabel: adminMessage(messages, 'guardReviewKickAndBlockActionLabel'),
        messages,
      })
    })
}

async function createReviewRequest(input: {
  readonly store: ModerationStore
  readonly session: Session | undefined
  readonly memberId: string | undefined
  readonly reason: string | undefined
  readonly actionType: ReviewActionType
  readonly actionLabel: string
  readonly messages: AdminMessages
}) {
  const { store, session, memberId, reason, actionType, actionLabel } = input
  if (!session?.guildId || !session.channelId || !memberId?.trim() || !reason?.trim()) {
    return adminMessage(input.messages, 'guardReviewRequestMissingArgs')
  }
  await store.createReview({
    platform: session.platform,
    botSelfId: session.selfId,
    guildId: session.guildId,
    channelId: session.channelId,
    memberId: memberId.trim(),
    actionType,
    status: 'pending',
    reason: reason.trim(),
    operatorMemberId: session.userId,
    resolutionNote: null,
    payload: { source: 'admin-command' },
  })
  await store.appendEvent({
    platform: session.platform,
    botSelfId: session.selfId,
    guildId: session.guildId,
    channelId: session.channelId,
    memberId: memberId.trim(),
    type: 'review_created',
    level: 'high',
    summary: adminMessage(input.messages, 'guardReviewRequestEventSummary', {
      operatorMemberId: session.userId,
      actionLabel,
    }),
    payload: { actionType, reason: reason.trim() },
  })
  return adminMessage(input.messages, 'guardReviewRequestSuccess', {
    actionLabel,
    memberId: memberId.trim(),
    reason: reason.trim(),
  })
}

function formatPendingMembers(
  records: GuardMemberAdminRecord[],
  messages: AdminMessages,
) {
  if (!records.length) {
    return adminMessage(messages, 'guardPendingMembersEmpty')
  }
  const lines = records.map((record) => adminMessage(messages, 'guardPendingMemberLine', {
    memberId: record.memberId,
    deadlineAt: record.deadlineAt.toISOString(),
  }))
  return [adminMessage(messages, 'guardPendingMembersHeader'), ...lines].join('\n')
}

function formatWarningCounter(
  counter: Awaited<ReturnType<ModerationStore['getWarningCounter']>>,
  memberId: string,
  messages: AdminMessages,
) {
  if (!counter) {
    return adminMessage(messages, 'guardWarningCounterEmpty', { memberId })
  }
  const lastReason = counter.lastReason || adminMessage(messages, 'guardWarningCounterNoReason')
  return adminMessage(messages, 'guardWarningCounterSummary', {
    memberId,
    total: counter.total,
    lastReason,
  })
}

function formatPendingReviews(
  reviews: Awaited<ReturnType<ModerationStore['listPendingReviews']>>,
  messages: AdminMessages,
) {
  if (!reviews.length) {
    return adminMessage(messages, 'guardPendingReviewsEmpty')
  }
  const lines = reviews
    .sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
    .map((review) => adminMessage(messages, 'guardPendingReviewLine', {
      memberId: review.memberId,
      actionType: review.actionType,
      reason: review.reason,
    }))
  return [adminMessage(messages, 'guardPendingReviewsHeader'), ...lines].join('\n')
}

function parseMemberIds(input: string | undefined) {
  return (input || '')
    .split(/[\s,，]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseBatchMutePayload(payload: string | undefined) {
  const source = (payload || '').trim()
  if (!source) {
    return null
  }
  const separatorIndex = source.search(/\s/)
  if (separatorIndex < 0) {
    return null
  }
  const secondsText = source.slice(0, separatorIndex)
  const memberIdsText = source.slice(separatorIndex + 1)
  const seconds = Number(secondsText)
  if (!Number.isInteger(seconds) || seconds < 0) {
    return null
  }
  const memberIds = parseMemberIds(memberIdsText)
  if (!memberIds.length) {
    return null
  }
  return { seconds, memberIds }
}

function adminMessage(
  messages: AdminMessages,
  key: keyof ReturnType<typeof resolveAdminMessages>,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(messages[key], variables)
}

function adminCommandDescription(key: keyof ReturnType<typeof resolveAdminMessages>) {
  return adminMessage(resolveAdminMessages(), key)
}

async function getAdminMessages(deps: AdminCommandDeps): Promise<AdminMessages> {
  return deps.adminSettings?.getMessages() ?? resolveAdminMessages()
}
