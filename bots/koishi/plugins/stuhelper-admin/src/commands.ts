import type { Context, Session } from 'koishi'

import {
  GUARD_MEMBER_TABLE,
  type GuardMemberRecord,
} from '@stuhelper/koishi-shared'
import {
  COMMAND_POLICY_IDS,
  canExecuteCommand,
  type ModerationStore,
  type ReviewActionType,
} from '@stuhelper/koishi-moderation-core'

interface AdminCommandDeps {
  moderationStore: ModerationStore
}

export function registerAdminCommands(ctx: Context, deps: AdminCommandDeps) {
  registerStatusCommand(ctx, deps)
  registerWarningCommand(ctx, deps)
  registerReviewListCommand(ctx, deps)
  registerBatchMuteCommand(ctx, deps)
  registerKickReviewCommand(ctx, deps)
  registerBlockReviewCommand(ctx, deps)
}

function registerStatusCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审状态 [guildId:text]', '查看当前群待认证成员', { authority: 3 })
    .action(async ({ session }, guildId) => {
      const targetGuildId = resolveGuildId(session, guildId)
      const denial = await ensureAdminCommandAccess(
        deps.moderationStore,
        session,
        COMMAND_POLICY_IDS.guardStatus,
        targetGuildId,
      )
      if (denial) {
        return denial
      }
      return formatPendingMembers(await listActiveGuardMembers(ctx, targetGuildId))
    })
}

function registerWarningCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审警告 <memberId:text> [guildId:text]', '查看成员当前警告次数', { authority: 3 })
    .action(async ({ session }, memberId, guildId) => {
      const targetGuildId = resolveGuildId(session, guildId)
      const denial = await ensureAdminCommandAccess(
        deps.moderationStore,
        session,
        COMMAND_POLICY_IDS.guardWarnings,
        targetGuildId,
      )
      if (denial) {
        return denial
      }
      if (!targetGuildId || !memberId?.trim()) {
        return '请在群聊中执行，或显式传入群号和成员 ID。'
      }
      return formatWarningCounter(await deps.moderationStore.getWarningCounter(targetGuildId, memberId.trim()), memberId.trim())
    })
}

function registerReviewListCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审复核 [guildId:text]', '查看当前群待复核队列', { authority: 3 })
    .action(async ({ session }, guildId) => {
      const targetGuildId = resolveGuildId(session, guildId)
      const denial = await ensureAdminCommandAccess(
        deps.moderationStore,
        session,
        COMMAND_POLICY_IDS.guardReviews,
        targetGuildId,
      )
      if (denial) {
        return denial
      }
      return formatPendingReviews(await deps.moderationStore.listPendingReviews(targetGuildId))
    })
}

function registerBatchMuteCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审禁言 <payload:text>', '批量禁言待认证成员', { authority: 3 })
    .action(async ({ session }, payload) => {
      const denial = await ensureAdminCommandAccess(
        deps.moderationStore,
        session,
        COMMAND_POLICY_IDS.guardMute,
      )
      if (denial) {
        return denial
      }
      if (!session?.guildId || !session.channelId) {
        return '请在目标群聊中执行批量禁言。'
      }
      const parsed = parseBatchMutePayload(payload)
      if (!parsed) {
        return '请提供禁言秒数和成员 ID 列表，例如：群审批量禁言 120 10001,10002'
      }
      const targets = await listActiveGuardMembers(ctx, session.guildId, parsed.memberIds)
      if (!targets.length) {
        return '没有找到可操作的待认证成员。'
      }
      for (const target of targets) {
        await session.bot.muteGuildMember(target.guildId, target.memberId, parsed.seconds * 1000)
        await deps.moderationStore.appendEvent({
          platform: session.platform,
          botSelfId: session.selfId,
          guildId: target.guildId,
          channelId: target.channelId,
          memberId: target.memberId,
          type: 'action_executed',
          level: 'high',
          summary: `管理员批量禁言了 ${target.memberId}`,
          payload: { seconds: parsed.seconds, reason: '管理员批量禁言' },
        })
        await ctx.database.set(GUARD_MEMBER_TABLE, { id: target.id }, { mutedAt: new Date(), updatedAt: new Date() })
      }
      return `已批量禁言 ${targets.length} 名成员。`
    })
}

function registerKickReviewCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审踢人申请 <memberId> <reason:text>', '提交踢人复核申请', { authority: 4 })
    .action(async ({ session }, memberId, reason) => {
      const denial = await ensureAdminCommandAccess(
        deps.moderationStore,
        session,
        COMMAND_POLICY_IDS.guardKickRequest,
      )
      if (denial) {
        return denial
      }
      return createReviewRequest(deps.moderationStore, session, memberId, reason, 'kick', '踢人')
    })
}

function registerBlockReviewCommand(ctx: Context, deps: AdminCommandDeps) {
  ctx.command('群审拉黑申请 <memberId> <reason:text>', '提交踢人并拉黑复核申请', { authority: 4 })
    .action(async ({ session }, memberId, reason) => {
      const denial = await ensureAdminCommandAccess(
        deps.moderationStore,
        session,
        COMMAND_POLICY_IDS.guardBlockRequest,
      )
      if (denial) {
        return denial
      }
      return createReviewRequest(deps.moderationStore, session, memberId, reason, 'kick_and_block', '踢人并拉黑')
    })
}

async function ensureAdminCommandAccess(
  store: ModerationStore,
  session: Session | undefined,
  commandId: string,
  targetGuildId = session?.guildId,
) {
  const guildId = targetGuildId
  if (!session || !guildId) {
    return
  }
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    store.getMemberRoles(guildId, session.userId),
  ])
  const allowed = canExecuteCommand({
    authority: resolveAuthority(session),
    memberRoles,
    policy,
  })
  if (allowed) {
    return
  }
  return '命令权限不足。'
}

async function createReviewRequest(
  store: ModerationStore,
  session: Session | undefined,
  memberId: string | undefined,
  reason: string | undefined,
  actionType: ReviewActionType,
  actionLabel: string,
) {
  if (!session?.guildId || !session.channelId || !memberId?.trim() || !reason?.trim()) {
    return '请在群聊中提供成员 ID 和原因。'
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
    summary: `${session.userId} 提交了${actionLabel}复核申请`,
    payload: { actionType, reason: reason.trim() },
  })
  return `已提交${actionLabel}复核申请：${memberId.trim()}，原因：${reason.trim()}`
}

async function listActiveGuardMembers(ctx: Context, guildId: string, memberIds?: string[]) {
  if (!guildId) {
    return []
  }
  const records = await ctx.database.get(GUARD_MEMBER_TABLE, {}) as GuardMemberRecord[]
  return records
    .filter((record) => record.guildId === guildId && !record.releasedAt && !record.kickedAt)
    .filter((record) => !memberIds?.length || memberIds.includes(record.memberId))
    .sort((left, right) => left.deadlineAt.getTime() - right.deadlineAt.getTime())
}

function formatPendingMembers(records: GuardMemberRecord[]) {
  if (!records.length) {
    return '当前没有待认证成员。'
  }
  const lines = records.map((record) => `${record.memberId} 截止 ${record.deadlineAt.toISOString()}`)
  return `待认证成员：\n${lines.join('\n')}`
}

function formatWarningCounter(
  counter: Awaited<ReturnType<ModerationStore['getWarningCounter']>>,
  memberId: string,
) {
  if (!counter) {
    return `${memberId} 当前没有警告记录。`
  }
  const lastReason = counter.lastReason || '无'
  return `${memberId} 当前累计警告 ${counter.total} 次，最近原因：${lastReason}`
}

function formatPendingReviews(reviews: Awaited<ReturnType<ModerationStore['listPendingReviews']>>) {
  if (!reviews.length) {
    return '当前没有待复核事项。'
  }
  const lines = reviews
    .sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
    .map((review) => `${review.memberId} [${review.actionType}] ${review.reason}`)
  return `待复核队列：\n${lines.join('\n')}`
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
  const [secondsText, memberIdsText] = source.split(/\s+/, 2)
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

function resolveAuthority(session: Session | undefined) {
  const target = session as { user?: { authority?: number } } | undefined
  return target?.user?.authority ?? 0
}

function resolveGuildId(session: Session | undefined, guildId: string | undefined) {
  return guildId?.trim() || session?.guildId || ''
}
