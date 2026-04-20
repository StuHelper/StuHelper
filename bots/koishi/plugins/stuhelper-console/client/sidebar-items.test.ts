import assert from 'node:assert/strict'
import test from 'node:test'

import { buildSidebarItems } from './sidebar-items'

test('buildSidebarItems preserves the fixed section order and badges', () => {
  const items = buildSidebarItems({
    pendingMembers: 8,
    pendingReviews: 12,
    policyCount: 81,
    auditCount: 17,
  })

  assert.deepEqual(
    items.map((item) => [item.id, item.count]),
    [
      ['dashboard', null],
      ['enforcement', 12],
      ['identity', 8],
      ['policy', 81],
      ['audit', 17],
    ],
  )
})
