import { h, type Context } from 'koishi'

import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'
import {
  MODERATION_EVENT_TABLE,
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
  type ModerationBot,
  type ModerationReportRecord,
  type ReviewActionType,
  type ReviewQueueRecord,
  type ReviewStatus,
} from '@stuhelper/koishi-moderation-core'

import {
  createAdmissionEvent,
  createReportActionEvent,
  createReviewResolvedEvent,
  requireGuardRecord,
  requireReportRecord,
  requireReviewRecord,
  resolveManagedBot,
} from './review-action-support'

const REVIEW_STATUS_PENDING = 'pending'
const REVIEW_STATUS_APPROVED = 'approved'
const REVIEW_STATUS_REJECTED = 'rejected'
const REVIEW_STATUS_EXECUTED = 'executed'

export type ReviewActionInput = {
  kind: 'review'
  itemId: string
  action: 'execute' | 'reject'
  note?: string
}

export type AdmissionActionInput = {
  kind: 'admission'
  itemId: string
  action: 'approve' | 'deny' | 'defer'
  note?: string
}

export type ReportActionInput = {
  kind: 'report'
  itemId: string
  action: 'dismiss' | 'escalate' | 'create-review'
  note?: string
}

export type WorkItemActionInput =
  | ReviewActionInput
  | AdmissionActionInput
  | ReportActionInput

export interface WorkItemActionDeps {
  ctx: Context
  moderationStore: {
    appendEvent: (input: Record<string, unknown>) => Promise<unknown>
    createReview: (input: Record<string, unknown>) => Promise<{ id: string; actionType: ReviewActionType }>
    removeReport?: (id: string) => Promise<unknown>
    transact?: <T>(callback: (store: WorkItemActionDeps['moderationStore']) => Promise<T>) => Promise<T>
    updateReportAIResult: (id: string, aiStatus: string, aiSeverity: string, aiSummary: string | null) => Promise<unknown>
  }
  actions: {
    kickMember: (
      bot: ModerationBot,
      guildId: string,
      channelId: string,
      memberId: string,
      permanent: boolean,
      reason: string,
    ) => Promise<unknown>
  }
  now?: () => Date
}

export interface WorkItemActionActor {
  memberId: string
  displayName: string | null
}

export async function handleWorkItemAction(
  deps: WorkItemActionDeps,
  input: WorkItemActionInput,
  actor: WorkItemActionActor,
) {
  switch (input.kind) {
    case 'review':
      return handleReviewAction(deps, input, actor)
    case 'admission':
      return handleAdmissionAction(deps, input, actor)
    case 'report':
      return handleReportAction(deps, input, actor)
  }
}

async function handleReviewAction(
  deps: WorkItemActionDeps,
  input: ReviewActionInput,
  actor: WorkItemActionActor,
) {
  const review = await requireReviewRecord(deps.ctx, input.itemId)
  if (review.status !== REVIEW_STATUS_PENDING) {
    throw new Error(`review is already resolved: ${input.itemId}`)
  }

  if (input.action === 'reject') {
    await updatePendingReview(deps.ctx, review, {
      status: REVIEW_STATUS_REJECTED,
      operatorMemberId: actor.memberId,
      resolutionNote: input.note || null,
    })
    await deps.moderationStore.appendEvent(createReviewResolvedEvent(review, 'info', actor, input.note))
    return `已驳回复核：${review.memberId}`
  }

  const bot = resolveManagedBot(deps.ctx, review.platform, review.botSelfId)
  const permanent = review.actionType === 'kick_and_block'
  const claimAt = getNow(deps)
  await claimPendingReview(deps.ctx, review, actor, input.note || null, claimAt)
  try {
    await deps.actions.kickMember(bot, review.guildId, review.channelId, review.memberId, permanent, review.reason)
    await finalizeClaimedReview(deps.ctx, review.id, actor, input.note || null, claimAt, getNow(deps))
    await deps.moderationStore.appendEvent(createReviewResolvedEvent(review, 'high', actor, input.note))
    await markGuardMemberKicked(deps.ctx, review, getNow(deps))
  } catch (error) {
    await rollbackReviewClaimSafely(deps.ctx, review.id, claimAt, getNow(deps), error)
    throw error
  }
  return `已执行复核动作：${review.memberId}`
}

async function handleAdmissionAction(
  deps: WorkItemActionDeps,
  input: AdmissionActionInput,
  actor: WorkItemActionActor,
) {
  const record = await requireGuardRecord(deps.ctx, input.itemId)
  const now = getNow(deps)

  if (input.action === 'approve') {
    return approveAdmission(deps, record, actor, now, input.note)
  }
  if (input.action === 'deny') {
    return denyAdmission(deps, record, actor, now, input.note)
  }
  return deferAdmission(deps, record, actor, now, input.note)
}

async function approveAdmission(
  deps: WorkItemActionDeps,
  record: GuardMemberRecord,
  actor: WorkItemActionActor,
  now: Date,
  note?: string,
) {
  const claimAt = new Date(now)
  await claimPendingGuardRecord(deps.ctx, record, claimAt)
  const bot = resolveManagedBot(deps.ctx, record.platform, record.botSelfId)
  try {
    await bot.muteGuildMember(record.guildId, record.memberId, 0)
    await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已通过人工准入，已解除限制。`)
    await finalizeGuardRecordRelease(deps.ctx, record.id, claimAt, now)
    await deps.moderationStore.appendEvent(createAdmissionEvent(record, 'join_released', 'info', '控制台已放行', actor, note))
  } catch (error) {
    await rollbackGuardClaimSafely(deps.ctx, record.id, claimAt, getNow(deps), error)
    throw error
  }
  return `已放行待准入成员：${record.memberId}`
}

async function denyAdmission(
  deps: WorkItemActionDeps,
  record: GuardMemberRecord,
  actor: WorkItemActionActor,
  now: Date,
  note?: string,
) {
  const claimAt = new Date(now)
  await claimPendingGuardRecord(deps.ctx, record, claimAt)
  const bot = resolveManagedBot(deps.ctx, record.platform, record.botSelfId)
  try {
    await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已被人工拒绝准入，机器人将移出群聊。`)
    await bot.kickGuildMember(record.guildId, record.memberId)
    await finalizeGuardRecordKick(deps.ctx, record.id, claimAt, now)
    await deps.moderationStore.appendEvent(createAdmissionEvent(record, 'action_executed', 'high', '控制台已拒绝准入并移出', actor, note, 'deny-admission'))
  } catch (error) {
    await rollbackGuardClaimSafely(deps.ctx, record.id, claimAt, getNow(deps), error)
    throw error
  }
  return `已拒绝待准入成员：${record.memberId}`
}

async function deferAdmission(
  deps: WorkItemActionDeps,
  record: GuardMemberRecord,
  actor: WorkItemActionActor,
  now: Date,
  note?: string,
) {
  const deferredDeadline = resolveDeferredDeadline(record, now)
  const result = await deps.ctx.database.set(GUARD_MEMBER_TABLE, {
    id: record.id,
    updatedAt: record.updatedAt,
    releasedAt: null,
    kickedAt: null,
  }, {
    deadlineAt: deferredDeadline,
    lastError: null,
    updatedAt: now,
  })
  if (result.matched !== 1) {
    throw new Error(`guard member is already being processed: ${record.id}`)
  }
  await deps.moderationStore.appendEvent(createAdmissionEvent(record, 'action_executed', 'medium', '控制台已延期准入处理', actor, note, 'defer-admission', deferredDeadline.toISOString()))
  return `已延期待准入成员：${record.memberId}`
}

async function handleReportAction(
  deps: WorkItemActionDeps,
  input: ReportActionInput,
  actor: WorkItemActionActor,
) {
  const report = await requireReportRecord(deps.ctx, input.itemId)
  const store = resolveTransactionalStore(deps)

  if (input.action === 'dismiss') {
    await store.transact(async (tx) => {
      await tx.removeReport?.(report.id)
      await tx.appendEvent(createReportActionEvent(report, '控制台已驳回举报', 'info', 'dismiss-report', actor, input.note))
    })
    return `已驳回举报：${report.targetMemberId}`
  }

  if (input.action === 'escalate') {
    const nextSummary = input.note?.trim() || report.aiSummary || report.reason
    await store.transact(async (tx) => {
      await tx.updateReportAIResult(report.id, 'completed', 'high', nextSummary)
      await tx.appendEvent(createReportActionEvent(report, '控制台已升级举报', 'high', 'escalate-report', actor, input.note))
    })
    return `已升级举报：${report.targetMemberId}`
  }

  await store.transact(async (tx) => {
    const review = await tx.createReview({
      platform: report.platform,
      botSelfId: report.botSelfId,
      guildId: report.guildId,
      channelId: report.channelId,
      memberId: report.targetMemberId,
      actionType: selectReportReviewAction(report),
      status: REVIEW_STATUS_PENDING,
      reason: report.reason,
      operatorMemberId: actor.memberId,
      resolutionNote: null,
      payload: {
        source: 'console-report',
        reportId: report.id,
        aiSeverity: report.aiSeverity,
        aiSummary: report.aiSummary,
        note: input.note || null,
        operatorMemberId: actor.memberId,
        operatorName: actor.displayName,
      },
    })
    await tx.removeReport?.(report.id)
    await tx.appendEvent({
      platform: report.platform,
      botSelfId: report.botSelfId,
      guildId: report.guildId,
      channelId: report.channelId,
      memberId: report.targetMemberId,
      type: 'review_created',
      level: 'high',
      summary: `控制台已把举报转入复核：${report.targetMemberId}`,
      payload: {
        actionType: review.actionType,
        reportId: report.id,
        reviewId: review.id,
        source: 'console',
        operatorMemberId: actor.memberId,
        operatorName: actor.displayName,
      },
    })
  })
  return `已把举报转成复核：${report.targetMemberId}`
}

function resolveDeferredDeadline(record: GuardMemberRecord, now: Date) {
  const initialWindowMs = record.deadlineAt.getTime() - record.joinedAt.getTime()
  if (initialWindowMs <= 0) {
    throw new Error(`guard record has invalid deadline window: ${record.id}`)
  }
  return new Date(Math.max(now.getTime(), record.deadlineAt.getTime()) + initialWindowMs)
}

function selectReportReviewAction(report: ModerationReportRecord): ReviewActionType {
  return report.aiSeverity === 'high' ? 'kick_and_block' : 'kick'
}

async function updatePendingReview(
  ctx: Context,
  review: ReviewQueueRecord,
  patch: {
    status: ReviewStatus
    operatorMemberId: string
    resolutionNote: string | null
  },
) {
  const updatedAt = new Date()
  const result = await ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: review.id,
    status: REVIEW_STATUS_PENDING,
    updatedAt: review.updatedAt,
  }, {
    ...patch,
    updatedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`review is already being processed: ${review.id}`)
  }
}

async function claimPendingReview(
  ctx: Context,
  review: ReviewQueueRecord,
  actor: WorkItemActionActor,
  resolutionNote: string | null,
  claimedAt: Date,
) {
  const result = await ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: review.id,
    status: REVIEW_STATUS_PENDING,
    updatedAt: review.updatedAt,
  }, {
    status: REVIEW_STATUS_APPROVED,
    operatorMemberId: actor.memberId,
    resolutionNote,
    updatedAt: claimedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`review is already being processed: ${review.id}`)
  }
}

async function finalizeClaimedReview(
  ctx: Context,
  reviewId: string,
  actor: WorkItemActionActor,
  resolutionNote: string | null,
  claimedAt: Date,
  executedAt: Date,
) {
  const result = await ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: reviewId,
    status: REVIEW_STATUS_APPROVED,
    updatedAt: claimedAt,
  }, {
    status: REVIEW_STATUS_EXECUTED,
    operatorMemberId: actor.memberId,
    resolutionNote,
    updatedAt: executedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`review execution lost claim: ${reviewId}`)
  }
}

async function rollbackClaimedReview(
  ctx: Context,
  reviewId: string,
  claimedAt: Date,
  rolledBackAt: Date,
) {
  await ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: reviewId,
    status: REVIEW_STATUS_APPROVED,
    updatedAt: claimedAt,
  }, {
    status: REVIEW_STATUS_PENDING,
    operatorMemberId: null,
    resolutionNote: null,
    updatedAt: rolledBackAt,
  })
}

async function rollbackReviewClaimSafely(
  ctx: Context,
  reviewId: string,
  claimedAt: Date,
  rolledBackAt: Date,
  originalError: unknown,
) {
  try {
    await rollbackClaimedReview(ctx, reviewId, claimedAt, rolledBackAt)
  } catch (rollbackError) {
    logRollbackFailure(ctx, 'review', reviewId, rollbackError, originalError)
  }
}

async function claimPendingGuardRecord(
  ctx: Context,
  record: GuardMemberRecord,
  claimedAt: Date,
) {
  const result = await ctx.database.set(GUARD_MEMBER_TABLE, {
    id: record.id,
    updatedAt: record.updatedAt,
    releasedAt: null,
    kickedAt: null,
  }, {
    updatedAt: claimedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`guard member is already being processed: ${record.id}`)
  }
}

async function finalizeGuardRecordRelease(
  ctx: Context,
  guardId: string,
  claimedAt: Date,
  releasedAt: Date,
) {
  const result = await ctx.database.set(GUARD_MEMBER_TABLE, {
    id: guardId,
    updatedAt: claimedAt,
    releasedAt: null,
    kickedAt: null,
  }, {
    releasedAt,
    lastError: null,
    updatedAt: releasedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`guard member release lost claim: ${guardId}`)
  }
}

async function finalizeGuardRecordKick(
  ctx: Context,
  guardId: string,
  claimedAt: Date,
  kickedAt: Date,
) {
  const result = await ctx.database.set(GUARD_MEMBER_TABLE, {
    id: guardId,
    updatedAt: claimedAt,
    releasedAt: null,
    kickedAt: null,
  }, {
    kickedAt,
    lastError: null,
    updatedAt: kickedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`guard member kick lost claim: ${guardId}`)
  }
}

async function rollbackGuardRecordClaim(
  ctx: Context,
  guardId: string,
  claimedAt: Date,
  rolledBackAt: Date,
  error: unknown,
) {
  const message = error instanceof Error ? error.message : String(error)
  await ctx.database.set(GUARD_MEMBER_TABLE, {
    id: guardId,
    updatedAt: claimedAt,
    releasedAt: null,
    kickedAt: null,
  }, {
    lastError: message,
    updatedAt: rolledBackAt,
  })
}

async function rollbackGuardClaimSafely(
  ctx: Context,
  guardId: string,
  claimedAt: Date,
  rolledBackAt: Date,
  originalError: unknown,
) {
  try {
    await rollbackGuardRecordClaim(ctx, guardId, claimedAt, rolledBackAt, originalError)
  } catch (rollbackError) {
    logRollbackFailure(ctx, 'guard', guardId, rollbackError, originalError)
  }
}

function logRollbackFailure(
  ctx: Context,
  scope: 'review' | 'guard',
  recordId: string,
  rollbackError: unknown,
  originalError: unknown,
) {
  ctx.logger('stuhelperGroupCenter').error(
    '%s rollback failed for %s after business error: %s | original: %s',
    scope,
    recordId,
    toErrorMessage(rollbackError),
    toErrorMessage(originalError),
  )
}

function toErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

function resolveTransactionalStore(deps: WorkItemActionDeps) {
  if (deps.moderationStore.transact && deps.moderationStore.removeReport) {
    return deps.moderationStore
  }

  return {
    ...deps.moderationStore,
    removeReport: async (id: string) => {
      await deps.ctx.database.remove(MODERATION_REPORT_TABLE, { id })
    },
    transact: async <T>(callback: (store: WorkItemActionDeps['moderationStore']) => Promise<T>) => {
      return deps.ctx.database.transact(async () => {
        const store = {
          ...deps.moderationStore,
          removeReport: async (id: string) => {
            await deps.ctx.database.remove(MODERATION_REPORT_TABLE, { id })
          },
        }
        return callback(store)
      })
    },
  }
}

async function markGuardMemberKicked(ctx: Context, review: ReviewQueueRecord, now: Date) {
  const [record] = await ctx.database.get(GUARD_MEMBER_TABLE, {
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    memberId: review.memberId,
  }) as GuardMemberRecord[]
  if (!record) {
    return
  }
  await ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, { kickedAt: now, lastError: null, updatedAt: now })
}

function getNow(deps: WorkItemActionDeps) {
  return deps.now ? deps.now() : new Date()
}
