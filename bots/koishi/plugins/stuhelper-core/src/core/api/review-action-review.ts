import type { Context } from 'koishi'
import {
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
  await updatePendingReview(deps, review, {
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
  await claimPendingReview({ deps, review, actor, resolutionNote: note || null, claimedAt: claimAt })
  try {
    await deps.actions.kickMember({
      bot,
      guildId: review.guildId,
      channelId: review.channelId,
      memberId: review.memberId,
      permanent: review.actionType === 'kick_and_block',
      reason: review.reason,
    })
    await finalizeClaimedReview({ deps, reviewId: review.id, actor, resolutionNote: note || null, claimedAt: claimAt, executedAt: getNow(deps) })
    await deps.moderationStore.appendEvent(createReviewResolvedEvent({ review, level: 'high', actor, note }))
    await markGuardMemberKicked(deps.ctx, review, getNow(deps))
  } catch (error) {
    await rollbackReviewClaimSafely({ deps, reviewId: review.id, claimedAt: claimAt, rolledBackAt: getNow(deps), originalError: error })
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
  deps: WorkItemActionDeps,
  review: ReviewQueueRecord,
  patch: { status: ReviewStatus, operatorMemberId: string, resolutionNote: string | null },
) {
  const matched = await deps.moderationStore.tryUpdatePendingReview({
    review,
    ...patch,
    updatedAt: getNow(deps),
  })
  if (!matched) {
    throw new Error(`review is already being processed: ${review.id}`)
  }
}

async function claimPendingReview(input: {
  readonly deps: WorkItemActionDeps
  readonly review: ReviewQueueRecord
  readonly actor: WorkItemActionActor
  readonly resolutionNote: string | null
  readonly claimedAt: Date
}) {
  const matched = await input.deps.moderationStore.tryClaimPendingReview({
    review: input.review,
    operatorMemberId: input.actor.memberId,
    resolutionNote: input.resolutionNote,
    claimedAt: input.claimedAt,
  })
  if (!matched) {
    throw new Error(`review is already being processed: ${input.review.id}`)
  }
}

async function finalizeClaimedReview(input: {
  readonly deps: WorkItemActionDeps
  readonly reviewId: string
  readonly actor: WorkItemActionActor
  readonly resolutionNote: string | null
  readonly claimedAt: Date
  readonly executedAt: Date
}) {
  const matched = await input.deps.moderationStore.tryFinalizeClaimedReview({
    reviewId: input.reviewId,
    operatorMemberId: input.actor.memberId,
    resolutionNote: input.resolutionNote,
    claimedAt: input.claimedAt,
    executedAt: input.executedAt,
  })
  if (!matched) {
    throw new Error(`review execution lost claim: ${input.reviewId}`)
  }
}

async function rollbackClaimedReview(input: {
  readonly deps: WorkItemActionDeps
  readonly reviewId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
}) {
  await input.deps.moderationStore.rollbackClaimedReview({
    reviewId: input.reviewId,
    claimedAt: input.claimedAt,
    rolledBackAt: input.rolledBackAt,
  })
}

async function rollbackReviewClaimSafely(input: {
  readonly deps: WorkItemActionDeps
  readonly reviewId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
  readonly originalError: unknown
}) {
  try {
    await rollbackClaimedReview(input)
  } catch (rollbackError) {
    logRollbackFailure({
      ctx: input.deps.ctx,
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
