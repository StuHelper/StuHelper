import assert from 'node:assert/strict'
import test from 'node:test'

import type { ReviewPageData } from '../page-types'
import {
  buildReviewModel,
  buildReviewSelectionPatch,
} from './review'

test('buildReviewModel preserves selected item when filters narrow the list', () => {
  const model = buildReviewModel(createReviewFixture(), {
    workspace: 'review',
    keyword: 'kick',
    itemId: 'rv-1',
  })

  assert.equal(model.selectedItem?.id, 'rv-1')
  assert.equal(model.filteredItems.some((item) => item.id === 'rv-1'), true)
})

test('buildReviewSelectionPatch keeps all-type filters when selecting a typed item', () => {
  const fixture = createReviewFixture()
  const patch = buildReviewSelectionPatch({
    workspace: null,
    guildId: null,
    keyword: '',
    itemId: 'gm-1',
    items: fixture.items,
  })

  assert.deepEqual(patch, {
    workspace: null,
    guildId: null,
    memberId: '2001',
    itemId: 'gm-1',
    keyword: '',
  })
})

test('buildReviewSelectionPatch preserves explicit type filters when stale selection falls back', () => {
  const fixture = createReviewFixture()
  const model = buildReviewModel(fixture, {
    workspace: 'review',
    itemId: 'missing',
  })
  const patch = buildReviewSelectionPatch({
    workspace: 'review',
    guildId: null,
    keyword: '',
    itemId: 'missing',
    items: model.filteredItems,
  })

  assert.deepEqual(patch, {
    workspace: 'review',
    guildId: null,
    memberId: '2002',
    itemId: 'rv-1',
    keyword: '',
  })
})

test('buildReviewSelectionPatch keeps keyword search from changing the type filter', () => {
  const fixture = createReviewFixture()
  const model = buildReviewModel(fixture, {
    workspace: null,
    keyword: '举报',
  })
  const patch = buildReviewSelectionPatch({
    workspace: null,
    guildId: null,
    keyword: '举报',
    itemId: null,
    items: model.filteredItems,
  })

  assert.equal(patch.workspace, null)
  assert.equal(patch.itemId, 'rp-1')
  assert.equal(patch.keyword, '举报')
})

test('buildReviewSelectionPatch preserves filters while clearing selection for empty results', () => {
  const patch = buildReviewSelectionPatch({
    workspace: 'admission',
    guildId: '1001',
    keyword: 'not-found',
    itemId: 'missing',
    items: [],
  })

  assert.deepEqual(patch, {
    workspace: 'admission',
    guildId: '1001',
    memberId: null,
    itemId: null,
    keyword: 'not-found',
  })
})

test('buildReviewSelectionPatch does not turn a clicked item guild into a guild filter', () => {
  const fixture = createReviewFixture()
  const patch = buildReviewSelectionPatch({
    workspace: null,
    guildId: null,
    keyword: '',
    itemId: 'rp-1',
    items: fixture.items,
  })

  assert.equal(patch.guildId, null)
  assert.equal(patch.memberId, '3003')
  assert.equal(patch.itemId, 'rp-1')
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
      {
        id: 'rp-1',
        kind: 'report',
        guildId: '2002',
        memberId: '3003',
        subjectLabel: 'Bob',
        status: 'pending',
        priority: 'low',
        createdAt: '2026-04-21T07:45:00.000Z',
        availableActions: ['dismiss', 'escalate'],
        relatedEventIds: [],
        reason: '举报骚扰',
        secondaryLabel: 'report',
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
