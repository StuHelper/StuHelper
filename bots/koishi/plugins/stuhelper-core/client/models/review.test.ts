import assert from 'node:assert/strict'
import test from 'node:test'

import type { ReviewPageData } from '../page-types'
import { buildReviewModel } from './review'

test('buildReviewModel preserves selected item when filters narrow the list', () => {
  const model = buildReviewModel(createReviewFixture(), {
    workspace: 'review',
    keyword: 'kick',
    itemId: 'rv-1',
  })

  assert.equal(model.selectedItem?.id, 'rv-1')
  assert.equal(model.filteredItems.some((item) => item.id === 'rv-1'), true)
})

function createReviewFixture(): ReviewPageData {
  return {
    generatedAt: '2026-04-21T08:00:00.000Z',
    items: [
      {
        id: 'rv-1',
        kind: 'review',
        guildId: '1001',
        memberId: '2002',
        subjectLabel: '2002',
        status: 'pending',
        priority: 'high',
        createdAt: '2026-04-21T07:55:00.000Z',
        availableActions: ['execute', 'reject'],
        relatedEventIds: ['evt-1'],
        reason: 'kick because spam',
        secondaryLabel: 'kick',
      },
      {
        id: 'gm-1',
        kind: 'admission',
        guildId: '1001',
        memberId: '2001',
        subjectLabel: 'Alice',
        status: 'bound_unverified',
        priority: 'medium',
        createdAt: '2026-04-21T07:50:00.000Z',
        availableActions: ['approve', 'deny', 'defer'],
        relatedEventIds: [],
        reason: '等待认证完成',
        secondaryLabel: '1001',
      },
    ],
    events: [
      {
        id: 'evt-1',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        memberId: '2002',
        type: 'review_created',
        level: 'high',
        summary: '创建了 kick 复核',
        payload: null,
        createdAt: '2026-04-21T07:58:00.000Z',
        updatedAt: '2026-04-21T07:58:00.000Z',
      },
    ],
  }
}
