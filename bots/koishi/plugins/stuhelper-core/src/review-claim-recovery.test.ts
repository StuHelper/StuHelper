import assert from 'node:assert/strict'
import test from 'node:test'

import { recoverStaleReviewClaims } from './review-claim-recovery'

test('recoverStaleReviewClaims marks overdue approved reviews as stuck_manual and records an audit event', async () => {
  const review = createReview({
    updatedAt: new Date('2026-04-21T08:00:00.000Z'),
  })
  const updates: Array<{ id: string; status: string }> = []
  const events: Array<{ type: string; summary: string }> = []

  const count = await recoverStaleReviewClaims({
    now: () => new Date('2026-04-21T08:10:00.000Z'),
    staleAfterMs: 5 * 60_000,
    listApprovedReviews: async () => [review],
    markReviewStuck: async (item) => {
      updates.push({ id: item.id, status: 'stuck_manual' })
      return true
    },
    appendEvent: async (event) => {
      events.push({
        type: String(event.type),
        summary: String(event.summary),
      })
    },
  })

  assert.equal(count, 1)
  assert.deepEqual(updates, [{ id: review.id, status: 'stuck_manual' }])
  assert.deepEqual(events, [{
    type: 'review_stuck',
    summary: `复核执行中断，需人工核查：${review.memberId}`,
  }])
})

test('recoverStaleReviewClaims ignores fresh claims and CAS misses', async () => {
  const count = await recoverStaleReviewClaims({
    now: () => new Date('2026-04-21T08:10:00.000Z'),
    staleAfterMs: 5 * 60_000,
    listApprovedReviews: async () => [
      createReview({ id: 'rv-fresh', updatedAt: new Date('2026-04-21T08:08:00.000Z') }),
      createReview({ id: 'rv-cas-miss', updatedAt: new Date('2026-04-21T08:00:00.000Z') }),
    ],
    markReviewStuck: async (item) => item.id !== 'rv-cas-miss',
    appendEvent: async () => undefined,
  })

  assert.equal(count, 0)
})

function createReview(overrides: Record<string, unknown> = {}) {
  return {
    id: 'rv-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2001',
    actionType: 'kick',
    status: 'approved',
    reason: '恶意刷屏',
    operatorMemberId: 'admin-42',
    resolutionNote: '执行中',
    payload: null,
    createdAt: new Date('2026-04-21T07:55:00.000Z'),
    updatedAt: new Date('2026-04-21T08:00:00.000Z'),
    ...overrides,
  }
}
