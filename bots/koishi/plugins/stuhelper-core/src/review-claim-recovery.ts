import type { Context } from 'koishi'

import type {
  ModerationEventRecord,
  ModerationStore,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'

const DEFAULT_STALE_AFTER_MS = 5 * 60_000
const RECOVERY_INTERVAL_MS = 60_000

interface RecoverStaleReviewClaimsDeps {
  listApprovedReviews: () => Promise<readonly ReviewQueueRecord[]>
  markReviewStuck: (review: ReviewQueueRecord) => Promise<boolean>
  appendEvent: (input: Omit<ModerationEventRecord, 'id' | 'createdAt' | 'updatedAt'>) => Promise<unknown>
  now?: () => Date
  staleAfterMs?: number
}

export async function recoverStaleReviewClaims(deps: RecoverStaleReviewClaimsDeps) {
  const now = deps.now ? deps.now() : new Date()
  const staleAfterMs = deps.staleAfterMs ?? DEFAULT_STALE_AFTER_MS
  const reviews = await deps.listApprovedReviews()
  let recovered = 0

  for (const review of reviews) {
    if (!isStaleClaim(review, now, staleAfterMs)) {
      continue
    }

    const marked = await deps.markReviewStuck(review)
    if (!marked) {
      continue
    }

    recovered += 1
    await deps.appendEvent({
      platform: review.platform,
      botSelfId: review.botSelfId,
      guildId: review.guildId,
      channelId: review.channelId,
      memberId: review.memberId,
      type: 'review_stuck',
      level: 'critical',
      summary: `复核执行中断，需人工核查：${review.memberId}`,
      payload: {
        reviewId: review.id,
        previousStatus: review.status,
        nextStatus: 'stuck_manual',
        operatorMemberId: review.operatorMemberId,
      },
    })
  }

  return recovered
}

function isStaleClaim(review: ReviewQueueRecord, now: Date, staleAfterMs: number) {
  return now.getTime() - review.updatedAt.getTime() >= staleAfterMs
}

export function registerReviewClaimRecovery(
  ctx: Context,
  moderationStore: Pick<ModerationStore, 'listApprovedReviews' | 'markReviewStuck' | 'appendEvent'>,
) {
  const logger = ctx.logger('stuhelperGroupCenter')
  let timer: NodeJS.Timeout | null = null

  const runRecovery = async () => {
    const recovered = await recoverStaleReviewClaims({
      listApprovedReviews: () => moderationStore.listApprovedReviews(),
      markReviewStuck: (review) => moderationStore.markReviewStuck(review, '复核执行阶段中断，已转入人工核查'),
      appendEvent: (event) => moderationStore.appendEvent(event),
    })
    if (recovered > 0) {
      logger.warn('recovered %d stale approved review claims', recovered)
    }
  }

  ctx.on('ready', () => {
    void runRecovery().catch((error) => {
      logger.error('review claim recovery failed: %s', toErrorMessage(error))
    })
    timer = setInterval(() => {
      void runRecovery().catch((error) => {
        logger.error('review claim recovery failed: %s', toErrorMessage(error))
      })
    }, RECOVERY_INTERVAL_MS)
  })

  ctx.on('dispose', () => {
    if (!timer) {
      return
    }
    clearInterval(timer)
    timer = null
  })
}

function toErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}
