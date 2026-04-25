import assert from 'node:assert/strict'
import test from 'node:test'

import type { StuhelperConsoleData } from '../src/console-types'
import { buildSidebarItems } from './sidebar-items'

test('buildSidebarItems preserves the fixed section order and aggregates badges', () => {
  const items = buildSidebarItems({
    title: 'StuHelper 群管中心',
    generatedAt: '2026-04-20T10:00:00.000Z',
    supportedCommandIds: [],
    overview: {} as StuhelperConsoleData['overview'],
    pendingMembers: Array.from({ length: 8 }, () => ({} as StuhelperConsoleData['pendingMembers'][number])),
    pendingReviews: Array.from({ length: 12 }, () => ({} as StuhelperConsoleData['pendingReviews'][number])),
    keywordRules: Array.from({ length: 50 }, () => ({} as StuhelperConsoleData['keywordRules'][number])),
    commandPolicies: Array.from({ length: 9 }, () => ({} as StuhelperConsoleData['commandPolicies'][number])),
    memberRoles: Array.from({ length: 11 }, () => ({} as StuhelperConsoleData['memberRoles'][number])),
    guardTemplates: Array.from({ length: 7 }, () => ({} as StuhelperConsoleData['guardTemplates'][number])),
    guardBindings: Array.from({ length: 4 }, () => ({} as StuhelperConsoleData['guardBindings'][number])),
    recentEvents: Array.from({ length: 10 }, () => ({} as StuhelperConsoleData['recentEvents'][number])),
    recentReports: Array.from({ length: 7 }, () => ({} as StuhelperConsoleData['recentReports'][number])),
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

test('buildSidebarItems falls back to zero counts when console data is missing', () => {
  const items = buildSidebarItems(undefined)

  assert.deepEqual(
    items.map((item) => [item.id, item.count]),
    [
      ['dashboard', null],
      ['enforcement', 0],
      ['identity', 0],
      ['policy', 0],
      ['audit', 0],
    ],
  )
})
