import type { Context } from 'koishi'
import {
  MODERATION_REVIEW_TABLE,
  type ReviewQueueRecord,
  type ReviewStatus,
} from '@stuhelper/koishi-moderation-core'
import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'

import {
  createReviewResolvedEvent,
  requireReviewRecord,
  resolveManagedBot,
} from './review-action-support'
import { assertConsoleGuildAccess } from './console-guild-scope'
import {
  getNow,
  REVIEW_STATUS_APPROVED,
  REVIEW_STATUS_EXECUTED,
  REVIEW_STATUS_PENDING,
  REVIEW_STATUS_REJECTED,
  toErrorMessage,
  type ReviewActionInput,
  type WorkItemActionActor,
  type WorkItemActionDeps,
} from './review-action-types'

export async function handleReviewAction(
  deps: WorkItemActionDeps,
  input: ReviewActionInput,
  actor: WorkItemActionActor,
) {
  const review = await requireReviewRecord(deps.ctx, input.itemId)
  assertConsoleGuildAccess(actor.guildScope, review.guildId, 'review work item')
  if (review.status !== REVIEW_STATUS_PENDING) {
    throw new Error(`review is already resolved: ${input.itemId}`)
  }
  if (input.action === 'reject') {
    return rejectReview({ deps, review, actor, note: input.note })
  }
  return executeReview({ deps, review, actor, note: input.note })
}

async function rejectReview(input: ReviewActionRuntimeInput) {
  const { deps, review, actor, note } = input
  await updatePendingReview(deps.ctx, review, {
    status: REVIEW_STATUS_REJECTED,
    operatorMemberId: actor.memberId,
    resolutionNote: note || null,
  })
  await deps.moderationStore.appendEvent(createReviewResolvedEvent({ review, level: 'info', actor, note }))
  return `已驳回复核：${review.memberId}`
}

async function executeReview(input: ReviewActionRuntimeInput) {
  const { deps, review, actor, note } = input
  const bot = resolveManagedBot(deps.ctx, review.platform, review.botSelfId)
  const claimAt = getNow(deps)
  await claimPendingReview({ ctx: deps.ctx, review, actor, resolutionNote: note || null, claimedAt: claimAt })
  try {
    await deps.actions.kickMember({
      bot,
      guildId: review.guildId,
      channelId: review.channelId,
      memberId: review.memberId,
      permanent: review.actionType === 'kick_and_block',
      reason: review.reason,
    })
    await finalizeClaimedReview({ ctx: deps.ctx, reviewId: review.id, actor, resolutionNote: note || null, claimedAt: claimAt, executedAt: getNow(deps) })
    await deps.moderationStore.appendEvent(createReviewResolvedEvent({ review, level: 'high', actor, note }))
    await markGuardMemberKicked(deps.ctx, review, getNow(deps))
  } catch (error) {
    await rollbackReviewClaimSafely({ ctx: deps.ctx, reviewId: review.id, claimedAt: claimAt, rolledBackAt: getNow(deps), originalError: error })
    throw error
  }
  return `已执行复核动作：${review.memberId}`
}

interface ReviewActionRuntimeInput {
  readonly deps: WorkItemActionDeps
  readonly review: ReviewQueueRecord
  readonly actor: WorkItemActionActor
  readonly note?: string
}

async function updatePendingReview(
  ctx: Context,
  review: ReviewQueueRecord,
  patch: { status: ReviewStatus, operatorMemberId: string, resolutionNote: string | null },
) {
  const updatedAt = new Date()
  const result = await ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: review.id,
    status: REVIEW_STATUS_PENDING,
    updatedAt: review.updatedAt,
  }, { ...patch, updatedAt })
  if (result.matched !== 1) {
    throw new Error(`review is already being processed: ${review.id}`)
  }
}

async function claimPendingReview(input: {
  readonly ctx: Context
  readonly review: ReviewQueueRecord
  readonly actor: WorkItemActionActor
  readonly resolutionNote: string | null
  readonly claimedAt: Date
}) {
  const result = await input.ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: input.review.id,
    status: REVIEW_STATUS_PENDING,
    updatedAt: input.review.updatedAt,
  }, {
    status: REVIEW_STATUS_APPROVED,
    operatorMemberId: input.actor.memberId,
    resolutionNote: input.resolutionNote,
    updatedAt: input.claimedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`review is already being processed: ${input.review.id}`)
  }
}

async function finalizeClaimedReview(input: {
  readonly ctx: Context
  readonly reviewId: string
  readonly actor: WorkItemActionActor
  readonly resolutionNote: string | null
  readonly claimedAt: Date
  readonly executedAt: Date
}) {
  const result = await input.ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: input.reviewId,
    status: REVIEW_STATUS_APPROVED,
    updatedAt: input.claimedAt,
  }, {
    status: REVIEW_STATUS_EXECUTED,
    operatorMemberId: input.actor.memberId,
    resolutionNote: input.resolutionNote,
    updatedAt: input.executedAt,
  })
  if (result.matched !== 1) {
    throw new Error(`review execution lost claim: ${input.reviewId}`)
  }
}

async function rollbackClaimedReview(input: {
  readonly ctx: Context
  readonly reviewId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
}) {
  await input.ctx.database.set(MODERATION_REVIEW_TABLE, {
    id: input.reviewId,
    status: REVIEW_STATUS_APPROVED,
    updatedAt: input.claimedAt,
  }, {
    status: REVIEW_STATUS_PENDING,
    operatorMemberId: null,
    resolutionNote: null,
    updatedAt: input.rolledBackAt,
  })
}

async function rollbackReviewClaimSafely(input: {
  readonly ctx: Context
  readonly reviewId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
  readonly originalError: unknown
}) {
  try {
    await rollbackClaimedReview(input)
  } catch (rollbackError) {
    logRollbackFailure({
      ctx: input.ctx,
      scope: 'review',
      recordId: input.reviewId,
      rollbackError,
      originalError: input.originalError,
    })
  }
}

async function markGuardMemberKicked(ctx: Context, review: ReviewQueueRecord, now: Date) {
  const [record] = await ctx.database.get(GUARD_MEMBER_TABLE, {
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    memberId: review.memberId,
  }) as GuardMemberRecord[]
  if (!record) return
  await ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, { kickedAt: now, lastError: null, updatedAt: now })
}

function logRollbackFailure(input: {
  readonly ctx: Context
  readonly scope: 'review' | 'guard'
  readonly recordId: string
  readonly rollbackError: unknown
  readonly originalError: unknown
}) {
  const { ctx, scope, recordId, rollbackError, originalError } = input
  ctx.logger('stuhelperGroupCenter').error(
    '%s rollback failed for %s after business error: %s | original: %s',
    scope,
    recordId,
    toErrorMessage(rollbackError),
    toErrorMessage(originalError),
  )
}
